package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/pulitzer/data"
)

type PriceReq struct {
	ID     uint64    `db:"id"`
	Asset  string    `db:"what_asset"`
	Time   time.Time `db:"what_time"`
	Status string    `db:"status"`
}

func PersistETHPrice(dbh *sqlx.DB, avp decimal.Decimal) error {
	q := `
		INSERT INTO price(asset, price) VALUES(:asset, :price)
		`
	qd := map[string]interface{}{
		"asset": "eth",
		"price": avp,
	}
	_, err := dbh.NamedExec(q, qd)
	if err != nil {
		log.Errorf("failed to insert ETH price, %v", err)
		return err
	}
	return nil
}

func GetOpenPriceRequests(dbh *sqlx.DB) ([]PriceReq, error) {
	var (
		err    error
		q      string
		result []PriceReq
	)
	q = `
		SELECT id, what_asset, what_time, status
		FROM price_req
		WHERE status='new'
		ORDER BY created_at
		`
	err = dbh.Select(&result, q)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch open price requests, %w", err)
		log.Error(err)
		return nil, err
	}

	return result, nil
}

func CloseRequest(dbh *sqlx.DB, rid uint64, data []data.Kline) error {
	var err error
	// start transaction
	dtx, err := dbh.Beginx()
	if err != nil {
		return err
	}

	// at the end of the function: commit if there are no errors,
	// roll back otherwise
	defer func() {
		if err != nil {
			dtx.Rollback() // nolint:errcheck
		} else {
			dtx.Commit() // nolint:errcheck
		}
	}()

	for _, d := range data {
		err = persistKline(dtx, "eth", d)
		if err != nil {
			return err
		}
	}
	err = closeRequest(dtx, rid)
	if err != nil {
		return err
	}
	return nil
}

func persistKline(dbt *sqlx.Tx, asset string, kl data.Kline) error {
	q := `
		INSERT INTO price(asset, price, created_at)
		VALUES(:asset, :price, :created_at)
		`
	qd := map[string]interface{}{
		"asset":      asset,
		"price":      kl.ClosePrice,
		"created_at": kl.CloseTime,
	}
	_, err := dbt.NamedExec(q, qd)
	if err != nil {
		err = fmt.Errorf("failed to insert historical ETH price for %s, %w", kl.CloseTime, err)
		log.Error(err)
		return err
	}
	return nil
}

func closeRequest(dbt *sqlx.Tx, id uint64) error {
	q := `
		UPDATE price_req SET status='succeeded'
		WHERE id=$1
		`
	_, err := dbt.Exec(q, id)
	if err != nil {
		err = fmt.Errorf("failed to close price request #%d, %w", id, err)
		log.Error(err)
		return err
	}
	return nil
}
