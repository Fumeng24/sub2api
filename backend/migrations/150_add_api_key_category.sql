-- API Key user-facing category. This is independent from scheduling group_id.

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS category varchar(20) NOT NULL DEFAULT 'other';

UPDATE api_keys
SET category = 'other'
WHERE category IS NULL OR category = '';

CREATE INDEX IF NOT EXISTS api_keys_category_idx ON api_keys (category);

COMMENT ON COLUMN api_keys.category IS 'User-facing API key category: openai, anthropic, or other; independent from group_id scheduling.';
