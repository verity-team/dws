//go:build dbtest

// Integration tests for the price request state machine.
//
// These tests need a live postgres database with the dws schema
// (deployments/db/01-schema.sql) loaded, they are therefore hidden behind the
// `dbtest` build tag and are *not* part of a plain `go test ./...` run:
//
//	make run_db
//	go test -tags dbtest -count=1 ./internal/pulitzer/db/...
//
// The connection string defaults to the dockerized development database and
// can be overridden with DWS_TEST_DB_DSN. Please note: the tests truncate the
// price and price_req tables, never point them at a production database.
package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/internal/pulitzer/data"
)

const defaultTestDSN = "host=localhost port=27501 user=postgres password=postgres dbname=dwsdb sslmode=disable"

// testDB connects to the test database and skips the calling test if no
// database is reachable.
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := os.Getenv("DWS_TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	dbh, err := sqlx.Open("postgres", dsn)
	require.NoError(t, err)
	dbh.SetMaxOpenConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbh.PingContext(ctx); err != nil {
		dbh.Close() // nolint:errcheck
		t.Skipf("no test database reachable: %v", err)
	}
	t.Cleanup(func() {
		dbh.Close() // nolint:errcheck
	})

	return dbh
}

// resetDB brings the tables used by these tests into a well defined state.
func resetDB(t *testing.T, dbh *sqlx.DB) {
	t.Helper()

	_, err := dbh.Exec(`TRUNCATE price, price_req RESTART IDENTITY`)
	require.NoError(t, err)
}

// insertPriceReq creates a price request in the given state. modifiedAt is set
// explicitly -- the price_req_update_timestamp trigger only fires on UPDATE,
// so this is the way to age a request for the retry eligibility tests.
func insertPriceReq(t *testing.T, dbh *sqlx.DB, ts time.Time, status string, modifiedAt time.Time) uint64 {
	t.Helper()

	q := `
		INSERT INTO price_req(what_asset, what_time, status, modified_at, created_at)
		VALUES('eth', $1, $2, $3, $3)
		RETURNING id
		`
	var id uint64
	require.NoError(t, dbh.Get(&id, q, ts.UTC(), status, modifiedAt.UTC()))
	return id
}

func requestStatus(t *testing.T, dbh *sqlx.DB, rid uint64) string {
	t.Helper()

	var status string
	require.NoError(t, dbh.Get(&status, `SELECT status FROM price_req WHERE id=$1`, rid))
	return status
}

func priceCount(t *testing.T, dbh *sqlx.DB) int {
	t.Helper()

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM price WHERE asset='eth'`))
	return n
}

func requestIDs(rqs []PriceReq) []uint64 {
	ids := make([]uint64, 0, len(rqs))
	for _, rq := range rqs {
		ids = append(ids, rq.ID)
	}
	return ids
}

// TestCloseRequestRejectsEmptyKlines: a request must not be recorded as
// succeeded when there is nothing to persist.
func TestCloseRequestRejectsEmptyKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	err := CloseRequest(dbh, rid, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no prices to persist")

	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	// the empty slice must be rejected just like the nil slice
	err = CloseRequest(dbh, rid, []data.Kline{})
	require.Error(t, err)
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))
}

// TestCloseRequestRejectsInvalidKlines: prices that would poison the price
// table are rejected and nothing is written.
func TestCloseRequestRejectsInvalidKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	err := CloseRequest(dbh, rid, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts},
		{ClosePrice: decimal.Zero, CloseTime: ts.Add(time.Minute)},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-positive close price")
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	err = CloseRequest(dbh, rid, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5)}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zero close time")
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))
}

// TestCloseRequestPersistsKlines: the happy path writes the prices and marks
// the request succeeded.
func TestCloseRequestPersistsKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	klines := []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts},
		{ClosePrice: decimal.NewFromFloat(2001.25), CloseTime: ts.Add(time.Minute)},
	}
	require.NoError(t, CloseRequest(dbh, rid, klines))

	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	require.Equal(t, 2, priceCount(t, dbh))

	var prices []decimal.Decimal
	require.NoError(t, dbh.Select(&prices,
		`SELECT price FROM price WHERE asset='eth' ORDER BY created_at`))
	require.Len(t, prices, 2)
	require.True(t, prices[0].Equal(klines[0].ClosePrice), "got %s", prices[0])
	require.True(t, prices[1].Equal(klines[1].ClosePrice), "got %s", prices[1])
}

// TestCloseRequestUnknownRequest: closing a request that does not exist must
// not silently succeed, and must not leave prices behind.
func TestCloseRequestUnknownRequest(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	err := CloseRequest(dbh, 4711, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts}})
	require.Error(t, err)
	require.Equal(t, 0, priceCount(t, dbh))

	require.Error(t, CloseRequest(dbh, 0, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts}}))
	require.Equal(t, 0, priceCount(t, dbh))
}

// TestFailRequest: a failure is recorded instead of leaving the request at
// 'new'.
func TestFailRequest(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	require.NoError(t, FailRequest(dbh, rid))
	require.Equal(t, "failed", requestStatus(t, dbh, rid))

	// failing again is allowed (and bumps modified_at, see the retry test)
	require.NoError(t, FailRequest(dbh, rid))
	require.Equal(t, "failed", requestStatus(t, dbh, rid))
}

// TestFailRequestDoesNotDowngradeSucceeded: a request whose prices were
// written must never be turned back into a failure.
func TestFailRequestDoesNotDowngradeSucceeded(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "succeeded", time.Now().UTC())

	require.Error(t, FailRequest(dbh, rid))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))

	// unknown requests are an error as well
	require.Error(t, FailRequest(dbh, 4711))
	require.Error(t, FailRequest(dbh, 0))
}

// TestFailRequestResetsRetryClock: every failure has to bump modified_at,
// otherwise a persistently failing request would be retried on every cycle
// once it became eligible the first time.
func TestFailRequestResetsRetryClock(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "failed", time.Now().UTC().Add(-2*FailedRetryDelay))

	rqs, err := GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.Contains(t, requestIDs(rqs), rid)

	require.NoError(t, FailRequest(dbh, rid))

	rqs, err = GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.NotContains(t, requestIDs(rqs), rid, "the retry clock was not reset")
}

// TestGetOpenPriceRequestsRetryEligibility: a failed request comes back once
// it is old enough -- and not before.
func TestGetOpenPriceRequestsRetryEligibility(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	now := time.Now().UTC()
	ts := now.Truncate(time.Minute)

	newReq := insertPriceReq(t, dbh, ts, "new", now)
	freshFail := insertPriceReq(t, dbh, ts.Add(time.Minute), "failed", now.Add(-FailedRetryDelay/2))
	staleFail := insertPriceReq(t, dbh, ts.Add(2*time.Minute), "failed", now.Add(-FailedRetryDelay-time.Minute))
	succeeded := insertPriceReq(t, dbh, ts.Add(3*time.Minute), "succeeded", now.Add(-24*time.Hour))

	rqs, err := GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	ids := requestIDs(rqs)

	require.Contains(t, ids, newReq, "a new request must be served")
	require.Contains(t, ids, staleFail, "a failed request must become eligible again")
	require.NotContains(t, ids, freshFail, "a failed request must back off before being retried")
	require.NotContains(t, ids, succeeded, "a succeeded request must never be served again")
}

// TestFailedRequestRetrySucceeds: the whole point of the failed state -- a
// transient failure must not wedge the request (and with it buck) forever.
func TestFailedRequestRetrySucceeds(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := time.Now().UTC().Truncate(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	// attempt #1: binance returned nothing
	require.Error(t, CloseRequest(dbh, rid, nil))
	require.NoError(t, FailRequest(dbh, rid))
	require.Equal(t, "failed", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	// age the request so it becomes eligible again
	_, err := dbh.Exec(
		`UPDATE price_req SET status='failed' WHERE id=$1`, rid)
	require.NoError(t, err)
	_, err = dbh.Exec(
		`ALTER TABLE price_req DISABLE TRIGGER price_req_update_timestamp`)
	require.NoError(t, err)
	_, err = dbh.Exec(
		`UPDATE price_req SET modified_at=$2 WHERE id=$1`,
		rid, time.Now().UTC().Add(-FailedRetryDelay-time.Minute))
	require.NoError(t, err)
	_, err = dbh.Exec(
		`ALTER TABLE price_req ENABLE TRIGGER price_req_update_timestamp`)
	require.NoError(t, err)

	rqs, err := GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.Contains(t, requestIDs(rqs), rid)

	// attempt #2 succeeds
	require.NoError(t, CloseRequest(dbh, rid, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(1999.75), CloseTime: ts},
	}))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	require.Equal(t, 1, priceCount(t, dbh))

	rqs, err = GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.NotContains(t, requestIDs(rqs), rid)
}
