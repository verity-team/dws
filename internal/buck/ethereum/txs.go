package ethereum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/buck/db"
	"github.com/verity-team/dws/internal/common"
)

type EthGetBlockByNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func GetTransactions(ctxt common.Context, blockNumber uint64) ([]common.Transaction, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{blockNumber, true},
		ID:      1,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	resp, err := client.Post(ctxt.ETHRPCURL, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		err := fmt.Errorf("failed to request block #%d, %v", blockNumber, err)
		log.Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to fetch block #%d, status code: %d, %v", blockNumber, resp.StatusCode, err)
		log.Error(err)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("failed to read block #%d, %v", blockNumber, err)
		log.Error(err)
		return nil, err
	}

	block, err := parseBlock(body)
	if err != nil {
		err := fmt.Errorf("failed to parse block #%d, %v", blockNumber, err)
		log.Error(err)
		return nil, err
	}
	log.Infof("block %d: %d transactions in total", blockNumber, len(block.Transactions))

	result, err := filterTransactions(ctxt, *block)
	if err != nil {
		return nil, err
	}
	log.Infof("block %d: %d *filtered* transactions", blockNumber, len(result))

	err = markFailedTxs(ctxt, result)
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
				err = fmt.Errorf("failed to process ETH tx '%s'", tx.Hash)
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
			// check that this is a stable coin tx
			erc20, ok := ctxt.StableCoins[strings.ToLower(tx.To)]
			if ok {
				// yes, actual receiver and amount are encoded in the input string
				receiver, amount, err := parseInputData(tx.Input)
				if err != nil {
					err = fmt.Errorf("failed to process ERC-20 tx '%s'", tx.Hash)
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
			tx.Status = string(api.Unconfirmed)
			result = append(result, tx)
		}
	}
	return result, nil
}

func parseBlock(body []byte) (*common.Block, error) {
	type pblock struct {
		common.Block
		HexSeconds string `json:"timestamp"`
		HexNumber  string `json:"number"`
	}
	type Response struct {
		Block pblock `json:"result"`
	}

	var resp Response
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	seconds, err := common.HexStringToDecimal(resp.Block.HexSeconds)
	if err != nil {
		return nil, err
	}
	number, err := common.HexStringToDecimal(resp.Block.HexNumber)
	if err != nil {
		return nil, err
	}
	ts := time.Unix(seconds.IntPart(), 0)
	result := common.Block{
		Hash:         resp.Block.Hash,
		Number:       uint64(number.IntPart()),
		Transactions: resp.Block.Transactions,
		Timestamp:    ts.UTC(),
	}
	return &result, nil
}

func markFailedTxs(ctxt common.Context, txs []common.Transaction) error {
	// check whether any of the _filtered_ transactions have failed
	for _, tx := range txs {
		rcpt, err := GetTransactionReceipt(ctxt.ETHRPCURL, tx.Hash)
		if err != nil {
			err = fmt.Errorf("failed to get tx receipt for %s, %v", tx.Hash, err)
			log.Error(err)
			return err
		}
		if strings.ToLower(rcpt.Status) != "0x1" {
			tx.Status = string(api.Failed)
		}
	}
	return nil
}

func parseInputData(input string) (string, decimal.Decimal, error) {
	// example: "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a76400000"

	if len(input) != 138 {
		return "", decimal.Zero, fmt.Errorf("input has invalid length")
	}

	// Remove "0x" prefix if present
	input = strings.TrimPrefix(input, "0x")

	// Ensure the input starts with "0xa9059cbb"
	if !strings.HasPrefix(input, "a9059cbb") {
		return "", decimal.Zero, fmt.Errorf("input does not start with the expected function signature")
	}

	// Extract the receiving address (next 32 bytes) and strip leading zeroes
	receivingAddress := strings.TrimLeft(input[8:72], "0")

	// Ensure receivingAddress is 40 characters long
	receivingAddress = strings.Repeat("0", 40-len(receivingAddress)) + receivingAddress

	// Extract the amount (next 32 bytes) and convert it to a uint64
	amountHex := input[72:]
	amount, err := common.HexStringToDecimal(amountHex)
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("failed to convert amount to uint64")
	}

	return "0x" + receivingAddress, amount, nil
}
