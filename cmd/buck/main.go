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
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
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
	saleParamJSON, present := os.LookupEnv("DWS_SALE_PARAMS")
	if !present {
		err := errors.New("DWS_SALE_PARAMS environment variable not set")
		log.Fatal(err)
	}
	erc20Json, present := os.LookupEnv("DWS_STABLE_COINS")
	if !present {
		err := errors.New("DWS_STABLE_COINS environment variable not set")
		log.Fatal(err)
	}

	ctxt, err := common.GetContext(erc20Json, saleParamJSON)
	if err != nil {
		log.Fatal(err)
	}
	ctxt.ReceivingAddr = strings.ToLower(daddr)
	ctxt.ETHRPCURL = url
	debugStore, present := os.LookupEnv("DWS_DEBUG_DATA_STORE")
	if present {
		ctxt.DebugDataStore = debugStore
	}
	blockCache, present := os.LookupEnv("DWS_BLOCK_CACHE")
	if present {
		ctxt.BlockCache = blockCache
	}

	dsn := common.GetDSN()
	dbh, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer dbh.Close()

	ctxt.DB = dbh
	ctxt.UpdateLastBlock = true

	log.Infof("receiving address: %v", ctxt.ReceivingAddr)
	log.Infof("ETH rpc url: %v", ctxt.ETHRPCURL)
	log.Infof("erc-20 data: %v", ctxt.StableCoins)
	log.Infof("sale params: %v", ctxt.SaleParams)

	fbn := flag.Int("set-final", -1, "set last finalized ETH block number")
	lbn := flag.Int("set-latest", -1, "set latest ETH block number")
	monitorFinal := flag.Bool("monitor-final", false, "monitor finalized ETH blocks")
	monitorLatest := flag.Bool("monitor-latest", false, "monitor latest ETH blocks")
	monitorOld := flag.Bool("monitor-old-unconfirmed", false, "check for old finalized ETH blocks with unconfirmed txs")
	port := flag.Uint("port", defaultPort, "Port for the healthcheck server")
	singleBlock := flag.Int("single-block", -1, "process the specified block number and terminate")
	flag.Parse()

	modes := map[string]bool{
		"--monitor-latest":          *monitorLatest,
		"--monitor-final":           *monitorFinal,
		"--monitor-old-unconfirmed": *monitorOld,
	}

	abi, err := ethereum.InitABI()
	if err != nil {
		return
	}
	ctxt.ABI = abi

	// latest blocks
	latestCtxt := *ctxt
	latestCtxt.CrawlerType = common.Latest

	// finalized blocks
	finalCtxt := *ctxt
	finalCtxt.CrawlerType = common.Finalized

	// old, unconfirmed blocks
	oldCtxt := *ctxt
	oldCtxt.CrawlerType = common.OldUnconfirmed

	if (*lbn > -1) && (*fbn > -1) {
		log.Error("pick either -set-latest XOR -set-final but not both")
		return
	}

	if *lbn >= 0 {
		err = db.SetLastBlock(latestCtxt, "eth", uint64(*lbn))
		if err != nil {
			log.Error(err)
			return
		}
		return
	}
	if *fbn >= 0 {
		err = db.SetLastBlock(finalCtxt, "eth", uint64(*fbn))
		if err != nil {
			log.Error(err)
			return
		}
		return
	}

	numberOfModes, err := checkFlags(modes)
	if err != nil {
		log.Error(err)
		return
	}
	// nothing to do?
	if numberOfModes == 0 {
		log.Info("nothing to do, exiting")
		return
	}

	if *singleBlock > 0 {
		if *monitorFinal {
			finalCtxt.UpdateLastBlock = false
			err = processETH(finalCtxt, uint64(*singleBlock))
		} else {
			latestCtxt.UpdateLastBlock = false
			err = processETH(latestCtxt, uint64(*singleBlock))
		}
		if err != nil {
			log.Errorf("failed to process single block %d, %v", *singleBlock, err)
			return
		}
		return
	}

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()
	g, ctx := errgroup.WithContext(context.Background())

	if *monitorLatest {
		_, err = s.Every("1m").Do(monitorETH, context.WithValue(ctx, common.BuckContext, &latestCtxt))
		if err != nil {
			log.Error(err)
			return
		}
	}

	if *monitorFinal {
		// don't clash with the healthcheck port of the other crawlers
		if *port == defaultPort {
			*port = defaultPort + 1
		}
		_, err = s.Every("1m").Do(monitorETH, context.WithValue(ctx, common.BuckContext, &finalCtxt))
		if err != nil {
			log.Error(err)
			return
		}
	}

	if *monitorOld {
		// don't clash with the healthcheck port of the other crawlers
		if *port == defaultPort {
			*port = defaultPort + 2
		}
		oldCtxt.UpdateLastBlock = false
		_, err = s.Every("15m").Do(monitorOldUnconfirmed, context.WithValue(ctx, common.BuckContext, &oldCtxt))
		if err != nil {
			log.Error(err)
			return
		}
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
			log.Info("buck - context canceled")
			return c.String(http.StatusServiceUnavailable, "{}\n")
		default:
			// all good, carry on
			err := runReadyProbe(*ctxt)
			if err != nil {
				return c.String(http.StatusServiceUnavailable, "{}\n")
			}
		}
		return c.String(http.StatusOK, "{}\n")
	})
	g.Go(func() error {
		// run live/ready probe server
		err := e.Start(fmt.Sprintf(":%d", *port))
		if !errors.Is(err, http.ErrServerClosed) {
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
				log.Info("buck/http - context canceled, stopping..")
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
				log.Info("buck/cron - context canceled, stopping..")
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
	log.Info("buck shutting down")
}

func runReadyProbe(ctxt common.Context) error {
	err := ctxt.DB.Ping()
	if err != nil {
		log.Error(err)
		return err
	}
	ctxt.MaxWaitInSeconds = 1
	_, err = ethereum.MostRecentBlockNumber(ctxt)
	if err != nil {
		log.Error(err)
	}
	return err
}

func monitorETH(ctx context.Context) error {
	ctxt, ok := ctx.Value(common.BuckContext).(*common.Context)
	if !ok {
		return errors.New("invalid buck context")
	}
	// most recent ETH block published
	mrbn, err := ethereum.MostRecentBlockNumber(*ctxt)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("===>> buck/%s tip of the ETH chain: %d", ctxt.CrawlerType, mrbn)

	// number of last block that was processed
	lpbn, err := db.GetLastBlock(ctxt.DB, "eth", ctxt.CrawlerType.String())
	if err != nil {
		return err
	}
	log.Infof("last block processed (from db): %d", lpbn)

	var startBlock uint64
	if lpbn <= 0 {
		// no valid last processed block value in the database?
		// process the current block
		startBlock = mrbn
	} else {
		startBlock = lpbn + 1
	}

	for i := startBlock; i <= mrbn; i++ {
		select {
		case <-ctx.Done():
			log.Infof("buck/%s - context canceled", ctxt.CrawlerType)
			return nil
		default:
			// keep going
		}
		err = processETH(*ctxt, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func processETH(ctxt common.Context, bn uint64) error {
	txs, err := ethereum.GetTransactions(ctxt, bn)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("block %d: %d filtered transactions", bn, len(txs))
	if len(txs) == 0 {
		err = db.SetLastBlock(ctxt, "eth", bn)
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
		log.Infof("requesting price for ETH/%s", blockTime.Format(time.RFC3339))
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

func monitorOldUnconfirmed(ctx context.Context) error {
	ctxt, ok := ctx.Value(common.BuckContext).(*common.Context)
	if !ok {
		return errors.New("buck/old-unconfirmed invalid buck context")
	}
	hashes, err := db.GetOldUnconfirmed(ctxt.DB)
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		log.Info("##### *no* unconfirmed txs older than 30 minutes")
		return nil
	}
	log.Infof("##### old unconfirmed txs: %v", hashes)

	// most recent *finalized* ETH block published
	mfbn, err := ethereum.MostRecentBlockNumber(*ctxt)
	if err != nil {
		return err
	}
	log.Infof("##### max finalized ETH block: %d", mfbn)

	txs, err := ethereum.GetTxsByHash(*ctxt, hashes)
	if err != nil {
		return err
	}
	for _, tx := range txs {
		select {
		case <-ctx.Done():
			log.Info("buck/old-unconfirmed - context canceled")
			return nil
		default:
			// keep going
		}
		if tx.BlockNumber <= mfbn {
			// finalized block hash does not match the tx block hash or
			// finalized block does not contain the tx in question
			//		=> fail tx
			failTx := tx.BlockHash != tx.FBBlockHash || !tx.FBContainsTx
			if failTx {
				log.Warnf("invalid old unconfirmed tx (%s)", tx.Hash)
				err = db.FailTx(*ctxt, tx)
				if err != nil {
					return err
				}
			} else {
				log.Infof("##### finalizing old tx %s", tx.Hash)
				err = db.FinalizeTx(*ctxt, tx)
				if err != nil {
					return err
				}
			}
		} else {
			log.Warnf("old unconfirmed tx (%s) not finalized yet", tx.Hash)
		}
	}
	return nil
}

func checkFlags(fm map[string]bool) (int, error) {
	var on []string
	for k, v := range fm {
		if v {
			on = append(on, k)
		}
	}
	switch len(on) {
	case 0:
		return 0, nil
	case 1:
		return 1, nil
	default:
		err := fmt.Errorf("please pick only *one* of these: %s", strings.Join(on, ", "))
		return len(on), err
	}
}
