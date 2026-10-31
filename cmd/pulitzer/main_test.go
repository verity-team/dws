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
func TestCalculatePriceIsOrderIndependent(t *testing.T) {
	prices := []sourcePrice{
		sp("binance", "2000"), sp("kraken", "2222"),
		sp("bitfinex", "2000"), sp("coinbase", "2000"),
		sp("cexio", "2000"), sp("kucoin", "2000"),
	}

	want, err := calculatePrice(prices)
	require.NoError(t, err)

	// every rotation and the reverse order have to produce the same figure
	for i := range prices {
		rotated := append(append([]sourcePrice{}, prices[i:]...), prices[:i]...)
		got, err := calculatePrice(rotated)
		require.NoError(t, err)
		assert.True(t, want.Equal(got), "rotation by %d: %s != %s", i, got, want)
	}
	reversed := make([]sourcePrice, 0, len(prices))
	for i := len(prices) - 1; i >= 0; i-- {
		reversed = append(reversed, prices[i])
	}
	got, err := calculatePrice(reversed)
	require.NoError(t, err)
	assert.True(t, want.Equal(got))
}

// a single gross outlier is dropped, it no longer discards the whole cycle
func TestCalculatePriceDropsOutlier(t *testing.T) {
	prices := []sourcePrice{
		sp("binance", "2000"), sp("kraken", "2010"), sp("bitfinex", "1990"),
		sp("coinbase", "2000"), sp("cexio", "2005"),
		// a stale quote, 50% off
		sp("kucoin", "3000"),
	}

	av, err := calculatePrice(prices)
	require.NoError(t, err)
	// the median of the five consistent sources, the outlier is not in it
	assert.True(t, decimal.RequireFromString("2000").Equal(av), "got %s", av)
}

// the written price is the median of the accepted sources, not their mean: a
// rogue source that stays inside the band does not move it
func TestCalculatePriceIsTheMedianOfTheAccepted(t *testing.T) {
	av, err := calculatePrice([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2000"),
		// 9.5% high, still accepted
		sp("e", "2190"),
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000").Equal(av), "got %s", av)

	// an even number of accepted sources: the mean of the two middle values
	av, err = calculatePrice([]sourcePrice{
		sp("a", "1990"), sp("b", "2000"), sp("c", "2010"), sp("d", "2020"),
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2005").Equal(av), "got %s", av)
}

// two camps of two, [a,a,b,b]: the median sits between them, so each source
// passes the per-source gate up to b = ~1.22a; the round has to be refused on
// the spread of the accepted sources instead (e.g. a USDT depeg moving both
// USDT venues by 17.6%)
func TestCalculatePriceRejectsASplitRound(t *testing.T) {
	prices := []sourcePrice{
		sp("binance", "2352"), sp("kucoin", "2352"),
		sp("kraken", "2000"), sp("coinbase", "2000"),
	}
	require.Equal(t, 4, len(rejectOutliers(sortPrices(prices))), "every source passes the per-source gate")
	_, err := calculatePrice(prices)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spread")
}

// a spread of exactly maxPriceDeviation around the median is still accepted,
// one just beyond it is not
func TestCalculatePriceSpreadBoundary(t *testing.T) {
	// median 2000, spread 200 = 10%
	_, err := calculatePrice([]sourcePrice{
		sp("a", "1900"), sp("b", "2000"), sp("c", "2000"), sp("d", "2100"),
	})
	require.NoError(t, err)

	_, err = calculatePrice([]sourcePrice{
		sp("a", "1900"), sp("b", "2000"), sp("c", "2000"), sp("d", "2100.01"),
	})
	require.Error(t, err)
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

func TestCalculatePriceQuorum(t *testing.T) {
	_, err := calculatePrice([]sourcePrice{sp("a", "2000"), sp("b", "2000"), sp("c", "2000")})
	assert.Error(t, err, "three sources are below the quorum")

	av, err := calculatePrice([]sourcePrice{sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2000")})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000").Equal(av))
}

// non-positive prices never make it into the price and do not blow up the
// median either
func TestCalculatePriceIgnoresNonPositivePrices(t *testing.T) {
	_, err := calculatePrice([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "0"), sp("d", "-1"),
	})
	assert.Error(t, err)

	av, err := calculatePrice([]sourcePrice{
		sp("a", "2000"), sp("b", "2000"), sp("c", "2000"), sp("d", "2000"), sp("e", "0"),
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2000").Equal(av))
}

// too few mutually consistent sources left after the outliers were dropped
func TestCalculatePriceTooFewAccepted(t *testing.T) {
	_, err := calculatePrice([]sourcePrice{
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

// requireQuote asserts which source's kline won, who corroborated it and the
// price that gets persisted.
func requireQuote(t *testing.T, q historicalQuote, source, witness, price string) {
	t.Helper()
	require.Equal(t, source, q.source)
	require.Equal(t, witness, q.witness)
	require.True(t, q.kline.ClosePrice.Equal(decimal.RequireFromString(price)), "got %s", q.kline.ClosePrice)
}

// binance fails to fetch -> the next two sources serve and corroborate
func TestSelectHistoricalFallsThroughOnFetchError(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("no klines")),
		stubSource("kraken", covering(fallbackTS, "2000.5"), nil),
		stubSource("coinbase", covering(fallbackTS, "2001"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "kraken", "coinbase", "2000.5")
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
		stubSource("kraken", covering(fallbackTS, "1999.75"), nil),
		stubSource("coinbase", covering(fallbackTS, "2000"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "kraken", "coinbase", "1999.75")
}

// binance serves the minute and kraken corroborates it -> binance's price is
// persisted and coinbase is not consulted
// the production chain: binance first, then the three fallbacks in order
func TestHistoricalSourcesOrder(t *testing.T) {
	var names []string
	for _, s := range historicalSources {
		names = append(names, s.name)
	}
	require.Equal(t, []string{"binance", "kraken", "coinbase", "bitstamp"}, names)
}

// a minute older than kraken's ~12 hour window, with binance geo-blocked:
// coinbase and bitstamp still serve and agree, so the request succeeds
func TestSelectHistoricalPastKrakensWindow(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("HTTP 451")),
		stubSource("kraken", covering(fallbackTS.Add(13*time.Hour), "2100"), nil),
		stubSource("coinbase", covering(fallbackTS, "2000"), nil),
		stubSource("bitstamp", covering(fallbackTS, "2000.4"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "coinbase", "bitstamp", "2000")
}

func TestSelectHistoricalPrefersBinance(t *testing.T) {
	consulted := false
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000.5"), nil),
		stubSource("kraken", covering(fallbackTS, "2010"), nil),
		{name: "coinbase", fetch: func(context.Context, time.Time) ([]data.Kline, error) {
			consulted = true
			return nil, errors.New("should not be reached")
		}},
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "binance", "kraken", "2000.5")
	require.False(t, consulted, "a later source must not be consulted once two agree")
}

// a single source serving the minute is not enough (#271): the request must
// fail -- and be retried -- rather than be priced by one venue for good
func TestSelectHistoricalRequiresASecondSource(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000.5"), nil),
		stubSource("kraken", nil, errors.New("kraken down")),
		stubSource("coinbase", nil, errors.New("coinbase down")),
	}
	_, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.Error(t, err)
	require.ErrorContains(t, err, "1 served")
	require.ErrorContains(t, err, "kraken down")
	require.ErrorContains(t, err, "coinbase down")
}

// binance is 15% off: kraken disagrees with it, coinbase sides with kraken ->
// kraken's price is persisted, binance's never is
func TestSelectHistoricalOutvotesARogueFirstSource(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2300"), nil),
		stubSource("kraken", covering(fallbackTS, "2000"), nil),
		stubSource("coinbase", covering(fallbackTS, "2001"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "kraken", "coinbase", "2000")
}

// kraken is the rogue one: coinbase sides with binance -> binance's price
func TestSelectHistoricalOutvotesARogueSecondSource(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000"), nil),
		stubSource("kraken", covering(fallbackTS, "1700"), nil),
		stubSource("coinbase", covering(fallbackTS, "2001"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "binance", "coinbase", "2000")
}

// no two sources agree -> an error naming the disagreement
func TestSelectHistoricalAllSourcesDisagree(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "1000"), nil),
		stubSource("kraken", covering(fallbackTS, "2000"), nil),
		stubSource("coinbase", covering(fallbackTS, "3000"), nil),
	}
	_, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.Error(t, err)
	require.ErrorContains(t, err, "disagree")
	require.ErrorContains(t, err, "3 served")
}

// exactly maxPriceDeviation apart the sources still agree, just beyond it not
func TestSelectHistoricalAgreementBoundary(t *testing.T) {
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000"), nil),
		stubSource("kraken", covering(fallbackTS, "2200"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "binance", "kraken", "2000")

	sources[1] = stubSource("kraken", covering(fallbackTS, "2200.01"), nil)
	_, err = selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.Error(t, err)
}

// only the kline of the requested minute is returned for persistence, and the
// corroboration is on that minute: the neighbouring minutes binance returns
// alongside it play no part (#271)
func TestSelectHistoricalUsesTheRequestedMinuteOnly(t *testing.T) {
	window := func(prev, req, next string) []data.Kline {
		return []data.Kline{
			{ClosePrice: decimal.RequireFromString(prev), CloseTime: fallbackTS.Add(-time.Second)},
			{ClosePrice: decimal.RequireFromString(req), CloseTime: fallbackTS.Add(59 * time.Second)},
			{ClosePrice: decimal.RequireFromString(next), CloseTime: fallbackTS.Add(time.Minute + 59*time.Second)},
		}
	}
	sources := []historicalSource{
		stubSource("binance", window("9000", "2000", "9000"), nil),
		stubSource("kraken", window("1", "2001", "1"), nil),
	}
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
	require.NoError(t, err)
	requireQuote(t, q, "binance", "kraken", "2000")
	require.Equal(t, fallbackTS.Add(59*time.Second), q.kline.CloseTime)
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
	_, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, HistoricalChainBudget)
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
	_, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, budget)
	elapsed := time.Since(start)

	require.Error(t, err, "a chain that never serves must fail, so the request is retried")
	require.Less(t, elapsed, 10*budget, "the chain must be bounded by the budget, not the sources' own timeouts")
}

// binance serves at once but its witness hangs: the budget still bounds the
// walk and the uncorroborated price is not used -- the request is retried
func TestSelectHistoricalBudgetSpentBeforeCorroboration(t *testing.T) {
	const budget = 100 * time.Millisecond
	sources := []historicalSource{
		stubSource("binance", covering(fallbackTS, "2000.5"), nil),
		sleepySource("kraken", 3*time.Second, covering(fallbackTS, "2000.5")),
		sleepySource("coinbase", 3*time.Second, covering(fallbackTS, "2000.5")),
	}
	start := time.Now()
	_, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, budget)
	require.Error(t, err)
	require.ErrorContains(t, err, "1 served")
	require.Less(t, time.Since(start), 10*budget)
}

// a healthy walk completes well under the budget: the fallbacks are reached
// quickly and the first agreeing pair wins without waiting on any timeout
func TestSelectHistoricalHealthyWalkIsFast(t *testing.T) {
	const budget = 5 * time.Second
	sources := []historicalSource{
		stubSource("binance", nil, errors.New("gap")),
		sleepySource("kraken", time.Millisecond, covering(fallbackTS, "2000")),
		sleepySource("coinbase", time.Millisecond, covering(fallbackTS, "1999.75")),
	}

	start := time.Now()
	q, err := selectHistorical(context.Background(), sources, fallbackTS, fallbackNow, budget)
	elapsed := time.Since(start)

	require.NoError(t, err)
	requireQuote(t, q, "kraken", "coinbase", "2000")
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
