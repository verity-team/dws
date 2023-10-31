package ethereum

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
