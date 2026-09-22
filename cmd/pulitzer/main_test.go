package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/internal/pulitzer/data"
	"github.com/verity-team/dws/internal/pulitzer/db"
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

// --------------------------------------------------- historical fallback ---

// covering builds a single final kline that serves the minute ts.
func covering(ts time.Time, price string) []data.Kline {
	return []data.Kline{{
		ClosePrice: decimal.RequireFromString(price),
		CloseTime:  ts.Add(59 * time.Second),
	}}
}

// stubSource builds a historical source with a canned fetch result.
func stubSource(name string, kls []data.Kline, err error) historicalSource {
	return historicalSource{name: name, fetch: func(context.Context, time.Time) ([]data.Kline, error) {
		return kls, err
	}}
}

// sleepySource fetches only after `d`, unless its context is cancelled first --
// which is how a hung venue behaves under a spent chain budget.
func sleepySource(name string, d time.Duration, kls []data.Kline) historicalSource {
	return historicalSource{name: name, fetch: func(ctx context.Context, _ time.Time) ([]data.Kline, error) {
		select {
		case <-time.After(d):
			return kls, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
}

var (
	fallbackTS  = time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	fallbackNow = fallbackTS.Add(5 * time.Minute) // the requested minute is long closed
)

// binance fails to fetch -> the next source's price is used
func TestSelectHistoricalFallsThroughOnFetchError(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("no klines")),
		stubSource("kraken", covering(fallbackTS, "2000.5"), nil),
	}
	klines, name, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	require.Equal(t, "kraken", name)
	require.True(t, klines[0].ClosePrice.Equal(decimal.RequireFromString("2000.5")))
}

// binance returns klines that do not cover the minute (a data gap) -> the
// next source is tried; this is the single-sourced stall #228 removes
func TestSelectHistoricalFallsThroughOnAGap(t *testing.T) {
	// binance's only kline opens far outside the lookup window
	gap := []data.Kline{{
		ClosePrice: decimal.RequireFromString("2000.5"),
		CloseTime:  fallbackTS.Add(9 * time.Minute),
	}}
	sources := []historicalSource{
		stubSource("binance", gap, nil),
		stubSource("coinbase", covering(fallbackTS, "1999.75"), nil),
	}
	klines, name, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	require.Equal(t, "coinbase", name)
	require.True(t, klines[0].ClosePrice.Equal(decimal.RequireFromString("1999.75")))
}

// binance serves the minute -> it wins and the fallbacks are not consulted
func TestSelectHistoricalPrefersBinance(t *testing.T) {
	consulted := false
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000.5"), nil),
		{name: "kraken", fetch: func(context.Context, time.Time) ([]data.Kline, error) {
			consulted = true
			return nil, errors.New("should not be reached")
		}},
	}
	_, name, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	require.Equal(t, "binance", name)
	require.False(t, consulted, "a later source must not be consulted once one serves the minute")
}

// every source fails to serve the minute -> an error naming each reason, so
// the caller fails the request (which is then retried after FailedRetryDelay)
func TestSelectHistoricalAllSourcesFail(t *testing.T) {
	outOfWindow := []data.Kline{{
		ClosePrice: decimal.RequireFromString("2000.5"),
		CloseTime:  fallbackTS.Add(30 * time.Minute),
	}}
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("binance down")),
		stubSource("kraken", nil, errors.New("kraken down")),
		stubSource("coinbase", outOfWindow, nil),
	}
	_, _, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.Error(t, err)
	require.ErrorContains(t, err, "binance down")
	require.ErrorContains(t, err, "kraken down")
	require.ErrorContains(t, err, "coinbase")
}

// a chain of hung sources is bounded by the budget: it returns in ~budget (not
// the sum of the sources' own timeouts), with an error, so the request is
// failed and retried rather than the serve cycle hanging. The in-flight fetch
// is actually cancelled -- sleepySource returns via ctx.Done(), not by
// out-sleeping the budget.
func TestSelectHistoricalBudgetCapsAHungChain(t *testing.T) {
	const budget = 100 * time.Millisecond
	// each source sleeps far longer than the budget: the healthy path cancels
	// them at ~budget. The sleep is bounded (not time.Hour) so that if the
	// deadline is ever removed the test fails fast -- the first source then
	// answers after this delay instead of hanging.
	const hung = 3 * time.Second
	sources := []historicalSource{
		sleepySource("binance", hung, covering(fallbackTS, "2000.5")),
		sleepySource("kraken", hung, covering(fallbackTS, "2000.5")),
		sleepySource("coinbase", hung, covering(fallbackTS, "2000.5")),
	}

	start := time.Now()
	_, _, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, budget)
	elapsed := time.Since(start)

	require.Error(t, err, "a chain that never serves must fail, so the request is retried")
	require.Less(t, elapsed, 10*budget, "the chain must be bounded by the budget, not the sources' own timeouts")
}

// a healthy walk completes well under the budget: the fallbacks are reached
// quickly and the first that serves wins without waiting on any timeout
func TestSelectHistoricalHealthyWalkIsFast(t *testing.T) {
	const budget = 5 * time.Second
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("gap")),
		stubSource("kraken", nil, errors.New("gap")),
		sleepySource("coinbase", time.Millisecond, covering(fallbackTS, "1999.75")),
	}

	start := time.Now()
	_, name, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, budget)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, "coinbase", name)
	require.Less(t, elapsed, budget, "a healthy three-source walk must finish well inside the budget")
}

// ------------------------------------------------ stuck request alert ---

// a request open past PriceReqAlertAge is surfaced for alerting on every failed
// attempt, while a younger one is not; the request is never terminal either way
func TestStuckPriceRequestAlert(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	minute := now.Truncate(time.Minute).Add(-time.Minute)

	// younger than the threshold: no alert
	young := db.PriceReq{ID: 1, Time: minute, CreatedAt: now.Add(-db.PriceReqAlertAge + time.Minute)}
	msg, due := stuckPriceRequestAlert(young, now)
	assert.False(t, due, "a request younger than the alert age must not alert")
	assert.Empty(t, msg)

	// exactly at the threshold: alert (age is not less than the threshold)
	atAge := db.PriceReq{ID: 2, Time: minute, CreatedAt: now.Add(-db.PriceReqAlertAge)}
	_, due = stuckPriceRequestAlert(atAge, now)
	assert.True(t, due, "a request at the alert age must alert")

	// well past the threshold: alert, and the line identifies the minute
	old := db.PriceReq{ID: 3, Time: minute, CreatedAt: now.Add(-2 * db.PriceReqAlertAge)}
	msg, due = stuckPriceRequestAlert(old, now)
	require.True(t, due, "a request older than the alert age must alert")
	assert.Contains(t, msg, minute.UTC().Format(time.RFC3339))
	assert.Contains(t, msg, "needs attention")
}
