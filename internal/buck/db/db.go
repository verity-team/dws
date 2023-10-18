package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type Label int

const (
	Latest Label = iota
	Finalized
)

func (l Label) String() string {
	switch l {
	case Latest:
		return "latest"
	case Finalized:
		return "finalized"
	}
	return "invalid label"
}

func GetLastBlock(dbh *sqlx.DB, chain string, l Label) (uint64, error) {
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
	err = dbh.Get(&result, q, chain, l.String())
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		err = fmt.Errorf("failed to fetch last block for %s/%s, %v", chain, l.String(), err)
		log.Error(err)
		return 0, err
	}

	return result, nil
}

func SetLastBlock(dbh *sqlx.DB, chain string, l Label, lbn uint64) error {
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
	_, err = dbh.Exec(q, chain, l.String(), lbn)
	if err != nil {
		err = fmt.Errorf("failed to set last block for %s/%s, %v", l.String(), chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func PersistTxs(ctxt common.Context, bn uint64, ethPrice decimal.Decimal, txs []common.Transaction) error {
	var err error

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
		err = persistTx(dtx, tx)
		if err != nil {
			return err
		}
	}

	err = updateLastBlock(dtx, "eth", Latest, bn)
	if err != nil {
		return err
	}
	log.Infof("updated latest eth block to %d", bn)

	return nil
}

func updateLastBlock(dbt *sqlx.Tx, chain string, l Label, lbn uint64) error {
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
	_, err = dbt.Exec(q, chain, l.String(), lbn)
	if err != nil {
		err = fmt.Errorf("failed to set last block for %s/%s, %v", l.String(), chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func persistTx(dtx *sqlx.Tx, tx common.Transaction) error {
	q := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			:address, :amount, :usd_amount, :asset, :tokens, :price, :tx_hash,
			:status, :block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash) DO NOTHING
		`
	_, err := dtx.NamedExec(q, tx)
	if err != nil {
		err = fmt.Errorf("failed to insert donation for %s, %v", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

func calcTokens(tx common.Transaction, tokenPrice, ethPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	var amount decimal.Decimal
	amount, err := decimal.NewFromString(tx.Value)
	if err != nil {
		err = fmt.Errorf("invalid amount for tx %s, %v", tx.Hash, err)
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
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		// db error -> connection? is borked?
		err = fmt.Errorf("failed to fetch current token price, %v", err)
		log.Error(err)
		return decimal.Zero, err
	}
	if err != nil && strings.Contains(err.Error(), "sql: no rows in result set") {
		// no token price record in the database -> return cheapest price
		return ctxt.SaleParams[0].Price, nil
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
		err = fmt.Errorf("failed to insert failed block with number %d, %v", b.Number, err)
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
	_, err := dbh.Exec(q, b.Number, b.Hash, b.Timestamp, tx.Hash)
	if err != nil {
		err = fmt.Errorf("failed to insert failed tx '%s', %v", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

func persistFinalizedBlock(dtx *sqlx.Tx, fb common.FinalizedBlock) error {
	q := `
		INSERT INTO finalized_block(
		 base_fee_per_gas,
		 gas_limit,
		 gas_used,
		 block_hash,
		 block_number,
		 receipts_root,
		 block_size,
		 state_root,
		 block_time,
		 transactions)
		VALUES(
		 :base_fee_per_gas,
		 :gas_limit,
		 :gas_used,
		 :block_hash,
		 :block_number,
		 :receipts_root,
		 :block_size,
		 :state_root,
		 :block_time,
		 :transactions)
		ON CONFLICT (block_hash) DO NOTHING
		`
	txs := strings.Join(fb.Transactions, ",")
	qd := map[string]interface{}{
		"base_fee_per_gas": fb.BaseFeePerGas,
		"gas_limit":        fb.GasLimit,
		"gas_used":         fb.GasUsed,
		"block_hash":       fb.Hash,
		"block_number":     fb.Number,
		"receipts_root":    fb.ReceiptsRoot,
		"block_size":       fb.Size,
		"state_root":       fb.StateRoot,
		"block_time":       fb.Timestamp,
		"transactions":     txs,
	}
	_, err := dtx.NamedExec(q, qd)
	if err != nil {
		err = fmt.Errorf("failed to insert finalized block #%d ('%s'), %v", fb.Number, fb.Hash, err)
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
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		err = fmt.Errorf("failed to fetch oldest unconfirmed block number, %v", err)
		log.Error(err)
		return 0, err
	}

	return result, nil
}

func FinalizeTxs(ctxt common.Context, fb common.FinalizedBlock) error {
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
	// whatever happens -- try and update the last block at the end
	defer func() {
		// only update last block if all went well
		if err == nil {
			err = updateLastBlock(dtx, "eth", Finalized, fb.Number)
			if err != nil {
				log.Infof("updated last finalized eth block to %d", fb.Number)
			}
		}
	}()

	err = persistFinalizedBlock(dtx, fb)
	if err != nil {
		return err
	}
	utotal, utokens, err := unconfirmedTxsValue(dtx, fb)
	if err != nil {
		return err
	}
	log.Infof("finalized block #%d confirms %s USD / %d tokens", fb.Number, utotal.StringFixed(2), utokens.IntPart())

	if utotal.IsZero() {
		// nothing to do - return
		return nil
	}
	ra, err := confirmTxs(dtx, fb)
	if err != nil {
		return err
	}
	if ra != -1 {
		log.Infof("confirmed %d transactions for finalized block #%d", ra, fb.Number)
	}
	_, newTokens, err := updateDonationStats(dtx, utotal, utokens)
	if err != nil {
		return err
	}
	if doUpdate, newP := newTokenPrice(ctxt, utokens, newTokens); doUpdate {
		err = updateTokenPrice(dtx, newP)
		if err != nil {
			return err
		}
	}

	return nil
}

func unconfirmedTxsValue(dtx *sqlx.Tx, fb common.FinalizedBlock) (decimal.Decimal, decimal.Decimal, error) {
	var err error
	q := `
		SELECT COALESCE(SUM(usd_amount), 0) AS total, COALESCE(SUM(tokens), 0) AS tokens
		FROM donation
		WHERE status='unconfirmed' AND tx_hash in (?)
	`
	query, args, err := sqlx.In(q, fb.Transactions)
	if err != nil {
		err = fmt.Errorf("failed to prep query: get unconfirmed transactions stats for block %d (%s), %v", fb.Number, fb.Hash, err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	query = dtx.Rebind(query)
	type statss struct {
		Total  decimal.Decimal
		Tokens decimal.Decimal
	}
	var stats statss
	err = dtx.Get(&stats, query, args...)
	if err != nil {
		err = fmt.Errorf("failed to get unconfirmed transactions stats for block %d (%s), %v", fb.Number, fb.Hash, err)
		log.Error(err)
		return decimal.Zero, decimal.Zero, err
	}
	return stats.Total, stats.Tokens, nil
}

func confirmTxs(dtx *sqlx.Tx, fb common.FinalizedBlock) (int64, error) {
	var err error
	q := `
		UPDATE donation SET status='confirmed'
		WHERE status='unconfirmed' AND tx_hash in (?)
	`
	query, args, err := sqlx.In(q, fb.Transactions)
	if err != nil {
		err = fmt.Errorf("failed to prep query: confirm transactions for block %d (%s), %v", fb.Number, fb.Hash, err)
		log.Error(err)
		return -1, err
	}
	query = dtx.Rebind(query)
	result, err := dtx.Exec(query, args...)
	if err != nil {
		err = fmt.Errorf("failed to confirm transactions for block %d (%s), %v", fb.Number, fb.Hash, err)
		log.Error(err)
		return -1, err
	}
	ra, err := result.RowsAffected()
	if err != nil {
		err = fmt.Errorf("failed to get the count of confirmed transactions for block %d (%s), %v", fb.Number, fb.Hash, err)
		log.Error(err)
		return -1, nil
	}
	return ra, nil
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
		err = fmt.Errorf("failed to update donation stats %v", err)
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
		err = fmt.Errorf("failed to update token price to '%s', %v", ntp.StringFixed(5), err)
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
		err = fmt.Errorf("failed to request price for %s/%s, %v", asset, ts, err)
		log.Error(err)
		return err
	}

	return nil
}
