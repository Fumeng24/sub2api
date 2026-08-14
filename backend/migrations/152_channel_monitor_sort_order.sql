ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_sort_order_id
    ON channel_monitors (sort_order ASC, id DESC);
