package ethereum

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
	c "github.com/verity-team/dws/internal/common"
)

func fetchBlock(ctxt c.Context, bn uint64) ([]byte, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", bn), true},
		ID:      1,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	params := c.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := c.HTTPPost(params)
	if err != nil {
		return nil, err
	}

	log.Infof("fetched block %d", bn)

	return body, nil
}

func GetBlock(ctxt c.Context, bn uint64) (*c.Block, error) {
	// try getting the finalized block from the cache
	body, err := getFinalizedBlockFromCache(ctxt, bn)
	if err != nil {
		log.Debugf("block %d not in cache", bn)
	} else if body != nil {
		log.Infof("****** block %d served from cache", bn)
		block, perr := parseBlock(body, bn)
		if perr == nil {
			return block, nil
		}
		// hmm .. maybe the cached copy was corrupted .. fetch again and retry
		perr = fmt.Errorf("failed to parse cached block #%d, %w", bn, perr)
		log.Error(perr)
	}

	// not found in cache (or the cached copy is unusable) -- get it from the
	// ethereum jsonrpc API provider
	body, err = fetchBlock(ctxt, bn)
	if err != nil {
		return nil, err
	}
	block, err := parseBlock(body, bn)
	if err != nil {
		err = fmt.Errorf("failed to parse block #%d, %w", bn, err)
		log.Error(err)
		return nil, err
	}

	// only persist block data that parsed and validated successfully, a bogus
	// body would poison the cache permanently
	if err := writeBlockToFile(ctxt, bn, body); err != nil {
		return nil, err
	}

	return block, nil
}

// parseBlock parses the response to an `eth_getBlockByNumber` request for
// block `bn`. A `null` result (block not available (yet) for the jsonrpc API
// provider) or a response carrying a different block unmarshals into a
// zero-valued/mismatching block and is rejected -- silently treating it as an
// empty block would cause the caller to skip the block for good.
func parseBlock(body []byte, bn uint64) (*c.Block, error) {
	type Response struct {
		Block c.Block `json:"result"`
	}

	var resp Response
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	b := resp.Block
	if b.Hash == "" {
		err = fmt.Errorf("no data for block #%d", bn)
		log.Error(err)
		return nil, err
	}
	if b.Number != bn {
		err = fmt.Errorf("block number mismatch, wanted #%d, got #%d (%s)", bn, b.Number, b.Hash)
		log.Error(err)
		return nil, err
	}
	log.Infof("parsed block %d, %s -- %d transactions", b.Number, b.Hash, len(b.Transactions))
	return &b, nil
}

func GetFinalizedBlock(ctxt c.Context, blockNumber uint64) (*c.FinalizedBlock, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", blockNumber), false},
		ID:      1,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	params := c.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := c.HTTPPost(params)
	if err != nil {
		return nil, err
	}

	fb, err := parseFinalizedBlock(body)
	if err != nil {
		err = fmt.Errorf("failed to parse finalized block #%d, %w", blockNumber, err)
		log.Error(err)
		return nil, err
	}

	// only persist block data that parsed successfully
	err = writeBlockToFile(ctxt, blockNumber, body)
	if err != nil {
		return nil, err
	}

	return fb, nil
}

func parseFinalizedBlock(body []byte) (*c.FinalizedBlock, error) {
	type fblock struct {
		c.FinalizedBlock
		HexSeconds string `json:"timestamp"`
		HexNumber  string `json:"number"`
	}
	type Response struct {
		Block fblock `json:"result"`
	}

	var resp Response
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	seconds, err := c.HexStringToDecimal(resp.Block.HexSeconds)
	if err != nil {
		err = fmt.Errorf("failed to convert block timestamp, %w", err)
		return nil, err
	}
	number, err := c.HexStringToDecimal(resp.Block.HexNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %w", err)
		return nil, err
	}
	bn, err := c.ToUint64(number, "block number")
	if err != nil {
		return nil, err
	}
	ts := time.Unix(seconds.IntPart(), 0)
	result := c.FinalizedBlock{
		BaseFeePerGas: resp.Block.BaseFeePerGas,
		GasLimit:      resp.Block.GasLimit,
		GasUsed:       resp.Block.GasUsed,
		Hash:          resp.Block.Hash,
		Number:        bn,
		ReceiptsRoot:  resp.Block.ReceiptsRoot,
		Size:          resp.Block.Size,
		StateRoot:     resp.Block.StateRoot,
		Timestamp:     ts.UTC(),
		Transactions:  resp.Block.Transactions,
	}
	return &result, nil
}
