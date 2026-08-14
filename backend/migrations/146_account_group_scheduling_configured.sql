ALTER TABLE account_groups
    ADD COLUMN IF NOT EXISTS scheduling_configured BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE account_groups
SET weight = 1
WHERE weight <= 0;

UPDATE account_groups
SET scheduling_configured = TRUE
WHERE role <> 'primary'
   OR weight <> 100
   OR sort_order <> priority;

ALTER TABLE account_groups
    DROP CONSTRAINT IF EXISTS chk_account_groups_weight,
    ADD CONSTRAINT chk_account_groups_weight CHECK (weight > 0);

COMMENT ON COLUMN account_groups.scheduling_configured IS 'Whether role/weight/sort_order were explicitly configured by an admin';
