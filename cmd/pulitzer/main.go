package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
	"github.com/verity-team/dws/internal/pulitzer/data"
	"github.com/verity-team/dws/internal/pulitzer/db"
	"golang.org/x/sync/errgroup"
)

const (
	// minPriceSources is the quorum: a median has to be taken over enough
	// inputs to be more than "the middle one of three". Four also keeps the
	// two USDT quoted sources (binance, kucoin) from making up the majority of
	// a cycle and dragging a stable coin depeg into the written price.
	minPriceSources = 4
	// minAcceptedPrices is how many mutually consistent sources have to
	// survive the outlier rejection for the cycle to produce a price.
	minAcceptedPrices = 3
)

// maxPriceDeviation is the relative deviation from the median a single source
// may show before it is dropped from the cycle.
var maxPriceDeviation = decimal.NewFromFloat(0.10)

var (
	bts, rev, version string
)

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Warn("Error loading .env file")
	}
	version = fmt.Sprintf("pulitzer::%s::%s", bts, rev)
	log.Info("version = ", version)

	// common.OpenDB proves the connection works before the first scheduled
	// job runs; the error it returns is safe to log, see common.RedactDBError
	dbh, err := common.OpenDB(common.GetDSN())
	if err != nil {
		log.Fatalf("pulitzer: %v", err)
	}
	defer func() { _ = dbh.Close() }()
	dbh.SetMaxOpenConns(10)
	dbh.SetMaxIdleConns(5)
	dbh.SetConnMaxLifetime(5 * time.Minute)

	port := flag.Uint("port", 8081, "Port for the healthcheck server")
	flag.Parse()

	// listen for interrupt signals (like Ctrl-C).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create an errorgroup.
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		select {
		case <-sigChan:
			log.Info("pulitzer: received an interrupt, canceling...")
			cancel()
		case <-ctx.Done():
			log.Info("pulitzer: interrupt handler, canceling...")
			// If context is done, return the context error.
			return ctx.Err()
		}
		return nil
	})

	ctx = context.WithValue(ctx, common.DBHandle, dbh)

	// without a global panic handler gocron does not recover from panics in
	// scheduled jobs i.e. a single panic would terminate the process
	gocron.SetPanicHandler(func(jobName string, recoverData interface{}) {
		log.Errorf("pulitzer: job '%s' panicked: %v", jobName, recoverData)
	})

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()

	_, err = s.Every("1m").Do(getETHPrice, ctx)
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.Every("10s").Do(servePriceRequests, ctx)
	if err != nil {
		log.Fatal(err)
	}

	// gocron discards the error a scheduled job returns unless an error event
	// listener is registered; it only covers the jobs scheduled so far, hence
	// this call comes after all of the s.Every(..).Do(..) calls above
	common.RegisterJobErrorListener(s, "pulitzer")

	// healthcheck endpoints
	e := echo.New()
	e.GET("/live", func(c echo.Context) error {
		return c.String(http.StatusOK, "{}\n")
	})
	e.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"version": version})
	})
	e.GET("/ready", func(c echo.Context) error {
		select {
		case <-ctx.Done():
			log.Info("pulitzer - context canceled")
			return c.String(http.StatusServiceUnavailable, "{}\n")
		default:
			// all good, carry on
			err := runReadyProbe(dbh)
			if err != nil {
				return c.String(http.StatusServiceUnavailable, "{}\n")
			}
		}
		return c.String(http.StatusOK, "{}\n")
	})

	g.Go(func() error {
		// run live/ready probe server
		err := e.Start(fmt.Sprintf(":%d", *port))
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		log.Error(err)
		return err
	})

	g.Go(func() error {
		// shut down live/ready probe server if needed
		<-ctx.Done()
		// The context is canceled
		log.Info("pulitzer/http - context canceled, stopping..")
		sdc, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := e.Shutdown(sdc); err != nil {
			log.Error(err)
		}
		return ctx.Err()
	})

	g.Go(func() error {
		// start cron jobs
		s.StartAsync()

		// shut down cron jobs if needed
		<-ctx.Done()
		// The context is canceled
		log.Info("pulitzer/cron - context canceled, stopping..")
		// StopBlockingChan() is a no-op for a scheduler started with
		// StartAsync(); Stop() waits for the jobs that are still running so
		// that the database handle is not closed underneath them
		s.Stop()
		return ctx.Err()
	})

	// a context cancelation is how an orderly shutdown ends, anything else is
	// a failure the orchestrator needs to see (crash loop backoff, alerting)
	failed := false
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("errgroup.Wait(): %v", err)
		failed = true
	}
	log.Info("pulitzer shutting down")
	if failed {
		os.Exit(1)
	}
}

// sourcePrice is an ethereum price together with the venue it came from. The
// source name makes a rejected outlier identifiable in the log and gives the
// sort below a stable tie breaker.
type sourcePrice struct {
	source string
	price  decimal.Decimal
}

// medianPrice returns the median of a slice that is sorted in ascending order.
// For an even number of entries it is the mean of the two middle values.
func medianPrice(prices []sourcePrice) decimal.Decimal {
	n := len(prices)
	mid := n / 2
	if n%2 == 1 {
		return prices[mid].price
	}
	return prices[mid-1].price.Add(prices[mid].price).Div(decimal.NewFromInt(2))
}

// sortPrices returns a copy of prices ordered by price and, for equal prices,
// by source name. Both the median and the log output are then a function of
// the prices alone and no longer of the (randomized) order in which the
// goroutines happened to be collected.
func sortPrices(prices []sourcePrice) []sourcePrice {
	sorted := make([]sourcePrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		if c := sorted[i].price.Cmp(sorted[j].price); c != 0 {
			return c < 0
		}
		return sorted[i].source < sorted[j].source
	})
	return sorted
}

// rejectOutliers drops the sources that deviate from the median by more than
// maxPriceDeviation and returns the ones that remain.
//
// Every source is compared against a single reference value -- the median --
// rather than against every other source pairwise, which makes the outcome
// symmetric and independent of the order the prices arrived in. A venue that
// is off on its own is dropped; the remaining, mutually consistent sources
// still produce a price for this cycle.
func rejectOutliers(sorted []sourcePrice) []sourcePrice {
	median := medianPrice(sorted)
	if !median.IsPositive() {
		return nil
	}
	accepted := make([]sourcePrice, 0, len(sorted))
	for _, sp := range sorted {
		deviation := sp.price.Sub(median).Div(median).Abs()
		if deviation.GreaterThan(maxPriceDeviation) {
			log.Warnf(
				"rejecting the ethereum price from %s ($%s): %s%% off the median ($%s)",
				sp.source, sp.price.StringFixed(2),
				deviation.Mul(decimal.NewFromInt(100)).StringFixed(2), median.StringFixed(2))
			continue
		}
		accepted = append(accepted, sp)
	}
	return accepted
}

// calculateAveragePrice turns the prices collected in one cycle into the
// figure that gets written to the database: the mean of the sources that are
// within maxPriceDeviation of the median.
func calculateAveragePrice(prices []sourcePrice) (decimal.Decimal, error) {
	usable := make([]sourcePrice, 0, len(prices))
	for _, sp := range prices {
		if !sp.price.IsPositive() {
			log.Warnf("ignoring the non-positive ethereum price from %s ($%s)", sp.source, sp.price.StringFixed(2))
			continue
		}
		usable = append(usable, sp)
	}
	if len(usable) < minPriceSources {
		return decimal.Zero, fmt.Errorf("got %d usable price(s), need at least %d", len(usable), minPriceSources)
	}

	sorted := sortPrices(usable)
	accepted := rejectOutliers(sorted)
	if len(accepted) < minAcceptedPrices {
		return decimal.Zero, fmt.Errorf(
			"only %d of %d price(s) are within %s%% of the median, need at least %d",
			len(accepted), len(sorted),
			maxPriceDeviation.Mul(decimal.NewFromInt(100)).StringFixed(0), minAcceptedPrices)
	}

	sum := decimal.Zero
	for _, sp := range accepted {
		sum = sum.Add(sp.price)
	}
	return sum.Div(decimal.NewFromInt(int64(len(accepted)))), nil
}

func servePriceRequests(ctx context.Context) error {
	select {
	case <-ctx.Done():
		log.Info("pulitzer/historic - context canceled")
		return nil
	default:
		// keep going
	}
	dbh, ok := ctx.Value(common.DBHandle).(*sqlx.DB)
	if !ok {
		return errors.New("pulitzer/historic invalid database handle")
	}
	rqs, err := db.GetOpenPriceRequests(dbh)
	if err != nil {
		return err
	}
	for _, rq := range rqs {
		// now is captured *before* the fetch and threaded into CloseRequest so
		// a kline served in the last fraction of its minute is judged against
		// the clock as it stood then, not after the round trip (see #227)
		now := time.Now().UTC()
		// a single request that cannot be fulfilled must not block the
		// remaining (older or newer) requests in the queue
		klines, source, err := historicalKlines(ctx, rq.Time, now)
		if err != nil {
			err = fmt.Errorf("failed to obtain historical prices for %s, %w", rq.Time, err)
			log.Error(err)
			failPriceRequest(dbh, rq, now)
			continue
		}
		err = db.CloseRequest(dbh, rq.ID, rq.Time, klines, now)
		if err != nil {
			err = fmt.Errorf("failed to persist historical prices for request #%d/%s, %w", rq.ID, rq.Time, err)
			log.Error(err)
			failPriceRequest(dbh, rq, now)
			continue
		}
		log.Infof("obtained %d historical price(s) for request #%d/%s from %s", len(klines), rq.ID, rq.Time, source)
	}
	return nil
}

// HistoricalChainBudget bounds the wall-clock time selectHistorical may spend
// walking the whole fallback chain for one request.
//
// The chain fetches from binance, kraken and coinbase in sequence, each with
// the default 10s common.HTTPGet client timeout, so three hung sources would
// otherwise take ~30s -- and servePriceRequests is scheduled every 10s and
// cannot overlap itself under SingletonModeAll, so a slow run delays every
// later request. A single 15s deadline over the whole chain caps that: it is
// generous for a healthy walk (three reachable venues answer in well under a
// second combined) yet still lets one genuinely slow-but-working source run out
// its own 10s client timeout once, while guaranteeing the chain returns in
// ~15s rather than ~30s. The deadline is threaded into each fetch (see
// HTTPGetCtx), so a request already in flight when the budget is spent is
// actually cancelled, not merely prevented from starting the next source.
const HistoricalChainBudget = 15 * time.Second

// historicalSource pairs a venue name with its historical kline fetcher.
type historicalSource struct {
	name  string
	fetch func(context.Context, time.Time) ([]data.Kline, error)
}

// historicalSources is the ordered fallback chain the backfill path walks to
// price a minute. binance is tried first, unchanged; kraken and coinbase back
// it up so a binance-specific data gap or outage no longer stalls donation
// finalization the way a single hardcoded source did (see #228). The live
// price path already fans out to six venues -- this gives the backfill, the
// path that has to unblock a stalled crawler, source diversity of its own.
var historicalSources = []historicalSource{
	{"binance", data.GetHistoricalPriceFromBinance},
	{"kraken", data.GetHistoricalPriceFromKraken},
	{"coinbase", data.GetHistoricalPriceFromCoinbase},
}

// historicalKlines returns the klines from the first configured source that
// can actually serve the requested minute, together with the name of that
// source. It is the production entry point; selectHistorical holds the logic so
// it can be exercised with stub sources and a shorter budget.
func historicalKlines(ctx context.Context, ts, now time.Time) ([]data.Kline, string, error) {
	return selectHistorical(ctx, historicalSources, ts, now, HistoricalChainBudget)
}

// selectHistorical walks the sources in order and returns the klines from the
// first one that can serve the requested minute.
//
// The whole walk shares a single budget: ctx is given a deadline of budget and
// each source's fetch is bound to it, so one hung source cannot make the chain
// run past ~budget and a fetch still in flight when the budget is spent is
// cancelled rather than left to its own 10s client timeout. ctx is the caller's
// (the gocron job context), so a shutdown also cancels an in-flight fetch.
//
// A source is skipped -- in favour of the next -- when it errors (a cancelled
// fetch is just another error) or when what it returns does not cover the
// minute with a final kline (db.ServesMinute, the same gate CloseRequest
// applies). Only when every source has been tried and none can serve the minute
// is an error returned, carrying the reason each one gave so a genuine gap --
// or a spent budget -- is diagnosable; the caller then fails the request and it
// is retried.
func selectHistorical(ctx context.Context, sources []historicalSource, ts, now time.Time, budget time.Duration) ([]data.Kline, string, error) {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	var errs []error
	for _, s := range sources {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("chain budget (%v) spent before %s: %w", budget, s.name, err))
			break
		}
		klines, err := s.fetch(ctx, ts)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
			continue
		}
		if !db.ServesMinute(ts, now, klines) {
			errs = append(errs, fmt.Errorf(
				"%s: no final kline within %v of the requested minute", s.name, common.PriceLookupWindow))
			continue
		}
		return klines, s.name, nil
	}
	return nil, "", fmt.Errorf(
		"no historical source could serve %s, %w", ts.UTC().Format(time.RFC3339), errors.Join(errs...))
}

// failPriceRequest records a price request that could not be served so that the
// failure is visible; the request is retried later instead of silently
// remaining at 'new' and being picked up on every single cycle.
//
// The request is never abandoned -- FailRequest keeps it eligible for retry
// forever, so the stall self-heals when the underlying fault clears. But a
// request that has stayed unfulfilled past db.PriceReqAlertAge is additionally
// logged at error level on every failed attempt, so that a fault which no
// source can work around (a pulitzer-side egress/DNS/CA/NetworkPolicy problem
// hitting all venues at once, or a genuinely missing minute) is alertable and
// gets a human, rather than only ever showing up as a slowly falling behind
// crawler.
func failPriceRequest(dbh *sqlx.DB, rq db.PriceReq, now time.Time) {
	if err := db.FailRequest(dbh, rq.ID); err != nil {
		log.Errorf("failed to record the failure of price request #%d, %v", rq.ID, err)
	}
	if msg, due := stuckPriceRequestAlert(rq, now); due {
		log.Error(msg)
	}
}

// stuckPriceRequestAlert returns an alert line, and whether one is due, for a
// request that could not be served this cycle. A request open longer than
// db.PriceReqAlertAge has resisted every attempt so far; it keeps being retried
// (it never becomes terminal) but is surfaced at error level so it is visible
// and alertable. The age is measured from created_at, so no schema column is
// needed to track it.
func stuckPriceRequestAlert(rq db.PriceReq, now time.Time) (string, bool) {
	age := now.UTC().Sub(rq.CreatedAt.UTC())
	if age < db.PriceReqAlertAge {
		return "", false
	}
	return fmt.Sprintf(
		"price request #%d for %s has been unfulfilled for %v (>= %v): no price source can serve the minute -- needs attention (still retrying every %v)",
		rq.ID, rq.Time.UTC().Format(time.RFC3339), age.Truncate(time.Second), db.PriceReqAlertAge, db.FailedRetryDelay), true
}

func getETHPrice(ctx context.Context) (decimal.Decimal, error) {
	select {
	case <-ctx.Done():
		log.Info("pulitzer/latest - context canceled")
		return decimal.Zero, nil
	default:
		// keep going
	}
	dbh, ok := ctx.Value(common.DBHandle).(*sqlx.DB)
	if !ok {
		return decimal.Zero, errors.New("pulitzer/latest invalid database handle")
	}
	// Create a wait group to synchronize the goroutines
	var wg sync.WaitGroup

	// ethereum price sources and the functions to call to get the price
	sources := map[string]func() (decimal.Decimal, error){
		"binance":  data.GetBinanceETHPrice,
		"kraken":   data.GetKrakenETHPrice,
		"bitfinex": data.GetBitfinexETHPrice,
		"coinbase": data.GetCoinbaseETHPrice,
		"cexio":    data.GetCexIOETHUSDLastPrice,
		"kucoin":   data.GetKuCoinETHUSDTPrice,
	}

	// channels the go routines will use to send back the price
	channels := make(map[string]chan decimal.Decimal, len(sources))

	// start one go routine per price source
	for k := range sources {
		channels[k] = make(chan decimal.Decimal, 1)

		// start the go routine to fetch the ethereum price from the source
		wg.Add(1)
		go func(source string, f func() (decimal.Decimal, error), ch chan decimal.Decimal) {
			defer wg.Done()
			fetchPrice(source, f, ch)
		}(k, sources[k], channels[k])
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Receive and print the results from the channels. The sources are walked
	// in a fixed order so that the log of a cycle can be reproduced.
	names := make([]string, 0, len(channels))
	for k := range channels {
		names = append(names, k)
	}
	sort.Strings(names)

	prices := make([]sourcePrice, 0, len(names))
	log.Infof("==> %v", time.Now().UTC())
	for _, k := range names {
		price, ok := <-channels[k]
		if ok {
			log.Infof("ethereum price from %s: $%s", k, price.StringFixed(2))
			prices = append(prices, sourcePrice{source: k, price: price})
		} else {
			err := fmt.Errorf("failed to get ethereum price from %s", k)
			log.Error(err)
		}
	}

	av, err := calculateAveragePrice(prices)
	if err != nil {
		log.Error(err)
		return decimal.Zero, err
	}
	log.Infof("average price: $%s", av.StringFixed(2))

	err = db.PersistETHPrice(dbh, av)
	if err != nil {
		log.Errorf("failed to persist ETH price, %v", err)
		return decimal.Zero, err
	}

	return av, nil
}

// fetchPrice fetches the ethereum price from a single source and sends it to
// ch.
//
// A panic in a price source has to be recovered here: it happens in the fan-out
// go routine and gocron's panic handler only covers the goroutine the scheduled
// job itself runs in. A source that panics -- just like one that returns an
// error -- drops out of this cycle: ch is closed without a value and the
// receiving side treats it as a failed source.
func fetchPrice(source string, f func() (decimal.Decimal, error), ch chan<- decimal.Decimal) {
	defer close(ch)
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic while fetching ethereum price from %s, %v", source, r)
		}
	}()

	price, err := f()
	if err != nil {
		log.Errorf("error fetching ethereum price from %s, %v", source, err)
		return
	}
	ch <- price
}

// runReadyProbe reports whether this instance can do its work. That depends on
// the database -- the only resource pulitzer cannot do without. It
// deliberately does *not* call out to an exchange: a third party being slow,
// or answering a probe with a rate limit error, would take an otherwise
// healthy pod out of service (and burn the very request budget the price fetch
// needs), while a single exchange being unreachable costs one of six sources.
func runReadyProbe(dbh *sqlx.DB) error {
	if err := dbh.Ping(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
