ALTER TABLE invoice_requests
    ADD COLUMN IF NOT EXISTS source_order_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source_orders_json JSONB NOT NULL DEFAULT '[]'::jsonb;
