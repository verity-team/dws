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
		if tx.Pending || tx.Absent {
			// still in the mempool or gone from the chain altogether; either
			// way there is no block to fetch
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
		if txs[i].Pending || txs[i].Absent {
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
	result, err := parseTxByHash(body, hs)
	if err != nil {
		return nil, err
	}
	writeTxsToFile(ctxt, time.Now().UTC().Unix(), body)

	addFBData(ctxt, result)
	return result, nil
}

// parseTxByHash decodes an `eth_getTransactionByHash` batch response. hs is
// the batch that was requested: a jsonrpc response carries the id of the
// request it answers, not the transaction hash, so hs is what a `null` result
// -- a transaction the provider knows nothing about -- is resolved against.
//
// Such a transaction is reported as an absent one rather than dropped on the
// floor: it is the only evidence there is that a transaction we watched get
// mined is gone from both the chain and the mempool. Whether that is enough
// to fail the donation is c.TxByHash.Judge's decision, not ours.
func parseTxByHash(body []byte, hs []c.Hashable) ([]c.TxByHash, error) {
	var resp []TxByHashBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res []c.TxByHash
	for _, d := range resp {
		if d.Result != nil {
			res = append(res, *d.Result)
			continue
		}
		// the ids handed to the provider are 1-based indexes into hs; a
		// response we cannot attribute to a request is unusable
		if d.ID < 1 || d.ID > len(hs) {
			log.Warnf("tx not found on chain, unattributable jsonrpc id %d, skipping", d.ID)
			continue
		}
		hash := hs[d.ID-1].GetHash()
		log.Warnf("tx '%s' found neither on chain nor in the mempool", hash)
		res = append(res, c.TxByHash{Hash: hash, Absent: true})
	}
	return res, nil
}
