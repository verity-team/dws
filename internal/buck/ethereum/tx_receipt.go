package ethereum

import (
	"encoding/json"

	"github.com/verity-team/dws/internal/common"
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
	requestData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{txHash},
		"id":      1,
	}
	requestBytes, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	params := common.HTTPParams{
		URL:         url,
		RequestBody: requestBytes,
	}
	body, err := common.HTTPPost(params)
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
