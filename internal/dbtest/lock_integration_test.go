//go:build dbtest

package dbtest

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// tryLock reports whether a session other than the lock holder's could take
// the test lock; it releases it again right away
func tryLock(t *testing.T, db *sql.DB) bool {
	t.Helper()

	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close() // nolint:errcheck
	var ok bool
	require.NoError(t, conn.QueryRowContext(context.Background(), `SELECT pg_try_advisory_lock($1)`, LockKey).Scan(&ok))
	if ok {
		_, err = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, LockKey)
		require.NoError(t, err)
	}
	return ok
}

// #255: while one test package holds the lock no other session gets it, and
// releasing it lets the next one in
func TestLockSerializesSessions(t *testing.T) {
	db, err := sql.Open("postgres", DSN())
	require.NoError(t, err)
	defer db.Close() // nolint:errcheck
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("no test database reachable: %v", err)
	}

	// waits for a test package that holds the lock right now
	unlock, err := Lock(context.Background(), db)
	require.NoError(t, err)
	require.False(t, tryLock(t, db), "the lock must be exclusive")

	// a second Lock waits until the first is released
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	_, err = Lock(ctx2, db)
	require.Error(t, err, "a second session must wait for the lock")

	unlock()
	unlock2, err := Lock(context.Background(), db)
	require.NoError(t, err)
	unlock2()
}
