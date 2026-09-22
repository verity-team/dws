package data

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a coinbase exchange candle reply, newest first. Each row is
// [ time(open), low, high, open, close, volume ] and every field is a JSON
// number. The second candle's close carries more digits than a float64 can
// hold, to prove the close is parsed from the raw token rather than through a
// float.
const coinbaseCandles = `[` +
	`[1790056260,2735.29,2737.77,2735.29,2737.03,141.19703289],` +
	`[1790056140,2734.9,2737.27,2736.25,2737.0300000000001,61.41534725]]`

func TestParseCoinbaseCandlesSuccess(t *testing.T) {
	klines, err := parseCoinbaseCandles([]byte(coinbaseCandles))
	require.NoError(t, err)
	require.Len(t, klines, 2)

	// close is index 4; the close time is the last second of the open minute
	assert.True(t, decimal.RequireFromString("2737.03").Equal(klines[0].ClosePrice), klines[0].ClosePrice.String())
	assert.Equal(t, time.Unix(1790056319, 0).UTC(), klines[0].CloseTime)
	assert.True(t, decimal.RequireFromString("141.19703289").Equal(klines[0].Volume))

	// the exact-token path: a float64 would have rounded this to 2737.03
	assert.Equal(t, "2737.0300000000001", klines[1].ClosePrice.String())
	assert.Equal(t, time.Unix(1790056199, 0).UTC(), klines[1].CloseTime)
}

// an empty array (the venue had no data for the interval) is an error: the
// caller has nothing to persist
func TestParseCoinbaseCandlesEmpty(t *testing.T) {
	klines, err := parseCoinbaseCandles([]byte(`[]`))
	assert.ErrorContains(t, err, "no valid candles")
	assert.Nil(t, klines)
}

func TestParseCoinbaseCandlesInvalidTime(t *testing.T) {
	_, err := parseCoinbaseCandles([]byte(`[["notanumber",1,2,3,2737.03,5]]`))
	assert.ErrorContains(t, err, "candle time")
}

func TestParseCoinbaseCandlesInvalidClosePrice(t *testing.T) {
	_, err := parseCoinbaseCandles([]byte(`[[1790056260,1,2,3,"n/a",5]]`))
	assert.ErrorContains(t, err, "close price")

	_, err = parseCoinbaseCandles([]byte(`[[1790056260,1,2,3,null,5]]`))
	assert.ErrorContains(t, err, "close price")
}

// rows that are not shaped like a candle are skipped; if all of them are,
// there is nothing to persist
func TestParseCoinbaseCandlesSkipsMalformedRows(t *testing.T) {
	_, err := parseCoinbaseCandles([]byte(`[[1790056260,1,2]]`))
	assert.ErrorContains(t, err, "no valid candles")
}

// the coinbase exchange endpoint answers a bad request with a JSON object,
// not an array
func TestParseCoinbaseCandlesInvalidJSON(t *testing.T) {
	_, err := parseCoinbaseCandles([]byte(`{"message":"NotFound"}`))
	assert.Error(t, err)
}

// decimalFromJSONNumber accepts a bare number and a quoted one, and rejects
// anything that is not a number instead of yielding zero
func TestDecimalFromJSONNumber(t *testing.T) {
	d, err := decimalFromJSONNumber([]byte(`2737.03`))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2737.03").Equal(d))

	d, err = decimalFromJSONNumber([]byte(`"2737.03"`))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2737.03").Equal(d))

	_, err = decimalFromJSONNumber([]byte(`"n/a"`))
	assert.ErrorContains(t, err, "not a number")
}
