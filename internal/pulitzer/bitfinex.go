package pulitzer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type BitfinexTickerResponse struct {
	Mid string `json:"mid"`
}

func GetBitfinexETHPrice() (decimal.Decimal, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	bitfinexAPIURL := "https://api.bitfinex.com/v1/pubticker/ETHUSD"
	req, err := http.NewRequest("GET", bitfinexAPIURL, nil)
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
		return decimal.Zero, fmt.Errorf("bitfinex request failed with status: %s", response.Status)
	}

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
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

	log.Info("bitfinex: ", midValue)
	return midValue, nil
}
