package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/delphi/db"
)

type DelphiServer struct {
	db *sqlx.DB
}

func NewDelphiServer(db *sqlx.DB) *DelphiServer {
	return &DelphiServer{
		db: db,
	}
}

func (s *DelphiServer) ConnectWallet(ctx echo.Context) error {
	var cr api.ConnectionRequest
	err := ctx.Bind(&cr)
	if err != nil {
		log.Error("failed to bind POST param (ConnectionRequest)")
		return err
	}
	cr.Address = strings.ToLower(cr.Address)
	if err = db.ConnectWallet(s.db, cr); err != nil {
		log.Error("failed to log wallet connection")
		return err
	}
	udr, err := s.getUserData(cr.Address)
	if err != nil {
		return err
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
		return err
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

func (s *DelphiServer) GenerateCode(ctx echo.Context, params api.GenerateCodeParams) error {
	return nil
}

func (s *DelphiServer) DonationData(ctx echo.Context) error {
	dd, err := db.GetDonationData(s.db)
	if err != nil {
		return err
	}
	ra, present := os.LookupEnv("DWS_DONATION_ADDRESS")
	if !present {
		err = errors.New("DWS_DONATION_ADDRESS environment variable not set")
		log.Error(err)
		return err
	}
	dd.ReceivingAddress = ra
	// if we failed to fetch an ETH price, the status should be set to "paused"
	if dd.Prices[0].Price == "0.00" {
		dd.Status = api.Paused
	}
	return ctx.JSON(http.StatusOK, *dd)
}

func verifySig(from, msg, sigHex string) (bool, error) {
	sig, err := hexutil.Decode(sigHex)
	if err != nil {
		err = fmt.Errorf("invalid sig ('%s'), %v", sigHex, err)
		log.Error(err)
		return false, err
	}

	msgHash := accounts.TextHash([]byte(msg))
	if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
		sig[crypto.RecoveryIDOffset] -= 27
	}

	recovered, err := crypto.SigToPub(msgHash, sig)
	if err != nil {
		err = fmt.Errorf("failed to recover publick key from signature, %v", err)
		log.Error(err)
		return false, err
	}

	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	result := strings.EqualFold(from, recoveredAddr.Hex())
	return result, nil
}
