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

	"golang.org/x/exp/slices"
)

type EthGetBlockByNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type Transaction struct {
	Hash  string `json:"hash"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Gas   string `json:"gas"`
	Nonce string `json:"nonce"`
	Input string `json:"input"`
	Type  string `json:"type"`
}

func GetTransactions(apiURL string, blockNumber string, contracts []string) ([]Transaction, error) {
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
		Timeout: 5 * time.Second,
	}

	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Transactions []Transaction `json:"transactions"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	// Filter transactions based on the input condition
	filteredTransactions := make([]Transaction, 0)
	for _, tx := range result.Transactions {
		if tx.Input == "0x0" {
			filteredTransactions = append(filteredTransactions, tx)
		}
		if strings.HasPrefix(tx.Input, "0xa9059cbb") {
			// check that this is either usdt or usdc
			if slices.Contains(contracts, strings.ToLower(tx.To)) {
				filteredTransactions = append(filteredTransactions, tx)
			}
		}
	}

	return filteredTransactions, nil
}

func ParseInputData(input string) (string, uint64, error) {
	// example: "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a76400000"

	if len(input) != 138 {
		return "", 0, fmt.Errorf("input has invalid length")
	}

	// Remove "0x" prefix if present
	input = strings.TrimPrefix(input, "0x")

	// Ensure the input starts with "0xa9059cbb"
	if !strings.HasPrefix(input, "a9059cbb") {
		return "", 0, fmt.Errorf("Input does not start with the expected function signature")
	}

	// Extract the receiving address (next 32 bytes) and strip leading zeroes
	receivingAddress := strings.TrimLeft(input[8:72], "0")

	// Ensure receivingAddress is 40 characters long
	receivingAddress = strings.Repeat("0", 40-len(receivingAddress)) + receivingAddress

	// Extract the amount (next 32 bytes) and convert it to a uint64
	amountHex := input[72:]
	amount, err := strconv.ParseUint(amountHex, 16, 64)
	if err != nil {
		return "", 0, fmt.Errorf("Failed to convert amount to uint64")
	}

	return "0x" + receivingAddress, amount, nil
}
