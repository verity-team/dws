package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckQuoteAge(t *testing.T) {
	now := time.Now().UTC()

	assert.NoError(t, checkQuoteAge("test", now))
	assert.NoError(t, checkQuoteAge("test", now.Add(-MaxQuoteAge+10*time.Second)))
	// a little clock drift between the venue and us is tolerated
	assert.NoError(t, checkQuoteAge("test", now.Add(MaxQuoteSkew-5*time.Second)))

	assert.Error(t, checkQuoteAge("test", now.Add(-MaxQuoteAge-time.Second)))
	assert.Error(t, checkQuoteAge("test", now.Add(-40*time.Minute)))
	assert.Error(t, checkQuoteAge("test", now.Add(MaxQuoteSkew+time.Minute)))
	assert.Error(t, checkQuoteAge("test", time.Time{}))
}

func cexIOTicker(ts time.Time, last string) []byte {
	return []byte(fmt.Sprintf(
		`{"timestamp":"%d","low":"1990","high":"2010","last":"%s","volume":"1","volume30d":"1","bid":1999.5,"ask":2000.5,"priceChange":"1","priceChangePercentage":"0.1","pair":"ETH:USD"}`,
		ts.Unix(), last))
}

func TestParseCexIOTickerFresh(t *testing.T) {
	price, err := parseCexIOTicker(cexIOTicker(time.Now().UTC(), "2000.50"))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000.50").Equal(price))
}

// a cached quote tens of minutes old is usually still within the deviation
// gate around spot and would otherwise be averaged in at full weight
func TestParseCexIOTickerStale(t *testing.T) {
	price, err := parseCexIOTicker(cexIOTicker(time.Now().UTC().Add(-40*time.Minute), "2000.50"))
	assert.Error(t, err)
	assert.True(t, price.IsZero())
}

func TestParseCexIOTickerBadTimestamp(t *testing.T) {
	for _, body := range []string{
		`{"timestamp":"","last":"2000.50"}`,
		`{"timestamp":"not-a-timestamp","last":"2000.50"}`,
		`{"last":"2000.50"}`,
	} {
		price, err := parseCexIOTicker([]byte(body))
		assert.Error(t, err, body)
		assert.True(t, price.IsZero(), body)
	}
}

func TestParseCexIOTickerInvalidJSON(t *testing.T) {
	_, err := parseCexIOTicker([]byte(`{"error":"invalid symbol"`))
	assert.Error(t, err)
}

func kuCoinTicker(ts time.Time, price string) []byte {
	return []byte(fmt.Sprintf(
		`{"code":"200000","data":{"time":%d,"sequence":"1","price":"%s","size":"1","bestBid":"1999.5","bestBidSize":"1","bestAsk":"2000.5","bestAskSize":"1"}}`,
		ts.UnixMilli(), price))
}

func TestParseKuCoinTickerFresh(t *testing.T) {
	price, err := parseKuCoinTicker(kuCoinTicker(time.Now().UTC(), "2000.50"))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000.50").Equal(price))
}

func TestParseKuCoinTickerStale(t *testing.T) {
	price, err := parseKuCoinTicker(kuCoinTicker(time.Now().UTC().Add(-40*time.Minute), "2000.50"))
	assert.Error(t, err)
	assert.True(t, price.IsZero())
}

func TestParseKuCoinTickerMissingTimestamp(t *testing.T) {
	price, err := parseKuCoinTicker([]byte(`{"code":"200000","data":{"price":"2000.50"}}`))
	assert.Error(t, err)
	assert.True(t, price.IsZero())
}

func TestParseKuCoinTickerInvalidJSON(t *testing.T) {
	_, err := parseKuCoinTicker([]byte(`not json`))
	assert.Error(t, err)
}
