package ethereum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const MaxWaitInSeconds = 5

func GetBlockNumber(apiURL string) (uint64, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	requestData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("eth_blockNumber request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
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

func GetLatestFinalizedBlockNumber(apiURL string) (uint64, error) {
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

	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	blockNumber, err := parseLatestFinalizedBlock(body)
	if err != nil {
		return 0, err
	}
	log.Infof("latest finalized block number: %d", blockNumber)

	return blockNumber, nil
}

func parseLatestFinalizedBlock(body []byte) (uint64, error) {
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
