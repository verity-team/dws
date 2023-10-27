package db

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"crypto/rand"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"github.com/shopspring/decimal"
	"github.com/verity-team/dws/api"
)

func ConnectWallet(db *sqlx.DB, wc api.ConnectionRequest) error {
	q := `
		INSERT INTO wallet_connection (code, address)
		SELECT :code, :address
		WHERE NOT EXISTS (
		  SELECT 1
		  FROM wallet_connection
		  WHERE
			 code = :code
			 AND address = :address
			 AND id = (
				SELECT id FROM wallet_connection
				WHERE address = :address
				ORDER BY id DESC LIMIT 1)
		)
	 `
	_, err := db.NamedExec(q, wc)
	if err != nil {
		log.Errorf("failed to insert wallet connection data, %v", err)
		return err
	}

	return nil
}

func GetDonationData(db *sqlx.DB) (*api.DonationData, error) {
	// ethereum price
	q1 := `
		SELECT asset, price, created_at FROM price
		WHERE
			asset='eth'
			AND created_at > NOW() - INTERVAL '3 minutes'
		ORDER BY id DESC
		LIMIT 1
		`
	var ethp api.Price
	err := db.Get(&ethp, q1)
	if err != nil {
		err = fmt.Errorf("failed to fetch an ETH price that is newer than 3 minutes, %v", err)
		log.Error(err)
	}

	// truth token price
	q2 := `
		SELECT asset, price, created_at FROM price
		WHERE
			asset='truth'
		ORDER BY created_at DESC
		LIMIT 1
		`
	var truthp api.Price
	err = db.Get(&truthp, q2)
	if err != nil {
		err = fmt.Errorf("failed to fetch the TRUTH price, %v", err)
		log.Error(err)
		return nil, err
	}

	type dstats struct {
		Total  decimal.Decimal `db:"total"`
		Tokens int             `db:"tokens"`
		Status string          `db:"status"`
	}

	// donation stats
	q3 := `
		SELECT total, tokens, status FROM donation_stats
		ORDER BY created_at DESC
		LIMIT 1
		`
	var ds dstats
	err = db.Get(&ds, q3)
	if err != nil {
		err = fmt.Errorf("failed to fetch donation stats, %v", err)
		log.Error(err)
		return nil, err
	}

	var result api.DonationData
	result.Prices = append(result.Prices, ethp)
	result.Prices = append(result.Prices, truthp)

	result.Stats = api.DonationStats{
		Total:  ds.Total.StringFixed(2),
		Tokens: strconv.Itoa(ds.Tokens),
	}

	result.Status = api.DonationDataStatus(ds.Status)
	return &result, nil
}

func GetUserDonationData(db *sqlx.DB, address string) ([]api.Donation, error) {
	// fetch donations made by this user/address
	q1 := `
		SELECT
			amount, usd_amount, asset, tokens, price, tx_hash, status, block_time
		FROM donation
		WHERE address=$1
		ORDER BY id
		`
	var result []api.Donation
	err := db.Select(&result, q1, address)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch donation records for %s, %v", address, err)
		log.Error(err)
		return nil, err
	}

	return result, nil
}

func GetUserData(db *sqlx.DB, address string) (*api.UserData, error) {
	// fetch donations made by this user/address
	q1 := `
		SELECT
			 us_total,
			 us_tokens,
			 us_staked,
			 us_reward,
			 us_status,
			 us_code,
			 us_modified_at
		FROM update_user_data($1)
		`
	var result api.UserData
	err := db.Get(&result, q1, address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		err = fmt.Errorf("failed to fetch user data for %s, %v", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}

func GetAffiliateCode(db *sqlx.DB, address string) (*api.AffiliateCode, error) {
	// fetch donations made by this user/address
	q1 := `
		SELECT
			address, COALESCE(affiliate_code, '') AS code, created_at
		FROM user_data
		WHERE address=$1
		`
	var result api.AffiliateCode
	err := db.Get(&result, q1, address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		err = fmt.Errorf("failed to fetch affiliate code for %s, %v", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}

func genAFC() (string, error) {
	buff := make([]byte, 8)
	_, err := rand.Read(buff)
	if err != nil {
		err = fmt.Errorf("failed to generate an affiliate code, %v", err)
		log.Error(err)
		return "", err
	}
	return hex.EncodeToString(buff), nil
}

func GenerateAffiliateCode(db *sqlx.DB, address string) (*api.AffiliateCode, error) {
	// fetch donations made by this user/address
	q1 := `
		INSERT INTO user_data AS ud(address, affiliate_code)
		VALUES($1, $2)
		ON CONFLICT(address)
		DO UPDATE SET
			 affiliate_code = EXCLUDED.affiliate_code
			 WHERE ud.affiliate_code IS NULL
		RETURNING address, affiliate_code AS code, created_at
		`

	var result api.AffiliateCode
	afc, err := genAFC()
	if err != nil {
		return nil, err
	}
	err = db.QueryRowx(q1, address, afc).StructScan(&result)
	if err != nil {
		err = fmt.Errorf("failed to set affiliate code for '%s', %v", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}
