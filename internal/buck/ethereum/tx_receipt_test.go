package ethereum

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TxReceiptSuite struct {
	suite.Suite
	body []byte
	path string
}

func (suite *TxReceiptSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(suite.path)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.path, err)
	}
}

func (suite *TxReceiptSuite) TestSuccess() {
	rcpt, err := parseTxReceipt(suite.body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "0x442cf6", rcpt.BlockNumber)
	assert.Equal(suite.T(), "0x1", rcpt.Status)
	assert.Equal(suite.T(), "0x631e9b031b16b18172a2b9d66c3668a68a668d20", rcpt.From)
}

func TestTxReceiptSuite(t *testing.T) {
	s := new(TxReceiptSuite)
	s.path = "testdata/eth_getTransactionReceipt.json"
	suite.Run(t, s)
}
