package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/pulitzer"
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

	dsn := getDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()

	_, err = s.Every("1m").Do(getETHPrice, db)
	if err != nil {
		log.Fatal(err)
	}
	s.StartBlocking()
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

func getETHPrice(db *sqlx.DB) (decimal.Decimal, error) {
	// Create a wait group to synchronize the goroutines
	var wg sync.WaitGroup

	// ethereum price sources and the functions to call to get the price
	sources := map[string]func() (decimal.Decimal, error){
		"binance":  pulitzer.GetWeightedAvgPriceFromBinance,
		"kraken":   pulitzer.GetKrakenETHPrice,
		"bitfinex": pulitzer.GetBitfinexETHPrice,
		"coinbase": pulitzer.GetCoinbaseETHPrice,
		"cexio":    pulitzer.GetCexIOETHUSDLastPrice,
		"kucoin":   pulitzer.GetKuCoinETHUSDTPrice,
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
	} else {
		log.Infof("average price: $%s", av.StringFixed(2))
	}

	err = persistETHPrice(db, av)
	if err != nil {
		log.Errorf("failed to persist ETH price, %v", err)
		return decimal.Zero, err
	}

	return av, nil
}

func persistETHPrice(db *sqlx.DB, avp decimal.Decimal) error {
	q := `
		INSERT INTO price(asset, price) VALUES(:asset, :price)
		`
	qd := map[string]interface{}{
		"asset": "eth",
		"price": avp,
	}
	_, err := db.NamedExec(q, qd)
	if err != nil {
		log.Errorf("Failed to insert ETH price, %v", err)
		return err
	}
	return nil
}

func getDSN() string {
	var (
		host, port, user, passwd, database string
		present                            bool
	)

	host, present = os.LookupEnv("DWS_DB_HOST")
	if !present {
		log.Fatal("DWS_DB_HOST variable not set")
	}
	port, present = os.LookupEnv("DWS_DB_PORT")
	if !present {
		log.Fatal("DWS_DB_PORT variable not set")
	}
	user, present = os.LookupEnv("DWS_DB_USER")
	if !present {
		log.Fatal("DWS_DB_USER variable not set")
	}
	passwd, present = os.LookupEnv("DWS_DB_PASSWORD")
	if !present {
		log.Fatal("DWS_DB_PASSWORD variable not set")
	}
	database, present = os.LookupEnv("DWS_DB_DATABASE")
	if !present {
		log.Fatal("DWS_DB_DATABASE variable not set")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, passwd, database)

	log.Infof("host: '%s'", host)
	log.Infof("database: '%s'", database)
	return dsn
}
