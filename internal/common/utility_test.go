package common

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestConvertValue(t *testing.T) {
	_, err := HexStringToDecimal("not a number")
	assert.NotNil(t, err, "Expected an error")

	a, err := HexStringToDecimal("0xa")
	assert.Nil(t, err)
	assert.Equal(t, decimal.NewFromInt(10), a)

	a, err = HexStringToDecimal("b")
	assert.Nil(t, err)
	assert.Equal(t, decimal.NewFromInt(11), a)

	d, err := decimal.NewFromString("20000000000000000000")
	assert.Nil(t, err)
	a, err = HexStringToDecimal("1158e460913d00000")
	assert.Nil(t, err)
	assert.True(t, a.Equal(d))
}
