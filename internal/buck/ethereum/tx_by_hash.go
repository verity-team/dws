package ethereum

import (
	"fmt"
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
	var txs []common.TxByHash
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
		txs = append(txs, br...)
	}
	err := addFBData(ctxt, txs)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func addFBData(ctxt common.Context, txs []common.TxByHash) error {
	blocks := make(map[uint64]bool)
	for _, tx := range txs {
		if _, exists := blocks[tx.BlockNumber]; !exists {
			blocks[tx.BlockNumber] = true
		}
	}
	// get the block times for the blocks that finalize the given transactions
	fbs := make(map[uint64]common.FinalizedBlock)
	fbHashes := make(map[uint64]map[string]bool)
	for bn := range blocks {
		fb, err := GetFinalizedBlock(ctxt, bn)
		if err != nil {
			return err
		}
		fbs[fb.Number] = *fb
		fbHashes[fb.Number] = fb.TXMap()
	}
	// now set the block hash/time for the finalized transactions
	for i := 0; i < len(txs); i++ {
		if fb, exists := fbs[txs[i].BlockNumber]; exists {
			txs[i].FBBlockTime = fb.Timestamp
			txs[i].FBBlockHash = fb.Hash
			// does the finalized block actually contain the tx?
			_, txs[i].FBContainsTx = fbHashes[fb.Number][txs[i].Hash]
		} else {
			err := fmt.Errorf("internal error: no finalized block for tx '%s' and block number %d", txs[i].Hash, txs[i].BlockNumber)
			log.Error(err)
			return err
		}
	}
	return nil
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
