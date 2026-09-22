-- Force error-stop for the whole script. This is a psql meta-command, so it
-- travels in the file and takes effect however psql is invoked (`psql -f`, or
-- fed on stdin) -- WITHOUT it the guard below RAISEs but psql, lacking
-- ON_ERROR_STOP, prints the error and runs the DROPs anyway, wiping the very
-- data the guard exists to protect. It is idempotent with the docker-init path,
-- which already runs with ON_ERROR_STOP=1.
--
-- LIMITATION: a non-psql loader (a migration tool executing this SQL directly)
-- ignores this meta-command; there the RAISE EXCEPTION below still fires, but
-- whether it aborts the batch is up to that tool.
\set ON_ERROR_STOP on

-- GUARD (must stay the FIRST statement, before any DROP) -----------------
-- This file is a DESTRUCTIVE DROP/CREATE script: every table below is dropped
-- with CASCADE and recreated empty. Docker only ever loads it into an *empty*
-- data directory (deployments/docker/db.yaml mounts it under
-- /docker-entrypoint-initdb.d, and postgres runs init scripts only when the
-- datadir is empty), but a manual `psql -f 01-schema.sql` against a populated
-- database would wipe every donation on record.
--
-- So: refuse to run when donation data is already present, unless the operator
-- deliberately sets the override. The block is a no-op on a fresh database --
-- `to_regclass` returns NULL for a table that does not exist yet, so a first
-- install and the docker-init path both fall straight through it.
--
-- To wipe and reload a populated (dev) database on purpose, set the override in
-- the same psql session, e.g.:
--   psql ... -c "SET dws.allow_destructive_reload='on'" -f deployments/db/01-schema.sql
-- (see README, "database schema changes").
DO $$
DECLARE
    has_data boolean;
BEGIN
    -- a fresh install (and the docker-init path) has no donation table yet:
    -- fall straight through. A bare `SELECT ... FROM donation` here would fail
    -- at *plan* time -- plpgsql plans the whole IF expression before it runs,
    -- so the `to_regclass` short-circuit would not save it -- hence the
    -- existence check returns early and the row count is read with dynamic SQL,
    -- which is only planned once we know the table is there.
    IF to_regclass('public.donation') IS NULL THEN
        RETURN;
    END IF;
    EXECUTE 'SELECT EXISTS (SELECT 1 FROM donation)' INTO has_data;
    IF has_data
       AND lower(coalesce(current_setting('dws.allow_destructive_reload', true), 'off'))
           NOT IN ('on', 'true', '1', 'yes')
    THEN
        RAISE EXCEPTION 'refusing to load deployments/db/01-schema.sql: the database already holds donation data and this script DROPs every table. To wipe and reload on purpose, set dws.allow_destructive_reload, e.g. psql -c "SET dws.allow_destructive_reload=''on''" -f deployments/db/01-schema.sql (see README, "database schema changes").';
    END IF;
END
$$;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION trigger_update_modified_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = timezone('utc', now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TYPE IF EXISTS asset_enum CASCADE;
CREATE TYPE asset_enum AS ENUM ('eth', 'truth', 'usdc', 'usdt');

--- wallet_connection ----------------------------------------------------
DROP TABLE IF EXISTS wallet_connection;
CREATE TABLE wallet_connection (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(16) NOT NULL,
    address VARCHAR(42) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE INDEX ON wallet_connection (address);

--- donation ----------------------------------------------------
DROP TYPE IF EXISTS donation_status_enum CASCADE;
CREATE TYPE donation_status_enum AS ENUM ('confirmed', 'unconfirmed', 'failed');
DROP TABLE IF EXISTS donation;
CREATE TABLE donation (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL,
    amount NUMERIC(20,10) NOT NULL,
    -- the crawler always converts the donated amount to USD (stable coin
    -- donations are denominated in USD to begin with) -> never NULL
    usd_amount NUMERIC(12,2) NOT NULL,
    asset asset_enum NOT NULL,
    tokens BIGINT NOT NULL,
    price NUMERIC(15,5) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,
    status donation_status_enum NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    block_time TIMESTAMP NOT NULL,
    -- number of consecutive old-unconfirmed crawler runs this transaction has
    -- been observed absent from both chain and mempool; reset to 0 the moment
    -- it reappears. A single transient provider `null` (a lagging replica, a
    -- bad batch element) must not fail a donation whose money verifiably
    -- arrived, so a donation is only declared dropped after a *sustained*
    -- absence -- see c.SustainedAbsenceRuns and c.TxByHash.Judge.
    absent_count INTEGER NOT NULL DEFAULT 0,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_update_timestamp
BEFORE UPDATE ON donation
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

CREATE INDEX ON donation (address);
-- NB: no separate index on tx_hash -- the UNIQUE constraint above already
-- creates one (donation_tx_hash_key) and a second copy of it only costs
-- write time and disk.
CREATE INDEX ON donation (block_hash);
CREATE INDEX ON donation (block_time);
-- Serves GetOldUnconfirmed, which looks for `status = 'unconfirmed'` by block
-- time every 15 minutes. That predicate is highly selective -- next to every
-- row is 'confirmed' -- so the planner uses this index and reads only the
-- handful of rows that match instead of the whole table.
--
-- It does *not* help the aggregate updateDonationStats runs while holding the
-- donation_stats row lock. `status = 'confirmed'` matches next to every row,
-- so the planner rightly keeps scanning the table sequentially (measured on
-- 300k rows at 0.3% unconfirmed, pg14: Parallel Seq Scan either way). Making
-- that aggregate cheap needs a different change -- an incrementally
-- maintained total, or a covering index -- and not this one.
CREATE INDEX ON donation (status, block_time);

--- price ----------------------------------------------------
DROP TABLE IF EXISTS price;
CREATE TABLE price (
    id BIGSERIAL PRIMARY KEY,
    asset asset_enum NOT NULL,
    price NUMERIC(15,5) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),

    -- one price per asset and point in time. The historical price requests
    -- filed by `buck` are served with the ten klines that start at the minute
    -- asked for, so two requests a few minutes apart return overlapping
    -- windows and used to insert the same minute twice: identical values, so
    -- reads were unaffected, but the table grows without bound and the
    -- `ORDER BY created_at DESC LIMIT 1` reads resolve the tie arbitrarily.
    -- pulitzer's persistKline inserts ON CONFLICT DO NOTHING against this
    -- constraint. It doubles as the (asset, created_at) index the price
    -- lookups need: GetETHPrice and getTokenPrice both filter by asset and
    -- order/bound by created_at.
    UNIQUE(asset, created_at)
);
CREATE INDEX ON price (created_at);

INSERT INTO price(asset, price) VALUES('truth', 0.001);

--- last_block ----------------------------------------------------
DROP TABLE IF EXISTS last_block;
CREATE TABLE last_block (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL,
    label VARCHAR(16) NOT NULL,
    value BIGINT NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),

    UNIQUE(chain, label)
);

CREATE TRIGGER last_block_update_timestamp
BEFORE UPDATE ON last_block
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

--- donation_stats ----------------------------------------------------
DROP TYPE IF EXISTS donation_stats_status_enum CASCADE;
CREATE TYPE donation_stats_status_enum AS ENUM ('open', 'paused', 'closed');
DROP TABLE IF EXISTS donation_stats;
CREATE TABLE donation_stats (
    id BIGSERIAL PRIMARY KEY,
    total NUMERIC(12,2) NOT NULL,
    tokens BIGINT NOT NULL,
    status donation_stats_status_enum NOT NULL DEFAULT 'open',

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_stats_update_timestamp
BEFORE UPDATE ON donation_stats
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

-- donation_stats holds the campaign totals and the campaign status: exactly
-- one row. The writers (updateDonationStats, setCampaignStatus) never target a
-- row by id and the readers take the first row they find -- a second row would
-- make the two disagree about which one is authoritative.
CREATE UNIQUE INDEX donation_stats_single_row ON donation_stats ((true));

INSERT INTO donation_stats(total, tokens) VALUES(0, 0);

--- user_data ----------------------------------------------------
DROP TYPE IF EXISTS user_data_status_enum CASCADE;
CREATE TYPE user_data_status_enum AS ENUM ('none', 'staking', 'unstaking');
DROP TABLE IF EXISTS user_data;
CREATE TABLE user_data (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    total NUMERIC(12,2) NOT NULL DEFAULT 0.0,
    tokens BIGINT NOT NULL DEFAULT 0,
    staked BIGINT NOT NULL DEFAULT 0,
    reward BIGINT NOT NULL DEFAULT 0,
    status user_data_status_enum NOT NULL DEFAULT 'none',
    -- an affiliate code identifies exactly one user: wallet_connection rows
    -- are created by looking the code up in this table
    affiliate_code VARCHAR(16) UNIQUE,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER user_data_update_timestamp
BEFORE UPDATE ON user_data
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();
-- NB: no separate index on address -- the UNIQUE constraint above already
-- creates one (user_data_address_key) and a second copy of it only costs
-- write time and disk.

--- failed_block ----------------------------------------------------
DROP TABLE IF EXISTS failed_block;
CREATE TABLE failed_block (
    id BIGSERIAL PRIMARY KEY,
    block_number BIGINT NOT NULL UNIQUE,
    block_hash VARCHAR(66) NOT NULL,
    block_time TIMESTAMP NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER failed_block_update_timestamp
BEFORE UPDATE ON failed_block
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

--- failed_tx ----------------------------------------------------
DROP TABLE IF EXISTS failed_tx;
CREATE TABLE failed_tx (
    id BIGSERIAL PRIMARY KEY,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    block_time TIMESTAMP NOT NULL,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER failed_tx_update_timestamp
BEFORE UPDATE ON failed_tx
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

--- price_req ----------------------------------------------------
DROP TYPE IF EXISTS price_req_status_enum CASCADE;
CREATE TYPE price_req_status_enum AS ENUM ('new', 'succeeded', 'failed');
DROP TABLE IF EXISTS price_req;
CREATE TABLE price_req (
    id BIGSERIAL PRIMARY KEY,
    what_asset asset_enum NOT NULL,
    what_time TIMESTAMP NOT NULL,
    status price_req_status_enum NOT NULL DEFAULT 'new',

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    UNIQUE(what_asset, what_time)
);
CREATE TRIGGER price_req_update_timestamp
BEFORE UPDATE ON price_req
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();
-- GetOpenPriceRequests polls every 10s with
--   WHERE status='new' OR (status='failed' AND modified_at < now() - interval)
-- so (status, modified_at) lets the planner satisfy the filter from the index
-- instead of scanning the whole table (see #229). A failed request is retried
-- forever (there is no terminal 'dead' state): an outage self-heals when it
-- ends, and a request that stays unfulfilled too long is surfaced by
-- servePriceRequests at error level for alerting instead.
CREATE INDEX ON price_req (status, modified_at);

--- update_user_data() ---------------------------------------------
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
