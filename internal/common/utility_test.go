package common

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertValue(t *testing.T) {
	_, err := HexStringToDecimal("not a number")
	assert.NotNil(t, err, "Expected an error")

	// an empty hex string is *not* zero; callers rely on this to detect the
	// `null` block number/transaction index of a pending transaction
	_, err = HexStringToDecimal("")
	assert.NotNil(t, err, "Expected an error")
	_, err = HexStringToDecimal("0x")
	assert.NotNil(t, err, "Expected an error")

	a, err := HexStringToDecimal("0xa")
	assert.Nil(t, err)
	assert.Equal(t, decimal.NewFromInt(10), a)

	a, err = HexStringToDecimal("b")
	assert.Nil(t, err)
	assert.Equal(t, decimal.NewFromInt(11), a)

	d, err := decimal.NewFromString("20000000000000000000")
	assert.Nil(t, err)
	a, err = HexStringToDecimal("1158e460913d00000")
	assert.Nil(t, err)
	assert.True(t, a.Equal(d))
}

// the jsonrpc API provider credentials live in ETH_RPC_URL; nothing beyond
// the host may reach a log line or an error message
func TestRedactURL(t *testing.T) {
	const projectID = "0123456789abcdef0123456789abcdef"

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"infura style path", "https://sepolia.infura.io/v3/" + projectID, "https://sepolia.infura.io/<redacted>"},
		{"key in the query string", "https://rpc.example.com/eth?apikey=" + projectID, "https://rpc.example.com/<redacted>"},
		{"credentials in the userinfo", "https://user:s3cret@rpc.example.com", "https://rpc.example.com/<redacted>"},
		{"host only", "https://api.kraken.com", "https://api.kraken.com"},
		{"host and port", "http://localhost:8545", "http://localhost:8545"},
		{"path without a secret", "https://api.binance.com/api/v3/ticker", "https://api.binance.com/<redacted>"},
		{"unparseable", "://%%not a url", "<redacted>"},
		{"no host", "not-a-url", "<redacted>"},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactURL(tc.in)
			assert.Equal(t, tc.want, got)
			assert.NotContains(t, got, projectID)
			assert.NotContains(t, got, "s3cret")
		})
	}
}

func TestValidateRPCURL(t *testing.T) {
	const projectID = "0123456789abcdef0123456789abcdef"

	for _, tc := range []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"https provider", "https://sepolia.infura.io/v3/" + projectID, false},
		{"https host only", "https://api.example.com", false},
		{"http localhost (local dev node)", "http://localhost:8545", false},
		{"http 127.0.0.1", "http://127.0.0.1:8545", false},
		{"http ipv6 loopback", "http://[::1]:8545", false},
		{"http non-loopback host rejected", "http://rpc.example.com/" + projectID, true},
		{"http public ip rejected", "http://8.8.8.8:8545", true},
		{"non-http scheme rejected", "ftp://rpc.example.com", true},
		{"empty rejected", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRPCURL(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				// the provider credentials must never leak into the error
				assert.NotContains(t, err.Error(), projectID)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToUint64(t *testing.T) {
	v, err := ToUint64(decimal.NewFromInt(0), "block number")
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), v)

	v, err = ToUint64(decimal.NewFromInt(18_000_000), "block number")
	assert.NoError(t, err)
	assert.Equal(t, uint64(18_000_000), v)

	// the largest value IntPart() can still render faithfully
	maxOK, err := decimal.NewFromString("9223372036854775807")
	require.NoError(t, err)
	v, err = ToUint64(maxOK, "block number")
	assert.NoError(t, err)
	assert.Equal(t, uint64(math.MaxInt64), v)

	// a negative value used to wrap around to ~1.8e19 instead of being
	// rejected
	_, err = ToUint64(decimal.NewFromInt(-1), "block number")
	assert.ErrorContains(t, err, "negative block number")

	// a value beyond the int64 range used to be truncated by IntPart() before
	// the conversion was even reached
	tooBig, err := decimal.NewFromString("9223372036854775808")
	require.NoError(t, err)
	_, err = ToUint64(tooBig, "transaction index")
	assert.ErrorContains(t, err, "transaction index")

	huge, err := decimal.NewFromString("340282366920938463463374607431768211456")
	require.NoError(t, err)
	_, err = ToUint64(huge, "block number")
	assert.Error(t, err)
}
