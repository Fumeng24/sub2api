CREATE TABLE IF NOT EXISTS invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    user_name VARCHAR(100) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    invoice_type VARCHAR(40) NOT NULL DEFAULT 'company_vat_general',
    title VARCHAR(255) NOT NULL,
    tax_id VARCHAR(100) NOT NULL DEFAULT '',
    item_name VARCHAR(100) NOT NULL DEFAULT '',
    amount DECIMAL(20, 2) NOT NULL,
    tax_rate DECIMAL(8, 4) NOT NULL DEFAULT 0.02,
    tax_fee DECIMAL(20, 2) NOT NULL DEFAULT 0,
    receiver_email VARCHAR(255) NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    admin_note TEXT NOT NULL DEFAULT '',
    invoice_no VARCHAR(128) NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    processed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invoice_requests_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'completed', 'cancelled')),
    CONSTRAINT invoice_requests_amount_positive_check CHECK (amount > 0),
    CONSTRAINT invoice_requests_tax_rate_nonnegative_check CHECK (tax_rate >= 0),
    CONSTRAINT invoice_requests_tax_fee_nonnegative_check CHECK (tax_fee >= 0)
);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_id ON invoice_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_status ON invoice_requests(status);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_created_at ON invoice_requests(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_status ON invoice_requests(user_id, status);
