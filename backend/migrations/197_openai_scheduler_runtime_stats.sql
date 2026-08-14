-- Durable model-scoped runtime health used to resume OpenAI scheduling after
-- a process restart. Long-term group ranking still comes from usage/ops
-- experience statistics; this table only preserves the short-term EWMA.
CREATE TABLE IF NOT EXISTS openai_account_scheduler_runtime_stats (
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    canonical_model TEXT NOT NULL,
    error_rate_ewma DOUBLE PRECISION NOT NULL DEFAULT 0,
    ttft_ewma       DOUBLE PRECISION NULL,
    sample_count    BIGINT NOT NULL DEFAULT 0,
    ttft_samples    BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    transient_failure_streak BIGINT NOT NULL DEFAULT 0,
    transient_last_failure_at TIMESTAMPTZ NULL,
    transient_block_until TIMESTAMPTZ NULL,
    slow_reserve_marked_at TIMESTAMPTZ NULL,
    slow_reserve_last_touched_at TIMESTAMPTZ NULL,
    slow_reserve_expires_at TIMESTAMPTZ NULL,
    slow_reserve_reason TEXT NULL,
    slow_reserve_ttft_ms INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, canonical_model)
);

CREATE INDEX IF NOT EXISTS idx_openai_scheduler_runtime_stats_updated
    ON openai_account_scheduler_runtime_stats (updated_at DESC);
