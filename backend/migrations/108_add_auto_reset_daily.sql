-- Add auto_reset_daily column to user_subscriptions.
-- When true and daily quota is exhausted, the gateway will automatically
-- trigger ResetSubscriptionWithCost to refresh the daily window in place
-- of returning a quota-exceeded error.

ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS auto_reset_daily BOOLEAN NOT NULL DEFAULT FALSE;
