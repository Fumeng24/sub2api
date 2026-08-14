-- Migration: 159_add_account_monitors
-- 账号监控：与 channel-monitor 完全独立的一套。为单个 api_key 类上游账号配置探针，
-- 周期性用该账号自己的 credentials.api_key + base_url 打一次模型心跳。
--
-- 表结构说明：
--   - account_monitors        账号监控配置表（一行 = 一个被监控账号；account_id 唯一）
--   - account_monitor_checks  探测历史明细表（一次探测一行）
--
-- 设计要点：
--   - 纯管理员视角，无任何用户可达路由；不存储凭证（运行时按 account_id 实时读账号）。
--   - account_id 唯一：一个账号至多一个监控。
--   - 账号被删除时 account_monitors 行 ON DELETE CASCADE（监控离开账号无意义），
--     其历史再通过 account_monitor_checks 的 CASCADE 连带清除。
--   - (enabled, last_checked_at) 索引服务调度器扫描到期监控。
--   - checks 上 (account_monitor_id, checked_at DESC) 服务 admin 趋势聚合；
--     单独的 (checked_at) 索引服务定期清理超期明细的 DELETE。

CREATE TABLE IF NOT EXISTS account_monitors (
    id               BIGSERIAL PRIMARY KEY,
    account_id       BIGINT       NOT NULL,
    provider         VARCHAR(20)  NOT NULL DEFAULT 'openai',  -- openai / anthropic / gemini
    model            VARCHAR(200) NOT NULL DEFAULT 'gpt-5.4-mini',
    enabled          BOOLEAN      NOT NULL DEFAULT TRUE,
    interval_seconds INT          NOT NULL DEFAULT 60,
    jitter_seconds   INT          NOT NULL DEFAULT 0,
    last_checked_at  TIMESTAMPTZ,
    created_by       BIGINT       NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT account_monitors_provider_check CHECK (provider IN ('openai', 'anthropic', 'gemini')),
    CONSTRAINT account_monitors_interval_check CHECK (interval_seconds BETWEEN 15 AND 3600),
    CONSTRAINT account_monitors_jitter_check   CHECK (jitter_seconds BETWEEN 0 AND 3600),
    CONSTRAINT fk_account_monitors_account_id
        FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_account_monitors_account_id
    ON account_monitors (account_id);
CREATE INDEX IF NOT EXISTS idx_account_monitors_enabled_last_checked
    ON account_monitors (enabled, last_checked_at);

CREATE TABLE IF NOT EXISTS account_monitor_checks (
    id                 BIGSERIAL PRIMARY KEY,
    account_monitor_id BIGINT       NOT NULL,
    model              VARCHAR(200) NOT NULL,
    status             VARCHAR(20)  NOT NULL,    -- operational / degraded / failed / error
    latency_ms         INT,
    ping_latency_ms    INT,
    message            VARCHAR(500) NOT NULL DEFAULT '',
    checked_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT account_monitor_checks_status_check
        CHECK (status IN ('operational', 'degraded', 'failed', 'error')),
    CONSTRAINT fk_account_monitor_checks_monitor_id
        FOREIGN KEY (account_monitor_id) REFERENCES account_monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_account_monitor_checks_monitor_checked
    ON account_monitor_checks (account_monitor_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_account_monitor_checks_checked_at
    ON account_monitor_checks (checked_at);
