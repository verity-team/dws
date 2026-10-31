package ethereum

import (
	"errors"
	"fmt"
	"io/fs"
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
	if block := getCachedBlock(ctxt, bn); block != nil {
		return block, nil
	}

	// not found in cache (or the cached copy is unusable) -- get it from the
	// ethereum jsonrpc API provider
	body, err := fetchBlock(ctxt, bn)
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
	writeBlockToFile(ctxt, bn, body)

	return block, nil
}

// getCachedBlock returns the cached copy of block bn, or nil if there is none
// or it cannot be trusted; the caller then fetches the block from the jsonrpc
// API provider. The cache only saves bandwidth, so nothing about it -- a read
// error included -- may fail the block.
//
// A cached copy is never used on its own word: the cache directory comes from
// the environment and anyone able to write to it could otherwise feed
// fabricated blocks to the crawlers. Its hash, timestamp (it prices the ETH
// donations) and ordered transaction hashes must match the provider's header
// of the same block, fetched without the transaction bodies. That rejects a
// substituted, stale, truncated or corrupted copy and any added, removed or
// reordered transaction. It does not re-hash the transaction bodies, so a
// copy whose transaction fields were doctored under their genuine hashes is
// not caught: the directory must be as well protected as the binary.
func getCachedBlock(ctxt c.Context, bn uint64) *c.Block {
	body, err := getFinalizedBlockFromCache(ctxt, bn)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debugf("block %d not in cache", bn)
		} else {
			log.Warnf("failed to read cached block #%d, %v", bn, err)
		}
		return nil
	}
	if body == nil {
		return nil
	}
	block, err := parseBlock(body, bn)
	if err != nil {
		// hmm .. maybe the cached copy was corrupted .. fetch again
		log.Warnf("failed to parse cached block #%d, %v", bn, err)
		return nil
	}
	header, _, err := fetchFinalizedBlock(ctxt, bn)
	if err != nil {
		log.Warnf("failed to fetch the header of cached block #%d, %v", bn, err)
		return nil
	}
	if err = matchHeader(block, header); err != nil {
		log.Warnf("cached block #%d does not match the provider's, discarding it, %v", bn, err)
		return nil
	}
	log.Infof("****** block %d served from cache", bn)
	return block
}

// matchHeader reports how block b differs from the header h the jsonrpc API
// provider reports for the same block, nil if it does not. The block numbers
// were validated by the parsers already.
func matchHeader(b *c.Block, h *c.FinalizedBlock) error {
	if c.NormalizeHash(b.Hash) != c.NormalizeHash(h.Hash) {
		return fmt.Errorf("block hash %s, the provider has %s", b.Hash, h.Hash)
	}
	if !b.Timestamp.Equal(h.Timestamp) {
		return fmt.Errorf("block time %v, the provider has %v", b.Timestamp, h.Timestamp)
	}
	if len(b.Transactions) != len(h.Transactions) {
		return fmt.Errorf("%d transactions, the provider has %d", len(b.Transactions), len(h.Transactions))
	}
	for i, tx := range b.Transactions {
		if c.NormalizeHash(tx.Hash) != c.NormalizeHash(h.Transactions[i]) {
			return fmt.Errorf("transaction #%d is %s, the provider has %s", i, tx.Hash, h.Transactions[i])
		}
	}
	return nil
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

// GetFinalizedBlock fetches the header of block blockNumber -- the block data
// without the transaction bodies, the transactions are listed by hash.
func GetFinalizedBlock(ctxt c.Context, blockNumber uint64) (*c.FinalizedBlock, error) {
	fb, body, err := fetchFinalizedBlock(ctxt, blockNumber)
	if err != nil {
		return nil, err
	}
	// only persist block data that parsed successfully. A header is never
	// written to the block cache, which serves full blocks (see GetBlock).
	writeDebugBlock(ctxt, blockNumber, body)
	return fb, nil
}

// fetchFinalizedBlock fetches and validates the header of block blockNumber;
// the raw response is returned as well.
func fetchFinalizedBlock(ctxt c.Context, blockNumber uint64) (*c.FinalizedBlock, []byte, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", blockNumber), false},
		ID:      1,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	params := c.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := c.HTTPPost(params)
	if err != nil {
		return nil, nil, err
	}

	fb, err := parseFinalizedBlock(body, blockNumber)
	if err != nil {
		err = fmt.Errorf("failed to parse finalized block #%d, %w", blockNumber, err)
		log.Error(err)
		return nil, nil, err
	}
	return fb, body, nil
}

// parseFinalizedBlock parses the (header only) response to an
// `eth_getBlockByNumber` request for block `bn`. It validates the response the
// way parseBlock does: a `null` result or a response carrying a different
// block is rejected rather than handed to a caller that may trust the
// requested block number.
func parseFinalizedBlock(body []byte, bn uint64) (*c.FinalizedBlock, error) {
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
	if resp.Block.Hash == "" {
		return nil, fmt.Errorf("no data for block #%d", bn)
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
	rbn, err := c.ToUint64(number, "block number")
	if err != nil {
		return nil, err
	}
	if rbn != bn {
		return nil, fmt.Errorf("block number mismatch, wanted #%d, got #%d (%s)", bn, rbn, resp.Block.Hash)
	}
	ts := time.Unix(seconds.IntPart(), 0)
	result := c.FinalizedBlock{
		BaseFeePerGas: resp.Block.BaseFeePerGas,
		GasLimit:      resp.Block.GasLimit,
		GasUsed:       resp.Block.GasUsed,
		Hash:          resp.Block.Hash,
		Number:        rbn,
		ReceiptsRoot:  resp.Block.ReceiptsRoot,
		Size:          resp.Block.Size,
		StateRoot:     resp.Block.StateRoot,
		Timestamp:     ts.UTC(),
		Transactions:  resp.Block.Transactions,
	}
	return &result, nil
}
