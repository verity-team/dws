# truth memes donation web site

The donation site of the TRUTH token sale. Donors send ETH, USDT or USDC to the
receiving address; the backend detects the donations on chain, prices them
and issues tokens against a price ladder, and the web site shows prices, the
campaign and each donor's history.

## components

| component | what it does |
|---|---|
| `buck` | Ethereum crawler, run as three processes: *latest* (records new donations as `unconfirmed`), *finalized* (confirms them, keeps the campaign totals and the token price) and *old-unconfirmed* (re-examines donations that stay unconfirmed) |
| `pulitzer` | writes the ETH/USD price (median of six exchanges) every minute and backfills the prices `buck` asks for |
| `delphi` | the REST API the web site uses (spec: [`api/delphi.yaml`](api/delphi.yaml)) |
| `dws/` | the web site (Next.js), see [`dws/README.md`](dws/README.md) |
| `cmd/tt` | a small tool that signs `delphi` requests the way the web site does |
| postgres | all state; schema: [`deployments/db/01-schema.sql`](deployments/db/01-schema.sql) |

## running the backend

Requirements: Go >= 1.26, docker, and `golangci-lint` (`make` lints before it
builds).

1. `make` -- builds `bin/buck`, `bin/delphi`, `bin/pulitzer` and `bin/tt`
1. `make run_db` -- starts a fresh postgres in docker, loaded with the schema
1. export the variables from `env.example`; the docker database serves no TLS,
   so also set `DWS_DB_SSLMODE=disable` (see [configuration](docs/configuration.md))
1. in separate terminals: `make run_pulitzer`, `make run_delphi`,
   `make run_buck_latest`, `make run_buck_final`, `make run_buck_old_unconfirmed`

Every service serves `/live` and `/ready`; the port is set with `-port`:

| service | default port |
|---|---:|
| `delphi` (REST API + health) | 8080 |
| `pulitzer` | 8081 |
| `buck` latest | 8082 |
| `buck` finalized | 8083 |
| `buck` old-unconfirmed | 8084 |

The services shut down gracefully on `SIGINT`/`SIGTERM` (in-flight requests and
jobs finish first) and exit non-zero when they fail to start or die on an
error -- including when the database is not reachable at startup, so something
has to restart them.

`make destroy_db` removes the docker database and its volume.

## tests

- `go test ./...` -- unit tests, no database needed
- `go test -tags dbtest -count=1 ./...` -- adds the database integration tests;
  they need a database loaded with the schema (`make run_db`) and **truncate
  its tables, so never point them at a production database**. The connection
  defaults to the docker database; override it with `DWS_TEST_DB_DSN`.

## documentation

- [architecture](docs/architecture.md) -- the crawlers, what counts as a
  donation, totals and limits, token pricing, price aggregation, rate limiting
- [API contract](docs/api.md) -- endpoints, request signing, error codes
- [configuration](docs/configuration.md) -- environment variables, startup
  validation, debug facilities
- [operations](docs/operations.md) -- runbooks: token price at startup,
  pausing the sale, a crawler stuck on a block
- [database schema changes](docs/database-schema-changes.md) -- bringing an
  existing database up to date with the schema (there is no migration runner)
- [frontend behaviour](docs/frontend.md) -- what the web site does against the
  API

## license

[GPL-3.0](LICENSE)
