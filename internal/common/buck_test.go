package common

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

const spJSON = `
[{"limit": 1050000000, "price": "0.001"}, {"limit": 2625000000, "price": "0.002"}, {"limit": 5250000000, "price": "0.0024"}]
`

func TestPriceBucket(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	a := c.priceBucket(decimal.NewFromInt(int64(0)))
	assert.Equal(t, decimal.NewFromFloat(0.001), a)
	a = c.priceBucket(decimal.NewFromInt(int64(1050000000)))
	assert.Equal(t, decimal.NewFromFloat(0.002), a)
	a = c.priceBucket(decimal.NewFromInt(int64(2625000000)))
	assert.Equal(t, decimal.NewFromFloat(0.0024), a)
	a = c.priceBucket(decimal.NewFromInt(int64(5250000000)))
	assert.Equal(t, decimal.NewFromFloat(0.0024), a)
}

func TestTokenSaleLimit(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	a := c.TokenSaleLimit()
	assert.Equal(t, decimal.NewFromInt(int64(5250000000)), a)
}

func TestNewTokenPrice(t *testing.T) {
	var (
		c   Context
		err error
	)
	c.SaleParams, err = getSaleParams(spJSON)
	assert.Nil(t, err)

	af, np := c.NewTokenPrice(decimal.NewFromInt(int64(0)), decimal.NewFromInt(int64(0)))
	assert.False(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.001), np)
	af, np = c.NewTokenPrice(decimal.NewFromInt(int64(0)), decimal.NewFromInt(int64(1049999999)))
	assert.False(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.001), np)
	af, np = c.NewTokenPrice(decimal.NewFromInt(int64(1049999999)), decimal.NewFromInt(int64(1050000000)))
	assert.True(t, af)
	assert.Equal(t, decimal.NewFromFloat(0.002), np)
}
