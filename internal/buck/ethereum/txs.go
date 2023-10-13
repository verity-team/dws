package ethereum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/buck"
)

type EthGetBlockByNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type Transaction struct {
	Hash      string          `db:"tx_hash" json:"hash"`
	From      string          `db:"address" json:"from"`
	To        string          `db:"-" json:"to"`
	Value     string          `db:"amount" json:"value"`
	Gas       string          `db:"-" json:"gas"`
	Nonce     string          `db:"-" json:"nonce"`
	Input     string          `db:"-" json:"input"`
	Type      string          `db:"-" json:"type"`
	Status    string          `db:"status"`
	Asset     string          `db:"asset"`
	Price     string          `db:"price"`
	Tokens    decimal.Decimal `db:"tokens"`
	USDAmount decimal.Decimal `db:"usd_amount"`
}

func GetTransactions(ctxt buck.Context, blockNumber uint64) ([]Transaction, error) {
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonResult struct {
		Transactions []Transaction `json:"transactions"`
	}

	err = json.Unmarshal(body, &jsonResult)
	if err != nil {
		return nil, err
	}

	result := make([]Transaction, 0)
	for _, tx := range jsonResult.Transactions {
		if tx.Input == "0x0" {
			// plain ETH tx -- only return txs that send ETH to the receiving
			// address
			if strings.ToLower(tx.To) == ctxt.ReceivingAddr {
				tx.Status = string(api.Unconfirmed)
				tx.Asset = "eth"
				result = append(result, tx)
				continue
			}
		}
		// ERC-20 transfer?
		if strings.HasPrefix(tx.Input, "0xa9059cbb") {
			// check that this is a stable coin tx
			erc20, ok := ctxt.StableCoins[strings.ToLower(tx.To)]
			if ok {
				// yes, actual receiver and amount are encoded in the input string
				receiver, amount, err := parseInputData(tx.Input)
				if err != nil {
					log.Error(err)
					continue
					// TODO: log these failed/malformed stable coin transfers to the
					// database -- they need to be processed by a human
				}
				// is this a stable coin tx to the receiving address?
				if strings.ToLower(receiver) == ctxt.ReceivingAddr {
					tx.Status = string(api.Unconfirmed)
					tx.To = ctxt.ReceivingAddr
					tx.Value = amount.Shift(erc20.Scale).StringFixed(erc20.Scale)
					tx.Asset = erc20.Asset
					result = append(result, tx)
				}
			}
		}
	}

	return result, nil
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
	amount, err := strconv.ParseUint(amountHex, 16, 64)
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("failed to convert amount to uint64")
	}

	return "0x" + receivingAddress, decimal.NewFromInt(int64(amount)), nil
}
