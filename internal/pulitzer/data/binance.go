package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const MaxWaitInSeconds = 8

// Struct to unmarshal the JSON response
type TickerData struct {
	Symbol           string `json:"symbol"`
	WeightedAvgPrice string `json:"weightedAvgPrice"`
}

func GetWeightedAvgPriceFromBinance() (decimal.Decimal, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	binanceAPIURL := "https://api.binance.com/api/v3/ticker?symbol=ETHUSDT&windowSize=1m"
	req, err := http.NewRequest("GET", binanceAPIURL, nil)
	if err != nil {
		return decimal.Zero, err
	}

	// Send the HTTP request
	response, err := client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("binance request failed with status: %s", response.Status)
	}

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
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
