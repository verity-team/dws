package data

import (
	"encoding/json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type CoinbaseResponse struct {
	Data struct {
		Amount   string `json:"amount"`
		Base     string `json:"base"`
		Currency string `json:"currency"`
	} `json:"data"`
}

func GetCoinbaseETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.coinbase.com/v2/prices/ETH-USD/spot",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Parse the JSON response
	var data CoinbaseResponse
	if err = json.Unmarshal(responseBody, &data); err != nil {
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
