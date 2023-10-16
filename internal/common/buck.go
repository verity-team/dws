package common

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

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
	SaleParams    []SaleParam
	StableCoins   map[string]ERC20
	ReceivingAddr string
	ETHRPCURL     string
	DB            *sqlx.DB
}

type Block struct {
	Hash         string        `db:"block_hash" json:"hash"`
	Number       uint64        `db:"block_number" json:"-"`
	Timestamp    time.Time     `db:"block_time" json:"-"`
	Transactions []Transaction `db:"-" json:"transactions"`
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
		err = fmt.Errorf("error decoding erc-20 JSON data, %v", err)
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
		err = fmt.Errorf("error decoding sale param JSON data, %v", err)
		log.Error(err)
		return nil, err
	}
	// sort in ascending order
	sort.Slice(sps, func(i, j int) bool {
		return sps[i].Limit < sps[j].Limit
	})
	return sps, nil
}
