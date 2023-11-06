package ethereum

import (
	"time"

	"github.com/goccy/go-json"

	"github.com/verity-team/dws/internal/common"
)

type TxReceiptBody struct {
	Jsonrpc string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  common.TxReceipt `json:"result"`
}

func GetTxReceipts(ctxt common.Context, txs []common.Transaction) ([]common.TxReceipt, error) {
	if len(txs) == 0 {
		return nil, nil
	}
	var res []common.TxReceipt
	batchSize := 127

	for i := 0; i < len(txs); i += batchSize {
		end := i + batchSize
		if end > len(txs) {
			end = len(txs)
		}

		batch := txs[i:end]

		// Process the batch
		br, err := doGetTxReceipts(ctxt, batch)
		if err != nil {
			return nil, err
		}
		res = append(res, br...)
	}
	return res, nil
}

func doGetTxReceipts[H common.Hashable](ctxt common.Context, txs []H) ([]common.TxReceipt, error) {
	if len(txs) == 0 {
		return nil, nil
	}
	rd := make([]map[string]interface{}, len(txs))
	for idx, tx := range txs {
		rq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getTransactionReceipt",
			"params":  []interface{}{tx.GetHash()},
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
	result, err := parseTxReceipt(body)
	if err != nil {
		return nil, err
	}
	writeTxReceiptsToFile(ctxt, time.Now().UTC().Unix(), body)

	return result, nil
}

func parseTxReceipt(body []byte) ([]common.TxReceipt, error) {
	var resp []TxReceiptBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res = make([]common.TxReceipt, len(resp))
	for i, d := range resp {
		res[i] = d.Result
	}
	return res, nil
}
