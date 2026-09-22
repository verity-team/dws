package ethereum

import (
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/verity-team/dws/api"
	c "github.com/verity-team/dws/internal/common"
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

func (suite *TxsSuite) TestInputDataUSDTLong() {
	input := "0xa9059cbb00000000000000000000000099870de8ae594e6e8705fc6689e89b4d039af1e2000000000000000000000000000000000000000000000000000000001bf6b3a295c343657f5eb02e5b549f80caa5e270c71972aded06826284169bf492b5c658"
	recipient, amount, err := parseInputData(suite.abi, "usdt", input)
	assert.Nil(suite.T(), err)
	to := "0x99870DE8AE594e6e8705fc6689E89B4d039AF1e2"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(469152674), 0), amount)
}

func (suite *TxsSuite) TestInputDataUSDTShort() {
	input := "0xa9059cbb0000000000000000000000000d0707963952f2fba59dd06f2b425ace40b492fe00000000000000000000000000000000000000000000000000000000176ad094"
	recipient, amount, err := parseInputData(suite.abi, "usdt", input)
	assert.Nil(suite.T(), err)
	to := "0x0D0707963952f2fBA59dD06f2b425ace40b492Fe"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(392876180), 0), amount)
}

func (suite *TxsSuite) TestInputDataUSDCShort() {
	input := "0xa9059cbb0000000000000000000000004667a044543e7f1b7d3a4b88396e024be0e34f36000000000000000000000000000000000000000000000000000000000d691330"
	recipient, amount, err := parseInputData(suite.abi, "usdc", input)
	assert.Nil(suite.T(), err)
	to := "0x4667A044543e7f1B7D3a4b88396e024BE0E34F36"
	assert.Equal(suite.T(), strings.ToLower(recipient), strings.ToLower(to))
	assert.Equal(suite.T(), decimal.NewFromBigInt(big.NewInt(224990000), 0), amount)
}

func (suite *TxsSuite) TestERC20Tx() {
	block, err := parseBlock(suite.body, testBlockNumber)
	assert.Nil(suite.T(), err)
	to := "0x4667A044543e7f1B7D3a4b88396e024BE0E34F36"
	contract := strings.ToLower("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	hash := "0x2bc8ef53d6a20b91e5dd3b856d78f5ed0aeb4f56232e6bf6de3783c6c58c13de"
	ctxt := c.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]c.ERC20{
			contract: {
				Asset:   "usdc",
				Address: contract,
				Scale:   6,
			},
		},
	}
	ctxt.ABI, err = InitABI()
	assert.Nil(suite.T(), err)
	txs, err := filterTransactions(ctxt, *block)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(txs))
	assert.Equal(suite.T(), hash, txs[0].Hash)
	assert.Equal(suite.T(), block.Hash, txs[0].BlockHash)
	assert.Equal(suite.T(), block.Number, txs[0].BlockNumber)
	// rendered at the scale of the `donation.amount` column rather than at a
	// hardcoded 6; for a 6 decimal stable coin the two render the same value,
	// see TestERC20RenderingIsValuePreserving
	assert.Equal(suite.T(), "224.9900000000", txs[0].Value)
	assert.True(suite.T(), decimal.RequireFromString("224.990000").Equal(decimal.RequireFromString(txs[0].Value)))
	assert.Equal(suite.T(), "2023-10-15T00:15:59Z", txs[0].BlockTime.Format(time.RFC3339))
}

func (suite *TxsSuite) TestETHTx() {
	block, err := parseBlock(suite.body, testBlockNumber)
	assert.Nil(suite.T(), err)
	to := "0x2051f9d1082008924f751eb396df1101d4b123e1"
	hash := "0x0aec48263d9ef216779aac6210c665723519251fbcb2b2d73cbb364c1b10f56d"
	contract := strings.ToLower("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	ctxt := c.Context{
		ReceivingAddr: strings.ToLower(to),
		StableCoins: map[string]c.ERC20{
			contract: {
				Asset:   "usdc",
				Address: contract,
				Scale:   6,
			},
		},
	}
	ctxt.ABI, err = InitABI()
	assert.Nil(suite.T(), err)
	txs, err := filterTransactions(ctxt, *block)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(txs))
	assert.Equal(suite.T(), hash, txs[0].Hash)
	assert.Equal(suite.T(), block.Hash, txs[0].BlockHash)
	assert.Equal(suite.T(), block.Number, txs[0].BlockNumber)
	assert.Equal(suite.T(), "0.00003592", txs[0].Value)
	assert.Equal(suite.T(), "2023-10-15T00:15:59Z", txs[0].BlockTime.Format(time.RFC3339))
}

// jsonrpc API providers may emit checksummed addresses and transaction
// hashes; the repo convention is lower case and every read path lowercases,
// so both have to be normalized before they reach the database
func (suite *TxsSuite) TestMixedCaseTxIsNormalized() {
	const (
		ethHash   = "0x0AEC48263D9EF216779AAC6210C665723519251FBCB2B2D73CBB364C1B10F56D"
		erc20Hash = "0x2BC8EF53D6A20B91E5DD3B856D78F5ED0AEB4F56232E6BF6DE3783C6C58C13DE"
		from      = "0xB938F65DfE303EdF96A511F1e7E3190f69036860"
		receiver  = "0x4667A044543e7f1B7D3a4b88396e024BE0E34F36"
		contract  = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
		// erc-20 transfer of 224.99 USDC to `receiver`
		erc20Input = "0xa9059cbb0000000000000000000000004667a044543e7f1b7d3a4b88396e024be0e34f36000000000000000000000000000000000000000000000000000000000d691330"
	)
	ctxt := c.Context{
		ABI:           suite.abi,
		ReceivingAddr: strings.ToLower(receiver),
		StableCoins: map[string]c.ERC20{
			strings.ToLower(contract): {
				Asset:   "usdc",
				Address: strings.ToLower(contract),
				Scale:   6,
			},
		},
	}
	block := c.Block{
		Hash:      "0x7d5b2b8e8d09ee1e3c1b0e1b0e1b0e1b0e1b0e1b0e1b0e1b0e1b0e1b0e1b0e1b",
		Number:    18352138,
		Timestamp: time.Unix(1697328959, 0).UTC(),
		Transactions: []c.Transaction{
			{
				TXH:   c.TXH{Hash: ethHash},
				From:  from,
				To:    receiver,
				Value: "0x2386f26fc10000",
				Input: "0x",
			},
			{
				TXH:   c.TXH{Hash: erc20Hash},
				From:  from,
				To:    contract,
				Value: "0x0",
				Input: erc20Input,
			},
		},
	}

	txs, err := filterTransactions(ctxt, block)
	suite.Require().Nil(err)
	suite.Require().Equal(2, len(txs))
	for _, tx := range txs {
		assert.Equal(suite.T(), strings.ToLower(tx.Hash), tx.Hash, "tx hash is not lower case")
		assert.Equal(suite.T(), strings.ToLower(from), tx.From, "donor address is not lower case")
	}
	assert.Equal(suite.T(), strings.ToLower(ethHash), txs[0].Hash)
	assert.Equal(suite.T(), "0.01000000", txs[0].Value)
	assert.Equal(suite.T(), strings.ToLower(erc20Hash), txs[1].Hash)
	assert.Equal(suite.T(), "224.9900000000", txs[1].Value)
	assert.True(suite.T(), decimal.RequireFromString("224.990000").Equal(decimal.RequireFromString(txs[1].Value)))
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

type ApplyTxReceiptsSuite struct {
	suite.Suite
	rcpts []c.TxReceipt
}

const (
	testRcptHash1 = "0xf270a01e1ffa619b5262df30dc93d5ea1cf4bff773d6494460a1755abae43989"
	testRcptHash2 = "0xa3f8edb39ec6e8d81c1859c53d5073f3ca22116f0cd23add11cf4b6c632b4633"
)

func (suite *ApplyTxReceiptsSuite) SetupTest() {
	body, err := os.ReadFile("testdata/eth_getTransactionReceipt.json")
	if err != nil {
		suite.Failf("failed to read test input", "%v", err)
	}
	suite.rcpts, err = parseTxReceipt(body)
	if err != nil {
		suite.Failf("failed to parse test input", "%v", err)
	}
	suite.Require().Equal(2, len(suite.rcpts))
}

func testTxs(hashes ...string) []c.Transaction {
	txs := make([]c.Transaction, 0, len(hashes))
	for _, h := range hashes {
		txs = append(txs, c.Transaction{
			TXH:    c.TXH{Hash: h},
			Status: string(api.Unconfirmed),
		})
	}
	return txs
}

// a truncated batch response must yield an error instead of panicking
func (suite *ApplyTxReceiptsSuite) TestTruncatedReceiptBatch() {
	txs := testTxs(testRcptHash1, testRcptHash2)
	var err error
	assert.NotPanics(suite.T(), func() {
		err = applyTxReceipts(4404251, txs, suite.rcpts[:1])
	})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), testRcptHash2)
}

// an empty batch response must yield an error instead of panicking
func (suite *ApplyTxReceiptsSuite) TestNoReceipts() {
	txs := testTxs(testRcptHash1)
	var err error
	assert.NotPanics(suite.T(), func() {
		err = applyTxReceipts(4404251, txs, nil)
	})
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), string(api.Unconfirmed), txs[0].Status)
}

// JSON-RPC 2.0 permits batch responses in any order
func (suite *ApplyTxReceiptsSuite) TestOutOfOrderReceipts() {
	txs := testTxs(testRcptHash1, testRcptHash2)
	rcpts := []c.TxReceipt{suite.rcpts[1], suite.rcpts[0]}
	// the receipt for the second tx reports failure
	rcpts[0].Status = "0x0"
	err := applyTxReceipts(4404251, txs, rcpts)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), string(api.Unconfirmed), txs[0].Status)
	assert.Equal(suite.T(), string(api.Failed), txs[1].Status)
}

// providers may return hashes in either casing
func (suite *ApplyTxReceiptsSuite) TestMixedCaseReceiptHash() {
	txs := testTxs(testRcptHash1)
	rcpts := []c.TxReceipt{suite.rcpts[0]}
	rcpts[0].TransactionHash = strings.ToUpper(rcpts[0].TransactionHash)
	err := applyTxReceipts(4404251, txs, rcpts)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), string(api.Unconfirmed), txs[0].Status)
}

// a 'null' result or an error object for a sub-request decodes into a
// zero value receipt -- it must not be correlated with any tx
func (suite *ApplyTxReceiptsSuite) TestNullAndErrorResults() {
	body, err := os.ReadFile("testdata/eth_getTransactionReceipt_partial.json")
	assert.Nil(suite.T(), err)
	rcpts, err := parseTxReceipt(body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 3, len(rcpts))

	txs := testTxs(testRcptHash1, testRcptHash2)
	assert.NotPanics(suite.T(), func() {
		err = applyTxReceipts(4404251, txs, rcpts)
	})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), testRcptHash2)

	// the tx that does have a receipt is processed normally
	txs = testTxs(testRcptHash1)
	err = applyTxReceipts(4404251, txs, rcpts)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), string(api.Unconfirmed), txs[0].Status)
}

func (suite *ApplyTxReceiptsSuite) TestNoTransactions() {
	err := applyTxReceipts(4404251, nil, nil)
	assert.Nil(suite.T(), err)
}

func TestApplyTxReceiptsSuite(t *testing.T) {
	suite.Run(t, new(ApplyTxReceiptsSuite))
}

// failed txs must be persisted -- otherwise we are losing them altogether.
// The ETH branch has always returned an error when the failed_tx write did
// not go through; the ERC-20 branch only logged it and carried on, so a brief
// postgres outage left the donation in neither `donation` nor `failed_tx`
// while the block was advanced past it.
func TestERC20FailedTxMustBePersisted(t *testing.T) {
	const (
		contract = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		receiver = "0x2051f9d1082008924f751eb396df1101d4b123e1"
		hash     = "0x2bc8ef53d6a20b91e5dd3b856d78f5ed0aeb4f56232e6bf6de3783c6c58c13de"
		// a transfer() selector with an input too short to unpack, so
		// parseInputData fails and the transaction has to be recorded as
		// failed
		badInput = "0xa9059cbbdeadbeef"
	)

	abis, err := InitABI()
	require.NoError(t, err)

	ctxt := c.Context{
		ReceivingAddr: receiver,
		StableCoins: map[string]c.ERC20{
			contract: {Asset: "usdc", Address: contract, Scale: 6},
		},
		ABI: abis,
		DB:  closedDB(t),
	}
	block := c.Block{
		Hash:      "0xbbbb111111111111111111111111111111111111111111111111111111111111",
		Number:    18_000_000,
		Timestamp: time.Unix(1697328959, 0).UTC(),
		Transactions: []c.Transaction{{
			TXH:   c.TXH{Hash: hash},
			From:  "0xcccc111111111111111111111111111111111111",
			To:    contract,
			Value: "0x0",
			Input: badInput,
		}},
	}

	txs, err := filterTransactions(ctxt, block)
	require.Error(t, err, "a failed_tx write that did not go through must not be swallowed")
	assert.Contains(t, err.Error(), "failed to persist failed tx")
	assert.Nil(t, txs)
}

// closedDB hands out a database handle whose every statement fails.
func closedDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dbh, err := sqlx.Open("postgres", "postgres://dws:dws@127.0.0.1:1/dwsdb?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	require.NoError(t, dbh.Close())
	return dbh
}

// The rendering scale used to be hardcoded at 6 while the `Shift` that
// precedes it is driven by the configured erc-20 scale. For the two coins
// configured today (USDC/USDT, scale 6) that is exactly value preserving, so
// moving to the scale of the `donation.amount` column cannot change a single
// stored amount: NUMERIC(20,10) normalizes both renderings to the same value.
//
// Confirmed in postgres 14 against the real column type:
//
//	SELECT '224.990000'::numeric(20,10) = '224.9900000000'::numeric(20,10);  -- t
func TestERC20RenderingIsValuePreservingForScale6(t *testing.T) {
	// the smallest unit, a representative donation and a large one
	for _, raw := range []string{"1", "224990000", "392876180", "999999999999999"} {
		amount := decimal.RequireFromString(raw)

		was, wasOK := amount.Shift(-6).StringFixed(6), amount.Shift(-6).IsPositive()
		is, isOK := recordedAmount(amount, 6, DonationAmountDecimals)

		require.True(t, wasOK, raw)
		require.True(t, isOK, raw)
		assert.True(t, decimal.RequireFromString(was).Equal(decimal.RequireFromString(is)),
			"raw %s: old rendering %s, new rendering %s", raw, was, is)
		// and the new rendering is exact, i.e. postgres has nothing left to
		// round when it stores it
		assert.True(t, amount.Shift(-6).Equal(decimal.RequireFromString(is)), raw)
	}
}

// the scale is derived from the column, so a token whose own scale exceeds it
// is rounded once, by us, at exactly the precision the column keeps -- the
// hardcoded 6 used to round an 18 decimal coin up by as much as 1.1e-7
func TestERC20RenderingHonoursTheTokenScale(t *testing.T) {
	// 1.234567890123456789 DAI
	amount := decimal.RequireFromString("1234567890123456789")

	old := amount.Shift(-18).StringFixed(6)
	assert.Equal(t, "1.234568", old)
	// the hardcoded rendering credited *more* than was transferred
	assert.True(t, decimal.RequireFromString(old).GreaterThan(amount.Shift(-18)))

	got, ok := recordedAmount(amount, 18, DonationAmountDecimals)
	require.True(t, ok)
	assert.Equal(t, "1.2345678901", got)
	// at the scale of the column, the error is bounded by half a unit in the
	// last place the column can hold
	delta := decimal.RequireFromString(got).Sub(amount.Shift(-18)).Abs()
	assert.True(t, delta.LessThanOrEqual(decimal.RequireFromString("0.00000000005")), delta.String())
}

// ethereum donations keep their scale: raising it is not value preserving and
// would widen the set of dust transfers that record a non-zero amount
func TestETHRenderingScaleIsUnchanged(t *testing.T) {
	amount := decimal.RequireFromString("1234567890123456789")
	got, ok := recordedAmount(amount, WeiDecimals, ETHAmountDecimals)
	require.True(t, ok)
	assert.Equal(t, "1.23456789", got)
	assert.Equal(t, int32(8), int32(ETHAmountDecimals))
}

func TestRecordedAmountRejectsWhatWouldBeStoredAsZero(t *testing.T) {
	// ETH: zero, one wei, and everything below half a unit in the last place
	for _, raw := range []string{"0", "1", "1000000000", "4999999999"} {
		value, ok := recordedAmount(decimal.RequireFromString(raw), WeiDecimals, ETHAmountDecimals)
		assert.False(t, ok, "%s wei must not be recorded", raw)
		assert.True(t, decimal.RequireFromString(value).IsZero(), value)
	}
	// the smallest ETH amount that does record something
	value, ok := recordedAmount(decimal.RequireFromString("5000000000"), WeiDecimals, ETHAmountDecimals)
	assert.True(t, ok)
	assert.Equal(t, "0.00000001", value)

	// erc-20: zero is rejected, one smallest unit of a 6 decimal coin is not
	_, ok = recordedAmount(decimal.Zero, 6, DonationAmountDecimals)
	assert.False(t, ok)
	value, ok = recordedAmount(decimal.RequireFromString("1"), 6, DonationAmountDecimals)
	assert.True(t, ok)
	assert.Equal(t, "0.0000010000", value)
}

// a zero value ETH transfer to the receiving address used to yield a donation
// row with amount 0, usd_amount 0 and 0 tokens for the price of 21000 gas.
// Rejecting `value: "0x0"` alone would have bought nothing -- one wei costs
// the same and renders as 0.00000000 just the same -- so both are turned away.
func TestZeroValueETHTransferIsNotADonation(t *testing.T) {
	const receiver = "0x2051f9d1082008924f751eb396df1101d4b123e1"

	ctxt := c.Context{ReceivingAddr: receiver}
	var err error
	ctxt.ABI, err = InitABI()
	require.NoError(t, err)

	for _, value := range []string{"0x0", "0x1", "0x12a05f1ff"} { // 0 wei, 1 wei, 4999999999 wei (half a unit short)
		block := c.Block{
			Hash:      "0xbbbb111111111111111111111111111111111111111111111111111111111111",
			Number:    18_000_000,
			Timestamp: time.Unix(1697328959, 0).UTC(),
			Transactions: []c.Transaction{{
				TXH:   c.TXH{Hash: "0xaaaa111111111111111111111111111111111111111111111111111111111111"},
				From:  "0xcccc111111111111111111111111111111111111",
				To:    receiver,
				Value: value,
				Input: "0x",
			}},
		}
		txs, err := filterTransactions(ctxt, block)
		require.NoError(t, err)
		assert.Empty(t, txs, "value %s must not produce a donation", value)
	}

	// a transfer that does carry value is still accepted
	block := c.Block{
		Hash:      "0xbbbb111111111111111111111111111111111111111111111111111111111111",
		Number:    18_000_000,
		Timestamp: time.Unix(1697328959, 0).UTC(),
		Transactions: []c.Transaction{{
			TXH:   c.TXH{Hash: "0xaaaa111111111111111111111111111111111111111111111111111111111111"},
			From:  "0xcccc111111111111111111111111111111111111",
			To:    receiver,
			Value: "0x2386f26fc10000", // 0.01 ETH
			Input: "0x",
		}},
	}
	txs, err := filterTransactions(ctxt, block)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "0.01000000", txs[0].Value)
}

// the same rule applies to a stable coin transfer of zero
func TestZeroValueERC20TransferIsNotADonation(t *testing.T) {
	const (
		contract = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		receiver = "0x4667a044543e7f1b7d3a4b88396e024be0e34f36"
		// transfer(0x4667...F36, 0)
		zeroTransfer = "0xa9059cbb0000000000000000000000004667a044543e7f1b7d3a4b88396e024be0e34f360000000000000000000000000000000000000000000000000000000000000000"
	)

	abis, err := InitABI()
	require.NoError(t, err)

	ctxt := c.Context{
		ReceivingAddr: receiver,
		StableCoins: map[string]c.ERC20{
			contract: {Asset: "usdc", Address: contract, Scale: 6},
		},
		ABI: abis,
	}
	block := c.Block{
		Hash:      "0xbbbb111111111111111111111111111111111111111111111111111111111111",
		Number:    18_000_000,
		Timestamp: time.Unix(1697328959, 0).UTC(),
		Transactions: []c.Transaction{{
			TXH:   c.TXH{Hash: "0xaaaa222222222222222222222222222222222222222222222222222222222222"},
			From:  "0xcccc111111111111111111111111111111111111",
			To:    contract,
			Value: "0x0",
			Input: zeroTransfer,
		}},
	}

	txs, err := filterTransactions(ctxt, block)
	require.NoError(t, err)
	assert.Empty(t, txs)
}
