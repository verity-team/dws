package data

import (
	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
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
	params := common.HTTPParams{
		URL: "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=ETH-USDT",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Parse the JSON response
	var kuCoinResponse KuCoinResponse
	if err := json.Unmarshal(responseBody, &kuCoinResponse); err != nil {
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
