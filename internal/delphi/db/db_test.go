package db

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

const schemaFile = "../../../deployments/db/01-schema.sql"

// schemaGuard returns the schema file and its destructive-reload guard: the
// first DO block.
func schemaGuard(t *testing.T) (schema, guard string) {
	t.Helper()
	b, err := os.ReadFile(schemaFile)
	require.NoError(t, err)
	schema = string(b)
	start := strings.Index(schema, "DO $$")
	require.GreaterOrEqual(t, start, 0, "no guard in %s", schemaFile)
	end := strings.Index(schema[start:], "$$;")
	require.Positive(t, end)
	return schema, schema[start : start+end+len("$$;")]
}

// AUD4-24: the guard has to cover every table the script drops, and it has to
// run before the first of them is dropped
func TestSchemaGuardCoversEveryTable(t *testing.T) {
	schema, guard := schemaGuard(t)

	firstDrop := strings.Index(schema, "DROP ")
	require.Positive(t, firstDrop)
	assert.Less(t, strings.Index(schema, guard), firstDrop, "the guard must come before the first DROP")
	// nothing but comments and the psql meta-command before it
	for _, line := range strings.Split(schema[:strings.Index(schema, guard)], "\n") {
		line = strings.TrimSpace(line)
		assert.True(t, line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, `\set `), "statement before the guard: %q", line)
	}

	tables := regexp.MustCompile(`(?m)^CREATE TABLE (\w+)`).FindAllStringSubmatch(schema, -1)
	require.NotEmpty(t, tables)
	for _, m := range tables {
		assert.Contains(t, guard, "('"+m[1]+"',", "table %s is not guarded", m[1])
	}
}

// updateUserDataFunc returns the update_user_data() definition in s.
func updateUserDataFunc(t *testing.T, s string) string {
	t.Helper()
	f := regexp.MustCompile(`(?s)CREATE OR REPLACE FUNCTION update_user_data\(.*?\$\$ LANGUAGE plpgsql;`).FindString(s)
	require.NotEmpty(t, f)
	return f
}

// AUD4-25: the per-address lock is taken before anything is read, and the
// migration statement in docs/database-schema-changes.md is the function the
// schema defines
func TestUpdateUserDataLocksBeforeSumming(t *testing.T) {
	schema, _ := schemaGuard(t)
	f := updateUserDataFunc(t, schema)

	lock := strings.Index(f, "PERFORM pg_advisory_xact_lock(hashtext('update_user_data'), hashtext(p_address));")
	require.Positive(t, lock, "no per-address transaction lock")
	assert.Less(t, lock, strings.Index(f, "IF EXISTS"))
	assert.Less(t, lock, strings.Index(f, "SUM("))

	doc, err := os.ReadFile("../../../docs/database-schema-changes.md")
	require.NoError(t, err)
	assert.Equal(t, f, updateUserDataFunc(t, string(doc)), "docs/database-schema-changes.md and 01-schema.sql disagree about update_user_data()")
}
