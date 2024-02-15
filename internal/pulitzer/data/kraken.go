package data

import (
	"errors"

	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type KrakenTickerResponse struct {
	Error  []interface{} `json:"error"`
	Result map[string]struct {
		C []string `json:"c"`
	} `json:"result"`
}

func GetKrakenETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.kraken.com/0/public/Ticker?pair=ETHUSD",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	// Unmarshal the JSON response
	var krakenResponse KrakenTickerResponse
	if err := json.Unmarshal(responseBody, &krakenResponse); err != nil {
		return decimal.Zero, err
	}

	var priceString string
	if len(krakenResponse.Result["XETHZUSD"].C) > 0 {
		// Extract the ETH price from the "c" array (the first element)
		priceString = krakenResponse.Result["XETHZUSD"].C[0]
	} else {
		return decimal.Zero, errors.New("invalid price data received from kraken")
	}

	// Parse the priceString as a decimal
	price, err := decimal.NewFromString(priceString)
	if err != nil {
		return decimal.Zero, err
	}

	log.Info("kraken: ", price)
	return price, nil
}
