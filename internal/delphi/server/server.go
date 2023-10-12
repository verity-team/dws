package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/verity-team/dws/api"
	"github.com/verity-team/dws/internal/delphi/db"
)

type stage struct {
	Limit int
	Price string
}

type DelphiServer struct {
	db *sqlx.DB
}

func NewDelphiServer(db *sqlx.DB) *DelphiServer {
	return &DelphiServer{
		db: db,
	}
}

func (s *DelphiServer) SetAffiliateCode(ctx echo.Context) error {
	var afc api.AffiliateRequest
	err := ctx.Bind(&afc)
	if err != nil {
		log.Error("failed to bind POST param (AffiliateRequest)")
		return err
	}
	return db.AddAffiliateCD(s.db, afc)
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
	sold, err := strconv.Atoi(dd.Stats.Tokens)
	if err != nil {
		err = fmt.Errorf("malformed tokens sold figure, %v", err)
		log.Error(err)
		return err
	}

	sps, err := saleParams()
	if err != nil {
		return err
	}

	// assume the highest price
	dd.Prices = append(dd.Prices, api.Price{
		Price: sps[len(sps)-1].Price,
		Asset: "truth",
		Ts:    time.Now().UTC(),
	})
	for _, sp := range sps {
		if sold < sp.Limit {
			dd.Prices[1].Price = sp.Price
			break
		}
	}
	return ctx.JSON(http.StatusOK, *dd)
}

func saleParams() ([]stage, error) {
	jsonString, present := os.LookupEnv("DWS_SALE_PARAMS")
	if !present {
		err := errors.New("DWS_SALE_PARAMS environment variable not set")
		log.Error(err)
		return nil, err
	}
	var sps []stage
	err := json.Unmarshal([]byte(jsonString), &sps)
	if err != nil {
		err = fmt.Errorf("error decoding JSON, %v", err)
		log.Error(err)
		return nil, err
	}

	return sps, nil
}

func (s *DelphiServer) UserData(ctx echo.Context, address string) error {
	dd, err := db.GetUserDonationData(s.db, address)
	if err != nil {
		return err
	}
	if dd == nil {
		return ctx.String(http.StatusNotFound, "no such address")
	}
	us, err := db.GetUserStats(s.db, address)
	if err != nil {
		return err
	}
	var result api.UserData = api.UserData{
		Donations: dd,
		Stats:     *us,
	}

	return ctx.JSON(http.StatusOK, result)
}
