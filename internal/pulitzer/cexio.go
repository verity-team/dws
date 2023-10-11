package pulitzer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type CexIOResponse struct {
	Timestamp             string  `json:"timestamp"`
	Low                   string  `json:"low"`
	High                  string  `json:"high"`
	Last                  string  `json:"last"`
	Volume                string  `json:"volume"`
	Volume30d             string  `json:"volume30d"`
	Bid                   float64 `json:"bid"`
	Ask                   float64 `json:"ask"`
	PriceChange           string  `json:"priceChange"`
	PriceChangePercentage string  `json:"priceChangePercentage"`
	Pair                  string  `json:"pair"`
}

func GetCexIOETHUSDLastPrice() (decimal.Decimal, error) {
	// Create an HTTP client with a custom timeout
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	// Make an HTTP GET request to the Cex.io API
	resp, err := client.Get("https://cex.io/api/ticker/ETH/USD")
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("cexio request failed with status: %s", resp.Status)
	}

	// Parse the JSON response
	var data CexIOResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return decimal.Zero, err
	}

	// Convert the "last" price to a decimal
	lastPrice, err := decimal.NewFromString(data.Last)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("cex.io: ", lastPrice)
	return lastPrice, nil
}
