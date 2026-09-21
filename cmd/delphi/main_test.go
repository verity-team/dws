package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
