package data

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a kraken OHLC reply: the result is keyed by kraken's altname spelling of the
// pair and carries a trailing `last` field. Each row is
// [ time(open), open, high, low, close, vwap, volume, count ].
const krakenOHLC = `{"error":[],"result":{"XETHZUSD":[` +
	`[1790056080,"2736.83","2737.20","2735.83","2735.83","2736.50","42.29054680",39],` +
	`[1790056140,"2736.24","2737.22","2735.67","2736.77","2736.64","1.10933821",30]],` +
	`"last":1790058240}}`

func TestParseKrakenOHLCSuccess(t *testing.T) {
	klines, err := parseKrakenOHLC([]byte(krakenOHLC))
	require.NoError(t, err)
	require.Len(t, klines, 2)

	// close is index 4; the close time is the last second of the open minute
	assert.True(t, decimal.RequireFromString("2735.83").Equal(klines[0].ClosePrice), klines[0].ClosePrice.String())
	assert.Equal(t, time.Unix(1790056139, 0).UTC(), klines[0].CloseTime)
	assert.True(t, decimal.RequireFromString("42.29054680").Equal(klines[0].Volume))

	assert.True(t, decimal.RequireFromString("2736.77").Equal(klines[1].ClosePrice), klines[1].ClosePrice.String())
	assert.Equal(t, time.Unix(1790056199, 0).UTC(), klines[1].CloseTime)
}

// the single pair entry is name-checked: a response keyed by a different
// instrument is rejected instead of being priced as ETH
func TestParseKrakenOHLCRejectsAnotherPair(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(
		`{"error":[],"result":{"XXBTZUSD":[[1790056080,"1","1","1","65000.0","1","1",1]],"last":1}}`))
	assert.ErrorContains(t, err, "XXBTZUSD")
}

// the plain-pair spelling (ETHUSD) is accepted as well as the altname
func TestParseKrakenOHLCAcceptsThePlainPair(t *testing.T) {
	klines, err := parseKrakenOHLC([]byte(
		`{"error":[],"result":{"ETHUSD":[[1790056080,"1","1","1","2735.83","1","1",1]],"last":1}}`))
	require.NoError(t, err)
	require.Len(t, klines, 1)
	assert.True(t, decimal.RequireFromString("2735.83").Equal(klines[0].ClosePrice))
}

// kraken answers HTTP 200 with a populated `error` array on failure
func TestParseKrakenOHLCReportsTheErrorArray(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(`{"error":["EQuery:Unknown asset pair"],"result":{}}`))
	assert.ErrorContains(t, err, "EQuery:Unknown asset pair")
}

// the `last` field is not a pair entry: a result carrying only `last` has no
// candles, and one carrying two pairs is ambiguous
func TestParseKrakenOHLCRejectsAmbiguousResults(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(`{"error":[],"result":{"last":1}}`))
	assert.ErrorContains(t, err, "exactly one pair")

	_, err = parseKrakenOHLC([]byte(
		`{"error":[],"result":{"XETHZUSD":[[1790056080,"1","1","1","2","1","1",1]],` +
			`"XXBTZUSD":[[1790056080,"1","1","1","3","1","1",1]],"last":1}}`))
	assert.ErrorContains(t, err, "exactly one pair")
}

// an empty candle array (the venue had no data for the interval) is an error:
// the caller has nothing to persist
func TestParseKrakenOHLCEmptyCandles(t *testing.T) {
	klines, err := parseKrakenOHLC([]byte(`{"error":[],"result":{"XETHZUSD":[],"last":1}}`))
	assert.ErrorContains(t, err, "no valid candles")
	assert.Nil(t, klines)
}

func TestParseKrakenOHLCInvalidClosePrice(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(
		`{"error":[],"result":{"XETHZUSD":[[1790056080,"1","1","1","n/a","1","1",1]],"last":1}}`))
	assert.ErrorContains(t, err, "close price")
}

// rows that are not shaped like a candle are skipped; if all of them are,
// there is nothing to persist
func TestParseKrakenOHLCSkipsMalformedRows(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(`{"error":[],"result":{"XETHZUSD":[[1,"2"]],"last":1}}`))
	assert.ErrorContains(t, err, "no valid candles")
}

func TestParseKrakenOHLCInvalidJSON(t *testing.T) {
	_, err := parseKrakenOHLC([]byte(`not json`))
	assert.Error(t, err)
}
