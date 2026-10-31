package data

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

// bitstampURLPair is how the OHLC endpoint spells the pair in its path, and
// bitstampPair how its response names it; the latter is checked so a candle
// set for another instrument is never mistaken for the ETH/USD price.
const (
	bitstampURLPair = "ethusd"
	bitstampPair    = "ETH/USD"
)

// GetHistoricalPriceFromBitstamp returns the one-minute candles the bitstamp
// exchange reports around the requested minute.
//
// It is the fallback historical source with deep history: kraken's OHLC
// endpoint only ever returns its latest 720 one-minute candles -- whatever
// `since` asks for -- so a minute older than ~12 hours can only be served by
// binance, coinbase and bitstamp. Without a third deep venue, a backfill after
// a long price outage would need binance and coinbase to both answer and agree
// (see selectHistorical), and a single one of them being down or geo-blocked
// would stall donation finalization.
//
// The candles are validated against the requested minute by the caller
// (db.MinuteKline), so a candle for a different minute is not mistaken for
// the price of the block that triggered the request.
func GetHistoricalPriceFromBitstamp(ctx context.Context, ts time.Time) ([]Kline, error) {
	// `start` is inclusive and in whole seconds; ask from one minute before
	// the requested one so the candle that opens on it is included at the
	// boundary, and for three candles so the minute after it is there too
	// (it is dropped by the caller while it is still open).
	minute := ts.UTC().Truncate(time.Minute)
	url := fmt.Sprintf(
		"https://www.bitstamp.net/api/v2/ohlc/%s/?step=60&limit=3&start=%d",
		bitstampURLPair, minute.Add(-time.Minute).Unix())
	params := common.HTTPParams{
		URL: url,
	}
	responseBody, err := common.HTTPGetCtx(ctx, params)
	if err != nil {
		return nil, err
	}

	klines, err := parseBitstampOHLC(responseBody)
	if err != nil {
		err = fmt.Errorf("failed to parse bitstamp candles for %s, %w", ts.UTC().Format(time.RFC3339), err)
		log.Error(err)
		return nil, err
	}
	return klines, nil
}

type bitstampOHLCResponse struct {
	Data struct {
		Pair string `json:"pair"`
		OHLC []struct {
			Timestamp string          `json:"timestamp"`
			Close     json.RawMessage `json:"close"`
			Volume    json.RawMessage `json:"volume"`
		} `json:"ohlc"`
	} `json:"data"`
}

// parseBitstampOHLC converts a bitstamp OHLC response into a non-empty slice
// of Kline values; an empty result is an error -- the caller has no prices to
// persist in that case. The pair the response names must be ETH/USD: an error
// reply carries no `data` and fails that check as well.
//
// The candle fields are JSON strings; the close is parsed straight from its
// textual form (decimalFromJSONNumber), never through float64.
func parseBitstampOHLC(responseBody []byte) ([]Kline, error) {
	var resp bitstampOHLCResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, err
	}
	if resp.Data.Pair != bitstampPair {
		return nil, fmt.Errorf("bitstamp: candles for pair '%s', expected '%s'", resp.Data.Pair, bitstampPair)
	}

	var klines []Kline
	for _, c := range resp.Data.OHLC {
		if c.Timestamp == "" || len(c.Close) == 0 || len(c.Volume) == 0 {
			log.Error("bitstamp: candles: invalid data format, skipping entry")
			continue
		}
		openSec, err := strconv.ParseInt(c.Timestamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bitstamp: invalid candle time ('%s'), %w", c.Timestamp, err)
		}
		p, err := decimalFromJSONNumber(c.Close)
		if err != nil {
			return nil, fmt.Errorf("bitstamp: error decoding close price, %w", err)
		}
		v, err := decimalFromJSONNumber(c.Volume)
		if err != nil {
			return nil, fmt.Errorf("bitstamp: error decoding volume, %w", err)
		}
		klines = append(klines, Kline{
			ClosePrice: p,
			Volume:     v,
			CloseTime:  closeTimeFromOpen(openSec),
		})
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("bitstamp: no valid candles in response (%d entries)", len(resp.Data.OHLC))
	}
	return klines, nil
}
