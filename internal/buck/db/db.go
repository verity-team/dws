package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
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

	// both crawlers issue tokens (the finalized one also recalculates the
	// donation stats at the end of this transaction) -> serialize with the
	// other donation stats writers before touching anything else. Holding the
	// lock is what makes the campaign decision below transaction consistent:
	// updateDonationStats runs in a donation stats writer, so the token total
	// cannot move while this transaction issues tokens.
	var (
		soldTokens     decimal.Decimal
		campaignStatus string
	)
	soldTokens, campaignStatus, err = lockDonationStats(dtx)
	if err != nil {
		return err
	}
	campaignClosed := campaignIsClosed(soldTokens, ctxt)
	// paused is the operator's manual kill switch, read here under the same
	// donation_stats lock as the token total so it cannot change while this
	// transaction issues tokens. While it is in effect the finalized crawler
	// records a freshly seen donation 'unconfirmed' (persistPausedTx) instead
	// of 'confirmed', so nothing is credited until the pause is lifted.
	campaignPaused := campaignStatus == campaignStatusPaused

	// the token price has to be read *inside* the transaction: read on the
	// pool it may be superseded by a tier change that commits before the
	// donations below are written, pricing them at the stale (cheaper) tier
	var tokenPrice decimal.Decimal
	tokenPrice, err = getTokenPrice(dtx, ctxt)
	if err != nil {
		return err
	}
	log.Infof("token price: %s", tokenPrice)

	for _, tx := range txs {
		tx.Price = tokenPrice.StringFixed(5)
		var calcErr error
		tx.Tokens, tx.USDAmount, calcErr = calcTokens(tx, tokenPrice, ethPrice)
		if calcErr != nil {
			log.Warnf("skipping tx %s: %v", tx.Hash, calcErr)
			continue
		}
		if campaignClosed && !campaignPaused && ctxt.CrawlerType == c.Finalized {
			// the campaign is over: the donation is recorded (the money
			// arrived and is refundable) but no tokens are issued -- the sale
			// cannot deliver them.
			//
			// A paused campaign takes precedence and is handled below: the
			// donation is recorded 'unconfirmed', so it is not confirmed here
			// and the closed rule -- which only fires on the transition to
			// 'confirmed' -- is evaluated when the pause is lifted, not now.
			// Zeroing the tokens of the 'unconfirmed' row it would leave
			// behind buys nothing and would strand the donation at 0 tokens if
			// the sale re-opened before it confirmed.
			//
			// Only the finalized crawler inserts a row that is *already*
			// 'confirmed', which is the state the rule is about. The latest
			// crawler inserts 'unconfirmed' rows: updateDonationStats never
			// sums those, so zeroing them buys nothing -- and it strands the
			// donation at 0 tokens for good if the campaign re-opens before
			// the donation is confirmed, since neither confirmation path
			// restores tokens. Such a row is priced when it transitions to
			// 'confirmed', by the campaign state that holds *then*.
			log.Warnf(
				"campaign is closed: issuing 0 tokens for donation '%s' from '%s' (%s %s / %s USD)",
				tx.Hash, tx.From, tx.Value, tx.Asset, tx.USDAmount.StringFixed(2))
			tx.Tokens = decimal.Zero
		}
		log.Infof("persisting tx: %5s -- a: %s, ausd: %s, t: %s, %s", tx.Asset, tx.Value, tx.USDAmount, tx.Tokens, tx.Hash)
		err = persistTx(dtx, tx, ctxt.CrawlerType, campaignClosed, campaignPaused)
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

// closedCampaignTokens is appended to the finalized crawler's upsert while
// the campaign is closed.
//
// The latest crawler runs ~13 minutes ahead of the finalized one, so donations
// mined after the token limit was crossed are inserted with tokens > 0: at
// that point nothing has told the database that the cap is reached. The cap is
// crossed in *finalization* order (updateDonationStats only ever sums
// 'confirmed' donations), so the transition to 'confirmed' is the coherent
// decision point: a donation that becomes confirmed after the campaign closed
// is over the cap and issues no tokens.
//
// The guard on the *existing* row's status is what keeps re-crawling
// idempotent: a donation that is already 'confirmed' was credited
// legitimately and keeps its tokens no matter how often its block is crawled
// again. Applying the closed rule is a state transition, not a re-pricing --
// amount/usd_amount/price are still never overwritten.
const closedCampaignTokens = `,
			tokens = CASE
				WHEN donation.status <> 'confirmed' AND EXCLUDED.status = 'confirmed'
				THEN 0
				ELSE donation.tokens
			END`

// donationUpsert is the shared INSERT ... ON CONFLICT (tx_hash) prefix of every
// donation write. The per-crawler and per-campaign-state conflict actions are
// appended to it so the column list exists in exactly one place.
const donationUpsert = `
		INSERT INTO donation(
			address, amount, usd_amount, asset, tokens, price, tx_hash, status,
			block_number, block_hash, block_time)
		VALUES(
			:address, :amount, :usd_amount, :asset, :tokens, :price, :tx_hash,
			:status, :block_number, :block_hash, :block_time)
		ON CONFLICT (tx_hash)`

// finalizedConflict is the finalized crawler's conflict action: the block
// identity is refreshed (a re-org may have re-mined the transaction), the
// sustained-absence streak is cleared and the status is advanced (the status
// expression is appended by the caller).
//
// Clearing absent_count is what keeps the sustained-absence guard honest: the
// finalized crawler seeing the transaction in a finalized block *is* a present
// observation, so any streak the old-unconfirmed crawler had accrued must not
// survive. Otherwise a row resurrected from 'failed' to 'unconfirmed' during a
// pause would carry a stale count >= c.SustainedAbsenceRuns, and the very first
// later transient `null` would drop a donation whose money verifiably arrived
// in a finalized block.
const finalizedConflict = `
		DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			block_number = EXCLUDED.block_number,
			block_time = EXCLUDED.block_time,
			absent_count = 0,
			`

// pausedStatusUpdate replaces the finalized upsert's plain
// `status = EXCLUDED.status` while the campaign is paused.
//
// Pause defers confirmation: a donation seen for the first time is stored
// 'unconfirmed' (persistPausedTx forces the *inserted* status, under the
// donation_stats lock the pause was read with), so updateDonationStats -- which
// sums only 'confirmed' rows -- credits nothing until an operator lifts the
// pause. The guard is the sharp edge: re-crawling a paused block must never
// downgrade a donation that is *already* 'confirmed', which would un-credit the
// donor and drop the amount out of the campaign totals. It keeps a confirmed
// row confirmed only when the incoming status is the pause's 'unconfirmed'; a
// receipt that reports a genuine revert (EXCLUDED.status = 'failed') still
// fails the donation, exactly as it does while the sale is open. This mirrors
// the closedCampaignTokens guard, which likewise turns on both the existing and
// the incoming status.
const pausedStatusUpdate = `status = CASE
				WHEN donation.status = 'confirmed' AND EXCLUDED.status = 'unconfirmed'
				THEN donation.status
				ELSE EXCLUDED.status
			END`

func persistTx(dtx *sqlx.Tx, tx c.Transaction, ct c.CrawlerType, campaignClosed, campaignPaused bool) error {
	var err error
	if ct != c.Latest && ct != c.Finalized {
		err = fmt.Errorf("invalid crawler type: %s", ct)
		log.Error(err)
		return err
	}
	if ct == c.Finalized && campaignPaused {
		return persistPausedTx(dtx, tx)
	}
	var q string
	if ct == c.Finalized {
		q = donationUpsert + finalizedConflict + `status = EXCLUDED.status`
		if campaignClosed {
			q += closedCampaignTokens
		}
	} else {
		// if the finalized crawler is running ahead of the latest
		// we do NOT want to overwrite the `block_*` properties and
		// the status
		q = donationUpsert + `
		DO NOTHING`
	}
	_, err = dtx.NamedExec(q, tx)
	if err != nil {
		err = fmt.Errorf("failed to upsert donation for %s, %w", tx.Hash, err)
		log.Error(err)
		return err
	}

	return nil
}

// persistPausedTx records a finalized-crawler donation while the campaign is
// paused. It defers confirmation instead of rejecting: a donation seen for the
// first time is stored 'unconfirmed' -- forcing the inserted status here, under
// the donation_stats lock the pause was read with, is what keeps the money on
// record without crediting it. A donation whose receipt reports a revert is
// still stored 'failed'. The conflict clause (pausedStatusUpdate) makes
// re-crawling a paused block safe: an already 'confirmed' donation is never
// downgraded to 'unconfirmed'. RETURNING is used to warn precisely -- only when
// the donation actually ends up held 'unconfirmed'.
func persistPausedTx(dtx *sqlx.Tx, tx c.Transaction) error {
	if tx.Status == string(api.Confirmed) {
		tx.Status = string(api.Unconfirmed)
	}
	q := donationUpsert + finalizedConflict + pausedStatusUpdate + `
		RETURNING status`
	rows, err := dtx.NamedQuery(q, tx)
	if err != nil {
		err = fmt.Errorf("failed to upsert paused donation for %s, %w", tx.Hash, err)
		log.Error(err)
		return err
	}
	defer rows.Close() // nolint:errcheck
	var status string
	if rows.Next() {
		if err = rows.Scan(&status); err != nil {
			err = fmt.Errorf("failed to read status of paused donation for %s, %w", tx.Hash, err)
			log.Error(err)
			return err
		}
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("failed to upsert paused donation for %s, %w", tx.Hash, err)
		log.Error(err)
		return err
	}
	if status == string(api.Unconfirmed) {
		log.Warnf(
			"campaign paused: donation '%s' from '%s' recorded 'unconfirmed'; it will be credited when the pause is lifted",
			tx.Hash, tx.From)
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

// getTokenPrice returns the token price the donations of this transaction are
// to be priced at. It is read inside the caller's transaction so that a tier
// change cannot slip in between the price lookup and the donation writes.
func getTokenPrice(dtx *sqlx.Tx, ctxt c.Context) (decimal.Decimal, error) {
	// select price as follows:
	// - if there are multiple token price rows then return the most recent
	//   one that is at least two minutes old
	// - if there is a single token price row just return it regardless of age
	//
	// the NOT EXISTS subquery has to be scoped to the token price rows:
	// pulitzer writes an `eth` row every minute, so an unscoped subquery is
	// dead from the first minute of operation and the single row case (fresh
	// deployment) falls through to ErrNoRows and the cheapest sale tier.
	q1 := `
		SELECT price
		FROM price
		WHERE
			 asset='truth'
			 AND (
				  created_at <= NOW() - INTERVAL '2 minutes'
				  OR NOT EXISTS (
						SELECT 1 FROM price p
						WHERE p.asset='truth' AND p.id <> price.id
				  )
			 )
		ORDER BY created_at DESC
		LIMIT 1
		`
	var result decimal.Decimal
	err := dtx.Get(&result, q1)
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

const (
	// campaignStatusOpen is the donation_stats status of a token sale that
	// still has tokens to issue.
	campaignStatusOpen = "open"
	// campaignStatusClosed is the donation_stats status of a token sale that
	// has reached its token limit: donations are still recorded but no tokens
	// are issued for them any more.
	campaignStatusClosed = "closed"
	// campaignStatusPaused is the operator's manual kill switch. It is set by
	// hand (UPDATE donation_stats SET status='paused') and is never written by
	// a crawler -- setCampaignStatus only ever manages the open <-> closed
	// pair. While it is in effect the finalized crawler records a freshly seen
	// donation 'unconfirmed' instead of 'confirmed' (persistPausedTx) and
	// confirmSingleTx does not confirm, so updateDonationStats -- which sums
	// only 'confirmed' rows -- credits nothing until an operator sets the
	// status back to 'open'.
	campaignStatusPaused = "paused"
)

// lockDonationStats takes an exclusive row lock on the donation stats and
// returns the confirmed token total and the campaign status.
// It has to be the *first* statement of every database transaction that ends
// up calling updateDonationStats or issuing tokens:
//   - all donation stats writers acquire the lock in the same order which
//     rules out deadlocks between them
//   - since the lock is taken in a separate, earlier statement the subsequent
//     aggregate UPDATE takes a fresh READ COMMITTED snapshot i.e. it sees the
//     donations committed by the writer we were waiting for. Locking within
//     the aggregate statement itself would *not* have that effect.
//   - the token total and the status are read under the same lock and hence
//     cannot change for the rest of the transaction: updateDonationStats is a
//     donation stats writer and blocks until this transaction is done. Reading
//     the status in the same statement is what lets a caller honour the manual
//     pause without a second lookup or a second lock, so the lock ordering is
//     left untouched.
//
// Please note: donation_stats holds a single row (enforced by the
// donation_stats_single_row index) so the lock covers every row in the table
// -- which is exactly what is needed here.
func lockDonationStats(dtx *sqlx.Tx) (decimal.Decimal, string, error) {
	q1 := `
		SELECT tokens, status
		FROM donation_stats
		FOR UPDATE
		`
	var row struct {
		Tokens decimal.Decimal `db:"tokens"`
		Status string          `db:"status"`
	}
	if err := dtx.Get(&row, q1); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// the campaign parameters are unknown -> refuse to issue tokens
			err = errors.New("donation_stats holds no row, is the database initialized?")
		}
		err = fmt.Errorf("failed to lock donation stats, %w", err)
		log.Error(err)
		return decimal.Zero, "", err
	}

	return row.Tokens, row.Status, nil
}

// campaignIsClosed decides whether the token sale can still deliver tokens.
//
// The decision is derived from the confirmed token total read under the
// donation stats lock, *not* from donation_stats.status: the status column
// records what the previous stats write concluded, while the limit itself
// comes from DWS_SALE_PARAMS and can be raised between runs. Trusting the
// stored status would have the first transaction after a raised limit issue 0
// tokens and only then re-open the campaign -- the donors in that batch would
// pay for tokens the sale had just put back on the shelf.
func campaignIsClosed(tokens decimal.Decimal, ctxt c.Context) bool {
	return tokens.GreaterThanOrEqual(ctxt.TokenSaleLimit())
}

func updateDonationStats(dtx *sqlx.Tx, ctxt c.Context) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	q1 := `
		WITH OldStats AS (
			 SELECT tokens AS old_tokens
			 FROM donation_stats
			 LIMIT 1
		),
		DonationSum AS (
			 -- COALESCE: SUM() over zero confirmed donations is NULL and
			 -- donation_stats.total/tokens are NOT NULL. This is reachable
			 -- (fresh database, every donation of the first finalized block
			 -- reverted) and would abort the crawler transaction.
			 SELECT
				  COALESCE(SUM(usd_amount), 0) AS total_usd_amount,
				  COALESCE(SUM(tokens), 0) AS total_tokens
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

	want := campaignStatusOpen
	if campaignIsClosed(newTokens, ctxt) {
		want = campaignStatusClosed
	}
	err = setCampaignStatus(dtx, want)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	return newTotal, newTokens, oldTokens, nil
}

// setCampaignStatus keeps the campaign status in sync with the token total
// that was just recomputed. The status is derived state, not a decision taken
// once: the sale closes when the confirmed tokens reach the limit and re-opens
// when they no longer do.
//
// Two reachable events put tokens back on sale:
//   - DWS_SALE_PARAMS' token limit is raised, so the same total is no longer
//     at the cap;
//   - a finalized re-crawl whose receipt reports an already *confirmed*
//     donation as reverted. persistTx writes status = EXCLUDED.status, the
//     donation becomes 'failed' and drops out of the aggregate below.
//
// Note it is *not* the old-unconfirmed crawler failing a phantom donation:
// failTx only ever transitions rows that are 'unconfirmed', and the aggregate
// above only ever sums rows that are 'confirmed', so such a donation was
// never in the total and failing it cannot lower anything.
//
// Without the re-open the tokens freed by either event could never be sold and
// every later donor would be issued 0 tokens although the limit is not
// reached -- the same harm the closed-campaign rule exists to prevent, in the
// other direction.
//
// Only the 'open' <-> 'closed' pair is managed here: 'paused' is the
// operator's manual lever and has to survive a crawler run untouched.
func setCampaignStatus(dtx *sqlx.Tx, want string) error {
	q1 := `
		UPDATE donation_stats
		SET status=$1::donation_stats_status_enum
		WHERE
			status <> $1::donation_stats_status_enum
			AND status IN ('open', 'closed')
		`
	res, err := dtx.Exec(q1, want)
	if err != nil {
		err = fmt.Errorf("failed to set donation_stats.status to '%s', %w", want, err)
		log.Error(err)
		return err
	}
	if ra, rerr := res.RowsAffected(); rerr == nil && ra > 0 {
		log.Warnf("campaign status changed to '%s'", want)
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

// oldUnconfirmedBatchLimit caps how many old-unconfirmed donations a single
// old-unconfirmed crawler run processes, so the resume path stays bounded.
//
// A pause holds every donation 'unconfirmed', so on a busy campaign the backlog
// can reach thousands of rows; without a cap GetOldUnconfirmed would return all
// of them and monitorOldUnconfirmed would hand the lot to eth_getTransactionByHash
// (and open a lock-taking FinalizeTx/FailTx transaction for each) in one run.
// Returning the oldest batch first (block_time ASC) lets successive 15-minute
// runs drain the backlog deterministically, oldest donations credited first.
// The eth layer (ethereum.GetData) further splits each run into HTTP batches of
// 127 sub-requests, so no single provider call approaches its batch limit -- the
// #201 failure mode; this cap bounds the *number* of those sub-batches per run.
// A few hundred keeps a run to a handful of them while still draining a large
// backlog in a manageable number of runs.
const oldUnconfirmedBatchLimit = 500

// GetOldUnconfirmed returns the transactions of the donations that are still
// unconfirmed 30 minutes after the block they were seen in, oldest first and
// capped at oldUnconfirmedBatchLimit per call. The block time is returned along
// with the hash: it is what decides whether a transaction that has vanished
// from the chain has been gone long enough to be a candidate for dropping (see
// c.DroppedTxGracePeriod). Whether that absence has been *sustained* rather
// than a single transient response is tracked separately, per run, by
// RecordAbsence.
func GetOldUnconfirmed(dbh *sqlx.DB) ([]c.UnconfirmedTx, error) {
	var (
		err    error
		q      string
		hashes []c.UnconfirmedTx
	)
	// oldest first (with tx_hash as a stable tie-breaker within a block time),
	// so a large backlog -- e.g. after a long pause -- drains deterministically
	// over successive runs and the oldest donors are credited first
	q = `
		SELECT DISTINCT tx_hash, timezone('utc', block_time) AS block_time
		FROM donation
		WHERE
			status = 'unconfirmed'
			AND timezone('utc', block_time) < timezone('utc', NOW()) - INTERVAL '30 minutes'
		ORDER BY block_time ASC, tx_hash ASC
		LIMIT $1
		`
	err = dbh.Select(&hashes, q, oldUnconfirmedBatchLimit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch old unconfirmed tx hashes, %w", err)
		log.Error(err)
		return nil, err
	}

	return hashes, nil
}

// RecordAbsence updates the sustained-absence bookkeeping for one donation on
// one old-unconfirmed run and returns the resulting consecutive-absence count.
//
// A transaction observed absent this run has its counter incremented; one
// observed present has it reset to 0 -- a single reappearance clears any
// accumulated absence, so only an *uninterrupted* run of absences can ever
// reach c.SustainedAbsenceRuns and let Judge return TxDropped. Only a row that
// is still 'unconfirmed' is touched: a donation confirmed or failed by another
// path has left the pool and its counter no longer matters. The returned count
// is fed straight into c.TxByHash.AbsentCount so the verdict this run is taken
// on the freshly written value.
func RecordAbsence(ctxt c.Context, hash string, absent bool) (int, error) {
	var (
		q     string
		count int
		err   error
	)
	if absent {
		q = `
			UPDATE donation SET absent_count = absent_count + 1
			WHERE tx_hash = $1 AND status = 'unconfirmed'
			RETURNING absent_count
			`
	} else {
		// only write when there is something to clear, so the common case (a
		// donation that was never absent) costs no row write
		q = `
			UPDATE donation SET absent_count = 0
			WHERE tx_hash = $1 AND status = 'unconfirmed' AND absent_count <> 0
			RETURNING absent_count
			`
	}
	err = ctxt.DB.Get(&count, q, c.NormalizeHash(hash))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// nothing to update: the donation is no longer 'unconfirmed', or it
			// is present and already at 0 -- either way the count is 0
			return 0, nil
		}
		err = fmt.Errorf("failed to record absence for %s, %w", hash, err)
		log.Error(err)
		return 0, err
	}

	return count, nil
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
	// other donation stats writers before touching anything else. The token
	// total is read under the same lock and cannot change for the rest of
	// this transaction.
	var (
		soldTokens     decimal.Decimal
		campaignStatus string
	)
	soldTokens, campaignStatus, err = lockDonationStats(dtx)
	if err != nil {
		return err
	}
	if campaignStatus == campaignStatusPaused {
		// pause defers confirmation: leave the donation 'unconfirmed' so the
		// old-unconfirmed crawler retries it once an operator lifts the pause.
		// Nothing is credited and nothing is failed -- the campaign totals stay
		// frozen. Judge never returns TxFinalize for a vanished transaction, so
		// a pause can only defer a confirmation here, never turn into a fail.
		log.Warnf("campaign paused: deferring confirmation of donation '%s' until the pause is lifted", tx.Hash)
		return nil
	}

	amount, tokens, err := confirmSingleTx(dtx, tx, campaignIsClosed(soldTokens, ctxt))
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

// confirmSingleTx moves a donation from 'unconfirmed' to 'confirmed'. The
// WHERE clause restricts the update to rows that actually make that
// transition, so zeroing the tokens of a donation that is confirmed while the
// campaign is closed cannot touch a donation that was credited legitimately
// earlier.
//
// The whole block identity is written, `block_hash` included. Judge compares
// the transaction's *current* block hash (from eth_getTransactionByHash)
// against the finalized block and returns TxFinalize only when the two match --
// but that block need not be the one the donation was first seen in: a re-org
// that re-mined the transaction into a different block leaves the recorded
// block_hash stale, and this update then replaces it. Writing block_number,
// block_time and block_hash together keeps all three belonging to one and the
// same block, which is precisely the consistency the donation(block_hash) index
// relies on -- a row split across two blocks would serve wrong sets from it.
func confirmSingleTx(dtx *sqlx.Tx, tx c.TxByHash, campaignClosed bool) (decimal.Decimal, decimal.Decimal, error) {
	q := `
		UPDATE donation SET
			block_number=$1,
			block_time=$2,
			-- an empty finalized block hash must not replace the hash the
			-- donation already carries; TxFinalize is unreachable without
			-- one, so this only rules out writing a value that is worse than
			-- what is there
			block_hash=COALESCE(NULLIF($5, ''), block_hash),
			status='confirmed',
			tokens=CASE WHEN $4 THEN 0 ELSE tokens END
		WHERE status='unconfirmed' AND tx_hash=$3
		RETURNING usd_amount, tokens
	`
	var amount, tokens decimal.Decimal
	err := dtx.QueryRowx(q, tx.BlockNumber, tx.FBBlockTime, tx.Hash, campaignClosed, c.NormalizeHash(tx.FBBlockHash)).Scan(&amount, &tokens)
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
	// the other donation stats writers before touching anything else. The pause
	// status is deliberately ignored here: a pause must never keep a genuinely
	// dropped transaction (positive evidence, see c.TxByHash.Judge) on the
	// books, and it cannot cause a fail either -- FailTx only ever runs for a
	// TxFail/TxDropped verdict, which a pause does not produce.
	_, _, err = lockDonationStats(dtx)
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
	// a transaction that vanished from the chain has no finalized block; the
	// time of the block it was originally seen in is all we can record for it
	blockTime := tx.FBBlockTime
	if blockTime.IsZero() {
		blockTime = tx.DBBlockTime
	}
	_, err := dtx.Exec(q1, tx.BlockNumber, tx.FBBlockHash, blockTime.UTC(), tx.Hash)
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
