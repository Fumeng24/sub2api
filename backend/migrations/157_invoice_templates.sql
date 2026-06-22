CREATE TABLE IF NOT EXISTS invoice_templates (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(80) NOT NULL DEFAULT '',
    invoice_type VARCHAR(40) NOT NULL DEFAULT 'company_vat_general',
    title VARCHAR(255) NOT NULL,
    tax_id VARCHAR(100) NOT NULL DEFAULT '',
    item_name VARCHAR(100) NOT NULL DEFAULT '',
    receiver_email VARCHAR(255) NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invoice_templates_invoice_type_check CHECK (invoice_type IN ('company_vat_general', 'company_vat_special', 'personal'))
);

CREATE INDEX IF NOT EXISTS idx_invoice_templates_user_id ON invoice_templates(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_templates_user_name ON invoice_templates(user_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_templates_user_default ON invoice_templates(user_id) WHERE is_default;
