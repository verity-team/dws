package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

	dsn := common.GetDSN()
	dbh, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
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

func calculateAveragePrice(prices []decimal.Decimal) (decimal.Decimal, error) {
	if len(prices) < 3 {
		return decimal.Zero, errors.New("less than 3 prices provided")
	}

	err := checkPriceDeviation(prices)
	if err != nil {
		return decimal.Zero, err
	}

	// Calculate the sum of prices
	sum := decimal.Zero
	for _, price := range prices {
		sum = sum.Add(price)
	}

	// Calculate the average by dividing the sum by the number of prices
	average := sum.Div(decimal.NewFromInt(int64(len(prices))))

	return average, nil
}

func checkPriceDeviation(prices []decimal.Decimal) error {
	if len(prices) < 3 {
		return errors.New("at least 3 prices are required for deviation check")
	}

	// Check deviation between all pairs of prices
	for i := 0; i < len(prices); i++ {
		for j := i + 1; j < len(prices); j++ {
			price1 := prices[i]
			price2 := prices[j]

			if price1.IsZero() || price2.IsZero() {
				return fmt.Errorf("zero price encountered: %s and %s", price1.StringFixed(2), price2.StringFixed(2))
			}

			// Calculate the percentage deviation
			deviation := price1.Sub(price2).Div(price1).Abs()

			if deviation.GreaterThan(decimal.NewFromFloat(0.10)) {
				return fmt.Errorf("price deviation between %s and %s is greater than 10%%", price1.StringFixed(2), price2.StringFixed(2))
			}
		}
	}

	return nil
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
		// a single request that cannot be fulfilled must not block the
		// remaining (older or newer) requests in the queue
		klines, err := data.GetHistoricalPriceFromBinance(rq.Time)
		if err != nil {
			err = fmt.Errorf("failed to obtain historical prices for %s, %w", rq.Time, err)
			log.Error(err)
			continue
		}
		err = db.CloseRequest(dbh, rq.ID, klines)
		if err != nil {
			err = fmt.Errorf("failed to persist historical prices for request #%d/%s, %w", rq.ID, rq.Time, err)
			log.Error(err)
			continue
		}
		log.Infof("obtained %d historical price(s) for request #%d/%s", len(klines), rq.ID, rq.Time)
	}
	return nil
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
		"binance":  data.GetWeightedAvgPriceFromBinance,
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

	// Receive and print the results from the channels
	var prices []decimal.Decimal

	log.Infof("==> %v", time.Now().UTC())
	for k := range channels {
		price, ok := <-channels[k]
		if ok {
			log.Infof("ethereum price from %s: $%s", k, price.StringFixed(2))
			if price.IsPositive() {
				prices = append(prices, price)
			}
		} else {
			err := fmt.Errorf("failed to get ethereum price from %s", k)
			log.Error(err)
		}
	}

	if len(prices) < 3 {
		err := errors.New("got less than 3 prices, giving up")
		log.Error(err)
		return decimal.Zero, err
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

func runReadyProbe(dbh *sqlx.DB) error {
	err := dbh.Ping()
	if err != nil {
		log.Error(err)
		return err
	}
	_, err = data.PingBinance()
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}
