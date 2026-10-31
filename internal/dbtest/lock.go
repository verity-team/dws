// Package dbtest serializes the database integration test packages (build
// tag `dbtest`).
//
// They share one database and truncate each other's tables, while `go test
// ./...` runs packages in parallel: run concurrently they fail at random
// (e.g. a price row one package expects to be gone is inserted by another).
// Each of them takes the same session level advisory lock in its TestMain, so
// they run one after the other whatever -p the caller picks.
package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	// the postgres driver the integration tests talk to the database with
	_ "github.com/lib/pq"
)

// DefaultDSN is the dockerized development database (`make run_db`).
const DefaultDSN = "host=localhost port=27501 user=postgres password=postgres dbname=dwsdb sslmode=disable"

// LockKey is the advisory lock the test packages serialize on. The single
// bigint key space does not overlap with the two-key form update_user_data()
// locks in, so the two cannot collide.
const LockKey int64 = 0x6477735f74657374 // "dws_test"

// DSN is the test database: DWS_TEST_DB_DSN, DefaultDSN if unset.
func DSN() string {
	if dsn := os.Getenv("DWS_TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return DefaultDSN
}

// Lock takes the session level advisory lock LockKey on a connection of its
// own and returns the function that releases it. It waits for as long as
// another session holds the lock, or until ctx is done.
func Lock(ctx context.Context, db *sql.DB) (func(), error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get a connection for the test lock, %w", err)
	}
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, LockKey); err != nil {
		conn.Close() // nolint:errcheck
		return nil, fmt.Errorf("failed to take the test lock, %w", err)
	}
	return func() {
		// closing the session releases the lock as well, the explicit unlock
		// only makes that independent of the connection pool
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, LockKey)
		conn.Close() // nolint:errcheck
	}, nil
}

// Run runs the tests of a database integration test package while holding
// the test lock; use it as `os.Exit(dbtest.Run(m))` in TestMain. Without a
// reachable database the tests run unlocked: each of them skips itself.
func Run(m *testing.M) int {
	db, err := sql.Open("postgres", DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbtest: %v\n", err)
		return 1
	}
	defer db.Close() // nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		return m.Run()
	}
	unlock, err := Lock(context.Background(), db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbtest: %v\n", err)
		return 1
	}
	defer unlock()
	return m.Run()
}
