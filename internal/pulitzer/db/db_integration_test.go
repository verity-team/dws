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
	c "github.com/verity-team/dws/internal/common"
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

// closedMinute returns a minute that has fully elapsed, which is what most
// historical price requests ask for by the time they are served.
//
// Not all of them: RequestPrice rounds the block time to the nearest minute
// and the latest crawler processes blocks that are seconds old, so a request
// can name the minute in progress or even the next one. That case is the
// subject of TestCloseRequestRejectsAStillOpenKline. The fixtures here want
// the other one -- a minute whose price is settled -- so that they exercise
// the validation or the persistence they were written for rather than
// finalKlines dropping a kline whose close price is still moving.
func closedMinute() time.Time {
	return time.Now().UTC().Truncate(time.Minute).Add(-30 * time.Minute)
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

// closeReq calls CloseRequest with the wall clock captured now -- what
// servePriceRequests passes for every request. The minute finality tests that
// depend on a specific now (TestCloseRequestRejectsAStillOpenKline,
// TestCloseRequestServesTheRequestedMinuteDespiteAnOpenTail) capture it
// themselves and call CloseRequest directly.
func closeReq(dbh *sqlx.DB, rid uint64, ts time.Time, kls []data.Kline) error {
	return CloseRequest(dbh, rid, ts, kls, time.Now().UTC())
}

// TestCloseRequestRejectsEmptyKlines: a request must not be recorded as
// succeeded when there is nothing to persist.
func TestCloseRequestRejectsEmptyKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	err := closeReq(dbh, rid, ts, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no prices to persist")

	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	// the empty slice must be rejected just like the nil slice
	err = closeReq(dbh, rid, ts, []data.Kline{})
	require.Error(t, err)
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))
}

// TestCloseRequestRejectsInvalidKlines: prices that would poison the price
// table are rejected and nothing is written.
func TestCloseRequestRejectsInvalidKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	err := closeReq(dbh, rid, ts, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts},
		{ClosePrice: decimal.Zero, CloseTime: ts.Add(time.Minute)},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-positive close price")
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	err = closeReq(dbh, rid, ts, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5)}})
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

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	klines := []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts},
		{ClosePrice: decimal.NewFromFloat(2001.25), CloseTime: ts.Add(time.Minute)},
	}
	require.NoError(t, closeReq(dbh, rid, ts, klines))

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

	ts := closedMinute()
	err := closeReq(dbh, 4711, ts, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts}})
	require.Error(t, err)
	require.Equal(t, 0, priceCount(t, dbh))

	require.Error(t, closeReq(dbh, 0, ts, []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts}}))
	require.Equal(t, 0, priceCount(t, dbh))
}

// TestFailRequest: a failure is recorded instead of leaving the request at
// 'new'.
func TestFailRequest(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
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

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "succeeded", time.Now().UTC())

	require.Error(t, FailRequest(dbh, rid))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))

	// unknown requests are an error as well
	require.Error(t, FailRequest(dbh, 4711))
	require.Error(t, FailRequest(dbh, 0))
}

// TestFailRequestNeverBecomesTerminal: a request that no source can serve is
// retried forever -- it never reaches a terminal failure state that would wedge
// buck permanently. Failing it many times over always leaves it 'failed', and
// once it has aged past FailedRetryDelay it is handed back to the retry poll.
func TestFailRequestNeverBecomesTerminal(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	// well past any hypothetical cap: the status stays 'failed' throughout
	for i := 0; i < 50; i++ {
		require.NoError(t, FailRequest(dbh, rid))
		require.Equal(t, "failed", requestStatus(t, dbh, rid), "attempt %d", i)
	}

	// aged past the retry delay, it is eligible again -- not abandoned
	_, err := dbh.Exec(`ALTER TABLE price_req DISABLE TRIGGER price_req_update_timestamp`)
	require.NoError(t, err)
	_, err = dbh.Exec(`UPDATE price_req SET modified_at=$2 WHERE id=$1`,
		rid, time.Now().UTC().Add(-FailedRetryDelay-time.Minute))
	require.NoError(t, err)
	_, err = dbh.Exec(`ALTER TABLE price_req ENABLE TRIGGER price_req_update_timestamp`)
	require.NoError(t, err)

	rqs, err := GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.Contains(t, requestIDs(rqs), rid, "a failed request must always come back for another attempt")
}

// TestFailRequestResetsRetryClock: every failure has to bump modified_at,
// otherwise a persistently failing request would be retried on every cycle
// once it became eligible the first time.
func TestFailRequestResetsRetryClock(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
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

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	// attempt #1: binance returned nothing
	require.Error(t, closeReq(dbh, rid, ts, nil))
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
	require.NoError(t, closeReq(dbh, rid, ts, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(1999.75), CloseTime: ts},
	}))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	require.Equal(t, 1, priceCount(t, dbh))

	rqs, err = GetOpenPriceRequests(dbh)
	require.NoError(t, err)
	require.NotContains(t, requestIDs(rqs), rid)
}

// TestCloseRequestRejectsKlinesOutsideTheLookupWindow: #219 regression.
// GetHistoricalPriceFromBinance asks for the 10 klines starting at the
// requested minute; across a binance data gap the first one returned opens
// minutes later. Recording the request as 'succeeded' with those prices is
// terminal -- RequestPrice cannot re-create the request and only 'failed'
// requests are retried -- while c.GetETHPrice still finds nothing for the
// block, so the finalized crawler loops on it forever. The request has to fail
// instead, so the existing retry path applies.
func TestCloseRequestRejectsKlinesOutsideTheLookupWindow(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	// the data gap: the first kline binance returns opens five minutes later
	var klines []data.Kline
	for i := 5; i < 10; i++ {
		klines = append(klines, data.Kline{
			ClosePrice: decimal.NewFromFloat(2000.5),
			CloseTime:  ts.Add(time.Duration(i)*time.Minute + 59*time.Second),
		})
	}
	err := closeReq(dbh, rid, ts, klines)
	require.Error(t, err)
	require.Contains(t, err.Error(), "within")

	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh), "no price may be written for a request that is not served")

	// ... and the request is retried rather than being wedged forever
	require.NoError(t, FailRequest(dbh, rid))
	require.Equal(t, "failed", requestStatus(t, dbh, rid))

	// c.GetETHPrice agrees: there is no price for the requested minute
	_, err = c.GetETHPrice(dbh, ts)
	require.Error(t, err)
}

// ... a kline that does fall inside the window serves the request, and the
// price it wrote is the one c.GetETHPrice returns.
func TestCloseRequestAcceptsKlinesInsideTheLookupWindow(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	// the kline covering the requested minute closes at :59 -- inside the
	// window -- the later ones are outside it
	klines := []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts.Add(59 * time.Second)},
		{ClosePrice: decimal.NewFromFloat(2001.5), CloseTime: ts.Add(time.Minute + 59*time.Second)},
	}
	require.NoError(t, closeReq(dbh, rid, ts, klines))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))

	price, err := c.GetETHPrice(dbh, ts)
	require.NoError(t, err)
	require.True(t, price.Equal(klines[0].ClosePrice), "got %s", price)
}

// the window the coverage check uses is the one c.GetETHPrice searches: a
// kline exactly at the edge is accepted, one just past it is not.
func TestCloseRequestCoverageMatchesTheLookupWindow(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	kline := func(offset time.Duration) []data.Kline {
		return []data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts.Add(offset)}}
	}

	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())
	require.NoError(t, closeReq(dbh, rid, ts, kline(c.PriceLookupWindow)))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	_, err := c.GetETHPrice(dbh, ts)
	require.NoError(t, err, "a kline at the edge of the window must be findable")

	resetDB(t, dbh)
	rid = insertPriceReq(t, dbh, ts, "new", time.Now().UTC())
	require.Error(t, closeReq(dbh, rid, ts, kline(c.PriceLookupWindow+time.Second)))
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))

	// ... and the same on the other side of the requested minute
	resetDB(t, dbh)
	rid = insertPriceReq(t, dbh, ts, "new", time.Now().UTC())
	require.Error(t, closeReq(dbh, rid, ts, kline(-c.PriceLookupWindow-time.Second)))
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))
}

// a request without a time is refused: the coverage check has nothing to
// measure against and must not be skipped.
func TestCloseRequestRejectsZeroRequestTime(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	rid := insertPriceReq(t, dbh, ts, "new", time.Now().UTC())

	err := closeReq(dbh, rid, time.Time{},
		[]data.Kline{{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zero request time")
	require.Equal(t, "new", requestStatus(t, dbh, rid))
	require.Equal(t, 0, priceCount(t, dbh))
}

// GetHistoricalPriceFromBinance returns the ten klines that start at the
// minute a request asks for, so the windows of two requests a few minutes
// apart overlap. Without UNIQUE(asset, created_at) the overlapping minutes
// were simply inserted a second time and the table grew without bound; with
// it, the insert has to yield to the row that is already there instead of
// failing the request -- which, since the price stays on record, would fail
// on every retry for good.
func TestCloseRequestToleratesOverlappingKlines(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	base := closedMinute()

	// the first request: minutes 0..3
	first := insertPriceReq(t, dbh, base, "new", time.Now().UTC())
	require.NoError(t, closeReq(dbh, first, base, klineWindow(base, 4)))
	require.Equal(t, "succeeded", requestStatus(t, dbh, first))
	require.Equal(t, 4, priceCount(t, dbh))

	// the second request, two minutes later: minutes 2..5, i.e. minutes 2
	// and 3 are already on record
	second := insertPriceReq(t, dbh, base.Add(2*time.Minute), "new", time.Now().UTC())
	require.NoError(t, closeReq(dbh, second, base.Add(2*time.Minute), klineWindow(base.Add(2*time.Minute), 4)))
	require.Equal(t, "succeeded", requestStatus(t, dbh, second))

	// six distinct minutes, not eight rows
	require.Equal(t, 6, priceCount(t, dbh))

	var dupes int
	require.NoError(t, dbh.Get(&dupes, `
		SELECT COUNT(*) FROM (
			SELECT asset, created_at FROM price
			GROUP BY asset, created_at HAVING COUNT(*) > 1
		) d`))
	require.Zero(t, dupes, "the price table must not carry duplicate (asset, created_at) rows")
}

// klineWindow builds `n` consecutive one-minute klines starting at ts.
func klineWindow(ts time.Time, n int) []data.Kline {
	kls := make([]data.Kline, 0, n)
	for i := 0; i < n; i++ {
		kls = append(kls, data.Kline{
			ClosePrice: decimal.NewFromFloat(2000).Add(decimal.NewFromInt(int64(i))),
			CloseTime:  ts.Add(time.Duration(i) * time.Minute),
		})
	}
	return kls
}

// the constraint itself: no two prices for the same asset and point in time
func TestPriceIsUniquePerAssetAndTime(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	ts := closedMinute()
	_, err := dbh.Exec(`INSERT INTO price(asset, price, created_at) VALUES('eth', 2000.5, $1)`, ts)
	require.NoError(t, err)

	_, err = dbh.Exec(`INSERT INTO price(asset, price, created_at) VALUES('eth', 2001.5, $1)`, ts)
	require.Error(t, err, "a second price for the same asset and minute has to be rejected")

	// a different asset at the same instant, and the same asset at a
	// different instant, are both fine
	_, err = dbh.Exec(`INSERT INTO price(asset, price, created_at) VALUES('truth', 0.001, $1)`, ts)
	require.NoError(t, err)
	_, err = dbh.Exec(`INSERT INTO price(asset, price, created_at) VALUES('eth', 2001.5, $1)`, ts.Add(time.Minute))
	require.NoError(t, err)
}

// a kline for the minute that is currently in progress carries a provisional
// close price. Under UNIQUE(asset, created_at) the row that gets there first
// is the one that stays, so storing a provisional price would make it the
// permanent record for that minute and discard the final one.
func TestKlineIsFinal(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 30, 45, 0, time.UTC)

	// the minute in progress: binance reports its close time as the last
	// instant of 12:30, truncated to the second by parseKlines. Its minute
	// ends at 12:31:00.
	open := data.Kline{CloseTime: time.Date(2026, 9, 21, 12, 30, 59, 0, time.UTC)}
	require.False(t, klineIsFinal(open, now))

	// the minute before it ends at 12:30:00 and, now being well past that plus
	// the margin, has closed
	closed := data.Kline{CloseTime: time.Date(2026, 9, 21, 12, 29, 59, 0, time.UTC)}
	require.True(t, klineIsFinal(closed, now))

	// the boundary is the end of the minute plus KlineFinalityMargin: the 12:30
	// kline is final at 12:31:00 + margin ...
	minuteEnd := time.Date(2026, 9, 21, 12, 31, 0, 0, time.UTC)
	require.True(t, klineIsFinal(open, minuteEnd.Add(KlineFinalityMargin)))
	// ... and not a moment before it
	require.False(t, klineIsFinal(open, minuteEnd.Add(KlineFinalityMargin-time.Millisecond)))
}

// the still open minute is dropped, the closed ones are kept. now is fixed
// mid-minute so the decision does not depend on where in the wall-clock minute
// the test happens to run (the finality margin makes the last two seconds after
// a boundary matter).
func TestFinalKlinesDropsTheOpenMinute(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 30, 30, 0, time.UTC)

	kls := []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000), CloseTime: time.Date(2026, 9, 21, 12, 28, 59, 0, time.UTC)},
		{ClosePrice: decimal.NewFromFloat(2001), CloseTime: time.Date(2026, 9, 21, 12, 29, 59, 0, time.UTC)},
		{ClosePrice: decimal.NewFromFloat(2002), CloseTime: time.Date(2026, 9, 21, 12, 30, 59, 0, time.UTC)}, // in progress
	}

	got := finalKlines(kls, now)
	require.Len(t, got, 2)
	require.True(t, got[0].ClosePrice.Equal(decimal.NewFromFloat(2000)))
	require.True(t, got[1].ClosePrice.Equal(decimal.NewFromFloat(2001)))
}

// a provisional price must not reach the price table -- and the request is
// left to the existing retry path rather than closed with a price that is
// still moving
func TestCloseRequestRejectsAStillOpenKline(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	// now is captured once and the request names the minute *after* it --
	// RequestPrice rounds the block time, so a block at M:31 asks for M+1 --
	// and that is the only kline the fetch could answer with. Threading the
	// same now into CloseRequest is the #227 fix: finality is judged against
	// the clock as it stood before the fetch, so the decision is deterministic
	// and no longer flips when the test straddles a minute boundary.
	now := time.Now().UTC()
	ts := now.Truncate(time.Minute).Add(time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", now)

	err := CloseRequest(dbh, rid, ts, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts.Add(59 * time.Second)},
	}, now)
	require.Error(t, err)
	require.Equal(t, 0, priceCount(t, dbh), "a provisional price must not be persisted")
	require.Equal(t, "new", requestStatus(t, dbh, rid), "the request must not be closed")
}

// once the minute has closed the same request is served normally, with the
// final price -- this is the retry that the rejection above leaves open
func TestCloseRequestAcceptsTheKlineOnceTheMinuteClosed(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	// a minute that closed a while ago
	ts := time.Now().UTC().Truncate(time.Minute).Add(-2 * time.Minute)
	rid := insertPriceReq(t, dbh, ts, "failed", time.Now().UTC())

	require.NoError(t, closeReq(dbh, rid, ts, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts.Add(59 * time.Second)},
	}))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	require.Equal(t, 1, priceCount(t, dbh))

	var price decimal.Decimal
	require.NoError(t, dbh.Get(&price, `SELECT price FROM price WHERE asset='eth'`))
	require.True(t, price.Equal(decimal.NewFromFloat(2000.5)), "got %s", price)
}

// the common case: a window that reaches into the current minute still serves
// the (closed) minute it was filed for
func TestCloseRequestServesTheRequestedMinuteDespiteAnOpenTail(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	base := time.Now().UTC().Truncate(time.Minute)
	// now is pinned mid-minute so the two requested minutes (which end at or
	// before `base`) are decisively final and the open tail (which ends at
	// base+2m) is decisively not -- independent of where in the wall-clock
	// minute the test runs.
	now := base.Add(30 * time.Second)
	ts := base.Add(-2 * time.Minute)
	rid := insertPriceReq(t, dbh, ts, "new", now)

	// the tail reaches into the minute *after* the current one, so it cannot
	// go final at now -- see TestCloseRequestRejectsAStillOpenKline
	openTail := base.Add(time.Minute + 59*time.Second)
	require.NoError(t, CloseRequest(dbh, rid, ts, []data.Kline{
		{ClosePrice: decimal.NewFromFloat(2000.5), CloseTime: ts.Add(59 * time.Second)},
		{ClosePrice: decimal.NewFromFloat(2001.5), CloseTime: ts.Add(time.Minute + 59*time.Second)},
		{ClosePrice: decimal.NewFromFloat(2002.5), CloseTime: openTail},
	}, now))
	require.Equal(t, "succeeded", requestStatus(t, dbh, rid))
	// the two closed minutes only
	require.Equal(t, 2, priceCount(t, dbh))

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM price WHERE created_at >= $1`, base))
	require.Zero(t, n, "the minute in progress must not have been persisted")
}
