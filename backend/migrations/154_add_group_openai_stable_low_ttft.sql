ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS openai_stable_low_ttft BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE groups
SET openai_stable_low_ttft = TRUE
WHERE platform = 'openai'
  AND name LIKE '%稳定%'
  AND deleted_at IS NULL;
