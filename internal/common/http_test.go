package common

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

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
