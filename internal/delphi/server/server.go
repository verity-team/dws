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
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/delphi/db"
)

const MAX_TIMESTAMP_AGE = 10

type DelphiServer struct {
	db *sqlx.DB
}

func NewDelphiServer(db *sqlx.DB) *DelphiServer {
	return &DelphiServer{
		db: db,
	}
}

func getError(code int, msg string, err error) (error, api.Error) {
	var (
		nerr error
		cerr api.Error
	)
	if msg != "" {
		nerr = fmt.Errorf(msg+", %v", err)
	} else {
		nerr = err
	}
	cerr = api.Error{
		Code:    code,
		Message: nerr.Error(),
	}
	return nerr, cerr
}

func (s *DelphiServer) ConnectWallet(ctx echo.Context) error {
	var cr api.ConnectionRequest
	err := ctx.Bind(&cr)
	if err != nil {
		err, cerr := getError(101, "failed to bind POST param (ConnectionRequest)", err)
		log.Error(err)
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	cr.Address = strings.ToLower(cr.Address)
	if err = db.ConnectWallet(s.db, cr); err != nil {
		err, cerr := getError(102, "failed to log wallet connection", err)
		log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	udr, err := s.getUserData(cr.Address)
	if err != nil {
		_, cerr := getError(103, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	if udr == nil {
		return ctx.NoContent(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, *udr)
}

func (s *DelphiServer) getUserData(address string) (*api.UserDataResult, error) {
	address = strings.ToLower(address)
	dd, err := db.GetUserDonationData(s.db, address)
	if err != nil {
		return nil, err
	}
	udata, err := db.GetUserData(s.db, address)
	if err != nil {
		return nil, err
	}
	if (dd == nil) && (udata == nil) {
		// no user data for the address given
		return nil, nil
	}

	result := api.UserDataResult{
		Donations: dd,
	}
	if udata != nil {
		result.UserData = *udata
	}

	return &result, nil
}

func (s *DelphiServer) UserData(ctx echo.Context, address string) error {
	udr, err := s.getUserData(address)
	if err != nil {
		_, cerr := getError(104, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}

	if udr == nil {
		return ctx.NoContent(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, *udr)
}

func (s *DelphiServer) Alive(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "{}\n")
}

func (s *DelphiServer) Ready(ctx echo.Context) error {
	err := s.db.Ping()
	if err != nil {
		return ctx.String(http.StatusServiceUnavailable, "{}\n")
	}
	return ctx.String(http.StatusOK, "{}\n")

}

func verifySig(from, msg, sigHex string) bool {
	sig, err := hexutil.Decode(sigHex)
	if err != nil {
		err = fmt.Errorf("invalid sig ('%s'), %v", sigHex, err)
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
		err = fmt.Errorf("failed to recover public key from sig ('%s'), %v", sigHex, err)
		log.Error(err)
		return false
	}

	recoveredAddr := crypto.PubkeyToAddress(*pk)
	return strings.EqualFold(from, recoveredAddr.Hex())
}

func getTS(tss string) (time.Time, error) {
	seconds, err := strconv.Atoi(tss)
	if err != nil {
		err = fmt.Errorf("failed to parse string with seconds since epoch ('%s'), %v", tss, err)
		log.Error(err)
		return time.Unix(0, 0), err
	}
	ts := time.Unix(int64(seconds), 0)
	return ts.UTC(), nil
}

func olderThan(ts time.Time, seconds int) bool {
	now := time.Now().UTC()
	tdif := now.Sub(ts.UTC())
	return tdif.Seconds() > float64(seconds)
}

func (s *DelphiServer) GenerateCode(ctx echo.Context, params api.GenerateCodeParams) error {
	ts, err := getTS(params.DelphiTs)
	if err != nil {
		_, cerr := getError(105, "", err)
		return ctx.JSON(http.StatusBadRequest, cerr)
	}
	if olderThan(ts, MAX_TIMESTAMP_AGE) {
		err = fmt.Errorf("/affiliate/code delphi-ts ('%s') is not recent enough for address '%s'", params.DelphiTs, params.DelphiKey)
		log.Error(err)
		cerr := api.Error{
			Code:    106,
			Message: "timestamp is not recent enough",
		}
		return ctx.JSON(http.StatusBadRequest, cerr)
	}

	path := strings.Join(strings.Split(ctx.Path(), "/")[1:], " ")
	msg := fmt.Sprintf("%s, %s", path, ts.Format("2006-01-02 15:04:05-07:00"))
	authOK := verifySig(params.DelphiKey, msg, params.DelphiSignature)
	if !authOK {
		return ctx.NoContent(http.StatusUnauthorized)
	}
	log.Infof("auth OK for /affiliate/code request, address '%s'", params.DelphiKey)
	afc, err := db.GetAffiliateCode(s.db, strings.ToLower(params.DelphiKey))
	if err != nil {
		_, cerr := getError(107, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	if afc != nil && afc.Code != "" {
		// we have an affiliate code for this address already
		return ctx.JSON(http.StatusOK, *afc)
	}
	afc, err = db.GenerateAffiliateCode(s.db, strings.ToLower(params.DelphiKey))
	if err != nil {
		_, cerr := getError(108, "", err)
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
		_, cerr := getError(109, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	ra, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		err = errors.New("DWS_DONATION_ADDRESS environment variable not set")
		log.Error(err)
		_, cerr := getError(110, "", err)
		return ctx.JSON(http.StatusInternalServerError, cerr)
	}
	dd.ReceivingAddress = ra
	// if we failed to fetch an ETH price, the status should be set to "paused"
	if dd.Prices[0].Price == "0.00" {
		dd.Status = api.Paused
	}
	return ctx.JSON(http.StatusOK, *dd)
}
