package data

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// binanceSymbol is the pair this source asks for; the response is checked
// against it.
const binanceSymbol = "ETHUSDT"

// Struct to unmarshal the JSON response
type TickerData struct {
	Symbol string `json:"symbol"`
	// LastPrice is the most recent trade price, i.e. spot.
	LastPrice string `json:"lastPrice"`
}

// GetBinanceETHPrice returns binance's last traded ETH price.
//
// It deliberately does *not* use `weightedAvgPrice`: that is a volume
// weighted average over the trailing window, so it structurally lags spot and
// keeps lagging in exactly the situation where the price matters most, a fast
// move. Every other venue in the aggregate contributes a spot quote, and
// mixing a trailing average into a median over spot prices compares unlike
// quantities -- it also makes binance the source most likely to be rejected as
// an outlier during a move, which costs the cycle a source when it can least
// afford one.
func GetBinanceETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.binance.com/api/v3/ticker?symbol=" + binanceSymbol + "&windowSize=1m",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	price, err := parseBinanceTicker(responseBody)
	if err != nil {
		log.Error(err)
		return decimal.Zero, err
	}

	log.Info("binance: ", price)
	return price, nil
}

// parseBinanceTicker extracts the last price from a binance rolling window
// ticker response. The symbol the venue answered for is compared with the one
// that was asked for: a response for a different pair is a price for a
// different asset and must not be averaged in.
func parseBinanceTicker(responseBody []byte) (decimal.Decimal, error) {
	// Define a struct to unmarshal the JSON response
	var tickerData TickerData

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &tickerData); err != nil {
		return decimal.Zero, err
	}

	if err := checkPair("binance", binanceSymbol, tickerData.Symbol); err != nil {
		return decimal.Zero, err
	}

	// Parse the lastPrice as a decimal
	lastPrice, err := decimal.NewFromString(tickerData.LastPrice)
	if err != nil {
		return decimal.Zero, fmt.Errorf("binance: invalid last price ('%s'), %w", tickerData.LastPrice, err)
	}

	return lastPrice, nil
}
