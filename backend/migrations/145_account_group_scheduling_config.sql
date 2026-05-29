ALTER TABLE account_groups
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'primary',
    ADD COLUMN IF NOT EXISTS weight INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS scheduling_configured BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE account_groups
SET sort_order = priority
WHERE sort_order IS NULL OR sort_order = 50;

ALTER TABLE account_groups
    DROP CONSTRAINT IF EXISTS chk_account_groups_role,
    ADD CONSTRAINT chk_account_groups_role CHECK (role IN ('primary', 'backup'));

ALTER TABLE account_groups
    DROP CONSTRAINT IF EXISTS chk_account_groups_weight,
    ADD CONSTRAINT chk_account_groups_weight CHECK (weight > 0);

CREATE INDEX IF NOT EXISTS idx_account_groups_group_role_sort
    ON account_groups (group_id, role, sort_order);

COMMENT ON COLUMN account_groups.role IS 'Per-group account scheduling role: primary or backup';
COMMENT ON COLUMN account_groups.weight IS 'Per-group configured scheduling weight';
COMMENT ON COLUMN account_groups.sort_order IS 'Per-group display and scheduling tie-break order';
COMMENT ON COLUMN account_groups.scheduling_configured IS 'Whether role/weight/sort_order were explicitly configured by an admin';
