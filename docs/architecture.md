# Architecture

## services

The backend is a `postgres` database and five processes: three `buck`
crawlers, `pulitzer` and `delphi`.

- **`buck` latest crawler** -- checks the latest blocks for donation
  transactions and inserts them as `unconfirmed`.
- **`buck` finalized crawler** -- checks the finalized blocks, confirms the
  donations in them and updates the campaign statistics and, if a tier
  boundary was crossed, the token price; see [token sale limit](#token-sale-limit).
- **`buck` old-unconfirmed crawler** -- every 15 minutes, re-examines donations
  that are older than 30 minutes and still `unconfirmed`, against their
  finalized block:
  - **confirmed** when the finalized block carries the transaction **and** its
    receipt from that block reports success (the status on the row came from
    the latest crawler, possibly from a block that was orphaned since, and the
    transaction may have reverted in the block it was re-mined into);
  - **failed** when the finalized block was fetched and does *not* carry the
    transaction, when the receipt from the finalized block reports a revert,
    or when the transaction is absent from both the chain and the mempool, the
    block it was first seen in is more than 24 hours old **and** it was
    observed absent on several consecutive runs (it was evicted by a re-org
    and never re-mined, so no money moved; the consecutive-run requirement
    keeps a single transient provider `null` from failing a donation whose
    money arrived). A run counts each transaction at most once: a repeated
    jsonrpc id or transaction hash in the provider's response is ignored, as
    is a result for a transaction that was not asked for. The campaign
    statistics are then recalculated; donation rows are never deleted;
  - **left untouched** and re-examined on a later run in every other case --
    still in the mempool, block not finalized yet, finalized block not
    fetchable (an error, a `null` result or a different block number), no
    receipt from the finalized block, an error object instead of a result
    from the provider (a rate limit is the absence of evidence, not evidence
    of absence), absent but younger than 24 hours, or not yet absent on
    enough consecutive runs.
- **`pulitzer`** -- writes the median ETH price of six exchanges every minute
  and serves the historical price requests `buck` files when it has no price
  for a block; see [price aggregation](#price-aggregation). A request that
  cannot be fulfilled is marked `failed` and retried a few minutes later; it
  is only marked `succeeded` (terminal) once a price that serves it -- inside
  the ±1.5 minute window the crawler searches -- was written.
- **`delphi`** -- the REST API the frontend uses; it only serves data from the
  database. The spec is `api/delphi.yaml`, see [api.md](api.md).

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

A transfer whose amount is **zero at that scale** is not recorded: the row
would credit nothing while every per-address aggregate and the campaign
aggregate (run under the `donation_stats` lock on every finalized block) kept
paying for it. The check runs after the conversion, to ethereum and stable
coin transfers alike, so one wei -- written down as zero -- is caught as well
as `value: 0x0`.

Only **top-level transactions** are seen: an ethereum transaction whose `to`
is the receiving address, or a direct `transfer()` call on a configured stable
coin contract whose calldata names the receiving address. A donation made from
a **contract wallet** -- a Safe, an ERC-4337 account, a multicall or any other
contract that forwards the money -- reaches the receiving address as an
internal call, and buck records it **nowhere**: there is neither a `donation`
nor a `failed_tx` row for it. Contract wallet donors are not a supported use
case (WalletConnect will still let such a wallet donate); the money arrives
and a donation like that needs handling out of band.

There is **no minimum donation**: only transfers that would be stored as zero
are turned away. Tokens are rounded **up** (see
[token amounts](frontend.md#token-amounts)), so a dust donation that records a
non-zero amount -- e.g. 1 base unit of USDC, $0.000001 -- is issued one whole
token. This is intended: the gas a transfer costs is orders of magnitude more
than the token is worth, so dust cannot be farmed for profit.

A transfer whose amount **does not fit the donation columns** is not recorded
as a donation either: it goes to `failed_tx`, like a transfer whose calldata
cannot be parsed. `donation.amount` is `NUMERIC(20,10)` and `usd_amount` is
`NUMERIC(12,2)`, so both hold less than 1e10. postgres refuses a larger value
(`numeric field overflow`) rather than truncating it, which would abort the
whole block on every run -- and for an ERC-20 that costs an attacker only gas:
`transfer(receivingAddr, 1e17)` from an account with no balance reverts, but it
is mined and persisted like any other transaction (as `failed`). The amount is checked when
the transaction is filtered (stable coins against both columns, since their
amount *is* the USD amount; ethereum against `amount`), and the USD amount of
an ethereum donation when it is priced at the block's ETH price. A donation
that cannot be priced is recorded in `failed_tx` in the same database
transaction as the rest of the block, so the block is still persisted and the
crawler moves on. No genuine donation comes near 1e10 USD; a `failed_tx` row
is for an operator to look into.

## donation totals ceiling

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

**Token price within a block.** `PersistTxs` reads the token price once per block and prices every donation in that block with it; the tier only moves afterwards, when the finalized crawler recalculates the donation statistics at the end of the block. A donation -- or a block of donations -- that crosses a tier boundary is therefore priced **entirely at the cheaper tier**; the next tier applies to later blocks only (and only once its price row is two minutes old, see `getTokenPrice`). This is intended and is not re-priced.
A donation held across a tier boundary during a pause is priced the same way,
see [pausing the sale](operations.md#pausing-the-sale-manual-kill-switch).

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
   *individually* and logged with its deviation, so one stale quote does not
   discard five mutually consistent prices;
1. at least **3** sources have to survive that step, otherwise the cycle
   writes nothing;
1. the survivors must not be spread wider than **10%** of their own median
   (highest minus lowest), otherwise the cycle writes nothing. The per-source
   gate alone lets two camps through: with four sources split two and two the
   median sits half way between them, so both pass up to ~22% apart -- a USDT
   depeg moving both USDT venues would shift the price by half of that;
1. the price written to the database is the **median** of the survivors, so a
   single source that stays just inside the band does not move it.

Every venue contributes a **spot** quote; for binance that is `lastPrice`,
not `weightedAvgPrice`, the trailing one minute average, which lags spot and
would make binance the source most likely to be dropped during a fast move.

A response that echoes the instrument it answered for is also checked against
the pair that was asked for, so a pair that is renamed, re-listed or mistyped
is rejected rather than averaged into the ethereum price as if it were ETH.
binance, coinbase and cex.io echo the instrument in the body and are validated
with `checkPair`. kraken is the special case: it keys the result by its own
spelling of the pair (`XETHZUSD` for `ETHUSD`), so the single entry of the
result map is taken and checked against that pair under either spelling, and the `error` array kraken populates while still
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
close price keeps moving until the minute ends; since `price` keeps one row
per asset and minute, storing that provisional value would make it the
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
availability: with only one historical venue reachable a backfill does not
succeed, and every backfilled minute needs its own request.

`delphi` reports the newest ETH price by `created_at`, not the last row
inserted: a backfill writes a past minute after the live price.

`pulitzer`'s `/ready` probe depends on the database and on nothing else. It
deliberately does not call out to an exchange: a third party being slow, or
answering the probe with a rate limit error, would take an otherwise healthy
pod out of service and burn the very request budget the price fetch needs.

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
