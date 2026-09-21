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

// FailedRetryDelay is the minimum amount of time a price request that failed
// has to sit still before it becomes eligible for another attempt.
//
// servePriceRequests runs every 10 seconds; without this delay a request that
// fails persistently (e.g. because binance has no klines for the minute in
// question) would be retried on every single cycle.
const FailedRetryDelay = 5 * time.Minute

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
	if _, err := dbh.NamedExec(q, qd); err != nil {
		log.Errorf("failed to insert ETH price, %v", err)
		return err
	}
	return nil
}

// GetOpenPriceRequests returns the price requests that are waiting to be
// served: the ones that were never attempted ('new') and the ones that failed
// long enough ago to be worth another attempt.
//
// Failed requests have to become eligible again: a request that is stuck in a
// non-'new' state can never be re-created (RequestPrice inserts with
// ON CONFLICT DO NOTHING), and buck refuses to advance past a block whose
// minute has no price.
func GetOpenPriceRequests(dbh *sqlx.DB) ([]PriceReq, error) {
	var (
		err    error
		q      string
		result []PriceReq
	)
	q = `
		SELECT id, what_asset, what_time, status
		FROM price_req
		WHERE
			status='new'
			OR (
				status='failed'
				AND modified_at < timezone('utc', NOW()) - $1::interval
			)
		ORDER BY created_at
		`
	err = dbh.Select(&result, q, retryInterval())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch open price requests, %w", err)
		log.Error(err)
		return nil, err
	}

	return result, nil
}

// retryInterval renders FailedRetryDelay as a postgres interval literal.
func retryInterval() string {
	return fmt.Sprintf("%d seconds", int64(FailedRetryDelay.Seconds()))
}

// CloseRequest persists the prices obtained for a price request and marks the
// request as succeeded.
//
// A request is only ever marked succeeded if prices were actually written:
// an empty kline set is rejected outright, so the function cannot record
// success for work it did not do.
func CloseRequest(dbh *sqlx.DB, rid uint64, data []data.Kline) (err error) {
	if err = validateKlines(rid, data); err != nil {
		return err
	}

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
			return
		}
		if cerr := dtx.Commit(); cerr != nil {
			err = fmt.Errorf("failed to commit price request #%d, %w", rid, cerr)
			log.Error(err)
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

// FailRequest records that a price request could not be served. The failure is
// thus visible instead of the request silently sitting at 'new' forever, and
// the request becomes eligible for another attempt FailedRetryDelay later.
//
// Every failure rewrites the row, so the price_req_update_timestamp trigger
// bumps modified_at: a request that keeps failing keeps backing off instead of
// being picked up on every cycle.
func FailRequest(dbh *sqlx.DB, rid uint64) error {
	if rid == 0 {
		err := errors.New("refusing to fail price request: invalid request id 0")
		log.Error(err)
		return err
	}
	q := `
		UPDATE price_req SET status='failed'
		WHERE id=$1 AND status<>'succeeded'
		`
	res, err := dbh.Exec(q, rid)
	if err != nil {
		err = fmt.Errorf("failed to mark price request #%d as failed, %w", rid, err)
		log.Error(err)
		return err
	}
	if err = expectOneRow(res, fmt.Sprintf("mark price request #%d as failed", rid)); err != nil {
		return err
	}
	log.Warnf("price request #%d marked as failed, retry in %v", rid, FailedRetryDelay)
	return nil
}

// validateKlines rejects kline sets that must not be persisted. An empty set
// is the important case: it would otherwise close the request with zero prices
// written.
func validateKlines(rid uint64, kls []data.Kline) error {
	if rid == 0 {
		err := errors.New("refusing to close price request: invalid request id 0")
		log.Error(err)
		return err
	}
	if len(kls) == 0 {
		err := fmt.Errorf("refusing to close price request #%d: no prices to persist", rid)
		log.Error(err)
		return err
	}
	for _, kl := range kls {
		if !kl.ClosePrice.IsPositive() {
			err := fmt.Errorf(
				"refusing to close price request #%d: non-positive close price '%s'",
				rid, kl.ClosePrice.String())
			log.Error(err)
			return err
		}
		if kl.CloseTime.IsZero() {
			err := fmt.Errorf("refusing to close price request #%d: zero close time", rid)
			log.Error(err)
			return err
		}
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
	res, err := dbt.NamedExec(q, qd)
	if err != nil {
		err = fmt.Errorf("failed to insert historical ETH price for %s, %w", kl.CloseTime, err)
		log.Error(err)
		return err
	}
	return expectOneRow(res, fmt.Sprintf("insert historical ETH price for %s", kl.CloseTime))
}

func closeRequest(dbt *sqlx.Tx, id uint64) error {
	q := `
		UPDATE price_req SET status='succeeded'
		WHERE id=$1
		`
	res, err := dbt.Exec(q, id)
	if err != nil {
		err = fmt.Errorf("failed to close price request #%d, %w", id, err)
		log.Error(err)
		return err
	}
	return expectOneRow(res, fmt.Sprintf("close price request #%d", id))
}

// expectOneRow turns a statement that affected no row into an error: without
// it a vanished or ineligible row would be reported as a success.
func expectOneRow(res sql.Result, what string) error {
	rows, err := res.RowsAffected()
	if err != nil {
		err = fmt.Errorf("failed to check the result of '%s', %w", what, err)
		log.Error(err)
		return err
	}
	if rows != 1 {
		err = fmt.Errorf("failed to %s: %d rows affected", what, rows)
		log.Error(err)
		return err
	}
	return nil
}
