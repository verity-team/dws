package ethereum

import (
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

func GetFinalizedBlock(ctxt common.Context, blockNumber uint64) (*common.FinalizedBlock, error) {
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

	params := common.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := common.HTTPPost(params)
	if err != nil {
		return nil, err
	}

	err = writeBlockToFile(ctxt, blockNumber, body, true)
	if err != nil {
		return nil, err
	}

	fb, err := parseFinalizedBlock(body)
	if err != nil {
		err = fmt.Errorf("failed to parse finalized block #%d, %w", blockNumber, err)
		log.Error(err)
		return nil, err
	}
	return fb, nil
}

func parseFinalizedBlock(body []byte) (*common.FinalizedBlock, error) {
	type fblock struct {
		common.FinalizedBlock
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
	seconds, err := common.HexStringToDecimal(resp.Block.HexSeconds)
	if err != nil {
		err = fmt.Errorf("failed to convert block timestamp, %w", err)
		return nil, err
	}
	number, err := common.HexStringToDecimal(resp.Block.HexNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %w", err)
		return nil, err
	}
	ts := time.Unix(seconds.IntPart(), 0)
	result := common.FinalizedBlock{
		BaseFeePerGas: resp.Block.BaseFeePerGas,
		GasLimit:      resp.Block.GasLimit,
		GasUsed:       resp.Block.GasUsed,
		Hash:          resp.Block.Hash,
		Number:        uint64(number.IntPart()),
		ReceiptsRoot:  resp.Block.ReceiptsRoot,
		Size:          resp.Block.Size,
		StateRoot:     resp.Block.StateRoot,
		Timestamp:     ts.UTC(),
		Transactions:  resp.Block.Transactions,
	}
	return &result, nil
}

func MostRecentBlockNumber(ctxt common.Context) (uint64, error) {
	label := "finalized"
	if ctxt.CrawlerType == common.Latest {
		label = "latest"
	}
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{label, false},
		ID:      1,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}

	params := common.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := common.HTTPPost(params)
	if err != nil {
		return 0, err
	}

	blockNumber, err := parseMostRecentBlockNumber(body)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

func parseMostRecentBlockNumber(body []byte) (uint64, error) {
	type Response struct {
		Result struct {
			Number string `json:"number"`
		} `json:"result"`
	}

	var resp Response
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return 0, err
	}
	blockNumber, err := strconv.ParseUint(resp.Result.Number, 0, 64)
	if err != nil {
		err = fmt.Errorf("failed to parse block number ('%s'), %w", resp.Result.Number, err)
		log.Error(err)
		return 0, err
	}
	return blockNumber, nil
}
