package common

import (
	"fmt"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var ethAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func IsValidETHAddress(address string) bool {
	return ethAddressRe.MatchString(address)
}

// NormalizeHash returns the canonical form of a block/transaction hash.
// jsonrpc API providers return either casing (and the two endpoints of a
// single provider need not agree with each other), so hashes are compared --
// and stored -- lower case and free of surrounding whitespace, the repo
// convention. Transactions are normalized at the point of acceptance
// (filterTransactions), hence every hash in the database is in this form.
func NormalizeHash(hash string) string {
	return strings.ToLower(strings.TrimSpace(hash))
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

// maxInt64Decimal is the largest value decimal.Decimal.IntPart() can render
// faithfully; anything above it is silently truncated to the low 64 bits.
var maxInt64Decimal = decimal.NewFromInt(math.MaxInt64)

// ToUint64 converts a decimal to a uint64, refusing the two values
// decimal.IntPart() cannot represent.
//
// A plain uint64(d.IntPart()) is wrong in both directions: a negative value
// wraps around to ~1.8e19 instead of being rejected, and a value beyond the
// int64 range is truncated by IntPart() before the conversion is even reached.
// Either one would be fed to the crawler as a block number or a transaction
// index, i.e. as a position on the chain that does not exist. `what` names the
// quantity so that the error identifies the field that was out of range.
func ToUint64(d decimal.Decimal, what string) (uint64, error) {
	if d.IsNegative() {
		return 0, fmt.Errorf("negative %s ('%s')", what, d.String())
	}
	if d.GreaterThan(maxInt64Decimal) {
		return 0, fmt.Errorf("%s ('%s') exceeds %d", what, d.String(), int64(math.MaxInt64))
	}
	return uint64(d.IntPart()), nil
}

// RedactURL returns a form of rawURL that is safe to put into a log line or an
// error message: the scheme and the host, and a fixed marker in place of
// everything else.
//
// The credentials of a jsonrpc API provider live in the part that is dropped:
// ETH_RPC_URL is of the form https://<network>.infura.io/v3/<project id>, and
// other providers pass the key in the query string instead. GetDSN is already
// careful to log the host, the database and the sslmode but never the
// password; an endpoint URL carrying an API key is the same class of secret.
//
// A URL that cannot be parsed is replaced wholesale -- the parse failure says
// nothing about which part of it holds the secret.
func RedactURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return redactedMarker
	}
	if u.Path == "" && u.RawQuery == "" && u.User == nil && u.Fragment == "" {
		return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}
	return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, redactedMarker)
}

// redactedMarker stands in for the part of a URL that was withheld.
const redactedMarker = "<redacted>"
