package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamManagementMigrationDoesNotAutoBindExistingAccounts(t *testing.T) {
	content, err := FS.ReadFile("191_add_upstream_management.sql")
	require.NoError(t, err)

	sql := strings.ToLower(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "alter table accounts add column if not exists upstream_id bigint null")
	require.NotContains(t, sql, "update accounts")
	require.NotContains(t, sql, "insert into upstreams")
}
