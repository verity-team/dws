package pulitzer

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/shopspring/decimal"
)

type KrakenTickerResponse struct {
	Error  []interface{} `json:"error"`
	Result map[string]struct {
		C []string `json:"c"`
	} `json:"result"`
}

func GetKrakenETHPrice() (decimal.Decimal, error) {
	// Define the Kraken API URL
	krakenAPIURL := "https://api.kraken.com/0/public/Ticker?pair=ETHUSD"

	// Create a context with a 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create an HTTP client with the context
	client := &http.Client{}

	// Create an HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, "GET", krakenAPIURL, nil)
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

	// Unmarshal the JSON response
	var krakenResponse KrakenTickerResponse
	if err := json.Unmarshal(responseBody, &krakenResponse); err != nil {
		return decimal.Zero, err
	}

	// Extract the ETH price from the "c" array (the first element)
	priceString := krakenResponse.Result["XETHZUSD"].C[0]

	// Parse the priceString as a decimal
	price, err := decimal.NewFromString(priceString)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("kraken: ", price)
	return price, nil
}
