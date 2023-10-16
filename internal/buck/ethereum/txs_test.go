package ethereum

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/verity-team/dws/internal/common"
)

func TestParseInputData(t *testing.T) {
	tests := []struct {
		input          string
		expectedAddr   string
		expectedAmount decimal.Decimal
		expectedError  string
	}{
		// Valid input case
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec052700000000000000000000000000000000000000000000000000000000000be293",
			expectedAddr:   "0x865a1f30b979e4bf3ab30562daee05f917ec0527",
			expectedAmount: decimal.NewFromInt32(778899),
			expectedError:  "",
		},
		// Input with invalid length
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a7640",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "input has invalid length",
		},
		// Input with incorrect function signature
		{
			input:          "0x12345678000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a76400000",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "input does not start with the expected function signature",
		},
		// Input with invalid amount
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000invalidamount000",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "failed to convert amount to uint64",
		},
	}

	for _, test := range tests {
		addr, amount, err := parseInputData(test.input)

		// Check the address, amount, and error
		assert.Equal(t, test.expectedAddr, addr, "Address mismatch")
		assert.Equal(t, test.expectedAmount, amount, "Amount mismatch")

		if test.expectedError != "" {
			assert.NotNil(t, err, "Expected an error")
			assert.Equal(t, test.expectedError, err.Error(), "Error message mismatch")
		} else {
			assert.Nil(t, err, "Expected no error")
		}
	}
}

type TxsSuite struct {
	suite.Suite
	body []byte
	path string
}

func (suite *TxsSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(suite.path)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.path, err)
	}
}

func (suite *TxsSuite) TestLatestBlockSuccess() {
	block, err := parseBlock(suite.body)
	hash := "0xf1199f7db7e1029fb8ea5479033ab4a221457e51fa482b4195d380fb83f89425"
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 200, len(block.Transactions))
	assert.Equal(suite.T(), hash, block.Hash)
	assert.Equal(suite.T(), uint64(4470811), block.Number)
	assert.Equal(suite.T(), "2023-10-11T16:04:00Z", block.Timestamp.Format(time.RFC3339))
}

func (suite *TxsSuite) TestLinkTxSuccess() {
	block, err := parseBlock(suite.body)
	assert.Nil(suite.T(), err)
	input := "0xa9059cbb000000000000000000000000ded1fe6b3f61c8f1d874bb86f086d10ffc3f015400000000000000000000000000000000000000000000000010cac896d2390000"
	from := "0x379738c60f658601Be79e267e79cC38cEA07c8f2"
	to := "0x779877A7B0D9E8603169DdbD7836e478b4624789"
	txHash := "0xf270a01e1ffa619b5262df30dc93d5ea1cf4bff773d6494460a1755abae43989"
	tx := block.Transactions[77]
	assert.Equal(suite.T(), input, tx.Input)
	assert.Equal(suite.T(), strings.ToLower(from), strings.ToLower(tx.From))
	assert.Equal(suite.T(), strings.ToLower(to), strings.ToLower(tx.To))
	assert.Equal(suite.T(), strings.ToLower(txHash), strings.ToLower(tx.Hash))
}

func (suite *TxsSuite) TestInputData() {
	input := "0xa9059cbb000000000000000000000000ded1fe6b3f61c8f1d874bb86f086d10ffc3f015400000000000000000000000000000000000000000000000010cac896d2390000"
	recipient, amount, err := parseInputData(input)
	assert.Nil(suite.T(), err)
	to := "0xDEd1Fe6B3f61c8F1d874bb86F086D10FFc3F0154"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromInt(1210000000000000000), amount)
}

func (suite *TxsSuite) TestERC20Tx() {
	block, err := parseBlock(suite.body)
	assert.Nil(suite.T(), err)
	to := "0xDEd1Fe6B3f61c8F1d874bb86F086D10FFc3F0154"
	contract := strings.ToLower("0x779877A7B0D9E8603169DdbD7836e478b4624789")
	hash := "0xf270a01e1ffa619b5262df30dc93d5ea1cf4bff773d6494460a1755abae43989"
	ctxt := common.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]common.ERC20{
			contract: {
				Asset:   "link",
				Address: contract,
				Scale:   18,
			},
		},
	}
	txs, err := filterTransactions(ctxt, *block)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(txs))
	assert.Equal(suite.T(), hash, txs[0].Hash)
	assert.Equal(suite.T(), block.Hash, txs[0].BlockHash)
	assert.Equal(suite.T(), block.Number, txs[0].BlockNumber)
	assert.Equal(suite.T(), "1.210000", txs[0].Value)
	assert.Equal(suite.T(), "2023-10-11T16:04:00Z", txs[0].BlockTime.Format(time.RFC3339))
}

func (suite *TxsSuite) TestETHTx() {
	block, err := parseBlock(suite.body)
	assert.Nil(suite.T(), err)
	to := "0x11aa6eeac7eae3c55b6fb9a4099adb5e420187ac"
	hash := "0x7f899903ddd10f184ee0074f5ecba96a4ba905882706a40c856d9e165ba90851"
	contract := strings.ToLower("0x779877A7B0D9E8603169DdbD7836e478b4624789")
	ctxt := common.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]common.ERC20{
			contract: {
				Asset:   "link",
				Address: contract,
				Scale:   18,
			},
		},
	}
	txs, err := filterTransactions(ctxt, *block)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(txs))
	assert.Equal(suite.T(), hash, txs[0].Hash)
	assert.Equal(suite.T(), block.Hash, txs[0].BlockHash)
	assert.Equal(suite.T(), block.Number, txs[0].BlockNumber)
	assert.Equal(suite.T(), "0.10000000", txs[0].Value)
	assert.Equal(suite.T(), "2023-10-11T16:04:00Z", txs[0].BlockTime.Format(time.RFC3339))
}

func TestTxsSuite(t *testing.T) {
	s := new(TxsSuite)
	s.path = "testdata/eth_blockNumber.json"
	suite.Run(t, s)
}
