package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/delphi/server"
)

// echo.New() leaves every server timeout at zero i.e. a slow or stalled client
// can hold on to a connection indefinitely
func TestSetServerTimeouts(t *testing.T) {
	e := echo.New()
	assert.Zero(t, e.Server.ReadHeaderTimeout)
	assert.Zero(t, e.Server.ReadTimeout)
	assert.Zero(t, e.Server.WriteTimeout)
	assert.Zero(t, e.Server.IdleTimeout)

	setServerTimeouts(e.Server)

	assert.Greater(t, e.Server.ReadHeaderTimeout, time.Duration(0))
	assert.Greater(t, e.Server.ReadTimeout, time.Duration(0))
	assert.Greater(t, e.Server.WriteTimeout, time.Duration(0))
	assert.Greater(t, e.Server.IdleTimeout, time.Duration(0))

	// reading the headers is part of reading the request
	assert.GreaterOrEqual(t, e.Server.ReadTimeout, e.Server.ReadHeaderTimeout)
	// the write timeout covers the handler as well, so it has to leave room for
	// the slowest legitimate response
	assert.Greater(t, e.Server.WriteTimeout, e.Server.ReadTimeout)
	// the shutdown grace period has to stay below the one of the orchestrator
	assert.Greater(t, shutdownTimeout, time.Duration(0))
	assert.Less(t, shutdownTimeout, 30*time.Second)
}

// fire n requests at h from a single peer and report the status codes seen
func fire(t *testing.T, e *echo.Echo, path string, n int, xff string) map[int]int {
	t.Helper()
	seen := make(map[int]int)
	for i := 0; i < n; i++ {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.10:54321"
		if xff != "" {
			req.Header.Set(echo.HeaderXForwardedFor, xff)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		seen[rec.Code]++
	}
	return seen
}

func rateLimitedEcho() *echo.Echo {
	e := echo.New()
	e.Use(echomiddleware.RateLimiterWithConfig(rateLimiterConfig()))
	h := func(c echo.Context) error { return c.String(http.StatusOK, "{}\n") }
	e.GET("/donation/data", h)
	e.GET("/live", h)
	e.GET("/ready", h)
	return e
}

func TestRateLimiterRejectsAFlood(t *testing.T) {
	e := rateLimitedEcho()

	// well beyond the burst allowance of a single peer
	seen := fire(t, e, "/donation/data", rateBurst*3, "")
	assert.Positive(t, seen[http.StatusOK], "legitimate traffic must get through")
	assert.Positive(t, seen[http.StatusTooManyRequests], "a flood must be rejected")
}

// the identifier is the direct peer address: X-Forwarded-For is attacker
// controlled and rotating it must not hand out a fresh budget
func TestRateLimiterIgnoresForwardedForHeader(t *testing.T) {
	e := rateLimitedEcho()

	seen := fire(t, e, "/donation/data", rateBurst*3, "")
	require.Positive(t, seen[http.StatusTooManyRequests])

	seen = fire(t, e, "/donation/data", 10, "203.0.113.7")
	assert.Zero(t, seen[http.StatusOK], "a spoofed X-Forwarded-For must not reset the budget")
}

// an orchestrator probe must never be taken out by the rate limiter
func TestRateLimiterSkipsHealthChecks(t *testing.T) {
	e := rateLimitedEcho()

	require.Positive(t, fire(t, e, "/donation/data", rateBurst*3, "")[http.StatusTooManyRequests])
	for _, p := range []string{"/live", "/ready"} {
		seen := fire(t, e, p, 200, "")
		assert.Equal(t, 200, seen[http.StatusOK], "%s must not be rate limited", p)
		assert.Zero(t, seen[http.StatusTooManyRequests], p)
	}
}

// the budget has to leave room for the frontend's legitimate polling: a
// browser session costs roughly three requests a minute
func TestRateLimitLeavesRoomForTheFrontend(t *testing.T) {
	const requestsPerSessionPerMinute = 3
	sessions := rateLimit * 60 / requestsPerSessionPerMinute
	assert.GreaterOrEqual(t, sessions, 1000, "the per-IP budget is too small for a shared frontend egress address")
	assert.GreaterOrEqual(t, rateBurst, rateLimit, "the burst must absorb a refresh spike")
}

// testEcho builds the server the way main() does; the handlers have no
// database, so only requests that are rejected before the first query may be
// sent to it.
func testEcho(t *testing.T) *echo.Echo {
	t.Helper()
	swagger, err := api.GetSwagger()
	require.NoError(t, err)
	swagger.Servers = nil
	return newEcho(swagger, server.NewDelphiServer(nil))
}

// http.Server defaults to 1 MB of request headers
func TestNewEchoLimitsHeaderBytes(t *testing.T) {
	e := testEcho(t)
	assert.Equal(t, maxHeaderBytes, e.Server.MaxHeaderBytes)
	assert.Less(t, e.Server.MaxHeaderBytes, http.DefaultMaxHeaderBytes)
	// timeouts are still in place
	assert.Positive(t, e.Server.ReadHeaderTimeout)
}

// authRequest builds a POST /affiliate/code request with the given headers.
func authRequest(key, ts, sig string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/affiliate/code", nil)
	req.RemoteAddr = "192.0.2.20:54321"
	req.Header.Set("delphi-key", key)
	req.Header.Set("delphi-ts", ts)
	req.Header.Set("delphi-signature", sig)
	return req
}

// the authentication headers are logged before the signature is checked; the
// spec bounds them so that the request validator rejects anything else
func TestAuthHeadersAreValidated(t *testing.T) {
	const key = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	sig := "0x" + strings.Repeat("ff", 65)

	for _, tc := range []struct {
		name         string
		key, ts, sig string
	}{
		{"key not hex", "0x" + strings.Repeat("zz", 20), ts, sig},
		{"key too long", key + "00", ts, sig},
		{"key without 0x", key[2:], ts, sig},
		{"key padded", key + strings.Repeat(" ", 4000), ts, sig},
		{"ts not digits", key, "17x", sig},
		{"ts negative", key, "-1", sig},
		{"ts too long", key, strings.Repeat("9", 13), sig},
		{"sig too long", key, ts, sig + "ff"},
		{"sig too short", key, ts, sig[:len(sig)-2]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testEcho(t).ServeHTTP(rec, authRequest(tc.key, tc.ts, tc.sig))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}

	// well formed headers pass the validator and fail the signature check
	rec := httptest.NewRecorder()
	testEcho(t).ServeHTTP(rec, authRequest(key, ts, sig))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// assertGenericError checks that a rejected request is answered with the
// numeric code and the generic message, and nothing of the input or the schema
func assertGenericError(t *testing.T, rec *httptest.ResponseRecorder, status, code int, leaks ...string) {
	t.Helper()
	assert.Equal(t, status, rec.Code)
	var cerr api.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cerr), rec.Body.String())
	assert.Equal(t, code, cerr.Code)
	assert.Equal(t, "invalid request", cerr.Message)
	for _, leak := range leaks {
		assert.NotContains(t, rec.Body.String(), leak)
	}
}

// AUD4-23: the request validator's error used to be returned verbatim
func TestValidationErrorsAreGeneric(t *testing.T) {
	const key = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	sig := "0x" + strings.Repeat("ff", 65)

	t.Run("header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		testEcho(t).ServeHTTP(rec, authRequest("0xLEAKME", ts, sig))
		assertGenericError(t, rec, http.StatusBadRequest, 116, "LEAKME", "delphi-key", "regular expression", "0x[0-9a-fA-F]")
	})
	t.Run("body", func(t *testing.T) {
		body := `{"code":"LEAKME-LEAKME-LEAKME","address":"` + key + `"}`
		req := httptest.NewRequest(http.MethodPost, "/wallet/connection", strings.NewReader(body))
		req.RemoteAddr = "192.0.2.21:54321"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("delphi-key", key)
		req.Header.Set("delphi-ts", ts)
		req.Header.Set("delphi-signature", sig)
		rec := httptest.NewRecorder()
		testEcho(t).ServeHTTP(rec, req)
		assertGenericError(t, rec, http.StatusBadRequest, 116, "LEAKME", "maximum", "length")
	})
	t.Run("query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/user/data/"+key+"?limit=LEAKME", nil)
		req.RemoteAddr = "192.0.2.22:54321"
		rec := httptest.NewRecorder()
		testEcho(t).ServeHTTP(rec, req)
		assertGenericError(t, rec, http.StatusBadRequest, 116, "LEAKME", "limit", "integer")
	})
}

// the generated parameter binding used to answer "Expected one value for
// delphi-key, got 2" and "Invalid format for parameter ...: <input>"
func TestBindingErrorsAreGeneric(t *testing.T) {
	const key = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	req := authRequest(key, strconv.FormatInt(time.Now().UTC().Unix(), 10), "0x"+strings.Repeat("ff", 65))
	req.Header.Add("delphi-key", key)
	rec := httptest.NewRecorder()
	testEcho(t).ServeHTTP(rec, req)
	assertGenericError(t, rec, http.StatusBadRequest, 117, "Expected", "delphi-key", "got 2")
}

// the validator rejects a path the spec does not know of
func TestUnknownPathIsGeneric(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/no/such/path", nil)
	req.RemoteAddr = "192.0.2.23:54321"
	rec := httptest.NewRecorder()
	testEcho(t).ServeHTTP(rec, req)
	assertGenericError(t, rec, http.StatusNotFound, 116, "no matching operation")
}

// echo-middleware >= v1.1 answers a known path with an unknown method with a
// 405 (it used to be a 404); it still takes the generic path
func TestUnknownMethodIsGeneric(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/donation/data", nil)
	req.RemoteAddr = "192.0.2.24:54321"
	rec := httptest.NewRecorder()
	testEcho(t).ServeHTTP(rec, req)
	assertGenericError(t, rec, http.StatusMethodNotAllowed, 116, "method")
}

// AUD2-10: the request validator rejects an offset above the cap; one at the
// cap passes it and reaches the handler (which rejects the address, code 103)
func TestUserDataOffsetIsCapped(t *testing.T) {
	for _, tc := range []struct {
		offset       int
		status, code int
	}{
		{server.MaxUserDataOffset + 1, http.StatusBadRequest, 116},
		{server.MaxUserDataOffset, http.StatusBadRequest, 103},
	} {
		path := "/user/data/0xnotanaddress?offset=" + strconv.Itoa(tc.offset)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.25:54321"
		rec := httptest.NewRecorder()
		testEcho(t).ServeHTTP(rec, req)
		assert.Equal(t, tc.status, rec.Code, "offset %d", tc.offset)
		var cerr api.Error
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cerr), rec.Body.String())
		assert.Equal(t, tc.code, cerr.Code, "offset %d", tc.offset)
	}
}
