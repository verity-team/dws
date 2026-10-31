package data

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a bitstamp OHLC reply, oldest first. Every field is a JSON string. The
// second candle's close carries more digits than a float64 can hold, to prove
// the close is parsed from the raw token rather than through a float.
const bitstampOHLC = `{"data": {"pair": "ETH/USD", "ohlc": [` +
	`{"timestamp": "1793186640", "open": "2360.63", "high": "2360.63", "low": "2360.47", "close": "2360.55", "volume": "0.592723"},` +
	`{"timestamp": "1793186700", "open": "2360.92", "high": "2360.92", "low": "2360.60", "close": "2360.6000000000001", "volume": "0.232971"}]}}`

func TestParseBitstampOHLCSuccess(t *testing.T) {
	klines, err := parseBitstampOHLC([]byte(bitstampOHLC))
	require.NoError(t, err)
	require.Len(t, klines, 2)

	// the close time is the last second of the open minute
	assert.True(t, decimal.RequireFromString("2360.55").Equal(klines[0].ClosePrice), klines[0].ClosePrice.String())
	assert.Equal(t, time.Unix(1793186699, 0).UTC(), klines[0].CloseTime)
	assert.True(t, decimal.RequireFromString("0.592723").Equal(klines[0].Volume))

	// the exact-token path: a float64 would have rounded this to 2360.6
	assert.Equal(t, "2360.6000000000001", klines[1].ClosePrice.String())
	assert.Equal(t, time.Unix(1793186759, 0).UTC(), klines[1].CloseTime)
}

// candles for another instrument, and an error reply (no `data` at all), are
// never taken for the ETH/USD price
func TestParseBitstampOHLCRejectsAnotherPair(t *testing.T) {
	_, err := parseBitstampOHLC([]byte(`{"data": {"pair": "BTC/USD", "ohlc": [{"timestamp": "1793186640", "close": "65000.00", "volume": "1"}]}}`))
	assert.ErrorContains(t, err, "pair 'BTC/USD'")

	_, err = parseBitstampOHLC([]byte(`{"code": "API0005", "errors": [{"message": "Invalid pair"}]}`))
	assert.ErrorContains(t, err, "pair ''")
}

// an empty candle list (the venue had no data for the interval) is an error:
// the caller has nothing to persist
func TestParseBitstampOHLCEmpty(t *testing.T) {
	klines, err := parseBitstampOHLC([]byte(`{"data": {"pair": "ETH/USD", "ohlc": []}}`))
	assert.ErrorContains(t, err, "no valid candles")
	assert.Nil(t, klines)
}

func TestParseBitstampOHLCInvalidTime(t *testing.T) {
	_, err := parseBitstampOHLC([]byte(`{"data": {"pair": "ETH/USD", "ohlc": [{"timestamp": "soon", "close": "2360.55", "volume": "1"}]}}`))
	assert.ErrorContains(t, err, "candle time")
}

func TestParseBitstampOHLCInvalidClosePrice(t *testing.T) {
	_, err := parseBitstampOHLC([]byte(`{"data": {"pair": "ETH/USD", "ohlc": [{"timestamp": "1793186640", "close": "n/a", "volume": "1"}]}}`))
	assert.ErrorContains(t, err, "close price")
}

// candles that miss a field are skipped; if all of them do, there is nothing
// to persist
func TestParseBitstampOHLCSkipsMalformedRows(t *testing.T) {
	_, err := parseBitstampOHLC([]byte(`{"data": {"pair": "ETH/USD", "ohlc": [{"timestamp": "1793186640", "volume": "1"}]}}`))
	assert.ErrorContains(t, err, "no valid candles")
}

func TestParseBitstampOHLCInvalidJSON(t *testing.T) {
	_, err := parseBitstampOHLC([]byte(`{`))
	assert.Error(t, err)
}
