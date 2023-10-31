package ethereum

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BlockNumberSuite struct {
	suite.Suite
	latest, finalized         []byte
	latestPath, finalizedPath string
}

func (suite *BlockNumberSuite) SetupTest() {
	var err error
	suite.latest, err = os.ReadFile(suite.latestPath)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.latestPath, err)
	}
	suite.finalized, err = os.ReadFile(suite.finalizedPath)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.finalizedPath, err)
	}
}

func (suite *BlockNumberSuite) TestLatestFinalizedBlockSuccess() {
	actual, err := parseMostRecentBlockNumber(suite.finalized)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(4489455), actual)
}

func TestBlockNumberSuite(t *testing.T) {
	s := new(BlockNumberSuite)
	s.latestPath = "testdata/latest_block.json"
	s.finalizedPath = "testdata/latest_finalized_block3.json"
	suite.Run(t, s)
}

func (suite *BlockNumberSuite) TestFinalizedBlockDetail() {
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
