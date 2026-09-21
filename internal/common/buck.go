package common

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/goccy/go-json"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type CrawlerType int

const (
	Latest CrawlerType = iota
	Finalized
	OldUnconfirmed
)

func (ct CrawlerType) String() string {
	switch ct {
	case Latest:
		return "latest"
	case Finalized:
		return "finalized"
	case OldUnconfirmed:
		return "old-unconfirmed"
	}
	return "invalid crawler type"
}

type SaleParam struct {
	Limit int
	Price decimal.Decimal
}

type ERC20 struct {
	Asset   string
	Address string
	Scale   int32
}

type Hashable interface {
	GetHash() string
}

func ToHashable[H Hashable](hs []H) []Hashable {
	var res []Hashable
	for _, h := range hs {
		res = append(res, h)
	}
	return res
}

type Fetchable interface {
	TxReceipt | TxByHash
}

type Fetcher[R Fetchable] interface {
	Fetch(Context, []Hashable) ([]R, error)
}

type Context struct {
	ABI              map[string]abi.ABI
	BlockCache       string
	CrawlerType      CrawlerType
	DB               *sqlx.DB
	DebugDataStore   string
	ETHRPCURL        string
	MaxWaitInSeconds int
	ReceivingAddr    string
	SaleParams       []SaleParam
	StableCoins      map[string]ERC20
	UpdateLastBlock  bool
}

func (c Context) priceBucket(tokens decimal.Decimal) decimal.Decimal {
	// find the correct price given the number of tokens sold
	// please note: the sales params slice is sorted in ascending order,
	// based on the limit property
	for _, sp := range c.SaleParams {
		if tokens.LessThan(decimal.NewFromInt(int64(sp.Limit))) {
			return sp.Price
		}
	}
	// we fell through the loop, return the max price
	return c.SaleParams[len(c.SaleParams)-1].Price
}

func (c Context) TokenSaleLimit() decimal.Decimal {
	// sales params slice is sorted
	max := c.SaleParams[len(c.SaleParams)-1].Limit
	return decimal.NewFromInt(int64(max))
}

func (c Context) NewTokenPrice(oldTokens, newTokens decimal.Decimal) (bool, decimal.Decimal) {
	// did we enter a new price range? do we need to update the price?
	currentP := c.priceBucket(oldTokens)
	newP := c.priceBucket(newTokens)

	return newP.GreaterThan(currentP), newP
}

type Block struct {
	Hash         string        `db:"block_hash" json:"hash"`
	Number       uint64        `db:"block_number" json:"-"`
	Timestamp    time.Time     `db:"block_time" json:"-"`
	Transactions []Transaction `db:"-" json:"transactions"`
}

func (b *Block) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	type block struct {
		Hash         string        `json:"hash"`
		HexNumber    string        `json:"number"`
		HexSeconds   string        `json:"timestamp"`
		Transactions []Transaction `json:"transactions"`
	}

	var pb block
	if err := json.Unmarshal(data, &pb); err != nil {
		return err
	}

	seconds, err := HexStringToDecimal(pb.HexSeconds)
	if err != nil {
		err = fmt.Errorf("failed to convert block timestamp, %w", err)
		return err
	}
	ts := time.Unix(seconds.IntPart(), 0)
	b.Timestamp = ts.UTC()

	number, err := HexStringToDecimal(pb.HexNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %w", err)
		return err
	}
	b.Number = uint64(number.IntPart())

	b.Hash = pb.Hash
	b.Transactions = pb.Transactions

	return nil
}

type TxReceipt struct {
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	EffectiveGasPrice string `json:"effectiveGasPrice"`
	From              string `json:"from"`
	GasUsed           string `json:"gasUsed"`
	Status            string `json:"status"`
	To                string `json:"to"`
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	Type              string `json:"type"`
}

type TXH struct {
	Hash string `db:"tx_hash" json:"hash"`
}

func (tx TXH) GetHash() string {
	return tx.Hash
}

type Transaction struct {
	TXH
	From        string          `db:"address" json:"from"`
	To          string          `db:"-" json:"to"`
	Value       string          `db:"amount" json:"value"`
	Gas         string          `db:"-" json:"gas"`
	Nonce       string          `db:"-" json:"nonce"`
	Input       string          `db:"-" json:"input"`
	Type        string          `db:"-" json:"type"`
	Status      string          `db:"status" json:"-"`
	Asset       string          `db:"asset" json:"-"`
	Price       string          `db:"price" json:"-"`
	Tokens      decimal.Decimal `db:"tokens" json:"-"`
	USDAmount   decimal.Decimal `db:"usd_amount" json:"-"`
	BlockNumber uint64          `db:"block_number" json:"-"`
	BlockHash   string          `db:"block_hash" json:"blockhash"`
	BlockTime   time.Time       `db:"block_time" json:"-"`
}

type FinalizedBlock struct {
	BaseFeePerGas string    `db:"base_fee_per_gas" json:"baseFeePerGas"`
	GasLimit      string    `db:"gas_limit" json:"gasLimit"`
	GasUsed       string    `db:"gas_used" json:"gasUsed"`
	Hash          string    `db:"block_hash" json:"hash"`
	Number        uint64    `db:"block_number" json:"-"`
	ReceiptsRoot  string    `db:"receipts_root" json:"receiptsRoot"`
	Size          string    `db:"block_size" json:"size"`
	StateRoot     string    `db:"state_root" json:"stateRoot"`
	Timestamp     time.Time `db:"block_time" json:"-"`
	Transactions  []string  `db:"transactions" json:"transactions"`
}

// TXMap returns the set of transaction hashes carried by the block, keyed by
// normalized hash -- look up with NormalizeHash(hash).
func (fb *FinalizedBlock) TXMap() map[string]bool {
	txm := make(map[string]bool, len(fb.Transactions))
	for _, hash := range fb.Transactions {
		txm[NormalizeHash(hash)] = true
	}
	return txm
}

const (
	// PendingBlockNumber is the block number of a transaction that is still
	// in the mempool and hence has no block. math.MaxUint64 is used as the
	// sentinel because it cannot be mistaken for a real ethereum block
	// number: it is not reachable in practice and it is *greater* than every
	// block number the chain will ever produce -- every "has this block been
	// finalized?" comparison is false for it.
	PendingBlockNumber uint64 = math.MaxUint64
	// PendingTransactionIndex is the position of a transaction that is still
	// in the mempool and hence has no position in a block. Same sentinel,
	// same reasoning: block position 0 is a real position, math.MaxUint64 is
	// not.
	PendingTransactionIndex uint64 = math.MaxUint64
)

type TxByHash struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      uint64 `json:"blockNumber"`
	From             string `json:"from"`
	Hash             string `json:"hash"`
	To               string `json:"to"`
	TransactionIndex uint64 `json:"transactionIndex"`
	FBBlockTime      time.Time
	FBBlockHash      string
	FBContainsTx     bool
	// FBDataAvailable indicates whether the data of the block that finalizes
	// this transaction could be fetched. If it is false the FB* fields above
	// are meaningless and no verdict may be derived from them.
	FBDataAvailable bool
	// Pending indicates that the transaction is still in the mempool: it has
	// no block (yet) and may still be mined.
	Pending bool
}

// TxVerdict is the outcome of judging an old unconfirmed transaction.
type TxVerdict int

const (
	// TxPending -- the transaction is still in the mempool, leave it alone.
	TxPending TxVerdict = iota
	// TxNotFinalizedYet -- the block containing the transaction has not been
	// finalized yet, leave it alone.
	TxNotFinalizedYet
	// TxNoFinalizedBlockData -- the block that finalizes the transaction
	// could not be fetched, leave it alone.
	TxNoFinalizedBlockData
	// TxFinalize -- the finalized block carries the transaction, confirm it.
	TxFinalize
	// TxFail -- the finalized block for the transaction's block number does
	// not carry it, fail it.
	TxFail
)

func (v TxVerdict) String() string {
	switch v {
	case TxPending:
		return "still in the mempool"
	case TxNotFinalizedYet:
		return "not finalized yet"
	case TxNoFinalizedBlockData:
		return "no finalized block data"
	case TxFinalize:
		return "finalize"
	case TxFail:
		return "fail"
	}
	return "invalid tx verdict"
}

// Judge decides what is to be done with an old unconfirmed transaction, given
// the number of the most recent finalized block (mfbn).
//
// A transaction is failed on positive evidence only: the finalized block for
// its block number exists and does *not* carry it (re-org, replacement).
// A pending transaction has no block at all and may still be mined, and a
// transaction whose finalized block could not be fetched was not looked at --
// absence of evidence is not evidence of failure. Both are left alone and are
// re-examined on the next run.
func (t TxByHash) Judge(mfbn uint64) TxVerdict {
	if t.Pending {
		return TxPending
	}
	if t.BlockNumber > mfbn {
		return TxNotFinalizedYet
	}
	if !t.FBDataAvailable {
		return TxNoFinalizedBlockData
	}
	// the finalized block hash must match the tx block hash and the finalized
	// block must actually contain the tx in question. Hashes are compared
	// normalized: a provider that reports a differently cased hash must not
	// cost a donor their donation.
	if NormalizeHash(t.BlockHash) != NormalizeHash(t.FBBlockHash) || !t.FBContainsTx {
		return TxFail
	}
	return TxFinalize
}

func (t *TxByHash) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	type tx struct {
		BlockHash        string `json:"blockHash"`
		BlockNumber      string `json:"blockNumber"`
		From             string `json:"from"`
		Hash             string `json:"hash"`
		To               string `json:"to"`
		TransactionIndex string `json:"transactionIndex"`
	}

	var pd tx
	if err := json.Unmarshal(data, &pd); err != nil {
		return err
	}

	t.BlockHash = pd.BlockHash
	t.From = pd.From
	t.Hash = pd.Hash
	t.To = pd.To

	// a transaction that is still in the mempool has no block: the jsonrpc
	// API provider reports `null` for its block number/hash and its position
	// in the block. That is a valid state, not a malformed response -- fail
	// the decode and a single pending transaction poisons the whole batch.
	if pd.BlockNumber == "" || pd.TransactionIndex == "" {
		t.Pending = true
		t.BlockNumber = PendingBlockNumber
		t.TransactionIndex = PendingTransactionIndex
		return nil
	}

	bn, err := HexStringToDecimal(pd.BlockNumber)
	if err != nil {
		err = fmt.Errorf("failed to convert block number, %w", err)
		return err
	}
	t.BlockNumber = uint64(bn.IntPart())

	tidx, err := HexStringToDecimal(pd.TransactionIndex)
	if err != nil {
		err = fmt.Errorf("failed to convert transaction index, %w", err)
		return err
	}
	t.TransactionIndex = uint64(tidx.IntPart())

	return nil
}

func GetContext(erc20Json, saleParamJSON string) (*Context, error) {
	var (
		err    error
		result Context
	)

	result.StableCoins, err = getStableCoins(erc20Json)
	if err != nil {
		return nil, err
	}
	result.SaleParams, err = getSaleParams(saleParamJSON)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func getStableCoins(erc20Json string) (map[string]ERC20, error) {
	var scs []ERC20
	err := json.Unmarshal([]byte(erc20Json), &scs)
	if err != nil {
		err = fmt.Errorf("error decoding erc-20 JSON data, %w", err)
		log.Error(err)
		return nil, err
	}

	result := make(map[string]ERC20, len(scs))
	for _, sc := range scs {
		addr := strings.ToLower(sc.Address)
		result[addr] = ERC20{
			Asset:   sc.Asset,
			Address: addr,
			Scale:   sc.Scale,
		}
	}
	return result, nil
}

func getSaleParams(saleParamJSON string) ([]SaleParam, error) {
	var sps []SaleParam
	err := json.Unmarshal([]byte(saleParamJSON), &sps)
	if err != nil {
		err = fmt.Errorf("error decoding sale param JSON data, %w", err)
		log.Error(err)
		return nil, err
	}
	// sort in ascending order, based on limit
	sort.Slice(sps, func(i, j int) bool {
		return sps[i].Limit < sps[j].Limit
	})
	return sps, nil
}
