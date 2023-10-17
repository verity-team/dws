package db

import (
	"fmt"
	"strings"

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

func PersistFinalizedBlock(dbh *sqlx.DB, fb common.FinalizedBlock) error {
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
	_, err := dbh.Exec(q, fb)
	if err != nil {
		err = fmt.Errorf("failed to insert finalized block '%s', %v", fb.Hash, err)
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
