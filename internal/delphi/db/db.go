package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"github.com/shopspring/decimal"
	"github.com/verity-team/dws/api"
)

// AddAffiliateCD adds a donation made by a user with an affiliate code to the
// database.
func AddAffiliateCD(db *sqlx.DB, afc api.AffiliateRequest) error {
	q := `
		 INSERT INTO afc_donation(afc, tx_hash)
		 VALUES(:afc, :tx_hash)
	 `
	qd := map[string]interface{}{
		"afc":     afc.Code,
		"tx_hash": afc.TxHash,
	}
	_, err := db.NamedExec(q, qd)
	if err != nil {
		log.Errorf("Failed to insert affiliate donation data, %v", err)
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
		err = fmt.Errorf("failed to fetch an ETH price that is newer than 5 minutes, %v", err)
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
			amount, usd_amount, asset, tokens, price, tx_hash, status, modified_at
		FROM donation
		WHERE address=$1
		ORDER BY id
		`
	var result []api.Donation
	err := db.Select(&result, q1, address)
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		err = fmt.Errorf("failed to fetch donation records for %s, %v", address, err)
		log.Error(err)
		return nil, err
	}

	return result, nil
}

func GetUserStats(db *sqlx.DB, address string) (*api.UserStats, error) {
	// fetch donations made by this user/address
	q1 := `
		SELECT
			us_total, us_tokens, us_staked, us_reward, us_staked, us_modified_at
		FROM update_user_stats($1)
		`
	var result api.UserStats
	err := db.Get(&result, q1, address)
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		err = fmt.Errorf("failed to fetch user stats for %s, %v", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}
