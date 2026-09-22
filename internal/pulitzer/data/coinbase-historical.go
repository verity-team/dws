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

// GetHistoricalPriceFromCoinbase returns the one-minute candles the coinbase
// exchange reports around the requested minute.
//
// It is a fallback historical source, tried after binance and kraken so a gap
// on one venue no longer stalls donation finalization. Note the host: the
// candles live on the coinbase *exchange* API (api.exchange.coinbase.com),
// not the retail api.coinbase.com/v2 the live spot client uses -- the latter
// serves only the current spot price and carries no history at all.
//
// The candles are validated against the requested minute by the caller (the
// same coverage gate CloseRequest applies), so a candle for a different minute
// is not mistaken for the price of the block that triggered the request.
func GetHistoricalPriceFromCoinbase(ctx context.Context, ts time.Time) ([]Kline, error) {
	// the candle endpoint returns the buckets whose start falls in [start,
	// end]; bracket the requested minute by a minute on either side so the
	// bucket that opens on it is included at the boundary. Both bounds are RFC
	// 3339 timestamps.
	minute := ts.UTC().Truncate(time.Minute)
	url := fmt.Sprintf(
		"https://api.exchange.coinbase.com/products/%s/candles?granularity=60&start=%s&end=%s",
		coinbasePair,
		minute.Add(-time.Minute).Format(time.RFC3339),
		minute.Add(time.Minute).Format(time.RFC3339))
	params := common.HTTPParams{
		URL: url,
	}
	responseBody, err := common.HTTPGetCtx(ctx, params)
	if err != nil {
		return nil, err
	}

	klines, err := parseCoinbaseCandles(responseBody)
	if err != nil {
		err = fmt.Errorf("failed to parse coinbase candles for %s, %w", ts.UTC().Format(time.RFC3339), err)
		log.Error(err)
		return nil, err
	}
	return klines, nil
}

// parseCoinbaseCandles converts a coinbase exchange candle response into a
// non-empty slice of Kline values; an empty result is an error -- the caller
// has no prices to persist in that case.
//
// The candle fields are JSON numbers, not strings, so each row is decoded as
// raw JSON tokens and the close is parsed straight from its textual form with
// decimal.NewFromString. Going through float64 would round the money-path
// price; the raw token is exact.
func parseCoinbaseCandles(responseBody []byte) ([]Kline, error) {
	var rows [][]json.RawMessage
	if err := json.Unmarshal(responseBody, &rows); err != nil {
		return nil, err
	}

	var klines []Kline
	for _, row := range rows {
		// [ time, low, high, open, close, volume ]
		if len(row) < 6 {
			log.Error("coinbase: candles: invalid data format, skipping entry")
			continue
		}
		var openSec int64
		if err := json.Unmarshal(row[0], &openSec); err != nil {
			return nil, fmt.Errorf("coinbase: invalid candle time ('%s'), %w", string(row[0]), err)
		}
		p, err := decimalFromJSONNumber(row[4])
		if err != nil {
			return nil, fmt.Errorf("coinbase: error decoding close price, %w", err)
		}
		v, err := decimalFromJSONNumber(row[5])
		if err != nil {
			return nil, fmt.Errorf("coinbase: error decoding volume, %w", err)
		}
		klines = append(klines, Kline{
			ClosePrice: p,
			Volume:     v,
			CloseTime:  closeTimeFromOpen(openSec),
		})
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("coinbase: no valid candles in response (%d entries)", len(rows))
	}
	return klines, nil
}

// decimalFromJSONNumber parses a raw JSON number token into a decimal without
// going through float64. A JSON string ("2600.55") is accepted too -- the
// surrounding quotes are stripped -- so a venue that quotes its numbers is
// handled, but anything that is not a number is rejected rather than silently
// yielding zero.
func decimalFromJSONNumber(raw json.RawMessage) (decimal.Decimal, error) {
	s := string(raw)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("'%s' is not a number, %w", string(raw), err)
	}
	return d, nil
}
