package data

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// krakenOHLCPair is the altname spelling kraken keys the OHLC result by
// (`XETHZUSD` for `ETHUSD`). The pair the venue answered for is accepted under
// either spelling and rejected otherwise, so a response for a different
// instrument is never priced as ETH.
const krakenOHLCPair = "XETHZUSD"

// KrakenOHLCResponse mirrors kraken's /0/public/OHLC reply. The result is
// keyed by kraken's own altname spelling of the pair (`XETHZUSD` for
// `ETHUSD`) and carries a trailing `last` field alongside it, so the pair
// entry is picked out by shape rather than by a hardcoded key -- see
// parseKrakenOHLC.
type KrakenOHLCResponse struct {
	Error  []interface{}              `json:"error"`
	Result map[string]json.RawMessage `json:"result"`
}

// GetHistoricalPriceFromKraken returns the one-minute OHLC candles kraken
// reports at or after the requested minute.
//
// It is a fallback historical source: the backfill path used to be
// single-sourced to binance, so a binance-specific data gap or outage stalled
// donation finalization even though five other live feeds were healthy. The
// candles are validated against the requested minute by the caller (the same
// coverage gate CloseRequest applies), so a candle for a different minute is
// not mistaken for the price of the block that triggered the request.
func GetHistoricalPriceFromKraken(ctx context.Context, ts time.Time) ([]Kline, error) {
	// OHLC returns the candles at or after `since`; ask from one minute before
	// the requested minute so the candle that opens on it is included even at
	// the boundary. `since` is inclusive and in whole seconds.
	since := ts.UTC().Truncate(time.Minute).Add(-time.Minute).Unix()
	url := fmt.Sprintf(
		"https://api.kraken.com/0/public/OHLC?pair=%s&interval=1&since=%d",
		krakenPair, since)
	params := common.HTTPParams{
		URL: url,
	}
	responseBody, err := common.HTTPGetCtx(ctx, params)
	if err != nil {
		return nil, err
	}

	klines, err := parseKrakenOHLC(responseBody)
	if err != nil {
		err = fmt.Errorf("failed to parse kraken OHLC for %s, %w", ts.UTC().Format(time.RFC3339), err)
		log.Error(err)
		return nil, err
	}
	return klines, nil
}

// parseKrakenOHLC converts a kraken OHLC response into a non-empty slice of
// Kline values; an empty result is an error -- the caller has no prices to
// persist in that case.
//
// It borrows the two guards parseKrakenTicker learned the hard way: kraken
// answers HTTP 200 with a populated `error` array when the request failed, and
// the candles are looked up under the single pair entry taken from the
// response rather than a hardcoded legacy key that a rename would silently
// break. The `last` field kraken returns alongside the pair is not a pair
// entry and is skipped.
func parseKrakenOHLC(responseBody []byte) ([]Kline, error) {
	var resp KrakenOHLCResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, err
	}

	if len(resp.Error) > 0 {
		return nil, fmt.Errorf("kraken: error response, %v", resp.Error)
	}

	// pick the single pair entry: the request asks for exactly one pair, so
	// the one array-valued entry of the result is the answer. `last` is a bare
	// number (the id for incremental polling), not a candle array, and is
	// ignored.
	var (
		pair string
		raw  json.RawMessage
		hits int
		rows [][]interface{}
	)
	for k, v := range resp.Result {
		if k == "last" {
			continue
		}
		hits++
		pair = k
		raw = v
	}
	if hits != 1 {
		return nil, fmt.Errorf("kraken: expected the OHLC for exactly one pair ('%s'), got %d", krakenPair, hits)
	}
	// the key kraken answered under has to be the pair that was asked for,
	// spelled either as ETHUSD or as its altname XETHZUSD; anything else is a
	// different instrument. checkPair ignores the separator and casing.
	if checkPair("kraken", krakenPair, pair) != nil && checkPair("kraken", krakenOHLCPair, pair) != nil {
		return nil, fmt.Errorf("kraken: OHLC response is for pair '%s', expected '%s' (or its altname '%s')", pair, krakenPair, krakenOHLCPair)
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}

	var klines []Kline
	for _, row := range rows {
		// [ time, open, high, low, close, vwap, volume, count ]
		if len(row) < 8 {
			log.Error("kraken: OHLC: invalid data format, skipping entry")
			continue
		}
		openSec, ok := row[0].(float64)
		if !ok {
			return nil, fmt.Errorf("kraken: invalid candle time: '%v'", row[0])
		}
		closeStr, ok := row[4].(string)
		if !ok {
			return nil, fmt.Errorf("kraken: invalid close price: '%v'", row[4])
		}
		p, err := decimal.NewFromString(closeStr)
		if err != nil {
			return nil, fmt.Errorf("kraken: error decoding close price ('%s'), %w", closeStr, err)
		}
		volStr, ok := row[6].(string)
		if !ok {
			return nil, fmt.Errorf("kraken: invalid volume figure: '%v'", row[6])
		}
		v, err := decimal.NewFromString(volStr)
		if err != nil {
			return nil, fmt.Errorf("kraken: error decoding volume ('%s'), %w", volStr, err)
		}
		klines = append(klines, Kline{
			ClosePrice: p,
			Volume:     v,
			CloseTime:  closeTimeFromOpen(int64(openSec)),
		})
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("kraken: no valid candles in response (%d entries)", len(rows))
	}
	return klines, nil
}
