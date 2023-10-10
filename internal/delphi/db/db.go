package db

import (
	"fmt"
	"strconv"
	"time"

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
	type price struct {
		Asset string          `db:"asset"`
		Price decimal.Decimal `db:"price"`
		TS    time.Time       `db:"created_at"`
	}

	// etereum price
	q1 := `
		SELECT asset, price, created_at FROM price
		WHERE
			asset='eth'
			AND created_at > NOW() - INTERVAL '3 minutes'
		ORDER BY id DESC
		LIMIT 1
		`
	var ethp price
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
	var truthp price
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
	p := api.Price{
		Asset: api.PriceAssetEth,
		Price: ethp.Price.StringFixed(2),
		Ts:    ethp.TS,
	}
	log.Info(truthp)
	result.Prices = append(result.Prices, p)
	p = api.Price{
		Asset: api.PriceAssetTruth,
		Price: truthp.Price.StringFixed(3),
		Ts:    truthp.TS,
	}
	result.Prices = append(result.Prices, p)

	result.Stats = api.DonationStats{
		Total:  ds.Total.StringFixed(2),
		Tokens: strconv.Itoa(ds.Tokens),
	}

	result.Status = api.DonationDataStatus(ds.Status)
	return &result, nil
}
