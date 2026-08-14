-- Make runtime health the primary signal for OpenAI account scheduling.
-- Preserve an administrator's explicit override; only fill blank legacy
-- values so this migration is safe to apply on existing installations.
INSERT INTO settings (key, value)
VALUES
    ('openai_advanced_scheduler_weight_priority', '0.2'),
    ('openai_advanced_scheduler_weight_load', '0.8'),
    ('openai_advanced_scheduler_weight_queue', '0.4'),
    ('openai_advanced_scheduler_weight_error_rate', '4.5'),
    ('openai_advanced_scheduler_weight_ttft', '2.0')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = NOW()
WHERE settings.value IS NULL OR BTRIM(settings.value) = '';
