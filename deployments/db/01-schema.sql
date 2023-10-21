CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION trigger_update_modified_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = timezone('utc', now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TYPE asset_enum AS ENUM ('eth', 'truth', 'usdc', 'usdt');

--- wallet_connection ----------------------------------------------------
DROP TABLE IF EXISTS wallet_connection;
CREATE TABLE wallet_connection (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(16) NOT NULL,
    address VARCHAR(42) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);

--- donation ----------------------------------------------------
CREATE TYPE donation_status_enum AS ENUM ('confirmed', 'unconfirmed', 'failed');
DROP TABLE IF EXISTS donation;
CREATE TABLE donation (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL,
    amount NUMERIC(20,10) NOT NULL,
    usd_amount NUMERIC(12,2),
    asset asset_enum NOT NULL,
    tokens BIGINT NOT NULL,
    price NUMERIC(15,5) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,
    status donation_status_enum NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    block_time TIMESTAMP NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_update_timestamp
BEFORE UPDATE ON donation
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

CREATE INDEX ON donation (address);
CREATE INDEX ON donation (tx_hash);
CREATE INDEX ON donation (block_hash);
CREATE INDEX ON donation (block_time);

--- price ----------------------------------------------------
DROP TABLE IF EXISTS price;
CREATE TABLE price (
    id BIGSERIAL PRIMARY KEY,
    asset asset_enum NOT NULL,
    price NUMERIC(15,5) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
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

INSERT INTO donation_stats(total, tokens) VALUES(0, 0);

--- user_data ----------------------------------------------------
CREATE TYPE user_data_status_enum AS ENUM ('none', 'staking', 'unstaking');
DROP TABLE IF EXISTS user_data;
CREATE TABLE user_data (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    total NUMERIC(12,2) NOT NULL,
    tokens BIGINT NOT NULL,
    staked BIGINT NOT NULL DEFAULT 0,
    reward BIGINT NOT NULL DEFAULT 0,
    status user_data_status_enum NOT NULL DEFAULT 'none',
    affiliate_code VARCHAR(16) NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER user_data_update_timestamp
BEFORE UPDATE ON user_data
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();
CREATE INDEX ON user_data (address);

--- finalized_block ----------------------------------------------------
DROP TABLE IF EXISTS finalized_block;
CREATE TABLE finalized_block (
    id BIGSERIAL PRIMARY KEY,
    base_fee_per_gas VARCHAR(16) NOT NULL,
    gas_limit VARCHAR(16) NOT NULL,
    gas_used VARCHAR(16) NOT NULL,
    block_hash VARCHAR(66) NOT NULL UNIQUE,
    block_number BIGINT NOT NULL,
    receipts_root VARCHAR(66) NOT NULL,
    block_size VARCHAR(16) NOT NULL,
    state_root VARCHAR(66) NOT NULL,
    block_time TIMESTAMP NOT NULL,
    -- enough to keep 400 tx hashes
    transactions VARCHAR(26799) NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER finalized_block_update_timestamp
BEFORE UPDATE ON finalized_block
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

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
    -- Check if there exists at least one confirmed donation record with a more recent modified_at timestamp
    IF EXISTS (
        SELECT 1
        FROM donation d
        WHERE d.address = p_address
          AND d.status = 'confirmed'
          AND d.modified_at > (
            SELECT COALESCE(MAX(us.modified_at), '2000-01-01'::timestamp)
            FROM user_data us
            WHERE us.address = p_address
          )
    ) THEN
        -- Get the sum of total and tokens from all confirmed donation records for the specified address
        SELECT
            -- in case of an ethereum donation we should be summing up the
            -- `usd_amount`; in case of a stable coin donation the `usd_amount`
            -- will be NULL.
            SUM(COALESCE(dtab.usd_amount, dtab.amount)),
            SUM(dtab.tokens)
        INTO
            ds_total,
            ds_tokens
        FROM donation dtab
        WHERE dtab.address = p_address
          AND dtab.status = 'confirmed';

        -- Update user_data record with the calculated totals
        INSERT INTO user_data(address, total, tokens)
        VALUES(p_address, ds_total, ds_tokens)
        ON CONFLICT (address)
        DO UPDATE SET
            total = ds_total,
            tokens = ds_tokens,
            modified_at = NOW()
        WHERE user_data.address = p_address;
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
