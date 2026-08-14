-- 用户专属分组折扣
-- discount_multiplier 保存相对于分组默认倍率的折扣系数：
--   - discount_multiplier = 0.8 表示按当前分组默认倍率的 8 折计费
--   - rate_multiplier 非 NULL 时仍作为固定专属倍率优先生效
--   - rate_multiplier NULL 且 discount_multiplier 非 NULL 时，实际倍率 = groups.rate_multiplier * discount_multiplier
ALTER TABLE user_group_rate_multipliers
    ADD COLUMN IF NOT EXISTS discount_multiplier DECIMAL(10,4);

COMMENT ON COLUMN user_group_rate_multipliers.discount_multiplier IS '专属折扣系数；NULL 表示不使用折扣，0.8 表示当前分组倍率的8折。rate_multiplier 非 NULL 时固定倍率优先生效。';
