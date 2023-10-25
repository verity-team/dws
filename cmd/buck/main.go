package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/buck/db"
	"github.com/verity-team/dws/internal/buck/ethereum"
	"github.com/verity-team/dws/internal/common"
)

var (
	bts, rev, version string
)

const defaultPort = 8082

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Warn("Error loading .env file")
	}
	version = fmt.Sprintf("buck::%s::%s", bts, rev)
	log.Info("version = ", version)

	// make sure these environment variables are set
	daddr, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		log.Fatal("DWS_DONATION_ADDRESS variable not set")
	}
	url, present := os.LookupEnv("ETH_RPC_URL")
	if !present {
		log.Fatal("ETH_RPC_URL variable not set")
	}
	saleParamJson, present := os.LookupEnv("DWS_SALE_PARAMS")
	if !present {
		err := errors.New("DWS_SALE_PARAMS environment variable not set")
		log.Fatal(err)
	}
	erc20Json, present := os.LookupEnv("DWS_STABLE_COINS")
	if !present {
		err := errors.New("DWS_STABLE_COINS environment variable not set")
		log.Fatal(err)
	}

	ctxt, err := common.GetContext(erc20Json, saleParamJson)
	if err != nil {
		log.Fatal(err)
	}
	ctxt.ReceivingAddr = strings.ToLower(daddr)
	ctxt.ETHRPCURL = url
	blockStorage, present := os.LookupEnv("DWS_BLOCK_STORAGE")
	if present {
		ctxt.BlockStorage = blockStorage
	}

	dsn := getDSN()
	dbh, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer dbh.Close()

	ctxt.DB = dbh

	log.Infof("receiving address: %v", ctxt.ReceivingAddr)
	log.Infof("ETH rpc url: %v", ctxt.ETHRPCURL)
	log.Infof("erc-20 data: %v", ctxt.StableCoins)
	log.Infof("sale params: %v", ctxt.SaleParams)

	fbn := flag.Int("set-final", -1, "set last finalized ETH block number")
	lbn := flag.Int("set-latest", -1, "set latest ETH block number")
	monitorFinal := flag.Bool("monitor-final", false, "monitor finalized ETH blocks")
	monitorLatest := flag.Bool("monitor-latest", false, "monitor latest ETH blocks")
	port := flag.Uint("port", defaultPort, "Port for the healthcheck server")
	singleBlock := flag.Int("single-block", -1, "process the specified block number and terminate")
	flag.Parse()

	if *monitorLatest && *monitorFinal {
		log.Fatal("pick either -monitor-latest XOR -monitor-final but not both")
	}
	if (*lbn > -1) && (*fbn > -1) {
		log.Fatal("pick either -set-latest XOR -set-final but not both")
	}

	if *lbn >= 0 {
		err = db.SetLastBlock(dbh, "eth", db.Latest, uint64(*lbn))
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if *fbn >= 0 {
		err = db.SetLastBlock(dbh, "eth", db.Finalized, uint64(*fbn))
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	// nothing to do?
	if !*monitorLatest && !*monitorFinal {
		os.Exit(0)
	}

	if *singleBlock > 0 {
		if *monitorFinal {
			err = processFinalized(*ctxt, uint64(*singleBlock))
		} else {
			err = processLatest(*ctxt, uint64(*singleBlock))
		}
		if err != nil {
			err = fmt.Errorf("failed to process single block %d, %v", *singleBlock, err)
			log.Error(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()

	if *monitorLatest {
		_, err = s.Every("1m").Do(monitorLatestETH, *ctxt)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *monitorFinal {
		_, err = s.Every("1m").Do(monitorFinalizedETH, *ctxt)
		if err != nil {
			log.Fatal(err)
		}
	}

	// don't clash with the healthcheck port of the ETH/latest crawler
	if *monitorFinal && (*port == defaultPort) {
		*port = defaultPort + 1
	}

	// healthcheck endpoints
	e := echo.New()
	e.GET("/live", func(c echo.Context) error {
		return c.String(http.StatusOK, "{}\n")
	})
	e.GET("/ready", func(c echo.Context) error {
		err := runReadyProbe(*ctxt, *monitorLatest)
		if err != nil {
			return c.String(http.StatusServiceUnavailable, "{}\n")
		}
		return c.String(http.StatusOK, "{}\n")
	})
	go func() {
		err := e.Start(fmt.Sprintf(":%d", *port))
		if err != http.ErrServerClosed {

			log.Fatal(err)
		}
	}()

	s.StartBlocking()
}

func runReadyProbe(ctxt common.Context, latest bool) error {
	err := ctxt.DB.Ping()
	if err != nil {
		log.Error(err)
		return err
	}
	if latest {
		_, err = ethereum.GetBlockNumber(ctxt.ETHRPCURL)
	} else {
		_, err = ethereum.GetMaxFinalizedBlockNumber(ctxt.ETHRPCURL)
	}
	if err != nil {
		log.Error(err)
	}
	return err
}

func monitorFinalizedETH(ctxt common.Context) error {
	// most recent *finalized* ETH block published
	fbn, err := ethereum.GetMaxFinalizedBlockNumber(ctxt.ETHRPCURL)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("##### latest finalized ETH block: %d", fbn)

	// number of last finalized block that was processed
	lfdb, err := db.GetLastBlock(ctxt.DB, "eth", db.Finalized)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("latest finalized block processed (from db): %d", lfdb)

	var startBlock uint64
	if lfdb <= 0 {
		// no valid latest finalized block value in the database?
		// get the block number of the oldest unconfirmed transaction
		startBlock, err = db.GetOldestUnconfirmed(ctxt.DB)
		if err != nil {
			return err
		}
		log.Infof("oldest unconfirmed tx block (from db): %d", startBlock)
		if startBlock == 0 {
			// no unconfirmed transactions -- nothing to do
			return nil
		}
	} else {
		startBlock = lfdb + 1
	}

	for i := startBlock; i <= fbn; i++ {
		err = processFinalized(ctxt, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func processFinalized(ctxt common.Context, bn uint64) error {
	fb, err := ethereum.GetFinalizedBlock(ctxt, bn)
	if err != nil {
		return err
	}
	log.Infof("finalized block %d: %d transaction hashes", bn, len(fb.Transactions))
	if len(fb.Transactions) == 0 {
		err = db.SetLastBlock(ctxt.DB, "eth", db.Finalized, bn)
		if err != nil {
			return err
		}
		return nil
	}

	err = db.FinalizeTxs(ctxt, *fb)
	if err != nil {
		log.Errorf("failed to confirm transactions for finalized block #%d", bn)
		return err
	}
	return nil
}

func monitorLatestETH(ctxt common.Context) error {
	// most recent ETH block published
	lbn, err := ethereum.GetBlockNumber(ctxt.ETHRPCURL)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("===>> tip of the ETH chain: %d", lbn)

	// number of last block that was processed
	latest, err := db.GetLastBlock(ctxt.DB, "eth", db.Latest)
	if err != nil {
		return err
	}
	log.Infof("latest block processed (from db): %d", latest)

	var startBlock uint64
	if latest <= 0 {
		// no valid latest block value in the database?
		// process the current block
		startBlock = lbn
	} else {
		startBlock = latest + 1
	}

	for i := startBlock; i <= lbn; i++ {
		err = processLatest(ctxt, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func processLatest(ctxt common.Context, bn uint64) error {
	txs, err := ethereum.GetTransactions(ctxt, bn)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("block %d: %d filtered transactions", bn, len(txs))
	if len(txs) == 0 {
		err = db.SetLastBlock(ctxt.DB, "eth", db.Latest, bn)
		if err != nil {
			return err
		}
		return nil
	}

	// we only get the ETH price if we need to persist transactions
	// get ETH price at the time the block was published
	blockTime := txs[0].BlockTime
	ethPrice, err := common.GetETHPrice(ctxt.DB, blockTime)
	if err != nil {
		// request the missing price and let's hope it is avaiable next time we
		// need it
		log.Infof("requesting price for ETH/%s", blockTime)
		err2 := db.RequestPrice(ctxt, "eth", blockTime)
		if err2 != nil {
			log.Errorf("failed to request price for ETH/%s, %v", blockTime, err2)
		}
		return err
	}
	log.Infof("eth price: %s", ethPrice)
	err = db.PersistTxs(ctxt, bn, ethPrice, txs)
	if err != nil {
		log.Error(err)
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
