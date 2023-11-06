package ethereum

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	c "github.com/verity-team/dws/internal/common"
)

type TxByHashBody struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  *c.TxByHash `json:"result"`
}

func addFBData(ctxt c.Context, txs []c.TxByHash) error {
	blocks := make(map[uint64]bool)
	for _, tx := range txs {
		if _, exists := blocks[tx.BlockNumber]; !exists {
			blocks[tx.BlockNumber] = true
		}
	}
	// get the block times for the blocks that finalize the given transactions
	fbs := make(map[uint64]c.FinalizedBlock)
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

type TXBHFetcher struct{}

func (txbh TXBHFetcher) Fetch(ctxt c.Context, hs []c.Hashable) ([]c.TxByHash, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	rd := make([]map[string]interface{}, len(hs))
	for idx, h := range hs {
		rq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getTransactionByHash",
			"params":  []interface{}{h.GetHash()},
			"id":      idx + 1,
		}
		rd[idx] = rq
	}
	requestBytes, err := json.Marshal(rd)
	if err != nil {
		return nil, err
	}

	params := c.HTTPParams{
		URL:         ctxt.ETHRPCURL,
		RequestBody: requestBytes,
	}
	body, err := c.HTTPPost(params)
	if err != nil {
		return nil, err
	}
	result, err := parseTxByHash(body)
	if err != nil {
		return nil, err
	}
	writeTxsToFile(ctxt, time.Now().UTC().Unix(), body)

	err = addFBData(ctxt, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseTxByHash(body []byte) ([]c.TxByHash, error) {
	var resp []TxByHashBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res []c.TxByHash = make([]c.TxByHash, len(resp))
	for i, d := range resp {
		if d.Result != nil {
			res[i] = *d.Result
		}
	}
	return res, nil
}
