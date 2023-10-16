package ethereum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TxReceiptBody struct {
	Jsonrpc string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  TxReceipt `json:"result"`
}

type TxReceipt struct {
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	EffectiveGasPrice string `json:"effectiveGasPrice"`
	From              string `json:"from"`
	GasUsed           string `json:"gasUsed"`
	Status            string `json:"status"`
	To                string `json:"to"`
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	Type              string `json:"type"`
}

func GetTransactionReceipt(url string, txHash string) (*TxReceipt, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	requestData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{txHash},
		"id":      1,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
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
	result, err := parseTxReceipt(body)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func parseTxReceipt(body []byte) (*TxReceipt, error) {
	var resp TxReceiptBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}