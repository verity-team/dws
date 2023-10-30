package ethereum

import (
	"fmt"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type TxByHashBody struct {
	Jsonrpc string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  *TxByHash `json:"result"`
}
type TxByHash struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      uint64 `json:"blockNumber"`
	From             string `json:"from"`
	Hash             string `json:"hash"`
	To               string `json:"to"`
	TransactionIndex uint64 `json:"transactionIndex"`
}

func (t *TxByHash) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	type tx struct {
		BlockHash        string `json:"blockHash"`
		BlockNumber      string `json:"blockNumber"`
		From             string `json:"from"`
		Hash             string `json:"hash"`
		To               string `json:"to"`
		TransactionIndex string `json:"transactionIndex"`
	}

	var pd tx
	if err := json.Unmarshal(data, &pd); err != nil {
		return err
	}

	bn, err := common.HexStringToDecimal(pd.BlockNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %w", err)
		return err
	}
	t.BlockNumber = uint64(bn.IntPart())

	tidx, err := common.HexStringToDecimal(pd.TransactionIndex)
	if err != nil {
		err = fmt.Errorf("failed to convert transaction index, %w", err)
		return err
	}
	t.TransactionIndex = uint64(tidx.IntPart())

	t.BlockHash = pd.BlockHash
	t.From = pd.From
	t.Hash = pd.Hash
	t.To = pd.To

	return nil
}

func GetTxsByHash(ctxt common.Context, txs []common.Transaction) ([]TxByHash, error) {
	if len(txs) == 0 {
		return nil, nil
	}
	var res []TxByHash
	batchSize := 127

	for i := 0; i < len(txs); i += batchSize {
		end := i + batchSize
		if end > len(txs) {
			end = len(txs)
		}

		batch := txs[i:end]

		// Process the batch
		br, err := doGetTxsByHash(ctxt, batch)
		if err != nil {
			return nil, err
		}
		res = append(res, br...)
	}
	return res, nil
}

func doGetTxsByHash(ctxt common.Context, txs []common.Transaction) ([]TxByHash, error) {
	if len(txs) == 0 {
		return nil, nil
	}
	rd := make([]map[string]interface{}, len(txs))
	for idx, tx := range txs {
		rq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getTransactionByHash",
			"params":  []interface{}{tx.Hash},
			"id":      idx + 1,
		}
		rd[idx] = rq
	}
	requestBytes, err := json.Marshal(rd)
	if err != nil {
		return nil, err
	}

	params := common.HTTPParams{
		URL:         ctxt.ETHRPCURL,
		RequestBody: requestBytes,
	}
	body, err := common.HTTPPost(params)
	if err != nil {
		return nil, err
	}
	result, err := parseTxByHash(body)
	if err != nil {
		return nil, err
	}
	err = writeTxsToFile(ctxt, txs[0].BlockNumber, body)
	if err != nil {
		log.Warnf("failed to persist tx receipts for block %d to disk", txs[0].BlockNumber)
	}

	return result, nil
}

func parseTxByHash(body []byte) ([]TxByHash, error) {
	var resp []TxByHashBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res []TxByHash = make([]TxByHash, len(resp))
	for i, d := range resp {
		if d.Result != nil {
			res[i] = *d.Result
		}
	}
	return res, nil
}
