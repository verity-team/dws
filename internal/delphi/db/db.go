package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"github.com/verity-team/dws/api"
)

// AddAffiliateCD adds a donation made by a user with an affiliate code to the
// database.
func AddAffiliateCD(db *sqlx.DB, afc api.AffiliateRequest) error {
	q := `
		 INSERT INTO afc_donation(afc, tx_hash)
		 VALUES(:afc, :tx_hash)
	 `
	qd := map[string]interface{}{
		"afc":     afc.Code,
		"tx_hash": afc.TxHash,
	}
	_, err := db.NamedExec(q, qd)
	if err != nil {
		log.Errorf("Failed to insert affiliate donation data, %v", err)
		return err
	}

	return nil
}

func GetDonationData() (*api.DonationData, error) {
	return nil, nil
}
