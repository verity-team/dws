package db

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"crypto/rand"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"github.com/shopspring/decimal"
	"github.com/verity-team/dws/api"
)

type walletConnection struct {
	Address string `db:"address" json:"address"`
	Code    string `db:"code" json:"code"`
}

func mostRecentWalletConnection(db *sqlx.DB, address string) (*walletConnection, error) {
	q1 := `
		SELECT
			address, code
		FROM wallet_connection
		WHERE address=$1
		ORDER BY id DESC
		LIMIT 1
		`
	var result walletConnection
	err := db.Get(&result, q1, address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		err = fmt.Errorf("failed to fetch most recent wallet connection for address '%s', %w", address, err)
		log.Error(err)
		return nil, err
	}
	return &result, nil
}

func ConnectWallet(db *sqlx.DB, req api.ConnectionRequest) error {
	// every read path lowercases the address -> normalize once, before the
	// lookup *and* the insert below, so that the two cannot disagree
	req.Address = strings.ToLower(req.Address)
	latest, err := mostRecentWalletConnection(db, req.Address)
	if err != nil {
		return err
	}
	if latest != nil && latest.Code == req.Code {
		// most recent wallet connection already has this affiliate code -> done
		return nil
	}
	var q string
	if req.Code != "none" {
		// we only want to insert a wallet_connection record if the
		// affiliate code exists in some user_data record in the database and
		// belongs to a *different* address: a donor must not refer himself.
		// The `ud.address <> :address` guard lives inside the INSERT ... SELECT
		// so the check and the write are one statement and cannot be raced --
		// a check-then-insert could see a third-party owner that a concurrent
		// affiliate-code change turns into the caller's own between the two.
		// :address is already lower cased (see above) and every address on
		// record is stored lower case, so the comparison is exact.
		q = `
		INSERT INTO wallet_connection (code, address)
		SELECT ud.affiliate_code AS code, :address
		FROM user_data AS ud
		WHERE ud.affiliate_code = :code
		  AND ud.address <> :address
	 `
	} else {
		// the user is connecting his wallet without an affiliate code
		q = `
		INSERT INTO wallet_connection (code, address) VALUES(:code, :address)
	 `
	}
	res, err := db.NamedExec(q, req)
	if err != nil {
		log.Errorf("failed to insert wallet connection data, %v", err)
		return err
	}
	ras, err := res.RowsAffected()
	if err != nil {
		log.Errorf("failed to get rows affected, %v", err)
		return err
	}
	if ras == 0 {
		// the insert matched no row: either the code does not exist, or it
		// belongs to the connecting address -- a rejected self-referral. One
		// lookup tells the two apart so the warning is accurate. (The caller
		// still gets a 200; surfacing a distinct error to the frontend is
		// tracked for a later release.)
		var owner string
		switch lerr := db.Get(&owner, `SELECT address FROM user_data WHERE affiliate_code = $1`, req.Code); {
		case lerr == nil && owner == req.Address:
			log.Warnf("rejected self-referral: address '%s' connected with its own affiliate code", req.Address)
		case errors.Is(lerr, sql.ErrNoRows):
			log.Warnf("wallet connection request with unknown affiliate code: '%s'", req.Code)
		default:
			log.Warnf("wallet connection request with an affiliate code that matched no user (code '%s'): %v", req.Code, lerr)
		}
	}
	return nil
}

func GetDonationData(db *sqlx.DB) (*api.DonationData, error) {
	// ethereum price
	q1 := `
		SELECT asset, price, created_at FROM price
		WHERE
			asset='eth'
			AND created_at > NOW() - INTERVAL '3 minutes'
		ORDER BY id DESC
		LIMIT 1
		`
	var (
		ethp    api.Price
		haveETH bool
	)
	switch ethErr := db.Get(&ethp, q1); {
	case ethErr == nil:
		haveETH = true
	case errors.Is(ethErr, sql.ErrNoRows):
		// pulitzer is behind/down. Reporting a zero valued price would let a
		// frontend price the donation widget at 0 USD; leave the ETH price
		// out instead -- the campaign is reported as 'paused'.
		log.Warn("no ETH price that is newer than 3 minutes, donations are paused")
	default:
		ethErr = fmt.Errorf("failed to fetch an ETH price that is newer than 3 minutes, %w", ethErr)
		log.Error(ethErr)
		return nil, ethErr
	}

	// truth token price
	q2 := `
		SELECT asset, price, created_at FROM price
		WHERE
			asset='truth'
		ORDER BY created_at DESC
		LIMIT 1
		`
	var truthp api.Price
	err := db.Get(&truthp, q2)
	if err != nil {
		err = fmt.Errorf("failed to fetch the TRUTH price, %w", err)
		log.Error(err)
		return nil, err
	}

	type dstats struct {
		Total  decimal.Decimal `db:"total"`
		Tokens int             `db:"tokens"`
		Status string          `db:"status"`
	}

	// donation stats
	q3 := `
		SELECT total, tokens, status FROM donation_stats
		ORDER BY created_at DESC
		LIMIT 1
		`
	var ds dstats
	err = db.Get(&ds, q3)
	if err != nil {
		err = fmt.Errorf("failed to fetch donation stats, %w", err)
		log.Error(err)
		return nil, err
	}

	var result api.DonationData
	if haveETH {
		result.Prices = append(result.Prices, ethp)
	}
	result.Prices = append(result.Prices, truthp)

	result.Stats = api.DonationStats{
		Total:  ds.Total.StringFixed(2),
		Tokens: strconv.Itoa(ds.Tokens),
	}

	result.Status = api.DonationDataStatus(ds.Status)
	return &result, nil
}

// userDonationQuery reads one page of the donation history of an address. The
// LIMIT is what keeps a single request from materializing an unbounded number
// of rows; the ORDER BY makes the paging stable.
const userDonationQuery = `
		SELECT
			amount, usd_amount, asset, tokens, price, tx_hash, status, block_time
		FROM donation
		WHERE address=$1
		ORDER BY id
		LIMIT $2 OFFSET $3
		`

// userDataQuery reads the donation summary of an address. us_code (the
// affiliate code) is deliberately *not* selected: this endpoint is
// unauthenticated and the referral code must only be handed out over the
// signature protected /affiliate/code path.
const userDataQuery = `
		SELECT
			 us_total,
			 us_tokens,
			 us_staked,
			 us_reward,
			 us_status,
			 us_modified_at
		FROM update_user_data($1)
		`

// GetUserDonationData returns one page of the donation history of the given
// address, oldest first. The page is bounded by the caller supplied limit so
// that a single request cannot materialize an unbounded number of rows.
func GetUserDonationData(db *sqlx.DB, address string, limit, offset int) ([]api.Donation, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid donation page size (%d) for address '%s'", limit, address)
	}
	if offset < 0 {
		return nil, fmt.Errorf("invalid donation page offset (%d) for address '%s'", offset, address)
	}
	var result []api.Donation
	err := db.Select(&result, userDonationQuery, address, limit, offset)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to fetch donation records for %s, %w", address, err)
		log.Error(err)
		return nil, err
	}

	return result, nil
}

func GetUserData(db *sqlx.DB, address string) (*api.UserData, error) {
	// fetch donations made by this user/address
	var result api.UserData
	err := db.Get(&result, userDataQuery, address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		err = fmt.Errorf("failed to fetch user data for %s, %w", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}

func GetAffiliateCode(db *sqlx.DB, address string) (*api.AffiliateCode, error) {
	// fetch the afiliate code for the given address
	q1 := `
		SELECT
			address, COALESCE(affiliate_code, '') AS code, created_at
		FROM user_data
		WHERE address=$1
		`
	var result api.AffiliateCode
	err := db.Get(&result, q1, address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		err = fmt.Errorf("failed to fetch affiliate code for address '%s', %w", address, err)
		log.Error(err)
		return nil, err
	}

	return &result, nil
}

func genAFC() (string, error) {
	buff := make([]byte, 8)
	if _, err := rand.Read(buff); err != nil {
		err = fmt.Errorf("failed to generate an affiliate code, %w", err)
		log.Error(err)
		return "", err
	}
	return hex.EncodeToString(buff), nil
}

func GenerateAffiliateCode(db *sqlx.DB, address string) (*api.AffiliateCode, error) {
	// fetch donations made by this user/address
	q1 := `
		INSERT INTO user_data AS ud(address, affiliate_code)
		VALUES($1, $2)
		ON CONFLICT(address)
		DO UPDATE SET
			 affiliate_code = EXCLUDED.affiliate_code
			 WHERE ud.affiliate_code IS NULL
		RETURNING address, affiliate_code AS code, created_at
		`

	var result api.AffiliateCode
	afc, err := genAFC()
	if err != nil {
		return nil, err
	}
	err = db.QueryRowx(q1, address, afc).StructScan(&result)
	if err == nil {
		return &result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("failed to set affiliate code for '%s', %w", address, err)
		log.Error(err)
		return nil, err
	}

	// the ON CONFLICT guard (affiliate_code IS NULL) did not hold -> this
	// address has an affiliate code already, generated by a request that beat
	// us to it. Return the code that is on record.
	existing, err := GetAffiliateCode(db, address)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.Code == "" {
		err = fmt.Errorf("failed to set affiliate code for '%s', no code on record", address)
		log.Error(err)
		return nil, err
	}

	return existing, nil
}
