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

	// Per-IP rate limit. This is an *in-process, per-instance* budget, not a
	// global one -- see the README. The numbers are deliberately generous: the
	// donation web site talks to this API from its own (server side) Next.js
	// route handlers, so in the normal case every legitimate request arrives
	// from the handful of frontend egress addresses rather than from the end
	// users. A single browser session costs roughly 3 requests a minute (the
	// ETH price once a minute, the user data once a minute, the donation data
	// on every refresh and on "donate"), so 50 req/s sustained leaves head
	// room for a four digit number of concurrent sessions behind one address
	// while still capping what a single client talking to this instance
	// directly can extract.
	rateLimit  = 50
	rateBurst  = 100
	rateExpiry = 3 * time.Minute
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

	// common.OpenDB proves the connection works before the server starts
	// serving; the error it returns is safe to log, see common.RedactDBError
	db, err := common.OpenDB(common.GetDSN())
	if err != nil {
		log.Fatalf("delphi: %v", err)
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
	e.Use(echomiddleware.RateLimiterWithConfig(rateLimiterConfig()))

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

// rateLimiterConfig builds the per-IP rate limiter.
//
// The visitor is identified by the *direct* peer address and not by
// echo.Context.RealIP(): the latter trusts the X-Forwarded-For and X-Real-IP
// request headers, which any internet client can set to an arbitrary value, so
// an attacker could rotate the header and never be limited at all.
//
// The health check endpoints are exempt -- an orchestrator probing this
// instance must not be able to exhaust the budget of the node it runs on, and
// a probe that is rejected with 429 would take the pod out of service.
func rateLimiterConfig() echomiddleware.RateLimiterConfig {
	store := echomiddleware.NewRateLimiterMemoryStoreWithConfig(
		echomiddleware.RateLimiterMemoryStoreConfig{
			Rate:      rateLimit,
			Burst:     rateBurst,
			ExpiresIn: rateExpiry,
		})
	return echomiddleware.RateLimiterConfig{
		Store: store,
		Skipper: func(c echo.Context) bool {
			switch c.Request().URL.Path {
			case "/live", "/ready", "/version":
				return true
			}
			return false
		},
		IdentifierExtractor: func(c echo.Context) (string, error) {
			addr := c.Request().RemoteAddr
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				// no port in the address -- use it as it is
				return addr, nil
			}
			return host, nil
		},
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
