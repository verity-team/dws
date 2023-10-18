package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type Kline struct {
	ClosePrice decimal.Decimal `json:"closePrice"`
	Volume     uint64          `json:"volume"`
	CloseTime  time.Time       `json:"klineCloseTime"`
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
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&klines); err != nil {
		err = fmt.Errorf("error decoding JSON response, %v", err)
		return nil, err
	}
	return klines, nil
}
