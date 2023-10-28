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

	var klines []Kline
	var klineData [][]interface{}
	if err = json.Unmarshal(responseBody, &klineData); err != nil {
		return nil, err
	}

	for _, kline := range klineData {
		if len(kline) != 12 {
			log.Error("binance: klines: invalid data format")
			continue
		}

		p, err := decimal.NewFromString(kline[4].(string))
		if err != nil {
			err = fmt.Errorf("error decoding close price ('%s') in JSON response, %w", kline[4].(string), err)
			return nil, err
		}
		v, err := decimal.NewFromString(kline[5].(string))
		if err != nil {
			err = fmt.Errorf("error decoding Volume ('%s') in JSON response, %w", kline[5].(string), err)
			return nil, err
		}
		ct := time.Unix(int64(kline[6].(float64)/1000), 0)
		k := Kline{
			ClosePrice: p,
			Volume:     v,
			CloseTime:  ct.UTC(),
		}
		klines = append(klines, k)
	}
	return klines, nil
}
