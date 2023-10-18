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

type KrakenTickerResponse struct {
	Error  []interface{} `json:"error"`
	Result map[string]struct {
		C []string `json:"c"`
	} `json:"result"`
}

func GetKrakenETHPrice() (decimal.Decimal, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	krakenAPIURL := "https://api.kraken.com/0/public/Ticker?pair=ETHUSD"
	req, err := http.NewRequest("GET", krakenAPIURL, nil)
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
		return decimal.Zero, fmt.Errorf("kraken request failed with status: %s", response.Status)
	}

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
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
