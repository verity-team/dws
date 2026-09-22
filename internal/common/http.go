package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/gommon/log"
)

type HTTPParams struct {
	URL              string
	RequestBody      []byte
	MaxWaitInSeconds int
}

const MaxWaitInSeconds = 10

// MaxResponseBytes bounds how much of a response body is read into memory.
//
// Without it a malfunctioning -- or hostile -- endpoint can stream for as long
// as the client timeout allows and the process grows by whatever arrives in
// that window. The bound is generous: the largest body any caller legitimately
// receives is an `eth_getBlockByNumber` response carrying every transaction of
// a block, a few megabytes at most.
const MaxResponseBytes = 32 << 20

// MaxLoggedBodyBytes is how much of the body of a failed response is written
// to the log. A rejected request is answered with an error document and an
// intermediary (a WAF challenge page, a captive portal) answers with a full
// HTML page -- neither belongs in the log in full, once a minute, forever.
const MaxLoggedBodyBytes = 512

func timeout(params HTTPParams) time.Duration {
	if params.MaxWaitInSeconds <= 0 {
		return time.Duration(MaxWaitInSeconds) * time.Second
	}
	return time.Duration(params.MaxWaitInSeconds) * time.Second
}

func HTTPGet(params HTTPParams) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout(params),
	}

	req, err := http.NewRequest("GET", params.URL, nil)
	if err != nil {
		err = fmt.Errorf("failed to prep request for url ('%s'), %w", RedactURL(params.URL), redactTransportError(err))
		log.Error(err)
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute GET request for url ('%s'), %w", RedactURL(params.URL), redactTransportError(err))
		log.Error(err)
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	return readResponse("GET", params.URL, response)
}

// HTTPGetCtx is the context-aware sibling of HTTPGet: the request is bound to
// ctx, so a caller with a deadline (the historical backfill chain, which caps
// its whole Binance->Kraken->Coinbase walk with one budget) actually cancels
// an in-flight request when the deadline is spent, rather than only preventing
// the next call from starting.
//
// It is deliberately separate from HTTPGet so the live-price and RPC paths keep
// their exact behaviour. The per-request client timeout still applies as an
// upper bound alongside ctx -- whichever fires first wins -- and the redaction,
// status and bounded-read handling are shared with HTTPGet via readResponse and
// redactTransportError. A cancelled request surfaces as an ordinary fetch error
// (context.Canceled / context.DeadlineExceeded, with the URL redacted), which
// the chain treats like any other failed source.
func HTTPGetCtx(ctx context.Context, params HTTPParams) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout(params),
	}

	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		err = fmt.Errorf("failed to prep request for url ('%s'), %w", RedactURL(params.URL), redactTransportError(err))
		log.Error(err)
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute GET request for url ('%s'), %w", RedactURL(params.URL), redactTransportError(err))
		log.Error(err)
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	return readResponse("GET", params.URL, response)
}

func HTTPPost(params HTTPParams) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout(params),
	}
	response, err := client.Post(params.URL, "application/json", bytes.NewBuffer(params.RequestBody))
	if err != nil {
		err = fmt.Errorf("post request for url ('%s') failed, %w", RedactURL(params.URL), redactTransportError(err))
		log.Error(err)
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	return readResponse("POST", params.URL, response)
}

// readResponse turns an HTTP response into the body the caller asked for.
//
// The status is checked *before* the body is read: a response that is not a
// 200 carries nothing any caller can use, so at most MaxLoggedBodyBytes of it
// are read -- enough to identify what answered -- instead of pulling an
// arbitrarily large error document into memory and writing it to the log in
// full on every cycle.
func readResponse(method, rawURL string, response *http.Response) ([]byte, error) {
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("%d status code for %s request with url ('%s')", response.StatusCode, method, RedactURL(rawURL))
		log.Error(err)
		if excerpt, rerr := io.ReadAll(io.LimitReader(response.Body, MaxLoggedBodyBytes)); rerr == nil && len(excerpt) > 0 {
			log.Infof("response excerpt: '%s'", string(excerpt))
		}
		return nil, err
	}

	responseBody, err := readBounded(response.Body, MaxResponseBytes)
	if err != nil {
		err = fmt.Errorf("failed to read response for %s request with url ('%s'), %w", method, RedactURL(rawURL), err)
		log.Error(err)
		return nil, err
	}

	return responseBody, nil
}

// redactTransportError removes the URL that net/http puts into the error it
// returns.
//
// *url.Error quotes the request URL verbatim, so wrapping it would put back
// exactly what RedactURL was called to remove. The cause it carries -- the
// dial, TLS or timeout error -- is what is worth logging and names no
// credentials.
func redactTransportError(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) && uerr.Err != nil {
		return uerr.Err
	}
	return err
}

// readBounded reads at most limit bytes from r. A body that is longer than
// that is an error rather than a silently truncated -- and hence unparseable
// or, worse, differently parseable -- document.
func readBounded(r io.Reader, limit int64) ([]byte, error) {
	// read one byte beyond the limit: a body that fills the limit exactly is
	// indistinguishable from a truncated one otherwise
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response body exceeds the %d byte limit", limit)
	}
	return data, nil
}
