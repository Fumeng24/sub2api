-- Pricing fields present in the Group entity but missing from older installs.
-- All additions are nullable and idempotent so existing pricing is preserved.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS search_price_per_1k DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS audio_realtime_price_per_min DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS audio_tts_price_per_million_chars DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS audio_stt_price_per_hour DECIMAL(20,8);

COMMENT ON COLUMN groups.search_price_per_1k IS
    'Search/tool price per 1000 calls (USD)';
COMMENT ON COLUMN groups.audio_realtime_price_per_min IS
    'Realtime voice price per minute (USD)';
COMMENT ON COLUMN groups.audio_tts_price_per_million_chars IS
    'Text-to-speech price per million characters (USD)';
COMMENT ON COLUMN groups.audio_stt_price_per_hour IS
    'Speech-to-text price per hour (USD)';
