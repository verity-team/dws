package data

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// a valid binance kline entry: 12 fields, close price at index 4, volume at
// index 5 and the close time (milliseconds) at index 6
const validKline = `[[1700000000000,"1999.00","2001.00","1998.00","2000.50","12.5",1700000059999,"25006.25",42,"6.25","12503.12","0"]]`

func TestParseKlinesSuccess(t *testing.T) {
	klines, err := parseKlines([]byte(validKline))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(klines))
	assert.True(t, decimal.RequireFromString("2000.50").Equal(klines[0].ClosePrice))
	assert.True(t, decimal.RequireFromString("12.5").Equal(klines[0].Volume))
	assert.Equal(t, time.Unix(1700000059, 0).UTC(), klines[0].CloseTime)
}

// binance returns an empty array when there is no data for the interval
func TestParseKlinesEmptyResponse(t *testing.T) {
	klines, err := parseKlines([]byte(`[]`))
	assert.Error(t, err)
	assert.Nil(t, klines)
}

// entries that do not have 12 fields are skipped; if *all* of them are
// skipped there is nothing to persist
func TestParseKlinesAllEntriesInvalid(t *testing.T) {
	klines, err := parseKlines([]byte(`[["1999.00","2000.50"],[]]`))
	assert.Error(t, err)
	assert.Nil(t, klines)
}

func TestParseKlinesSomeEntriesInvalid(t *testing.T) {
	klines, err := parseKlines([]byte(`[["1999.00"],[1700000000000,"1999.00","2001.00","1998.00","2000.50","12.5",1700000059999,"25006.25",42,"6.25","12503.12","0"]]`))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(klines))
	assert.True(t, decimal.RequireFromString("2000.50").Equal(klines[0].ClosePrice))
}

func TestParseKlinesInvalidJSON(t *testing.T) {
	klines, err := parseKlines([]byte(`{"code":-1121,"msg":"Invalid symbol."}`))
	assert.Error(t, err)
	assert.Nil(t, klines)
}

func TestParseKlinesInvalidClosePrice(t *testing.T) {
	klines, err := parseKlines([]byte(`[[1700000000000,"1999.00","2001.00","1998.00","not a price","12.5",1700000059999,"25006.25",42,"6.25","12503.12","0"]]`))
	assert.Error(t, err)
	assert.Nil(t, klines)
}
