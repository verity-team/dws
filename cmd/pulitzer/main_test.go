package main

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// a panicking price source must not take the process down; it is treated
// like any other failed source
func TestFetchPricePanicIsContained(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	assert.NotPanics(t, func() {
		fetchPrice("panicking", func() (decimal.Decimal, error) {
			var prices []decimal.Decimal
			return prices[0], nil
		}, ch)
	})
	_, ok := <-ch
	assert.False(t, ok, "expected the channel of a panicking source to be closed without a price")
}

func TestFetchPriceError(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	fetchPrice("failing", func() (decimal.Decimal, error) {
		return decimal.Zero, errors.New("boom")
	}, ch)
	_, ok := <-ch
	assert.False(t, ok, "expected the channel of a failing source to be closed without a price")
}

func TestFetchPriceSuccess(t *testing.T) {
	ch := make(chan decimal.Decimal, 1)
	fetchPrice("working", func() (decimal.Decimal, error) {
		return decimal.RequireFromString("2000.50"), nil
	}, ch)
	price, ok := <-ch
	assert.True(t, ok)
	assert.True(t, decimal.RequireFromString("2000.50").Equal(price))
	_, ok = <-ch
	assert.False(t, ok, "expected the channel to be closed after the price was sent")
}
