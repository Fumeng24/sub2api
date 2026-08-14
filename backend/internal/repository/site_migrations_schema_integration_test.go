//go:build integration

package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiteMigrationsSchemaIsUpToDate(t *testing.T) {
	tx := testTx(t)

	requireColumn(t, tx, "groups", "video_rate_independent", "boolean", 0, false)
	requireColumn(t, tx, "groups", "video_rate_multiplier", "numeric", 0, false)
	requireColumn(t, tx, "groups", "video_price_480p", "numeric", 0, true)
	requireColumn(t, tx, "groups", "video_price_720p", "numeric", 0, true)
	requireColumn(t, tx, "groups", "video_price_1080p", "numeric", 0, true)
	requireColumn(t, tx, "redeem_codes", "business_category", "character varying", 40, false)
	requireIndex(t, tx, "redeem_codes", "idx_redeem_codes_type_business_category")
}

func TestMigration148_CleansSoftDeletedAccountRefs(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)

	migrationPath := filepath.Join("..", "..", "migrations", "148_cleanup_soft_deleted_account_refs.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	var groupID, deletedAccountID, activeAccountID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO groups (name)
VALUES ($1)
RETURNING id
`, "migration-148-cleanup-soft-deleted-account-refs").Scan(&groupID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type, deleted_at)
VALUES ($1, $2, $3, NOW())
RETURNING id
`, "deleted-account", "openai", "oauth").Scan(&deletedAccountID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type)
VALUES ($1, $2, $3)
RETURNING id
`, "active-account", "openai", "oauth").Scan(&activeAccountID))

	_, err = tx.ExecContext(ctx, `
INSERT INTO account_groups (account_id, group_id, priority)
VALUES ($1, $2, 1), ($3, $2, 2)
`, deletedAccountID, groupID, activeAccountID)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
INSERT INTO scheduled_test_plans (account_id, model_id)
VALUES ($1, 'gpt-4o'), ($2, 'gpt-4o')
`, deletedAccountID, activeAccountID)
	require.NoError(t, err)

	var outboxBefore int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM scheduler_outbox
WHERE payload->>'reason' = 'cleanup_soft_deleted_account_refs'
`).Scan(&outboxBefore))

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var staleGroupRows, stalePlanRows, activeGroupRows, activePlanRows int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM account_groups
WHERE account_id = $1
`, deletedAccountID).Scan(&staleGroupRows))
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM scheduled_test_plans
WHERE account_id = $1
`, deletedAccountID).Scan(&stalePlanRows))
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM account_groups
WHERE account_id = $1 AND group_id = $2
`, activeAccountID, groupID).Scan(&activeGroupRows))
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM scheduled_test_plans
WHERE account_id = $1
`, activeAccountID).Scan(&activePlanRows))

	require.Zero(t, staleGroupRows)
	require.Zero(t, stalePlanRows)
	require.Equal(t, 1, activeGroupRows)
	require.Equal(t, 1, activePlanRows)

	var outboxAfter, accountGroupRows, scheduledTestPlanRows int
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM scheduler_outbox
WHERE payload->>'reason' = 'cleanup_soft_deleted_account_refs'
`).Scan(&outboxAfter))
	require.Equal(t, outboxBefore+1, outboxAfter)
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT
	(payload->>'account_group_rows')::INT,
	(payload->>'scheduled_test_plan_rows')::INT
FROM scheduler_outbox
WHERE payload->>'reason' = 'cleanup_soft_deleted_account_refs'
ORDER BY id DESC
LIMIT 1
`).Scan(&accountGroupRows, &scheduledTestPlanRows))
	require.Equal(t, 1, accountGroupRows)
	require.Equal(t, 1, scheduledTestPlanRows)
}
