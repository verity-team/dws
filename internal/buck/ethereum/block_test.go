package ethereum

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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

func (suite *BlockSuite) TestLatestFinalizedBlockSuccess() {
	actual, err := parseMostRecentBlockNumber(suite.finalized)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(4489455), actual)
}

func TestBlockSuite(t *testing.T) {
	s := new(BlockSuite)
	s.blockPath = "testdata/18352138.json"
	s.finalizedPath = "testdata/latest_finalized_block3.json"
	suite.Run(t, s)
}

func (suite *BlockSuite) TestFinalizedBlockDetail() {
	fb, err := parseFinalizedBlock(suite.finalized)
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
	block, err := parseBlock(suite.body)
	hash := "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367"
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))
	assert.Equal(suite.T(), hash, block.Hash)
	assert.Equal(suite.T(), uint64(18352138), block.Number)
	assert.Equal(suite.T(), "2023-10-15T00:15:59Z", block.Timestamp.Format(time.RFC3339))
}

func (suite *BlockSuite) TestUSDTTxSuccess() {
	block, err := parseBlock(suite.body)
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
