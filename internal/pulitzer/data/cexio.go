package data

import (
	"fmt"
	"strconv"
	"time"

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

// cexIOPair is the pair this source asks for; the response is checked against
// it.
const cexIOPair = "ETH/USD"

func GetCexIOETHUSDLastPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://cex.io/api/ticker/" + cexIOPair,
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	lastPrice, err := parseCexIOTicker(responseBody)
	if err != nil {
		log.Error(err)
		return decimal.Zero, err
	}

	log.Info("cex.io: ", lastPrice)
	return lastPrice, nil
}

// parseCexIOTicker extracts the last price from a cex.io ticker response. The
// venue publishes the time the quote was taken; a quote that is no longer
// current is rejected instead of being averaged in at full weight. It also
// echoes the pair it answered for, which is checked against the pair that was
// asked for.
func parseCexIOTicker(responseBody []byte) (decimal.Decimal, error) {
	// Parse the JSON response
	var data CexIOResponse
	if err := json.Unmarshal(responseBody, &data); err != nil {
		return decimal.Zero, err
	}

	if err := checkPair("cex.io", cexIOPair, data.Pair); err != nil {
		return decimal.Zero, err
	}

	seconds, err := strconv.ParseInt(data.Timestamp, 10, 64)
	if err != nil {
		return decimal.Zero, fmt.Errorf("cex.io: invalid quote timestamp ('%s'), %w", data.Timestamp, err)
	}
	if err = checkQuoteAge("cex.io", time.Unix(seconds, 0).UTC()); err != nil {
		return decimal.Zero, err
	}

	// Convert the "last" price to a decimal
	lastPrice, err := decimal.NewFromString(data.Last)
	if err != nil {
		return decimal.Zero, err
	}

	return lastPrice, nil
}
