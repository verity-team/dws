package data

import (
	"fmt"
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
