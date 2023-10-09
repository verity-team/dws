package pulitzer

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// Struct to unmarshal the JSON response
type TickerData struct {
	Symbol           string `json:"symbol"`
	WeightedAvgPrice string `json:"weightedAvgPrice"`
}

func GetWeightedAvgPriceFromBinance() (decimal.Decimal, error) {
	// Define the Binance API URL
	binanceAPIURL := "https://api.binance.com/api/v3/ticker?symbol=ETHUSDT"

	// Create a context with a 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create an HTTP client with the context
	client := &http.Client{}

	// Create an HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, "GET", binanceAPIURL, nil)
	if err != nil {
		return decimal.Zero, err
	}

	// Send the HTTP request
	response, err := client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return decimal.Zero, err
	}

	// Define a struct to unmarshal the JSON response
	var tickerData TickerData

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &tickerData); err != nil {
		return decimal.Zero, err
	}

	// Extract the weightedAvgPrice from the struct
	weightedAvgPriceString := tickerData.WeightedAvgPrice

	// Parse the weightedAvgPriceString as a decimal
	weightedAvgPrice, err := decimal.NewFromString(weightedAvgPriceString)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("binance: ", weightedAvgPrice)
	return weightedAvgPrice, nil
}
