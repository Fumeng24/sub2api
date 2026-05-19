-- 工单 SLA 截止与催办状态

ALTER TABLE IF EXISTS tickets
    ADD COLUMN IF NOT EXISTS sla_due_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sla_reminded_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_tickets_sla_due_at ON tickets(sla_due_at);
CREATE INDEX IF NOT EXISTS idx_tickets_sla_reminded_at ON tickets(sla_reminded_at);
