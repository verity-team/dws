package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type Kline struct {
	ClosePrice decimal.Decimal
	Volume     decimal.Decimal
	CloseTime  time.Time
}

func GetHistoricalPriceFromBinance(ts time.Time) ([]Kline, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=ETHUSDT&interval=1m&limit=10&startTime=%d", ts.UnixMilli())
	timeout := MaxWaitInSeconds * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("error creating HTTP request, %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("error sending HTTP request, %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Received non-200 status code, %v", resp.Status)
		return nil, err
	}

	var klines []Kline
	var klineData [][]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&klineData); err != nil {
		err = fmt.Errorf("error decoding JSON response, %v", err)
		return nil, err
	}

	for _, kline := range klineData {
		if len(kline) != 12 {
			fmt.Println("Invalid data format")
			continue
		}

		p, err := decimal.NewFromString(kline[4].(string))
		if err != nil {
			err = fmt.Errorf("error decoding close price ('%s') in JSON response, %v", kline[4].(string), err)
			return nil, err
		}
		v, err := decimal.NewFromString(kline[5].(string))
		if err != nil {
			err = fmt.Errorf("error decoding Volume ('%s') in JSON response, %v", kline[5].(string), err)
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
