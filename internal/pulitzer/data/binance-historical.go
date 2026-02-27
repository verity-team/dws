package data

import (
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

func GetHistoricalPriceFromBinance(ts time.Time) ([]Kline, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=ETHUSDT&interval=1m&limit=10&startTime=%d", ts.UnixMilli())
	params := common.HTTPParams{
		URL: url,
	}
	responseBody, err := common.HTTPGet(params)
	if err != nil {
		return nil, err
	}

	var klineData [][]interface{}
	if err = json.Unmarshal(responseBody, &klineData); err != nil {
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
	return klines, nil
}
