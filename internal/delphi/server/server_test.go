package server

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/api"
)

func TestVerifySigBoring(t *testing.T) {
	const (
		msg  = "aea3eb2f5a6a2efe002d3c88da52ba5a8702c9722ae2d67d1260e4318f5ccd6c"
		sig  = "0x93433430e249145433931dd4fda65090fcb250489e107d460b8adcef4a3c05f863c860d63c3f4ff92dcda4cce9755e4771e9dc6b91dafd5c900c7a5b99c169d71b"
		from = "0xb938F65DfE303EdF96A511F1e7E3190f69036860"
	)

	res := verifySig(from, msg, sig)
	assert.True(t, res)
}

func TestVerifySig(t *testing.T) {
	const (
		msg  = "Join the meme army!!"
		sig  = "0xf689927d8f85c4f11b5b7ff62cfe9c8631f1373d3fe74a168031d23a7b7faf6903cfaed61f9fd153c506ffe3e1fd912fb42271859d6307239d36272ed00947da1b"
		from = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	)

	res := verifySig(from, msg, sig)
	assert.True(t, res)
}

func TestVerifySigFriendly(t *testing.T) {
	const (
		msg  = "Hello Minh! How are you today? :)"
		sig  = "0xc9966a58f2e6b45c2b3d3205f869c419afd29c4c9dc64d68f416257a3fc236216746b44dcf95b4e0d7dd5ed97523fb179d41a87db5882292ef991385a6f87c7e1c"
		from = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	)

	res := verifySig(from, msg, sig)
	assert.True(t, res)
}

func TestVerifySigShortSignature(t *testing.T) {
	// a signature shorter than crypto.SignatureLength used to panic with
	// "index out of range" when the recovery id was read
	const (
		msg  = "wallet connection, 2023-10-23 18:45:19+00:00"
		from = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	)

	assert.NotPanics(t, func() {
		assert.False(t, verifySig(from, msg, "0x00"))
	})
	assert.NotPanics(t, func() {
		assert.False(t, verifySig(from, msg, "0x"))
	})
}

func TestVerifySigNonHex(t *testing.T) {
	const (
		msg  = "wallet connection, 2023-10-23 18:45:19+00:00"
		from = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	)

	// no 0x prefix, odd length and non-hex characters
	assert.False(t, verifySig(from, msg, "deadbeef"))
	assert.False(t, verifySig(from, msg, "0xabc"))
	assert.False(t, verifySig(from, msg, "0xzz"))
	assert.False(t, verifySig(from, msg, ""))
}

func TestVerifySigWrongSigner(t *testing.T) {
	const msg = "wallet connection, 2023-10-23 18:45:19+00:00"

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	other, err := crypto.GenerateKey()
	require.NoError(t, err)

	sig := signTestMsg(t, key, msg)
	assert.True(t, verifySig(crypto.PubkeyToAddress(key.PublicKey).Hex(), msg, sig))
	assert.False(t, verifySig(crypto.PubkeyToAddress(other.PublicKey).Hex(), msg, sig))
	// a valid signature over a different message must not authenticate
	assert.False(t, verifySig(crypto.PubkeyToAddress(key.PublicKey).Hex(), "affiliate code, 2023-10-23 18:45:19+00:00", sig))
}

func TestFormMsgWalletConnection(t *testing.T) {
	ts := time.Date(2023, 10, 23, 18, 45, 19, 0, time.UTC)
	const addr = "0xB938F65DfE303EdF96A511F1e7E3190f69036860"
	assert.Equal(t,
		"wallet connection, 0xb938f65dfe303edf96a511f1e7e3190f69036860, 2023-10-23 18:45:19+00:00",
		formMsg("/wallet/connection", addr, ts))
	assert.Equal(t,
		"affiliate code, 0xb938f65dfe303edf96a511f1e7e3190f69036860, 2023-10-23 18:45:19+00:00",
		formMsg("/affiliate/code", addr, ts))
}

// the signed message is bound to the address it is used for
func TestFormMsgBindsAddress(t *testing.T) {
	ts := time.Date(2023, 10, 23, 18, 45, 19, 0, time.UTC)
	a := formMsg("/affiliate/code", "0xb938f65dfe303edf96a511f1e7e3190f69036860", ts)
	b := formMsg("/affiliate/code", "0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154", ts)
	assert.NotEqual(t, a, b)
}

// signTestMsg signs msg the way an ethereum wallet's personal_sign does.
func signTestMsg(t *testing.T, key *ecdsa.PrivateKey, msg string) string {
	t.Helper()
	sig, err := crypto.Sign(accounts.TextHash([]byte(msg)), key)
	require.NoError(t, err)
	return hexutil.Encode(sig)
}

// connectWalletCtx builds an echo context for POST /wallet/connection.
func connectWalletCtx(t *testing.T, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/wallet/connection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	// the generated router registers the handler under this path; formMsg
	// derives the signed message from it
	ctx.SetPath("/wallet/connection")
	return ctx, rec
}

func TestConnectWalletInvalidTimestamp(t *testing.T) {
	ctx, rec := connectWalletCtx(t, `{"code":"none","address":"0xb938f65dfe303edf96a511f1e7e3190f69036860"}`)
	s := NewDelphiServer(nil)

	err := s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       "0xb938f65dfe303edf96a511f1e7e3190f69036860",
		DelphiTs:        "not-a-timestamp",
		DelphiSignature: "0x00",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConnectWalletStaleTimestamp(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	authTS := time.Now().UTC().Add(-time.Duration(MaxTimestampAge+60) * time.Second)
	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":"none","address":%q}`, addr))
	s := NewDelphiServer(nil)

	err = s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", addr, authTS.Truncate(time.Second))),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConnectWalletBadSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	other, err := crypto.GenerateKey()
	require.NoError(t, err)

	authTS := time.Now().UTC()
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":"none","address":%q}`, addr))
	s := NewDelphiServer(nil)

	// signed by somebody else
	err = s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, other, formMsg("/wallet/connection", addr, authTS.Truncate(time.Second))),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConnectWalletShortSignatureDoesNotPanic(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":"none","address":%q}`, addr))
	s := NewDelphiServer(nil)

	assert.NotPanics(t, func() {
		err := s.ConnectWallet(ctx, api.ConnectWalletParams{
			DelphiKey:       addr,
			DelphiTs:        strconv.FormatInt(time.Now().UTC().Unix(), 10),
			DelphiSignature: "0x00",
		})
		require.NoError(t, err)
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConnectWalletAddressMismatch(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	victim, err := crypto.GenerateKey()
	require.NoError(t, err)

	authTS := time.Now().UTC()
	attacker := crypto.PubkeyToAddress(key.PublicKey).Hex()
	victimAddr := crypto.PubkeyToAddress(victim.PublicKey).Hex()

	// correctly signed by the attacker but crediting the victim's address
	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":"none","address":%q}`, victimAddr))
	s := NewDelphiServer(nil)

	err = s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       attacker,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", attacker, authTS.Truncate(time.Second))),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConnectWalletInvalidAddress(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	authTS := time.Now().UTC()
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	ctx, rec := connectWalletCtx(t, `{"code":"none","address":"not-an-address"}`)
	s := NewDelphiServer(nil)

	err = s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", addr, authTS.Truncate(time.Second))),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// the campaign pause guard: the ETH price is picked by asset and compared
// numerically -- postgres renders a NUMERIC(15,5) zero as "0.00000"
func TestHaveETHPrice(t *testing.T) {
	tests := []struct {
		name   string
		prices []api.Price
		want   bool
	}{
		{
			name: "usable ETH price",
			prices: []api.Price{
				{Asset: api.PriceAssetEth, Price: "1798.12000"},
				{Asset: api.PriceAssetTruth, Price: "0.00100"},
			},
			want: true,
		},
		{
			name: "numeric zero as rendered by postgres",
			prices: []api.Price{
				{Asset: api.PriceAssetEth, Price: "0.00000"},
				{Asset: api.PriceAssetTruth, Price: "0.00100"},
			},
			want: false,
		},
		{
			name:   "zero without trailing digits",
			prices: []api.Price{{Asset: api.PriceAssetEth, Price: "0.00"}},
			want:   false,
		},
		{
			name:   "empty price",
			prices: []api.Price{{Asset: api.PriceAssetEth, Price: ""}},
			want:   false,
		},
		{
			name:   "garbage price",
			prices: []api.Price{{Asset: api.PriceAssetEth, Price: "n/a"}},
			want:   false,
		},
		{
			name:   "negative price",
			prices: []api.Price{{Asset: api.PriceAssetEth, Price: "-1.00000"}},
			want:   false,
		},
		{
			name:   "no ETH price at all",
			prices: []api.Price{{Asset: api.PriceAssetTruth, Price: "0.00100"}},
			want:   false,
		},
		{
			name:   "no prices at all",
			prices: nil,
			want:   false,
		},
		{
			// the ETH price is not first by contract, only by append order
			name: "ETH price is not the first entry",
			prices: []api.Price{
				{Asset: api.PriceAssetTruth, Price: "0.00100"},
				{Asset: api.PriceAssetEth, Price: "1798.12000"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, haveETHPrice(tc.prices))
		})
	}
}

// a timestamp in the *future* used to yield a negative age and pass the check
// unconditionally: a signature harvested over a far future timestamp stayed
// valid -- and replayable -- until that timestamp had passed
func TestAuthTSOutOfWindowFuture(t *testing.T) {
	now := time.Now().UTC()

	assert.False(t, authTSOutOfWindow(now), "a current timestamp must be accepted")
	assert.False(t, authTSOutOfWindow(now.Add(-time.Duration(MaxTimestampAge-5)*time.Second)))
	assert.True(t, authTSOutOfWindow(now.Add(-time.Duration(MaxTimestampAge+5)*time.Second)))

	// small clock drift is tolerated, a timestamp that is genuinely in the
	// future is not
	assert.False(t, authTSOutOfWindow(now.Add(time.Duration(MaxTimestampSkew-2)*time.Second)))
	assert.True(t, authTSOutOfWindow(now.Add(time.Duration(MaxTimestampSkew+5)*time.Second)))
	assert.True(t, authTSOutOfWindow(now.Add(24*time.Hour)))
	assert.True(t, authTSOutOfWindow(time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)))
}

func TestMaxTSAge(t *testing.T) {
	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "")
	require.NoError(t, os.Unsetenv("DWS_MAX_TIMESTAMP_AGE"))
	assert.Equal(t, MaxTimestampAge, maxTSAge())

	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "60")
	assert.Equal(t, 60, maxTSAge())

	// garbage and non-positive values fall back to the default
	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "not-a-number")
	assert.Equal(t, MaxTimestampAge, maxTSAge())
	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "0")
	assert.Equal(t, MaxTimestampAge, maxTSAge())
	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "-1")
	assert.Equal(t, MaxTimestampAge, maxTSAge())

	// the replay window cannot be widened without bound
	t.Setenv("DWS_MAX_TIMESTAMP_AGE", "86400")
	assert.Equal(t, MaxTimestampAgeCeiling, maxTSAge())
}

func TestConnectWalletFutureTimestamp(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	// the victim was tricked into signing a message carrying a timestamp
	// years in the future
	authTS := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx, rec := connectWalletCtx(t, fmt.Sprintf(`{"code":"none","address":%q}`, addr))
	s := NewDelphiServer(nil)

	err = s.ConnectWallet(ctx, api.ConnectWalletParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", addr, authTS)),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "auth OK")
}

func TestGenerateCodeFutureTimestamp(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	authTS := time.Now().UTC().Add(time.Hour)
	req := httptest.NewRequest(http.MethodPost, "/affiliate/code", nil)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	ctx.SetPath("/affiliate/code")
	s := NewDelphiServer(nil)

	err = s.GenerateCode(ctx, api.GenerateCodeParams{
		DelphiKey:       addr,
		DelphiTs:        strconv.FormatInt(authTS.Unix(), 10),
		DelphiSignature: signTestMsg(t, key, formMsg("/affiliate/code", addr, authTS.Truncate(time.Second))),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// the wrapped database error -- table, function and column names, the postgres
// dialect and the driver version -- must never reach the caller
func TestGetErrorDoesNotLeakInternals(t *testing.T) {
	err := fmt.Errorf(
		"failed to fetch user data for 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, %w",
		errors.New("pq: function update_user_data(character varying) does not exist"))

	cerr := getError(106, msgInternalError, "failed to fetch user data", err)
	assert.Equal(t, 106, cerr.Code)
	assert.Equal(t, msgInternalError, cerr.Message)
	assert.NotContains(t, cerr.Message, "pq:")
	assert.NotContains(t, cerr.Message, "update_user_data")
	assert.NotContains(t, cerr.Message, "0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154")
}

// end to end through the handler: a database failure produces a 500 whose body
// carries the numeric code and nothing else
func TestUserDataErrorResponseIsSanitized(t *testing.T) {
	// nothing is listening there, so the very first query fails
	dbh, err := sqlx.Open("postgres", "postgres://dws:dws@127.0.0.1:1/dwsdb?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	defer func() { _ = dbh.Close() }()

	const addr = "0xDEd1Fe6B3f61c8F1d874bb86F086D10FFc3F0154"
	req := httptest.NewRequest(http.MethodGet, "/user/data/"+addr, nil)
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, rec)
	ctx.SetPath("/user/data/:address")

	s := NewDelphiServer(dbh)
	require.NoError(t, s.UserData(ctx, addr, api.UserDataParams{}))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	var cerr api.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cerr))
	assert.Equal(t, 106, cerr.Code)
	assert.Equal(t, msgInternalError, cerr.Message)
	for _, leak := range []string{"pq:", "dial tcp", "127.0.0.1", "dwsdb", "donation", "connection refused"} {
		assert.NotContains(t, body, leak, "the response body must not carry internal detail")
	}
}

func TestPageParams(t *testing.T) {
	limit, offset := pageParams(api.UserDataParams{})
	assert.Equal(t, DefaultUserDataLimit, limit)
	assert.Equal(t, 0, offset)

	l, o := 10, 20
	limit, offset = pageParams(api.UserDataParams{Limit: &l, Offset: &o})
	assert.Equal(t, 10, limit)
	assert.Equal(t, 20, offset)

	// the caller cannot ask for more than the server is willing to serve
	huge := 1000000
	limit, _ = pageParams(api.UserDataParams{Limit: &huge})
	assert.Equal(t, MaxUserDataLimit, limit)

	// nor for a degenerate page
	for _, bad := range []int{0, -1, -1000} {
		limit, _ = pageParams(api.UserDataParams{Limit: &bad})
		assert.Equal(t, 1, limit, "limit %d", bad)
	}
	for _, bad := range []int{-1, -1000} {
		_, offset = pageParams(api.UserDataParams{Offset: &bad})
		assert.Equal(t, 0, offset, "offset %d", bad)
	}
}

// the referral code is only handed out over the signature protected
// /affiliate/code path, never in the unauthenticated user data response
func TestUserDataResponseHasNoAffiliateCode(t *testing.T) {
	res := api.UserDataResult{
		Donations: []api.Donation{{
			Amount: "1.23", Asset: api.DonationAssetEth, Tokens: "980000",
			Price: "0.002", TxHash: "0xdeadbeef", Status: api.Confirmed,
		}},
		UserData: api.UserData{
			Total: "31415", Tokens: "9880000", Staked: "0", Reward: "0",
			Status: api.None,
		},
	}
	body, err := json.Marshal(res)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "affiliate_code")
	assert.NotContains(t, string(body), "us_code")
}

// GET /version used to assemble the build stamp into a package level variable
// on every request. The value never changed, but two concurrent requests still
// wrote it while a third read it -- a data race, which this test provokes
// under `go test -race`.
func TestVersionIsRaceFree(t *testing.T) {
	s := NewDelphiServer(nil)
	e := echo.New()

	const callers = 16
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/version", nil)
			rec := httptest.NewRecorder()
			assert.NoError(t, s.Version(e.NewContext(req, rec)))
			assert.Equal(t, http.StatusOK, rec.Code)
		}()
	}
	wg.Wait()

	// the value is still the build stamp and it is stable
	assert.Equal(t, versionString(), versionString())
	assert.True(t, strings.HasPrefix(versionString(), "delphi::"))
}

// an address without a `user_data` row must not be reported with empty strings
// for every amount and an empty `status`: `""` is not one of the values the
// enum in api/delphi.yaml declares
func TestDefaultUserDataIsSpecCompliant(t *testing.T) {
	ud := defaultUserData()
	assert.Equal(t, "0", ud.Total)
	assert.Equal(t, "0", ud.Tokens)
	assert.Equal(t, "0", ud.Staked)
	assert.Equal(t, "0", ud.Reward)
	assert.Equal(t, api.None, ud.Status)
	assert.Contains(t, []api.UserDataStatus{api.None, api.Staking, api.Unstaking}, ud.Status)
}

// the whole user data result carries the defaults, not the zero value of the
// struct
func TestUserDataResultDefaults(t *testing.T) {
	// nothing is listening there, so the read path is not exercised; the
	// defaults have to be in place before the first query regardless
	res := api.UserDataResult{UserData: defaultUserData()}
	body, err := json.Marshal(res)
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"status":""`)
	assert.Contains(t, string(body), `"status":"none"`)
}

// within the `delphi-ts` replay window the signature is a bearer credential:
// anyone who can read the log can replay a failed request verbatim
func TestSigDigestDoesNotRevealTheSignature(t *testing.T) {
	const sig = "0x93433430e249145433931dd4fda65090fcb250489e107d460b8adcef4a3c05f863c860d63c3f4ff92dcda4cce9755e4771e9dc6b91dafd5c900c7a5b99c169d71b"

	digest := sigDigest(sig)
	assert.NotContains(t, sig, digest)
	assert.NotContains(t, digest, sig)
	assert.Len(t, digest, 16)
	// stable, so log lines can still be correlated with each other
	assert.Equal(t, digest, sigDigest(sig))
	// and it discriminates
	assert.NotEqual(t, digest, sigDigest(sig[:len(sig)-1]+"c"))
	assert.Equal(t, "<empty>", sigDigest(""))
}

// verifySig logs on every rejection; none of those log lines may carry the
// signature that was rejected
func TestVerifySigDoesNotLogTheSignature(t *testing.T) {
	const (
		msg  = "aea3eb2f5a6a2efe002d3c88da52ba5a8702c9722ae2d67d1260e4318f5ccd6c"
		from = "0xb938F65DfE303EdF96A511F1e7E3190f69036860"
	)
	// a well formed signature of the right length that recovers to nobody,
	// plus one that is not hex at all
	for _, sig := range []string{
		"0x" + strings.Repeat("ff", crypto.SignatureLength),
		"not-hex-at-all",
	} {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(os.Stderr) })

		assert.False(t, verifySig(from, msg, sig))
		assert.NotContains(t, buf.String(), sig, "the raw signature must not be logged")
	}
}
