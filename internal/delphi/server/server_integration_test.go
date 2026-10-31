//go:build dbtest

// Integration tests for the shape of the /user/data/{address} response and for
// the /wallet/connection write path.
//
// The response is what the frontend renders, so what matters is the JSON that
// actually goes over the wire rather than the Go value behind it; that needs a
// live postgres database with the dws schema (deployments/db/01-schema.sql)
// loaded, so these tests are hidden behind the `dbtest` build tag and are
// *not* part of a plain `go test ./...` run:
//
//	make run_db
//	go test -tags dbtest -count=1 ./internal/delphi/server/...
//
// The connection string defaults to the dockerized development database and
// can be overridden with DWS_TEST_DB_DSN. Please note: the tests truncate the
// tables they use, never point them at a production database.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/dbtest"
)

const integrationAddress = "0x00000000000000000000000000000000000000ff"

// TestMain serializes this package with the other database integration test
// packages: they share the database and truncate each other's tables, see
// dbtest.Run.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Run(m))
}

// integrationDB connects to the test database and skips the calling test if no
// database is reachable.
func integrationDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dbh, err := sqlx.Open("postgres", dbtest.DSN())
	require.NoError(t, err)

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

// resetUserData brings the tables these tests read into a well defined state.
func resetUserData(t *testing.T, dbh *sqlx.DB) {
	t.Helper()

	_, err := dbh.Exec(`TRUNCATE donation, user_data RESTART IDENTITY`)
	require.NoError(t, err)
}

// userDataBody performs a /user/data/{address} request and returns the raw
// response body.
func userDataBody(t *testing.T, dbh *sqlx.DB, address string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/user/data/"+address, nil)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	ctx.SetPath("/user/data/:address")

	s := NewDelphiServer(dbh)
	require.NoError(t, s.UserData(ctx, address, api.UserDataParams{}))
	require.Equal(t, http.StatusOK, rec.Code)

	return rec.Body.String()
}

// `donations` is a required array in api/delphi.yaml; a nil slice used to
// marshal to `null`, so a caller doing donations.map(..) threw on an address
// that had not donated.
func TestUserDataDonationsIsAlwaysAnArray(t *testing.T) {
	dbh := integrationDB(t)
	resetUserData(t, dbh)

	body := userDataBody(t, dbh, integrationAddress)

	assert.Contains(t, body, `"donations":[]`)
	assert.NotContains(t, body, `"donations":null`)

	// and it decodes as the array the spec declares
	var res api.UserDataResult
	require.NoError(t, json.Unmarshal([]byte(body), &res))
	assert.NotNil(t, res.Donations)
	assert.Empty(t, res.Donations)

	// the user data defaults travel with it
	assert.Equal(t, api.None, res.UserData.Status)
	assert.Equal(t, "0", res.UserData.Tokens)
}

// an address with a `user_data` row but no donations on the requested page is
// the other way to end up with an empty history. The user_data total is derived
// from the confirmed donations (update_user_data), so it must be backed by a
// real one -- a phantom total that no donation supports is now correctly zeroed.
func TestUserDataDonationsIsAnArrayWithUserDataPresent(t *testing.T) {
	dbh := integrationDB(t)
	resetUserData(t, dbh)

	_, err := dbh.Exec(`
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			$1, 1.0, 100.00, 'usdt', 50000, 0.001,
			'0xbbbb222222222222222222222222222222222222222222222222222222222222',
			'confirmed', 1, '0xblock', timezone('utc', now()))`,
		integrationAddress)
	require.NoError(t, err)

	// the donation is on the first page; ask for the second so the history is
	// empty while the user_data summary still travels with the response
	offset := 50
	req := httptest.NewRequest(http.MethodGet, "/user/data/"+integrationAddress+"?offset=50", nil)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	ctx.SetPath("/user/data/:address")

	s := NewDelphiServer(dbh)
	require.NoError(t, s.UserData(ctx, integrationAddress, api.UserDataParams{Offset: &offset}))
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, `"donations":[]`)
	assert.NotContains(t, body, `"donations":null`)

	var res api.UserDataResult
	require.NoError(t, json.Unmarshal([]byte(body), &res))
	assert.NotNil(t, res.Donations)
	assert.Equal(t, "50000", res.UserData.Tokens)
}

// a page past the end of the history must not regress to `null` either
func TestUserDataDonationsIsAnArrayPastTheLastPage(t *testing.T) {
	dbh := integrationDB(t)
	resetUserData(t, dbh)

	_, err := dbh.Exec(`
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			$1, 1.0, 100.00, 'usdt', 50000, 0.001,
			'0xaaaa111111111111111111111111111111111111111111111111111111111111',
			'confirmed', 1, '0xblock', timezone('utc', now()))`,
		integrationAddress)
	require.NoError(t, err)

	// the donation is on the first page, not on the second
	offset := 50
	req := httptest.NewRequest(http.MethodGet, "/user/data/"+integrationAddress+"?offset=50", nil)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	ctx.SetPath("/user/data/:address")

	s := NewDelphiServer(dbh)
	require.NoError(t, s.UserData(ctx, integrationAddress, api.UserDataParams{Offset: &offset}))
	require.Equal(t, http.StatusOK, rec.Code)

	assert.Contains(t, rec.Body.String(), `"donations":[]`)
	assert.NotContains(t, rec.Body.String(), `"donations":null`)

	// sanity: the first page does carry the donation
	body := userDataBody(t, dbh, integrationAddress)
	assert.Contains(t, body, `"donations":[{`)
}

// AUD4-12: the affiliate code is part of the signed /wallet/connection message.
// A correctly signed request records the code it was signed with; the same
// signature replayed with another code is rejected and records nothing.
func TestConnectWalletRecordsTheSignedCode(t *testing.T) {
	dbh := integrationDB(t)
	_, err := dbh.Exec(`TRUNCATE user_data, wallet_connection RESTART IDENTITY`)
	require.NoError(t, err)

	const (
		referrer  = "0x00000000000000000000000000000000000000ee"
		signed    = "0123456789abcdef"
		attackers = "fedcba9876543210"
	)
	_, err = dbh.Exec(`INSERT INTO user_data(address, affiliate_code) VALUES($1, $2), ('0x00000000000000000000000000000000000000dd', $3)`, referrer, signed, attackers)
	require.NoError(t, err)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	authTS := time.Now().UTC()
	params := api.ConnectWalletParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signConnection(t, key, addr, authTS, signed),
	}
	s := NewDelphiServer(dbh)

	codes := func() []string {
		var cc []string
		require.NoError(t, dbh.Select(&cc, `SELECT code FROM wallet_connection WHERE address=$1 ORDER BY id`, strings.ToLower(addr)))
		return cc
	}

	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":%q,"address":%q}`, signed, addr))
	require.NoError(t, s.ConnectWallet(ctx, params))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{signed}, codes())

	ctx, rec = connectWalletCtx(t, fmt.Sprintf(`{"code":%q,"address":%q}`, attackers, addr))
	require.NoError(t, s.ConnectWallet(ctx, params))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, []string{signed}, codes(), "a replayed signature must not record another code")
}

// #233: a correctly signed connection whose affiliate code is turned down gets
// a 422 that says why -- an unknown code (119) or the caller's own code (120)
// -- and records nothing.
func TestConnectWalletReportsARejectedCode(t *testing.T) {
	dbh := integrationDB(t)
	_, err := dbh.Exec(`TRUNCATE user_data, wallet_connection RESTART IDENTITY`)
	require.NoError(t, err)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	const own = "00000000000000c1"
	_, err = dbh.Exec(`INSERT INTO user_data(address, affiliate_code) VALUES($1, $2)`, strings.ToLower(addr), own)
	require.NoError(t, err)
	s := NewDelphiServer(dbh)

	for _, tc := range []struct {
		code    string
		errCode int
	}{
		{"nosuchcode000000", 119},
		{own, 120},
	} {
		authTS := time.Now().UTC()
		params := api.ConnectWalletParams{
			DelphiKey:       addr,
			DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
			DelphiSignature: signConnection(t, key, addr, authTS, tc.code),
		}
		ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":%q,"address":%q}`, tc.code, addr))
		require.NoError(t, s.ConnectWallet(ctx, params))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, tc.code)
		assert.Contains(t, rec.Body.String(), fmt.Sprintf(`"code":%d`, tc.errCode), tc.code)
	}

	var n int
	require.NoError(t, dbh.Get(&n, `SELECT COUNT(*) FROM wallet_connection`))
	assert.Zero(t, n, "a turned down code must not be recorded")
}
