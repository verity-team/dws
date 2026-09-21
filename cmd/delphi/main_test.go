package main

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
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
