package db

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/verity-team/dws/internal/pulitzer/data"
)

// MinuteKline is the gate servePriceRequests uses to pick the kline of the
// requested minute out of what a historical source returned (and so whether
// to fall through to the next source). These tests pin it without a database.

func kln(price string, ct time.Time) data.Kline {
	return data.Kline{ClosePrice: decimal.RequireFromString(price), CloseTime: ct}
}

// the kline of the requested minute is picked out of the set; the other
// minutes a venue returns alongside it are not persisted (#271)
func TestMinuteKlinePicksTheRequestedMinute(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute) // the minute is long closed
	kls := []data.Kline{
		kln("1999.5", ts.Add(-time.Second)),
		kln("2000.5", ts.Add(59*time.Second)),
		kln("2001.5", ts.Add(time.Minute+59*time.Second)),
	}
	kl, err := MinuteKline(ts, now, kls)
	require.NoError(t, err)
	require.True(t, kl.ClosePrice.Equal(decimal.RequireFromString("2000.5")), "got %s", kl.ClosePrice)
	require.Equal(t, ts.Add(59*time.Second), kl.CloseTime)
	// the returned kline always passes CloseRequest's own gate
	require.NoError(t, validateKlines(1, ts, finalKlines([]data.Kline{kl}, now)))
}

// a source that only returns the minute in progress serves nothing: the open
// kline is dropped, so servePriceRequests must fall through to the next source
func TestMinuteKlineRejectsAStillOpenMinute(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 30, 30, 0, time.UTC)
	ts := now.Truncate(time.Minute) // the minute in progress
	_, err := MinuteKline(ts, now, []data.Kline{kln("2000.5", ts.Add(59*time.Second))})
	require.Error(t, err)
}

// a neighbouring minute does not stand in for the requested one, even when it
// is inside the lookup window: the corroborating source is compared on the
// same minute
func TestMinuteKlineRejectsANeighbouringMinute(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	_, err := MinuteKline(ts, now, []data.Kline{
		kln("1999.5", ts.Add(-time.Second)),
		kln("2001.5", ts.Add(time.Minute+59*time.Second)),
	})
	require.Error(t, err)
}

// a poisoned kline in the (final) set fails the whole set
func TestMinuteKlineRejectsAPoisonedSet(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	_, err := MinuteKline(ts, now, []data.Kline{
		kln("2000.5", ts.Add(59*time.Second)),
		{ClosePrice: decimal.Zero, CloseTime: ts.Add(time.Minute + 59*time.Second)},
	})
	require.Error(t, err)
}

// two klines for the requested minute are ambiguous
func TestMinuteKlineRejectsADuplicatedMinute(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	_, err := MinuteKline(ts, now, []data.Kline{
		kln("2000.5", ts.Add(59*time.Second)),
		kln("2100.5", ts.Add(30*time.Second)),
	})
	require.Error(t, err)
}

func TestMinuteKlineRejectsEmpty(t *testing.T) {
	ts := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	now := ts.Add(5 * time.Minute)
	_, err := MinuteKline(ts, now, nil)
	require.Error(t, err)
	_, err = MinuteKline(ts, now, []data.Kline{})
	require.Error(t, err)
}
