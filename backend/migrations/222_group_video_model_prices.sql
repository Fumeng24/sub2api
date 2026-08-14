-- Per-model-family video pricing overrides used by the billing and admin APIs.
-- Keep this additive so installations that already have the column are safe.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS video_model_prices JSONB;

COMMENT ON COLUMN groups.video_model_prices IS
    'Optional per-model-family and resolution video pricing overrides';
