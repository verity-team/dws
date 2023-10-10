CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION trigger_update_modified_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = timezone('utc', now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TYPE asset_enum AS ENUM ('eth', 'truth', 'usdc', 'usdt');

--- afc_donation ----------------------------------------------------
DROP TABLE IF EXISTS afc_donation;
CREATE TABLE afc_donation (
    id BIGSERIAL PRIMARY KEY,
    afc VARCHAR(16) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,

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

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_update_timestamp
BEFORE UPDATE ON donation
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

CREATE INDEX ON donation (address);

INSERT INTO donation(address, amount, usd_amount, asset, tokens, price, tx_hash, status) VALUES('0xb938F65DfE303EdF96A511F1e7E3190f69036860', 0.9, 1430.289, 'eth', 715145, 0.002, '0x240abc1e911aba167d1215f8a6b7e8583645a28b1855d0b7d15c70dc7aa9f6cf', 'unconfirmed');
INSERT INTO donation(address, amount, asset, tokens, price, tx_hash, status) VALUES('0xb938F65DfE303EdF96A511F1e7E3190f69036860', 2999, 'usdt', 1499500, 0.002, '0x40bb9bf9753521917c745d8553e9b4b6a6b8a8615c24ad2049508f3c385d83e4', 'confirmed');
INSERT INTO donation(address, amount, asset, tokens, price, tx_hash, status) VALUES('0xb938F65DfE303EdF96A511F1e7E3190f69036860', 2999, 'usdt', 1499500, 0.002, '0xaec43e48ce0af0d8d81462722b406619af4586dcac43bae33be4d0ad4ac848b2', 'confirmed');
INSERT INTO donation(address, amount, asset, tokens, price, tx_hash, status) VALUES('0x379738c60f658601Be79e267e79cC38cEA07c8f2', 999, 'usdt', 499500, 0.002, '0xc4de0d133dd7f67852d377f3553702bcffc188a3c4a35068acf6d74887bf78dd', 'confirmed');
INSERT INTO donation(address, amount, asset, tokens, price, tx_hash, status) VALUES('0x379738c60f658601Be79e267e79cC38cEA07c8f2', 98765432, 'usdc', 49382716000, 0.002, '0x1ff542cbadf3a5f918f2af81b76995089bdef5671bc9c166ecc60e91ceb2e81a', 'failed');
INSERT INTO donation(address, amount, usd_amount, asset, tokens, price, tx_hash, status) VALUES('0x379738c60f658601Be79e267e79cC38cEA07c8f2', 1.8, 2845.368, 'eth', 1429479, 0.002, '0x2636d1d9b9519d8be9520f9b0f342c73405d9512ecd5af643bfc98f69f40e444', 'unconfirmed');
INSERT INTO donation(address, amount, usd_amount, asset, tokens, price, tx_hash, status) VALUES('0x379738c60f658601Be79e267e79cC38cEA07c8f2', 1.8, 2845.368, 'eth', 1429479, 0.002, '0x447b793e048cc61824083005cf28d58ca38635adb0c3e76af63c565cae8877ee', 'confirmed');


--- price ----------------------------------------------------
DROP TABLE IF EXISTS price;
CREATE TABLE price (
    id BIGSERIAL PRIMARY KEY,
    asset asset_enum NOT NULL,
    price NUMERIC(15,5) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE INDEX ON price (created_at);

INSERT INTO price(asset, price) VALUES('truth', 0.002);

--- last_block ----------------------------------------------------
DROP TABLE IF EXISTS last_block;
CREATE TABLE last_block (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL,
    last_block BIGINT NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);

CREATE TRIGGER last_block_update_timestamp
BEFORE UPDATE ON last_block
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

INSERT INTO last_block(chain, last_block) VALUES('eth', 18312345);

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

--- user_stats ----------------------------------------------------
CREATE TYPE user_stats_status_enum AS ENUM ('none', 'staking', 'unstaking');
DROP TABLE IF EXISTS user_stats;
CREATE TABLE user_stats (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    total NUMERIC(12,2) NOT NULL,
    tokens BIGINT NOT NULL,
    staked BIGINT NOT NULL DEFAULT 0,
    reward BIGINT NOT NULL DEFAULT 0,
    status user_stats_status_enum NOT NULL DEFAULT 'none',

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER user_stats_update_timestamp
BEFORE UPDATE ON user_stats
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();
CREATE INDEX ON user_stats (address);

--- update_user_stats() ---------------------------------------------
CREATE OR REPLACE FUNCTION update_user_stats(p_address VARCHAR(42))
RETURNS TABLE (
    us_id BIGINT,
    us_address VARCHAR(42),
    us_total NUMERIC(12, 2),
    us_tokens BIGINT,
    us_staked BIGINT,
    us_reward BIGINT,
    us_status user_stats_status_enum,
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
            FROM user_stats us
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

        -- Update user_stats record with the calculated totals
        INSERT INTO user_stats(address, total, tokens)
        VALUES(p_address, ds_total, ds_tokens)
        ON CONFLICT (address)
        DO UPDATE SET
            total = ds_total,
            tokens = ds_tokens,
            modified_at = NOW()
        WHERE user_stats.address = p_address;
    END IF;

    -- Return the updated user_stats record
    RETURN QUERY
    SELECT *
    FROM user_stats
    WHERE user_stats.address = p_address;
END;
$$ LANGUAGE plpgsql;
