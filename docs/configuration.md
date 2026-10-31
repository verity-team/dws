# Configuration

The services are configured through the environment; `env.example` has a
working development set.

| variable | used by | meaning |
|---|---|---|
| `DWS_DB_HOST`, `DWS_DB_PORT`, `DWS_DB_USER`, `DWS_DB_PASSWORD`, `DWS_DB_DATABASE` | all | database connection |
| `DWS_DB_SSLMODE` | all | `lib/pq` sslmode, default `require` (the dockerized dev database needs `disable`) |
| `DWS_DONATION_ADDRESS` | `buck`, `delphi` | the receiving address donations are credited to |
| `ETH_RPC_URL` | `buck` | the jsonrpc provider, credentials included |
| `DWS_STABLE_COINS` | `buck` | JSON list of `{asset, address, scale}`, the accepted stable coins |
| `DWS_SALE_PARAMS` | `buck` | JSON list of `{limit, price}`, the token price ladder |
| `DWS_MAX_TIMESTAMP_AGE` | `delphi` | max age of a signed request's `delphi-ts`, seconds (default 30, capped at 300) |
| `DWS_DEBUG_DATA_STORE`, `DWS_BLOCK_CACHE` | `buck` | optional debug directories, see below |
| `DWS_TT_PRIVATE_KEY` | `cmd/tt` | the key the test tool signs requests with |
| `DWS_TEST_DB_DSN` | tests | overrides the database the `dbtest` tests use |

## startup validation

The donation configuration is checked at startup; a service logs the offending
entry and refuses to start when

- `DWS_DONATION_ADDRESS` is not a well-formed ethereum address;
- `DWS_STABLE_COINS` is empty, has two entries with the same contract address
  (compared case-insensitively), or an entry with an unknown asset (lower case,
  one `buck` holds an erc-20 ABI for), an invalid contract address or a scale
  outside 2..18. The scale bound is a plausibility check: it catches a scale of
  1 (a 1 USDC transfer credited as $100,000) but not a wrong-but-plausible 5
  for a 6-decimal coin;
- `DWS_SALE_PARAMS` is empty, or an entry has a limit or price that is not
  greater than zero, a price with more than 5 decimals or one too large for
  `price.price` (`NUMERIC(15,5)`). Sorted by limit, the limits must strictly
  ascend and the prices must not descend: the price on record only ever moves
  up, so a descending price (a transposed digit) would sell every later tier
  at the earlier, higher price. `buck` also refuses to start when the token
  price on record is not one of the tier prices, see
  [token price at startup](operations.md#token-price-at-startup);
- `ETH_RPC_URL` is not `https`. `buck` trusts its provider completely -- a
  plaintext endpoint would let an on-path attacker fabricate or fail
  donations. `http` is allowed for a loopback host only (local dev nodes). A
  redirect from `https` to `http` is refused by every HTTP client, so the
  guarantee holds past the first hop.

Misconfiguration is caught here because it would otherwise only do damage once
donations are processed, e.g. a scale of zero turns a 500 USDT donation into a
500,000,000 USD one.

## database connection

All services open the database through `common.OpenDB`: a service that cannot
connect at startup logs the reason and exits non-zero, with no retry of its
own -- something has to restart it (`restart: always` in compose, a Kubernetes
Deployment). The connection attempt is bounded by `common.DBConnectTimeout`
(10 s), carried in the DSN as `connect_timeout` so that it covers the dial and
the startup handshake of every connection the pool opens; `lib/pq` ignores a
context deadline on the dial.

`common.GetDSN` **quotes every value** it interpolates (escaping `\` and `'`),
so a password containing ` host=elsewhere` or ` dbname=x` cannot reassign
another key. `DWS_DB_SSLMODE` is checked against the modes `lib/pq` supports
when the configuration is read.

## secrets in logs

- `ETH_RPC_URL` carries the provider credentials in its path or query, so only
  its scheme and host are logged or put into an error message; the database
  password is never logged either.
- `common.RedactDBError` withholds a database error unless it comes from an
  origin known to carry nothing of the connection string: a `*pq.Error` (e.g.
  "password authentication failed"), a `net.Error`, `lib/pq`'s fixed sentinels
  (incl. `pq: SSL is not enabled on the server`, the usual first error against
  the dev database), `driver.ErrBadConn`, and TLS/x509 verification failures.
- A rejected `delphi-signature` is logged as a truncated SHA-256 digest: inside
  the timestamp window it is a bearer credential.
- `cmd/tt` reads its key from `DWS_TT_PRIVATE_KEY`, not from a flag, which
  would put it into `argv` (`ps aux`) and the shell history.

## signed request window

`DWS_MAX_TIMESTAMP_AGE` sets how old the `delphi-ts` timestamp of a signed
request may be: default 30 s, capped at 300 s so a stray value cannot widen
the replay window without bound. A timestamp more than 5 s in the *future* is
always rejected -- a signature over a far future timestamp would otherwise stay
replayable until then.

## debug facilities

Both are `buck` only and off unless set; leave them off in production.

- `DWS_DEBUG_DATA_STORE` -- a directory into which `buck` dumps the raw
  jsonrpc responses it works from (`<crawler>-<block>.json`, `txs-*.json`,
  `txr-*.json`), so a crawl can be inspected after the fact.
- `DWS_BLOCK_CACHE` -- a directory in which the finalized crawler keeps the
  finalized blocks it fetched (`fb-<block>.json`) and serves them from later. A
  cached block is only used if its hash, timestamp and ordered transaction
  hashes match the header the provider reports (`eth_getBlockByNumber`
  without bodies), so the cache saves bandwidth, not requests, and cannot
  stand in for a provider that is down. The transaction bodies are not
  re-hashed, so protect the directory like the binary.

For both, read/write failures are logged as warnings and do not affect the
crawl; the directories are created on first use, files are written `0600`,
and they grow with every block.
