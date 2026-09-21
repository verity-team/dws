package ethereum

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BlockNumberSuite struct {
	suite.Suite
	finalized     []byte
	finalizedPath string
}

func (suite *BlockNumberSuite) SetupTest() {
	var err error
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
	s.finalizedPath = "testdata/latest_finalized_block3.json"
	suite.Run(t, s)
}
