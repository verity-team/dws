# Operations

Runbooks for running the backend. Bringing an existing database up to date
with the schema is covered in [database schema changes](database-schema-changes.md).

## token price at startup

Donations are priced at the token price on record (the newest `truth` row in
`price`, see `getTokenPrice`), and the crawlers only ever replace it when the
confirmed total crosses into the next tier. When `buck` starts it therefore
makes sure that price is one of the tiers configured in `DWS_SALE_PARAMS`:

- **no token price on record** (a fresh database): the price of the first tier
  is recorded;
- **a token price that is not a tier price**: `buck` refuses to start with

  ```
  the token price on record (0.00100) is not the price of any DWS_SALE_PARAMS tier [...]: fix DWS_SALE_PARAMS, or record the price of the current tier -- see 'token price at startup' in docs/operations.md
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

## pausing the sale (manual kill switch)

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

**Resume lag.** A donation confirms only once it is older than 30 minutes *and* the old-unconfirmed crawler's next run (every 15 minutes) comes round, so expect a lag of up to ~45 minutes after unpausing before the queued donations are credited. That is acceptable for an emergency lever but worth knowing mid-incident. Each run confirms at most `oldUnconfirmedBatchLimit` (500) donations, oldest first, so that no single `eth_getTransactionByHash` batch grows unbounded and wedges the resume (the [#201](https://github.com/verity-team/dws/issues/201) provider-batch-limit failure mode). A backlog larger than that -- a long pause on a busy campaign -- therefore drains over several runs and takes proportionally longer to credit in full; it always drains, and the oldest donors are credited first.

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
block: 4404251 -- no tx receipt for tx '0xabc...' (0 usable receipt(s) for 1 tx(s)); if this persists see 'a crawler is stuck on a block' in docs/operations.md
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
