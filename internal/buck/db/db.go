package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	c "github.com/verity-team/dws/internal/common"
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

func SetLastBlock(ctxt c.Context, chain string, lbn uint64) error {
	if !ctxt.UpdateLastBlock {
		return nil
	}
	var (
		err error
		q   string
	)

	if ctxt.CrawlerType != c.Latest && ctxt.CrawlerType != c.Finalized {
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

func PersistTxs(ctxt c.Context, bn uint64, ethPrice decimal.Decimal, txs []c.Transaction) (err error) {
	if ctxt.CrawlerType != c.Latest && ctxt.CrawlerType != c.Finalized {
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
			return
		}
		if cerr := dtx.Commit(); cerr != nil {
			err = fmt.Errorf("failed to commit block #%d transaction, %w", bn, cerr)
			log.Error(err)
		}
	}()

	if ctxt.CrawlerType == c.Finalized {
		// the finalized crawler recalculates the donation stats at the end of
		// this transaction -> serialize with the other donation stats writers
		// before touching anything else
		err = lockDonationStats(dtx)
		if err != nil {
			return err
		}
	}

	for _, tx := range txs {
		tx.Price = tokenPrice.StringFixed(5)
		var calcErr error
		tx.Tokens, tx.USDAmount, calcErr = calcTokens(tx, tokenPrice, ethPrice)
		if calcErr != nil {
			log.Warnf("skipping tx %s: %v", tx.Hash, calcErr)
			continue
		}
		log.Infof("persisting tx: %5s -- a: %s, ausd: %s, t: %s, %s", tx.Asset, tx.Value, tx.USDAmount, tx.Tokens, tx.Hash)
		err = persistTx(dtx, tx, ctxt.CrawlerType)
		if err != nil {
			return err
		}
	}

	if ctxt.CrawlerType == c.Finalized {
		var total, newTokens, oldTokens decimal.Decimal
		total, newTokens, oldTokens, err = updateDonationStats(dtx, ctxt)
		if err != nil {
			return err
		}
		log.Infof("updated donation stats: total %s, tokens %s, block %d", total.StringFixed(2), newTokens, bn)
		if doUpdate, newP := ctxt.NewTokenPrice(oldTokens, newTokens); doUpdate {
			err = updateTokenPrice(dtx, newP)
			if err != nil {
				return err
			}
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

func persistTx(dtx *sqlx.Tx, tx c.Transaction, ct c.CrawlerType) error {
	var err error
	if ct != c.Latest && ct != c.Finalized {
		err = fmt.Errorf("invalid crawler type: %s", ct)
		log.Error(err)
		return err
	}
	var q string
	if ct == c.Finalized {
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

func calcTokens(tx c.Transaction, tokenPrice, ethPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
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

func getTokenPrice(ctxt c.Context) (decimal.Decimal, error) {
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

func PersistFailedBlock(dbh *sqlx.DB, b c.Block) error {
	q := `
		INSERT INTO failed_block(
			block_number, block_hash, block_time)
		VALUES(
			:block_number, :block_hash, :block_time)
		ON CONFLICT (block_number) DO NOTHING
		`
	if _, err := dbh.NamedExec(q, b); err != nil {
		err = fmt.Errorf("failed to insert failed block with number %d, %w", b.Number, err)
		log.Error(err)
		return err
	}

	return nil
}

func PersistFailedTx(dbh *sqlx.DB, b c.Block, tx c.Transaction) error {
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

// lockDonationStats takes an exclusive row lock on the donation stats.
// It has to be the *first* statement of every database transaction that ends
// up calling updateDonationStats:
//   - all donation stats writers acquire the lock in the same order which
//     rules out deadlocks between them
//   - since the lock is taken in a separate, earlier statement the subsequent
//     aggregate UPDATE takes a fresh READ COMMITTED snapshot i.e. it sees the
//     donations committed by the writer we were waiting for. Locking within
//     the aggregate statement itself would *not* have that effect.
//
// Please note: donation_stats holds a single row (this is not enforced by a
// constraint) so the lock covers every row in the table -- which is exactly
// what is needed here.
func lockDonationStats(dtx *sqlx.Tx) error {
	q1 := `
		SELECT tokens
		FROM donation_stats
		FOR UPDATE
		`
	if _, err := dtx.Exec(q1); err != nil {
		err = fmt.Errorf("failed to lock donation stats, %w", err)
		log.Error(err)
		return err
	}

	return nil
}

func updateDonationStats(dtx *sqlx.Tx, ctxt c.Context) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	q1 := `
		WITH OldStats AS (
			 SELECT tokens AS old_tokens
			 FROM donation_stats
			 LIMIT 1
		),
		DonationSum AS (
			 SELECT
				  SUM(usd_amount) AS total_usd_amount,
				  SUM(tokens) AS total_tokens
			 FROM donation
			 WHERE status = 'confirmed'
		)
		UPDATE donation_stats
		SET total = (SELECT total_usd_amount FROM DonationSum),
			 tokens = (SELECT total_tokens FROM DonationSum)
		RETURNING total, tokens, (SELECT old_tokens FROM OldStats)
		`
	var newTotal, newTokens, oldTokens decimal.Decimal
	err := dtx.QueryRowx(q1).Scan(&newTotal, &newTokens, &oldTokens)
	if err != nil {
		err = fmt.Errorf("failed to update donation stats %w", err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	if newTokens.GreaterThanOrEqual(ctxt.TokenSaleLimit()) {
		err = closeCampaign(dtx)
		if err != nil {
			return decimal.Zero, decimal.Zero, decimal.Zero, err
		}
	}
	return newTotal, newTokens, oldTokens, nil
}

func closeCampaign(dtx *sqlx.Tx) error {
	q1 := `
		UPDATE donation_stats
		SET status='closed'
		`
	if _, err := dtx.Exec(q1); err != nil {
		err = fmt.Errorf("failed to set donation_stats.status to 'closed', %w", err)
		log.Error(err)
		return err
	}

	return nil
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

func RequestPrice(ctxt c.Context, asset string, ts time.Time) error {
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

func GetOldUnconfirmed(dbh *sqlx.DB) ([]c.TXH, error) {
	var (
		err    error
		q      string
		hashes []c.TXH
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

func FinalizeTx(ctxt c.Context, tx c.TxByHash) (err error) {
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
			return
		}
		if cerr := dtx.Commit(); cerr != nil {
			err = fmt.Errorf("failed to commit finalized tx '%s', %w", tx.Hash, cerr)
			log.Error(err)
		}
	}()

	// this transaction recalculates the donation stats -> serialize with the
	// other donation stats writers before touching anything else
	err = lockDonationStats(dtx)
	if err != nil {
		return err
	}

	amount, tokens, err := confirmSingleTx(dtx, tx)
	if err != nil {
		return err
	}
	log.Infof("finalized tx '%s' confirms %s USD / %d tokens", tx.Hash, amount.StringFixed(2), tokens.IntPart())

	if amount.IsZero() {
		// nothing to do - return
		return nil
	}
	_, newTokens, oldTokens, err := updateDonationStats(dtx, ctxt)
	if err != nil {
		return err
	}
	if doUpdate, newP := ctxt.NewTokenPrice(oldTokens, newTokens); doUpdate {
		err = updateTokenPrice(dtx, newP)
		if err != nil {
			return err
		}
	}

	return nil
}

func confirmSingleTx(dtx *sqlx.Tx, tx c.TxByHash) (decimal.Decimal, decimal.Decimal, error) {
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
		if errors.Is(err, sql.ErrNoRows) {
			// tx was already confirmed or no longer unconfirmed
			log.Infof("tx '%s' not found with status 'unconfirmed', skipping", tx.Hash)
			return decimal.Zero, decimal.Zero, nil
		}
		err = fmt.Errorf("failed to confirm single transaction (%s), %w", tx.Hash, err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	return amount, tokens, nil
}

func FailTx(ctxt c.Context, tx c.TxByHash) (err error) {
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
			return
		}
		if cerr := dtx.Commit(); cerr != nil {
			err = fmt.Errorf("failed to commit failed tx '%s', %w", tx.Hash, cerr)
			log.Error(err)
		}
	}()

	// failing a donation removes it from the campaign totals -> serialize with
	// the other donation stats writers before touching anything else
	err = lockDonationStats(dtx)
	if err != nil {
		return err
	}

	var failed bool
	failed, err = failTx(dtx, tx)
	if err != nil {
		return err
	}
	if !failed {
		// the donation was confirmed (or failed) by another crawler while we
		// were fetching the transaction data from the ethereum node -> leave
		// the donation and the campaign totals alone
		log.Warnf("tx '%s' is no longer unconfirmed, donation stats left untouched", tx.Hash)
		return nil
	}
	log.Infof("failed tx '%s'", tx.Hash)

	// the donation no longer counts towards the campaign totals
	var total, tokens decimal.Decimal
	total, tokens, _, err = updateDonationStats(dtx, ctxt)
	if err != nil {
		return err
	}
	log.Infof("updated donation stats: total %s, tokens %s", total.StringFixed(2), tokens)

	return nil
}

// failTx records the transaction in the failed_tx table and marks the
// corresponding donation as 'failed'. The donation row is retained (it is not
// deleted) so that the aggregates -- which only ever count 'confirmed'
// donations -- stay consistent and the donation history is preserved.
// It returns true if the donation actually transitioned from 'unconfirmed' to
// 'failed'.
func failTx(dtx *sqlx.Tx, tx c.TxByHash) (bool, error) {
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
		return false, err
	}
	q2 := `
		UPDATE donation
		SET status='failed'
		WHERE tx_hash=$1 AND status='unconfirmed'
		`
	res, err := dtx.Exec(q2, tx.Hash)
	if err != nil {
		err = fmt.Errorf("failed to mark transaction (%s) as failed, %w", tx.Hash, err)
		log.Error(err)
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		err = fmt.Errorf("failed to get the number of donations failed for tx (%s), %w", tx.Hash, err)
		log.Error(err)
		return false, err
	}

	return ra > 0, nil
}
