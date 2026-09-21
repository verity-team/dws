package common

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

const spJSON = `
[{"limit": 1050000000, "price": "0.001"}, {"limit": 2625000000, "price": "0.002"}, {"limit": 5250000000, "price": "0.0024"}]
`

func TestPriceBucket(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	a := c.priceBucket(decimal.NewFromInt(int64(0)))
	assert.Equal(t, decimal.NewFromFloat(0.001), a)
	a = c.priceBucket(decimal.NewFromInt(int64(1049999999)))
	assert.Equal(t, decimal.NewFromFloat(0.001), a)
	a = c.priceBucket(decimal.NewFromInt(int64(1050000000)))
	assert.Equal(t, decimal.NewFromFloat(0.002), a)
	a = c.priceBucket(decimal.NewFromInt(int64(2625000000)))
	assert.Equal(t, decimal.NewFromFloat(0.0024), a)
	a = c.priceBucket(decimal.NewFromInt(int64(5250000000)))
	assert.Equal(t, decimal.NewFromFloat(0.0024), a)
}

func TestTokenSaleLimit(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	a := c.TokenSaleLimit()
	assert.Equal(t, decimal.NewFromInt(int64(5250000000)), a)
}

func TestNewTokenPrice(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	af, np := c.NewTokenPrice(decimal.NewFromInt(int64(0)), decimal.NewFromInt(int64(0)))
	assert.False(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.001), np)
	af, np = c.NewTokenPrice(decimal.NewFromInt(int64(0)), decimal.NewFromInt(int64(1049999999)))
	assert.False(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.001), np)
	af, np = c.NewTokenPrice(decimal.NewFromInt(int64(1049999999)), decimal.NewFromInt(int64(1050000000)))
	assert.True(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.002), np)
}

const pendingTxJSON = `{
	"blockHash": null,
	"blockNumber": null,
	"from": "0xa0ebed29f62dfd4b3d83af9e79e3c23170d45621",
	"hash": "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7",
	"to": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	"transactionIndex": null
}`

const minedTxJSON = `{
	"blockHash": "0x634f3802299d60d38418bad02789a8b36a3051bc62a293b45deeafe5ec51fa34",
	"blockNumber": "0x119ab54",
	"from": "0xa0ebed29f62dfd4b3d83af9e79e3c23170d45621",
	"hash": "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7",
	"to": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	"transactionIndex": "0x14"
}`

// a transaction that is still in the mempool must decode without error --
// otherwise it kills the decode of the whole batch it is part of
func TestUnmarshalPendingTxByHash(t *testing.T) {
	var tx TxByHash
	err := json.Unmarshal([]byte(pendingTxJSON), &tx)
	assert.Nil(t, err)
	assert.True(t, tx.Pending)
	assert.Equal(t, PendingBlockNumber, tx.BlockNumber)
	assert.Equal(t, PendingTransactionIndex, tx.TransactionIndex)
	// the sentinel must not be mistaken for a real block
	assert.NotEqual(t, uint64(0), tx.BlockNumber)
	assert.Equal(t, "", tx.BlockHash)
	assert.Equal(t, "0xa0ebed29f62dfd4b3d83af9e79e3c23170d45621", tx.From)
	assert.Equal(t, "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7", tx.Hash)
	assert.Equal(t, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", tx.To)
}

func TestUnmarshalMinedTxByHash(t *testing.T) {
	var tx TxByHash
	err := json.Unmarshal([]byte(minedTxJSON), &tx)
	assert.Nil(t, err)
	assert.False(t, tx.Pending)
	assert.Equal(t, uint64(18459476), tx.BlockNumber)
	assert.Equal(t, uint64(20), tx.TransactionIndex)
	assert.Equal(t, "0x634f3802299d60d38418bad02789a8b36a3051bc62a293b45deeafe5ec51fa34", tx.BlockHash)
}

// an invalid (non-empty, non-hex) block number is still an error
func TestUnmarshalInvalidTxByHash(t *testing.T) {
	var tx TxByHash
	err := json.Unmarshal([]byte(`{"blockNumber": "zork", "transactionIndex": "0x1"}`), &tx)
	assert.NotNil(t, err)
	err = json.Unmarshal([]byte(`{"blockNumber": "0x1", "transactionIndex": "zork"}`), &tx)
	assert.NotNil(t, err)
}

func TestJudge(t *testing.T) {
	const (
		bh   = "0x634f3802299d60d38418bad02789a8b36a3051bc62a293b45deeafe5ec51fa34"
		mfbn = uint64(18459500)
	)
	tests := []struct {
		name     string
		tx       TxByHash
		expected TxVerdict
	}{
		{
			// a pending tx has no block: it may still be mined, failing it
			// would destroy a legitimate donation
			name:     "pending tx is never failed",
			tx:       TxByHash{Pending: true, BlockNumber: PendingBlockNumber, TransactionIndex: PendingTransactionIndex},
			expected: TxPending,
		},
		{
			// absence of evidence is not evidence of failure
			name:     "tx without finalized block data is never failed",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: bh},
			expected: TxNoFinalizedBlockData,
		},
		{
			name:     "tx in a block that is not finalized yet",
			tx:       TxByHash{BlockNumber: mfbn + 1, BlockHash: bh, FBBlockHash: bh, FBContainsTx: true, FBDataAvailable: true},
			expected: TxNotFinalizedYet,
		},
		{
			name:     "finalized block carries the tx",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: bh, FBBlockHash: bh, FBContainsTx: true, FBDataAvailable: true},
			expected: TxFinalize,
		},
		{
			name:     "finalized block does not carry the tx",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: bh, FBBlockHash: bh, FBContainsTx: false, FBDataAvailable: true},
			expected: TxFail,
		},
		{
			// providers may report either casing; a casing difference must
			// not cost a donor their donation
			name:     "block hash casing differs",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: strings.ToUpper(bh), FBBlockHash: bh, FBContainsTx: true, FBDataAvailable: true},
			expected: TxFinalize,
		},
		{
			name:     "block hash with surrounding whitespace",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: " " + bh + "\n", FBBlockHash: bh, FBContainsTx: true, FBDataAvailable: true},
			expected: TxFinalize,
		},
		{
			name:     "finalized block hash differs from the tx block hash",
			tx:       TxByHash{BlockNumber: 18459476, BlockHash: bh, FBBlockHash: "0xdeadbeef", FBContainsTx: true, FBDataAvailable: true},
			expected: TxFail,
		},
		{
			// the zero value must never be failed either
			name:     "zero value tx is never failed",
			tx:       TxByHash{},
			expected: TxNoFinalizedBlockData,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.tx.Judge(mfbn))
		})
	}
}

// the tx set of a finalized block is keyed by normalized hash -- addFBData
// relies on this to decide whether the block carries a transaction
func TestTXMapNormalizesHashes(t *testing.T) {
	const hash = "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7"
	fb := FinalizedBlock{Transactions: []string{strings.ToUpper(hash), " " + hash + " "}}
	txm := fb.TXMap()
	assert.Equal(t, 1, len(txm))
	assert.True(t, txm[NormalizeHash(hash)])
	assert.True(t, txm[hash])
}

func TestNormalizeHash(t *testing.T) {
	const hash = "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7"
	assert.Equal(t, hash, NormalizeHash(hash))
	assert.Equal(t, hash, NormalizeHash(strings.ToUpper(hash)))
	assert.Equal(t, hash, NormalizeHash("\t "+hash+" \n"))
	assert.Equal(t, "", NormalizeHash("   "))
}

const scJSON = `
[{"asset": "usdc", "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "scale": 6}, {"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 6}]
`

// the erc-20 ABI map buck is initialized with; only its keys matter here
func testAssets() map[string]abi.ABI {
	return map[string]abi.ABI{"usdc": {}, "usdt": {}}
}

func TestGetContextValidConfig(t *testing.T) {
	tests := []struct {
		name  string
		erc20 string
		sp    string
	}{
		{
			name:  "reference configuration",
			erc20: scJSON,
			sp:    spJSON,
		},
		{
			name:  "single stable coin, single sale param",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 18}]`,
			sp:    `[{"limit": 1, "price": "0.00001"}]`,
		},
		{
			name:  "lower case contract address",
			erc20: `[{"asset": "usdc", "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "scale": 6}]`,
			sp:    spJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxt, err := GetContext(tt.erc20, tt.sp, testAssets())
			assert.Nil(t, err)
			assert.NotNil(t, ctxt)
			assert.NotEmpty(t, ctxt.StableCoins)
			assert.NotEmpty(t, ctxt.SaleParams)
			assert.Equal(t, testAssets(), ctxt.ABI)
		})
	}
}

func TestGetContextInvalidConfig(t *testing.T) {
	tests := []struct {
		name  string
		erc20 string
		sp    string
		want  string
	}{
		{
			name:  "erc-20 json is not valid json",
			erc20: `{`,
			sp:    spJSON,
			want:  "error decoding erc-20 JSON data",
		},
		{
			name:  "sale param json is not valid json",
			erc20: scJSON,
			sp:    `{`,
			want:  "error decoding sale param JSON data",
		},
		{
			name:  "no stable coins",
			erc20: `[]`,
			sp:    spJSON,
			want:  "no erc-20 stable coins configured",
		},
		{
			name:  "scale missing",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7"}]`,
			sp:    spJSON,
			want:  "scale must be greater than zero, got 0",
		},
		{
			name:  "scale is zero",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 0}]`,
			sp:    spJSON,
			want:  "scale must be greater than zero, got 0",
		},
		{
			name:  "scale is negative",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": -6}]`,
			sp:    spJSON,
			want:  "scale must be greater than zero, got -6",
		},
		{
			name:  "one of several stable coins has a zero scale",
			erc20: `[{"asset": "usdc", "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "scale": 6}, {"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 0}]`,
			sp:    spJSON,
			want:  "erc-20 entry 'usdt' (0xdac17f958d2ee523a2206206994597c13d831ec7): scale must be greater than zero",
		},
		{
			name:  "contract address is invalid",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ecZ", "scale": 6}]`,
			sp:    spJSON,
			want:  "invalid contract address",
		},
		{
			name:  "contract address is too short",
			erc20: `[{"asset": "usdt", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec", "scale": 6}]`,
			sp:    spJSON,
			want:  "invalid contract address",
		},
		{
			name:  "contract address is empty",
			erc20: `[{"asset": "usdt", "address": "", "scale": 6}]`,
			sp:    spJSON,
			want:  "invalid contract address",
		},
		{
			name:  "contract address is missing",
			erc20: `[{"asset": "usdt", "scale": 6}]`,
			sp:    spJSON,
			want:  "invalid contract address",
		},
		{
			name:  "asset is unknown",
			erc20: `[{"asset": "dai", "address": "0x6B175474E89094C44Da98b954EedeAC495271d0F", "scale": 18}]`,
			sp:    spJSON,
			want:  "unknown asset 'dai', expected one of [usdc usdt]",
		},
		{
			name:  "asset has the wrong case",
			erc20: `[{"asset": "USDT", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 6}]`,
			sp:    spJSON,
			want:  "unknown asset 'USDT', expected one of [usdc usdt]",
		},
		{
			name:  "asset is missing",
			erc20: `[{"address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "scale": 6}]`,
			sp:    spJSON,
			want:  "unknown asset ''",
		},
		{
			name:  "no sale params",
			erc20: scJSON,
			sp:    `[]`,
			want:  "no token sale parameters configured",
		},
		{
			name:  "sale param price is zero",
			erc20: scJSON,
			sp:    `[{"limit": 1050000000, "price": "0"}]`,
			want:  "token price must be greater than zero, got 0",
		},
		{
			name:  "sale param price is missing",
			erc20: scJSON,
			sp:    `[{"limit": 1050000000}]`,
			want:  "token price must be greater than zero, got 0",
		},
		{
			name:  "sale param price is negative",
			erc20: scJSON,
			sp:    `[{"limit": 1050000000, "price": "-0.001"}]`,
			want:  "token price must be greater than zero, got -0.001",
		},
		{
			name:  "one of several sale params has a zero price",
			erc20: scJSON,
			sp:    `[{"limit": 1050000000, "price": "0.001"}, {"limit": 2625000000, "price": "0"}]`,
			want:  "sale parameter #2 (limit 2625000000): token price must be greater than zero",
		},
		{
			name:  "sale param limit is zero",
			erc20: scJSON,
			sp:    `[{"limit": 0, "price": "0.001"}]`,
			want:  "token limit must be greater than zero, got 0",
		},
		{
			name:  "sale param limit is negative",
			erc20: scJSON,
			sp:    `[{"limit": -1, "price": "0.001"}]`,
			want:  "token limit must be greater than zero, got -1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxt, err := GetContext(tt.erc20, tt.sp, testAssets())
			assert.Nil(t, ctxt)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// buck cannot process a single donation without an erc-20 ABI
func TestGetContextWithoutSupportedAssets(t *testing.T) {
	ctxt, err := GetContext(scJSON, spJSON, map[string]abi.ABI{})
	assert.Nil(t, ctxt)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "the ABI map is empty")
}
