package common

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// apiKey stands in for the provider credentials ETH_RPC_URL carries.
const apiKey = "0123456789abcdef0123456789abcdef"

func TestHTTPGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	body, err := HTTPGet(HTTPParams{URL: srv.URL + "/v3/" + apiKey})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
}

// the URL of a failed request ends up in the error the caller logs and wraps;
// it must not carry the provider credentials
func TestHTTPGetErrorDoesNotLeakTheURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := HTTPGet(HTTPParams{URL: srv.URL + "/v3/" + apiKey})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey)
	assert.Contains(t, err.Error(), "403")
}

func TestHTTPPostErrorDoesNotLeakTheURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := HTTPPost(HTTPParams{URL: srv.URL + "/v3/" + apiKey, RequestBody: []byte(`{}`)})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey)
	assert.Contains(t, err.Error(), "429")
}

// a connection level failure carries the URL as well
func TestHTTPPostConnectionErrorDoesNotLeakTheURL(t *testing.T) {
	// nothing is listening on port 1
	_, err := HTTPPost(HTTPParams{
		URL:              "http://127.0.0.1:1/v3/" + apiKey,
		RequestBody:      []byte(`{}`),
		MaxWaitInSeconds: 1,
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey)
}

// the body of a non-200 response used to be read in full and then written to
// the log in full -- a WAF challenge page or a captive portal answer, once a
// minute, indefinitely -- before the status was even looked at
func TestFailedResponseBodyIsTruncatedInTheLog(t *testing.T) {
	// an order of magnitude more than MaxLoggedBodyBytes
	body := strings.Repeat("x", 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	_, err := HTTPGet(HTTPParams{URL: srv.URL})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")

	logged := buf.String()
	assert.NotContains(t, logged, body, "the body must not be logged in full")
	assert.Less(t, len(logged), 4*MaxLoggedBodyBytes,
		"at most %d bytes of the body belong in the log, got %d bytes of output", MaxLoggedBodyBytes, len(logged))
}

// a 200 response longer than MaxResponseBytes is an error, not a silently
// truncated document
func TestReadBoundedRejectsAnOversizedBody(t *testing.T) {
	_, err := readBounded(strings.NewReader(strings.Repeat("x", 11)), 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")

	// a body that fills the limit exactly is still accepted
	body, err := readBounded(strings.NewReader(strings.Repeat("x", 10)), 10)
	require.NoError(t, err)
	assert.Len(t, body, 10)
}

func TestMaxResponseBytesIsEnforced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for i := 0; i < 64; i++ {
			if _, err := fmt.Fprint(w, strings.Repeat("y", 1<<20)); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	_, err := HTTPGet(HTTPParams{URL: srv.URL})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

// HTTPGetCtx binds the request to the context: a server that never answers is
// abandoned as soon as the context deadline fires, well before the 10s client
// timeout, so the caller's chain budget actually bounds an in-flight request.
func TestHTTPGetCtxCancelsAnInFlightRequest(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release // hang until the test lets go
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := HTTPGetCtx(ctx, HTTPParams{URL: srv.URL + "/v3/" + apiKey})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey, "a cancelled request must not leak the URL")
	assert.Less(t, elapsed, 5*time.Second, "the request must be cancelled by the context, not wait out the client timeout")
}

// the healthy path is unchanged from HTTPGet: a 200 body is returned, and a
// non-200 is an error carrying the status but not the credentials
func TestHTTPGetCtxSuccessAndRedaction(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer ok.Close()

	body, err := HTTPGetCtx(context.Background(), HTTPParams{URL: ok.URL + "/v3/" + apiKey})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer bad.Close()

	_, err = HTTPGetCtx(context.Background(), HTTPParams{URL: bad.URL + "/v3/" + apiKey})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), apiKey)
	assert.Contains(t, err.Error(), "403")
}

// an https endpoint redirecting to http must not move the request onto
// plaintext; the plaintext target is never contacted (#288)
func TestRedirectFromHTTPSToHTTPIsRefused(t *testing.T) {
	var hit atomic.Bool
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit.Store(true)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/v3/"+apiKey, http.StatusFound)
	}))
	defer secure.Close()

	// the production client, trusting the test server's certificate
	client := newClient(HTTPParams{})
	client.Transport = secure.Client().Transport
	_, err := client.Get(secure.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, errSchemeDowngrade)
	assert.False(t, hit.Load(), "the plaintext target must not be contacted")
	assert.NotContains(t, redactTransportError(err).Error(), apiKey)
}

// the other redirects keep working: https to https ...
func TestRedirectFromHTTPSToHTTPSIsFollowed(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, srv.URL+"/here", http.StatusMovedPermanently)
			return
		}
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newClient(HTTPParams{})
	client.Transport = srv.Client().Transport
	resp, err := client.Get(srv.URL + "/moved")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/here", resp.Request.URL.Path)
}

// ... and http to http (the loopback dev nodes), through the exported helpers
func TestRedirectFromHTTPToHTTPIsFollowed(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	body, err := HTTPGet(HTTPParams{URL: srv.URL})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
	body, err = HTTPGetCtx(context.Background(), HTTPParams{URL: srv.URL})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
	body, err = HTTPPost(HTTPParams{URL: srv.URL, RequestBody: []byte(`{}`)})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
}

// the policy itself: a downgrade is refused at any hop, and net/http's
// default redirect limit still applies
func TestRefuseSchemeDowngrade(t *testing.T) {
	req := func(raw string) *http.Request {
		r, err := http.NewRequest("GET", raw, nil)
		require.NoError(t, err)
		return r
	}
	https, plain := req("https://a.example/"), req("http://b.example/")

	assert.NoError(t, refuseSchemeDowngrade(https, nil))
	assert.NoError(t, refuseSchemeDowngrade(https, []*http.Request{plain}))
	assert.NoError(t, refuseSchemeDowngrade(plain, []*http.Request{plain}))
	assert.ErrorIs(t, refuseSchemeDowngrade(plain, []*http.Request{https}), errSchemeDowngrade)
	assert.ErrorIs(t, refuseSchemeDowngrade(plain, []*http.Request{plain, https}), errSchemeDowngrade)

	via := make([]*http.Request, maxRedirects)
	for i := range via {
		via[i] = https
	}
	assert.Error(t, refuseSchemeDowngrade(https, via))
	assert.NoError(t, refuseSchemeDowngrade(https, via[:maxRedirects-1]))
}
