-- Runtime scheduling state is account-scoped. The old per-group cooldown map
-- could make a shared account look healthy in one group while it was unhealthy
-- in another, so remove it and refresh affected scheduler snapshots.
WITH affected AS (
    UPDATE accounts
    SET extra = COALESCE(extra, '{}'::jsonb) - 'group_temp_unschedulable',
        updated_at = NOW()
    WHERE COALESCE(extra, '{}'::jsonb) ? 'group_temp_unschedulable'
    RETURNING id
)
INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload)
SELECT
    'account_changed',
    id,
    NULL,
    jsonb_build_object('reason', 'remove_legacy_group_temp_unschedulable')
FROM affected;
