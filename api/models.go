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

// Defines values for UserDataStatus.
const (
	None      UserDataStatus = "none"
	Staking   UserDataStatus = "staking"
	Unstaking UserDataStatus = "unstaking"
)

// AffiliateCode defines model for affiliate_code.
type AffiliateCode struct {
	// Address address of the wallet that requested the affiliate code
	Address string `db:"address" json:"address"`

	// Code affiliate code generated for the requesting wallet address
	Code string `db:"code" json:"code"`

	// Ts date/time at which the affiliate code was added
	Ts time.Time `db:"created_at" json:"ts"`
}

// ConnectionRequest defines model for connection_request.
type ConnectionRequest struct {
	// Address address of the wallet that connected
	Address string `db:"address" json:"address"`

	// Code affiliate code, pass `none` if there's no code in the URL
	Code string `db:"code" json:"code"`
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
	Ts     time.Time `db:"block_time" json:"ts"`

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
	Tokens string `db:"tokens" json:"tokens"`

	// Total total funds raised in USD
	Total string `db:"total" json:"total"`
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
	// AffiliateCode affiliate code generated for this wallet address
	AffiliateCode *string `db:"us_code" json:"affiliate_code,omitempty"`

	// Reward staking rewards the user is eligible to claim
	Reward string `db:"us_reward" json:"reward"`

	// Staked number of tokens the user staked; must be <= `tokens`
	Staked string         `db:"us_staked" json:"staked"`
	Status UserDataStatus `db:"us_status" json:"status"`

	// Tokens number of tokens the user is eligible to claim
	Tokens string `db:"us_tokens" json:"tokens"`

	// Total total funds donated by this address in USD
	Total string `db:"us_total" json:"total"`

	// Ts date/time at which the staking status was last modified
	Ts *time.Time `db:"us_modified_at" json:"ts,omitempty"`
}

// UserDataStatus defines model for UserData.Status.
type UserDataStatus string

// UserDataResult defines model for user_data_result.
type UserDataResult struct {
	// Donations an array of donations
	Donations []Donation `json:"donations"`
	UserData  UserData   `json:"user_data"`
}

// DelphiKey defines model for delphi_key.
type DelphiKey = string

// DelphiSignature defines model for delphi_signature.
type DelphiSignature = string

// DelphiTs defines model for delphi_ts.
type DelphiTs = string

// GenerateCodeParams defines parameters for GenerateCode.
type GenerateCodeParams struct {
	// DelphiKey a key/address (public)
	DelphiKey DelphiKey `json:"delphi-key"`

	// DelphiTs caller timestamp (number of milliseconds since Unix epoch) -- included
	// to prevent replay attacks; must not be older than 5 seconds
	DelphiTs DelphiTs `json:"delphi-ts"`

	// DelphiSignature signature over the path, timestamp and body
	DelphiSignature DelphiSignature `json:"delphi-signature"`
}

// ConnectWalletJSONRequestBody defines body for ConnectWallet for application/json ContentType.
type ConnectWalletJSONRequestBody = ConnectionRequest
