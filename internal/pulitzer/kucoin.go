package pulitzer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type KuCoinResponse struct {
	Code string `json:"code"`
	Data Data   `json:"data"`
}

type Data struct {
	Time        int64  `json:"time"`
	Sequence    string `json:"sequence"`
	Price       string `json:"price"`
	Size        string `json:"size"`
	BestBid     string `json:"bestBid"`
	BestBidSize string `json:"bestBidSize"`
	BestAsk     string `json:"bestAsk"`
	BestAskSize string `json:"bestAskSize"`
}

func GetKuCoinETHUSDTPrice() (decimal.Decimal, error) {
	// Create an HTTP client with a custom timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Make an HTTP GET request to the KuCoin API
	resp, err := client.Get("https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=ETH-USDT")
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var kuCoinResponse KuCoinResponse
	err = json.NewDecoder(resp.Body).Decode(&kuCoinResponse)
	if err != nil {
		return decimal.Zero, err
	}

	// Convert the "price" to a decimal
	price, err := decimal.NewFromString(kuCoinResponse.Data.Price)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("kucoin: ", price)
	return price, nil
}
