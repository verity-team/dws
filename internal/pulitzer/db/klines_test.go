package db

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	c "github.com/verity-team/dws/internal/common"
	"github.com/verity-team/dws/internal/pulitzer/data"
)

// ServesMinute is the gate servePriceRequests uses to decide whether a
// historical source can serve a minute (and so whether to fall through to the
// next source). These tests pin it without a database; they mirror the
// persistence-time gate CloseRequest applies.

func kln(price string, ct time.Time) data.Kline {
	return data.Kline{ClosePrice: decimal.RequireFromString(price), CloseTime: ct}
}

func TestServesMinuteAcceptsAFinalCoveringKline(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute) // the minute is long closed
	kls := []data.Kline{kln("2000.5", ts.Add(59*time.Second))}
	require.True(t, ServesMinute(ts, now, kls))
}

// a source that only returns the minute in progress serves nothing: the open
// kline is dropped, so servePriceRequests must fall through to the next source
func TestServesMinuteRejectsAStillOpenMinute(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 30, 30, 0, time.UTC)
	ts := now.Truncate(time.Minute) // the minute in progress
	kls := []data.Kline{kln("2000.5", ts.Add(59*time.Second))}
	require.False(t, ServesMinute(ts, now, kls))
}

// a kline outside PriceLookupWindow does not serve the minute even when final
func TestServesMinuteRejectsAnOutOfWindowKline(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	kls := []data.Kline{kln("2000.5", ts.Add(c.PriceLookupWindow+time.Second))}
	require.False(t, ServesMinute(ts, now, kls))
}

// a poisoned kline in the (final) set fails the whole set, exactly as
// CloseRequest would refuse to persist it
func TestServesMinuteRejectsAPoisonedSet(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	kls := []data.Kline{
		kln("2000.5", ts.Add(59*time.Second)),
		{ClosePrice: decimal.Zero, CloseTime: ts.Add(time.Minute + 59*time.Second)},
	}
	require.False(t, ServesMinute(ts, now, kls))
}

func TestServesMinuteRejectsEmpty(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	require.False(t, ServesMinute(ts, now, nil))
	require.False(t, ServesMinute(ts, now, []data.Kline{}))
}
