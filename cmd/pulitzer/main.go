package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sync"
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
	defer dbh.Close()

	port := flag.Uint("port", 8081, "Port for the healthcheck server")
	flag.Parse()

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()
	g, ctx := errgroup.WithContext(context.Background())

	_, err = s.Every("1m").Do(getETHPrice, dbh, ctx)
	if err != nil {
		log.Error(err)
		return
	}
	_, err = s.Every("10s").Do(servePriceRequests, dbh, ctx)
	if err != nil {
		log.Error(err)
		return
	}

	// healthcheck endpoints
	e := echo.New()
	e.GET("/live", func(c echo.Context) error {
		return c.String(http.StatusOK, "{}\n")
	})
	e.GET("/version", func(c echo.Context) error {
		return c.String(http.StatusOK, fmt.Sprintf(`{"version": "%s"}`, version))
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
		if err != http.ErrServerClosed {
			log.Error(err)
		}
		return err
	})

	g.Go(func() error {
		// shut down live/ready probe server if needed
		for {
			select {
			case <-ctx.Done():
				// The context is canceled
				log.Info("pulitzer/http - context canceled, stopping..")
				sdc, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				if err := e.Shutdown(sdc); err != nil {
					log.Error(err)
				}
				return ctx.Err()
			case <-time.After(5 * time.Second):
				// all good -- carry on
			}
		}
	})

	g.Go(func() error {
		// start cron jobs
		s.StartAsync()

		// shut down cron jobs if needed
		for {
			select {
			case <-ctx.Done():
				// The context is canceled
				log.Info("pulitzer/cron - context canceled, stopping..")
				s.StopBlockingChan()
				return ctx.Err()
			case <-time.After(5 * time.Second):
				// all good -- carry on
			}
		}
	})

	if err := g.Wait(); err != nil {
		log.Errorf("errgroup.Wait(): %v", err)
	}
	log.Info("pulitzer shutting down")
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

			// Calculate the percentage deviation
			deviation := price1.Sub(price2).Div(price1).Abs()

			if deviation.GreaterThan(decimal.NewFromFloat(0.10)) {
				return fmt.Errorf("price deviation between %s and %s is greater than 10%%", price1.StringFixed(2), price2.StringFixed(2))
			}
		}
	}

	return nil
}

func servePriceRequests(dbh *sqlx.DB, ctx context.Context) error {
	select {
	case <-ctx.Done():
		log.Info("pulitzer/historic - context canceled")
		return nil
	default:
		// keep going
	}
	rqs, err := db.GetOpenPriceRequests(dbh)
	if err != nil {
		return err
	}
	for _, rq := range rqs {
		klines, err := data.GetHistoricalPriceFromBinance(rq.Time)
		if err != nil {
			err = fmt.Errorf("failed to obtain historical prices for %s, %w", rq.Time, err)
			log.Error(err)
			return err
		}
		err = db.CloseRequest(dbh, rq.ID, klines)
		if err != nil {
			err = fmt.Errorf("failed to persist historical prices for request #%d/%s, %w", rq.ID, rq.Time, err)
			log.Error(err)
			return err
		}
		log.Infof("obtained historical prices for request #%d/%s", rq.ID, rq.Time)
	}
	return nil
}

func getETHPrice(dbh *sqlx.DB, ctx context.Context) (decimal.Decimal, error) {
	select {
	case <-ctx.Done():
		log.Info("pulitzer/latest - context canceled")
		return decimal.Zero, nil
	default:
		// keep going
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
			defer close(ch)

			price, err := f()
			if err != nil {
				log.Errorf("error fetching ethereum price from %s, %v", source, err)
				return
			}
			ch <- price
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
