//go:build dbtest

// Integration tests for the donation stats write path.
//
// These tests need a live postgres database with the dws schema
// (deployments/db/01-schema.sql) loaded, they are therefore hidden behind the
// `dbtest` build tag and are *not* part of a plain `go test ./...` run:
//
//	make run_db
//	go test -tags dbtest -count=1 ./internal/buck/db/...
//
// The connection string defaults to the dockerized development database and
// can be overridden with DWS_TEST_DB_DSN. Please note: the tests truncate the
// donation tables, never point them at a production database.
package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	c "github.com/verity-team/dws/internal/common"
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
	// the concurrency test needs a connection per writer plus the blocker
	// and the monitoring queries
	dbh.SetMaxOpenConns(10)

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

	_, err := dbh.Exec(`TRUNCATE donation, failed_tx, user_data RESTART IDENTITY`)
	require.NoError(t, err)
	_, err = dbh.Exec(`DELETE FROM donation_stats`)
	require.NoError(t, err)
	_, err = dbh.Exec(`INSERT INTO donation_stats(total, tokens) VALUES(0, 0)`)
	require.NoError(t, err)
}

func testContext(dbh *sqlx.DB) c.Context {
	return c.Context{
		CrawlerType: c.OldUnconfirmed,
		DB:          dbh,
		// a single, huge price bucket: neither the token price nor the
		// campaign status are supposed to change during these tests
		SaleParams: []c.SaleParam{{Limit: 1_000_000_000, Price: decimal.NewFromFloat(0.001)}},
	}
}

func insertDonation(t *testing.T, dbh *sqlx.DB, hash, status, usdAmount string, tokens int64) {
	t.Helper()

	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			'0x00000000000000000000000000000000000000ff', 1.0, $1, 'usdt', $2,
			0.001, $3, $4, 1, '0xblock', timezone('utc', now()))
		`
	_, err := dbh.Exec(q, usdAmount, tokens, hash, status)
	require.NoError(t, err)
}

func donationStats(t *testing.T, dbh *sqlx.DB) (string, int64) {
	t.Helper()

	var (
		total  decimal.Decimal
		tokens int64
	)
	err := dbh.QueryRowx(`SELECT total, tokens FROM donation_stats`).Scan(&total, &tokens)
	require.NoError(t, err)

	return total.StringFixed(2), tokens
}

func donationStatus(t *testing.T, dbh *sqlx.DB, hash string) string {
	t.Helper()

	var status string
	err := dbh.Get(&status, `SELECT status FROM donation WHERE tx_hash=$1`, hash)
	require.NoError(t, err)

	return status
}

func failedTxCount(t *testing.T, dbh *sqlx.DB, hash string) int {
	t.Helper()

	var n int
	err := dbh.Get(&n, `SELECT COUNT(*) FROM failed_tx WHERE tx_hash=$1`, hash)
	require.NoError(t, err)

	return n
}

// waitForLockWaiters blocks until `want` backends are waiting for a lock.
func waitForLockWaiters(t *testing.T, dbh *sqlx.DB, want int) {
	t.Helper()

	q := `
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE
			datname = current_database()
			AND wait_event_type = 'Lock'
			AND pid <> pg_backend_pid()
		`
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		require.NoError(t, dbh.Get(&n, q))
		if n >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d blocked donation stats writers", want)
}

func txByHash(hash string) c.TxByHash {
	return c.TxByHash{
		Hash:         hash,
		BlockHash:    "0xblock",
		BlockNumber:  1,
		FBBlockHash:  "0xfinalized",
		FBBlockTime:  time.Now().UTC(),
		FBContainsTx: false,
	}
}

// P2: two concurrent donation stats writers must not lose each other's
// donation. Both writers are parked on the donation_stats lock, released at
// the same time and are thus guaranteed to race.
func TestConcurrentFinalizeTxKeepsDonationStatsConsistent(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	ctxt := testContext(dbh)

	hashes := []string{
		"0x1111111111111111111111111111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222222222222222222222222222",
	}
	insertDonation(t, dbh, hashes[0], "unconfirmed", "100.00", 1000)
	insertDonation(t, dbh, hashes[1], "unconfirmed", "200.00", 2000)

	// park both writers on the donation stats lock
	blocker, err := dbh.Beginx()
	require.NoError(t, err)
	_, err = blocker.Exec(`SELECT tokens FROM donation_stats FOR UPDATE`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, len(hashes))
	for _, hash := range hashes {
		wg.Add(1)
		go func(hash string) {
			defer wg.Done()
			errs <- FinalizeTx(ctxt, txByHash(hash))
		}(hash)
	}

	waitForLockWaiters(t, dbh, len(hashes))
	require.NoError(t, blocker.Rollback())

	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	for _, hash := range hashes {
		require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	}
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "300.00", total, "donation stats lost an update")
	require.EqualValues(t, 3000, tokens, "donation stats lost an update")
}

// P2: the finalized crawler block writer serializes with the other donation
// stats writers as well.
func TestConcurrentPersistTxsAndFinalizeTxKeepDonationStatsConsistent(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	finalizeCtxt := testContext(dbh)
	blockCtxt := testContext(dbh)
	blockCtxt.CrawlerType = c.Finalized

	oldHash := "0x3333333333333333333333333333333333333333333333333333333333333333"
	newHash := "0x4444444444444444444444444444444444444444444444444444444444444444"
	insertDonation(t, dbh, oldHash, "unconfirmed", "100.00", 1000)

	newTx := c.Transaction{
		TXH:         c.TXH{Hash: newHash},
		From:        "0x00000000000000000000000000000000000000ff",
		Value:       "200",
		Asset:       "usdt",
		Status:      "confirmed",
		BlockNumber: 2,
		BlockHash:   "0xblock2",
		BlockTime:   time.Now().UTC(),
	}

	blocker, err := dbh.Beginx()
	require.NoError(t, err)
	_, err = blocker.Exec(`SELECT tokens FROM donation_stats FOR UPDATE`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs <- FinalizeTx(finalizeCtxt, txByHash(oldHash))
	}()
	go func() {
		defer wg.Done()
		errs <- PersistTxs(blockCtxt, 2, decimal.NewFromInt(1), []c.Transaction{newTx})
	}()

	waitForLockWaiters(t, dbh, 2)
	require.NoError(t, blocker.Rollback())

	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	// 100 USD / 1000 tokens (old, finalized) + 200 USD / 200000 tokens (new
	// block, 200 USD at a token price of 0.001)
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "300.00", total, "donation stats lost an update")
	require.EqualValues(t, 201000, tokens, "donation stats lost an update")
}

// P3: failing an unconfirmed donation keeps the row, marks it 'failed' and
// recomputes the donation stats in the same transaction.
func TestFailTxMarksDonationAsFailedAndUpdatesStats(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	ctxt := testContext(dbh)

	confirmedHash := "0x5555555555555555555555555555555555555555555555555555555555555555"
	failedHash := "0x6666666666666666666666666666666666666666666666666666666666666666"
	insertDonation(t, dbh, confirmedHash, "confirmed", "100.00", 1000)
	insertDonation(t, dbh, failedHash, "unconfirmed", "50.00", 500)
	// stale stats: the unconfirmed donation was counted by mistake
	_, err := dbh.Exec(`UPDATE donation_stats SET total=150.00, tokens=1500`)
	require.NoError(t, err)

	require.NoError(t, FailTx(ctxt, txByHash(failedHash)))

	// the donation row is retained
	require.Equal(t, "failed", donationStatus(t, dbh, failedHash))
	require.Equal(t, 1, failedTxCount(t, dbh, failedHash))
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "100.00", total)
	require.EqualValues(t, 1000, tokens)

	// failing the same tx again is a no-op: the donation is no longer
	// 'unconfirmed' -> the stats are left alone
	_, err = dbh.Exec(`UPDATE donation_stats SET total=999.99, tokens=9999`)
	require.NoError(t, err)

	require.NoError(t, FailTx(ctxt, txByHash(failedHash)))

	require.Equal(t, "failed", donationStatus(t, dbh, failedHash))
	require.Equal(t, 1, failedTxCount(t, dbh, failedHash))
	total, tokens = donationStats(t, dbh)
	require.Equal(t, "999.99", total, "donation stats must not be touched")
	require.EqualValues(t, 9999, tokens, "donation stats must not be touched")
}

// P3: the race this fixes -- the donation was confirmed while the
// old-unconfirmed crawler was talking to the ethereum node. The confirmed
// donation must survive and the donation stats must not be touched.
func TestFailTxLeavesConfirmedDonationAlone(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	ctxt := testContext(dbh)

	hash := "0x7777777777777777777777777777777777777777777777777777777777777777"
	insertDonation(t, dbh, hash, "confirmed", "100.00", 1000)
	// sentinel values: a recompute would overwrite them with 100.00 / 1000
	_, err := dbh.Exec(`UPDATE donation_stats SET total=999.99, tokens=9999`)
	require.NoError(t, err)

	require.NoError(t, FailTx(ctxt, txByHash(hash)))

	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	// the failed_tx record is kept as evidence of the race
	require.Equal(t, 1, failedTxCount(t, dbh, hash))
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "999.99", total, "donation stats must not be touched")
	require.EqualValues(t, 9999, tokens, "donation stats must not be touched")

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM donation WHERE tx_hash=$1`, hash))
	require.Equal(t, 1, n, fmt.Sprintf("confirmed donation %s was deleted", hash))
}

// resetPrices empties the price table; the token price rows the test needs
// are inserted with insertPrice.
func resetPrices(t *testing.T, dbh *sqlx.DB) {
	t.Helper()

	_, err := dbh.Exec(`TRUNCATE price RESTART IDENTITY`)
	require.NoError(t, err)
}

// insertPrice records a price for the given asset, `age` old.
func insertPrice(t *testing.T, dbh *sqlx.DB, asset, price string, age time.Duration) {
	t.Helper()

	q := `
		INSERT INTO price(asset, price, created_at)
		VALUES($1, $2, timezone('utc', now()) - $3::interval)
		`
	_, err := dbh.Exec(q, asset, price, fmt.Sprintf("%d seconds", int(age.Seconds())))
	require.NoError(t, err)
}

func donationTokens(t *testing.T, dbh *sqlx.DB, hash string) (string, int64) {
	t.Helper()

	var (
		usdAmount decimal.Decimal
		tokens    int64
	)
	err := dbh.QueryRowx(`SELECT usd_amount, tokens FROM donation WHERE tx_hash=$1`, hash).Scan(&usdAmount, &tokens)
	require.NoError(t, err)

	return usdAmount.StringFixed(2), tokens
}

func campaignStatus(t *testing.T, dbh *sqlx.DB) string {
	t.Helper()

	var status string
	require.NoError(t, dbh.Get(&status, `SELECT status FROM donation_stats`))

	return status
}

func donationTx(hash, value string) c.Transaction {
	return c.Transaction{
		TXH:         c.TXH{Hash: hash},
		From:        "0x00000000000000000000000000000000000000ff",
		Value:       value,
		Asset:       "usdt",
		BlockNumber: 42,
		BlockHash:   "0xblock42",
		BlockTime:   time.Now().UTC(),
	}
}

// M2: SUM() over zero confirmed donations is NULL and donation_stats.total is
// NOT NULL -- the finalized crawler must not wedge on a block whose donations
// were all skipped.
func TestPersistTxsWithoutConfirmedDonations(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	ctxt := testContext(dbh)
	ctxt.CrawlerType = c.Finalized

	hash := "0x8888888888888888888888888888888888888888888888888888888888888888"
	// an amount that cannot be parsed -> the donation is skipped and the
	// database holds no confirmed donation at all
	tx := donationTx(hash, "not-a-number")
	tx.Status = "confirmed"

	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM donation`))
	require.Equal(t, 0, n, "the donation should have been skipped")
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "0.00", total)
	require.EqualValues(t, 0, tokens)
}

// M3: donation_stats is the single row every writer updates without a WHERE
// clause and every reader picks with LIMIT 1 -- a second row would make the
// two disagree.
func TestDonationStatsHoldsASingleRow(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	_, err := dbh.Exec(`INSERT INTO donation_stats(total, tokens) VALUES(1, 1)`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "donation_stats_single_row")
}

// M4: the "single row" escape hatch of the token price query has to look at
// the token price rows only -- pulitzer fills the price table with an `eth`
// row every minute.
func TestGetTokenPriceIgnoresOtherAssets(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	ctxt := testContext(dbh)

	// the only token price row, younger than the two minute grace period
	insertPrice(t, dbh, "truth", "0.00250", 30*time.Second)
	// ... and the price rows pulitzer writes every minute
	insertPrice(t, dbh, "eth", "1798.12", time.Minute)
	insertPrice(t, dbh, "eth", "1799.34", 0)

	dtx, err := dbh.Beginx()
	require.NoError(t, err)
	defer dtx.Rollback() // nolint:errcheck

	price, err := getTokenPrice(dtx, ctxt)
	require.NoError(t, err)
	require.Equal(t, "0.00250", price.StringFixed(5), "the single token price row was ignored")
}

// M4: with more than one token price row on record a tier change only takes
// effect after the two minute grace period.
func TestGetTokenPriceHonoursGracePeriod(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	ctxt := testContext(dbh)

	insertPrice(t, dbh, "truth", "0.00250", 5*time.Minute)
	insertPrice(t, dbh, "truth", "0.00500", 0)
	insertPrice(t, dbh, "eth", "1799.34", 0)

	dtx, err := dbh.Beginx()
	require.NoError(t, err)
	defer dtx.Rollback() // nolint:errcheck

	price, err := getTokenPrice(dtx, ctxt)
	require.NoError(t, err)
	require.Equal(t, "0.00250", price.StringFixed(5), "a token price younger than 2 minutes was used")
}

// M5: a closed campaign records the donation but issues no tokens -- the sale
// cannot deliver them.
func TestPersistTxsClosedCampaignIssuesNoTokens(t *testing.T) {
	for _, ct := range []c.CrawlerType{c.Latest, c.Finalized} {
		t.Run(ct.String(), func(t *testing.T) {
			dbh := testDB(t)
			resetDB(t, dbh)
			resetPrices(t, dbh)
			insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
			ctxt := testContext(dbh)
			ctxt.CrawlerType = ct

			_, err := dbh.Exec(`UPDATE donation_stats SET status='closed'`)
			require.NoError(t, err)

			hash := "0x9999999999999999999999999999999999999999999999999999999999999999"
			tx := donationTx(hash, "200")
			if ct == c.Finalized {
				tx.Status = "confirmed"
			} else {
				tx.Status = "unconfirmed"
			}

			require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

			// the donation is on record (the money arrived and is refundable)
			// but it does not credit the donor with a single token
			usdAmount, tokens := donationTokens(t, dbh, hash)
			require.Equal(t, "200.00", usdAmount)
			require.EqualValues(t, 0, tokens, "tokens were issued for a closed campaign")
			require.Equal(t, "closed", campaignStatus(t, dbh))
		})
	}
}

// M5: an open campaign issues tokens as before.
func TestPersistTxsOpenCampaignIssuesTokens(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
	ctxt := testContext(dbh)
	ctxt.CrawlerType = c.Finalized

	hash := "0xaaaa111111111111111111111111111111111111111111111111111111111111"
	tx := donationTx(hash, "200")
	tx.Status = "confirmed"

	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	usdAmount, tokens := donationTokens(t, dbh, hash)
	require.Equal(t, "200.00", usdAmount)
	require.EqualValues(t, 200000, tokens)
	require.Equal(t, "open", campaignStatus(t, dbh))
}
