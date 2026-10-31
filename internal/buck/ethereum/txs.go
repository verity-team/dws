package ethereum

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethc "github.com/ethereum/go-ethereum/common"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/buck/db"
	c "github.com/verity-team/dws/internal/common"
)

const (
	// DonationAmountDecimals is the number of decimal places a donated amount
	// is rendered with before it is written to `donation.amount`, which is
	// NUMERIC(20,10) -- see deployments/db/01-schema.sql. Rendering at
	// exactly the scale of the column is what keeps the stored value from
	// depending on the scale of the token: `Shift` is already driven by the
	// configured erc-20 scale, so pinning the rendering at 6 was only correct
	// for as long as every configured coin happened to have 6 decimals.
	// c.GetContext accepts any scale greater than zero, so an 18 decimal
	// stable coin is a supported configuration -- and it used to be credited
	// up to 1.1e-7 too much per donation, because StringFixed rounds half
	// away from zero rather than truncating.
	DonationAmountDecimals = 10
	// ETHAmountDecimals is the scale ethereum donations are rendered with.
	//
	// It is deliberately *not* DonationAmountDecimals. Raising it is not
	// value preserving -- it changes the amount recorded for every donation
	// whose wei value is not a multiple of 1e10 -- and it would widen the set
	// of dust transfers that record a non-zero amount, each of which is
	// issued a whole token by the Ceil in calcTokens. Changing it belongs
	// with a minimum donation amount, which is a business decision.
	ETHAmountDecimals = 8
	// WeiDecimals is the scale of the ethereum base unit.
	WeiDecimals = 18
)

// recordedAmount renders a transferred amount exactly the way it will be
// stored and reports whether anything at all is left of it at that scale.
//
// The second return value is what keeps a transfer that records nothing out
// of the donation table. `value: "0x0"` is the obvious case, but rejecting
// only that buys nothing: one wei costs the same 21000 gas and renders as
// 0.00000000 just the same. Every such row would carry amount 0, usd_amount 0
// and 0 tokens -- it records no donation, while the per-address aggregates
// and the full-table aggregate updateDonationStats runs under the
// donation_stats lock on every finalized block pay for it forever.
//
// Note this is not a minimum donation amount: the only transfers turned away
// are the ones that would have been stored as zero anyway.
func recordedAmount(raw decimal.Decimal, scale, places int32) (string, bool) {
	value := roundedAmount(raw, scale, places)
	return value.StringFixed(places), value.IsPositive()
}

// roundedAmount is the value recordedAmount renders. StringFixed rounds to
// `places` internally; rounding here first makes the value that is tested
// (recordedAmount, amountFits) and the value that is stored the same one.
func roundedAmount(raw decimal.Decimal, scale, places int32) decimal.Decimal {
	return raw.Shift(-scale).Round(places)
}

// amountFits reports whether a transferred amount can be written to the
// donation row at all. usd is set for the stable coins: their amount *is* the
// usd_amount, so it has to fit that (narrower) column as well. The usd_amount
// of an ethereum donation depends on the ETH price and is checked when it is
// computed, see calcTokens.
//
// Nothing else bounds the amount, and a transfer that does not fit is not a
// matter of rounding: postgres refuses it with `numeric field overflow`, which
// rolls back the whole block. The block is retried on every run, by the latest
// crawler first and then by the finalized one, so a single such transfer
// wedges both crawlers for good -- and skipping the block drops every
// legitimate donation in it. For an ERC-20 it does not even take the money: a
// `transfer(receivingAddr, 1e17)` from an empty account reverts, but it is
// mined, its calldata is what is parsed here and the reverted transaction is
// persisted like any other (as 'failed'). Such a transfer is recorded in
// failed_tx instead, like a transfer whose calldata cannot be parsed: no
// genuine donation comes anywhere near 1e10 USD.
func amountFits(raw decimal.Decimal, scale, places int32, usd bool) bool {
	value := roundedAmount(raw, scale, places)
	if !c.DonationAmountColumn.Fits(value) {
		return false
	}
	return !usd || c.DonationUSDAmountColumn.Fits(value)
}

// persistFailedTx records a transaction to the receiving address that cannot
// be recorded as a donation in failed_tx. cause says why, it is logged.
//
// failed txs must be persisted -- otherwise we are losing them altogether: the
// transaction ends up in neither `donation` nor `failed_tx` while the block is
// advanced past it. A failed write is hence returned and fails the block.
func persistFailedTx(ctxt c.Context, b c.Block, tx c.Transaction, cause error) error {
	log.Error(cause)
	err := db.PersistFailedTx(ctxt.DB, b, tx)
	if err != nil {
		err = fmt.Errorf("failed to persist failed tx '%s', %w", tx.Hash, err)
		log.Error(err)
		return err
	}
	return nil
}

func GetTransactions(ctxt c.Context, blockNumber uint64) ([]c.Transaction, error) {
	block, err := GetBlock(ctxt, blockNumber)
	if err != nil {
		return nil, err
	}
	log.Infof("block %d (%v): %d transactions in total", blockNumber, block.Timestamp.Format(time.RFC3339), len(block.Transactions))

	result, err := filterTransactions(ctxt, *block)
	if err != nil {
		return nil, err
	}

	err = markFailedTxs(ctxt, block.Number, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func filterTransactions(ctxt c.Context, b c.Block) ([]c.Transaction, error) {
	var result []c.Transaction
	for _, tx := range b.Transactions {
		var txBelongsToUs bool
		if tx.Input == "0x" {
			// plain ETH tx -- only return txs that send ETH to the receiving
			// address
			if strings.ToLower(tx.To) != ctxt.ReceivingAddr {
				continue
			}
			tx.Asset = "eth"
			amount, err := c.HexStringToDecimal(tx.Value)
			if err != nil {
				err = fmt.Errorf("failed to process ETH tx '%s', %w", tx.Hash, err)
				if err = persistFailedTx(ctxt, b, tx, err); err != nil {
					return nil, err
				}
				continue
			}
			value, ok := recordedAmount(amount, WeiDecimals, ETHAmountDecimals)
			if !ok {
				log.Warnf("skipping ETH tx ('%s') that transfers no recordable amount (%s wei)", tx.Hash, amount.String())
				continue
			}
			if !amountFits(amount, WeiDecimals, ETHAmountDecimals, false) {
				err = fmt.Errorf(
					"ETH tx '%s' transfers an amount (%s wei) that does not fit donation.amount %s",
					tx.Hash, amount.String(), c.DonationAmountColumn)
				if err = persistFailedTx(ctxt, b, tx, err); err != nil {
					return nil, err
				}
				continue
			}
			tx.Value = value
			txBelongsToUs = true
		}
		// ERC-20 transfer?
		if strings.HasPrefix(tx.Input, "0xa9059cbb") {
			if tx.Input == "0xa9059cbb" {
				// tx with malformed input, ignore it
				log.Warnf("malformed tx ('%s') with input '%s'", tx.Hash, tx.Input)
				continue
			}
			// check that this is a stable coin tx
			erc20, ok := ctxt.StableCoins[strings.ToLower(tx.To)]
			if ok {
				// yes, actual receiver and amount are encoded in the input string
				receiver, amount, err := parseInputData(ctxt.ABI, erc20.Asset, tx.Input)
				if err != nil {
					err = fmt.Errorf("failed to process ERC-20 tx '%s', %w", tx.Hash, err)
					if err = persistFailedTx(ctxt, b, tx, err); err != nil {
						return nil, err
					}
					continue
				}
				// is this a stable coin tx to the receiving address?
				if strings.ToLower(receiver) != ctxt.ReceivingAddr {
					continue
				}
				value, vok := recordedAmount(amount, erc20.Scale, DonationAmountDecimals)
				if !vok {
					log.Warnf("skipping %s tx ('%s') that transfers no recordable amount (%s base units)", erc20.Asset, tx.Hash, amount.String())
					continue
				}
				if !amountFits(amount, erc20.Scale, DonationAmountDecimals, true) {
					err = fmt.Errorf(
						"%s tx '%s' transfers an amount (%s base units) that does not fit donation.amount %s / donation.usd_amount %s",
						erc20.Asset, tx.Hash, amount.String(), c.DonationAmountColumn, c.DonationUSDAmountColumn)
					if err = persistFailedTx(ctxt, b, tx, err); err != nil {
						return nil, err
					}
					continue
				}
				tx.To = ctxt.ReceivingAddr
				tx.Value = value
				tx.Asset = erc20.Asset
				txBelongsToUs = true
			}
		}
		if txBelongsToUs {
			// the repo convention is lower case and every read path
			// lowercases: normalize here, at the point of acceptance, so a
			// provider that emits checksummed values cannot write a donation
			// row that the exact-case lookups elsewhere will never find
			// again (a mixed case tx_hash makes the old-unconfirmed crawler
			// fail a perfectly valid donation).
			tx.Hash = c.NormalizeHash(tx.Hash)
			tx.From = strings.ToLower(strings.TrimSpace(tx.From))
			tx.BlockNumber = b.Number
			tx.BlockTime = b.Timestamp
			if ctxt.CrawlerType == c.Finalized {
				tx.Status = string(api.Confirmed)
			} else {
				tx.Status = string(api.Unconfirmed)
			}
			result = append(result, tx)
		}
	}
	return result, nil
}

func markFailedTxs(ctxt c.Context, bn uint64, txs []c.Transaction) error {
	// check whether any of the _filtered_ transactions have failed
	rcpts, err := GetTxReceipts(ctxt, c.ToHashable(txs))
	if err != nil {
		err = fmt.Errorf("failed to get tx receipts for block %d, %w", bn, err)
		log.Error(err)
		return err
	}
	return applyTxReceipts(bn, txs, rcpts)
}

// applyTxReceipts marks the transactions whose receipt does not report success
// as failed.
//
// Receipts are correlated with transactions by hash: JSON-RPC 2.0 allows the
// responses in a batch to come back in any order and individual sub-requests
// may yield a null result (tx not indexed yet) or an error object. Relying on
// the position of a receipt in the response would thus be incorrect (and could
// panic for a truncated response).
func applyTxReceipts(bn uint64, txs []c.Transaction, rcpts []c.TxReceipt) error {
	// providers may return either casing, the repo convention is lower case
	byHash := make(map[string]c.TxReceipt, len(rcpts))
	for _, rcpt := range rcpts {
		hash := c.NormalizeHash(rcpt.TransactionHash)
		if hash == "" {
			// null result or error object in the batch response; the
			// transaction(s) affected are reported below
			continue
		}
		byHash[hash] = rcpt
	}
	for i := range txs {
		rcpt, ok := byHash[c.NormalizeHash(txs[i].Hash)]
		if !ok {
			// the message points at the runbook: this error repeats on every
			// run for as long as the provider refuses the receipt, and the
			// block -- and with it the crawler -- does not advance
			err := fmt.Errorf(
				"block: %d -- no tx receipt for tx '%s' (%d usable receipt(s) for %d tx(s)); if this persists see 'a crawler is stuck on a block' in the README",
				bn, txs[i].Hash, len(byHash), len(txs))
			log.Error(err)
			return err
		}
		if strings.ToLower(rcpt.Status) != "0x1" {
			txs[i].Status = string(api.Failed)
		}
	}
	return nil
}

func parseInputData(abis map[string]abi.ABI, erc20, input string) (string, decimal.Decimal, error) {
	// look up ABI
	abi, ok := abis[erc20]
	if !ok {
		return "", decimal.Zero, fmt.Errorf("no ABI for ERC-20 '%s'", erc20)
	}
	di, err := hex.DecodeString(input[2:])
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("failed to decode input, '%s'", input)
	}
	signature, data := di[:4], di[4:]

	method, err := abi.MethodById(signature)
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("failed to find method for input, '%s'", input)
	}

	var args = make(map[string]interface{})
	err = method.Inputs.UnpackIntoMap(args, data)
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("failed to unpack input, '%s'", input)
	}

	rawValue, ok := args["_value"]
	if !ok {
		return "", decimal.Zero, fmt.Errorf("missing '_value' in input args, '%s'", input)
	}
	bigValue, ok := rawValue.(*big.Int)
	if !ok {
		return "", decimal.Zero, fmt.Errorf("unexpected type for '_value' in input, '%s'", input)
	}
	rawTo, ok := args["_to"]
	if !ok {
		return "", decimal.Zero, fmt.Errorf("missing '_to' in input args, '%s'", input)
	}
	toAddr, ok := rawTo.(ethc.Address)
	if !ok {
		return "", decimal.Zero, fmt.Errorf("unexpected type for '_to' in input, '%s'", input)
	}
	value := decimal.NewFromBigInt(bigValue, 0)
	to := toAddr.String()
	return to, value, nil
}
