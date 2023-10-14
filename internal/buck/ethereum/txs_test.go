package ethereum

import (
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (suite *TxsSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(suite.path)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.path, err)
	}
}

func (suite *TxsSuite) TestLatestBlockSuccess() {
	txs, err := parseTransactions(suite.body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 200, len(txs))
}

func (suite *TxsSuite) TestLinkTxSuccess() {
	txs, err := parseTransactions(suite.body)
	assert.Nil(suite.T(), err)
	input := "0xa9059cbb000000000000000000000000ded1fe6b3f61c8f1d874bb86f086d10ffc3f015400000000000000000000000000000000000000000000000010cac896d2390000"
	from := "0x379738c60f658601Be79e267e79cC38cEA07c8f2"
	to := "0x779877A7B0D9E8603169DdbD7836e478b4624789"
	txHash := "0xf270a01e1ffa619b5262df30dc93d5ea1cf4bff773d6494460a1755abae43989"
	assert.Equal(suite.T(), input, txs[77].Input)
	assert.Equal(suite.T(), strings.ToLower(from), strings.ToLower(txs[77].From))
	assert.Equal(suite.T(), strings.ToLower(to), strings.ToLower(txs[77].To))
	assert.Equal(suite.T(), strings.ToLower(txHash), strings.ToLower(txs[77].Hash))
}

func (suite *TxsSuite) TestInputData() {
	input := "0xa9059cbb000000000000000000000000ded1fe6b3f61c8f1d874bb86f086d10ffc3f015400000000000000000000000000000000000000000000000010cac896d2390000"
	recipient, amount, err := parseInputData(input)
	assert.Nil(suite.T(), err)
	to := "0xDEd1Fe6B3f61c8F1d874bb86F086D10FFc3F0154"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromInt(1210000000000000000), amount)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTxsSuite(t *testing.T) {
	s := new(TxsSuite)
	s.path = "testdata/eth_blockNumber.json"
	suite.Run(t, s)
}
