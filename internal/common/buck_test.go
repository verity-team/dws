package common

import (
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
