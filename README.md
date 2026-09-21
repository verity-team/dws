# truth memes donation web site

## general

We want to build a 1-page web site similar to https://memeinator.com/ -- the donation widget needs to be on top of the page. It shall allow a user to donate ETH, USDT or USDC.

The backend used by the donation web site frontend is called `delphi` and it will expose a [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) to the frontend.

## how to run the dws backend

The `dws` backend consists of a `postgres` database and 5 services
- `buck`: ETH/latest crawler, checks the latest blocks for donation transactions and inserts these into the database (in state `unconfirmed`)
- `buck`: ETH/finalized crawler, checks the finalized blocks for donation transactions and confrm them, also updates the donation campaign statistics and the token price (if/as needed), see [token sale limit](#token-sale-limit)
- `buck`: ETH/old-unconfirmed crawler, checks for donations that are older than 30 minutes but still unconfirmed, attempts to fetch the respective finalized blocks and confirm these donation transactions. A donation is marked as `failed` if its finalized block was fetched and does *not* carry the transaction, or if the transaction is absent from both the chain and the mempool and the block it was originally seen in is more than 24 hours old (it was evicted by a re-org and never re-mined, so no money moved); the donation campaign statistics are then recalculated -- the donation records are retained, they are *not* deleted. Donations whose transaction is still in the mempool, whose block has not been finalized yet, whose finalized block could not be fetched, for which the jsonrpc provider answered with an error object instead of a result (a rate limit or a bad spell is the absence of evidence, not evidence of absence) or that are absent from the chain but younger than 24 hours are left untouched and re-examined on a later run
- `pulitzer`: pulls the ETH price from 6 exchanges and inserts an average price into the database every minute, see [price aggregation](#price-aggregation). It also serves the historical price requests filed by `buck`: a request that cannot be fulfilled is marked `failed` and retried a few minutes later, a request is only marked `succeeded` once prices that actually serve it were written, i.e. at least one of them has to fall inside the same ±1.5 minute window the donation crawler searches for a price -- across an exchange data gap the prices returned can be minutes off, and `succeeded` is terminal
- `delphi`: [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) server -- only serves data from the database

The backend services are written in `go` -- you will thus need `go` on your development system. For testing purposes the `postgres` database can be run in a docker container i.e. you will need docker as well.

Once you have these in place you can simply run
1. `go mod tidy` to fetch the dependencies
1. `make` to build the services
1. `make run_db` to run a fresh database in a docker container
1. `make run_pulitzer` in a separate terminal
1. `make run_delphi` in a separate terminal
1. `make run_buck_latest` in a separate terminal
1. `make run_buck_final` in a separate terminal
1. `make run_buck_old_unconfirmed` in a separate terminal

All services support the `-p` command-line flag allowing you to set the port they are listening to. The default ports are as follows


|service          | port   | content served |
|:-----------------:|------:|--------------------------------------------|
|delphi      | 8080  | REST API + /live and /ready healtcheck endpoints |
|pulitzer      | 8081  | /live and /ready healtcheck endpoints |
|buck/latest      | 8082  | /live and /ready healtcheck endpoints |
|buck/final      | 8083  | /live and /ready healtcheck endpoints |
|buck/old-unconfirmed      | 8084  | /live and /ready healtcheck endpoints |

All services shut down gracefully on `SIGINT`/`SIGTERM` i.e. they stop accepting new work, wait for the requests and the scheduled jobs that are still in flight and only then close the database handle. A service that fails to start or dies on an error terminates with a non-zero exit status.

## configuration

The services are configured via the environment, see `env.example` for the full set of variables. The donation related configuration is validated at startup: the service logs the offending entry and refuses to start when

- `DWS_DONATION_ADDRESS` (`buck`, `delphi`) is not a well-formed ethereum address
- `DWS_STABLE_COINS` (`buck`) is empty or one of its entries has an unknown asset (lower case, one of the assets `buck` holds an erc-20 ABI for), an invalid contract address or a scale that is not greater than zero
- `DWS_SALE_PARAMS` (`buck`) is empty or one of its entries has a token limit or a token price that is not greater than zero

These values used to be accepted as-is and only did damage once donations were processed e.g. a scale of zero leaves a stable coin donation in its smallest unit and turns a 500 USDT donation into a 500,000,000 USD one.

`DWS_MAX_TIMESTAMP_AGE` (`delphi`) sets how old the `delphi-ts` timestamp of a
signed request may be. It defaults to 30 seconds and is capped at 300 seconds:
the replay window must not be widened without bound by a stray environment
variable. Independently of it, a timestamp more than 5 seconds in the *future*
is rejected -- otherwise a signature harvested over a far future timestamp
would stay valid, and replayable, until that timestamp had come and gone.

## token sale limit

Whether the sale can still deliver tokens is derived, on every write, from the confirmed token total read under the `donation_stats` lock compared against `DWS_SALE_PARAMS`' token limit -- *not* from the stored `donation_stats.status`, which only records what the previous statistics write concluded and goes stale the moment the configured limit changes. `donation_stats.status` follows that same comparison, so the campaign is `closed` once the confirmed tokens reach the limit and `open` again when they no longer do. Two things reopen it: the limit is raised, or a finalized re-crawl whose receipt reports an already `confirmed` donation as reverted -- that donation becomes `failed` and drops out of the total. (The old-unconfirmed crawler marking a vanished transaction `failed` does *not*: it only ever transitions `unconfirmed` rows, which were never in the total to begin with.) `paused` is the operator's manual lever and is never touched by a crawler.

While the sale is closed donations are still recorded -- the money arrived and is refundable -- but they issue **0 tokens**. The rule is applied when a donation *enters* the `confirmed` state, not when it is first inserted: the latest crawler runs ~13 minutes ahead of the finalized one, so the donations mined in that window are inserted while the total is still under the limit. Accordingly the latest crawler, which inserts `unconfirmed` rows, never zeroes them -- those rows are not counted towards the campaign anyway, and zeroing them would strand the donation at 0 tokens if the sale reopened before it was confirmed. A donation that is *already* `confirmed` keeps its tokens no matter how often its block is crawled again -- applying the rule is a state transition, not a re-pricing. That cuts both ways: a donation confirmed at 0 tokens while the sale was closed keeps 0 even if the limit is later raised and the sale reopens. Reopening credits the donations confirmed *after* it, not the ones already settled, so raising the limit is not a way to retroactively compensate donors who were mined during the closed window -- they need handling out of band.

## rate limiting

`delphi` applies a per-IP rate limit (50 requests/second, bursts of 100) using
an **in-process, in-memory** store. Two consequences:

1. **The limit is per instance, not global.** Every replica keeps its own
   counters, so `n` replicas admit up to `n` times the configured rate. A
   global budget belongs at the proxy/CDN layer in front of the service; this
   limiter is the last line of defence that travels with the binary.
1. **The visitor is the direct peer address**, not `X-Forwarded-For`. That
   header is attacker controlled, and an identifier taken from it can be
   rotated at will, which would leave the limiter with nothing to limit.

The numbers are deliberately generous because the donation web site talks to
this API from its own (server side) Next.js route handlers: in the normal case
every legitimate request arrives from the handful of frontend egress addresses
rather than from the end users. One browser session costs roughly three
requests a minute (the ETH price once a minute, the user data once a minute,
the donation data on every refresh and whenever "donate" is pressed), so 50
req/s leaves head room for a four digit number of concurrent sessions behind a
single address while still capping what one client talking to an instance
directly can extract. Per-*user* limits have to be applied where the user's own
address is still visible, i.e. in front of the frontend.

`/live`, `/ready` and `/version` are exempt: an orchestrator probe must not be
able to exhaust the budget of the node it runs on, and a probe answered with
429 would take the pod out of service.

## API contract

`GET /user/data/{address}` is unauthenticated, and two properties of it follow
from that:

- **the donation history is paginated.** The `limit` (1..100, default 50) and
  `offset` (>= 0, default 0) query parameters select the page; the records are
  ordered oldest first. The server caps `limit` at 100 no matter what is
  passed, so a single request can no longer turn ~100 bytes of input into
  megabytes of database read and egress.
- **the response no longer carries `affiliate_code`.** Donation addresses are
  public on chain, so anyone could previously walk the donor set into a
  complete off-chain profile *including the referral graph*. The affiliate code
  is now only served over `POST /affiliate/code`, which requires a signature
  from the address it belongs to.

Error responses carry the numeric error code and a fixed, generic message. The
underlying error -- which names tables, functions and columns and gives away
the SQL dialect and the driver version -- is logged server side only; quote the
numeric code when reporting a problem.

The signature in `delphi-signature` is taken over a message built from the
words of the endpoint path, the lower cased address in `delphi-key` and the
`delphi-ts` timestamp, e.g.

```
affiliate code, 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, 2023-10-23 18:45:19+00:00
```

The address is part of the signed message, so a signature is bound to the
account it is used for. **This changed the message format**: the backend and
the frontend have to be rolled out together, a new frontend talking to an old
backend (or the other way round) will see every signed request rejected with a
401.

## price aggregation

`pulitzer` fetches the ETH price from six venues every minute and combines them
as follows:

1. sources that failed, panicked or returned a non-positive price drop out;
1. at least **4** sources have to remain (the quorum). A median needs enough
   inputs to be more than "the middle one of three", and four also keeps the
   two USDT quoted venues (binance, kucoin) from making up the majority of a
   cycle and dragging a stable coin depeg into the written price;
1. the remaining prices are sorted (by price, then by source name) and their
   **median** is taken. Every source is then compared against that single
   reference value, so the outcome is symmetric and does not depend on the
   order the goroutines happened to finish in;
1. a source deviating from the median by more than **10%** is dropped
   *individually* and logged with its deviation. One stale quote no longer
   discards five mutually consistent prices;
1. at least **3** sources have to survive that step, otherwise the cycle
   writes nothing;
1. the price written to the database is the mean of the survivors.

Quotes are also rejected when the venue says they are stale: cex.io and kucoin
publish the time their quote was taken, and a quote older than 2 minutes (or
more than 30 seconds in the future) is discarded -- a cached quote tens of
minutes old is usually still well within 10% of spot and would otherwise be
averaged in at full weight. The other four venues (binance, kraken, bitfinex,
coinbase) do not expose a timestamp at all; there the only freshness bound
available is the request itself, which is made afresh in every cycle under the
HTTP client timeout.

`pulitzer`'s `/ready` probe depends on the database and on nothing else. It
deliberately does not call out to an exchange: a third party being slow, or
answering the probe with a rate limit error, would take an otherwise healthy
pod out of service and burn the very request budget the price fetch needs.

## tests

`go test ./...` runs the unit tests, no database required.

The database integration tests for the `buck` donation statistics, for the `pulitzer` price request state machine and for the `delphi` affiliate code/donation data are hidden behind the `dbtest` build tag since they need a live database with the schema in `deployments/db/01-schema.sql` loaded. They truncate the tables they use, so *never* point them at a production database:

1. `make run_db`
1. `go test -tags dbtest -count=1 ./internal/buck/db/...`
1. `go test -tags dbtest -count=1 ./internal/pulitzer/db/...`
1. `go test -tags dbtest -count=1 ./internal/delphi/db/...`

The connection string defaults to the dockerized development database and can be overridden with `DWS_TEST_DB_DSN`. The three packages share the database, run them one after the other (`go test -tags dbtest -count=1 -p 1 ./...`) -- concurrently they truncate each other's tables.

## requirements & rules
1. all amounts are passed as strings and should be decoded to a `decimal` type to preserve precision
1. users may be sent to our web site via a link that contains an affiliate code e.g.

    ```https://tm.io/donate?afc=AivCuktyds0```

    If such an affiliate code is present then it needs to be passed to the `delphi` backend after a donation transaction. We cannot capture it in the ethereum transaction if the latter involves calling an `erc-20` smart contract -- the `data` property (of the ethereum transaction) holds the arguments of the function invoked on the smart contract.

1. the frontend needs to get the token price from the `dsw` backend upon each refresh

1. the frontend needs to get the ETH price from the `dsw` backend
    1. upon each refresh
    1. once a minute
    1. whenever the user presses the "donate" button and the asset he is donating is ETH

## use cases

### connect web3 wallet (ethereum mainnet) - no donations yet

A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `delphi` backend to get the user data associated with the wallet address
1. if the user has not donated anything the backend will return an empty `json` dict.

### connect web3 wallet (ethereum mainnet) - with donation data
A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `delphi` backend to get the user data associated with the wallet address
1. the data returned by the `delphi` backend should look similar to this:

    ```json
    {
      "donations": [
        {
          "amount": "1.2345600000",
          "asset": "eth",
          "price": "0.00100",
          "status": "confirmed",
          "tokens": "1981414",
          "ts": "2023-10-21T07:38:36Z",
          "tx_hash": "0xf98c0fe5c1bf72cad294c5ba60ca5b62d3d519fe7cc71cc3cdfb064892fa90e2",
          "usd_amount": "1981.41"
        },
        {
          "amount": "0.9876540000",
          "asset": "eth",
          "price": "0.00100",
          "status": "confirmed",
          "tokens": "1586905",
          "ts": "2023-10-21T07:56:12Z",
          "tx_hash": "0x7ddb66c1be4ec06e0edd6c2f4ad8b878b2fb5b8075157f5881f0a7f35c5c7cb4",
          "usd_amount": "1586.90"
        }
      ],
      "user_data": {
        "reward": "0",
        "staked": "0",
        "status": "none",
        "tokens": "3568319",
        "total": "3568.31",
        "ts": "2023-10-21T08:28:38.744069Z"
      }
    }
    ```

    The first `n-1` stanzas are the donations made by the user. The last stanza is a summary.

1. display the data above to the user
1. if he has not staked any tokens allow him to do so
1. if he _has_ staked tokens already allow him to unstake them (takes 7 days from the time the unstaking was requested)

### donate

Display the token price and allow the user to
1. select the asset (ETH, USDT or USDC)
1. specify the amount he wants to donate

Whenever the amount changes: display the amount of tokens corresponding to the donation amount specified.

If the user is donating ETH the token amount needs to be calculated as follows:

    ceiling(DA * EP / TP)

where
- DA = donation amount in ETH
- EP = ethereum price in USD
- TP = token price

If the user is donating in stable coin the formula is simpler:

    ceiling(DA / TP)

After the user presses the "donate" button: construct the ethereum transaction and [send it](https://docs.metamask.io/wallet/how-to/send-transactions/) to the network.

If the user came via an affiliate code - call the `delphi` backend and pass the `txHash` and the affiliate code to it.

### stake funds

### unstake funds
