package main

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a panicking price source must not take the process down; it is treated
// like any other failed source
func TestFetchPricePanicIsContained(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	assert.NotPanics(t, func() {
		fetchPrice("panicking", func() (decimal.Decimal, error) {
			var prices []decimal.Decimal
			return prices[0], nil
		}, ch)
	})
	_, ok := <-ch
	assert.False(t, ok, "expected the channel of a panicking source to be closed without a price")
}

func TestFetchPriceError(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	fetchPrice("failing", func() (decimal.Decimal, error) {
		return decimal.Zero, errors.New("boom")
	}, ch)
	_, ok := <-ch
	assert.False(t, ok, "expected the channel of a failing source to be closed without a price")
}

func TestFetchPriceSuccess(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	fetchPrice("working", func() (decimal.Decimal, error) {
		return decimal.RequireFromString("2000.50"), nil
	}, ch)
	price, ok := <-ch
	assert.True(t, ok)
	assert.True(t, decimal.RequireFromString("2000.50").Equal(price))
	_, ok = <-ch
	assert.False(t, ok, "expected the channel to be closed after the price was sent")
}

func sp(source, price string) sourcePrice {
	return sourcePrice{source: source, price: decimal.RequireFromString(price)}
}

func TestMedianPrice(t *testing.T) {
	// odd number of entries: the middle value
	assert.True(t, decimal.RequireFromString("2000").Equal(
		medianPrice(sortPrices([]sourcePrice{sp("a", "1990"), sp("b", "2010"), sp("c", "2000")}))))
	// even number of entries: the mean of the two middle values
	assert.True(t, decimal.RequireFromString("2000").Equal(
		medianPrice(sortPrices([]sourcePrice{sp("a", "1990"), sp("b", "2010"), sp("c", "1900"), sp("d", "2100")}))))
}

// the old pairwise gate divided by prices[i] and walked a slice built from a
// map, so 100 and 111 were 11% apart (reject) or 9.9% apart (accept) depending
// on the order the sources happened to be collected in
func TestCalculateAveragePriceIsOrderIndependent(t *testing.T) {
	prices := []sourcePrice{
		sp("binance", "2000"), sp("kraken", "2222"),
		sp("bitfinex", "2000"), sp("coinbase", "2000"),
		sp("cexio", "2000"), sp("kucoin", "2000"),
	}

	want, err := calculateAveragePrice(prices)
	require.NoError(t, err)

	// every rotation and the reverse order have to produce the same figure
	for i := range prices {
		rotated := append(append([]sourcePrice{}, prices[i:]...), prices[:i]...)
		got, err := calculateAveragePrice(rotated)
		require.NoError(t, err)
		assert.True(t, want.Equal(got), "rotation by %d: %s != %s", i, got, want)
	}
	reversed := make([]sourcePrice, 0, len(prices))
	for i := len(prices) - 1; i >= 0; i-- {
		reversed = append(reversed, prices[i])
	}
	got, err := calculateAveragePrice(reversed)
	require.NoError(t, err)
	assert.True(t, want.Equal(got))
}

// a single gross outlier is dropped, it no longer discards the whole cycle
func TestCalculateAveragePriceDropsOutlier(t *testing.T) {
	prices := []sourcePrice{
		sp("binance", "2000"), sp("kraken", "2010"), sp("bitfinex", "1990"),
		sp("coinbase", "2000"), sp("cexio", "2005"),
		// a stale quote, 50% off
		sp("kucoin", "3000"),
	}

	av, err := calculateAveragePrice(prices)
	require.NoError(t, err)
	// the mean of the five consistent sources, the outlier is not in it
	assert.True(t, decimal.RequireFromString("2001").Equal(av), "got %s", av)
}

func TestRejectOutliers(t *testing.T) {
	accepted := rejectOutliers(sortPrices([]sourcePrice{
		sp("binance", "2000"), sp("kraken", "2000"),
		sp("bitfinex", "2000"), sp("kucoin", "3000"),
	}))
	require.Equal(t, 3, len(accepted))
	for _, a := range accepted {
		assert.NotEqual(t, "kucoin", a.source)
	}
}

// exactly on the 10% boundary the source is still accepted
func TestRejectOutliersBoundary(t *testing.T) {
	accepted := rejectOutliers(sortPrices([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2200"),
	}))
	assert.Equal(t, 4, len(accepted))

	accepted = rejectOutliers(sortPrices([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2200.01"),
	}))
	assert.Equal(t, 3, len(accepted))
}

func TestCalculateAveragePriceQuorum(t *testing.T) {
	_, err := calculateAveragePrice([]sourcePrice{sp("a", "2000"), sp("b", "2000"), sp("c", "2000")})
	assert.Error(t, err, "three sources are below the quorum")

	av, err := calculateAveragePrice([]sourcePrice{sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2000")})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000").Equal(av))
}

// non-positive prices never make it into the average and do not blow up the
// median either
func TestCalculateAveragePriceIgnoresNonPositivePrices(t *testing.T) {
	_, err := calculateAveragePrice([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "0"), sp("d", "-1"),
	})
	assert.Error(t, err)

	av, err := calculateAveragePrice([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2000"), sp("e", "0"),
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000").Equal(av))
}

// too few mutually consistent sources left after the outliers were dropped
func TestCalculateAveragePriceTooFewAccepted(t *testing.T) {
	_, err := calculateAveragePrice([]sourcePrice{
		sp("a", "1000"), sp("b", "2000"), sp("c", "3000"), sp("d", "4000"),
	})
	assert.Error(t, err)
}

// readiness depends on the database and on nothing else: an exchange being
// slow or rate limiting us must not take an otherwise healthy pod out of
// service
func TestRunReadyProbeDependsOnTheDatabase(t *testing.T) {
	// nothing is listening there
	dbh, err := sqlx.Open("postgres", "postgres://dws:dws@127.0.0.1:1/dwsdb?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	defer func() { _ = dbh.Close() }()

	assert.Error(t, runReadyProbe(dbh))
}
