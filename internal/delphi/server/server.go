package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/shopspring/decimal"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/common"
	"github.com/verity-team/dws/internal/delphi/db"
)

const (
	// MaxTimestampAge is how far in the past a `delphi-ts` timestamp may lie.
	// A wallet signature prompt is a human interaction, so the window has to
	// leave room for the user to read the message and click "sign".
	MaxTimestampAge = 30
	// MaxTimestampAgeCeiling caps whatever DWS_MAX_TIMESTAMP_AGE asks for; the
	// replay window must not be widened without bound by a stray environment
	// variable.
	MaxTimestampAgeCeiling = 300
	// MaxTimestampSkew is how far in the future a `delphi-ts` timestamp may
	// lie. Without an upper bound a signature over a far future timestamp
	// stays valid -- and replayable -- for as long as that timestamp is in the
	// future. The allowance only covers clock drift between the caller and us.
	MaxTimestampSkew = 5
	// MaxUserDataLimit caps the number of donation records a single
	// /user/data/{address} call may return.
	MaxUserDataLimit = 100
	// DefaultUserDataLimit is the page size used when the caller does not ask
	// for one.
	DefaultUserDataLimit = 50
)

// The messages handed to the caller are deliberately generic: the wrapped
// error carries table, function and column names as well as the postgres
// dialect and the driver version, none of which an internet client gets to
// see. The numeric code stays in the response so that a report can be
// correlated with the server log.
const (
	msgInvalidRequest = "invalid request"
	msgInternalError  = "internal error"
)

var (
	bts, rev, version string
)

type DelphiServer struct {
	db *sqlx.DB
}

func NewDelphiServer(db *sqlx.DB) *DelphiServer {
	return &DelphiServer{
		db: db,
	}
}

// getError logs the internal error -- with the context the caller passes in --
// and returns the payload the client gets: the numeric error code and a fixed,
// generic message.
func getError(code int, publicMsg, context string, err error) api.Error {
	nerr := err
	if context != "" {
		nerr = fmt.Errorf("%s, %w", context, err)
	}
	log.Errorf("delphi error %d: %v", code, nerr)
	return api.Error{
		Code:    code,
		Message: publicMsg,
	}
}

func (s *DelphiServer) ConnectWallet(ctx echo.Context, params api.ConnectWalletParams) error {
	authTS, err := getTS(params.DelphiTs)
	if err != nil {
		cerr := getError(113, msgInvalidRequest, "failed to parse the /wallet/connection delphi-ts header", err)
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	if authTSOutOfWindow(authTS) {
		err = fmt.Errorf("/wallet/connection delphi-ts ('%s') is outside the accepted window for address '%s'", params.DelphiTs, params.DelphiKey)
		log.Error(err)
		cerr := api.Error{
			Code:    114,
			Message: "timestamp is not recent enough",
		}
		return ctx.JSON(http.StatusBadRequest, cerr)
	}

	authOK := verifySig(params.DelphiKey, formMsg(ctx.Path(), params.DelphiKey, authTS), params.DelphiSignature)
	if !authOK {
		return ctx.NoContent(http.StatusUnauthorized)
	}
	log.Infof("auth OK for /wallet/connection request, address '%s'", params.DelphiKey)

	var cr api.ConnectionRequest
	err = ctx.Bind(&cr)
	if err != nil {
		cerr := getError(101, msgInvalidRequest, "failed to bind POST param (ConnectionRequest)", err)
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	if !common.IsValidETHAddress(cr.Address) {
		cerr := api.Error{Code: 102, Message: "invalid ethereum address"}
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	// the caller may only connect the wallet it has proven ownership of
	if !strings.EqualFold(cr.Address, params.DelphiKey) {
		log.Errorf("/wallet/connection address mismatch, delphi-key '%s'", params.DelphiKey)
		return ctx.NoContent(http.StatusUnauthorized)
	}
	// every read path lowercases the address; do the same before writing so
	// the duplicate detection in db.ConnectWallet can match
	cr.Address = strings.ToLower(cr.Address)
	if err = db.ConnectWallet(s.db, cr); err != nil {
		cerr := getError(104, msgInternalError, "failed to log wallet connection", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	return ctx.JSON(http.StatusOK, struct{}{})
}

func (s *DelphiServer) getUserData(address string, limit, offset int) (*api.UserDataResult, error) {
	var result api.UserDataResult
	address = strings.ToLower(address)
	dd, err := db.GetUserDonationData(s.db, address, limit, offset)
	if err != nil {
		return nil, err
	}
	udata, err := db.GetUserData(s.db, address)
	if err != nil {
		return nil, err
	}
	if (dd == nil) && (udata == nil) {
		// no user data for the address given
		return &result, nil
	}

	result.Donations = dd
	if udata != nil {
		result.UserData = *udata
	}

	return &result, nil
}

// pageParams turns the optional limit/offset query parameters into a page the
// server is willing to serve. The caller cannot ask for more than
// MaxUserDataLimit records, so the size of the response -- and of the database
// read behind it -- stays bounded no matter what is passed in.
func pageParams(params api.UserDataParams) (limit, offset int) {
	limit = DefaultUserDataLimit
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit < 1 {
		limit = 1
	}
	if limit > MaxUserDataLimit {
		limit = MaxUserDataLimit
	}
	if params.Offset != nil && *params.Offset > 0 {
		offset = *params.Offset
	}
	return limit, offset
}

func (s *DelphiServer) UserData(ctx echo.Context, address string, params api.UserDataParams) error {
	if !common.IsValidETHAddress(address) {
		cerr := api.Error{Code: 103, Message: "invalid ethereum address"}
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	limit, offset := pageParams(params)
	udr, err := s.getUserData(address, limit, offset)
	if err != nil {
		cerr := getError(106, msgInternalError, "failed to fetch user data", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	return ctx.JSON(http.StatusOK, *udr)
}

func (s *DelphiServer) Alive(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "{}\n")
}

func (s *DelphiServer) Ready(ctx echo.Context) error {
	if err := s.db.Ping(); err != nil {
		return ctx.String(http.StatusServiceUnavailable, "{}\n")
	}
	return ctx.String(http.StatusOK, "{}\n")

}

func (s *DelphiServer) Version(ctx echo.Context) error {
	version = fmt.Sprintf("delphi::%s::%s", bts, rev)
	log.Info("version = ", version)
	return ctx.JSON(http.StatusOK, map[string]string{"version": version})
}

func verifySig(from, msg, sigHex string) bool {
	sig, err := hexutil.Decode(sigHex)
	if err != nil {
		err = fmt.Errorf("invalid sig ('%s'), %w", sigHex, err)
		log.Error(err)
		return false
	}

	if len(sig) != crypto.SignatureLength {
		err = fmt.Errorf("invalid sig length (%d), expected %d", len(sig), crypto.SignatureLength)
		log.Error(err)
		return false
	}

	msgHash := accounts.TextHash([]byte(msg))
	// ethereum "black magic" :(
	if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
		sig[crypto.RecoveryIDOffset] -= 27
	}

	pk, err := crypto.SigToPub(msgHash, sig)
	if err != nil {
		err = fmt.Errorf("failed to recover public key from sig ('%s'), %w", sigHex, err)
		log.Error(err)
		return false
	}

	recoveredAddr := crypto.PubkeyToAddress(*pk)
	return strings.EqualFold(from, recoveredAddr.Hex())
}

func getTS(tss string) (time.Time, error) {
	seconds, err := strconv.Atoi(tss)
	if err != nil {
		err = fmt.Errorf("failed to parse string with seconds since epoch ('%s'), %w", tss, err)
		log.Error(err)
		return time.Unix(0, 0), err
	}
	ts := time.Unix(int64(seconds), 0)
	return ts.UTC(), nil
}

// maxTSAge returns the accepted age of a `delphi-ts` timestamp. The value can
// be lowered or raised via DWS_MAX_TIMESTAMP_AGE but never beyond
// MaxTimestampAgeCeiling: an unbounded window is an unbounded replay window.
func maxTSAge() int {
	envVar, present := os.LookupEnv("DWS_MAX_TIMESTAMP_AGE")
	if !present {
		return MaxTimestampAge
	}
	val, err := strconv.Atoi(envVar)
	if err != nil || val <= 0 {
		log.Errorf("invalid DWS_MAX_TIMESTAMP_AGE ('%s'), falling back to %d seconds", envVar, MaxTimestampAge)
		return MaxTimestampAge
	}
	if val > MaxTimestampAgeCeiling {
		log.Warnf("DWS_MAX_TIMESTAMP_AGE (%d) exceeds the %d second ceiling, capping it", val, MaxTimestampAgeCeiling)
		return MaxTimestampAgeCeiling
	}
	return val
}

// authTSOutOfWindow reports whether the caller supplied timestamp lies outside
// the accepted window. The window is bounded on *both* sides: a timestamp in
// the future yields a negative age and would otherwise pass forever, turning a
// single harvested signature into a signature that stays replayable until that
// timestamp has come and gone.
func authTSOutOfWindow(authTS time.Time) bool {
	age := time.Now().UTC().Sub(authTS.UTC()).Seconds()
	if age > float64(maxTSAge()) {
		return true
	}
	return age < -float64(MaxTimestampSkew)
}

// formMsg builds the message the caller has to sign: the words that make up
// the endpoint path, the address the caller claims to own and the timestamp.
// Binding the address in makes the signature specific to the account it is
// used for instead of being a bare "path + time" token.
func formMsg(urlPath, address string, authTS time.Time) string {
	parts := strings.Split(urlPath, "/")[1:]
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	path := strings.Join(nonEmpty, " ")
	return fmt.Sprintf("%s, %s, %s", path, strings.ToLower(address), authTS.Format("2006-01-02 15:04:05-07:00"))
}

func (s *DelphiServer) GenerateCode(ctx echo.Context, params api.GenerateCodeParams) error {
	authTS, err := getTS(params.DelphiTs)
	if err != nil {
		cerr := getError(107, msgInvalidRequest, "failed to parse the /affiliate/code delphi-ts header", err)
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	if authTSOutOfWindow(authTS) {
		err = fmt.Errorf("/affiliate/code delphi-ts ('%s') is outside the accepted window for address '%s'", params.DelphiTs, params.DelphiKey)
		log.Error(err)
		cerr := api.Error{
			Code:    108,
			Message: "timestamp is not recent enough",
		}
		return ctx.JSON(http.StatusBadRequest, cerr)
	}

	authOK := verifySig(params.DelphiKey, formMsg(ctx.Path(), params.DelphiKey, authTS), params.DelphiSignature)
	if !authOK {
		return ctx.NoContent(http.StatusUnauthorized)
	}
	log.Infof("auth OK for /affiliate/code request, address '%s'", params.DelphiKey)
	afc, err := db.GetAffiliateCode(s.db, strings.ToLower(params.DelphiKey))
	if err != nil {
		cerr := getError(109, msgInternalError, "failed to fetch the affiliate code", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	if afc != nil && afc.Code != "" {
		// we have an affiliate code for this address already
		return ctx.JSON(http.StatusOK, *afc)
	}
	afc, err = db.GenerateAffiliateCode(s.db, strings.ToLower(params.DelphiKey))
	if err != nil {
		cerr := getError(110, msgInternalError, "failed to generate an affiliate code", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	if afc != nil {
		// return the newly generated affiliate code
		return ctx.JSON(http.StatusOK, *afc)
	}
	return nil
}

func (s *DelphiServer) DonationData(ctx echo.Context) error {
	dd, err := db.GetDonationData(s.db)
	if err != nil {
		cerr := getError(111, msgInternalError, "failed to fetch donation data", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	ra, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		err = errors.New("DWS_DONATION_ADDRESS environment variable not set")
		cerr := getError(112, msgInternalError, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	dd.ReceivingAddress = ra
	// if we failed to fetch an ETH price and the campaign is not closed yet,
	// the status should be set to "paused"
	if dd.Status != api.Closed && !haveETHPrice(dd.Prices) {
		dd.Status = api.Paused
	}
	return ctx.JSON(http.StatusOK, *dd)
}

// haveETHPrice reports whether the prices given carry a usable ETH price i.e.
// one a frontend can price a donation with.
//
// The price is picked by asset (the order in which the prices are assembled is
// not part of the API) and compared numerically: it is scanned from a
// NUMERIC(15,5) column and rendered as e.g. "0.00000" by postgres, a string
// comparison against "0.00" never matches.
func haveETHPrice(prices []api.Price) bool {
	for _, p := range prices {
		if p.Asset != api.PriceAssetEth {
			continue
		}
		price, err := decimal.NewFromString(p.Price)
		if err != nil {
			log.Errorf("invalid ETH price '%s', %v", p.Price, err)
			return false
		}
		return price.IsPositive()
	}
	log.Warn("no ETH price available")
	return false
}
