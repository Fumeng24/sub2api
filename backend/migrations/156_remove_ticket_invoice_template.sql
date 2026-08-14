UPDATE settings
SET value = jsonb_set(
    value::jsonb,
    '{templates}',
    COALESCE((
        SELECT jsonb_agg(template)
        FROM jsonb_array_elements(COALESCE(value::jsonb -> 'templates', '[]'::jsonb)) AS template
        WHERE template ->> 'key' <> 'billing_invoice_request'
    ), '[]'::jsonb),
    true
)::text,
updated_at = NOW()
WHERE key = 'ticket_system_config'
  AND value IS NOT NULL
  AND value <> ''
  AND jsonb_typeof(value::jsonb -> 'templates') = 'array'
  AND value::jsonb -> 'templates' @> '[{"key":"billing_invoice_request"}]'::jsonb;
