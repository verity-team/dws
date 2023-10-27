package data

import (
	"encoding/json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type BitfinexTickerResponse struct {
	Mid string `json:"mid"`
}

func GetBitfinexETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.bitfinex.com/v1/pubticker/ETHUSD",
	}
	responseBody, err := common.HTTPGet(params)
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
