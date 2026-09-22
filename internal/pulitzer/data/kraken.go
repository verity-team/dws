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

// krakenAltNamePair is the altname spelling kraken keys its ticker and OHLC
// results by (`XETHZUSD` for `ETHUSD`). A result is accepted under either
// spelling and rejected under any other, so a response for a different
// instrument is never priced as ETH.
const krakenAltNamePair = "XETHZUSD"

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
// Three things the previous version got wrong:
//
//   - kraken answers HTTP 200 with a populated `error` array when the request
//     failed. Ignoring it turned an error response into "no price data", which
//     is the same outcome but hides why.
//   - the result used to be looked up under the hardcoded legacy key
//     `XETHZUSD`. The request asks for exactly one pair, so the single entry of
//     the result map *is* the answer; a miss under a hardcoded key would be a
//     permanent, silent loss of this source the day kraken renames it.
//   - that single entry was never checked against the pair that was asked for:
//     kraken keys it by its own spelling of the pair, so it is now checked
//     under either spelling and a response for a different instrument is
//     rejected rather than averaged into the ethereum price.
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

	var (
		pair        string
		priceString string
	)
	for p, ticker := range krakenResponse.Result {
		if len(ticker.C) == 0 {
			return decimal.Zero, fmt.Errorf("kraken: no last trade price for pair '%s'", p)
		}
		pair = p
		// Extract the ETH price from the "c" array (the first element)
		priceString = ticker.C[0]
	}
	// the key kraken answered under has to be the pair that was asked for,
	// spelled either as ETHUSD or as its altname XETHZUSD; anything else is a
	// different instrument. checkPair ignores the separator and casing.
	if checkPair("kraken", krakenPair, pair) != nil && checkPair("kraken", krakenAltNamePair, pair) != nil {
		return decimal.Zero, fmt.Errorf(
			"kraken: ticker response is for pair '%s', expected '%s' (or its altname '%s')",
			pair, krakenPair, krakenAltNamePair)
	}

	// Parse the priceString as a decimal
	price, err := decimal.NewFromString(priceString)
	if err != nil {
		return decimal.Zero, fmt.Errorf("kraken: invalid last trade price ('%s'), %w", priceString, err)
	}

	return price, nil
}
