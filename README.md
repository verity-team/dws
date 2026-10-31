# truth memes donation web site

## general

We want to build a 1-page web site similar to https://memeinator.com/ -- the donation widget needs to be on top of the page. It shall allow a user to donate ETH, USDT or USDC.

The backend used by the donation web site frontend is called `delphi` and it will expose a [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) to the frontend.

## how to run the dws backend

The `dws` backend consists of a `postgres` database and 5 services
- `buck`: ETH/latest crawler, checks the latest blocks for donation transactions and inserts these into the database (in state `unconfirmed`)
- `buck`: ETH/finalized crawler, checks the finalized blocks for donation transactions and confrm them, also updates the donation campaign statistics and the token price (if/as needed), see [token sale limit](#token-sale-limit)
- `buck`: ETH/old-unconfirmed crawler, checks for donations that are older than 30 minutes but still unconfirmed, attempts to fetch the respective finalized blocks and confirm these donation transactions. A transaction its finalized block carries is confirmed only once its receipt from that block reports success: the status on the donation row came from the latest crawler, possibly from a block that was orphaned since, and the transaction may have reverted in the block it was re-mined into. A donation is marked as `failed` if its finalized block was fetched and does *not* carry the transaction, if the receipt from the finalized block reports a revert, or if the transaction is absent from both the chain and the mempool and the block it was originally seen in is more than 24 hours old **and** it has been observed absent on several consecutive runs (it was evicted by a re-org and never re-mined, so no money moved -- the consecutive-run requirement keeps a single transient provider `null` from failing a donation whose money arrived; a run counts each transaction at most once: a repeated jsonrpc id or transaction hash in the provider's response is ignored, as is a result for a transaction that was not asked for); the donation campaign statistics are then recalculated -- the donation records are retained, they are *not* deleted. Donations whose transaction is still in the mempool, whose block has not been finalized yet, whose finalized block could not be fetched (an error, a `null` result or a response carrying a different block number), for which no receipt from the finalized block could be fetched, for which the jsonrpc provider answered with an error object instead of a result (a rate limit or a bad spell is the absence of evidence, not evidence of absence), that are absent from the chain but younger than 24 hours, or whose absence has not yet persisted across enough consecutive runs are left untouched and re-examined on a later run
- `pulitzer`: pulls the ETH price from 6 exchanges and inserts their median price into the database every minute, see [price aggregation](#price-aggregation). It also serves the historical price requests filed by `buck`, with the price of the requested minute only and only once two venues agree on it within 10%, see [price aggregation](#price-aggregation): a request that cannot be fulfilled is marked `failed` and retried a few minutes later, a request is only marked `succeeded` once prices that actually serve it were written, i.e. at least one of them has to fall inside the same ±1.5 minute window the donation crawler searches for a price -- across an exchange data gap the prices returned can be minutes off, and `succeeded` is terminal
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
- `DWS_STABLE_COINS` (`buck`) is empty, has two entries with the same contract address (compared case-insensitively; the later one used to silently replace the earlier one) or one of its entries has an unknown asset (lower case, one of the assets `buck` holds an erc-20 ABI for), an invalid contract address or a scale outside the plausible range for a USD stable coin (2..18 decimals). The scale bound is a plausibility check, not a correctness guarantee: it rejects gross misconfiguration such as a scale of 1 (which would turn a 1 USDC transfer into a $100,000 credit) but cannot catch a wrong-but-plausible value such as 5 for a 6-decimal coin
- `DWS_SALE_PARAMS` (`buck`) is empty or one of its entries has a token limit or a token price that is not greater than zero, or a token price with more than 5 decimal places or too large for `price.price` (`NUMERIC(15,5)`). Sorted by limit, the tiers must have strictly ascending limits (no duplicates) and non-decreasing prices: the price on record only ever moves up, so a descending price (e.g. a transposed digit) would silently sell every later tier at the earlier, higher price. The column keeps 5 decimals: `0.0000125` used to be stored -- and sold -- as `0.00001` (~25% more tokens per dollar), and a price below `0.000005` as `0`, on which every crawler run panicked. `buck` also refuses to start when the token price on record is not one of the configured tier prices, see [token price at startup](#token-price-at-startup)
- `ETH_RPC_URL` (`buck`) is not an `https` URL. `buck` trusts its json-rpc provider completely -- a plaintext or downgraded endpoint would let an on-path attacker fabricate or fail donations -- so a non-`https` provider is refused at startup. `http` is permitted only for a loopback host (`localhost`/`127.0.0.0/8`/`[::1]`) to keep local dev nodes working. A redirect from `https` to plain `http` is refused by every HTTP client (json-rpc provider and exchanges alike), so the guarantee holds past the first hop

These values used to be accepted as-is and only did damage once donations were processed e.g. a scale of zero leaves a stable coin donation in its smallest unit and turns a 500 USDT donation into a 500,000,000 USD one.

Secrets are kept out of the logs. `ETH_RPC_URL` carries the jsonrpc API
provider credentials in its path or query string, so only its scheme and host
are ever logged or put into an error message -- the same treatment
`common.GetDSN` already gives the database password. A `delphi-signature` that
was rejected is logged as a truncated SHA-256 digest rather than verbatim:
inside the `delphi-ts` window the signature is a bearer credential and anyone
with log access could otherwise replay the request.

The database connection string is handled at the root instead. `common.GetDSN`
**quotes every value** it interpolates, escaping `\` and `'` per `lib/pq`'s
quoted-value syntax. Unquoted -- as it used to be -- `lib/pq` reads a value up
to the first space, so a password containing a space followed by `key=value`
silently *reassigned that key*: a password ending in ` host=elsewhere.invalid`
sent the connection somewhere else and put the tail of the password into the
resulting dial error, and one ending in ` dbname=x` connected to a different
database. Quoting removes the whole class -- every value arrives as the key it
was written for, whatever it contains, and a DSN this repo builds can no
longer be misparsed at all. `DWS_DB_SSLMODE` is checked against the modes
`lib/pq` supports while the configuration is read, so an unsupported one is
refused at startup rather than on some later connection attempt.

All three services open the database through `common.OpenDB`, which makes a
connection at startup and refuses to start if it cannot. **This is a change in
startup behaviour.** A service used to start with the database down and
recover on its first successful query; it now logs the reason and exits
non-zero, with no retry of its own, which assumes something will restart it --
a `restart: always` in compose, or the default restart policy of a Kubernetes
Deployment.

The connection attempt is bounded by `common.DBConnectTimeout` (10 seconds),
carried in the DSN as `connect_timeout`, which `lib/pq` applies to the dial
and to the startup handshake. So an unroutable or blackholed host costs ten
seconds rather than the OS SYN timeout of roughly two minutes. It has to
travel in the DSN rather than in a context: `lib/pq` reaches `dial()` via
`DialOpen` -> `Connector.open(context.Background())` and discards the caller's
context on the way, so a `PingContext` deadline never gets near the dial --
and in the DSN the bound covers every connection the pool opens later as
well, not just the first one.

`common.RedactDBError` sits behind the quoting as defence in depth, on the
assumption that it could be wrong. It passes through the origins whose message
is known to carry nothing of the connection string -- a `*pq.Error` (the
server saying "password authentication failed", "database does not exist"), a
`net.Error` (which names the host and port already logged above), `lib/pq`'s
fixed sentinels including `ErrSSLNotSupported`, `driver.ErrBadConn`, and TLS
or x509 certificate verification failures -- and withholds everything else.
The SSL sentinel matters more than it looks: the default is `sslmode=require`
and the dockerized development database serves no TLS, so `pq: SSL is not
enabled on the server` is the first thing most developers hit, and it needs to
say so.

`cmd/tt`, the little tool that signs a request the way the frontend does,
reads the ECDSA private key from `DWS_TT_PRIVATE_KEY`. It used to be a
`--private-key` command-line flag, which put the raw key into `argv` -- where
every user on the box can read it via `ps aux` -- and into the shell history.

`DWS_MAX_TIMESTAMP_AGE` (`delphi`) sets how old the `delphi-ts` timestamp of a
signed request may be. It defaults to 30 seconds and is capped at 300 seconds:
the replay window must not be widened without bound by a stray environment
variable. Independently of it, a timestamp more than 5 seconds in the *future*
is rejected -- otherwise a signature harvested over a far future timestamp
would stay valid, and replayable, until that timestamp had come and gone.

Two optional debug facilities, both `buck` only and both off unless set; leave
them off in production:

- `DWS_DEBUG_DATA_STORE` names a directory into which `buck` dumps the raw
  jsonrpc responses it works from -- every block it fetches
  (`<crawler>-<block>.json`), the transaction objects and the receipts
  (`txs-*.json`, `txr-*.json`) -- so a crawl can be replayed and inspected
  after the fact. Write failures are logged as warnings and otherwise ignored;
  the crawl itself is not affected.
- `DWS_BLOCK_CACHE` names a directory in which the finalized crawler keeps
  every finalized block it fetched (`fb-<block>.json`) and from which `buck`
  serves later requests for the same block instead of downloading it again. A
  cached copy is never trusted on its own: its block hash, timestamp and
  ordered transaction hashes must match the block header the provider reports
  (`eth_getBlockByNumber` without the transaction bodies), so the cache saves
  bandwidth, not requests, and cannot stand in for a provider that is down. A
  copy that fails to parse or does not match is discarded and the block is
  fetched again. The transaction bodies are not re-hashed, so the directory
  must be as well protected as the binary. Like the debug store, read and
  write failures are logged as warnings and otherwise ignored.

Both directories are created on first use and the files are written `0600`;
the data is what the provider returned, not secrets, but the volume grows with
every block.

## what counts as a donation

A transfer is recorded as a donation when it reaches the receiving address and
carries an amount that survives being written down. A stable coin transfer is
recognized by its erc-20 `transfer` selector (`0xa9059cbb`); the calldata is
matched case-insensitively, the jsonrpc spec does not pin the case of hex data.
The amount is rendered at
the scale of the `donation.amount` column (`NUMERIC(20,10)`); the scale is
derived from the column rather than hardcoded, so it cannot drift from it and
does not depend on the number of decimals the token itself uses. Ethereum
donations are the one exception and stay at 8 decimals: widening them is not
value preserving for the donations already on record.

A transfer whose amount is **zero at that scale** is not recorded at all. Such
a row carried `amount 0`, `usd_amount 0` and `0` tokens -- it recorded no
donation, while every per-address aggregate and the full-table aggregate that
runs under the `donation_stats` lock on every finalized block kept paying for
it. Rejecting only `value: 0x0` would have achieved nothing, since one wei
costs the same 21000 gas and is written down as zero just the same, so the
test is applied after the conversion, to ethereum and stable coin transfers
alike.

Only **top-level transactions** are seen: an ethereum transaction whose `to`
is the receiving address, or a direct `transfer()` call on a configured stable
coin contract whose calldata names the receiving address. A donation made from
a **contract wallet** -- a Safe, an ERC-4337 account, a multicall or any other
contract that forwards the money -- reaches the receiving address as an
internal call, and buck records it **nowhere**: there is neither a `donation`
nor a `failed_tx` row for it. Contract wallet donors are not a supported use
case (WalletConnect will still let such a wallet donate); the money arrives
and a donation like that needs handling out of band.

This is deliberately **not** a minimum donation amount: the only transfers
turned away are the ones that would have been stored as zero anyway. A dust
threshold above zero is a separate, commercial decision.

Tokens are rounded **up** (`ceiling`, see the formulas under [donate](#donate)
below), so a dust donation that records a non-zero amount -- e.g. 1 base unit
of USDC, $0.000001 -- is issued one whole token. This is intended and there is
no minimum donation: the gas a transfer costs is orders of magnitude more than
the token is worth, so dust cannot be farmed for profit (#251).

A transfer whose amount **does not fit the donation columns** is not recorded
as a donation either: it goes to `failed_tx`, like a transfer whose calldata
cannot be parsed. `donation.amount` is `NUMERIC(20,10)` and `usd_amount` is
`NUMERIC(12,2)`, so both hold less than 1e10. postgres does not truncate a
larger value, it refuses it with `numeric field overflow`, and that rolled back
the whole block -- on every run, the latest crawler first and then the
finalized one. For an ERC-20 this costs only gas: `transfer(receivingAddr,
1e17)` from an account with no balance reverts, but it is mined and is
persisted like any other transaction (as `failed`). The amount is checked when
the transaction is filtered (stable coins against both columns, since their
amount *is* the USD amount; ethereum against `amount`), and the USD amount of
an ethereum donation when it is priced at the block's ETH price. A donation
that cannot be priced is recorded in `failed_tx` in the same database
transaction as the rest of the block, so the block is still persisted and the
crawler moves on. No genuine donation comes near 1e10 USD; a `failed_tx` row
is for an operator to look into.

### donation totals ceiling

The totals are bounded the same way: `donation_stats.total` (the campaign
total) and `user_data.total` (the total per address) are `NUMERIC(12,2)` as
well, i.e. at most `9,999,999,999.99` USD. Every single donation fits
`usd_amount`, but their sum need not, and there is no row to reject: the
total is the campaign's (or the address'), not the latest donation's.

- `buck` sums the confirmed donations and checks the result against the
  column before writing it. A campaign total above the ceiling stops the
  statistics write -- and with it the block or the confirmation it belongs to,
  on every run -- with `the confirmed donations total ... USD, which does not
  fit donation_stats.total NUMERIC(12,2)` instead of a bare `numeric field
  overflow`. Nothing is written and nothing is lost; the crawler resumes once
  the column is widened, e.g. `ALTER TABLE donation_stats ALTER COLUMN total
  TYPE NUMERIC(14,2);` (raising the precision of a `NUMERIC` at the same scale
  rewrites nothing).
- `update_user_data()` is SQL and has no such check: an address whose confirmed
  donations exceed the ceiling makes the function fail with `numeric field
  overflow`, and `delphi` answers that address' `/user/data` request with a
  generic error. An explicit check would not change that outcome -- the
  request has to fail either way, and the postgres error already names the
  cause -- so it is not worth a function migration. Widening it means
  widening `user_data.total` and the `NUMERIC(12, 2)` declarations in the
  function (`us_total`, `ds_total`) together.

Reaching either ceiling takes ~1e10 USD of actual donations, so it is not an
attack, merely a limit to know about.

## token sale limit

Whether the sale can still deliver tokens is derived, on every write, from the confirmed token total read under the `donation_stats` lock compared against `DWS_SALE_PARAMS`' token limit -- *not* from the stored `donation_stats.status`, which only records what the previous statistics write concluded and goes stale the moment the configured limit changes. `donation_stats.status` follows that same comparison, so the campaign is `closed` once the confirmed tokens reach the limit and `open` again when they no longer do. Two things reopen it: the limit is raised, or a finalized re-crawl whose receipt reports an already `confirmed` donation as reverted -- that donation becomes `failed` and drops out of the total. (The old-unconfirmed crawler marking a vanished transaction `failed` does *not*: it only ever transitions `unconfirmed` rows, which were never in the total to begin with.) `paused` is the operator's manual lever and is never touched by a crawler.

While the sale is closed donations are still recorded -- the money arrived and is refundable -- but they issue **0 tokens**. The rule is applied when a donation *enters* the `confirmed` state, not when it is first inserted: the latest crawler runs ~13 minutes ahead of the finalized one, so the donations mined in that window are inserted while the total is still under the limit. Accordingly the latest crawler, which inserts `unconfirmed` rows, never zeroes them -- those rows are not counted towards the campaign anyway, and zeroing them would strand the donation at 0 tokens if the sale reopened before it was confirmed. A donation that is *already* `confirmed` keeps its tokens no matter how often its block is crawled again -- applying the rule is a state transition, not a re-pricing. That cuts both ways: a donation confirmed at 0 tokens while the sale was closed keeps 0 even if the limit is later raised and the sale reopens. Reopening credits the donations confirmed *after* it, not the ones already settled, so raising the limit is not a way to retroactively compensate donors who were mined during the closed window -- they need handling out of band.

### token price at startup

Donations are priced at the token price on record (the newest `truth` row in
`price`, see `getTokenPrice`), and the crawlers only ever replace it when the
confirmed total crosses into the next tier. When `buck` starts it therefore
makes sure that price is one of the tiers configured in `DWS_SALE_PARAMS`:

- **no token price on record** (a fresh database): the price of the first tier
  is recorded. The schema used to seed `truth = 0.001` instead, so a ladder
  starting at, say, `0.0025` sold all of tier 1 at `0.001` -- 2.5x the tokens
  -- and nothing noticed;
- **a token price that is not a tier price**: `buck` refuses to start with

  ```
  the token price on record (0.00100) is not the price of any DWS_SALE_PARAMS tier [...]: fix DWS_SALE_PARAMS, or record the price of the current tier -- see 'token price at startup' in the README
  ```

  Either `DWS_SALE_PARAMS` is wrong -- fix it and restart -- or the price on
  record is (the old seed under a different ladder, a ladder changed
  mid-sale). Then:

  - **no donation has been priced yet** (`SELECT COUNT(*) FROM donation;` is
    0): delete the token price rows and restart `buck`, which records the
    first tier's price:

    ```sql
    DELETE FROM price WHERE asset='truth';
    ```

  - **donations exist**: record the price of the tier the confirmed total
    (`SELECT tokens FROM donation_stats;`) falls in -- the first tier whose
    `limit` is greater than it, the last tier if none is -- and restart
    `buck` **after two minutes**: with more than one token price row on
    record a new one only takes effect once it is two minutes old, and until
    then `buck` still reads the old one and refuses to start.

    ```sql
    INSERT INTO price(asset, price) VALUES('truth', <TIER PRICE>);
    ```

    The donations already priced at the wrong price keep it; they need
    handling out of band.

`delphi` serves the newest `truth` row as the token price; on a fresh database
`/donation/data` fails (`no TRUTH price on record`) until `buck` has started
once.

### pausing the sale (manual kill switch)

`paused` is the operator's manual lever to **halt token issuance without stopping the crawlers**. Use it for a live incident -- a pricing bug, a compromised receiving address, a bad `DWS_SALE_PARAMS` deploy, a legal hold -- where you want to keep *recording* donations but stop *crediting* them. Unlike `open`/`closed` it is **never written or cleared by a crawler**; it is set and unset by hand:

```sql
-- halt issuance
UPDATE donation_stats SET status='paused';

-- resume: re-derive the status from the confirmed token total against the
-- limit of the LAST (highest) DWS_SALE_PARAMS entry, which is the sale's token
-- limit (TokenSaleLimit()); substitute that number for <LIMIT>, so a campaign
-- that was sold out when it was paused reports 'closed' again at once
UPDATE donation_stats
SET status = (CASE WHEN tokens >= <LIMIT> THEN 'closed' ELSE 'open' END)::donation_stats_status_enum
WHERE status='paused';
```

The resume recipe re-derives rather than bluntly setting `open` on purpose. Issuance is always correct -- it is derived at runtime from the confirmed `tokens` against `DWS_SALE_PARAMS`, not from the stored status -- but delphi serves `donation_stats.status` straight to the frontend. A bare `SET status='open'` on a campaign that was `closed` when it was paused would show a sold-out sale as `open` (and a donor could see `open` yet be issued 0 tokens) until the next block that actually carries a donation recomputes the status -- which may never come on a quiet, sold-out sale. Substituting the sale's token limit -- the `limit` of the **last (highest)** `DWS_SALE_PARAMS` entry, which is what `TokenSaleLimit()` compares against; the earlier entries are cheaper tiers, not the cap -- for `<LIMIT>` fixes the displayed status immediately; the crawler still re-derives it on every later stats write, so an approximate `<LIMIT>` is self-correcting.

While `status='paused'`:

- the finalized crawler records a freshly seen donation as `unconfirmed` instead of `confirmed`, and `FinalizeTx` (the old-unconfirmed crawler's confirmation path) does not confirm. `updateDonationStats` sums only `confirmed` rows, so `donation_stats` freezes -- **nothing is credited** -- with no special-casing;
- the donation is still **recorded** with its real token count and stays visible to the frontend (delphi serves `donation_stats.status`; the donate form shows `paused`/`closed` to users and refuses to start a donation unless the status is `open`), and `last_block` keeps advancing, so there is no backlog and no re-crawl storm on resume;
- a pause **cannot un-credit a donation**: a re-crawl of an already-`confirmed` donation keeps it `confirmed` (a genuine revert still fails it, exactly as while the sale is open);
- a pause is **not failed by a single transient response, but a sustained absence past the grace period still fails**. A donation parked `unconfirmed` is subject to the old-unconfirmed `Judge`, and its block time keeps ageing during the pause. A donation whose money arrived is not dropped by one stray `null` from a lagging provider node: a transaction is only declared dropped after it has aged past the 24h grace period **and** been observed absent on `SustainedAbsenceRuns` (3) consecutive runs (~45 min). A genuinely dropped transaction -- absent on every poll -- still fails; a transient blip resets the counter and never does.

On resume, the donations parked `unconfirmed` during the pause are credited through the **normal** path: the old-unconfirmed crawler picks them up and `FinalizeTx` confirms them against the campaign state that holds *then* (still subject to the closed-campaign rule above). There is no backfill and no retroactive logic. A pause is a hold, not a cap: donors who paid during it get their tokens when it resumes.

**Token price on resume.** The confirmed total is frozen during a pause, so the token-price tier does not advance while paused. A donation held across a tier boundary therefore confirms on resume at the tier that was in effect **when the pause began** (the cheaper one), not the tier its confirmation time would fall in. This is the same effect as the ~13-minute latest-vs-finalized crawler lag, stretched to the length of the pause; it is intended -- the backlog is sold at the pause-time tier -- and is not re-priced.

**Token price within a block.** The same holds at a smaller scale outside a pause. `PersistTxs` reads the token price once per block and prices every donation in that block with it; the tier only moves afterwards, when the finalized crawler recalculates the donation statistics at the end of the block. A donation -- or a block of donations -- that crosses a tier boundary is therefore priced **entirely at the cheaper tier**; the next tier applies to later blocks only (and only once its price row is two minutes old, see `getTokenPrice`). This is intended and is not re-priced.

**Resume lag.** A donation confirms only once it is older than 30 minutes *and* the old-unconfirmed crawler's next run (every 15 minutes) comes round, so expect a lag of up to ~45 minutes after unpausing before the queued donations are credited. That is acceptable for an emergency lever but worth knowing mid-incident. Each run confirms at most `oldUnconfirmedBatchLimit` (500) donations, oldest first, so that no single `eth_getTransactionByHash` batch grows unbounded and wedges the resume (the [#201](https://github.com/verity-team/dws/issues/201) provider-batch-limit failure mode). A backlog larger than that -- a long pause on a busy campaign -- therefore drains over several runs and takes proportionally longer to credit in full; it always drains, and the oldest donors are credited first.

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
  `offset` (0..10000, default 0) query parameters select the page; the records
  are ordered oldest first. The server caps `limit` at 100 no matter what is
  passed, so a single request can no longer turn ~100 bytes of input into
  megabytes of database read and egress. An `offset` above 10000 is rejected
  with a 400 (code 116 from the request validator, 118 from the handler):
  postgres reads every skipped row, and 100 full pages is far beyond the
  donation history of any real donor.
- **the response no longer carries `affiliate_code`.** Donation addresses are
  public on chain, so anyone could previously walk the donor set into a
  complete off-chain profile *including the referral graph*. The affiliate code
  is now only served over `POST /affiliate/code`, which requires a signature
  from the address it belongs to.

`donations` is always an array. It used to be serialized as `null` for an
address with no donations -- and for any page past the end of the history --
although the spec declares it required and of type array, so the frontend had
to carry a null check before it could iterate. `user_data` is likewise
defaulted to `"0"` amounts and `status: "none"` rather than the empty strings
the zero value produced; `""` is not one of the values the `status` enum
declares.

Error responses carry the numeric error code and a fixed, generic message. The
underlying error -- which names tables, functions and columns and gives away
the SQL dialect and the driver version -- is logged server side only; quote the
numeric code when reporting a problem. The same holds for a request that is
rejected before it reaches a handler: the request validator (code 116, also an
unknown path, answered with a 404, and a known path with an unknown method,
answered with a 405) and the parameter binding of the generated
server (code 117) name the offending input and the schema it failed, so the
client gets `invalid request` and the detail is logged.

The signature in `delphi-signature` is taken over a message built from the
words of the endpoint path, the lower cased address in `delphi-key` and the
`delphi-ts` timestamp, e.g.

```
affiliate code, 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, 2023-10-23 18:45:19+00:00
```

`POST /wallet/connection` appends the affiliate code from the request body,
exactly as sent (`none` without one):

```
wallet connection, 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, 2023-10-23 18:45:19+00:00, none
```

The address is part of the signed message, so a signature is bound to the
account it is used for; on `/wallet/connection` it is bound to the affiliate
code as well, so a harvested signature cannot be replayed within the
`delphi-ts` window with somebody else's code. **Appending the code changed the
`/wallet/connection` message format** (the `/affiliate/code` message is
unchanged): deploy `delphi` and the frontend together -- a frontend that signs
the old message is rejected with a 401 by the new `delphi`, and vice versa.

`POST /wallet/connection` additionally rejects a *self-referral*: an address cannot connect with an affiliate code that belongs to itself (attaching another wallet's code to your own address is the normal referral and is allowed; the signature already prevents attaching a code to an address you do not control). **Binding the address in changed the message format**: the backend and
the frontend have to be rolled out together, a new frontend talking to an old
backend (or the other way round) will see every signed request rejected with a
401.

The authentication headers are bounded by the spec: `delphi-key` is a `0x`
address (40 hex digits), `delphi-ts` 1 to 12 decimal digits and
`delphi-signature` `0x` plus 130 hex digits; anything else is a 400 before the
handler runs. The request line and headers together are capped at 8 KB (the
`net/http` default is 1 MB).

Only the canonical (EIP-2 low `s`) form of a signature is accepted, with a
recovery id of 27/28 or 0/1: `(r, N-s)` with the recovery id flipped recovers
the same address but is answered with a 401. Wallets sign with low `s`.

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
1. the survivors must not be spread wider than **10%** of their own median
   (highest minus lowest), otherwise the cycle writes nothing. The per-source
   gate alone lets two camps through: with four sources split two and two the
   median sits half way between them, so both pass up to ~22% apart -- a USDT
   depeg moving both USDT venues would shift the price by half of that;
1. the price written to the database is the **median** of the survivors, so a
   single source that stays just inside the band does not move it.

Every venue contributes a **spot** quote. binance used to contribute
`weightedAvgPrice`, the volume weighted average over the trailing one minute
window: it structurally lags spot, which makes binance the source most likely
to be dropped by the deviation gate during a fast move -- exactly when the
cycle can least afford to lose one. `lastPrice` from the same endpoint is
apples-to-apples with the other five.

A response that echoes the instrument it answered for is also checked against
the pair that was asked for, so a pair that is renamed, re-listed or mistyped
is rejected rather than averaged into the ethereum price as if it were ETH.
binance, coinbase and cex.io echo the instrument in the body and are validated
with `checkPair`. kraken is the special case: it keys the result by its own
spelling of the pair (`XETHZUSD` for `ETHUSD`), which used to be hardcoded --
the single entry of the result map is taken instead and checked against that
pair under either spelling, and the `error` array kraken populates while still
answering HTTP 200 is reported rather than ignored. kucoin and bitfinex query a
fixed, pair-specific endpoint whose body carries no instrument identifier, so
there is nothing to check against there.

Quotes are also rejected when the venue says they are stale: cex.io and kucoin
publish the time their quote was taken, and a quote older than 2 minutes (or
more than 30 seconds in the future) is discarded -- a cached quote tens of
minutes old is usually still well within 10% of spot and would otherwise be
averaged in at full weight. The other venues offer no timestamp the code acts
on: binance, kraken and coinbase do not expose one at all, and bitfinex's v1
`pubticker` does publish a `timestamp` field but the code does not consult it.
There the only freshness bound available is the request itself, which is made
afresh in every cycle under the HTTP client timeout.

Historical price requests are only ever closed with prices for minutes that
have **finished**. A request whose window reaches into the minute currently in
progress is answered with that minute's kline as it stands so far, and its
close price keeps moving until the minute ends; since `price` now keeps one
row per asset and minute, storing that provisional value would make it the
permanent record and the final price would be dropped as a conflict. Such a
kline is discarded instead. If it was the minute the request asked for, the
request is not served and the existing retry path asks again a few minutes
later, by which time the minute has closed -- a price recorded a few minutes
late is worth more than one recorded wrong, because the USD amount a donation
is credited with is never rewritten.

A historical price is permanent and is needed exactly when the live quorum
failed, so a single venue does not set it. The sources (binance, then kraken,
then coinbase, then bitstamp) are walked in order under one 15 second budget;
kraken only ever returns its latest 720 one-minute candles, so for a minute
older than ~12 hours it cannot answer and binance, coinbase and bitstamp are
the candidates -- bitstamp, with deep history, is what keeps that case from
needing *both* binance and coinbase (binance answers some regions with HTTP
451). Each source that
has a final kline for the requested minute is compared with the ones that had
it before, and the first two that agree within **10%** win -- the earlier
source's price is written, for the requested minute only (the neighbouring
minutes a venue returns alongside it are single-sourced and would shadow the
live price of their minute). A venue that disagrees does not end the walk, the
next one may side with either. If no two sources serve and agree, the request
is failed and retried like any other unfulfilled request. The cost is
availability: with only one historical venue reachable a backfill no longer
succeeds, and every backfilled minute now needs its own request.

`delphi` reports the newest ETH price by `created_at`, not the last row
inserted: a backfill writes a past minute after the live price.

`pulitzer`'s `/ready` probe depends on the database and on nothing else. It
deliberately does not call out to an exchange: a third party being slow, or
answering the probe with a rate limit error, would take an otherwise healthy
pod out of service and burn the very request budget the price fetch needs.

## a crawler is stuck on a block

`buck` advances past a block only with positive evidence for every donation it
found in it, so a jsonrpc API provider that never returns the receipt of a
filtered transaction -- or never returns the block itself -- holds the crawler
on that block: the scheduled-job error listener logs the same error on every
run and the cursor does not move. That is deliberate -- absence of evidence is
not evidence of failure, and skipping a block silently would lose the donations
in it -- but it means a wedged block needs an operator.

The error names the block and the transaction and ends with a fixed suffix
pointing here, so it can be grepped for:

```
block: 4404251 -- no tx receipt for tx '0xabc...' (0 usable receipt(s) for 1 tx(s)); if this persists see 'a crawler is stuck on a block' in the README
```

Look the transaction up on a block explorer first:

- the transaction exists and succeeded -- the provider is broken for this
  request. Fix or replace it (`ETH_RPC_URL`); the crawler retries the block on
  its next run and nothing is lost.
- the provider is permanently unable to answer for this block/transaction and
  there is no replacement -- the block can be skipped by moving the cursor of
  the wedged crawler(s) past it. Set **only** the cursor of the crawler that
  is actually stuck, and verify in `last_block` which one that is:

  ```
  SELECT label, value, modified_at FROM last_block WHERE chain='eth';
  ```

  `value` is the last block a crawler *processed*, so the wedged crawler's row
  sits at 4404250 and its `modified_at` has stopped advancing. The two
  crawlers are rarely wedged together: the finalized crawler trails the latest
  tip by roughly 13 minutes, so if only the latest crawler is stuck, moving the
  finalized cursor as well jumps it past finalized-but-uncrawled blocks --
  the donations in those miss the finalized pass and are only confirmed later
  by the old-unconfirmed crawler, i.e. credited late and at the tier in force
  at that time. Set the stuck cursor to the wedged block so the next run
  resumes at the block after it:

  ```
  bin/buck -set-latest 4404251   # only if label 'latest' is stuck
  bin/buck -set-final 4404251    # only if label 'finalized' is stuck
  ```

  Skipping means the donations of that block are *not recorded* for now. The
  skip is recoverable, not final: once the provider is fixed or replaced the
  block can be re-crawled with `bin/buck -monitor-final -single-block 4404251`
  (`-monitor-latest` for the latest crawler), which processes just that block
  without touching the cursor -- the donation writes are `ON CONFLICT
  (tx_hash)` upserts, so a re-crawl duplicates nothing already on record.
  Until then the money may still have arrived: whoever sent it has a
  transaction hash and needs handling out of band, like the donors of a paused
  campaign above.

The old-unconfirmed crawler has no cursor and cannot be wedged this way; a
transaction whose receipt it cannot fetch is simply re-examined on the next
run.

## tests

`go test ./...` runs the unit tests, no database required.

The database integration tests for the `buck` donation statistics, for the `pulitzer` price request state machine and for the `delphi` affiliate code/donation data are hidden behind the `dbtest` build tag since they need a live database with the schema in `deployments/db/01-schema.sql` loaded. They truncate the tables they use, so *never* point them at a production database:

1. `make run_db`
1. `go test -tags dbtest -count=1 ./internal/buck/db/...`
1. `go test -tags dbtest -count=1 ./internal/pulitzer/db/...`
1. `go test -tags dbtest -count=1 ./internal/delphi/db/...`

The connection string defaults to the dockerized development database and can be overridden with `DWS_TEST_DB_DSN`. The packages share the database and truncate each other's tables, so they serialize themselves: each one's `TestMain` holds a session level advisory lock (`pg_advisory_lock`, see `internal/dbtest`) for the whole package run, and the next one waits for it. `go test -tags dbtest -count=1 ./...` is safe with any `-p`; `-p 1` is no longer required.

A database created before the indexes and the `price` uniqueness constraint
were added to `deployments/db/01-schema.sql` has to be brought up to date, see
[database schema changes](#database-schema-changes).

## database schema changes

`deployments/db/01-schema.sql` is the schema a *fresh* database is created
with; there is no migration runner. The statements below bring an existing
database in line with it.

**`01-schema.sql` is a destructive DROP/CREATE script** -- every table is
dropped with `CASCADE` and recreated empty. It is only ever meant to run
against an empty database: docker loads it through
`/docker-entrypoint-initdb.d` (`deployments/docker/db.yaml`), and postgres runs
init scripts *only* when the data directory is empty. To guard against a manual
`psql -f 01-schema.sql` wiping a populated database, the file forces
`\set ON_ERROR_STOP on` and its first statement **aborts the load when any
table already holds operational data** -- donations, affiliate codes
(`user_data`), wallet connections, prices, crawler positions (`last_block`),
the campaign totals/status (`donation_stats`), failed transactions and
price requests -- and names the tables it found populated. The one row the
script seeds itself (the zero `donation_stats` row) does not count, and on a
fresh database, where the tables do not yet exist, the
guard is a no-op. The `\set` is a psql meta-command, so the
guard aborts however psql is invoked -- with or without an `ON_ERROR_STOP` flag;
without it a bare `RAISE EXCEPTION` would only print, and psql would run the
`DROP`s anyway. **Limitation:** the meta-command only helps when the file is fed
through psql -- a non-psql loader (a migration tool running the SQL directly)
ignores it, and whether the `RAISE` then aborts the batch is up to that tool.
The normal dev reset -- `make destroy_db && make run_db` -- discards the volume
and re-initialises from empty, so it is unaffected. To deliberately wipe and
reload a populated (dev) database in place, set the override in the same psql
session:

```sh
psql ... -c "SET dws.allow_destructive_reload='on'" -f deployments/db/01-schema.sql
```

The statements in the rest of this section, by contrast, are migrations: they
bring an existing populated database in line without dropping anything.

**The two `donation` index statements are safe to run at any time**, with any
version of the binaries running: they add and drop an index and no code path
depends on either.

**The `price` constraint is not.** `persistKline` and the constraint have to
be deployed together, because each half is broken without the other:

- the **new** `pulitzer` against a database that has **not** been migrated
  fails every `persistKline` with `there is no unique or exclusion constraint
  matching the ON CONFLICT specification`. `CloseRequest` rolls back, the
  request is marked `failed` and retried every 5 minutes, and `buck` will not
  advance past a block whose minute has no price;
- adding the constraint while the **old** `pulitzer` is still running wedges
  it on the first overlapping kline window -- the unique violation it has no
  `ON CONFLICT` for -- which is the exact terminal failure this change exists
  to remove.

Neither loses money and both clear themselves once the binary and the schema
agree, but both stall the price feed until they do. So:

1. **stop `pulitzer`** (`buck` and `delphi` may keep running -- see below);
1. run the three `price` statements below;
1. **start the new `pulitzer`**.

`buck` may keep running through this, although it *does* write to `price`: it
inserts the `truth` rows carrying the token price (`updateTokenPrice`, inside
`PersistTxs`' transaction). That is safe here for two reasons. Those rows take
`created_at` from `DEFAULT now()`, so the dedupe `DELETE` -- which only
matches rows sharing an asset *and* a timestamp -- cannot touch them, and the
duplicated historical klines are `eth` rows in any case. And the `ALTER TABLE`
takes an ACCESS EXCLUSIVE lock on `price` alone, so at worst it briefly blocks
that insert; no second table is involved, so there is no deadlock to
construct. `delphi` only reads from `price`.

### donation indexes (any time, any binary)

`donation` gets an index on `(status, block_time)`. It serves
`GetOldUnconfirmed`, which looks for `status = 'unconfirmed'` by block time
every 15 minutes: that predicate matches a handful of rows, so the planner
uses the index instead of reading the table. It does **not** speed up the
aggregate `updateDonationStats` runs under the `donation_stats` lock --
`status = 'confirmed'` matches next to every row, so the planner keeps
scanning sequentially, as it should.

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS donation_status_block_time_idx ON donation (status, block_time);
```

The separate index on `donation (tx_hash)` is redundant -- the `UNIQUE`
constraint on the column already provides `donation_tx_hash_key`, which is
what the `ON CONFLICT (tx_hash)` upserts use. Dropping it only removes write
cost and disk.

```sql
DROP INDEX CONCURRENTLY IF EXISTS donation_tx_hash_idx;
```

### price uniqueness (with `pulitzer` stopped)

`price` gets `UNIQUE(asset, created_at)`. **It will not be created while
duplicates are on record**, and there will be some: the historical price
requests used to be served with the ten klines starting at the minute asked for, so
two requests a few minutes apart used to insert the same minutes twice. Check
first, and only then delete the duplicates -- the values of a duplicated
minute are identical, so keeping the lowest `id` of each group loses nothing:

```sql
SELECT asset, created_at, COUNT(*) FROM price GROUP BY asset, created_at HAVING COUNT(*) > 1;
```

```sql
DELETE FROM price p USING price q WHERE p.asset = q.asset AND p.created_at = q.created_at AND p.id > q.id;
```

```sql
ALTER TABLE price ADD CONSTRAINT price_asset_created_at_key UNIQUE (asset, created_at);
```

The constraint doubles as the `(asset, created_at)` index the price lookups
need: `GetETHPrice` and `getTokenPrice` both filter by asset and order/bound
by `created_at`. `persistKline` inserts `ON CONFLICT DO NOTHING` against it,
so an overlapping kline window yields to the price that is already on record
instead of failing the whole request.

### wallet_connection / user_data indexes (any time, any binary)

`wallet_connection` gets an index on `(address)` -- `mostRecentWalletConnection`
looks a connection up by address -- and the redundant `user_data (address)`
index is dropped: `user_data.address` is `UNIQUE`, so it already has
`user_data_address_key`, and a second copy of it only costs write time and
disk (the same redundancy dropped from `donation (tx_hash)`). Neither is
depended on by any code path, so both are safe at any time, with any version of
the binaries running.

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS wallet_connection_address_idx ON wallet_connection (address);
```

```sql
DROP INDEX CONCURRENTLY IF EXISTS user_data_address_idx;
```

### donation_stats single-row guard (check first)

`donation_stats` gets a unique index on `((true))` that admits **exactly one
row**: the writers (`updateDonationStats`, `setCampaignStatus`) never target a
row by id and the readers take the first row they find, so a second row would
make them disagree about which is authoritative. The index **will not be
created while more than one row is on record**. Check first, and reduce to the
single most recent row (the one the readers already prefer, `ORDER BY
created_at DESC LIMIT 1`) only if the count is greater than one:

```sql
SELECT COUNT(*) FROM donation_stats;
```

```sql
DELETE FROM donation_stats a USING donation_stats b WHERE a.created_at < b.created_at OR (a.created_at = b.created_at AND a.id < b.id);
```

```sql
CREATE UNIQUE INDEX donation_stats_single_row ON donation_stats ((true));
```

### user_data affiliate_code uniqueness (check first)

`user_data.affiliate_code` gets a `UNIQUE` constraint: an affiliate code
identifies exactly one user, and `ConnectWallet`/`GenerateAffiliateCode` rely
on that (the former looks a code up to attach a connection, the latter's
concurrent-writer handling depends on the unique violation). The constraint
**will not be created while a code is shared by two rows**. Codes are random,
so there should be none; a collision is a real conflict that has to be resolved
by hand (which user keeps the code) before the constraint can be added:

```sql
SELECT affiliate_code, COUNT(*) FROM user_data WHERE affiliate_code IS NOT NULL GROUP BY affiliate_code HAVING COUNT(*) > 1;
```

```sql
ALTER TABLE user_data ADD CONSTRAINT user_data_affiliate_code_key UNIQUE (affiliate_code);
```

### donation.usd_amount NOT NULL (check first, with the new `buck`)

`donation.usd_amount` becomes `NOT NULL`: the crawler now denominates every
donation in USD (ethereum at the block's ETH price, stable coins in USD to
begin with), so the column is never NULL going forward. Older stable-coin rows
may carry a NULL where the USD amount equals the token `amount`; backfill those
before tightening the column, or the `ALTER` aborts. Deploy with the new `buck`
(the one that always writes `usd_amount`) so nothing writes a fresh NULL after
the constraint is on:

```sql
SELECT COUNT(*) FROM donation WHERE usd_amount IS NULL;
```

The current crawler always writes `usd_amount` for **every** asset (`calcTokens` converts an ethereum donation at the block's ETH price and a stable coin donation is already USD), so a database written only by `buck` has no NULL rows and the count above is `0` -- nothing to back fill. NULLs can only come from hand-inserted or imported rows. For a **stable coin** row `amount` is already the USD value, so it can be back filled directly; an **ethereum** row's `amount` is in ETH and its USD value depends on the ETH price at its block, which is not recoverable from the row alone -- do **not** apply `ROUND(amount, 2)` to it. Back fill only the stable-coin rows and resolve any ethereum NULLs by hand before adding the constraint:

```sql
-- stable-coin rows: amount is already USD
UPDATE donation SET usd_amount = ROUND(amount, 2)
WHERE usd_amount IS NULL AND asset IN ('usdc', 'usdt');
-- any remaining NULLs are ethereum rows; there must be none left before the ALTER:
SELECT COUNT(*) FROM donation WHERE usd_amount IS NULL;  -- must be 0
```

```sql
ALTER TABLE donation ALTER COLUMN usd_amount SET NOT NULL;
```

### update_user_data() function (with the new `delphi`)

`update_user_data(p_address)` is replaced. The previous version recomputed the
per-user total only when a confirmed donation's `modified_at` was newer than
`user_data.modified_at`; since both are transaction-start timestamps, a poll
that committed after an in-flight confirmation -- or a `user_data` row created
affiliate-code-first -- left the donor's total permanently short. The new
version drops that gate, recomputes from the confirmed donations and writes
only on change. `CREATE OR REPLACE` swaps the body in place; deploy it with the
new `delphi` that reads through it:

```sql
CREATE OR REPLACE FUNCTION update_user_data(p_address VARCHAR(42))
RETURNS TABLE (
    us_id BIGINT,
    us_address VARCHAR(42),
    us_total NUMERIC(12, 2),
    us_tokens BIGINT,
    us_staked BIGINT,
    us_reward BIGINT,
    us_status user_data_status_enum,
    us_code VARCHAR(16),
    us_modified_at TIMESTAMP,
    us_created_at TIMESTAMP
)
AS $$
DECLARE
    ds_total NUMERIC(12, 2);
    ds_tokens BIGINT;
BEGIN
    -- Serialize the recompute per address. Under READ COMMITTED a reader
    -- whose SUM ran before a confirmation committed could block on the row
    -- lock of a faster reader that already saw it, re-check its stale total
    -- against the newer row, overwrite it and return the stale total. With
    -- the lock taken before anything is read, the later reader sums later:
    -- every statement below takes a fresh snapshot (the function is
    -- VOLATILE), which sees what the previous lock holder committed. The lock
    -- is transaction scoped: delphi calls the function in an autocommit
    -- statement, so it is released when that statement commits. The two-key
    -- form namespaces it; it cannot collide with a single-key advisory lock.
    PERFORM pg_advisory_xact_lock(hashtext('update_user_data'), hashtext(p_address));

    -- Recompute the per-address totals from the confirmed donations. There is
    -- deliberately no `modified_at` gate: both timestamps are transaction-start
    -- times, so a poll that commits after an in-flight confirmation would skip
    -- the donation forever, and a user_data row created affiliate-code-first
    -- (newer than an already confirmed donation) would read zero forever.
    --
    -- The recompute-and-write only runs when there is something to write from
    -- or to: at least one confirmed donation for the address, OR an existing
    -- user_data row. The first disjunct keeps /user/data/{arbitrary} -- which
    -- is unauthenticated and callable at ~50/s -- from inserting a row for an
    -- address that never donated (a row-creation/enumeration spam vector). The
    -- second is what still zeroes an existing row when its donations all leave
    -- 'confirmed' after a re-org: no confirmed donation remains, so the first
    -- disjunct is false, but the row must be brought to zero all the same.
    IF EXISTS (
        SELECT 1
        FROM donation d
        WHERE d.address = p_address
          AND d.status = 'confirmed'
    ) OR EXISTS (
        SELECT 1
        FROM user_data us
        WHERE us.address = p_address
    ) THEN
        -- Get the sum of total and tokens from all confirmed donation records for the specified address
        SELECT
            -- `usd_amount` is the donated amount in USD for every asset: the
            -- crawler converts ethereum donations at the ETH price of the
            -- block and stable coin donations are denominated in USD already.
            -- Summing the same column as the campaign totals
            -- (updateDonationStats) keeps the two aggregates consistent.
            COALESCE(SUM(dtab.usd_amount), 0),
            COALESCE(SUM(dtab.tokens), 0)
        INTO
            ds_total,
            ds_tokens
        FROM donation dtab
        WHERE dtab.address = p_address
          AND dtab.status = 'confirmed';

        -- Write the recomputed totals, but only when they actually changed:
        -- the WHERE on the conflict target leaves the row untouched (it does
        -- not error) when nothing moved, so `modified_at` -- served to the API
        -- as `ts` -- is not bumped on every poll.
        INSERT INTO user_data(address, total, tokens)
        VALUES(p_address, ds_total, ds_tokens)
        ON CONFLICT (address)
        DO UPDATE SET
            total = ds_total,
            tokens = ds_tokens
        WHERE user_data.total IS DISTINCT FROM ds_total
           OR user_data.tokens IS DISTINCT FROM ds_tokens;
    END IF;

    -- Return the updated user_data record
    RETURN QUERY
    SELECT
        id,
        address,
        total,
        tokens,
        staked,
        reward,
        status,
        affiliate_code,
        modified_at,
        created_at
    FROM user_data
    WHERE user_data.address = p_address;
END;
$$ LANGUAGE plpgsql;
```

### update_user_data() per-address lock (any time, any binary)

`update_user_data(p_address)` now takes a transaction scoped advisory lock on
the address (`pg_advisory_xact_lock`) before it reads anything. Two concurrent
`/user/data/{address}` reads of the same address used to be able to race under
READ COMMITTED: the one whose `SUM` ran before a confirmation committed could
overwrite the newer total written by the other and return the stale one (it
healed on the next read). The lock serializes the recompute per address, so
the later reader always sums later. Readers of different addresses only wait
for each other on a (rare) hash collision, and the lock is released when the calling statement
commits.

Apply it by running the `CREATE OR REPLACE FUNCTION` statement from
[update_user_data() function](#update_user_data-function-with-the-new-delphi)
above -- it is the current definition, lock included. It is safe at any time,
with any version of the binaries running: the signature and the result are
unchanged, and `CREATE OR REPLACE` swaps the body in place.

### donation.absent_count (migrate BEFORE deploying the new `buck`)

`donation` gains `absent_count`, the number of consecutive old-unconfirmed runs a transaction has been observed absent from both chain and mempool. It is what keeps a single transient provider `null` from failing a donation whose money arrived: a donation is dropped only after it has aged past the 24h grace period **and** been absent on `SustainedAbsenceRuns` (3) consecutive runs. The column is additive and has a constant default, so `ADD COLUMN` is a metadata-only change on PostgreSQL 11+ (no table rewrite). Existing rows backfill to `0`, which simply means "not yet observed absent":

```sql
ALTER TABLE donation ADD COLUMN absent_count INTEGER NOT NULL DEFAULT 0;
```

**Apply this migration before deploying the new `buck`.** The new binary's old-unconfirmed crawler calls `RecordAbsence` -- which reads and writes `absent_count` -- for **every** polled transaction on **every** run, before it judges any of them, and a failure there aborts the whole run. So a new `buck` on an un-migrated schema fails with `pq: column "absent_count" does not exist` on the first transaction of every run: no donation is confirmed or failed until the `ALTER` lands. It is loud in the logs and self-heals the moment the column exists (no corruption, no lost donations -- the finalized crawler is unaffected and `last_block` keeps advancing), but it is a needless outage of the old-unconfirmed path. Migrate first.

Migrate-**before**-deploy is safe with the old `buck` still running: it never reads or writes `absent_count` (every `donation` insert is column-listed, there is no `SELECT *`/`RETURNING *`), so the column is simply ignored until the new binary starts. On that first run the new binary starts counting from `0`, so a transaction already long dropped at migration time takes up to `SustainedAbsenceRuns` more runs (~45 min) to fail -- an additive, safe delay.

### token price seed removed (check BEFORE deploying the new `buck`)

`01-schema.sql` no longer seeds `INSERT INTO price(asset, price)
VALUES('truth', 0.001)`; `buck` records the first tier's price itself, see
[token price at startup](#token-price-at-startup). There is nothing to
migrate, but an existing database still carries the seeded row (or the tier
rows written since), and the new `buck` refuses to start if the price on
record is not one of the `DWS_SALE_PARAMS` tier prices. Check before deploying:

```sql
SELECT price, created_at FROM price WHERE asset='truth' ORDER BY created_at DESC LIMIT 1;
```

If that is one of the configured tier prices -- `0.00100` with the first tier
at `0.001`, as in `env.example` -- nothing needs to be done. Otherwise follow
the runbook in [token price at startup](#token-price-at-startup) *before*
starting the new `buck`: the old `buck` has been pricing donations at that
price all along.

### price_req poll index (v2.0.1, any time, any binary)

`price_req` gets an index on `(status, modified_at)`. The historical backfill
poll runs every 10 seconds --

```sql
SELECT ... FROM price_req WHERE status='new' OR (status='failed' AND modified_at < now() - interval '5 minutes');
```

-- and used to scan the whole table (#229); the index lets the planner satisfy
the filter from it instead. This is the **only** schema change in the v2.0.1
price work: the backfill's fallback sources and its retry-forever behaviour need
no new column or enum value (a failed request stays `failed` and is retried
indefinitely, exactly as before, and a request that has stayed unfulfilled too
long is surfaced by `pulitzer` at error level for alerting rather than tracked
in a column).

It is **additive and safe at any time, with any version of the binaries
running**: no code path depends on the index, it only changes how the poll is
planned. Build it `CONCURRENTLY` so it takes no lock that blocks the poll or the
`price_req` writers. `CREATE INDEX CONCURRENTLY` **cannot run inside a
transaction block**, so run it on its own (psql autocommits each statement by
default -- do not wrap it in `BEGIN`/`COMMIT`):

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS price_req_status_modified_at_idx ON price_req (status, modified_at);
```

Verify it built `valid` -- a `CONCURRENTLY` build that is interrupted leaves an
invalid index that has to be dropped and recreated:

```sql
SELECT indisvalid FROM pg_class c JOIN pg_index i ON i.indexrelid=c.oid WHERE c.relname='price_req_status_modified_at_idx';
```

### failed_tx block NULL-able (migrate BEFORE deploying the new `buck`)

`failed_tx.block_number` and `failed_tx.block_hash` may now be `NULL`: a
transaction that vanished from both chain and mempool (a dropped transaction)
has no block, and it used to be recorded as `block_number = 0, block_hash = ''`
-- block 0 is a real block. `DROP NOT NULL` only relaxes a constraint: it is a
metadata-only change, it rewrites and validates nothing and the old `buck`
keeps working against it (it still writes `0`/`''`):

```sql
ALTER TABLE failed_tx ALTER COLUMN block_number DROP NOT NULL, ALTER COLUMN block_hash DROP NOT NULL;
```

**Apply it before deploying the new `buck`.** The new binary writes `NULL` for
a dropped transaction; on an un-migrated schema that insert fails with `null
value in column "block_number" ... violates not-null constraint`, `FailTx`
rolls back and the old-unconfirmed run stops at that transaction -- on every
run, until the `ALTER` lands. Nothing is lost or corrupted (the donation stays
`unconfirmed` and is failed on the first run after the migration), but it is a
needless stall of the old-unconfirmed path.

Optionally bring the rows already on record in line; `(0, '')` is unambiguous,
the real block 0 has a hash:

```sql
UPDATE failed_tx SET block_number = NULL, block_hash = NULL WHERE block_number = 0 AND block_hash = '';
```

### failed_block and pgcrypto removed (optional, any time, any binary)

`01-schema.sql` no longer creates the `failed_block` table or the `pgcrypto`
extension. Nothing ever wrote `failed_block` (failed transactions go to
`failed_tx`) and nothing calls a `pgcrypto` function. An existing database
does not need to change: no binary refers to either, so leaving them in place
is harmless. Dropping them is safe at any time, with any version of the
binaries running:

```sql
DROP TABLE IF EXISTS failed_block;
DROP EXTENSION IF EXISTS pgcrypto;
```

`failed_block` should be empty; check with `SELECT COUNT(*) FROM failed_block;`
first if in doubt. Without `CASCADE`, `DROP EXTENSION` refuses to run if
anything else in the database depends on `pgcrypto` -- in that case keep it.

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
