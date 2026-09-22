package data

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// coinbasePair is the pair this source asks for; the response is checked
// against it.
const coinbasePair = "ETH-USD"

type CoinbaseResponse struct {
	Data struct {
		Amount   string `json:"amount"`
		Base     string `json:"base"`
		Currency string `json:"currency"`
	} `json:"data"`
}

func GetCoinbaseETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.coinbase.com/v2/prices/" + coinbasePair + "/spot",
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	price, err := parseCoinbaseSpot(responseBody)
	if err != nil {
		log.Error(err)
		return decimal.Zero, err
	}

	log.Info("coinbase: ", price)
	return price, nil
}

// parseCoinbaseSpot extracts the spot price from a coinbase prices response.
// The base and the quote currency the venue answered for are compared with the
// pair that was asked for; both are echoed in the response and neither used to
// be looked at.
func parseCoinbaseSpot(responseBody []byte) (decimal.Decimal, error) {
	// Parse the JSON response
	var data CoinbaseResponse
	if err := json.Unmarshal(responseBody, &data); err != nil {
		return decimal.Zero, err
	}

	if err := checkPair("coinbase", coinbasePair, data.Data.Base+"-"+data.Data.Currency); err != nil {
		return decimal.Zero, err
	}

	// Convert the price to a decimal
	price, err := decimal.NewFromString(data.Data.Amount)
	if err != nil {
		return decimal.Zero, fmt.Errorf("coinbase: invalid spot price ('%s'), %w", data.Data.Amount, err)
	}

	return price, nil
}
