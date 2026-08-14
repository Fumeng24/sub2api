INSERT INTO scheduler_history (event_type, account_id, group_id, payload, created_at)
SELECT o.event_type, o.account_id, o.group_id, o.payload, o.created_at
FROM scheduler_outbox o
WHERE NOT EXISTS (
    SELECT 1
    FROM scheduler_history h
    WHERE h.event_type = o.event_type
      AND h.created_at = o.created_at
      AND h.account_id IS NOT DISTINCT FROM o.account_id
      AND h.group_id IS NOT DISTINCT FROM o.group_id
);
