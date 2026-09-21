package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/common"
	"github.com/verity-team/dws/internal/delphi/server"
	"golang.org/x/sync/errgroup"
)

const (
	// the API only accepts small JSON bodies (see the body limit below), so a
	// client that needs longer than this to deliver its request headers/body is
	// either broken or attempting a slowloris style attack
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	// generous enough for the slowest database backed response, low enough to
	// stop a stalled client from pinning a connection indefinitely
	writeTimeout = 30 * time.Second
	idleTimeout  = 120 * time.Second
	// bound for draining the in-flight requests on SIGTERM; comfortably below
	// the usual 30s orchestrator grace period
	shutdownTimeout = 10 * time.Second
)

var (
	bts, rev, version string
)

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Warn("Error loading .env file")
	}
	version = fmt.Sprintf("delphi::%s::%s", bts, rev)
	log.Info("version = ", version)

	// make sure these environment variables are set
	daddr, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		log.Fatal("DWS_DONATION_ADDRESS variable not set")
	}
	// this is the address the users are told to send their ETH to; a stray
	// newline in the secret must not be handed out to every visitor
	daddr = strings.TrimSpace(daddr)
	if !common.IsValidETHAddress(daddr) {
		log.Fatal("DWS_DONATION_ADDRESS is not a valid ethereum address")
	}
	// make sure every consumer reads the normalized value
	if err = os.Setenv("DWS_DONATION_ADDRESS", daddr); err != nil {
		log.Fatalf("failed to set DWS_DONATION_ADDRESS, %v", err)
	}
	_, present = os.LookupEnv("DWS_SALE_PARAMS")
	if !present {
		log.Fatal("DWS_SALE_PARAMS variable not set")
	}

	port := flag.String("port", "8080", "Port for test HTTP server")
	flag.Parse()

	swagger, err := api.GetSwagger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	dsn := common.GetDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	// Create an instance of our handler which satisfies the generated interface
	ds := server.NewDelphiServer(db)

	blv := echomiddleware.BodyLimitConfig{
		Limit: "1K",
	}

	// This is how you set up a basic Echo router
	e := echo.New()
	// echo.New() leaves all server timeouts unset i.e. a slow or stalled client
	// can hold on to a connection forever; the body limit does not help here
	// since it only covers the request body
	setServerTimeouts(e.Server)
	// recover from panics in downstream middleware/handlers; must be first so
	// that it also covers the middleware registered below
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.Logger())

	e.Use(echomiddleware.BodyLimitWithConfig(blv))
	e.Use(echomiddleware.Secure())

	e.Use(middleware.OapiRequestValidator(swagger))

	api.RegisterHandlers(e, ds)

	// listen for interrupt signals (like Ctrl-C).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create an errorgroup.
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		select {
		case <-sigChan:
			log.Info("delphi: received an interrupt, canceling...")
			cancel()
		case <-ctx.Done():
			log.Info("delphi: interrupt handler, canceling...")
			// If context is done, return the context error.
			return ctx.Err()
		}
		return nil
	})

	g.Go(func() error {
		// run the API server
		err := e.Start(net.JoinHostPort("0.0.0.0", *port))
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		log.Error(err)
		return err
	})

	g.Go(func() error {
		// shut down the API server if needed
		<-ctx.Done()
		// The context is canceled
		log.Info("delphi/http - context canceled, stopping..")
		sdc, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := e.Shutdown(sdc); err != nil {
			log.Error(err)
		}
		return ctx.Err()
	})

	// a context cancelation is how an orderly shutdown ends, anything else is
	// a failure the orchestrator needs to see (crash loop backoff, alerting)
	failed := false
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("delphi: errgroup.Wait(): %v", err)
		failed = true
	}
	log.Info("delphi shutting down")
	if failed {
		os.Exit(1)
	}
}

// setServerTimeouts puts an upper bound on how long a single client can occupy
// a connection, in each phase of the request/response cycle.
func setServerTimeouts(s *http.Server) {
	s.ReadHeaderTimeout = readHeaderTimeout
	s.ReadTimeout = readTimeout
	s.WriteTimeout = writeTimeout
	s.IdleTimeout = idleTimeout
}
