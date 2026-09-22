package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/internal/buck/db"
	eth "github.com/verity-team/dws/internal/buck/ethereum"
	c "github.com/verity-team/dws/internal/common"
)

// errBoom is the canned failure the doubles below return; the assertions match
// on it with errors.Is so that a swallowed or re-wrapped error is visible.
var errBoom = errors.New("boom")

// recorder captures what the crawler loops did: which blocks were fetched and
// which cursor/persistence/verdict calls were made, in order.
type recorder struct {
	fetched     []uint64     // blocks handed to getTransactions
	cursor      []uint64     // blocks handed to setLastBlock
	persisted   []uint64     // blocks handed to persistTxs
	prices      []time.Time  // block times looked up via getETHPrice
	priceReqs   []time.Time  // block times handed to requestPrice
	failed      []c.TxByHash // txs handed to FailTx
	finalized   []c.TxByHash // txs handed to FinalizeTx
	absences    []string     // "<hash>:<absent>" handed to recordAbsence
	mrbnCalls   int
	byHashCalls int
}

// stubDeps is an all-successful set of doubles; every test overrides only the
// behaviour it is about.
func stubDeps(r *recorder) buckDeps {
	return buckDeps{
		mostRecentBlockNumber: func(c.Context) (uint64, error) {
			r.mrbnCalls++
			return 0, nil
		},
		getLastBlock: func(*sqlx.DB, string, string) (uint64, error) { return 0, nil },
		setLastBlock: func(_ c.Context, _ string, lbn uint64) error {
			r.cursor = append(r.cursor, lbn)
			return nil
		},
		getTransactions: func(_ c.Context, bn uint64) ([]c.Transaction, error) {
			r.fetched = append(r.fetched, bn)
			return nil, nil
		},
		getETHPrice: func(_ *sqlx.DB, ts time.Time) (decimal.Decimal, error) {
			r.prices = append(r.prices, ts)
			return decimal.RequireFromString("1234.56"), nil
		},
		requestPrice: func(_ c.Context, _ string, ts time.Time) error {
			r.priceReqs = append(r.priceReqs, ts)
			return nil
		},
		persistTxs: func(_ c.Context, bn uint64, _ decimal.Decimal, _ []c.Transaction) error {
			r.persisted = append(r.persisted, bn)
			return nil
		},
		getOldUnconfirmed: func(*sqlx.DB) ([]c.UnconfirmedTx, error) { return nil, nil },
		getTxsByHash: func(c.Context, []c.Hashable) ([]c.TxByHash, error) {
			r.byHashCalls++
			return nil, nil
		},
		// by default report an absence as already sustained, so the dispatch
		// tests exercise the verdict->action mapping without threading a run
		// count; the tests about the count itself override this.
		recordAbsence: func(_ c.Context, hash string, absent bool) (int, error) {
			r.absences = append(r.absences, fmt.Sprintf("%s:%t", hash, absent))
			if absent {
				return c.SustainedAbsenceRuns, nil
			}
			return 0, nil
		},
		failTx: func(_ c.Context, tx c.TxByHash) error {
			r.failed = append(r.failed, tx)
			return nil
		},
		finalizeTx: func(_ c.Context, tx c.TxByHash) error {
			r.finalized = append(r.finalized, tx)
			return nil
		},
	}
}

// buckCtx wraps a buck context the way main() hands it to the scheduled jobs.
func buckCtx(ct c.CrawlerType) context.Context {
	ctxt := &c.Context{CrawlerType: ct}
	return context.WithValue(context.Background(), c.BuckContext, ctxt)
}

func TestMonitorETHRejectsInvalidBuckContext(t *testing.T) {
	r := &recorder{}
	err := monitorETHWith(context.Background(), stubDeps(r))
	assert.ErrorContains(t, err, "invalid buck context")
	assert.Zero(t, r.mrbnCalls, "no chain access without a valid context")
}

// The block range is lpbn+1 .. mrbn inclusive; a cursor of zero (no usable
// value in the database) starts at the tip rather than at block 1 -- walking
// the chain from genesis would take days and the crawler would never catch up.
func TestMonitorETHBlockRange(t *testing.T) {
	tests := []struct {
		name string
		lpbn uint64
		mrbn uint64
		want []uint64
	}{
		{name: "no cursor starts at the tip", lpbn: 0, mrbn: 100, want: []uint64{100}},
		{name: "cursor one behind the tip", lpbn: 99, mrbn: 100, want: []uint64{100}},
		{name: "cursor several blocks behind", lpbn: 97, mrbn: 100, want: []uint64{98, 99, 100}},
		{name: "cursor at the tip does nothing", lpbn: 100, mrbn: 100, want: nil},
		{name: "cursor ahead of the tip does nothing", lpbn: 101, mrbn: 100, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &recorder{}
			deps := stubDeps(r)
			deps.mostRecentBlockNumber = func(c.Context) (uint64, error) { return tc.mrbn, nil }
			deps.getLastBlock = func(*sqlx.DB, string, string) (uint64, error) { return tc.lpbn, nil }

			require.NoError(t, monitorETHWith(buckCtx(c.Latest), deps))
			assert.Equal(t, tc.want, r.fetched)
			// every block came back empty, so the cursor advanced to each
			assert.Equal(t, tc.want, r.cursor)
		})
	}
}

func TestMonitorETHQueriesTheCursorForItsOwnCrawlerType(t *testing.T) {
	var chain, label string
	r := &recorder{}
	deps := stubDeps(r)
	deps.getLastBlock = func(_ *sqlx.DB, ch, lb string) (uint64, error) {
		chain, label = ch, lb
		return 0, nil
	}
	require.NoError(t, monitorETHWith(buckCtx(c.Finalized), deps))
	assert.Equal(t, "eth", chain)
	assert.Equal(t, "finalized", label)
}

// An error must stop the loop *without* advancing the cursor: a crawler that
// skipped past a failed block would lose every donation in it for good.
func TestMonitorETHStopsOnErrorWithoutAdvancing(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.mostRecentBlockNumber = func(c.Context) (uint64, error) { return 103, nil }
	deps.getLastBlock = func(*sqlx.DB, string, string) (uint64, error) { return 100, nil }
	deps.getTransactions = func(_ c.Context, bn uint64) ([]c.Transaction, error) {
		r.fetched = append(r.fetched, bn)
		if bn == 102 {
			return nil, errBoom
		}
		return nil, nil
	}

	err := monitorETHWith(buckCtx(c.Latest), deps)
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, []uint64{101, 102}, r.fetched, "block 103 must not be touched")
	assert.Equal(t, []uint64{101}, r.cursor, "the failed block must not advance the cursor")
}

func TestMonitorETHPropagatesTipError(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.mostRecentBlockNumber = func(c.Context) (uint64, error) { return 0, errBoom }

	err := monitorETHWith(buckCtx(c.Latest), deps)
	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, r.fetched)
	assert.Empty(t, r.cursor)
}

func TestMonitorETHPropagatesCursorReadError(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.mostRecentBlockNumber = func(c.Context) (uint64, error) { return 100, nil }
	deps.getLastBlock = func(*sqlx.DB, string, string) (uint64, error) { return 0, errBoom }

	err := monitorETHWith(buckCtx(c.Latest), deps)
	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, r.fetched, "an unknown cursor must not start at the tip")
}

// A canceled context ends the run cleanly (nil) between blocks -- shutdown is
// not a failure and must not be reported as one.
func TestMonitorETHStopsOnContextCancelation(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.mostRecentBlockNumber = func(c.Context) (uint64, error) { return 105, nil }
	deps.getLastBlock = func(*sqlx.DB, string, string) (uint64, error) { return 100, nil }

	ctx, cancel := context.WithCancel(buckCtx(c.Latest))
	deps.getTransactions = func(_ c.Context, bn uint64) ([]c.Transaction, error) {
		r.fetched = append(r.fetched, bn)
		cancel()
		return nil, nil
	}
	defer cancel()

	assert.NoError(t, monitorETHWith(ctx, deps))
	assert.Equal(t, []uint64{101}, r.fetched, "the loop must not continue past cancelation")
}

// An empty block advances the cursor and must not touch the price or donation
// tables.
func TestProcessETHEmptyBlockAdvancesCursorOnly(t *testing.T) {
	r := &recorder{}
	require.NoError(t, processETHWith(c.Context{}, 4711, stubDeps(r)))
	assert.Equal(t, []uint64{4711}, r.cursor)
	assert.Empty(t, r.prices)
	assert.Empty(t, r.persisted)
}

func TestProcessETHPropagatesCursorWriteError(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.setLastBlock = func(c.Context, string, uint64) error { return errBoom }

	assert.ErrorIs(t, processETHWith(c.Context{}, 4711, deps), errBoom)
}

func TestProcessETHPropagatesFetchError(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.getTransactions = func(c.Context, uint64) ([]c.Transaction, error) { return nil, errBoom }

	assert.ErrorIs(t, processETHWith(c.Context{}, 4711, deps), errBoom)
	assert.Empty(t, r.cursor, "a block we could not fetch must not advance the cursor")
}

func testBlockTime() time.Time {
	return time.Date(2023, 10, 15, 0, 15, 59, 0, time.UTC)
}

func blockTxs() []c.Transaction {
	return []c.Transaction{
		{TXH: c.TXH{Hash: "0xaa"}, BlockTime: testBlockTime()},
		{TXH: c.TXH{Hash: "0xbb"}, BlockTime: testBlockTime()},
	}
}

// The ETH price is looked up for the time of the block being processed, not
// for "now" -- a donation is valued at the rate that applied when it was made.
func TestProcessETHPersistsTxsAtTheBlockTimePrice(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.getTransactions = func(c.Context, uint64) ([]c.Transaction, error) { return blockTxs(), nil }
	var (
		gotBn    uint64
		gotPrice decimal.Decimal
		gotTxs   []c.Transaction
	)
	deps.persistTxs = func(_ c.Context, bn uint64, p decimal.Decimal, txs []c.Transaction) error {
		gotBn, gotPrice, gotTxs = bn, p, txs
		return nil
	}

	require.NoError(t, processETHWith(c.Context{}, 4711, deps))
	assert.Equal(t, []time.Time{testBlockTime()}, r.prices)
	assert.Equal(t, uint64(4711), gotBn)
	assert.Equal(t, "1234.56", gotPrice.String())
	assert.Equal(t, blockTxs(), gotTxs)
	assert.Empty(t, r.cursor, "a non-empty block advances the cursor inside PersistTxs")
	assert.Empty(t, r.priceReqs)
}

// A missing price must leave the block unprocessed *and* unrecorded: the price
// is requested and the block is retried on the next run.
func TestProcessETHRequestsMissingPrice(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.getTransactions = func(c.Context, uint64) ([]c.Transaction, error) { return blockTxs(), nil }
	deps.getETHPrice = func(*sqlx.DB, time.Time) (decimal.Decimal, error) { return decimal.Zero, errBoom }
	var gotAsset string
	deps.requestPrice = func(_ c.Context, asset string, ts time.Time) error {
		gotAsset = asset
		r.priceReqs = append(r.priceReqs, ts)
		return nil
	}

	err := processETHWith(c.Context{}, 4711, deps)
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, "eth", gotAsset)
	assert.Equal(t, []time.Time{testBlockTime()}, r.priceReqs)
	assert.Empty(t, r.persisted)
	assert.Empty(t, r.cursor)
}

// A failed price *request* must not mask the missing price: the block still
// has to be retried.
func TestProcessETHReturnsPriceErrorWhenTheRequestAlsoFails(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.getTransactions = func(c.Context, uint64) ([]c.Transaction, error) { return blockTxs(), nil }
	deps.getETHPrice = func(*sqlx.DB, time.Time) (decimal.Decimal, error) { return decimal.Zero, errBoom }
	deps.requestPrice = func(c.Context, string, time.Time) error { return errors.New("request failed") }

	err := processETHWith(c.Context{}, 4711, deps)
	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, r.persisted)
	assert.Empty(t, r.cursor)
}

func TestProcessETHPropagatesPersistError(t *testing.T) {
	r := &recorder{}
	deps := stubDeps(r)
	deps.getTransactions = func(c.Context, uint64) ([]c.Transaction, error) { return blockTxs(), nil }
	deps.persistTxs = func(c.Context, uint64, decimal.Decimal, []c.Transaction) error { return errBoom }

	assert.ErrorIs(t, processETHWith(c.Context{}, 4711, deps), errBoom)
	assert.Empty(t, r.cursor)
}

func TestMonitorOldUnconfirmedRejectsInvalidBuckContext(t *testing.T) {
	r := &recorder{}
	err := monitorOldUnconfirmedWith(context.Background(), stubDeps(r))
	assert.ErrorContains(t, err, "buck/old-unconfirmed invalid buck context")
	assert.Zero(t, r.byHashCalls)
}

// Nothing to check means no chain access at all.
func TestMonitorOldUnconfirmedWithoutCandidates(t *testing.T) {
	r := &recorder{}
	require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), stubDeps(r)))
	assert.Zero(t, r.mrbnCalls)
	assert.Zero(t, r.byHashCalls)
	assert.Empty(t, r.failed)
	assert.Empty(t, r.finalized)
}

const (
	testTxHash    = "0x3c8273e0d522380ed5c1caf943ede820251581dc77cc6b80b68a47d926b586cc"
	testBlockHash = "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367"
	testMFBN      = uint64(4489455)
)

func unconfirmed(hash string, bt time.Time) []c.UnconfirmedTx {
	return []c.UnconfirmedTx{{TXH: c.TXH{Hash: hash}, BlockTime: bt}}
}

// oldUnconfirmedDeps wires the doubles for the old-unconfirmed crawler: the
// database hands back `hashes` and the chain hands back `txs`.
func oldUnconfirmedDeps(r *recorder, hashes []c.UnconfirmedTx, txs []c.TxByHash) buckDeps {
	deps := stubDeps(r)
	deps.getOldUnconfirmed = func(*sqlx.DB) ([]c.UnconfirmedTx, error) { return hashes, nil }
	deps.mostRecentBlockNumber = func(c.Context) (uint64, error) {
		r.mrbnCalls++
		return testMFBN, nil
	}
	deps.getTxsByHash = func(c.Context, []c.Hashable) ([]c.TxByHash, error) {
		r.byHashCalls++
		return txs, nil
	}
	return deps
}

// Only TxFail and TxDropped may fail a donation. Every other verdict means we
// do not know yet -- failing on absence of evidence would destroy a donation
// that is merely slow.
func TestMonitorOldUnconfirmedVerdictDispatch(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name         string
		blockTime    time.Time
		tx           c.TxByHash
		wantFail     bool
		wantFinalize bool
	}{
		{
			name: "finalized block carries the tx",
			tx: c.TxByHash{
				Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
				FBBlockHash: testBlockHash, FBContainsTx: true, FBDataAvailable: true,
			},
			wantFinalize: true,
		},
		{
			name: "finalized block does not carry the tx",
			tx: c.TxByHash{
				Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
				FBBlockHash: testBlockHash, FBContainsTx: false, FBDataAvailable: true,
			},
			wantFail: true,
		},
		{
			name: "finalized block hash differs",
			tx: c.TxByHash{
				Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
				FBBlockHash: "0xdeadbeef", FBContainsTx: true, FBDataAvailable: true,
			},
			wantFail: true,
		},
		{
			name: "still in the mempool",
			tx: c.TxByHash{
				Hash: testTxHash, Pending: true,
				BlockNumber: c.PendingBlockNumber, TransactionIndex: c.PendingTransactionIndex,
			},
		},
		{
			name: "block not finalized yet",
			tx:   c.TxByHash{Hash: testTxHash, BlockNumber: testMFBN + 1, BlockHash: testBlockHash},
		},
		{
			name: "no finalized block data",
			tx: c.TxByHash{
				Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
				FBDataAvailable: false,
			},
		},
		{
			name:      "absent within the grace period",
			blockTime: now.Add(-1 * time.Hour),
			tx:        c.TxByHash{Hash: testTxHash, Absent: true},
		},
		{
			name:      "absent beyond the grace period",
			blockTime: now.Add(-c.DroppedTxGracePeriod - time.Hour),
			tx:        c.TxByHash{Hash: testTxHash, Absent: true},
			wantFail:  true,
		},
		{
			name:      "absent with an unknown block time",
			blockTime: time.Time{},
			tx:        c.TxByHash{Hash: testTxHash, Absent: true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &recorder{}
			deps := oldUnconfirmedDeps(r,
				unconfirmed(testTxHash, tc.blockTime), []c.TxByHash{tc.tx})

			require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
			assert.Equal(t, tc.wantFail, len(r.failed) == 1, "FailTx calls: %d", len(r.failed))
			assert.Equal(t, tc.wantFinalize, len(r.finalized) == 1, "FinalizeTx calls: %d", len(r.finalized))
		})
	}
}

// #224 F1: a transaction absent past the grace period but not yet on a
// *sustained* run of absences (a single transient provider `null`) must not be
// failed. recordAbsence reports the run count; below the threshold the verdict
// is TxAbsent and the donation is left to be retried.
func TestMonitorOldUnconfirmedDoesNotFailATransientAbsence(t *testing.T) {
	bt := time.Now().UTC().Add(-c.DroppedTxGracePeriod - time.Hour)
	r := &recorder{}
	deps := oldUnconfirmedDeps(r,
		unconfirmed(testTxHash, bt),
		[]c.TxByHash{{Hash: testTxHash, Absent: true}})
	// this is the first run the transaction is seen absent -> count 1
	deps.recordAbsence = func(_ c.Context, _ string, _ bool) (int, error) {
		return 1, nil
	}

	require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
	assert.Empty(t, r.failed, "a single transient absence must not fail a donation")
	assert.Empty(t, r.finalized)
}

// #224 F1: the observation is recorded for every polled transaction -- absent
// or present -- so the sustained-absence counter is kept up to date each run.
func TestMonitorOldUnconfirmedRecordsEveryObservation(t *testing.T) {
	bt := time.Now().UTC().Add(-c.DroppedTxGracePeriod - time.Hour)
	r := &recorder{}
	deps := oldUnconfirmedDeps(r,
		unconfirmed(testTxHash, bt),
		[]c.TxByHash{{Hash: testTxHash, Absent: true}})

	require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
	assert.Equal(t, []string{testTxHash + ":true"}, r.absences)
}

// An absent transaction carries no block data at all, so the block time
// recorded for the donation is what Judge has to go on; it is stamped onto the
// transaction before the verdict is taken.
func TestMonitorOldUnconfirmedStampsDBBlockTime(t *testing.T) {
	bt := time.Now().UTC().Add(-c.DroppedTxGracePeriod - time.Hour)
	r := &recorder{}
	deps := oldUnconfirmedDeps(r,
		unconfirmed(testTxHash, bt),
		[]c.TxByHash{{Hash: testTxHash, Absent: true}})

	require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
	require.Len(t, r.failed, 1)
	assert.Equal(t, bt, r.failed[0].DBBlockTime)
}

// Providers return either casing and a row may predate the normalization, so
// both sides of the hash -> block time map are normalized; a donation must not
// be kept alive forever because the two spellings disagree.
func TestMonitorOldUnconfirmedStampsDBBlockTimeRegardlessOfHashCasing(t *testing.T) {
	const upper = "0x3C8273E0D522380ED5C1CAF943EDE820251581DC77CC6B80B68A47D926B586CC"
	tests := []struct {
		name      string
		dbHash    string
		chainHash string
	}{
		{name: "provider reports upper case", dbHash: testTxHash, chainHash: upper},
		{name: "database row is upper case", dbHash: upper, chainHash: testTxHash},
		{name: "database row has surrounding whitespace", dbHash: " " + testTxHash + "\n", chainHash: testTxHash},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bt := time.Now().UTC().Add(-c.DroppedTxGracePeriod - time.Hour)
			r := &recorder{}
			deps := oldUnconfirmedDeps(r,
				unconfirmed(tc.dbHash, bt),
				[]c.TxByHash{{Hash: tc.chainHash, Absent: true}})

			require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
			require.Len(t, r.failed, 1)
			assert.Equal(t, bt, r.failed[0].DBBlockTime)
		})
	}
}

// Only an absent transaction is stamped: a mined transaction's own block data
// is the evidence and must not be overwritten.
func TestMonitorOldUnconfirmedDoesNotStampAMinedTx(t *testing.T) {
	bt := time.Now().UTC().Add(-c.DroppedTxGracePeriod - time.Hour)
	r := &recorder{}
	deps := oldUnconfirmedDeps(r,
		unconfirmed(testTxHash, bt),
		[]c.TxByHash{{
			Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
			FBBlockHash: testBlockHash, FBContainsTx: true, FBDataAvailable: true,
		}})

	require.NoError(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps))
	require.Len(t, r.finalized, 1)
	assert.True(t, r.finalized[0].DBBlockTime.IsZero())
}

func TestMonitorOldUnconfirmedPropagatesErrors(t *testing.T) {
	failing := c.TxByHash{
		Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
		FBBlockHash: testBlockHash, FBContainsTx: false, FBDataAvailable: true,
	}
	confirming := failing
	confirming.FBContainsTx = true

	tests := []struct {
		name   string
		tx     c.TxByHash
		break_ func(*buckDeps)
	}{
		{
			name: "candidate lookup fails",
			tx:   failing,
			break_: func(d *buckDeps) {
				d.getOldUnconfirmed = func(*sqlx.DB) ([]c.UnconfirmedTx, error) { return nil, errBoom }
			},
		},
		{
			name: "finalized block number lookup fails",
			tx:   failing,
			break_: func(d *buckDeps) {
				d.mostRecentBlockNumber = func(c.Context) (uint64, error) { return 0, errBoom }
			},
		},
		{
			name: "transaction lookup fails",
			tx:   failing,
			break_: func(d *buckDeps) {
				d.getTxsByHash = func(c.Context, []c.Hashable) ([]c.TxByHash, error) { return nil, errBoom }
			},
		},
		{
			name:   "failing the donation fails",
			tx:     failing,
			break_: func(d *buckDeps) { d.failTx = func(c.Context, c.TxByHash) error { return errBoom } },
		},
		{
			name:   "confirming the donation fails",
			tx:     confirming,
			break_: func(d *buckDeps) { d.finalizeTx = func(c.Context, c.TxByHash) error { return errBoom } },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &recorder{}
			deps := oldUnconfirmedDeps(r,
				unconfirmed(testTxHash, time.Now().UTC()), []c.TxByHash{tc.tx})
			tc.break_(&deps)

			assert.ErrorIs(t, monitorOldUnconfirmedWith(buckCtx(c.OldUnconfirmed), deps), errBoom)
		})
	}
}

// A canceled context stops the verdict loop between transactions and is not a
// failure.
func TestMonitorOldUnconfirmedStopsOnContextCancelation(t *testing.T) {
	failing := c.TxByHash{
		Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
		FBBlockHash: testBlockHash, FBContainsTx: false, FBDataAvailable: true,
	}
	r := &recorder{}
	deps := oldUnconfirmedDeps(r,
		unconfirmed(testTxHash, time.Now().UTC()), []c.TxByHash{failing, failing, failing})

	ctx, cancel := context.WithCancel(buckCtx(c.OldUnconfirmed))
	defer cancel()
	deps.failTx = func(_ c.Context, tx c.TxByHash) error {
		r.failed = append(r.failed, tx)
		cancel()
		return nil
	}

	assert.NoError(t, monitorOldUnconfirmedWith(ctx, deps))
	assert.Len(t, r.failed, 1, "the loop must not continue past cancelation")
}

func TestCheckFlags(t *testing.T) {
	tests := []struct {
		name    string
		modes   map[string]bool
		want    int
		wantErr bool
	}{
		{name: "no mode", modes: map[string]bool{"--monitor-latest": false, "--monitor-final": false}, want: 0},
		{name: "one mode", modes: map[string]bool{"--monitor-latest": true, "--monitor-final": false}, want: 1},
		{
			name:    "two modes",
			modes:   map[string]bool{"--monitor-latest": true, "--monitor-final": true},
			want:    2,
			wantErr: true,
		},
		{
			name: "three modes",
			modes: map[string]bool{
				"--monitor-latest": true, "--monitor-final": true, "--monitor-old-unconfirmed": true,
			},
			want:    3,
			wantErr: true,
		},
		{name: "empty map", modes: map[string]bool{}, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkFlags(tc.modes)
			assert.Equal(t, tc.want, got)
			if tc.wantErr {
				assert.ErrorContains(t, err, "please pick only *one* of these")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// liveWiring names the function every buckDeps field must be wired to. The
// wiring is checked by identity rather than by nil-ness because failTx and
// finalizeTx are the only pair in the struct that share a signature, and hence
// the only pair that can be transposed without a compile error: transposed,
// monitorOldUnconfirmed confirms every dead donation and fails every good one,
// and FailTx recomputes the campaign totals off the wrong rows.
func liveWiring() map[string]any {
	return map[string]any{
		"mostRecentBlockNumber": eth.MostRecentBlockNumber,
		"getLastBlock":          db.GetLastBlock,
		"setLastBlock":          db.SetLastBlock,
		"getTransactions":       eth.GetTransactions,
		"getETHPrice":           c.GetETHPrice,
		"requestPrice":          db.RequestPrice,
		"persistTxs":            db.PersistTxs,
		"getOldUnconfirmed":     db.GetOldUnconfirmed,
		"recordAbsence":         db.RecordAbsence,
		"failTx":                db.FailTx,
		"finalizeTx":            db.FinalizeTx,
	}
}

// getTxsByHash binds the type parameter and the fetcher of eth.GetData, so it
// is a closure with no stable code pointer to compare against. It is also the
// only field of its type in buckDeps and therefore cannot be transposed with
// another one -- non-nil is both all that can be checked and all that matters.
const closureWiredField = "getTxsByHash"

// funcName resolves a function value's code pointer to its fully qualified
// name; comparing names rather than raw pointers makes a mis-wire readable in
// the failure output.
func funcName(p uintptr) string {
	if f := runtime.FuncForPC(p); f != nil {
		return f.Name()
	}
	return fmt.Sprintf("unresolvable func at %#x", p)
}

// Every dependency must be wired, and wired to the right function: a field
// left nil panics the scheduled job on its first run and a field pointing at
// the wrong function corrupts donations silently. The loop walks buckDeps
// reflectively so that a field added without an assertion fails the test
// instead of going unchecked.
func TestLiveDepsAreWiredToTheRightFunctions(t *testing.T) {
	want := liveWiring()
	v := reflect.ValueOf(liveDeps())
	require.NotZero(t, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		field := v.Field(i)
		require.False(t, field.IsNil(), "buckDeps.%s is not wired", name)
		if name == closureWiredField {
			continue
		}
		fn, ok := want[name]
		require.True(t, ok, "buckDeps.%s has no wiring assertion, add one to liveWiring()", name)
		assert.Equal(t, funcName(reflect.ValueOf(fn).Pointer()), funcName(field.Pointer()),
			"buckDeps.%s is wired to the wrong function", name)
		delete(want, name)
	}
	assert.Empty(t, want, "liveWiring() names field(s) buckDeps does not have")
}

// withLiveDeps substitutes the production wiring for the duration of the test.
func withLiveDeps(t *testing.T, deps buckDeps) {
	t.Helper()
	orig := liveDeps
	t.Cleanup(func() { liveDeps = orig })
	liveDeps = func() buckDeps { return deps }
}

// mustNotPanic runs fn and reports the nil dereference a mis-wired buckDeps
// causes as a test failure rather than a crashed test binary -- that panic is
// exactly the production failure mode guarded against here.
func mustNotPanic(t *testing.T, fn func() error) error {
	t.Helper()
	var err error
	require.NotPanics(t, func() { err = fn() })
	return err
}

// The wrappers are what gocron and the --single-block path actually call, so
// each of them has to hand the *production* wiring to its loop. A wrapper
// passing a zero or partly populated buckDeps compiles happily and then nil
// panics on the first scheduled run: the panic handler logs it once a minute
// and no block is crawled until somebody reads the logs.
func TestWrappersHandLiveDepsToTheLoop(t *testing.T) {
	t.Run("monitorETH", func(t *testing.T) {
		r := &recorder{}
		deps := stubDeps(r)
		deps.mostRecentBlockNumber = func(c.Context) (uint64, error) {
			r.mrbnCalls++
			return 4711, nil
		}
		deps.getLastBlock = func(*sqlx.DB, string, string) (uint64, error) { return 4710, nil }
		withLiveDeps(t, deps)

		require.NoError(t, mustNotPanic(t, func() error { return monitorETH(buckCtx(c.Latest)) }))
		assert.Equal(t, 1, r.mrbnCalls)
		assert.Equal(t, []uint64{4711}, r.fetched)
	})

	t.Run("processETH", func(t *testing.T) {
		r := &recorder{}
		withLiveDeps(t, stubDeps(r))

		require.NoError(t, mustNotPanic(t, func() error { return processETH(c.Context{}, 4711) }))
		assert.Equal(t, []uint64{4711}, r.fetched)
		assert.Equal(t, []uint64{4711}, r.cursor)
	})

	t.Run("monitorOldUnconfirmed", func(t *testing.T) {
		r := &recorder{}
		withLiveDeps(t, oldUnconfirmedDeps(r,
			unconfirmed(testTxHash, time.Now().UTC()),
			[]c.TxByHash{{
				Hash: testTxHash, BlockNumber: testMFBN - 1, BlockHash: testBlockHash,
				FBBlockHash: testBlockHash, FBContainsTx: true, FBDataAvailable: true,
			}}))

		require.NoError(t, mustNotPanic(t, func() error {
			return monitorOldUnconfirmed(buckCtx(c.OldUnconfirmed))
		}))
		assert.Equal(t, 1, r.byHashCalls)
		assert.Len(t, r.finalized, 1)
	})
}

// The exported entry points are thin wrappers around the *With variants; they
// must hand the buck context through unchanged.
func TestEntryPointsRejectAnInvalidBuckContext(t *testing.T) {
	assert.ErrorContains(t, monitorETH(context.Background()), "invalid buck context")
	assert.ErrorContains(t, monitorOldUnconfirmed(context.Background()),
		"buck/old-unconfirmed invalid buck context")
}
