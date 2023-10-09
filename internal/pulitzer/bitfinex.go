package pulitzer

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type BitfinexTickerResponse struct {
	Mid string `json:"mid"`
}

func GetBitfinexETHPrice() (decimal.Decimal, error) {
	// Define the Bitfinex API URL
	bitfinexAPIURL := "https://api.bitfinex.com/v1/pubticker/ETHUSD"

	// Create a context with a 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create an HTTP client with the context
	client := &http.Client{}

	// Create an HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, "GET", bitfinexAPIURL, nil)
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
	var bitfinexResponse BitfinexTickerResponse

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &bitfinexResponse); err != nil {
		return decimal.Zero, err
	}

	// Parse the "mid" value as a decimal
	midValue, err := decimal.NewFromString(bitfinexResponse.Mid)
	if err != nil {
		return decimal.Zero, err
	}

	return midValue, nil
}
