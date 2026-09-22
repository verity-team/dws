package db

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testPageAddress = "0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154"

// a degenerate page is rejected before the query is issued -- the nil handle
// would panic if it were not
func TestGetUserDonationDataRejectsInvalidPage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		limit, offset int
	}{
		{"zero limit", 0, 0},
		{"negative limit", -1, 0},
		{"negative offset", 10, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result, err := GetUserDonationData(nil, testPageAddress, tc.limit, tc.offset)
				assert.Error(t, err)
				assert.Nil(t, result)
			})
		})
	}
}

// the affiliate code is only handed out over the signature protected
// /affiliate/code path; the unauthenticated user data query must not read it
func TestGetUserDataDoesNotSelectTheAffiliateCode(t *testing.T) {
	assert.NotContains(t, userDataQuery, "us_code")
	assert.Contains(t, userDataQuery, "update_user_data")
}

// the donation page is bounded in the SQL itself, not only in the caller, and
// ordered oldest on-chain first (block_time) with id as a deterministic
// tie-break so paging is stable -- see #231.
func TestUserDonationQueryIsPaginated(t *testing.T) {
	q := strings.ToUpper(userDonationQuery)
	assert.Contains(t, q, "LIMIT $2")
	assert.Contains(t, q, "OFFSET $3")
	assert.Contains(t, q, "ORDER BY BLOCK_TIME, ID")
}
