package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/api"
)

// PrivateKeyEnvVar is where the ECDSA private key used to sign the request is
// read from.
//
// It used to be a command-line flag, which put the raw key into `argv`: it is
// then readable by every user on the box via `ps aux` / /proc/<pid>/cmdline
// for as long as the process lives, and it is persisted in the shell history
// on top of that.
const PrivateKeyEnvVar = "DWS_TT_PRIVATE_KEY"

func main() {
	url := flag.String("url", "http://localhost:8080/affiliate/code", "URL for the POST request")
	delphiKey := flag.String("delphi-key", "0xb938F65DfE303EdF96A511F1e7E3190f69036860", "eth address")
	timeoutSeconds := flag.Int("timeout", 5, "request timeout in seconds")
	simulateStaleTS := flag.Bool("old-ts", false, "simulate stale auth timestamp")
	simulateFutureTS := flag.Bool("future-ts", false, "simulate an auth timestamp in the future")
	msg := flag.String("msg", "affiliate code", "message to sign")
	flag.Parse()

	privateKey, present := os.LookupEnv(PrivateKeyEnvVar)
	if !present || strings.TrimSpace(privateKey) == "" {
		log.Fatalf("please set the %s environment variable", PrivateKeyEnvVar)
	}

	// the parse error carries the key material it choked on
	pk, err := crypto.HexToECDSA(strings.TrimSpace(privateKey))
	if err != nil {
		log.Fatalf("failed to parse the private key in %s", PrivateKeyEnvVar)
	}

	client := &http.Client{
		Timeout: time.Duration(*timeoutSeconds) * time.Second,
	}

	req, err := http.NewRequest("POST", *url, nil)
	if err != nil {
		log.Fatal("error creating request:", err)
	}
	req.Header.Set("delphi-key", *delphiKey)

	ts := time.Now().UTC()

	if *simulateStaleTS {
		// -10 days
		ts = ts.Add(-1 * time.Hour * 24 * 10)
	}
	if *simulateFutureTS {
		// +10 days
		ts = ts.Add(time.Hour * 24 * 10)
	}
	req.Header.Set("delphi-ts", fmt.Sprintf("%d", ts.Unix()))

	signature, err := signMessage(*msg, *delphiKey, pk, ts)
	if err != nil {
		log.Fatalf("error signing message: %v", err)
	}
	req.Header.Set("delphi-signature", signature)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("error sending request:", err)
	}
	defer func() { _ = resp.Body.Close() }()

	log.Info("Status Code:", resp.Status)

	// Read the response body
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		log.Fatal(err)
	}
	log.Info(string(responseBody))

	// a non-2xx answer carries an error document, not an affiliate code:
	// unmarshalling it yields the zero value, which used to be printed as if
	// it were the result
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("failed to request affiliate code with status: %s", resp.Status)
	}

	// Define a struct to unmarshal the JSON response
	var data api.AffiliateCode

	// Unmarshal the JSON response into the struct
	if err := json.Unmarshal(responseBody, &data); err != nil {
		log.Fatal(err)
	}

	log.Info(data)
}

// maxResponseBytes bounds what this tool reads into memory; the endpoint it
// talks to answers with a handful of bytes.
const maxResponseBytes = 1 << 20

// signMessage builds and signs the message delphi expects: the words of the
// endpoint path, the lower cased address and the timestamp.
func signMessage(msg, address string, pk *ecdsa.PrivateKey, ts time.Time) (string, error) {
	tmsg := fmt.Sprintf("%s, %s, %s", msg, strings.ToLower(address), ts.Format("2006-01-02 15:04:05-07:00"))
	log.Info(tmsg)
	msgHash := accounts.TextHash([]byte(tmsg))
	signature, err := crypto.Sign(msgHash, pk)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signature), nil
}
