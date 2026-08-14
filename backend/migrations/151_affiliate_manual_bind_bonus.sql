-- 邀请绑定礼包改为用户手动领取。
-- 历史已绑定用户可能已经通过旧逻辑自动获得过绑定奖励，统一标记为已处理，避免上线后重复领取。

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS bind_bonus_claimed_at TIMESTAMPTZ NULL;

UPDATE user_affiliates
SET bind_bonus_claimed_at = COALESCE(bind_bonus_claimed_at, updated_at, NOW()),
    updated_at = NOW()
WHERE inviter_id IS NOT NULL
  AND bind_bonus_claimed_at IS NULL;

COMMENT ON COLUMN user_affiliates.bind_bonus_claimed_at IS '邀请绑定礼包领取时间；NULL 表示未领取或不适用';
