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
