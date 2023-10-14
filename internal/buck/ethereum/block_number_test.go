package ethereum

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BlockNumberSuite struct {
	suite.Suite
	latest, finalized         []byte
	latestPath, finalizedPath string
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
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

func (suite *BlockNumberSuite) TestLatestBlockSuccess() {
	actual, err := parseLatestBlock(suite.latest)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(4487104), actual)
}

func (suite *BlockNumberSuite) TestLatestFinalizedBlockSuccess() {
	actual, err := parseLatestFinalizedBlock(suite.finalized)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(4486885), actual)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestBlockNumberSuite(t *testing.T) {
	s := new(BlockNumberSuite)
	s.latestPath = "testdata/latest_block.json"
	s.finalizedPath = "testdata/latest_finalized_block.json"
	suite.Run(t, s)
}
