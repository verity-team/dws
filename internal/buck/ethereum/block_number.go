package ethereum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
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
		return 0, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	// Parse the JSON response
	var responseData map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&responseData); err != nil {
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
