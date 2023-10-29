package data

import (
	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
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
	params := common.HTTPParams{
		URL: "https://cex.io/api/ticker/ETH/USD",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Parse the JSON response
	var data CexIOResponse
	if err = json.Unmarshal(responseBody, &data); err != nil {
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
