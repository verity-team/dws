package ethereum

import (
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/verity-team/dws/internal/common"
)

type TxsSuite struct {
	suite.Suite
	body []byte
	path string
	abi  map[string]abi.ABI
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
	hash := "0xfd7724ea905f528af6466ff6229630ec1d7bd4d9df21cbd089f9d67337dfd367"
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 99, len(block.Transactions))
	assert.Equal(suite.T(), hash, block.Hash)
	assert.Equal(suite.T(), uint64(18352138), block.Number)
	assert.Equal(suite.T(), "2023-10-15T00:15:59Z", block.Timestamp.Format(time.RFC3339))
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

func (suite *TxsSuite) TestInputDataUSDTLong() {
	input := "0xa9059cbb00000000000000000000000099870de8ae594e6e8705fc6689e89b4d039af1e2000000000000000000000000000000000000000000000000000000001bf6b3a295c343657f5eb02e5b549f80caa5e270c71972aded06826284169bf492b5c658"
	recipient, amount, err := parseInputData(suite.abi["usdt"], input)
	assert.Nil(suite.T(), err)
	to := "0x99870DE8AE594e6e8705fc6689E89B4d039AF1e2"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(469152674), 1), amount)
}

func (suite *TxsSuite) TestInputDataUSDTShort() {
	input := "0xa9059cbb0000000000000000000000000d0707963952f2fba59dd06f2b425ace40b492fe00000000000000000000000000000000000000000000000000000000176ad094"
	recipient, amount, err := parseInputData(suite.abi["usdt"], input)
	assert.Nil(suite.T(), err)
	to := "0x0D0707963952f2fBA59dD06f2b425ace40b492Fe"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(392876180), 1), amount)
}

func (suite *TxsSuite) TestInputDataUSDCShort() {
	input := "0xa9059cbb0000000000000000000000004667a044543e7f1b7d3a4b88396e024be0e34f36000000000000000000000000000000000000000000000000000000000d691330"
	recipient, amount, err := parseInputData(suite.abi["usdc"], input)
	assert.Nil(suite.T(), err)
	to := "0x4667A044543e7f1B7D3a4b88396e024BE0E34F36"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(224990000), 1), amount)
}

func (suite *TxsSuite) TestERC20Tx() {
	block, err := parseBlock(suite.body)
	assert.Nil(suite.T(), err)
	to := "0x4667A044543e7f1B7D3a4b88396e024BE0E34F36"
	contract := strings.ToLower("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	hash := "0x2bc8ef53d6a20b91e5dd3b856d78f5ed0aeb4f56232e6bf6de3783c6c58c13de"
	ctxt := common.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]common.ERC20{
			contract: {
				Asset:   "usdc",
				Address: contract,
				Scale:   6,
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
	to := "0x2051f9d1082008924f751eb396df1101d4b123e1"
	hash := "0x0aec48263d9ef216779aac6210c665723519251fbcb2b2d73cbb364c1b10f56d"
	contract := strings.ToLower("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	ctxt := common.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]common.ERC20{
			contract: {
				Asset:   "usdc",
				Address: contract,
				Scale:   6,
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
	s.path = "testdata/18352138.json"
	var err error
	s.abi, err = InitABI()
	if err != nil {
		t.Fatalf("failed to init ABI, %s", err)
	}
	suite.Run(t, s)
}
