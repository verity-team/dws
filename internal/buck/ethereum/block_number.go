package ethereum

import (
	"fmt"
	"strconv"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	c "github.com/verity-team/dws/internal/common"
)

func MostRecentBlockNumber(ctxt c.Context) (uint64, error) {
	label := "finalized"
	if ctxt.CrawlerType == c.Latest {
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

	params := c.HTTPParams{
		URL:              ctxt.ETHRPCURL,
		RequestBody:      requestBytes,
		MaxWaitInSeconds: ctxt.MaxWaitInSeconds,
	}
	body, err := c.HTTPPost(params)
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
