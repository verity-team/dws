package ethereum

import (
	"time"

	"github.com/goccy/go-json"

	c "github.com/verity-team/dws/internal/common"
)

type TxReceiptBody struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  c.TxReceipt `json:"result"`
}

func GetData[R c.Fetchable](ctxt c.Context, hs []c.Hashable, fr c.Fetcher[R]) ([]R, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	var res []R
	batchSize := 127

	for i := 0; i < len(hs); i += batchSize {
		end := i + batchSize
		if end > len(hs) {
			end = len(hs)
		}

		batch := hs[i:end]

		br, err := fr.Fetch(ctxt, batch)
		if err != nil {
			return nil, err
		}
		res = append(res, br...)
	}
	return res, nil
}

type txrFetcher struct{}

func (txfr txrFetcher) Fetch(ctxt c.Context, hs []c.Hashable) ([]c.TxReceipt, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	rd := make([]map[string]interface{}, len(hs))
	for idx, h := range hs {
		rq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getTransactionReceipt",
			"params":  []interface{}{h.GetHash()},
			"id":      idx + 1,
		}
		rd[idx] = rq
	}
	requestBytes, err := json.Marshal(rd)
	if err != nil {
		return nil, err
	}

	params := c.HTTPParams{
		URL:         ctxt.ETHRPCURL,
		RequestBody: requestBytes,
	}
	body, err := c.HTTPPost(params)
	if err != nil {
		return nil, err
	}
	result, err := parseTxReceipt(body)
	if err != nil {
		return nil, err
	}
	writeTxReceiptsToFile(ctxt, time.Now().UTC().Unix(), body)

	return result, nil
}

func parseTxReceipt(body []byte) ([]c.TxReceipt, error) {
	var resp []TxReceiptBody
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	var res = make([]c.TxReceipt, len(resp))
	for i, d := range resp {
		res[i] = d.Result
	}
	return res, nil
}
