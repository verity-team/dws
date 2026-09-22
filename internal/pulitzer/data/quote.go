package data

import (
	"fmt"
	"strings"
	"time"
)

const (
	// MaxQuoteAge bounds how old a venue's own timestamp for a quote may be.
	// A cached quote tens of minutes old is usually still well within the
	// deviation gate around spot and would otherwise be averaged into the
	// price this minute at full weight.
	MaxQuoteAge = 2 * time.Minute
	// MaxQuoteSkew is how far a venue timestamp may lie in the future before
	// the quote is rejected; it only covers clock drift between the venue and
	// us.
	MaxQuoteSkew = 30 * time.Second
)

// checkQuoteAge rejects a quote whose venue timestamp is too old -- or so far
// in the future that the timestamp cannot be trusted at all.
//
// It is only applied to the sources that publish a timestamp for the quote
// (cex.io and kucoin). The remaining venues do not expose one, so the only
// freshness bound available there is the request itself: the price is fetched
// over a fresh HTTP request in every cycle, with the client timeout in
// common.HTTPGet as the upper bound.
func checkQuoteAge(source string, ts time.Time) error {
	if ts.IsZero() {
		return fmt.Errorf("%s: no quote timestamp in the response", source)
	}
	age := time.Since(ts.UTC())
	if age > MaxQuoteAge {
		return fmt.Errorf("%s: quote is %s old, the limit is %s", source, age.Truncate(time.Second), MaxQuoteAge)
	}
	if age < -MaxQuoteSkew {
		return fmt.Errorf("%s: quote timestamp is %s in the future", source, (-age).Truncate(time.Second))
	}
	return nil
}

// checkPair rejects a response that is not for the pair that was requested.
//
// Every venue echoes the instrument it answered for and none of the sources
// used to look at it: a pair that is renamed, re-listed or simply mistyped in
// the request would have been averaged into the ethereum price as if it were
// ETH. The comparison ignores the separator and the casing -- venues spell the
// same pair `ETHUSD`, `ETH-USD`, `ETH/USD` and `ETH:USD` -- so it only fires
// on a genuinely different instrument.
func checkPair(source, want, got string) error {
	if got == "" {
		return fmt.Errorf("%s: no pair in the response, expected '%s'", source, want)
	}
	if normalizePair(want) != normalizePair(got) {
		return fmt.Errorf("%s: response is for pair '%s', expected '%s'", source, got, want)
	}
	return nil
}

// pairSeparators are the characters venues use to separate base from quote.
var pairSeparators = strings.NewReplacer("-", "", "/", "", ":", "", "_", "", " ", "")

// normalizePair reduces a pair identifier to base+quote in upper case.
func normalizePair(pair string) string {
	return strings.ToUpper(pairSeparators.Replace(pair))
}
