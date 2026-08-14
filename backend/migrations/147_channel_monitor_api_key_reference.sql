ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS api_key_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitors_api_key_id_fkey'
          AND table_name = 'channel_monitors'
    ) THEN
        ALTER TABLE channel_monitors
            ADD CONSTRAINT channel_monitors_api_key_id_fkey
            FOREIGN KEY (api_key_id)
            REFERENCES api_keys (id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_api_key_id
    ON channel_monitors (api_key_id)
    WHERE api_key_id IS NOT NULL;

COMMENT ON COLUMN channel_monitors.api_key_id IS 'Linked user API key ID; runtime prefers current api_keys.key when present';
