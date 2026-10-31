package ethereum

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	c "github.com/verity-team/dws/internal/common"
)

// block number of the block in `testdata/18352138.json`
const testBlockNumber uint64 = 18352138

// block number of the block in `testdata/latest_finalized_block3.json`
const testFinalizedBlockNumber uint64 = 4489455

type BlockSuite struct {
	suite.Suite
	body                     []byte
	finalized                []byte
	blockPath, finalizedPath string
}

func (suite *BlockSuite) SetupTest() {
	var err error
	suite.finalized, err = os.ReadFile(suite.finalizedPath)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.finalizedPath, err)
	}
	suite.body, err = os.ReadFile(suite.blockPath)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.blockPath, err)
	}
}

func TestBlockSuite(t *testing.T) {
	s := new(BlockSuite)
	s.blockPath = "testdata/18352138.json"
	s.finalizedPath = "testdata/latest_finalized_block3.json"
	suite.Run(t, s)
}

func (suite *BlockSuite) TestFinalizedBlockDetail() {
	fb, err := parseFinalizedBlock(suite.finalized, testFinalizedBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "0x4a89b8", fb.BaseFeePerGas)
	assert.Equal(suite.T(), "0x1c9c380", fb.GasLimit)
	assert.Equal(suite.T(), "0x1abc8cc", fb.GasUsed)
	assert.Equal(suite.T(), "0x8338e3bd232ffba854afac95fbe6f1fe660b58180ac70aef80da39cff9887f18", fb.Hash)
	assert.Equal(suite.T(), uint64(4489455), fb.Number)
	assert.Equal(suite.T(), "0x6b72a4c5d07c9ba76a2e1d633d559c3db2fe352694be1bf1d2eaaa7ed72e1b3f", fb.ReceiptsRoot)
	assert.Equal(suite.T(), "0x30a80", fb.Size)
	assert.Equal(suite.T(), "0x971d75a283461085518ee1ea6e4826c38561a5c755d13f558c5a4df48c5179eb", fb.StateRoot)
	assert.Equal(suite.T(), "2023-10-14T14:44:48Z", fb.Timestamp.Format(time.RFC3339))
	assert.Equal(suite.T(), 104, len(fb.Transactions))
	assert.Equal(suite.T(), "0x0f360f24515f3dd90429f6ea05629771a9dabb7a8b7b95268ad6fdf83e82a6a5", fb.Transactions[0])
	assert.Equal(suite.T(), "0xc0ec900e31ba2ae5d18b200806845454820787ceab414a892272588ee371b3ea", fb.Transactions[len(fb.Transactions)-1])
}

func (suite *BlockSuite) TestParseBlock() {
	block, err := parseBlock(suite.body, testBlockNumber)
	hash := "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367"
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))
	assert.Equal(suite.T(), hash, block.Hash)
	assert.Equal(suite.T(), testBlockNumber, block.Number)
	assert.Equal(suite.T(), "2023-10-15T00:15:59Z", block.Timestamp.Format(time.RFC3339))
}

func (suite *BlockSuite) TestUSDTTxSuccess() {
	block, err := parseBlock(suite.body, testBlockNumber)
	assert.Nil(suite.T(), err)
	input := "0xa9059cbb0000000000000000000000007c298d22e78ead0b20c6a32dec24c6d0b9f2074f000000000000000000000000000000000000000000000000000000007ac5f665"
	from := "0x974caa59e49682cda0ad2bbe82983419a2ecc400"
	to := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	txHash := "0x3c8273e0d522380ed5c1caf943ede820251581dc77cc6b80b68a47d926b586cc"
	tx := block.Transactions[44]
	assert.Equal(suite.T(), input, tx.Input)
	assert.Equal(suite.T(), "0x0", tx.Value)
	assert.Equal(suite.T(), strings.ToLower(from), strings.ToLower(tx.From))
	assert.Equal(suite.T(), strings.ToLower(to), strings.ToLower(tx.To))
	assert.Equal(suite.T(), strings.ToLower(txHash), strings.ToLower(tx.Hash))
}

// a `null` result (the jsonrpc API provider does not have the block yet) must
// not be mistaken for an empty block -- that would cause the caller to skip
// the block for good
func (suite *BlockSuite) TestParseBlockNullResult() {
	body, err := os.ReadFile("testdata/null_block.json")
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", "testdata/null_block.json", err)
	}
	block, err := parseBlock(body, testBlockNumber)
	assert.Nil(suite.T(), block)
	assert.ErrorContains(suite.T(), err, "no data for block #18352138")
}

// a response carrying a block other than the one requested must be rejected
func (suite *BlockSuite) TestParseBlockNumberMismatch() {
	block, err := parseBlock(suite.body, testBlockNumber+1)
	assert.Nil(suite.T(), block)
	assert.ErrorContains(suite.T(), err, "block number mismatch, wanted #18352139, got #18352138")
}

// parseFinalizedBlock validates its response like parseBlock: a `null` result
// is not a block ...
func (suite *BlockSuite) TestParseFinalizedBlockNullResult() {
	body, err := os.ReadFile("testdata/null_block.json")
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", "testdata/null_block.json", err)
	}
	fb, err := parseFinalizedBlock(body, testFinalizedBlockNumber)
	assert.Nil(suite.T(), fb)
	assert.ErrorContains(suite.T(), err, "no data for block #4489455")
}

// ... and a response carrying a block other than the one requested is rejected
func (suite *BlockSuite) TestParseFinalizedBlockNumberMismatch() {
	fb, err := parseFinalizedBlock(suite.finalized, testFinalizedBlockNumber+1)
	assert.Nil(suite.T(), fb)
	assert.ErrorContains(suite.T(), err, "block number mismatch, wanted #4489456, got #4489455")
}

// ... as is one without a block hash
func (suite *BlockSuite) TestParseFinalizedBlockWithoutHash() {
	body := strings.Replace(string(suite.finalized),
		"0x8338e3bd232ffba854afac95fbe6f1fe660b58180ac70aef80da39cff9887f18", "", 1)
	fb, err := parseFinalizedBlock([]byte(body), testFinalizedBlockNumber)
	assert.Nil(suite.T(), fb)
	assert.ErrorContains(suite.T(), err, "no data for block #4489455")
}

// headerOf turns a full `eth_getBlockByNumber` response into the header only
// response the provider gives for the same block: the transactions are listed
// by hash
func headerOf(t *testing.T, full []byte) []byte {
	resp, result := decodeBlock(t, full)
	txs, ok := result["transactions"].([]interface{})
	assert.True(t, ok)
	hashes := make([]string, len(txs))
	for i, tx := range txs {
		m, ok := tx.(map[string]interface{})
		assert.True(t, ok)
		hashes[i], _ = m["hash"].(string)
	}
	result["transactions"] = hashes
	body, err := json.Marshal(resp)
	assert.Nil(t, err)
	return body
}

// decodeBlock decodes a block response; result is the block, a part of resp
func decodeBlock(t *testing.T, body []byte) (resp, result map[string]interface{}) {
	assert.Nil(t, json.Unmarshal(body, &resp))
	result, ok := resp["result"].(map[string]interface{})
	assert.True(t, ok)
	return resp, result
}

// blockServer serves the header only (`false`) and the full (`true`) response
// for the `testdata/18352138.json` block and counts the requests of each kind;
// it answers 500 when failing is set
type blockServer struct {
	mu            sync.Mutex
	full, header  []byte
	fulls, heads  int
	failing       bool
	srv           *httptest.Server
	cacheDir, url string
}

func newBlockServer(t *testing.T, full []byte) *blockServer {
	bs := &blockServer{full: full, header: headerOf(t, full), cacheDir: t.TempDir()}
	bs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rq struct {
			Params []interface{} `json:"params"`
		}
		rb, err := io.ReadAll(r.Body)
		if err != nil || json.Unmarshal(rb, &rq) != nil || len(rq.Params) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bs.mu.Lock()
		defer bs.mu.Unlock()
		if bs.failing {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if withTxs, _ := rq.Params[1].(bool); withTxs {
			bs.fulls++
			_, _ = w.Write(bs.full)
			return
		}
		bs.heads++
		_, _ = w.Write(bs.header)
	}))
	t.Cleanup(bs.srv.Close)
	bs.url = bs.srv.URL
	return bs
}

func (bs *blockServer) counts() (int, int) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.heads, bs.fulls
}

func (bs *blockServer) ctxt() c.Context {
	return c.Context{ETHRPCURL: bs.url, BlockCache: bs.cacheDir, CrawlerType: c.Finalized}
}

func (bs *blockServer) cachePath() string {
	return filepath.Join(bs.cacheDir, fmt.Sprintf("fb-%d.json", testBlockNumber))
}

// a fetched finalized block is cached, and a cached block that matches the
// provider's header is served from the cache: only the header is fetched
func (suite *BlockSuite) TestGetBlockServesValidatedCachedBlock() {
	bs := newBlockServer(suite.T(), suite.body)

	block, err := GetBlock(bs.ctxt(), testBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))
	cached, err := os.ReadFile(bs.cachePath())
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), suite.body, cached)

	block, err = GetBlock(bs.ctxt(), testBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367", block.Hash)
	heads, fulls := bs.counts()
	assert.Equal(suite.T(), 1, heads)
	assert.Equal(suite.T(), 1, fulls)
}

// a cached block that does not match the provider's header is never used,
// the block is fetched from the provider instead
func (suite *BlockSuite) TestGetBlockDiscardsMismatchingCachedBlock() {
	const (
		blockHash = "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367"
		txHash    = "0x3c8273e0d522380ed5c1caf943ede820251581dc77cc6b80b68a47d926b586cc"
		forged    = "0x1111111111111111111111111111111111111111111111111111111111111111"
	)
	full := string(suite.body)
	withoutLastTx, result := decodeBlock(suite.T(), suite.body)
	txs, _ := result["transactions"].([]interface{})
	result["transactions"] = txs[:len(txs)-1]
	truncated, err := json.Marshal(withoutLastTx)
	assert.Nil(suite.T(), err)

	tests := []struct {
		name   string
		cached string
	}{
		{"different block hash", strings.Replace(full, blockHash, forged, 1)},
		{"different block time", strings.Replace(full, `"timestamp": "0x652b2f3f"`, `"timestamp": "0x652b2f40"`, 1)},
		{"different transaction", strings.Replace(full, txHash, forged, 1)},
		{"missing transaction", string(truncated)},
		{"unparsable", "{"},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			bs := newBlockServer(suite.T(), suite.body)
			assert.NotEqual(suite.T(), full, tt.cached)
			assert.Nil(suite.T(), os.WriteFile(bs.cachePath(), []byte(tt.cached), 0o600))

			block, err := GetBlock(bs.ctxt(), testBlockNumber)
			assert.Nil(suite.T(), err)
			assert.Equal(suite.T(), blockHash, block.Hash)
			assert.Equal(suite.T(), 99, len(block.Transactions))
			assert.Equal(suite.T(), txHash, block.Transactions[44].Hash)
			_, fulls := bs.counts()
			assert.Equal(suite.T(), 1, fulls)
			// ... and the provider's copy replaces the bad one
			cached, err := os.ReadFile(bs.cachePath())
			assert.Nil(suite.T(), err)
			assert.Equal(suite.T(), suite.body, cached)
		})
	}
}

// a cached block cannot stand in for a provider that is down: without the
// header it cannot be validated
func (suite *BlockSuite) TestGetBlockCachedBlockNeedsProvider() {
	bs := newBlockServer(suite.T(), suite.body)
	assert.Nil(suite.T(), os.WriteFile(bs.cachePath(), suite.body, 0o600))
	bs.mu.Lock()
	bs.failing = true
	bs.mu.Unlock()

	block, err := GetBlock(bs.ctxt(), testBlockNumber)
	assert.Nil(suite.T(), block)
	assert.NotNil(suite.T(), err)
}

// the cache only saves requests: failing to read or write it must not fail
// the block
func (suite *BlockSuite) TestGetBlockToleratesCacheIOFailures() {
	// the cache "directory" is a file: reading and writing fb-*.json fail
	bs := newBlockServer(suite.T(), suite.body)
	notADir := filepath.Join(bs.cacheDir, "file")
	assert.Nil(suite.T(), os.WriteFile(notADir, nil, 0o600))
	ctxt := bs.ctxt()
	ctxt.BlockCache = notADir

	block, err := GetBlock(ctxt, testBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))

	// the cached block is unreadable (a directory)
	bs = newBlockServer(suite.T(), suite.body)
	assert.Nil(suite.T(), os.Mkdir(bs.cachePath(), 0o700))
	block, err = GetBlock(bs.ctxt(), testBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))
	heads, fulls := bs.counts()
	assert.Equal(suite.T(), 0, heads)
	assert.Equal(suite.T(), 1, fulls)
}

// GetFinalizedBlock fetches a header; it must never land in the block cache,
// which serves full blocks
func (suite *BlockSuite) TestGetFinalizedBlockIsNotCached() {
	bs := newBlockServer(suite.T(), suite.body)
	fb, err := GetFinalizedBlock(bs.ctxt(), testBlockNumber)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(fb.Transactions))
	_, err = os.Stat(bs.cachePath())
	assert.True(suite.T(), os.IsNotExist(err))
}
