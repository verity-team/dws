package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
)

func main() {
	privateKey := flag.String("private-key", "", "private key for generating delphi-signature")
	url := flag.String("url", "http://localhost:8080/affiliate/code", "URL for the POST request")
	delphiKey := flag.String("delphi-key", "0xb938F65DfE303EdF96A511F1e7E3190f69036860", "eth address")
	timeoutSeconds := flag.Int("timeout", 5, "request timeout in seconds")
	simulateStaleTS := flag.Bool("old-ts", false, "simulate stale auth timestamp")
	flag.Parse()

	if *privateKey == "" {
		log.Fatal("please pass a private key")
	}

	pk, err := crypto.HexToECDSA(*privateKey)
	if err != nil {
		log.Fatalf("error parsing private key: %v", err)
	}

	client := &http.Client{
		Timeout: time.Duration(*timeoutSeconds) * time.Second,
	}

	req, err := http.NewRequest("POST", *url, nil)
	if err != nil {
		log.Fatal("error creating request:", err)
		return
	}
	req.Header.Set("delphi-key", *delphiKey)

	ts := time.Now()

	if *simulateStaleTS {
		// -10 days
		ts = ts.Add(-1 * time.Hour * 24 * 10)
	}
	req.Header.Set("delphi-ts", fmt.Sprintf("%d", ts.Unix()))

	signature, err := signMessage(pk, ts)
	if err != nil {
		log.Fatalf("error signing message: %v", err)
		return
	}
	req.Header.Set("delphi-signature", signature)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("error sending request:", err)
		return
	}
	defer resp.Body.Close()

	log.Info("Status Code:", resp.Status)
	if resp.StatusCode != http.StatusOK {
		log.Errorf("failed to request affiliate code with status: %s", resp.Status)
	}

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Info(string(responseBody))
	// Define a struct to unmarshal the JSON response
	var data api.AffiliateCode

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &data); err != nil {
		log.Fatal(err)
	}

	log.Info(data)
}

func signMessage(pk *ecdsa.PrivateKey, ts time.Time) (string, error) {
	msg := fmt.Sprintf("get affiliate code, %s", ts.Format("2006-01-02 15:04:05-07:00"))
	msgHash := accounts.TextHash([]byte(msg))
	signature, err := crypto.Sign(msgHash, pk)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signature), nil
}
