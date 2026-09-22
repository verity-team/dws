//go:build dbtest

// Integration tests for the delphi read/write paths that talk to the
// database: the affiliate code and the donation data served to the frontend.
//
// These tests need a live postgres database with the dws schema
// (deployments/db/01-schema.sql) loaded, they are therefore hidden behind the
// `dbtest` build tag and are *not* part of a plain `go test ./...` run:
//
//	make run_db
//	go test -tags dbtest -count=1 ./internal/delphi/db/...
//
// The connection string defaults to the dockerized development database and
// can be overridden with DWS_TEST_DB_DSN. Please note: the tests truncate the
// tables they use, never point them at a production database.
package db

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/api"
)

const (
	defaultTestDSN = "host=localhost port=27501 user=postgres password=postgres dbname=dwsdb sslmode=disable"
	testAddress    = "0x00000000000000000000000000000000000000ff"
)

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
	// the affiliate code race needs a connection per writer
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

	_, err := dbh.Exec(`TRUNCATE donation, user_data, wallet_connection, price RESTART IDENTITY`)
	require.NoError(t, err)
	_, err = dbh.Exec(`DELETE FROM donation_stats`)
	require.NoError(t, err)
	_, err = dbh.Exec(`INSERT INTO donation_stats(total, tokens) VALUES(0, 0)`)
	require.NoError(t, err)
}

// insertPrice records a price for the given asset, `age` old.
func insertPrice(t *testing.T, dbh *sqlx.DB, asset, price string, age time.Duration) {
	t.Helper()

	q := `
		INSERT INTO price(asset, price, created_at)
		VALUES($1, $2, timezone('utc', now()) - make_interval(secs => $3))
		`
	_, err := dbh.Exec(q, asset, price, age.Seconds())
	require.NoError(t, err)
}

func priceFor(dd *api.DonationData, asset api.PriceAsset) *api.Price {
	for i := range dd.Prices {
		if dd.Prices[i].Asset == asset {
			return &dd.Prices[i]
		}
	}
	return nil
}

// M6: a second affiliate code request for an address that has a code already
// must return that code instead of failing.
func TestGenerateAffiliateCodeIsIdempotent(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	first, err := GenerateAffiliateCode(dbh, testAddress)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotEmpty(t, first.Code)

	second, err := GenerateAffiliateCode(dbh, testAddress)
	require.NoError(t, err, "an address that has an affiliate code already must not fail")
	require.NotNil(t, second)
	require.Equal(t, first.Code, second.Code, "the affiliate code on record was replaced")
}

// M6: two concurrent /affiliate/code requests for the same address -- the
// loser used to get an HTTP 500 although a code exists.
func TestConcurrentGenerateAffiliateCode(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const writers = 4
	var wg sync.WaitGroup
	results := make(chan *api.AffiliateCode, writers)
	errs := make(chan error, writers)
	start := make(chan struct{})

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			afc, err := GenerateAffiliateCode(dbh, testAddress)
			results <- afc
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var code string
	for afc := range results {
		require.NotNil(t, afc)
		require.NotEmpty(t, afc.Code)
		require.Equal(t, testAddress, afc.Address)
		if code == "" {
			code = afc.Code
		}
		require.Equal(t, code, afc.Code, "concurrent requests got different affiliate codes")
	}

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM user_data WHERE address=$1`, testAddress))
	require.Equal(t, 1, n)
}

// M6: an affiliate code identifies exactly one user -- ConnectWallet looks the
// code up and would insert one wallet connection per colliding user.
func TestAffiliateCodeIsUnique(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	_, err := dbh.Exec(`INSERT INTO user_data(address, affiliate_code) VALUES($1, 'deadbeefdeadbeef')`, testAddress)
	require.NoError(t, err)
	_, err = dbh.Exec(`INSERT INTO user_data(address, affiliate_code) VALUES($1, 'deadbeefdeadbeef')`, "0x00000000000000000000000000000000000000ee")
	require.Error(t, err)
	require.Contains(t, err.Error(), "affiliate_code")

	// ... but an address without an affiliate code is not a duplicate
	_, err = dbh.Exec(`INSERT INTO user_data(address) VALUES($1)`, "0x00000000000000000000000000000000000000ee")
	require.NoError(t, err)
	_, err = dbh.Exec(`INSERT INTO user_data(address) VALUES($1)`, "0x00000000000000000000000000000000000000dd")
	require.NoError(t, err)
}

// M7: with pulitzer down the ETH price is left out altogether -- a zero
// valued price would let a frontend price the donation widget at 0 USD.
func TestGetDonationDataWithoutRecentETHPrice(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", time.Minute)
	insertPrice(t, dbh, "eth", "1798.12", 4*time.Minute)

	dd, err := GetDonationData(dbh)
	require.NoError(t, err)
	require.NotNil(t, dd)
	// the truth price and nothing else -- in particular no zero valued
	// placeholder that a frontend would read as 0 USD
	require.Len(t, dd.Prices, 1, "prices: %+v", dd.Prices)
	require.Nil(t, priceFor(dd, api.PriceAssetEth), "a stale ETH price was served")
	truth := priceFor(dd, api.PriceAssetTruth)
	require.NotNil(t, truth)
	require.Equal(t, "0.00100", truth.Price)
}

// M7/M8: the happy path -- a recent ETH price is served as is.
func TestGetDonationDataWithETHPrice(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)
	insertPrice(t, dbh, "truth", "0.00100", time.Minute)
	insertPrice(t, dbh, "eth", "1798.12", 30*time.Second)

	dd, err := GetDonationData(dbh)
	require.NoError(t, err)
	require.NotNil(t, dd)
	eth := priceFor(dd, api.PriceAssetEth)
	require.NotNil(t, eth)
	require.Equal(t, "1798.12000", eth.Price)
	require.Equal(t, api.DonationDataStatus("open"), dd.Status)
}

// callUpdateUserData invokes the update_user_data() plpgsql function the way
// GetUserData does, discarding the returned row.
func callUpdateUserData(t *testing.T, dbh *sqlx.DB, address string) {
	t.Helper()
	_, err := dbh.Exec(`SELECT * FROM update_user_data($1)`, address)
	require.NoError(t, err)
}

// insertConfirmedDonation inserts one confirmed donation with an explicit
// modified_at (so a test can order it relative to a user_data row). The
// BEFORE UPDATE timestamp trigger fires only on UPDATE, so an explicit
// modified_at on INSERT survives.
func insertConfirmedDonation(t *testing.T, dbh *sqlx.DB, address, usd string, tokens int, txHash string, modifiedAgo time.Duration) {
	t.Helper()
	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash,
			status, block_number, block_hash, block_time, modified_at)
		VALUES(
			$1, $2, $3, 'usdc', $4, 0.001, $5,
			'confirmed', 1, '0xblock', timezone('utc', now()),
			timezone('utc', now()) - make_interval(secs => $6))
		`
	_, err := dbh.Exec(q, address, usd, usd, tokens, txHash, modifiedAgo.Seconds())
	require.NoError(t, err)
}

func userDataTotals(t *testing.T, dbh *sqlx.DB, address string) (string, int) {
	t.Helper()
	var ud struct {
		Total  string `db:"total"`
		Tokens int    `db:"tokens"`
	}
	require.NoError(t, dbh.Get(&ud, `SELECT total::text AS total, tokens FROM user_data WHERE address=$1`, address))
	return ud.Total, ud.Tokens
}

// FIX 1 / trap 1: a never-donated address that is merely *queried* must not get
// a user_data row. /user/data/{address} is unauthenticated and callable at
// ~50/s, so an always-upsert would be a row-creation/enumeration spam vector.
func TestUpdateUserDataDoesNotCreateRowForNonDonor(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const addr = "0x00000000000000000000000000000000000000aa"
	callUpdateUserData(t, dbh, addr)

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM user_data WHERE address=$1`, addr))
	require.Equal(t, 0, n, "a never-donated queried address must not get a user_data row")
}

// FIX 1 / race: a GET /user/data poll that commits inside a finalized-crawler
// commit window bumps user_data.modified_at past an in-flight donation's
// modified_at; the gated function then never folds that donation in. Ground
// truth 600, stuck at 300.
func TestUpdateUserDataRaceDoesNotLoseDonation(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const addr = "0x00000000000000000000000000000000000000bb"
	// two confirmed donations, 300 each, both with an OLDER modified_at
	insertConfirmedDonation(t, dbh, addr, "300.00", 300000, "0xrace1", 10*time.Minute)
	insertConfirmedDonation(t, dbh, addr, "300.00", 300000, "0xrace2", 10*time.Minute)
	// a stale user_data row (only the first donation folded in) whose
	// modified_at a poll bumped to *after* both donations' modified_at
	_, err := dbh.Exec(
		`INSERT INTO user_data(address, total, tokens, modified_at) VALUES($1, 300.00, 300000, timezone('utc', now()))`, addr)
	require.NoError(t, err)

	callUpdateUserData(t, dbh, addr)

	total, tokens := userDataTotals(t, dbh, addr)
	require.Equal(t, "600.00", total, "the in-flight donation was lost")
	require.Equal(t, 600000, tokens)
}

// FIX 1 / affiliate-first: a donor who calls /affiliate/code before the
// donation is first materialized inserts a user_data row whose modified_at is
// permanently newer than the already-confirmed donation; the gated function
// reads 0 forever.
func TestUpdateUserDataAffiliateFirstReadsRealTotal(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const addr = "0x00000000000000000000000000000000000000cc"
	// an already-confirmed donation, older
	insertConfirmedDonation(t, dbh, addr, "1500.00", 1500000, "0xaff1", 10*time.Minute)
	// user_data row created affiliate-code-first: fresh modified_at, zero totals
	_, err := dbh.Exec(
		`INSERT INTO user_data(address, affiliate_code, total, tokens, modified_at) VALUES($1, 'affcode0000000cc', 0, 0, timezone('utc', now()))`, addr)
	require.NoError(t, err)

	callUpdateUserData(t, dbh, addr)

	total, tokens := userDataTotals(t, dbh, addr)
	require.Equal(t, "1500.00", total, "the confirmed donation was never folded in")
	require.Equal(t, 1500000, tokens)
	// the recompute must touch only total/tokens, not the affiliate code
	var code string
	require.NoError(t, dbh.Get(&code, `SELECT affiliate_code FROM user_data WHERE address=$1`, addr))
	require.Equal(t, "affcode0000000cc", code)
}

// FIX 1 / trap 2: when an address's confirmed donations all leave 'confirmed'
// (a re-org's confirmed->failed path) the existing user_data row must be
// zeroed. A gate of merely "EXISTS confirmed donation" would skip it -- no
// confirmed donation remains -- and leave the stale non-zero total.
func TestUpdateUserDataZeroesRowWhenDonationsRevert(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const addr = "0x00000000000000000000000000000000000000dd"
	insertConfirmedDonation(t, dbh, addr, "500.00", 500000, "0xrev1", 10*time.Minute)
	callUpdateUserData(t, dbh, addr)
	total, _ := userDataTotals(t, dbh, addr)
	require.Equal(t, "500.00", total, "precondition: the donation should be folded in")

	// the donation reverts (confirmed -> failed)
	_, err := dbh.Exec(`UPDATE donation SET status='failed' WHERE tx_hash='0xrev1'`)
	require.NoError(t, err)

	callUpdateUserData(t, dbh, addr)
	total, tokens := userDataTotals(t, dbh, addr)
	require.Equal(t, "0.00", total, "a reverted donation must zero the stale total")
	require.Equal(t, 0, tokens)
}

// FIX 1 / write-on-change: a repeated poll with no change must not bump
// user_data.modified_at (served to the API as `ts`).
func TestUpdateUserDataDoesNotBumpTimestampWithoutChange(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const addr = "0x00000000000000000000000000000000000000ee"
	insertConfirmedDonation(t, dbh, addr, "42.00", 42000, "0xnb1", time.Minute)
	callUpdateUserData(t, dbh, addr) // materialize the row

	var ts1 time.Time
	require.NoError(t, dbh.Get(&ts1, `SELECT modified_at FROM user_data WHERE address=$1`, addr))

	// poll again -- nothing changed
	callUpdateUserData(t, dbh, addr)

	var ts2 time.Time
	require.NoError(t, dbh.Get(&ts2, `SELECT modified_at FROM user_data WHERE address=$1`, addr))
	require.Equal(t, ts1, ts2, "modified_at was bumped on a no-op poll")
}

// FIX 3a: a donor must not refer himself. The affiliate code's owning address
// must differ from the connecting address; a genuine third-party referral is
// still accepted and an unknown code is still rejected.
func TestConnectWalletRejectsSelfReferral(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	const (
		owner = testAddress
		code  = "selfcode00000000"
	)
	_, err := dbh.Exec(`INSERT INTO user_data(address, affiliate_code) VALUES($1, $2)`, owner, code)
	require.NoError(t, err)

	countConn := func(address string) int {
		var n int
		require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM wallet_connection WHERE address=$1`, address))
		return n
	}

	// self-referral: the code owner connects with his own code -> rejected
	require.NoError(t, ConnectWallet(dbh, api.ConnectionRequest{Address: owner, Code: code}))
	require.Equal(t, 0, countConn(owner), "self-referral must not create a wallet_connection")

	// a genuine third-party referral -> accepted
	const other = "0x00000000000000000000000000000000000000e1"
	require.NoError(t, ConnectWallet(dbh, api.ConnectionRequest{Address: other, Code: code}))
	require.Equal(t, 1, countConn(other), "a genuine third-party referral must be accepted")

	// an unknown code -> rejected, as before
	const third = "0x00000000000000000000000000000000000000e2"
	require.NoError(t, ConnectWallet(dbh, api.ConnectionRequest{Address: third, Code: "nosuchcode000000"}))
	require.Equal(t, 0, countConn(third), "an unknown code must be rejected")
}

// M1: the wallet connection is recorded lower case -- the duplicate detection
// looks the address up lower case.
func TestConnectWalletLowercasesAddress(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	req := api.ConnectionRequest{Address: "0x00000000000000000000000000000000000000FF", Code: "none"}
	require.NoError(t, ConnectWallet(dbh, req))

	var addresses []string
	require.NoError(t, dbh.Select(&addresses, `SELECT address FROM wallet_connection`))
	require.Equal(t, []string{testAddress}, addresses)

	// the same connection request must not add a second row
	require.NoError(t, ConnectWallet(dbh, req))
	require.NoError(t, dbh.Select(&addresses, `SELECT address FROM wallet_connection`))
	require.Equal(t, []string{testAddress}, addresses)
}

// insertDonationWithBlockTime inserts one confirmed donation with an explicit
// block_time (the on-chain time). Insertion order fixes the BIGSERIAL id, so a
// test can create a donation whose id is higher than another's while its
// block_time is earlier -- the late-confirmed case.
func insertDonationWithBlockTime(t *testing.T, dbh *sqlx.DB, address, txHash string, blockTimeAgo time.Duration) {
	t.Helper()
	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash,
			status, block_number, block_hash, block_time)
		VALUES(
			$1, 1, 100, 'usdc', 1000, 0.001, $2,
			'confirmed', 1, '0xblock',
			timezone('utc', now()) - make_interval(secs => $3))
		`
	_, err := dbh.Exec(q, address, txHash, blockTimeAgo.Seconds())
	require.NoError(t, err)
}

// #231: GetUserDonationData must return donations oldest on-chain (block_time)
// first, not in insertion (id) order. A donation confirmed after a long pending
// period is inserted late -- it gets a higher id than donations already stored
// -- yet its block_time is earlier, so it is older on-chain and must sort
// first. ORDER BY id alone would place it last.
func TestGetUserDonationDataOrdersByBlockTime(t *testing.T) {
	dbh := testDB(t)
	resetDB(t, dbh)

	// inserted FIRST -> lower id, but a *later* block_time (newer on-chain)
	insertDonationWithBlockTime(t, dbh, testAddress, "0xnewer", 1*time.Hour)
	// inserted SECOND -> higher id, but an *earlier* block_time: the
	// late-confirmed donation that is older on-chain
	insertDonationWithBlockTime(t, dbh, testAddress, "0xolder", 2*time.Hour)

	got, err := GetUserDonationData(dbh, testAddress, 100, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []string{"0xolder", "0xnewer"}, []string{got[0].TxHash, got[1].TxHash},
		"donations must be returned oldest block_time first, not id order")

	// paging one row at a time walks the same order with no skip or repeat
	page0, err := GetUserDonationData(dbh, testAddress, 1, 0)
	require.NoError(t, err)
	require.Len(t, page0, 1)
	require.Equal(t, "0xolder", page0[0].TxHash, "first page must be the oldest donation")

	page1, err := GetUserDonationData(dbh, testAddress, 1, 1)
	require.NoError(t, err)
	require.Len(t, page1, 1)
	require.Equal(t, "0xnewer", page1[0].TxHash, "second page must be the newer donation")
}
