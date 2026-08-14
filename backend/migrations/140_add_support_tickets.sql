-- 支持工单系统

CREATE TABLE IF NOT EXISTS tickets (
    id                    BIGSERIAL PRIMARY KEY,
    ticket_no             VARCHAR(32) NOT NULL UNIQUE,
    user_id               BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email            VARCHAR(255) NOT NULL DEFAULT '',
    user_name             VARCHAR(100) NOT NULL DEFAULT '',
    subject               VARCHAR(200) NOT NULL,
    category              VARCHAR(50) NOT NULL DEFAULT 'general',
    priority              VARCHAR(20) NOT NULL DEFAULT 'normal',
    status                VARCHAR(20) NOT NULL DEFAULT 'open',
    source                VARCHAR(30) NOT NULL DEFAULT 'user',
    template_key          VARCHAR(80) NOT NULL DEFAULT '',
    context_type          VARCHAR(50) NOT NULL DEFAULT '',
    context_id            VARCHAR(128) NOT NULL DEFAULT '',
    context_data          JSONB NOT NULL DEFAULT '{}'::jsonb,
    assignee_id           BIGINT REFERENCES users(id) ON DELETE SET NULL,
    escalated_at          TIMESTAMPTZ,
    escalated_by          BIGINT REFERENCES users(id) ON DELETE SET NULL,
    escalation_reason     VARCHAR(500) NOT NULL DEFAULT '',
    last_message_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_user_message_at  TIMESTAMPTZ,
    last_admin_message_at TIMESTAMPTZ,
    resolved_at           TIMESTAMPTZ,
    closed_at             TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE IF EXISTS tickets
    ADD COLUMN IF NOT EXISTS template_key VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS context_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS escalated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS escalation_reason VARCHAR(500) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS ticket_messages (
    id          BIGSERIAL PRIMARY KEY,
    ticket_id   BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL DEFAULT 'user',
    sender_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    sender_name VARCHAR(100) NOT NULL DEFAULT '',
    visibility  VARCHAR(20) NOT NULL DEFAULT 'public',
    body        TEXT NOT NULL,
    attachments JSONB NOT NULL DEFAULT '[]'::jsonb,
    edited_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE IF EXISTS ticket_messages
    ADD COLUMN IF NOT EXISTS attachments JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS ticket_reads (
    id                   BIGSERIAL PRIMARY KEY,
    ticket_id            BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    actor_type           VARCHAR(20) NOT NULL DEFAULT 'user',
    actor_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id BIGINT REFERENCES ticket_messages(id) ON DELETE SET NULL,
    read_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ticket_reads_ticket_actor_unique UNIQUE (ticket_id, actor_type, actor_id)
);

CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee_id ON tickets(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tickets_template_key ON tickets(template_key);
CREATE INDEX IF NOT EXISTS idx_tickets_escalated_at ON tickets(escalated_at);
CREATE INDEX IF NOT EXISTS idx_tickets_last_message_at ON tickets(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_status_last_message_at ON tickets(status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_user_status ON tickets(user_id, status);

CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_created_at ON ticket_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_sender_type ON ticket_messages(sender_type);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_visibility ON ticket_messages(visibility);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket_created ON ticket_messages(ticket_id, created_at);

CREATE INDEX IF NOT EXISTS idx_ticket_reads_ticket_id ON ticket_reads(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_reads_actor ON ticket_reads(actor_type, actor_id);
