package common

import (
	"fmt"
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

func (c Context) PriceBucket(tokens decimal.Decimal) decimal.Decimal {
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

type Transaction struct {
	Hash        string          `db:"tx_hash" json:"hash"`
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

func (fb *FinalizedBlock) TXMap() map[string]bool {
	txm := make(map[string]bool)
	for _, hash := range fb.Transactions {
		txm[hash] = true
	}
	return txm
}

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

	t.BlockHash = pd.BlockHash
	t.From = pd.From
	t.Hash = pd.Hash
	t.To = pd.To

	return nil
}

func GetContext(erc20Json, saleParamJson string) (*Context, error) {
	var (
		err    error
		result Context
	)

	result.StableCoins, err = getStableCoins(erc20Json)
	if err != nil {
		return nil, err
	}
	result.SaleParams, err = getSaleParams(saleParamJson)
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

func getSaleParams(saleParamJson string) ([]SaleParam, error) {
	var sps []SaleParam
	err := json.Unmarshal([]byte(saleParamJson), &sps)
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
