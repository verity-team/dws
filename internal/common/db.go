package common

import (
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
)

func GetLatestETHPrice(db *sqlx.DB) (decimal.Decimal, error) {
	// ethereum price, must not be older than 3 minutes
	q1 := `
		SELECT price FROM price
		WHERE
			asset='eth'
			AND created_at > NOW() - INTERVAL '3 minutes'
		ORDER BY id DESC
		LIMIT 1
		`
	var ethp decimal.Decimal
	if err := db.Get(&ethp, q1); err != nil {
		err = fmt.Errorf("failed to fetch an ETH price that is newer than 3 minutes, %w", err)
		log.Error(err)
		return decimal.Zero, err
	}
	return ethp, nil
}

func GetETHPrice(db *sqlx.DB, ts time.Time) (decimal.Decimal, error) {
	// find a price that is in a 3 minute interval of the timestamp and closest
	// to the timestamp
	q := `
		SELECT price
		FROM price
			WHERE asset = 'eth'
			  AND created_at >= $1::timestamp AT TIME ZONE 'UTC' - INTERVAL '1.5 minutes'
			  AND created_at <= $1::timestamp AT TIME ZONE 'UTC' + INTERVAL '1.5 minutes'
			ORDER BY ABS(EXTRACT(EPOCH FROM (created_at - $1::timestamp AT TIME ZONE 'UTC'))) ASC
			LIMIT 1
		`
	var ethp decimal.Decimal
	if err := db.Get(&ethp, q, ts); err != nil {
		err = fmt.Errorf("failed to fetch ETH price for time %v, %w", ts, err)
		log.Error(err)
		return decimal.Zero, err
	}
	return ethp, nil
}

func GetDonationStats(db *sqlx.DB) (decimal.Decimal, decimal.Decimal, error) {
	// donation stats
	q3 := `
		SELECT total, tokens FROM donation_stats
		ORDER BY created_at DESC
		LIMIT 1
		`
	var ds api.DonationStats
	err := db.Get(&ds, q3)
	if err != nil {
		err = fmt.Errorf("failed to fetch donation stats, %w", err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	total, err := decimal.NewFromString(ds.Total)
	if err != nil {
		err = fmt.Errorf("invalid total, %w", err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	tokens, err := decimal.NewFromString(ds.Tokens)
	if err != nil {
		err = fmt.Errorf("invalid tokens, %w", err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}

	return total, tokens, nil
}

func GetDSN() string {
	var (
		host, port, user, passwd, database string
		present                            bool
	)

	host, present = os.LookupEnv("DWS_DB_HOST")
	if !present {
		log.Fatal("DWS_DB_HOST variable not set")
	}
	port, present = os.LookupEnv("DWS_DB_PORT")
	if !present {
		log.Fatal("DWS_DB_PORT variable not set")
	}
	user, present = os.LookupEnv("DWS_DB_USER")
	if !present {
		log.Fatal("DWS_DB_USER variable not set")
	}
	passwd, present = os.LookupEnv("DWS_DB_PASSWORD")
	if !present {
		log.Fatal("DWS_DB_PASSWORD variable not set")
	}
	database, present = os.LookupEnv("DWS_DB_DATABASE")
	if !present {
		log.Fatal("DWS_DB_DATABASE variable not set")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC", host, port, user, passwd, database)

	log.Infof("host: '%s'", host)
	log.Infof("database: '%s'", database)
	return dsn
}
