package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// PriceLookupWindow is how far GetETHPrice looks on either side of the
// timestamp it is asked for.
//
// It is exported because it defines what "a price for this minute" means:
// pulitzer must not record a price request as served unless the prices it
// persisted actually fall inside this window, otherwise the request is
// terminal ('succeeded' requests are never served again) while buck still
// cannot price the block -- and the finalized crawler loops on it forever.
const PriceLookupWindow = 90 * time.Second

// PriceLookupInterval renders PriceLookupWindow as a postgres interval
// literal.
func PriceLookupInterval() string {
	return fmt.Sprintf("%d seconds", int64(PriceLookupWindow.Seconds()))
}

func GetETHPrice(db *sqlx.DB, ts time.Time) (decimal.Decimal, error) {
	// find a price that is within PriceLookupWindow of the timestamp and
	// closest to it
	q := `
		SELECT price
		FROM price
			WHERE asset = 'eth'
			  AND created_at >= $1::timestamp AT TIME ZONE 'UTC' - $2::interval
			  AND created_at <= $1::timestamp AT TIME ZONE 'UTC' + $2::interval
			ORDER BY ABS(EXTRACT(EPOCH FROM (created_at - $1::timestamp AT TIME ZONE 'UTC'))) ASC
			LIMIT 1
		`
	var ethp decimal.Decimal
	if err := db.Get(&ethp, q, ts, PriceLookupInterval()); err != nil {
		err = fmt.Errorf("failed to fetch ETH price for time %v, %w", ts, err)
		log.Error(err)
		return decimal.Zero, err
	}
	return ethp, nil
}

// DBConnectTimeout bounds a single connection attempt. It is put into the DSN
// as `connect_timeout`, which lib/pq applies to the dial *and* to the startup
// handshake.
//
// It has to be carried in the DSN rather than through a context: lib/pq's
// Open path reaches dial() as DialOpen -> Connector.open(context.Background())
// and discards whatever context the caller passed, so a PingContext deadline
// never gets near the dial. Without `connect_timeout` dial() takes its "wait
// indefinitely" branch, and an unroutable or blackholed host hangs the caller
// for the OS SYN timeout -- around two minutes on Linux. In the DSN the bound
// also covers every connection the pool opens later, not just the first one.
const DBConnectTimeout = 10 * time.Second

// dbPingTimeout bounds the startup connectivity check as a whole. It is
// deliberately longer than DBConnectTimeout so that a connection that times
// out is reported by lib/pq, which says why, rather than by a bare context
// deadline -- and so that a slow but successful connect is not refused.
const dbPingTimeout = DBConnectTimeout + 5*time.Second

// errDSNRedacted stands in for a database error that may quote the connection
// string back.
var errDSNRedacted = errors.New(
	"withheld: the driver error may quote the connection string, check the DWS_DB_* variables")

// RedactDBError returns an error that is safe to log for a failed database
// connection.
//
// This is defence in depth, not the primary control. The primary control is
// quoteDSNValue: a DSN built by GetDSN cannot be misparsed and cannot have
// one of its values reassigned by another, so lib/pq has nothing of the
// connection string to quote back. This function assumes that could be wrong
// and withholds anything it does not recognize.
//
// What it recognizes are the origins whose message is known not to contain
// connection string material, because withholding those would cost an
// operator the diagnostics that actually identify the problem:
//
//   - *pq.Error is built from the server's error response ("password
//     authentication failed for user ...", "database ... does not exist");
//   - net.Error names the host and port, which GetDSN already logs;
//   - pq.ErrSSLNotSupported and its siblings are package level sentinels with
//     fixed text. ErrSSLNotSupported is the single most common local
//     misconfiguration: GetDSN defaults to sslmode=require and the
//     dockerized development database serves no TLS;
//   - driver.ErrBadConn is the fixed string "driver: bad connection";
//   - a TLS or x509 verification failure under verify-ca/verify-full names
//     the certificate and the host, never the DSN.
//
// Anything else is withheld. TestRedactDBErrorPassThroughsCarryNoDSN pins
// that each of the above really is free of connection string material.
func RedactDBError(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return err
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return err
	}
	for _, sentinel := range safeDBSentinels {
		if errors.Is(err, sentinel) {
			return err
		}
	}
	if isTLSVerificationError(err) {
		return err
	}
	return errDSNRedacted
}

// safeDBSentinels are the package level error values whose text is fixed and
// carries nothing of the connection string.
var safeDBSentinels = []error{
	driver.ErrBadConn,
	pq.ErrSSLNotSupported,
	pq.ErrSSLKeyHasWorldPermissions,
	pq.ErrSSLKeyUnknownOwnership,
	pq.ErrNotSupported,
	pq.ErrInFailedTransaction,
	pq.ErrCouldNotDetectUsername,
}

// isTLSVerificationError reports whether err is a certificate verification
// failure. Those name the certificate, its issuer and the host that was
// dialled -- the host is in the log already -- and are what an operator needs
// to see under sslmode=verify-ca or verify-full.
func isTLSVerificationError(err error) bool {
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return true
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return true
	}
	var hostname x509.HostnameError
	if errors.As(err, &hostname) {
		return true
	}
	var invalid x509.CertificateInvalidError
	return errors.As(err, &invalid)
}

// OpenDB opens the database handle and proves it can reach the database
// before the caller carries on.
//
// sql.Open on its own proves nothing: lib/pq's driver does not implement
// driver.DriverContext, so the DSN is stored and only handed to the driver
// when the first connection is made. Whatever is wrong with the connection --
// the host, the credentials, the sslmode, the database not being up yet --
// therefore surfaces at an arbitrary later moment, wrapped into whichever
// query happened to run first, and is reported again on every cycle for as
// long as the service runs. Connecting once here reduces that to one line and
// a refusal to start.
//
// Note this is a change in startup behaviour, not only in logging: a service
// used to start with the database down and recover on its first successful
// query, and now exits non-zero with no retry of its own. That assumes
// something restarts it.
//
// What keeps the connection string out of the resulting message is
// quoteDSNValue, not this check -- see RedactDBError for the rest.
func OpenDB(dsn string) (*sqlx.DB, error) {
	dbh, err := sqlx.Open("postgres", dsn)
	if err != nil {
		// unreachable for a DSN GetDSN built, since sql.Open does not parse
		// it, but the contract of sql.Open does not promise that
		return nil, fmt.Errorf("failed to open the database handle, %w", RedactDBError(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
	defer cancel()
	if err := dbh.PingContext(ctx); err != nil {
		_ = dbh.Close()
		return nil, fmt.Errorf("failed to reach the database, %w", RedactDBError(err))
	}

	return dbh, nil
}

func GetDSN() string {
	var (
		host, port, user, passwd, database string
		present                            bool
	)

	host, present = os.LookupEnv("DWS_DB_HOST")
	if !present {
		log.Fatal("DWS_DB_HOST variable not set")
	}
	port, present = os.LookupEnv("DWS_DB_PORT")
	if !present {
		log.Fatal("DWS_DB_PORT variable not set")
	}
	user, present = os.LookupEnv("DWS_DB_USER")
	if !present {
		log.Fatal("DWS_DB_USER variable not set")
	}
	passwd, present = os.LookupEnv("DWS_DB_PASSWORD")
	if !present {
		log.Fatal("DWS_DB_PASSWORD variable not set")
	}
	database, present = os.LookupEnv("DWS_DB_DATABASE")
	if !present {
		log.Fatal("DWS_DB_DATABASE variable not set")
	}
	sslmode := DefaultSSLMode
	if sm, ok := os.LookupEnv("DWS_DB_SSLMODE"); ok {
		// a stray newline out of a secret store must not turn into an
		// unsupported mode
		sslmode = strings.TrimSpace(sm)
	}
	if !validSSLModes[sslmode] {
		// lib/pq would otherwise report this as `unsupported sslmode "..."`
		// on the first connection attempt, i.e. at an arbitrary later moment
		log.Fatalf("DWS_DB_SSLMODE ('%s') is not one of %s", sslmode, sslModeList())
	}

	// every value is quoted, see quoteDSNValue: unquoted, a password
	// containing a space silently reassigns whichever key follows it.
	// connect_timeout is ours and is a plain integer.
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d TimeZone=UTC",
		quoteDSNValue(host), quoteDSNValue(port), quoteDSNValue(user),
		quoteDSNValue(passwd), quoteDSNValue(database), quoteDSNValue(sslmode),
		int64(DBConnectTimeout.Seconds()))

	log.Infof("host: '%s'", host)
	log.Infof("database: '%s'", database)
	log.Infof("sslmode: '%s'", sslmode)
	log.Infof("connect timeout: %v", DBConnectTimeout)
	return dsn
}

// DefaultSSLMode is the sslmode used when DWS_DB_SSLMODE is not set. It is the
// mode that requires encryption, so a deployment has to opt *out* of TLS
// rather than forget to opt in.
const DefaultSSLMode = "require"

// validSSLModes is the set lib/pq accepts (see its ssl.go). The empty string
// is deliberately excluded: lib/pq treats it as "require", but an explicitly
// empty DWS_DB_SSLMODE is far more likely to be a misconfiguration than a
// request for the default.
var validSSLModes = map[string]bool{
	"disable":     true,
	"require":     true,
	"verify-ca":   true,
	"verify-full": true,
}

// sslModeList renders validSSLModes for an error message, in a stable order.
func sslModeList() string {
	modes := make([]string, 0, len(validSSLModes))
	for m := range validSSLModes {
		modes = append(modes, m)
	}
	sort.Strings(modes)
	return strings.Join(modes, ", ")
}

// quoteDSNValue renders a connection parameter value as a lib/pq quoted
// string literal.
//
// The values used to be interpolated raw, and lib/pq's parseOpts reads an
// unquoted value up to the first space -- so a password containing a space
// followed by `key=value` silently *reassigned that key*. A password ending
// in ` host=elsewhere.invalid` sent the connection somewhere else and put the
// tail of the password into the resulting dial error; one ending in
// ` dbname=x` connected to a different database. Both are a connection nobody
// asked for, and both leak password material into an error message that looks
// entirely ordinary.
//
// Inside a single-quoted value parseOpts takes the character after a
// backslash literally and ends the value at the first unescaped quote, so
// escaping `\` and `'` is sufficient: no value can influence the parse, every
// value arrives as the key it was written for, and the malformed-DSN error
// that motivated OpenDB's connectivity check cannot occur for a DSN this
// function built.
func quoteDSNValue(v string) string {
	var b strings.Builder
	b.Grow(len(v) + 2)
	b.WriteByte('\'')
	for _, r := range v {
		if r == '\\' || r == '\'' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('\'')
	return b.String()
}
