package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPair(t *testing.T) {
	// the same pair, spelled the way the different venues spell it
	for _, got := range []string{"ETHUSD", "ETH-USD", "ETH/USD", "ETH:USD", "eth_usd", "eth usd"} {
		assert.NoError(t, checkPair("venue", "ETH-USD", got), got)
	}
	// a genuinely different instrument
	for _, got := range []string{"BTCUSD", "ETHUSDT", "ETHEUR", "USDETH"} {
		assert.Error(t, checkPair("venue", "ETH-USD", got), got)
	}
	// and a response that does not say what it is for at all
	assert.ErrorContains(t, checkPair("venue", "ETH-USD", ""), "no pair in the response")
}

// ---------------------------------------------------------------- binance ---

const binanceTicker = `{
	"symbol": "ETHUSDT",
	"priceChange": "-8.1",
	"weightedAvgPrice": "2550.00",
	"openPrice": "2560.00",
	"lastPrice": "2600.55",
	"volume": "123.4"
}`

// binance used to contribute `weightedAvgPrice`, a trailing volume weighted
// average, while every other venue contributes spot
func TestParseBinanceTickerUsesTheLastPrice(t *testing.T) {
	price, err := parseBinanceTicker([]byte(binanceTicker))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2600.55").Equal(price), price.String())
	assert.False(t, decimal.RequireFromString("2550.00").Equal(price), "the VWAP must not be used")
}

func TestParseBinanceTickerRejectsAnotherSymbol(t *testing.T) {
	_, err := parseBinanceTicker([]byte(`{"symbol":"BTCUSDT","lastPrice":"65000.00"}`))
	assert.ErrorContains(t, err, "BTCUSDT")

	_, err = parseBinanceTicker([]byte(`{"lastPrice":"2600.55"}`))
	assert.ErrorContains(t, err, "no pair in the response")
}

func TestParseBinanceTickerInvalidPrice(t *testing.T) {
	_, err := parseBinanceTicker([]byte(`{"symbol":"ETHUSDT","lastPrice":"n/a"}`))
	assert.ErrorContains(t, err, "invalid last price")

	_, err = parseBinanceTicker([]byte(`not json`))
	assert.Error(t, err)
}

// ----------------------------------------------------------------- kraken ---

func TestParseKrakenTickerSuccess(t *testing.T) {
	price, err := parseKrakenTicker([]byte(`{"error":[],"result":{"XETHZUSD":{"c":["2600.55","0.5"]}}}`))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2600.55").Equal(price), price.String())
}

// the result used to be looked up under the hardcoded legacy key `XETHZUSD`;
// a rename would have been a permanent, silent loss of this source
func TestParseKrakenTickerTakesTheKeyFromTheResponse(t *testing.T) {
	price, err := parseKrakenTicker([]byte(`{"error":[],"result":{"ETHUSD":{"c":["2600.55","0.5"]}}}`))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2600.55").Equal(price), price.String())
}

// kraken answers HTTP 200 with a populated `error` array on failure
func TestParseKrakenTickerReportsTheErrorArray(t *testing.T) {
	_, err := parseKrakenTicker([]byte(`{"error":["EQuery:Unknown asset pair"],"result":{}}`))
	assert.ErrorContains(t, err, "EQuery:Unknown asset pair")

	// even when a result is served alongside it
	_, err = parseKrakenTicker([]byte(`{"error":["EService:Busy"],"result":{"XETHZUSD":{"c":["2600.55","0.5"]}}}`))
	assert.ErrorContains(t, err, "EService:Busy")
}

func TestParseKrakenTickerRejectsAmbiguousResults(t *testing.T) {
	_, err := parseKrakenTicker([]byte(`{"error":[],"result":{}}`))
	assert.ErrorContains(t, err, "exactly one pair")

	_, err = parseKrakenTicker([]byte(`{"error":[],"result":{"XETHZUSD":{"c":["1"]},"XXBTZUSD":{"c":["2"]}}}`))
	assert.ErrorContains(t, err, "exactly one pair")

	_, err = parseKrakenTicker([]byte(`{"error":[],"result":{"XETHZUSD":{"c":[]}}}`))
	assert.ErrorContains(t, err, "no last trade price")

	_, err = parseKrakenTicker([]byte(`{"error":[],"result":{"XETHZUSD":{"c":["n/a"]}}}`))
	assert.ErrorContains(t, err, "invalid last trade price")
}

// --------------------------------------------------------------- coinbase ---

func TestParseCoinbaseSpotSuccess(t *testing.T) {
	price, err := parseCoinbaseSpot([]byte(`{"data":{"amount":"2600.55","base":"ETH","currency":"USD"}}`))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2600.55").Equal(price), price.String())
}

func TestParseCoinbaseSpotRejectsAnotherPair(t *testing.T) {
	_, err := parseCoinbaseSpot([]byte(`{"data":{"amount":"65000.00","base":"BTC","currency":"USD"}}`))
	assert.ErrorContains(t, err, "BTC-USD")

	_, err = parseCoinbaseSpot([]byte(`{"data":{"amount":"2400.00","base":"ETH","currency":"EUR"}}`))
	assert.ErrorContains(t, err, "ETH-EUR")

	_, err = parseCoinbaseSpot([]byte(`{"data":{"amount":"2600.55"}}`))
	assert.Error(t, err)
}

func TestParseCoinbaseSpotInvalidPrice(t *testing.T) {
	_, err := parseCoinbaseSpot([]byte(`{"data":{"amount":"","base":"ETH","currency":"USD"}}`))
	assert.ErrorContains(t, err, "invalid spot price")
}

// ----------------------------------------------------------------- cex.io ---

// cexIOPairTicker is a fresh cex.io ticker for the given pair.
func cexIOPairTicker(pair string) []byte {
	return []byte(fmt.Sprintf(
		`{"timestamp":"%d","low":"1990","high":"2010","last":"2600.55","volume":"1","volume30d":"1","bid":1999.5,"ask":2000.5,"priceChange":"1","priceChangePercentage":"0.1","pair":"%s"}`,
		time.Now().UTC().Unix(), pair))
}

func TestParseCexIOTickerRejectsAnotherPair(t *testing.T) {
	_, err := parseCexIOTicker(cexIOPairTicker("BTC:USD"))
	assert.ErrorContains(t, err, "BTC:USD")

	// a response that does not say what it is for
	_, err = parseCexIOTicker(cexIOPairTicker(""))
	assert.ErrorContains(t, err, "no pair in the response")
}

// the venue spells it with a colon, the request with a slash
func TestParseCexIOTickerAcceptsTheRequestedPair(t *testing.T) {
	price, err := parseCexIOTicker(cexIOPairTicker("ETH:USD"))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2600.55").Equal(price), price.String())
}
