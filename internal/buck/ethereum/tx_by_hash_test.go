package ethereum

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	c "github.com/verity-team/dws/internal/common"
)

type TxByHashSuite struct {
	suite.Suite
	body []byte
	path string
}

func (suite *TxByHashSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(suite.path)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.path, err)
	}
}

func (suite *TxByHashSuite) TestSuccess() {
	txs, err := parseTxByHash(suite.body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 2, len(txs))
	tx := txs[0]
	assert.Equal(suite.T(), uint64(18459476), tx.BlockNumber)
	assert.Equal(suite.T(), uint64(20), tx.TransactionIndex)
	assert.Equal(suite.T(), "0xa0ebed29f62dfd4b3d83af9e79e3c23170d45621", tx.From)
	assert.Equal(suite.T(), "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7", tx.Hash)
	tx = txs[1]
	assert.Equal(suite.T(), uint64(18459264), tx.BlockNumber)
	assert.Equal(suite.T(), uint64(76), tx.TransactionIndex)
	assert.Equal(suite.T(), "0x655061986e756f2f4a89cd748258ba251805367d", tx.From)
	assert.Equal(suite.T(), "0x50ddd63a864794c2375281929e025b56ef9356cd188b3b1299daf15fb0e842ca", tx.Hash)
}

func TestTxByHashSuite(t *testing.T) {
	s := new(TxByHashSuite)
	s.path = "testdata/txbyhash.json"
	suite.Run(t, s)
}

// a single transaction that is still in the mempool ("blockNumber": null)
// must not kill the decode of the batch it is part of
func (suite *TxByHashSuite) TestPendingTxInBatch() {
	body, err := os.ReadFile("testdata/txbyhash_pending.json")
	assert.Nil(suite.T(), err)

	txs, err := parseTxByHash(body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 3, len(txs))

	// the mined transactions are present and correct
	tx := txs[0]
	assert.False(suite.T(), tx.Pending)
	assert.Equal(suite.T(), uint64(18459476), tx.BlockNumber)
	assert.Equal(suite.T(), uint64(20), tx.TransactionIndex)
	assert.Equal(suite.T(), "0xa0ebed29f62dfd4b3d83af9e79e3c23170d45621", tx.From)
	assert.Equal(suite.T(), "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7", tx.Hash)
	tx = txs[2]
	assert.False(suite.T(), tx.Pending)
	assert.Equal(suite.T(), uint64(18459264), tx.BlockNumber)
	assert.Equal(suite.T(), uint64(76), tx.TransactionIndex)
	assert.Equal(suite.T(), "0x655061986e756f2f4a89cd748258ba251805367d", tx.From)
	assert.Equal(suite.T(), "0x50ddd63a864794c2375281929e025b56ef9356cd188b3b1299daf15fb0e842ca", tx.Hash)

	// the pending transaction is flagged as such
	tx = txs[1]
	assert.True(suite.T(), tx.Pending)
	assert.Equal(suite.T(), c.PendingBlockNumber, tx.BlockNumber)
	assert.Equal(suite.T(), "", tx.BlockHash)
	assert.Equal(suite.T(), "0xfeedfacecafebeef1122334455667788990011223344556677889900aabbccdd", tx.Hash)

	// ... and it is never failed, no matter how far the chain has moved on
	assert.Equal(suite.T(), c.TxPending, tx.Judge(uint64(99999999)))
}

// blockNumberRequested extracts the block number an `eth_getBlockByNumber`
// request body asks for
func blockNumberRequested(suite *TxByHashSuite, body []byte) string {
	var rq struct {
		Params []interface{} `json:"params"`
	}
	assert.Nil(suite.T(), json.Unmarshal(body, &rq))
	assert.NotEmpty(suite.T(), rq.Params)
	bn, ok := rq.Params[0].(string)
	assert.True(suite.T(), ok)
	return bn
}

// a failed finalized block lookup for one transaction must not prevent the
// others from being judged
func (suite *TxByHashSuite) TestAddFBDataToleratesFailedLookup() {
	fb, err := os.ReadFile("testdata/fb_18459476.json")
	assert.Nil(suite.T(), err)

	var (
		mu        sync.Mutex
		requested []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rb, rerr := io.ReadAll(r.Body)
		if rerr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bn := blockNumberRequested(suite, rb)
		mu.Lock()
		requested = append(requested, bn)
		mu.Unlock()
		// block 0x119aa80 (18459264) is unavailable
		if bn != "0x119ab54" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fb)
	}))
	defer srv.Close()

	body, err := os.ReadFile("testdata/txbyhash_pending.json")
	assert.Nil(suite.T(), err)
	txs, err := parseTxByHash(body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 3, len(txs))

	ctxt := c.Context{ETHRPCURL: srv.URL, CrawlerType: c.OldUnconfirmed}
	addFBData(ctxt, txs)

	// no block is fetched for the pending transaction
	mu.Lock()
	assert.Equal(suite.T(), []string{"0x119ab54", "0x119aa80"}, requested)
	mu.Unlock()

	// the transaction whose block could be fetched is judged
	assert.True(suite.T(), txs[0].FBDataAvailable)
	assert.True(suite.T(), txs[0].FBContainsTx)
	assert.Equal(suite.T(), "0x634f3802299d60d38418bad02789a8b36a3051bc62a293b45deeafe5ec51fa34", txs[0].FBBlockHash)
	assert.Equal(suite.T(), c.TxFinalize, txs[0].Judge(uint64(18459500)))

	// the pending transaction is left alone
	assert.False(suite.T(), txs[1].FBDataAvailable)
	assert.Equal(suite.T(), c.TxPending, txs[1].Judge(uint64(18459500)))

	// the transaction whose block lookup failed is left alone, *not* failed
	assert.False(suite.T(), txs[2].FBDataAvailable)
	assert.Equal(suite.T(), c.TxNoFinalizedBlockData, txs[2].Judge(uint64(18459500)))
}

// the finalized block does not carry the transaction => it is failed
func (suite *TxByHashSuite) TestAddFBDataMissingTxIsFailed() {
	fb, err := os.ReadFile("testdata/fb_18459476.json")
	assert.Nil(suite.T(), err)
	// remove the tx from the finalized block (re-org/replacement)
	fb = []byte(strings.Replace(string(fb),
		"0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7",
		"0x1111111111111111111111111111111111111111111111111111111111111111", 1))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fb)
	}))
	defer srv.Close()

	txs, err := parseTxByHash(suite.body)
	assert.Nil(suite.T(), err)
	txs = txs[:1]

	addFBData(c.Context{ETHRPCURL: srv.URL, CrawlerType: c.OldUnconfirmed}, txs)
	assert.True(suite.T(), txs[0].FBDataAvailable)
	assert.False(suite.T(), txs[0].FBContainsTx)
	assert.Equal(suite.T(), c.TxFail, txs[0].Judge(uint64(18459500)))
}

// a jsonrpc API provider that reports the tx/block hashes of
// `eth_getTransactionByHash` in a different casing than those of
// `eth_getBlockByNumber` must not cause a good donation to be failed
func (suite *TxByHashSuite) TestAddFBDataToleratesHashCasing() {
	fb, err := os.ReadFile("testdata/fb_18459476.json")
	assert.Nil(suite.T(), err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fb)
	}))
	defer srv.Close()

	// upper case the tx hash and the block hash in the tx response; the
	// finalized block fixture carries both in lower case
	const (
		txHash    = "0xad246b9af8a4bfd4043d6b0a700c70f084c79aa22a8677007716fc70ccefc7e7"
		blockHash = "0x634f3802299d60d38418bad02789a8b36a3051bc62a293b45deeafe5ec51fa34"
	)
	body := string(suite.body)
	body = strings.ReplaceAll(body, txHash, strings.ToUpper(txHash))
	body = strings.ReplaceAll(body, blockHash, strings.ToUpper(blockHash))

	txs, err := parseTxByHash([]byte(body))
	assert.Nil(suite.T(), err)
	txs = txs[:1]
	// the value received is stored as is, only comparisons are normalized
	assert.Equal(suite.T(), strings.ToUpper(txHash), txs[0].Hash)

	addFBData(c.Context{ETHRPCURL: srv.URL, CrawlerType: c.OldUnconfirmed}, txs)
	assert.True(suite.T(), txs[0].FBDataAvailable)
	// the upper case tx hash is still found in the finalized block's tx set
	assert.True(suite.T(), txs[0].FBContainsTx)
	assert.Equal(suite.T(), c.TxFinalize, txs[0].Judge(uint64(18459500)))
}
