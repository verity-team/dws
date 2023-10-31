package ethereum

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethc "github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/buck/db"
	"github.com/verity-team/dws/internal/common"
)

func GetTransactions(ctxt common.Context, blockNumber uint64) ([]common.Transaction, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", blockNumber), true},
		ID:      1,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	params := common.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := common.HTTPPost(params)
	if err != nil {
		return nil, err
	}

	log.Infof("fetched block %d", blockNumber)

	err = writeBlockToFile(ctxt, blockNumber, body, false)
	if err != nil {
		return nil, err
	}
	block, err := parseBlock(body)
	if err != nil {
		err = fmt.Errorf("failed to parse block #%d, %w", blockNumber, err)
		log.Error(err)
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

func filterTransactions(ctxt common.Context, b common.Block) ([]common.Transaction, error) {
	result := make([]common.Transaction, 0)
	for _, tx := range b.Transactions {
		var txBelongsToUs bool
		if tx.Input == "0x" {
			// plain ETH tx -- only return txs that send ETH to the receiving
			// address
			if strings.ToLower(tx.To) != ctxt.ReceivingAddr {
				continue
			}
			tx.Asset = "eth"
			amount, err := common.HexStringToDecimal(tx.Value)
			if err != nil {
				err = fmt.Errorf("failed to process ETH tx '%s', %w", tx.Hash, err)
				log.Error(err)
				err = db.PersistFailedTx(ctxt.DB, b, tx)
				if err != nil {
					log.Error(err)
				}
				continue
			}
			tx.Value = amount.Shift(-18).StringFixed(8)
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
					log.Error(err)
					err = db.PersistFailedTx(ctxt.DB, b, tx)
					if err != nil {
						log.Error(err)
					}
					continue
				}
				// is this a stable coin tx to the receiving address?
				if strings.ToLower(receiver) != ctxt.ReceivingAddr {
					continue
				}
				tx.To = ctxt.ReceivingAddr
				tx.Value = amount.Shift(-erc20.Scale).StringFixed(6)
				tx.Asset = erc20.Asset
				txBelongsToUs = true
			}
		}
		if txBelongsToUs {
			tx.BlockNumber = b.Number
			tx.BlockTime = b.Timestamp
			if ctxt.CrawlerType == common.Finalized {
				tx.Status = string(api.Confirmed)
			} else {
				tx.Status = string(api.Unconfirmed)
			}
			result = append(result, tx)
		}
	}
	return result, nil
}

func parseBlock(body []byte) (*common.Block, error) {
	type Response struct {
		Block common.Block `json:"result"`
	}

	var resp Response
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	b := resp.Block
	log.Infof("parsed block %d, %s -- %d transactions", b.Number, b.Hash, len(b.Transactions))
	return &b, nil
}

func markFailedTxs(ctxt common.Context, bn uint64, txs []common.Transaction) error {
	// check whether any of the _filtered_ transactions have failed
	rcpts, err := GetTxReceipts(ctxt, txs)
	if err != nil {
		err = fmt.Errorf("failed to get tx receipts for block %d, %w", bn, err)
		log.Error(err)
		return err
	}
	for i, tx := range txs {
		rcpt := rcpts[i]
		if tx.Hash != rcpt.TransactionHash {
			err = fmt.Errorf("block: %d -- receipt hash ('%s') does not match tx hash ('%s')", bn, rcpt.TransactionHash, tx.Hash)
			return err
		}
		if strings.ToLower(rcpt.Status) != "0x1" {
			tx.Status = string(api.Failed)
		}
	}
	return nil
}

func parseInputData(abis map[string]abi.ABI, erc20, input string) (string, decimal.Decimal, error) {
	// look up ABI
	abi, ok := abis[erc20]
	if !ok {
		return "", decimal.Zero, fmt.Errorf("No ABI for ERC-20 '%s'", erc20)
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

	value := decimal.NewFromBigInt(args["_value"].(*big.Int), 0)
	to := args["_to"].(ethc.Address).String()
	return to, value, nil
}
