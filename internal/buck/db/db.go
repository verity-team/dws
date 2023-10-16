package db

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/common"
)

func GetLastBlock(dbh *sqlx.DB, chain string) (uint64, uint64, error) {
	q1 := `
		SELECT
			latest, finalized
		FROM last_block
		WHERE chain=$1
		`
	var result []uint64
	err := dbh.Select(&result, q1, chain)
	if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {
		err = fmt.Errorf("failed to fetch last block for %s, %v", chain, err)
		log.Error(err)
		return 0, 0, err
	}

	return result[0], result[1], nil
}

func SetLastBlock(dbh *sqlx.DB, chain, label string, lbn uint64) error {
	var (
		err error
		q   string
	)
	if label != "latest" && label != "finalized" {
		err = fmt.Errorf("invalid label ('%s'), must be one of: latest, finalized", label)
		log.Error(err)
		return err
	}
	q = fmt.Sprintf(`
		INSERT INTO last_block(chain, %s) VALUES($1, $2)
		ON CONFLICT (chain)
		DO UPDATE SET %s = $2
	`, label, label)
	_, err = dbh.Exec(q, chain, lbn)
	if err != nil {
		err = fmt.Errorf("failed to set %s last block for %s, %v", label, chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func PersistTxs(ctxt common.Context, bn uint64, txs []common.Transaction) error {
	if len(txs) == 0 {
		return nil
	}
	var err error

	// get token price
	tokenPrice, err := getTokenPrice(ctxt)
	if err != nil {
		return err
	}
	log.Infof("token price: %s", tokenPrice)
	// get ETH price
	ethPrice, err := common.GetETHPrice(ctxt.DB)
	if err != nil {
		return err
	}
	log.Infof("eth price: %s", ethPrice)

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

	err = updateLastBlock(dtx, "eth", "latest", bn)
	if err != nil {
		return err
	}
	log.Infof("updated latest block to %d", bn)

	return nil
}

func updateLastBlock(dbt *sqlx.Tx, chain, label string, lbn uint64) error {
	var (
		err error
		q   string
	)
	if label != "latest" && label != "finalized" {
		err = fmt.Errorf("invalid label ('%s'), must be one of: latest, finalized", label)
		log.Error(err)
		return err
	}
	q = fmt.Sprintf(`
		INSERT INTO last_block(chain, %s) VALUES($1, $2)
		ON CONFLICT (chain)
		DO UPDATE SET %s = $2
	`, label, label)
	_, err = dbt.Exec(q, chain, lbn)
	if err != nil {
		err = fmt.Errorf("failed to set %s last block for %s, %v", label, chain, err)
		log.Error(err)
		return err
	}

	return nil
}

func persistTx(dtx *sqlx.Tx, tx common.Transaction) error {
	ethQ := `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			:address, :amount, :usd_amount, :asset, :tokens, :price, :tx_hash,
			:status, :block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash) DO NOTHING
		`
	usdxQ := `
		INSERT INTO donation(
			address, amount, asset, tokens, price, tx_hash, status, block_number,
			block_hash, block_time)
		VALUES(
			:address, :amount, :asset, :tokens, :price, :tx_hash, :status,
			:block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash) DO NOTHING
		`
	var q string = ethQ
	if tx.Asset != "eth" {
		q = usdxQ
	}
	_, err := dtx.NamedExec(q, tx)
	if err != nil {
		err = fmt.Errorf("failed to insert donation for %s, %v", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

func updateDonationData(dtx *sqlx.Tx, ctxt common.Context, txs []common.Transaction) (decimal.Decimal, decimal.Decimal, error) {
	if len(txs) == 0 {
		return decimal.Zero, decimal.Zero, nil
	}
	total, tokens, err := common.GetDonationStats(ctxt.DB)
	if err != nil {
		return decimal.Zero, decimal.Zero, nil
	}
	var newTotal, newTokens decimal.Decimal
	newTotal = total
	newTokens = tokens
	for _, tx := range txs {
		if tx.Status == string(api.Failed) {
			// skip failed transactions
			continue
		}
		newTokens.Add(tx.Tokens)
		if tx.Asset == "eth" {
			newTotal.Add(tx.USDAmount)
		} else {
			amount, err := decimal.NewFromString(tx.Value)
			if err != nil {
				err = fmt.Errorf("invalid amount for tx %s, %v", tx.Hash, err)
				log.Error(err)
				continue
			}
			newTotal.Add(amount)
		}
	}
	if newTotal.GreaterThan(total) || newTokens.GreaterThan(tokens) {
		err = updateDonationStats(dtx, newTotal, newTokens)
		if err != nil {
			return decimal.Zero, decimal.Zero, nil
		}
	}
	return tokens, newTokens, nil
}

func updateDonationStats(dtx *sqlx.Tx, newTotal, newTokens decimal.Decimal) error {
	q1 := `
		UPDATE donation_stats
			SET total = $1, tokens = $2
		`
	_, err := dtx.Exec(q1, newTotal, newTokens.IntPart())
	if err != nil {
		err = fmt.Errorf("failed to update donation stats %v", err)
		log.Error(err)
		return err
	}

	return nil
}

func updateTokenPrice(dtx *sqlx.Tx, newTokenPrice decimal.Decimal) error {
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
	_, err := dtx.Exec(q1, newTokenPrice.StringFixed(5))
	if err != nil {
		err = fmt.Errorf("failed to update token price %v", err)
		log.Error(err)
		return err
	}

	return nil
}

func doWeNeedToUpdatePrice(ctxt common.Context, tokens, newTokens decimal.Decimal) (bool, decimal.Decimal) {
	// did we enter a new price range? do we need to update the price?
	currentP := priceBucket(ctxt, tokens)
	newP := priceBucket(ctxt, tokens)

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
	return amount.Div(tokenPrice).Ceil(), decimal.Zero, nil
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
	err := ctxt.DB.Select(&result, q1)
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
