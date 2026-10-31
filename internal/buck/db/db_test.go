package db

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	c "github.com/verity-team/dws/internal/common"
)

// AUD4-1: the usd_amount of an ETH donation only exists once it is priced; one
// that does not fit donation.usd_amount NUMERIC(12,2) must be an error (the
// caller records the transaction in failed_tx) and not reach the upsert, where
// it would roll back the whole block.
func TestCalcTokensBoundsTheUSDAmount(t *testing.T) {
	tokenPrice := decimal.RequireFromString("0.001")
	ethPrice := decimal.RequireFromString("2000")

	for _, tc := range []struct {
		name    string
		asset   string
		value   string
		wantUSD string
		wantErr bool
	}{
		{name: "eth", asset: "eth", value: "1.50000000", wantUSD: "3000.00"},
		// 5e6 ETH at 2000 USD is 1e10 USD
		{name: "eth over the usd_amount bound", asset: "eth", value: "5000000.00000000", wantErr: true},
		{name: "eth just under the usd_amount bound", asset: "eth", value: "4999999.99999000", wantUSD: "9999999999.98"},
		{name: "stable coin", asset: "usdc", value: "224.9900000000", wantUSD: "224.99"},
		{name: "stable coin over the usd_amount bound", asset: "usdc", value: "10000000000.0000000000", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := c.Transaction{TXH: c.TXH{Hash: "0xabc"}, Asset: tc.asset, Value: tc.value}
			tokens, usd, err := calcTokens(tx, tokenPrice, ethPrice)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not fit donation.usd_amount NUMERIC(12,2)")
				assert.True(t, tokens.IsZero())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantUSD, usd.StringFixed(2))
			assert.True(t, usd.Div(tokenPrice).Ceil().Equal(tokens))
		})
	}
}
