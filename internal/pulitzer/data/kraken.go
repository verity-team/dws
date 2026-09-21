package data

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// krakenPair is the pair this source asks for. Kraken keys the result by its
// own, altname-mapped spelling of it (`XETHZUSD` for `ETHUSD`), so the key is
// taken from the response rather than assumed, see parseKrakenTicker.
const krakenPair = "ETHUSD"

type KrakenTickerResponse struct {
	Error  []interface{} `json:"error"`
	Result map[string]struct {
		C []string `json:"c"`
	} `json:"result"`
}

func GetKrakenETHPrice() (decimal.Decimal, error) {
	params := common.HTTPParams{
		URL: "https://api.kraken.com/0/public/Ticker?pair=" + krakenPair,
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return decimal.Zero, err
	}

	price, err := parseKrakenTicker(responseBody)
	if err != nil {
		log.Error(err)
		return decimal.Zero, err
	}

	log.Info("kraken: ", price)
	return price, nil
}

// parseKrakenTicker extracts the last trade price from a kraken ticker
// response.
//
// Two things the previous version got wrong:
//
//   - kraken answers HTTP 200 with a populated `error` array when the request
//     failed. Ignoring it turned an error response into "no price data", which
//     is the same outcome but hides why.
//   - the result used to be looked up under the hardcoded legacy key
//     `XETHZUSD`. The request asks for exactly one pair, so the single entry of
//     the result map *is* the answer; a miss under a hardcoded key would be a
//     permanent, silent loss of this source the day kraken renames it.
func parseKrakenTicker(responseBody []byte) (decimal.Decimal, error) {
	// Unmarshal the JSON response
	var krakenResponse KrakenTickerResponse
	if err := json.Unmarshal(responseBody, &krakenResponse); err != nil {
		return decimal.Zero, err
	}

	if len(krakenResponse.Error) > 0 {
		return decimal.Zero, fmt.Errorf("kraken: error response, %v", krakenResponse.Error)
	}
	if len(krakenResponse.Result) != 1 {
		return decimal.Zero, fmt.Errorf(
			"kraken: expected the ticker for exactly one pair ('%s'), got %d",
			krakenPair, len(krakenResponse.Result))
	}

	var priceString string
	for pair, ticker := range krakenResponse.Result {
		if len(ticker.C) == 0 {
			return decimal.Zero, fmt.Errorf("kraken: no last trade price for pair '%s'", pair)
		}
		// Extract the ETH price from the "c" array (the first element)
		priceString = ticker.C[0]
	}

	// Parse the priceString as a decimal
	price, err := decimal.NewFromString(priceString)
	if err != nil {
		return decimal.Zero, fmt.Errorf("kraken: invalid last trade price ('%s'), %w", priceString, err)
	}

	return price, nil
}
