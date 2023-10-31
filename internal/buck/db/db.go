package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

func GetLastBlock(dbh *sqlx.DB, chain string, label string) (uint64, error) {
	var (
		err    error
		q      string
		result uint64
	)
	q = `
		SELECT value
		FROM last_block
		WHERE chain=$1 AND label=$2
		`
	err = dbh.Get(&result, q, chain, label)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch last block for %s/%s, %w", chain, label, err)
		log.Error(err)
		return 0, err
	}

	return result, nil
}

func SetLastBlock(ctxt common.Context, chain string, lbn uint64) error {
	if !ctxt.UpdateLastBlock {
		return nil
	}
	var (
		err error
		q   string
	)

	if ctxt.CrawlerType != common.Latest && ctxt.CrawlerType != common.Finalized {
		err = fmt.Errorf("invalid crawler type: %s", ctxt.CrawlerType)
		log.Error(err)
		return err
	}

	q = `
		INSERT INTO last_block(chain, label, value) VALUES($1, $2, $3)
		ON CONFLICT (chain, label)
		DO UPDATE SET value = $3
		WHERE last_block.chain=$1 and last_block.label = $2
	`
	_, err = ctxt.DB.Exec(q, chain, ctxt.CrawlerType.String(), lbn)
	if err != nil {
		err = fmt.Errorf("failed to set last block for %s/%s, %w", ctxt.CrawlerType.String(), chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func PersistTxs(ctxt common.Context, bn uint64, ethPrice decimal.Decimal, txs []common.Transaction) error {
	var err error

	if ctxt.CrawlerType != common.Latest && ctxt.CrawlerType != common.Finalized {
		err = fmt.Errorf("invalid crawler type: %s", ctxt.CrawlerType)
		log.Error(err)
		return err
	}

	// get token price
	tokenPrice, err := getTokenPrice(ctxt)
	if err != nil {
		return err
	}
	log.Infof("token price: %s", tokenPrice)

	// start transaction
	dtx, err := ctxt.DB.Beginx()
	if err != nil {
		return err
	}
	// at the end of the function: commit if there are no errors,
	// roll back otherwise
	defer func() {
		if err != nil {
			dtx.Rollback() // nolint:errcheck
		} else {
			dtx.Commit() // nolint:errcheck
		}
	}()

	for _, tx := range txs {
		tx.Price = tokenPrice.StringFixed(5)
		tx.Tokens, tx.USDAmount, err = calcTokens(tx, tokenPrice, ethPrice)
		if err != nil {
			// something is wrong with this tx -- skip it
			continue
		}
		log.Infof("persisting tx: %5s -- a: %s, ausd: %s, t: %s, %s", tx.Asset, tx.Value, tx.USDAmount, tx.Tokens, tx.Hash)
		err = persistTx(dtx, tx, ctxt.CrawlerType)
		if err != nil {
			return err
		}
	}

	if ctxt.UpdateLastBlock {
		err = updateLastBlock(dtx, "eth", ctxt.CrawlerType.String(), bn)
		if err != nil {
			return err
		}
		log.Infof("updated last %s eth block to %d", ctxt.CrawlerType, bn)
	}

	return nil
}

func updateLastBlock(dbt *sqlx.Tx, chain string, label string, lbn uint64) error {
	var (
		err error
		q   string
	)
	q = `
		INSERT INTO last_block(chain, label, value) VALUES($1, $2, $3)
		ON CONFLICT (chain, label)
		DO UPDATE SET value = $3
		WHERE last_block.chain=$1 and last_block.label = $2
	`
	_, err = dbt.Exec(q, chain, label, lbn)
	if err != nil {
		err = fmt.Errorf("failed to update last block for %s/%s, %w", label, chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func persistTx(dtx *sqlx.Tx, tx common.Transaction, ct common.CrawlerType) error {
	var err error
	if ct != common.Latest && ct != common.Finalized {
		err = fmt.Errorf("invalid crawler type: %s", ct)
		log.Error(err)
		return err
	}
	var q string
	if ct == common.Finalized {
		q = `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			:address, :amount, :usd_amount, :asset, :tokens, :price, :tx_hash,
			:status, :block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash)
		DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			block_number = EXCLUDED.block_number,
			block_time = EXCLUDED.block_time,
			status = EXCLUDED.status
		`
	} else {
		// if the finalized crawler is running ahead of the latest
		// we do NOT want to overwrite the `block_*` properties and
		// the status
		q = `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			:address, :amount, :usd_amount, :asset, :tokens, :price, :tx_hash,
			:status, :block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash)
		DO NOTHING
		`
	}
	_, err = dtx.NamedExec(q, tx)
	if err != nil {
		err = fmt.Errorf("failed to upsert donation for %s, %w", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

func calcTokens(tx common.Transaction, tokenPrice, ethPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	var amount decimal.Decimal
	amount, err := decimal.NewFromString(tx.Value)
	if err != nil {
		err = fmt.Errorf("invalid amount for tx %s, %w", tx.Hash, err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}

	if tx.Asset == "eth" {
		// convert ETH amount to USD and then calculate based on the
		// token price
		usdAmount := amount.Mul(ethPrice)
		return usdAmount.Div(tokenPrice).Ceil(), usdAmount, nil
	}
	// amount is already denominated in USD
	return amount.Div(tokenPrice).Ceil(), amount, nil
}

func getTokenPrice(ctxt common.Context) (decimal.Decimal, error) {
	// select price as follows:
	// - if there are multiple rows than return the most recent one
	//   that is at least two minutes old
	// - if there is a single row just return it regardless of age
	q1 := `
		SELECT price
		FROM price
		WHERE
			 asset='truth'
			 AND (
				  created_at <= NOW() - INTERVAL '2 minutes'
				  OR NOT EXISTS (SELECT 1 FROM price p WHERE p.id <> price.id)
			 )
		ORDER BY created_at DESC
		LIMIT 1
		`
	var result decimal.Decimal
	err := ctxt.DB.Get(&result, q1)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no token price record in the database -> return cheapest price
			return ctxt.SaleParams[0].Price, nil
		}
		// db error -> connection? is borked?
		err = fmt.Errorf("failed to fetch current token price, %w", err)
		log.Error(err)
		return decimal.Zero, err
	}

	return result, nil
}

func PersistFailedBlock(dbh *sqlx.DB, b common.Block) error {
	q := `
		INSERT INTO failed_block(
			block_number, block_hash, block_time)
		VALUES(
			:block_number, :block_hash, :block_time)
		ON CONFLICT (block_number) DO NOTHING
		`
	_, err := dbh.NamedExec(q, b)
	if err != nil {
		err = fmt.Errorf("failed to insert failed block with number %d, %w", b.Number, err)
		log.Error(err)
		return err
	}

	return nil
}

func PersistFailedTx(dbh *sqlx.DB, b common.Block, tx common.Transaction) error {
	q := `
		INSERT INTO failed_tx(
			block_number, block_hash, block_time, tx_hash)
		VALUES(
			$1, $2, $3, $4)
		ON CONFLICT (tx_hash) DO NOTHING
		`
	_, err := dbh.Exec(q, b.Number, b.Hash, b.Timestamp.UTC(), tx.Hash)
	if err != nil {
		err = fmt.Errorf("failed to insert failed tx '%s', %w", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

func GetOldestUnconfirmed(dbh *sqlx.DB) (uint64, error) {
	var (
		err    error
		q      string
		result uint64
	)
	q = `
		SELECT block_number
		FROM donation
		WHERE status='unconfirmed'
		ORDER BY block_time
		LIMIT 1
		`
	err = dbh.Get(&result, q)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch oldest unconfirmed block number, %w", err)
		log.Error(err)
		return 0, err
	}

	return result, nil
}

func updateDonationStats(dtx *sqlx.Tx, incTotal, incTokens decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	q1 := `
		WITH updated_stats AS (
		 UPDATE donation_stats
		 SET total = total + $1,
			  tokens = tokens + $2
		 RETURNING total, tokens
		)
		SELECT total, tokens
		FROM updated_stats
		`
	var newTotal, newTokens decimal.Decimal
	err := dtx.QueryRowx(q1, incTotal, incTokens.IntPart()).Scan(&newTotal, &newTokens)
	if err != nil {
		err = fmt.Errorf("failed to update donation stats %w", err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}

	return newTotal, newTokens, nil
}

func updateTokenPrice(dtx *sqlx.Tx, ntp decimal.Decimal) error {
	q1 := `
		WITH check_existing AS (
			 SELECT 1
			 FROM price
			 WHERE asset = 'truth' AND price = $1
			 LIMIT 1
		)
		INSERT INTO price (asset, price)
		SELECT 'truth', $1
		WHERE NOT EXISTS (SELECT 1 FROM check_existing)
		RETURNING *;
		`
	_, err := dtx.Exec(q1, ntp.StringFixed(5))
	if err != nil {
		err = fmt.Errorf("failed to update token price to '%s', %w", ntp.StringFixed(5), err)
		log.Error(err)
		return err
	}

	return nil
}

func newTokenPrice(ctxt common.Context, incTokens, newTokens decimal.Decimal) (bool, decimal.Decimal) {
	// did we enter a new price range? do we need to update the price?
	currentP := priceBucket(ctxt, newTokens.Sub(incTokens))
	newP := priceBucket(ctxt, newTokens)

	return newP.GreaterThan(currentP), newP
}

func priceBucket(ctxt common.Context, tokens decimal.Decimal) decimal.Decimal {
	// find the correct price given the number of tokens sold
	for _, sp := range ctxt.SaleParams {
		if tokens.LessThan(decimal.NewFromInt(int64(sp.Limit))) {
			return sp.Price
		}
	}
	// we fell through the loop, return the max price
	return ctxt.SaleParams[len(ctxt.SaleParams)-1].Price
}

func RequestPrice(ctxt common.Context, asset string, ts time.Time) error {
	q := `
		INSERT INTO price_req(what_asset, what_time) VALUES($1, $2)
	   ON CONFLICT(what_asset, what_time) DO NOTHING
	`
	_, err := ctxt.DB.Exec(q, asset, ts.Round(time.Minute))
	if err != nil {
		err = fmt.Errorf("failed to request price for %s/%s, %w", asset, ts, err)
		log.Error(err)
		return err
	}

	return nil
}

func GetOldUnconfirmed(dbh *sqlx.DB) ([]string, error) {
	var (
		err    error
		q      string
		hashes []string
	)
	q = `
		SELECT DISTINCT tx_hash
		FROM donation
		WHERE
			status = 'unconfirmed'
			AND timezone('utc', block_time) < timezone('utc', NOW()) - INTERVAL '30 minutes'
		ORDER BY 1
		`
	err = dbh.Select(&hashes, q)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch old unconfirmed tx hashes, %w", err)
		log.Error(err)
		return nil, err
	}

	return hashes, nil
}

func FinalizeTx(ctxt common.Context, tx common.TxByHash) error {
	var err error
	// start transaction
	dtx, err := ctxt.DB.Beginx()
	if err != nil {
		return err
	}

	// at the end of the function: commit if there are no errors,
	// roll back otherwise
	defer func() {
		if err != nil {
			dtx.Rollback() // nolint:errcheck
		} else {
			dtx.Commit() // nolint:errcheck
		}
	}()

	amount, tokens, err := confirmSingleTx(dtx, tx)
	if err != nil {
		return err
	}
	log.Infof("finalized tx '%s' confirms %s USD / %d tokens", tx.Hash, amount.StringFixed(2), tokens.IntPart())

	if amount.IsZero() {
		// nothing to do - return
		return nil
	}
	_, newTokens, err := updateDonationStats(dtx, amount, tokens)
	if err != nil {
		return err
	}
	if doUpdate, newP := newTokenPrice(ctxt, tokens, newTokens); doUpdate {
		err = updateTokenPrice(dtx, newP)
		if err != nil {
			return err
		}
	}

	return nil
}

func confirmSingleTx(dtx *sqlx.Tx, tx common.TxByHash) (decimal.Decimal, decimal.Decimal, error) {
	q := `
		UPDATE donation SET
			block_number=$1,
			block_time=$2,
			status='confirmed'
		WHERE status='unconfirmed' AND tx_hash=$3
		RETURNING usd_amount, tokens
	`
	var amount, tokens decimal.Decimal
	err := dtx.QueryRowx(q, tx.BlockNumber, tx.FBBlockTime, tx.Hash).Scan(&amount, &tokens)
	if err != nil {
		err = fmt.Errorf("failed to confirm single transaction (%s), %w", tx.Hash, err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	return amount, tokens, nil
}

func FailTx(ctxt common.Context, tx common.TxByHash) error {
	var err error
	// start transaction
	dtx, err := ctxt.DB.Beginx()
	if err != nil {
		return err
	}

	// at the end of the function: commit if there are no errors,
	// roll back otherwise
	defer func() {
		if err != nil {
			dtx.Rollback() // nolint:errcheck
		} else {
			dtx.Commit() // nolint:errcheck
		}
	}()

	err = failTx(dtx, tx)
	if err != nil {
		return err
	}
	log.Infof("failed tx '%s'", tx.Hash)

	return nil
}

func failTx(dtx *sqlx.Tx, tx common.TxByHash) error {
	q1 := `
		INSERT INTO failed_tx(
			block_number, block_hash, block_time, tx_hash)
		VALUES(
			$1, $2, $3, $4)
		ON CONFLICT (tx_hash) DO NOTHING
		`
	_, err := dtx.Exec(q1, tx.BlockNumber, tx.FBBlockHash, tx.FBBlockTime.UTC(), tx.Hash)
	if err != nil {
		err = fmt.Errorf("failed to insert failed tx '%s', %w", tx.Hash, err)
		log.Error(err)
		return err
	}
	q2 := `
		DELETE FROM donation WHERE tx_hash=$1
	`
	_, err = dtx.Exec(q2, tx.Hash)
	if err != nil {
		err = fmt.Errorf("failed to delete transaction (%s), %w", tx.Hash, err)
		log.Error(err)
		return err
	}
	return nil
}
