package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

func HexStringToDecimal(hexValue string) (decimal.Decimal, error) {
	var (
		amount int64
		err    error
	)
	hexValue = strings.ToLower(hexValue)
	if strings.HasPrefix(hexValue, "0x") {
		amount, err = strconv.ParseInt(hexValue, 0, 64)
	} else {
		amount, err = strconv.ParseInt(hexValue, 16, 64)
	}
	if err != nil {
		err := fmt.Errorf("failed to convert '%s' to int64, %v", hexValue, err)
		return decimal.Zero, err
	}
	return decimal.NewFromInt(amount), nil
}
