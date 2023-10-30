package ethereum

import (
	"time"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type TxByHashBody struct {
	Jsonrpc string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  *common.TxByHash `json:"result"`
}

func GetTxsByHash(ctxt common.Context, hashes []string) ([]common.TxByHash, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	var res []common.TxByHash
	batchSize := 127

	for i := 0; i < len(hashes); i += batchSize {
		end := i + batchSize
		if end > len(hashes) {
			end = len(hashes)
		}

		batch := hashes[i:end]

		// Process the batch
		br, err := doGetTxsByHash(ctxt, batch)
		if err != nil {
			return nil, err
		}
		res = append(res, br...)
	}
	return res, nil
}

func doGetTxsByHash(ctxt common.Context, hashes []string) ([]common.TxByHash, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	rd := make([]map[string]interface{}, len(hashes))
	for idx, hash := range hashes {
		rq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getTransactionByHash",
			"params":  []interface{}{hash},
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
	err = writeTxsToFile(ctxt, time.Now().Unix(), body)
	if err != nil {
		log.Warnf("failed to persist %d txs starting with hash '%s' to disk", len(hashes), hashes[0])
	}

	return result, nil
}

func parseTxByHash(body []byte) ([]common.TxByHash, error) {
	var resp []TxByHashBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res []common.TxByHash = make([]common.TxByHash, len(resp))
	for i, d := range resp {
		if d.Result != nil {
			res[i] = *d.Result
		}
	}
	return res, nil
}
