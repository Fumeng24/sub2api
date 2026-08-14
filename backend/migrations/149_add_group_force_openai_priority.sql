-- Add per-group OpenAI priority forcing.
-- When enabled, /v1/responses requests for the group are forwarded with
-- service_tier=priority, while billing keeps the normal group multiplier.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS force_openai_priority boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN groups.force_openai_priority IS 'OpenAI 分组是否强制为 /v1/responses 请求启用 service_tier=priority；计费仍按普通倍率。';
