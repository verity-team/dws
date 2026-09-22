package common

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// malformedDSN carries a password with a space in it. lib/pq stops at the
// space and reports the remainder as a key that is missing its '=' -- i.e. it
// quotes the tail of the password back.
const (
	malformedDSN   = "host=localhost port=5432 user=dws password=hunter2 s3cretTail dbname=dwsdb sslmode=disable"
	leakedFragment = "s3cretTail"
)

// sql.Open does not parse the connection string for lib/pq: the driver does
// not implement driver.DriverContext, so the DSN is handed over when the
// first connection is made and not before.
//
// This pins lib/pq's behaviour, which is the point of the test -- if a future
// version diagnoses the DSN in Open, the timing of every connection error
// changes and this test says so. It is *not* a reachable production path any
// more: GetDSN quotes every value (see quoteDSNValue), so a DSN this repo
// builds cannot be misparsed however exotic the password. The malformed DSN
// below is hand written to exercise the driver.
func TestMalformedDSNSurfacesOnConnectNotOnOpen(t *testing.T) {
	dbh, err := sqlx.Open("postgres", malformedDSN)
	require.NoError(t, err, "sql.Open is expected not to parse the DSN")
	defer func() { _ = dbh.Close() }()

	perr := dbh.Ping()
	require.Error(t, perr)
	// the raw driver error is exactly what must never reach a log line
	assert.Contains(t, perr.Error(), leakedFragment)
}

// the whole point: whatever lib/pq says about a malformed connection string,
// no part of the string comes out
func TestRedactDBErrorWithholdsTheConnectionString(t *testing.T) {
	dbh, err := sqlx.Open("postgres", malformedDSN)
	require.NoError(t, err)
	defer func() { _ = dbh.Close() }()

	redacted := RedactDBError(dbh.Ping())
	require.Error(t, redacted)
	assert.NotContains(t, redacted.Error(), leakedFragment)
	assert.NotContains(t, redacted.Error(), "hunter2")
	assert.NotContains(t, redacted.Error(), "dws")
	assert.Contains(t, redacted.Error(), "DWS_DB_")

	// and wrapping it the way OpenDB does keeps it clean
	wrapped := fmt.Errorf("failed to reach the database, %w", redacted)
	assert.NotContains(t, wrapped.Error(), leakedFragment)
}

// a network failure names the host and port, which GetDSN already logs, so it
// is passed through: withholding it would cost the operator the single most
// common diagnostic
func TestRedactDBErrorKeepsNetworkErrors(t *testing.T) {
	// nothing is listening on port 1
	dbh, err := sqlx.Open("postgres",
		"host=127.0.0.1 port=1 user=dws password=s3cretTail dbname=dwsdb sslmode=disable connect_timeout=1")
	require.NoError(t, err)
	defer func() { _ = dbh.Close() }()

	perr := dbh.Ping()
	require.Error(t, perr)
	var netErr net.Error
	require.ErrorAs(t, perr, &netErr, "expected a net.Error for an unreachable host")

	redacted := RedactDBError(perr)
	assert.Equal(t, perr, redacted)
	assert.Contains(t, redacted.Error(), "127.0.0.1:1")
	// the DSN of an unreachable host is well formed, so nothing of it is
	// quoted back in the first place
	assert.NotContains(t, redacted.Error(), leakedFragment)
}

// an error the *server* produced carries no connection string, and it is what
// tells an operator that the password or the database name is wrong
func TestRedactDBErrorKeepsServerErrors(t *testing.T) {
	pqErr := &pq.Error{
		Severity: "FATAL",
		Code:     "28P01",
		Message:  `password authentication failed for user "dws"`,
	}
	assert.Equal(t, error(pqErr), RedactDBError(pqErr))
	assert.Contains(t, RedactDBError(pqErr).Error(), "password authentication failed")

	// also when it is wrapped
	wrapped := fmt.Errorf("query failed, %w", pqErr)
	assert.Equal(t, wrapped, RedactDBError(wrapped))
}

func TestRedactDBErrorWithholdsAnythingElse(t *testing.T) {
	assert.Nil(t, RedactDBError(nil))

	// the shape lib/pq's parseOpts returns
	parseErr := errors.New(`missing "=" after "s3cretTail" in connection info string"`)
	redacted := RedactDBError(parseErr)
	assert.NotContains(t, redacted.Error(), leakedFragment)
	assert.ErrorIs(t, redacted, errDSNRedacted)
}

// OpenDB refuses to hand back a handle it could not use
func TestOpenDBRejectsAMalformedDSN(t *testing.T) {
	dbh, err := OpenDB(malformedDSN)
	require.Error(t, err)
	assert.Nil(t, dbh)
	assert.NotContains(t, err.Error(), leakedFragment)
	assert.Contains(t, err.Error(), "failed to reach the database")
}

func TestOpenDBRejectsAnUnreachableDatabase(t *testing.T) {
	dbh, err := OpenDB("host=127.0.0.1 port=1 user=dws password=s3cretTail dbname=dwsdb sslmode=disable connect_timeout=1")
	require.Error(t, err)
	assert.Nil(t, dbh)
	assert.Contains(t, err.Error(), "failed to reach the database")
	assert.NotContains(t, err.Error(), leakedFragment)
}

// quoteDSNValue is the primary control: with the values quoted, no password
// can reassign another key, which is what made the origin allow-list leaky.
// An unquoted password ending in ` host=...` used to redirect the connection
// *and* hand the tail of the password to the dial error, which RedactDBError
// passes through as a net.Error.
func TestQuoteDSNValue(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"plain", "hunter2", `'hunter2'`},
		{"with a space", "hunter2 s3cret", `'hunter2 s3cret'`},
		{"key injection", "hunter2 host=elsewhere.invalid", `'hunter2 host=elsewhere.invalid'`},
		{"single quote", "hun'ter2", `'hun\'ter2'`},
		{"backslash", `hun\ter2`, `'hun\\ter2'`},
		{"quote and backslash", `a'\b`, `'a\'\\b'`},
		{"terminator attempt", `x' host='elsewhere.invalid`, `'x\' host=\'elsewhere.invalid'`},
		{"empty", "", `''`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, quoteDSNValue(tc.in))
		})
	}
}

// the end to end property: whatever the password contains, the connection
// goes to the configured host and no part of the password reaches the error
func TestDSNValuesCannotBeReassignedByThePassword(t *testing.T) {
	const (
		host    = "127.0.0.1"
		leakTag = "s3cretTail"
	)
	t.Setenv("DWS_DB_HOST", host)
	t.Setenv("DWS_DB_PORT", "1")
	t.Setenv("DWS_DB_USER", "dws")
	t.Setenv("DWS_DB_DATABASE", "dwsdb")
	t.Setenv("DWS_DB_SSLMODE", "disable")

	for _, passwd := range []string{
		"hunter2 host=" + leakTag + ".invalid",
		"hunter2 dbname=" + leakTag,
		"hunter2 sslmode=" + leakTag,
		"hunter2 " + leakTag,
		"hunter2'" + leakTag,
		`hunter2\` + leakTag,
	} {
		t.Setenv("DWS_DB_PASSWORD", passwd)
		dsn := GetDSN()

		dbh, err := sqlx.Open("postgres", dsn)
		require.NoError(t, err)
		perr := dbh.Ping()
		_ = dbh.Close()

		require.Error(t, perr, "nothing is listening on port 1")
		// the connection went where it was told to, not where the password said
		assert.Contains(t, perr.Error(), host+":1", "password %q redirected the connection", passwd)
		// and no part of the password came back, raw or redacted
		assert.NotContains(t, perr.Error(), leakTag, "password %q leaked into the error", passwd)
		assert.NotContains(t, RedactDBError(perr).Error(), leakTag)
	}
}

// the values the DSN carries survive quoting intact, including the awkward ones
func TestGetDSNQuotesEveryValue(t *testing.T) {
	t.Setenv("DWS_DB_HOST", "db.internal")
	t.Setenv("DWS_DB_PORT", "5432")
	t.Setenv("DWS_DB_USER", "dws user")
	t.Setenv("DWS_DB_PASSWORD", `p a s s'\w`)
	t.Setenv("DWS_DB_DATABASE", "dwsdb")
	t.Setenv("DWS_DB_SSLMODE", "verify-full")

	dsn := GetDSN()
	assert.Contains(t, dsn, `host='db.internal'`)
	assert.Contains(t, dsn, `user='dws user'`)
	assert.Contains(t, dsn, `password='p a s s\'\\w'`)
	assert.Contains(t, dsn, `sslmode='verify-full'`)
	// the connect timeout is ours and bounds the dial, see DBConnectTimeout
	assert.Contains(t, dsn, fmt.Sprintf("connect_timeout=%d", int64(DBConnectTimeout.Seconds())))
}

// connect_timeout has to reach lib/pq through the DSN: it discards the context
// before dial(), and without the parameter dial() waits indefinitely
func TestDSNCarriesAConnectTimeout(t *testing.T) {
	assert.Positive(t, int64(DBConnectTimeout.Seconds()),
		"connect_timeout is rendered in whole seconds and must not round to 0")
	assert.Greater(t, dbPingTimeout, DBConnectTimeout,
		"the ping deadline must outlast a single connection attempt, so lib/pq reports the reason")
}

// an unsupported sslmode is rejected while the configuration is being read,
// rather than by lib/pq on some later connection attempt
func TestValidSSLModes(t *testing.T) {
	for _, mode := range []string{"disable", "require", "verify-ca", "verify-full"} {
		assert.True(t, validSSLModes[mode], mode)
	}
	// lib/pq treats "" as require; an explicitly empty setting is a mistake
	assert.False(t, validSSLModes[""])
	assert.False(t, validSSLModes["prefer"])
	assert.False(t, validSSLModes["Require"])
	assert.Equal(t, "disable, require, verify-ca, verify-full", sslModeList())
	assert.True(t, validSSLModes[DefaultSSLMode])
}

// every origin RedactDBError passes through has to be free of connection
// string material -- that is the premise the pass-through rests on
func TestRedactDBErrorPassThroughsCarryNoDSN(t *testing.T) {
	const leakTag = "s3cretTail"

	// the sentinels carry fixed text
	for _, sentinel := range safeDBSentinels {
		assert.Same(t, sentinel, RedactDBError(sentinel))
		assert.NotContains(t, sentinel.Error(), leakTag)
		// nothing that looks like a connection parameter
		for _, key := range []string{"password=", "dbname=", "host=", "sslmode=", "user="} {
			assert.NotContains(t, sentinel.Error(), key, "%v", sentinel)
		}
	}
	assert.Equal(t, "pq: SSL is not enabled on the server", pq.ErrSSLNotSupported.Error())
	assert.Equal(t, "driver: bad connection", driver.ErrBadConn.Error())

	// the TLS/x509 verification errors name the certificate and the host
	tlsErrs := []error{
		&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}},
		x509.UnknownAuthorityError{},
		// HostnameError.Error() consults the certificate, so it needs one
		x509.HostnameError{Host: "db.internal", Certificate: &x509.Certificate{
			Subject: pkix.Name{CommonName: "db.internal"},
		}},
		x509.CertificateInvalidError{Reason: x509.Expired},
	}
	for _, e := range tlsErrs {
		assert.Equal(t, e, RedactDBError(e), "%T must be passed through", e)
		for _, key := range []string{"password=", "dbname=", "sslmode="} {
			assert.NotContains(t, e.Error(), key, "%T", e)
		}
	}
	// wrapped, the way lib/pq hands them over
	wrapped := fmt.Errorf("failed to connect, %w", x509.UnknownAuthorityError{})
	assert.Equal(t, wrapped, RedactDBError(wrapped))
}

// the SSL sentinel is what a developer sees against the dockerized database,
// which serves no TLS while GetDSN defaults to sslmode=require
func TestRedactDBErrorKeepsTheSSLSentinel(t *testing.T) {
	redacted := RedactDBError(pq.ErrSSLNotSupported)
	require.Error(t, redacted)
	assert.Contains(t, redacted.Error(), "SSL is not enabled on the server")
	assert.NotErrorIs(t, redacted, errDSNRedacted)

	// and wrapped into a query failure
	wrapped := fmt.Errorf("failed to fetch donation records, %w", pq.ErrSSLNotSupported)
	assert.Contains(t, RedactDBError(wrapped).Error(), "SSL is not enabled on the server")
}
