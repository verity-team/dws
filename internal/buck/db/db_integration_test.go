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

// saleLimit is the token sale limit the tests run with, soldOutHash the
// donation used to drive the campaign totals up to it.
const (
	saleLimit   = 1_000_000_000
	soldOutHash = "0xdddd111111111111111111111111111111111111111111111111111111111111"
)

func testContext(dbh *sqlx.DB) c.Context {
	return c.Context{
		CrawlerType: c.OldUnconfirmed,
		DB:          dbh,
		// a single, huge price bucket: neither the token price nor the
		// campaign status are supposed to change during these tests
		SaleParams: []c.SaleParam{{Limit: saleLimit, Price: decimal.NewFromFloat(0.001)}},
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

// M5: a closed campaign records the donation -- the money arrived and is
// refundable -- but the sale cannot deliver tokens for it.
//
// Only the row the finalized crawler inserts is zeroed, because that row is
// inserted *as* 'confirmed'. The latest crawler inserts an 'unconfirmed' row,
// which updateDonationStats never sums; the closed rule is applied to it when
// it transitions to 'confirmed' (see TestClosedCampaignZeroesTokensOnConfirmation),
// not at insert time -- zeroing it here would strand the donation at 0 tokens
// if the campaign re-opened before it was confirmed.
func TestPersistTxsClosedCampaignIssuesNoTokens(t *testing.T) {
	for _, tc := range []struct {
		ct     c.CrawlerType
		status string
		tokens int64
	}{
		{c.Finalized, "confirmed", 0},
		{c.Latest, "unconfirmed", 200000},
	} {
		t.Run(tc.ct.String(), func(t *testing.T) {
			dbh := testDB(t)
			resetDB(t, dbh)
			resetPrices(t, dbh)
			insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

			// a closed campaign the crawler could actually have produced
			ctxt := sellOutCampaign(t, dbh)
			ctxt.CrawlerType = tc.ct

			hash := "0x9999999999999999999999999999999999999999999999999999999999999999"
			tx := donationTx(hash, "200")
			tx.Status = tc.status

			require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

			usdAmount, tokens := donationTokens(t, dbh, hash)
			require.Equal(t, "200.00", usdAmount)
			require.EqualValues(t, tc.tokens, tokens)
			require.Equal(t, tc.status, donationStatus(t, dbh, hash))
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

func insertDonationAt(t *testing.T, dbh *sqlx.DB, hash, status string, age time.Duration) time.Time {
	t.Helper()

	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			'0x00000000000000000000000000000000000000ff', 1.0, 10.00, 'usdt', 100,
			0.001, $1, $2, 1, '0xblock', timezone('utc', now()) - $3::interval)
		RETURNING timezone('utc', block_time)
		`
	var bt time.Time
	require.NoError(t, dbh.Get(&bt, q, hash, status, fmt.Sprintf("%d seconds", int64(age.Seconds()))))

	return bt
}

// #216: the old-unconfirmed crawler needs the donation's block time to decide
// whether a transaction that vanished from the chain has been gone long
// enough to be declared dead.
func TestGetOldUnconfirmedReturnsBlockTime(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	oldHash := "0x8888888888888888888888888888888888888888888888888888888888888888"
	youngHash := "0x9999999999999999999999999999999999999999999999999999999999999999"
	confirmedHash := "0xaaaa111111111111111111111111111111111111111111111111111111111111"
	oldBT := insertDonationAt(t, dbh, oldHash, "unconfirmed", 25*time.Hour)
	insertDonationAt(t, dbh, youngHash, "unconfirmed", 10*time.Minute)
	insertDonationAt(t, dbh, confirmedHash, "confirmed", 25*time.Hour)

	txs, err := GetOldUnconfirmed(dbh)
	require.NoError(t, err)
	require.Equal(t, 1, len(txs))
	require.Equal(t, oldHash, txs[0].Hash)
	require.WithinDuration(t, oldBT, txs[0].BlockTime, time.Second)

	// ... and the block time is old enough for the transaction to be
	// declared dead once the provider reports it as absent
	tx := c.TxByHash{Hash: txs[0].Hash, Absent: true, DBBlockTime: txs[0].BlockTime}
	require.Equal(t, c.TxDropped, tx.Judge(uint64(1)))
}

// #216: a dropped transaction has no finalized block; the failed_tx record
// must still carry the block time the donation was seen with.
func TestFailTxForDroppedTxRecordsDonationBlockTime(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	ctxt := testContext(dbh)

	hash := "0xbbbb111111111111111111111111111111111111111111111111111111111111"
	goodHash := "0xcccc111111111111111111111111111111111111111111111111111111111111"
	bt := insertDonationAt(t, dbh, hash, "unconfirmed", 25*time.Hour)
	insertDonation(t, dbh, goodHash, "confirmed", "100.00", 1000)
	// stale stats: the phantom donation was counted by mistake
	_, err := dbh.Exec(`UPDATE donation_stats SET total=110.00, tokens=1100`)
	require.NoError(t, err)

	// no FB* data at all: the provider knows nothing about the transaction
	require.NoError(t, FailTx(ctxt, c.TxByHash{Hash: hash, Absent: true, DBBlockTime: bt}))

	require.Equal(t, "failed", donationStatus(t, dbh, hash))
	require.Equal(t, 1, failedTxCount(t, dbh, hash))
	var ftBT time.Time
	require.NoError(t, dbh.Get(&ftBT, `SELECT block_time FROM failed_tx WHERE tx_hash=$1`, hash))
	require.WithinDuration(t, bt, ftBT, time.Second)

	// the phantom donation no longer counts towards the campaign totals
	total, tokens := donationStats(t, dbh)
	require.Equal(t, "100.00", total)
	require.EqualValues(t, 1000, tokens)
}

// #220 regression: the latest crawler runs ~13 minutes ahead of the finalized
// one, so a donation mined just after the cap was crossed is inserted while
// the campaign is still 'open' and is credited tokens. When the finalized
// crawler reaches that block the campaign is closed -- and its upsert
// deliberately does not touch `tokens`, so the non-zero count used to stand.
// The closed rule has to be applied on the transition to 'confirmed'.
func TestClosedCampaignZeroesTokensOnConfirmation(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	hash := "0xeeee111111111111111111111111111111111111111111111111111111111111"
	tx := donationTx(hash, "50")

	// 1. the latest crawler sees the donation while the campaign is open
	latest := testContext(dbh)
	latest.CrawlerType = c.Latest
	latest.SaleParams = []c.SaleParam{{Limit: soldOutTokens, Price: decimal.NewFromFloat(0.001)}}
	tx.Status = "unconfirmed"
	require.NoError(t, PersistTxs(latest, 42, decimal.NewFromInt(1), []c.Transaction{tx}))
	require.Equal(t, "open", campaignStatus(t, dbh))
	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 50000, tokens, "the latest crawler credits tokens while the campaign is open")

	// 2. the cap is crossed on chain, the campaign closes
	finalized := sellOutCampaign(t, dbh)

	// 3. the finalized crawler reaches the block and confirms the donation
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(finalized, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	// the donation is on record but it issues no tokens: it became confirmed
	// after the campaign closed, hence it is over the cap
	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	usdAmount, tokens := donationTokens(t, dbh, hash)
	require.Equal(t, "50.00", usdAmount, "the donation must not be re-priced")
	require.EqualValues(t, 0, tokens, "a donation confirmed after the campaign closed issues no tokens")

	// ... and the campaign totals do not overshoot the token sale limit
	_, statsTokens := donationStats(t, dbh)
	require.EqualValues(t, soldOutTokens, statsTokens)
	require.Equal(t, "closed", campaignStatus(t, dbh))
}

// ... the idempotency guard: a donation that was already confirmed (and
// legitimately credited) must keep its tokens when its block is re-crawled
// after the campaign closed. The closed rule is a state transition, not a
// re-pricing.
func TestClosedCampaignKeepsTokensOfConfirmedDonation(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	hash := "0xeeee222222222222222222222222222222222222222222222222222222222222"
	tx := donationTx(hash, "50")
	tx.Status = "confirmed"

	// the finalized crawler confirms the donation while the campaign is open
	ctxt := testContext(dbh)
	ctxt.CrawlerType = c.Finalized
	ctxt.SaleParams = []c.SaleParam{{Limit: soldOutTokens, Price: decimal.NewFromFloat(0.001)}}
	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))
	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 50000, tokens)

	// the campaign closes and the very same block is crawled again
	sellOutCampaign(t, dbh)
	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	usdAmount, tokens := donationTokens(t, dbh, hash)
	require.Equal(t, "50.00", usdAmount)
	require.EqualValues(t, 50000, tokens, "an already confirmed donation must keep its tokens")
	require.Equal(t, "closed", campaignStatus(t, dbh))
}

// ... the same transition, taken by the old-unconfirmed crawler: FinalizeTx
// confirms a donation the finalized crawler never got to, and it has to
// respect the closed campaign as well.
func TestFinalizeTxIssuesNoTokensWhenCampaignClosed(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	hash := "0xeeee333333333333333333333333333333333333333333333333333333333333"
	insertDonation(t, dbh, hash, "unconfirmed", "50.00", 50000)
	ctxt := sellOutCampaign(t, dbh)
	ctxt.CrawlerType = c.OldUnconfirmed

	ftx := txByHash(hash)
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(ctxt, ftx))

	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	usdAmount, tokens := donationTokens(t, dbh, hash)
	require.Equal(t, "50.00", usdAmount)
	require.EqualValues(t, 0, tokens, "a donation confirmed after the campaign closed issues no tokens")

	_, statsTokens := donationStats(t, dbh)
	require.EqualValues(t, soldOutTokens, statsTokens)
	require.Equal(t, "closed", campaignStatus(t, dbh))
}

// ... and FinalizeTx keeps crediting tokens while the campaign is open.
func TestFinalizeTxIssuesTokensWhenCampaignOpen(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	hash := "0xeeee444444444444444444444444444444444444444444444444444444444444"
	insertDonation(t, dbh, hash, "unconfirmed", "200.00", 200000)

	ftx := txByHash(hash)
	ftx.FBBlockHash = "0xblock"
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(testContext(dbh), ftx))

	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 200000, tokens)
	require.Equal(t, "open", campaignStatus(t, dbh))
}

// the campaign status is derived state: it is recomputed from the confirmed
// donations on every donation stats write, so the sale re-opens when the
// total no longer reaches the limit.
//
// The reachable way for the total to *fall* is a finalized re-crawl whose
// receipt reports an already confirmed donation as reverted: persistTx writes
// status = EXCLUDED.status, the donation becomes 'failed' and drops out of the
// aggregate. (It is not the old-unconfirmed crawler failing a phantom
// donation -- failTx only transitions 'unconfirmed' rows and the aggregate
// only sums 'confirmed' ones, so such a donation was never in the total.)
func TestCampaignReopensWhenTotalFallsBackUnderTheLimit(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	ctxt := sellOutCampaign(t, dbh)

	// the block is re-crawled and the receipt now reports a revert
	revertSoldOut(t, ctxt, dbh)

	_, statsTokens := donationStats(t, dbh)
	require.EqualValues(t, 0, statsTokens)
	require.Equal(t, "open", campaignStatus(t, dbh), "the sale has tokens left and must re-open")
}

// ... and a raised token sale limit re-opens it as well.
func TestCampaignReopensWhenTheLimitIsRaised(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	ctxt := sellOutCampaign(t, dbh)

	// the operator puts more tokens on sale; the next stats write notices
	raised := ctxt
	raised.SaleParams = []c.SaleParam{{Limit: 400_000, Price: decimal.NewFromFloat(0.001)}}
	hash := "0xeeee555555555555555555555555555555555555555555555555555555555555"
	tx := donationTx(hash, "50")
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(raised, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	require.Equal(t, "open", campaignStatus(t, dbh))
	_, statsTokens := donationStats(t, dbh)
	require.EqualValues(t, soldOutTokens+50000, statsTokens)
}

// ... but 'paused' is the operator's manual lever and a crawler run must
// never touch it.
func TestCampaignStatusPausedIsLeftAlone(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)

	_, err := dbh.Exec(`UPDATE donation_stats SET status='paused'`)
	require.NoError(t, err)

	ctxt := testContext(dbh)
	ctxt.CrawlerType = c.Finalized
	hash := "0xeeee666666666666666666666666666666666666666666666666666666666666"
	tx := donationTx(hash, "200")
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	require.Equal(t, "paused", campaignStatus(t, dbh))
}

// soldOutTokens is the token sale limit sellOutCampaign drives the campaign
// into: a single 50 USD donation at 0.001/token is a quarter of it.
const soldOutTokens = 200_000

// sellOutCampaign drives the campaign into 'closed' the way the crawler
// actually does -- a confirmed donation that reaches the token sale limit,
// with donation_stats recomputed from it -- rather than by seeding a state no
// crawler can produce. The returned context is the finalized crawler's, with
// the sale limit that donation exactly fills.
func sellOutCampaign(t *testing.T, dbh *sqlx.DB) c.Context {
	t.Helper()

	ctxt := testContext(dbh)
	ctxt.CrawlerType = c.Finalized
	ctxt.SaleParams = []c.SaleParam{{Limit: soldOutTokens, Price: decimal.NewFromFloat(0.001)}}

	tx := donationTx(soldOutHash, "200")
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(ctxt, 41, decimal.NewFromInt(1), []c.Transaction{tx}))

	_, statsTokens := donationStats(t, dbh)
	require.GreaterOrEqual(t, statsTokens, int64(soldOutTokens))
	require.Equal(t, "closed", campaignStatus(t, dbh))

	return ctxt
}

// revertSoldOut re-crawls the block that sold the campaign out; the receipt
// now reports the transaction as reverted, so the finalized upsert flips the
// donation from 'confirmed' to 'failed' (status = EXCLUDED.status) and it
// stops counting towards the campaign.
func revertSoldOut(t *testing.T, ctxt c.Context, dbh *sqlx.DB) {
	t.Helper()

	tx := donationTx(soldOutHash, "200")
	tx.Status = "failed"
	require.NoError(t, PersistTxs(ctxt, 41, decimal.NewFromInt(1), []c.Transaction{tx}))
	require.Equal(t, "failed", donationStatus(t, dbh, soldOutHash))
}

// #223 review: the latest crawler must not zero the tokens of the
// 'unconfirmed' row it inserts. updateDonationStats never sums unconfirmed
// donations, so the zero buys nothing -- and if the campaign re-opens before
// the donation is confirmed, neither confirmation path ever restores the
// tokens (the finalized upsert only touches `tokens` while closed, and
// confirmSingleTx keeps them when open). The donor is left confirmed at 0
// tokens with the campaign open.
func TestReopenedCampaignCreditsDonationInsertedWhileClosed(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
	ctxt := sellOutCampaign(t, dbh)

	// 1. the latest crawler picks the donation up while the campaign is closed
	hash := "0xf111111111111111111111111111111111111111111111111111111111111111"
	latest := ctxt
	latest.CrawlerType = c.Latest
	tx := donationTx(hash, "50")
	tx.Status = "unconfirmed"
	require.NoError(t, PersistTxs(latest, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	// 2. the campaign re-opens: the donation that sold it out had reverted
	revertSoldOut(t, ctxt, dbh)
	require.Equal(t, "open", campaignStatus(t, dbh))

	// 3. the finalized crawler confirms the donation, campaign open
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(ctxt, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	usdAmount, tokens := donationTokens(t, dbh, hash)
	require.Equal(t, "50.00", usdAmount)
	require.EqualValues(t, 50000, tokens,
		"a donation confirmed while the campaign is open must be credited")

	_, statsTokens := donationStats(t, dbh)
	require.EqualValues(t, 50000, statsTokens)
}

// ... the same sequence, with the old-unconfirmed crawler doing the
// confirming.
func TestReopenedCampaignCreditsDonationConfirmedByFinalizeTx(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
	ctxt := sellOutCampaign(t, dbh)

	hash := "0xf222222222222222222222222222222222222222222222222222222222222222"
	latest := ctxt
	latest.CrawlerType = c.Latest
	tx := donationTx(hash, "50")
	tx.Status = "unconfirmed"
	require.NoError(t, PersistTxs(latest, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	revertSoldOut(t, ctxt, dbh)
	require.Equal(t, "open", campaignStatus(t, dbh))

	oldUnconfirmed := ctxt
	oldUnconfirmed.CrawlerType = c.OldUnconfirmed
	ftx := txByHash(hash)
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(oldUnconfirmed, ftx))

	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 50000, tokens,
		"a donation confirmed while the campaign is open must be credited")
}

// #223 review: donation_stats.status reflects the *previous* stats write, not
// the configured token sale limit. After the operator raises
// DWS_SALE_PARAMS' limit the stored status still says 'closed', so the first
// finalized block would confirm its donations with 0 tokens and only then
// re-open the campaign. The decision has to be re-derived from the token
// total read under the same lock.
func TestRaisedTokenSaleLimitIssuesTokensImmediately(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
	ctxt := sellOutCampaign(t, dbh)

	// the operator puts more tokens on sale; nothing has told the database
	raised := ctxt
	raised.SaleParams = []c.SaleParam{{Limit: 400_000, Price: decimal.NewFromFloat(0.001)}}
	require.Equal(t, "closed", campaignStatus(t, dbh))

	hash := "0xf333333333333333333333333333333333333333333333333333333333333333"
	tx := donationTx(hash, "50")
	tx.Status = "confirmed"
	require.NoError(t, PersistTxs(raised, 42, decimal.NewFromInt(1), []c.Transaction{tx}))

	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 50000, tokens, "the raised token sale limit takes effect immediately")
	require.Equal(t, "open", campaignStatus(t, dbh))
}

// ... and the old-unconfirmed crawler must not issue 0 tokens on a stale
// 'closed' either.
func TestRaisedTokenSaleLimitIssuesTokensOnFinalizeTx(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	resetPrices(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", 5*time.Minute)
	ctxt := sellOutCampaign(t, dbh)

	hash := "0xf444444444444444444444444444444444444444444444444444444444444444"
	insertDonation(t, dbh, hash, "unconfirmed", "50.00", 50000)

	raised := ctxt
	raised.CrawlerType = c.OldUnconfirmed
	raised.SaleParams = []c.SaleParam{{Limit: 400_000, Price: decimal.NewFromFloat(0.001)}}
	require.Equal(t, "closed", campaignStatus(t, dbh))

	ftx := txByHash(hash)
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(raised, ftx))

	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 50000, tokens, "the raised token sale limit takes effect immediately")
	require.Equal(t, "open", campaignStatus(t, dbh))
}

// donationBlock returns the block identity recorded for a donation.
func donationBlock(t *testing.T, dbh *sqlx.DB, hash string) (uint64, string, time.Time) {
	t.Helper()

	var (
		number uint64
		bhash  string
		btime  time.Time
	)
	err := dbh.QueryRowx(
		`SELECT block_number, block_hash, block_time FROM donation WHERE tx_hash=$1`, hash,
	).Scan(&number, &bhash, &btime)
	require.NoError(t, err)

	return number, bhash, btime
}

// confirmSingleTx used to set block_number, block_time and status but not
// block_hash, so the block identity it wrote was only coherent because the
// caller happens to reject every transaction whose recorded block hash
// differs from the finalized one. The update has to be self-contained: a row
// carrying the block number and time of one block and the hash of another is
// exactly what the donation(block_hash) index would then serve wrong sets
// from.
func TestFinalizeTxWritesTheWholeBlockIdentity(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const (
		hash   = "0xeeee999999999999999999999999999999999999999999999999999999999999"
		fbHash = "0xfinalizedblockhash"
	)
	insertDonation(t, dbh, hash, "unconfirmed", "200.00", 200000)

	// the donation was inserted with block_hash '0xblock'
	_, before, _ := donationBlock(t, dbh, hash)
	require.Equal(t, "0xblock", before)

	ftx := txByHash(hash)
	ftx.BlockNumber = 42
	ftx.BlockHash = fbHash
	ftx.FBBlockHash = fbHash
	ftx.FBBlockTime = time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(testContext(dbh), ftx))

	number, bhash, btime := donationBlock(t, dbh, hash)
	require.EqualValues(t, 42, number)
	require.Equal(t, fbHash, bhash, "block_hash has to be updated along with the rest of the block identity")
	require.Equal(t, ftx.FBBlockTime.UTC(), btime.UTC())
	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
}

// hashes are stored lower case throughout (c.NormalizeHash); a provider that
// reports a checksummed block hash must not write a row the exact-case
// lookups elsewhere will never find again
func TestFinalizeTxNormalizesTheBlockHash(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const hash = "0xeeee888888888888888888888888888888888888888888888888888888888888"
	insertDonation(t, dbh, hash, "unconfirmed", "200.00", 200000)

	ftx := txByHash(hash)
	ftx.BlockHash = "0xABCDEF0123456789"
	ftx.FBBlockHash = "0xABCDEF0123456789"
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(testContext(dbh), ftx))

	_, bhash, _ := donationBlock(t, dbh, hash)
	require.Equal(t, "0xabcdef0123456789", bhash)
}

// TxFinalize is unreachable without a finalized block hash, but an empty one
// must never replace the hash the donation already carries
func TestFinalizeTxKeepsTheBlockHashWhenTheFinalizedOneIsEmpty(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const hash = "0xeeee777777777777777777777777777777777777777777777777777777777777"
	insertDonation(t, dbh, hash, "unconfirmed", "200.00", 200000)

	ftx := txByHash(hash)
	ftx.BlockHash = ""
	ftx.FBBlockHash = ""
	ftx.FBContainsTx = true
	require.NoError(t, FinalizeTx(testContext(dbh), ftx))

	_, bhash, _ := donationBlock(t, dbh, hash)
	require.Equal(t, "0xblock", bhash)
	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
}

// insertZeroDonation records the kind of row the crawler used to create for a
// zero value transfer: amount 0, usd_amount 0, 0 tokens.
func insertZeroDonation(t *testing.T, dbh *sqlx.DB, hash, status string) {
	t.Helper()

	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			'0x00000000000000000000000000000000000000ff', 0, 0, 'eth', 0,
			0.001, $1, $2, 1, '0xblock', timezone('utc', now()))
		`
	_, err := dbh.Exec(q, hash, status)
	require.NoError(t, err)
}

// filterTransactions no longer accepts a transfer that would be recorded as
// zero, but rows written before that are still on the chain and still in the
// database. They must not be stranded: the finalized crawler skips them now,
// so the old-unconfirmed crawler is what settles them -- c.TxByHash.Judge
// returns TxFinalize for a transaction its finalized block carries, and
// FinalizeTx confirms it with the 0 tokens it already had.
func TestPreExistingZeroAmountDonationStillSettles(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const hash = "0xeeee000000000000000000000000000000000000000000000000000000000000"
	insertZeroDonation(t, dbh, hash, "unconfirmed")

	// the tx is on chain and its finalized block carries it
	ftx := txByHash(hash)
	ftx.FBBlockHash = "0xblock"
	ftx.FBContainsTx = true
	ftx.FBDataAvailable = true
	require.Equal(t, c.TxFinalize, ftx.Judge(ftx.BlockNumber))

	require.NoError(t, FinalizeTx(testContext(dbh), ftx))

	require.Equal(t, "confirmed", donationStatus(t, dbh, hash))
	_, tokens := donationTokens(t, dbh, hash)
	require.EqualValues(t, 0, tokens)

	// and it contributes nothing to the campaign totals, exactly as before
	total, statsTokens := donationStats(t, dbh)
	require.Equal(t, "0.00", total)
	require.EqualValues(t, 0, statsTokens)
}
