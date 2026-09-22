package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/buck/db"
	eth "github.com/verity-team/dws/internal/buck/ethereum"
	c "github.com/verity-team/dws/internal/common"
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
	if err := c.ValidateRPCURL(url); err != nil {
		log.Fatal(err)
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

	daddr = strings.ToLower(strings.TrimSpace(daddr))
	if !c.IsValidETHAddress(daddr) {
		log.Fatal("DWS_DONATION_ADDRESS is not a valid ethereum address")
	}

	// the erc-20 ABI map doubles as the set of assets buck is able to process;
	// it is needed to validate the erc-20 configuration below
	abi, err := eth.InitABI()
	if err != nil {
		log.Fatalf("failed to initialize the erc-20 ABI, %v", err)
	}

	ctxt, err := c.GetContext(erc20Json, saleParamJSON, abi)
	if err != nil {
		log.Fatal(err)
	}
	ctxt.ReceivingAddr = daddr
	ctxt.ETHRPCURL = url
	debugStore, present := os.LookupEnv("DWS_DEBUG_DATA_STORE")
	if present {
		ctxt.DebugDataStore = debugStore
	}
	blockCache, present := os.LookupEnv("DWS_BLOCK_CACHE")
	if present {
		ctxt.BlockCache = blockCache
	}

	// c.OpenDB proves the connection works before the crawler starts; the
	// error it returns is safe to log, see c.RedactDBError. The bare
	// log.Fatal(err) this replaces printed whatever lib/pq handed back, which
	// for a malformed connection string is a quoted fragment of it.
	dbh, err := c.OpenDB(c.GetDSN())
	if err != nil {
		log.Fatalf("buck: %v", err)
	}
	defer func() { _ = dbh.Close() }()
	dbh.SetMaxOpenConns(10)
	dbh.SetMaxIdleConns(5)
	dbh.SetConnMaxLifetime(5 * time.Minute)

	ctxt.DB = dbh
	ctxt.UpdateLastBlock = true

	log.Infof("receiving address: %v", ctxt.ReceivingAddr)
	// the jsonrpc API provider credentials live in the part of ETH_RPC_URL
	// that RedactURL withholds
	log.Infof("ETH rpc url: %v", c.RedactURL(ctxt.ETHRPCURL))
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

	// latest blocks
	latestCtxt := *ctxt
	latestCtxt.CrawlerType = c.Latest

	// finalized blocks
	finalCtxt := *ctxt
	finalCtxt.CrawlerType = c.Finalized

	// old, unconfirmed blocks
	oldCtxt := *ctxt
	oldCtxt.CrawlerType = c.OldUnconfirmed

	if (*lbn > -1) && (*fbn > -1) {
		log.Fatal("pick either -set-latest XOR -set-final but not both")
	}

	if *lbn >= 0 {
		err = db.SetLastBlock(latestCtxt, "eth", uint64(*lbn))
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if *fbn >= 0 {
		err = db.SetLastBlock(finalCtxt, "eth", uint64(*fbn))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	numberOfModes, err := checkFlags(modes)
	if err != nil {
		log.Fatal(err)
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
			log.Fatalf("failed to process single block %d, %v", *singleBlock, err)
		}
		return
	}

	var ctype c.CrawlerType

	switch {
	case *monitorFinal:
		ctype = c.Finalized
	case *monitorLatest:
		ctype = c.Latest
	case *monitorOld:
		ctype = c.OldUnconfirmed
	}

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
			log.Infof("buck/%v: received an interrupt, canceling...", ctype)
			cancel()
		case <-ctx.Done():
			log.Infof("buck/%v: interrupt handler, canceling...", ctype)
			// If context is done, return the context error.
			return ctx.Err()
		}
		return nil
	})
	// without a global panic handler gocron does not recover from panics in
	// scheduled jobs i.e. a single panic would terminate the process
	gocron.SetPanicHandler(func(jobName string, recoverData interface{}) {
		log.Errorf("buck/%v: job '%s' panicked: %v", ctype, jobName, recoverData)
	})

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()

	if *monitorLatest {
		_, err = s.Every("1m").Do(monitorETH, context.WithValue(ctx, c.BuckContext, &latestCtxt))
		if err != nil {
			log.Fatal(err)
		}
	}

	if *monitorFinal {
		// don't clash with the healthcheck port of the other crawlers
		if *port == defaultPort {
			*port = defaultPort + 1
		}
		_, err = s.Every("1m").Do(monitorETH, context.WithValue(ctx, c.BuckContext, &finalCtxt))
		if err != nil {
			log.Fatal(err)
		}
	}

	if *monitorOld {
		// don't clash with the healthcheck port of the other crawlers
		if *port == defaultPort {
			*port = defaultPort + 2
		}
		oldCtxt.UpdateLastBlock = false
		_, err = s.Every("15m").Do(monitorOldUnconfirmed, context.WithValue(ctx, c.BuckContext, &oldCtxt))
		if err != nil {
			log.Fatal(err)
		}
	}

	// gocron discards the error a scheduled job returns unless an error event
	// listener is registered; it only covers the jobs scheduled so far, hence
	// this call comes after all of the s.Every(..).Do(..) calls above
	c.RegisterJobErrorListener(s, fmt.Sprintf("buck/%v", ctype))

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
			log.Infof("buck/%v - context canceled", ctype)
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
		log.Infof("buck/%v/http - context canceled, stopping..", ctype)
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
		log.Infof("buck/%v/cron - context canceled, stopping..", ctype)
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
		log.Errorf("buck/%v: errgroup.Wait(): %v", ctype, err)
		failed = true
	}
	log.Infof("buck/%v shutting down", ctype)
	if failed {
		os.Exit(1)
	}
}

func runReadyProbe(ctxt c.Context) error {
	err := ctxt.DB.Ping()
	if err != nil {
		log.Error(err)
		return err
	}
	ctxt.MaxWaitInSeconds = 1
	_, err = eth.MostRecentBlockNumber(ctxt)
	if err != nil {
		log.Error(err)
	}
	return err
}

// buckDeps is the set of ethereum/database operations the crawler loops
// perform. Production code uses liveDeps(), the tests substitute doubles: the
// block-range arithmetic of monitorETH and the verdict dispatch of
// monitorOldUnconfirmed decide whether a donation is recorded, confirmed or
// failed and must be exercised without a chain and without a database.
type buckDeps struct {
	mostRecentBlockNumber func(ctxt c.Context) (uint64, error)
	getLastBlock          func(dbh *sqlx.DB, chain, label string) (uint64, error)
	setLastBlock          func(ctxt c.Context, chain string, lbn uint64) error
	getTransactions       func(ctxt c.Context, bn uint64) ([]c.Transaction, error)
	getETHPrice           func(dbh *sqlx.DB, ts time.Time) (decimal.Decimal, error)
	requestPrice          func(ctxt c.Context, asset string, ts time.Time) error
	persistTxs            func(ctxt c.Context, bn uint64, ethPrice decimal.Decimal, txs []c.Transaction) error
	getOldUnconfirmed     func(dbh *sqlx.DB) ([]c.UnconfirmedTx, error)
	getTxsByHash          func(ctxt c.Context, hs []c.Hashable) ([]c.TxByHash, error)
	recordAbsence         func(ctxt c.Context, hash string, absent bool) (int, error)
	failTx                func(ctxt c.Context, tx c.TxByHash) error
	finalizeTx            func(ctxt c.Context, tx c.TxByHash) error
}

// liveDeps is the production wiring. It is a variable so that the tests can
// substitute it and verify that the wrappers below really hand *this* set of
// dependencies to the loops; nothing in production ever assigns to it.
var liveDeps = func() buckDeps {
	return buckDeps{
		mostRecentBlockNumber: eth.MostRecentBlockNumber,
		getLastBlock:          db.GetLastBlock,
		setLastBlock:          db.SetLastBlock,
		getTransactions:       eth.GetTransactions,
		getETHPrice:           c.GetETHPrice,
		requestPrice:          db.RequestPrice,
		persistTxs:            db.PersistTxs,
		getOldUnconfirmed:     db.GetOldUnconfirmed,
		getTxsByHash: func(ctxt c.Context, hs []c.Hashable) ([]c.TxByHash, error) {
			return eth.GetData[c.TxByHash](ctxt, hs, eth.TXBHFetcher{})
		},
		recordAbsence: db.RecordAbsence,
		failTx:        db.FailTx,
		finalizeTx:    db.FinalizeTx,
	}
}

func monitorETH(ctx context.Context) error {
	return monitorETHWith(ctx, liveDeps())
}

func monitorETHWith(ctx context.Context, deps buckDeps) error {
	ctxt, ok := ctx.Value(c.BuckContext).(*c.Context)
	if !ok {
		return errors.New("invalid buck context")
	}
	// most recent ETH block published
	mrbn, err := deps.mostRecentBlockNumber(*ctxt)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("===>> buck/%s tip of the ETH chain: %d", ctxt.CrawlerType, mrbn)

	// number of last block that was processed
	lpbn, err := deps.getLastBlock(ctxt.DB, "eth", ctxt.CrawlerType.String())
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
		err = processETHWith(*ctxt, i, deps)
		if err != nil {
			return err
		}
	}
	return nil
}

func processETH(ctxt c.Context, bn uint64) error {
	return processETHWith(ctxt, bn, liveDeps())
}

func processETHWith(ctxt c.Context, bn uint64, deps buckDeps) error {
	txs, err := deps.getTransactions(ctxt, bn)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("block %d: %d filtered transactions", bn, len(txs))
	if len(txs) == 0 {
		err = deps.setLastBlock(ctxt, "eth", bn)
		if err != nil {
			return err
		}
		return nil
	}

	// we only get the ETH price if we need to persist transactions
	// get ETH price at the time the block was published
	blockTime := txs[0].BlockTime
	ethPrice, err := deps.getETHPrice(ctxt.DB, blockTime)
	if err != nil {
		// request the missing price and let's hope it is avaiable next time we
		// need it
		log.Infof("requesting price for ETH/%s", blockTime.Format(time.RFC3339))
		err2 := deps.requestPrice(ctxt, "eth", blockTime)
		if err2 != nil {
			log.Errorf("failed to request price for ETH/%s, %v", blockTime, err2)
		}
		return err
	}
	log.Infof("eth price: %s", ethPrice)
	err = deps.persistTxs(ctxt, bn, ethPrice, txs)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func monitorOldUnconfirmed(ctx context.Context) error {
	return monitorOldUnconfirmedWith(ctx, liveDeps())
}

func monitorOldUnconfirmedWith(ctx context.Context, deps buckDeps) error {
	ctxt, ok := ctx.Value(c.BuckContext).(*c.Context)
	if !ok {
		return errors.New("buck/old-unconfirmed invalid buck context")
	}
	hashes, err := deps.getOldUnconfirmed(ctxt.DB)
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		log.Info("##### *no* unconfirmed txs older than 30 minutes")
		return nil
	}
	log.Infof("##### old unconfirmed txs: %v", hashes)

	// a transaction that is absent from the chain carries no block data; the
	// block time recorded for the donation is what decides whether it has
	// been gone long enough to be declared dead
	blockTimes := make(map[string]time.Time, len(hashes))
	for _, h := range hashes {
		blockTimes[c.NormalizeHash(h.Hash)] = h.BlockTime
	}

	// most recent *finalized* ETH block published
	mfbn, err := deps.mostRecentBlockNumber(*ctxt)
	if err != nil {
		return err
	}
	log.Infof("##### max finalized ETH block: %d", mfbn)

	txs, err := deps.getTxsByHash(*ctxt, c.ToHashable(hashes))
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
		// record this run's observation before judging: an absent transaction
		// has its consecutive-absence counter incremented, a present one has it
		// reset. The returned count -- which includes this run -- is what Judge
		// weighs against c.SustainedAbsenceRuns, so a single transient `null`
		// (count 1) can never drop a donation whose money arrived.
		var absentCount int
		absentCount, err = deps.recordAbsence(*ctxt, tx.Hash, tx.Absent)
		if err != nil {
			return err
		}
		if tx.Absent {
			tx.DBBlockTime = blockTimes[c.NormalizeHash(tx.Hash)]
			tx.AbsentCount = absentCount
		}
		// a donation is only ever failed on positive evidence (see
		// c.TxByHash.Judge): a pending tx or a tx we could not fetch the
		// finalized block for is left untouched and re-examined on a later
		// run
		switch verdict := tx.Judge(mfbn); verdict {
		case c.TxFail, c.TxDropped:
			log.Warnf("failing old unconfirmed tx (%s), %s", tx.Hash, verdict)
			err = deps.failTx(*ctxt, tx)
			if err != nil {
				return err
			}
		case c.TxFinalize:
			log.Infof("##### finalizing old tx %s", tx.Hash)
			err = deps.finalizeTx(*ctxt, tx)
			if err != nil {
				return err
			}
		default:
			log.Warnf("old unconfirmed tx (%s) left untouched, %s", tx.Hash, verdict)
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
