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

type Kline struct {
	ClosePrice decimal.Decimal
	Volume     decimal.Decimal
	CloseTime  time.Time
}

// closeTimeFromOpen turns a candle's open time (whole seconds, minute aligned)
// into the close time the rest of the pipeline keys on: the last whole second
// of the minute, i.e. minute start + 59s.
//
// It exists so the fallback sources agree with binance on the timestamp of a
// given minute. binance reports a kline's own close time (minute start +
// 59.999s, truncated to the second by parseKlines), whereas kraken and
// coinbase report the open time; without normalising them the same minute
// would land under two different `created_at` values and the
// UNIQUE(asset, created_at) dedupe -- and the overlap tolerance in
// persistKline -- would both break.
func closeTimeFromOpen(openSec int64) time.Time {
	return time.Unix(openSec, 0).UTC().Truncate(time.Minute).Add(59 * time.Second)
}

func GetHistoricalPriceFromBinance(ctx context.Context, ts time.Time) ([]Kline, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=ETHUSDT&interval=1m&limit=10&startTime=%d", ts.UnixMilli())
	params := common.HTTPParams{
		URL: url,
	}
	responseBody, err := common.HTTPGetCtx(ctx, params)
	if err != nil {
		return nil, err
	}

	klines, err := parseKlines(responseBody)
	if err != nil {
		err = fmt.Errorf("failed to parse binance klines for %s, %w", ts.UTC().Format(time.RFC3339), err)
		log.Error(err)
		return nil, err
	}
	return klines, nil
}

// parseKlines converts a binance klines response into a non-empty slice of
// Kline values; an empty result is an error -- the caller has no prices to
// persist in that case.
func parseKlines(responseBody []byte) ([]Kline, error) {
	var klineData [][]interface{}
	if err := json.Unmarshal(responseBody, &klineData); err != nil {
		return nil, err
	}

	var klines []Kline
	for _, kline := range klineData {
		if len(kline) != 12 {
			log.Error("binance: klines: invalid data format, skipping entry")
			continue
		}

		datum, ok := kline[4].(string)
		if !ok {
			return nil, fmt.Errorf("invalid close price: '%v'", kline[4])
		}
		p, err := decimal.NewFromString(datum)
		if err != nil {
			err = fmt.Errorf("error decoding close price ('%s') in JSON response, %w", datum, err)
			return nil, err
		}
		datum, ok = kline[5].(string)
		if !ok {
			return nil, fmt.Errorf("invalid volume figure: '%v'", kline[5])
		}
		v, err := decimal.NewFromString(datum)
		if err != nil {
			err = fmt.Errorf("error decoding Volume ('%s') in JSON response, %w", datum, err)
			return nil, err
		}
		fval, ok := kline[6].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid closing time: '%v'", kline[6])
		}
		ct := time.Unix(int64(fval/1000), 0)
		klines = append(klines, Kline{
			ClosePrice: p,
			Volume:     v,
			CloseTime:  ct.UTC(),
		})
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("no valid klines in response (%d entries)", len(klineData))
	}
	return klines, nil
}
