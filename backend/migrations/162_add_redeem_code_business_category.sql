-- Migration: 162_add_redeem_code_business_category
-- Classifies balance-affecting redeem_codes by business nature.
-- Empty string means historical/unclassified; unclassified admin_balance records are
-- intentionally excluded from invoiceable recharge and net recharge until classified.

ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS business_category VARCHAR(40) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_redeem_codes_type_business_category
    ON redeem_codes (type, business_category);
