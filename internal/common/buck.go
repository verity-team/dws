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
	SaleParams       []SaleParam
	StableCoins      map[string]ERC20
	ReceivingAddr    string
	ETHRPCURL        string
	DB               *sqlx.DB
	BlockStorage     string
	UpdateLastBlock  bool
	MaxWaitInSeconds int
	ABI              map[string]abi.ABI
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
	// sort in ascending order
	sort.Slice(sps, func(i, j int) bool {
		return sps[i].Limit < sps[j].Limit
	})
	return sps, nil
}
