package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/buck/db"
	"github.com/verity-team/dws/internal/buck/ethereum"
	"github.com/verity-team/dws/internal/common"
)

var (
	bts, rev, version string
)

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
	ctxt.ReceivingAddr = daddr
	ctxt.ETHRPCURL = strings.ToLower(url)

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

	sbn := flag.Int("sbn", 0, "start processing with this block number")
	flag.Parse()

	if *sbn > 0 {
		err = db.SetLastBlock(dbh, "eth", uint64(*sbn))
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	s := gocron.NewScheduler(time.UTC)
	_, err = s.Every("1m").Do(monitorETH, ctxt)
	if err != nil {
		log.Fatal(err)
	}
	s.StartBlocking()
}

func monitorETH(ctxt common.Context) error {
	// most recent ETH block published
	bn, err := ethereum.GetBlockNumber(ctxt.ETHRPCURL)
	if err != nil {
		log.Error(err)
		return err
	}

	// number of last block that was processed
	lbn, err := db.GetLastBlock(ctxt.DB, "eth")
	if err != nil {
		log.Error(err)
		return err
	}

	if lbn <= 0 {
		// no valid value in the database?
		// process the current block
		lbn = bn
	}

	for i := lbn + 1; i <= bn; i++ {
		txs, err := ethereum.GetTransactions(ctxt, i)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Infof("block %d: %d Transactions", i, len(txs))
		if len(txs) == 0 {
			continue
		}
		err = db.PersistTxs(ctxt, i, txs)
		if err != nil {
			log.Error(err)
			return err
		}
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
