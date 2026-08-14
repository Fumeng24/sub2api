CREATE TABLE IF NOT EXISTS scheduler_history (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    account_id BIGINT NULL,
    group_id BIGINT NULL,
    payload JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_history_created_at ON scheduler_history (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduler_history_group_id_created_at ON scheduler_history (group_id, created_at DESC) WHERE group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheduler_history_account_id_created_at ON scheduler_history (account_id, created_at DESC) WHERE account_id IS NOT NULL;
