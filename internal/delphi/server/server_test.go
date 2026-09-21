package server

import (
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
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
	assert.Equal(t, "wallet connection, 2023-10-23 18:45:19+00:00", formMsg("/wallet/connection", ts))
	assert.Equal(t, "affiliate code, 2023-10-23 18:45:19+00:00", formMsg("/affiliate/code", ts))
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
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", authTS.Truncate(time.Second))),
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
		DelphiSignature: signTestMsg(t, other, formMsg("/wallet/connection", authTS.Truncate(time.Second))),
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
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", authTS.Truncate(time.Second))),
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
		DelphiSignature: signTestMsg(t, key, formMsg("/wallet/connection", authTS.Truncate(time.Second))),
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
