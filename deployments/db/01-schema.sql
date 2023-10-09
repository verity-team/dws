CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION trigger_update_modified_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = timezone('utc', now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TABLE IF EXISTS afc_donation;
CREATE TABLE afc_donation (
    id BIGSERIAL PRIMARY KEY,
    afc VARCHAR(16) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);

DROP TABLE IF EXISTS donation;
CREATE TABLE donation (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL,
    amount NUMERIC(20,10) NOT NULL,
    usd_amount NUMERIC(12,2),
    asset VARCHAR(8) NOT NULL,
    tokens BIGINT NOT NULL,
    price NUMERIC(12,2) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    status VARCHAR(16) NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_update_timestamp
BEFORE UPDATE ON donation
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

DROP TABLE IF EXISTS price;
CREATE TABLE price (
    id BIGSERIAL PRIMARY KEY,
    asset VARCHAR(16) NOT NULL,
    price NUMERIC(12,2) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);

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

DROP TABLE IF EXISTS donation_stats;
CREATE TABLE donation_stats (
    id BIGSERIAL PRIMARY KEY,
    total NUMERIC(12,2) NOT NULL,
    tokens BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER donation_stats_update_timestamp
BEFORE UPDATE ON donation_stats
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();

DROP TABLE IF EXISTS user_stats;
CREATE TABLE user_stats (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL,
    total NUMERIC(12,2) NOT NULL,
    tokens BIGINT NOT NULL,
    staked BIGINT NOT NULL,
    reward BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL,

    modified_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now()),
    created_at TIMESTAMP NOT NULL DEFAULT timezone('utc', now())
);
CREATE TRIGGER user_stats_update_timestamp
BEFORE UPDATE ON user_stats
FOR EACH ROW
EXECUTE PROCEDURE trigger_update_modified_at();
