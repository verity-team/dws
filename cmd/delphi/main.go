package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/delphi/server"
)

var (
	bts, rev, version string
)

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Warn("Error loading .env file")
	}
	version = fmt.Sprintf("%s::%s", bts, rev)
	log.Info("version = ", version)

	// make sure these environment variables are set
	_, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		log.Fatal("DWS_DONATION_ADDRESS variable not set")
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

	dsn := getDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Create an instance of our handler which satisfies the generated interface
	ds := server.NewDelphiServer(db)

	blv := echomiddleware.BodyLimitConfig{
		Limit: "1K",
	}

	// This is how you set up a basic Echo router
	e := echo.New()
	// Log all requests
	e.Use(echomiddleware.Logger())

	e.Use(echomiddleware.BodyLimitWithConfig(blv))
	e.Use(echomiddleware.Secure())

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	e.Use(middleware.OapiRequestValidator(swagger))

	// We now register our petStore above as the handler for the interface
	api.RegisterHandlers(e, ds)

	// And we serve HTTP until the world ends.
	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", *port)))
}

func getDSN() string {
	var (
		host, port, user, passwd, database string
		present                            bool
	)

	host, present = os.LookupEnv("DWS_DB_HOST")
	if !present {
		log.Fatal("DWS_DB_HOST variable not set")
	}
	port, present = os.LookupEnv("DWS_DB_PORT")
	if !present {
		log.Fatal("DWS_DB_PORT variable not set")
	}
	user, present = os.LookupEnv("DWS_DB_USER")
	if !present {
		log.Fatal("DWS_DB_USER variable not set")
	}
	passwd, present = os.LookupEnv("DWS_DB_PASSWORD")
	if !present {
		log.Fatal("DWS_DB_PASSWORD variable not set")
	}
	database, present = os.LookupEnv("DWS_DB_DATABASE")
	if !present {
		log.Fatal("DWS_DB_DATABASE variable not set")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, passwd, database)

	log.Infof("host: '%s'", host)
	log.Infof("database: '%s'", database)
	return dsn
}
