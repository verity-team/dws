package ethereum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func GetBlockNumber(apiURL string) (uint64, error) {
	// Create a context with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define the request data
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

	// Create an HTTP POST request with the context
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.Status != "200 OK" {
		return 0, fmt.Errorf("Request failed with status: %s", resp.Status)
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
