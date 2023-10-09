package server

import (
	"net/http"

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
	return ctx.JSON(http.StatusOK, *dd)
}

func (s *DelphiServer) UserData(ctx echo.Context, address string) error {
	return nil
}
