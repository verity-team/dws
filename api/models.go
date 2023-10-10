// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.15.0 DO NOT EDIT.
package api

import (
	"time"
)

// Defines values for DonationAsset.
const (
	DonationAssetEth  DonationAsset = "eth"
	DonationAssetUsdc DonationAsset = "usdc"
	DonationAssetUsdt DonationAsset = "usdt"
)

// Defines values for DonationStatus.
const (
	Confirmed   DonationStatus = "confirmed"
	Failed      DonationStatus = "failed"
	Unconfirmed DonationStatus = "unconfirmed"
)

// Defines values for DonationDataStatus.
const (
	Closed DonationDataStatus = "closed"
	Open   DonationDataStatus = "open"
	Paused DonationDataStatus = "paused"
)

// Defines values for PriceAsset.
const (
	PriceAssetEth   PriceAsset = "eth"
	PriceAssetTruth PriceAsset = "truth"
)

// Defines values for UserStatsStatus.
const (
	None      UserStatsStatus = "none"
	Staking   UserStatsStatus = "staking"
	Unstaking UserStatsStatus = "unstaking"
)

// AffiliateRequest defines model for affiliate_request.
type AffiliateRequest struct {
	// Code affiliate code for the donation in question
	Code string `json:"code"`

	// TxHash transaction hash for the donation in question
	TxHash string `json:"tx_hash"`
}

// Donation defines model for donation.
type Donation struct {
	// Amount amount donated
	Amount string `db:"amount" json:"amount"`

	// Asset asset donated
	Asset DonationAsset `db:"asset" json:"asset"`

	// Price token price at the time of donation
	Price string `db:"price" json:"price"`

	// Status ethereum blockchain status of the donation
	Status DonationStatus `db:"status" json:"status"`

	// Tokens amount of tokens corresponding to the donated amount
	Tokens string    `db:"tokens" json:"tokens"`
	Ts     time.Time `db:"modified_at" json:"ts"`

	// TxHash transaction hash for the donation in question
	TxHash string `db:"tx_hash" json:"tx_hash"`

	// UsdAmount optional USD amount, omitted in case of USD stable coins
	UsdAmount *string `db:"usd_amount" json:"usd_amount,omitempty"`
}

// DonationAsset asset donated
type DonationAsset string

// DonationStatus ethereum blockchain status of the donation
type DonationStatus string

// DonationData defines model for donation_data.
type DonationData struct {
	// Prices an array of prices
	Prices []Price `json:"prices"`

	// ReceivingAddress receiving address for donations
	ReceivingAddress string             `json:"receiving_address"`
	Stats            DonationStats      `json:"stats"`
	Status           DonationDataStatus `json:"status"`
}

// DonationDataStatus defines model for DonationData.Status.
type DonationDataStatus string

// DonationStats defines model for donation_stats.
type DonationStats struct {
	// Tokens number of tokens claimable by donors
	Tokens string `json:"tokens"`

	// Total total funds raised in USD
	Total string `json:"total"`
}

// Error defines model for error.
type Error struct {
	// Code error code
	Code int `json:"code"`

	// Message error message
	Message string `json:"message"`
}

// Price defines model for price.
type Price struct {
	Asset PriceAsset `db:"asset" json:"asset"`

	// Price asset price in USD
	Price string    `db:"price" json:"price"`
	Ts    time.Time `db:"created_at" json:"ts"`
}

// PriceAsset defines model for Price.Asset.
type PriceAsset string

// UserData defines model for user_data.
type UserData struct {
	// Donations an array of donations
	Donations []Donation `json:"donations"`
	Stats     UserStats  `json:"stats"`
}

// UserStats defines model for user_stats.
type UserStats struct {
	// Reward staking rewards the user is eligible to claim
	Reward string `db:"us_reward" json:"reward"`

	// Staked number of tokens the user staked; must be <= `tokens`
	Staked string          `db:"us_staked" json:"staked"`
	Status UserStatsStatus `db:"us_status" json:"status"`

	// Tokens number of tokens the user is eligible to claim
	Tokens string `db:"us_tokens" json:"tokens"`

	// Total total funds donated by this address in USD
	Total string `db:"us_total" json:"total"`

	// Ts date/time at which the staking status was last modified
	Ts *time.Time `db:"us_modified_at" json:"ts,omitempty"`
}

// UserStatsStatus defines model for UserStats.Status.
type UserStatsStatus string

// SetAffiliateCodeJSONRequestBody defines body for SetAffiliateCode for application/json ContentType.
type SetAffiliateCodeJSONRequestBody = AffiliateRequest
