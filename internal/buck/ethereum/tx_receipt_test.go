package ethereum

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const txrPath = "testdata/eth_getTransactionReceipt.json"

type TxReceiptSuite struct {
	suite.Suite
	body []byte
}

func (suite *TxReceiptSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(txrPath)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", txrPath, err)
	}
}

func (suite *TxReceiptSuite) TestSuccess() {
	rcpts, err := parseTxReceipt(suite.body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 2, len(rcpts))
	rcpt := rcpts[0]
	assert.Equal(suite.T(), "0x44381b", rcpt.BlockNumber)
	assert.Equal(suite.T(), "0x1", rcpt.Status)
	assert.Equal(suite.T(), "0x379738c60f658601be79e267e79cc38cea07c8f2", rcpt.From)
	rcpt = rcpts[1]
	assert.Equal(suite.T(), "0x456d71", rcpt.BlockNumber)
	assert.Equal(suite.T(), "0x1", rcpt.Status)
	assert.Equal(suite.T(), "0x9a6394faa769f066b17bd329a9d5f028719bb0bc", rcpt.From)
}
func TestTxReceiptSuite(t *testing.T) {
	s := new(TxReceiptSuite)
	suite.Run(t, s)
}

func FuzzParseTxReceipt(f *testing.F) {
	body, err := os.ReadFile(txrPath)
	if err != nil {
		f.Fatalf("failed to read test input '%s', %v", txrPath, err)
	}
	f.Add(body)
	f.Fuzz(func(t *testing.T, body []byte) {
		rcpts, err := parseTxReceipt(body)
		if err != nil {
			t.Errorf("fuzz test - parseTxReceipt: %q, %v", rcpts, err)
		}
	})
}
