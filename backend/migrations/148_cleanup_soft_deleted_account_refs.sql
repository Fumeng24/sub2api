-- Clean up account-scoped scheduling rows left behind by direct DB soft deletes.
-- The normal application delete path removes account_groups before soft-deleting
-- accounts, but manual UPDATE accounts.deleted_at cannot trigger FK cascade.
WITH deleted_account_groups AS (
    DELETE FROM account_groups ag
    USING accounts a
    WHERE ag.account_id = a.id
      AND a.deleted_at IS NOT NULL
    RETURNING ag.group_id
),
deleted_scheduled_test_plans AS (
    DELETE FROM scheduled_test_plans stp
    USING accounts a
    WHERE stp.account_id = a.id
      AND a.deleted_at IS NOT NULL
    RETURNING stp.id
),
summary AS (
    SELECT
        (SELECT COUNT(*)::INT FROM deleted_account_groups) AS account_group_rows,
        (SELECT COUNT(*)::INT FROM deleted_scheduled_test_plans) AS scheduled_test_plan_rows,
        (
            SELECT COALESCE(
                jsonb_agg(DISTINCT group_id) FILTER (WHERE group_id IS NOT NULL),
                '[]'::jsonb
            )
            FROM deleted_account_groups
        ) AS group_ids
)
INSERT INTO scheduler_outbox (event_type, payload)
SELECT
    'full_rebuild',
    jsonb_build_object(
        'reason', 'cleanup_soft_deleted_account_refs',
        'account_group_rows', account_group_rows,
        'scheduled_test_plan_rows', scheduled_test_plan_rows,
        'group_ids', group_ids
    )
FROM summary
WHERE account_group_rows > 0 OR scheduled_test_plan_rows > 0;
