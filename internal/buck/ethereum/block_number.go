package ethereum

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

func GetBlockNumber(ctxt common.Context) (uint64, error) {
	requestData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}
	requestBytes, err := json.Marshal(requestData)
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

	blockNumber, err := parseLatestBlock(body)
	if err != nil {
		return 0, err
	}
	log.Infof("latest block number: %d", blockNumber)
	return blockNumber, nil
}

func parseLatestBlock(body []byte) (uint64, error) {
	// Parse the JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, err
	}

	// Extract the result from the response and convert it to a uint64
	resultStr, exists := responseData["result"].(string)
	if !exists {
		return 0, fmt.Errorf("Result not found in the response")
	}

	blockNumber, err := strconv.ParseUint(resultStr, 0, 64)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

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
		err = fmt.Errorf("failed to parse finalized block #%d, %v", blockNumber, err)
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
		err = fmt.Errorf("failed to convert block timestamp, %v", err)
		return nil, err
	}
	number, err := common.HexStringToDecimal(resp.Block.HexNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %v", err)
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

func GetMaxFinalizedBlockNumber(ctxt common.Context) (uint64, error) {
	request := EthGetBlockByNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{"finalized", false},
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

	blockNumber, err := parseMaxFinalizedBlock(body)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

func parseMaxFinalizedBlock(body []byte) (uint64, error) {
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
		log.Error(err)
		return 0, err
	}
	return blockNumber, nil
}
