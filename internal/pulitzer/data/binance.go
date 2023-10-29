package data

import (
	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// Struct to unmarshal the JSON response
type TickerData struct {
	Symbol           string `json:"symbol"`
	WeightedAvgPrice string `json:"weightedAvgPrice"`
}

func GetWeightedAvgPriceFromBinance() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.binance.com/api/v3/ticker?symbol=ETHUSDT&windowSize=1m",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Define a struct to unmarshal the JSON response
	var tickerData TickerData

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &tickerData); err != nil {
		return decimal.Zero, err
	}

	// Parse the weightedAvgPriceString as a decimal
	weightedAvgPrice, err := decimal.NewFromString(tickerData.WeightedAvgPrice)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("binance: ", weightedAvgPrice)
	return weightedAvgPrice, nil
}

func PingBinance() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL:              "https://api.binance.com/api/v3/ticker?symbol=ETHUSDT&windowSize=1m",
		MaxWaitInSeconds: 1,
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Define a struct to unmarshal the JSON response
	var tickerData TickerData

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &tickerData); err != nil {
		return decimal.Zero, err
	}

	// Parse the weightedAvgPriceString as a decimal
	weightedAvgPrice, err := decimal.NewFromString(tickerData.WeightedAvgPrice)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("binance: ", weightedAvgPrice)
	return weightedAvgPrice, nil
}
