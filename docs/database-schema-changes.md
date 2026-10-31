# Database schema changes

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

## donation indexes (any time, any binary)

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

## price uniqueness (with `pulitzer` stopped)

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

## wallet_connection / user_data indexes (any time, any binary)

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

## donation_stats single-row guard (check first)

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

## user_data affiliate_code uniqueness (check first)

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

## donation.usd_amount NOT NULL (check first, with the new `buck`)

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

## update_user_data() function (with the new `delphi`)

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

## update_user_data() per-address lock (any time, any binary)

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

## donation.absent_count (migrate BEFORE deploying the new `buck`)

`donation` gains `absent_count`, the number of consecutive old-unconfirmed runs a transaction has been observed absent from both chain and mempool. It is what keeps a single transient provider `null` from failing a donation whose money arrived: a donation is dropped only after it has aged past the 24h grace period **and** been absent on `SustainedAbsenceRuns` (3) consecutive runs. The column is additive and has a constant default, so `ADD COLUMN` is a metadata-only change on PostgreSQL 11+ (no table rewrite). Existing rows backfill to `0`, which simply means "not yet observed absent":

```sql
ALTER TABLE donation ADD COLUMN absent_count INTEGER NOT NULL DEFAULT 0;
```

**Apply this migration before deploying the new `buck`.** The new binary's old-unconfirmed crawler calls `RecordAbsence` -- which reads and writes `absent_count` -- for **every** polled transaction on **every** run, before it judges any of them, and a failure there aborts the whole run. So a new `buck` on an un-migrated schema fails with `pq: column "absent_count" does not exist` on the first transaction of every run: no donation is confirmed or failed until the `ALTER` lands. It is loud in the logs and self-heals the moment the column exists (no corruption, no lost donations -- the finalized crawler is unaffected and `last_block` keeps advancing), but it is a needless outage of the old-unconfirmed path. Migrate first.

Migrate-**before**-deploy is safe with the old `buck` still running: it never reads or writes `absent_count` (every `donation` insert is column-listed, there is no `SELECT *`/`RETURNING *`), so the column is simply ignored until the new binary starts. On that first run the new binary starts counting from `0`, so a transaction already long dropped at migration time takes up to `SustainedAbsenceRuns` more runs (~45 min) to fail -- an additive, safe delay.

## token price seed removed (check BEFORE deploying the new `buck`)

`01-schema.sql` no longer seeds `INSERT INTO price(asset, price)
VALUES('truth', 0.001)`; `buck` records the first tier's price itself, see
[token price at startup](operations.md#token-price-at-startup). There is nothing to
migrate, but an existing database still carries the seeded row (or the tier
rows written since), and the new `buck` refuses to start if the price on
record is not one of the `DWS_SALE_PARAMS` tier prices. Check before deploying:

```sql
SELECT price, created_at FROM price WHERE asset='truth' ORDER BY created_at DESC LIMIT 1;
```

If that is one of the configured tier prices -- `0.00100` with the first tier
at `0.001`, as in `env.example` -- nothing needs to be done. Otherwise follow
the runbook in [token price at startup](operations.md#token-price-at-startup) *before*
starting the new `buck`: the old `buck` has been pricing donations at that
price all along.

## price_req poll index (v2.0.1, any time, any binary)

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

## failed_tx block NULL-able (migrate BEFORE deploying the new `buck`)

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

## failed_block and pgcrypto removed (optional, any time, any binary)

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
