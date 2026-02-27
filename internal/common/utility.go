package common

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var ethAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func IsValidETHAddress(address string) bool {
	return ethAddressRe.MatchString(address)
}

func HexStringToDecimal(hexValue string) (decimal.Decimal, error) {
	// Remove "0x" prefix if present
	hexValue = strings.TrimPrefix(hexValue, "0x")
	hexValue = strings.ToLower(hexValue)
	bi := new(big.Int)
	_, result := bi.SetString(hexValue, 16)

	if !result {
		err := fmt.Errorf("failed to convert '%s' to big.Int", hexValue)
		return decimal.Zero, err
	}
	return decimal.NewFromBigInt(bi, 0), nil
}
