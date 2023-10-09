package pulitzer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type CoinbaseResponse struct {
	Data struct {
		Amount   string `json:"amount"`
		Base     string `json:"base"`
		Currency string `json:"currency"`
	} `json:"data"`
}

func GetCoinbaseETHPrice() (decimal.Decimal, error) {
	// Create an HTTP client with a custom timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Make an HTTP GET request to the Coinbase API
	resp, err := client.Get("https://api.coinbase.com/v2/prices/ETH-USD/spot")
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var data CoinbaseResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return decimal.Zero, err
	}

	// Convert the price to a decimal
	price, err := decimal.NewFromString(data.Data.Amount)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("coinbase: ", price)
	return price, nil
}
