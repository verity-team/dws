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

// addFBData annotates the given transactions with the data of the blocks that
// finalize them.
//
// Failures are contained per transaction: a transaction that is still pending
// has no block to fetch and a transaction whose finalized block cannot be
// fetched is left with `FBDataAvailable == false`. Either way the remaining
// transactions are annotated and judged as usual -- a single bad element must
// not disable the whole batch.
func addFBData(ctxt c.Context, txs []c.TxByHash) {
	// get the block times for the blocks that finalize the given transactions
	fbs := make(map[uint64]c.FinalizedBlock)
	fbHashes := make(map[uint64]map[string]bool)
	fetched := make(map[uint64]bool)
	for _, tx := range txs {
		if tx.Pending {
			// still in the mempool, there is no block to fetch
			continue
		}
		if fetched[tx.BlockNumber] {
			continue
		}
		fetched[tx.BlockNumber] = true
		fb, err := GetFinalizedBlock(ctxt, tx.BlockNumber)
		if err != nil {
			err = fmt.Errorf("failed to fetch finalized block #%d for tx '%s', %w", tx.BlockNumber, tx.Hash, err)
			log.Error(err)
			continue
		}
		fbs[fb.Number] = *fb
		fbHashes[fb.Number] = fb.TXMap()
	}
	// now set the block hash/time for the finalized transactions
	for i := 0; i < len(txs); i++ {
		if txs[i].Pending {
			continue
		}
		fb, exists := fbs[txs[i].BlockNumber]
		if !exists {
			log.Warnf("no finalized block #%d for tx '%s', it will be re-examined later", txs[i].BlockNumber, txs[i].Hash)
			continue
		}
		txs[i].FBBlockTime = fb.Timestamp
		txs[i].FBBlockHash = fb.Hash
		// does the finalized block actually contain the tx?
		_, txs[i].FBContainsTx = fbHashes[fb.Number][c.NormalizeHash(txs[i].Hash)]
		txs[i].FBDataAvailable = true
	}
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

	addFBData(ctxt, result)
	return result, nil
}

func parseTxByHash(body []byte) ([]c.TxByHash, error) {
	var resp []TxByHashBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res []c.TxByHash
	for _, d := range resp {
		if d.Result == nil {
			// the tx is neither in a block nor in the mempool; it was dropped
			// or never seen by this jsonrpc API provider. Not enough to
			// declare the donation dead -- skip it, it will be re-examined.
			log.Warnf("tx not found on chain (id: %d), skipping", d.ID)
			continue
		}
		res = append(res, *d.Result)
	}
	return res, nil
}
