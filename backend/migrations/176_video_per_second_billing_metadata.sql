-- Persist video generation billing metadata and keep image-size checks scoped
-- to non-video image rows.

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS video_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS video_resolution VARCHAR(10);

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS video_duration_seconds INTEGER;

ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_image_billing_size_check;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_billing_size_check
    CHECK (
        image_count <= 0
        OR video_count > 0
        OR billing_mode = 'video'
        OR (
            image_size IS NOT NULL
            AND image_size IN ('1K', '2K', '4K', 'mixed')
        )
    ) NOT VALID;
