CREATE TABLE IF NOT EXISTS upstreams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    base_url TEXT NOT NULL,
    kind VARCHAR(20) NOT NULL DEFAULT 'auto',
    credentials JSONB NOT NULL DEFAULT '{}'::jsonb,
    proxy_id BIGINT NULL REFERENCES proxies(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    last_probe_at TIMESTAMPTZ NULL,
    last_probe_error TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS upstream_id BIGINT NULL;

DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'upstreams_kind_check') THEN
		ALTER TABLE upstreams
			ADD CONSTRAINT upstreams_kind_check
			CHECK (kind IN ('auto', 'newapi', 'sub2api'));
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'upstreams_status_check') THEN
		ALTER TABLE upstreams
			ADD CONSTRAINT upstreams_status_check
			CHECK (status IN ('unknown', 'healthy', 'degraded', 'error'));
	END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'accounts_upstream_id_fkey') THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_upstream_id_fkey
            FOREIGN KEY (upstream_id) REFERENCES upstreams(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_upstreams_name ON upstreams(name);
CREATE INDEX IF NOT EXISTS idx_upstreams_kind ON upstreams(kind);
CREATE INDEX IF NOT EXISTS idx_upstreams_base_url ON upstreams(base_url);
CREATE INDEX IF NOT EXISTS idx_upstreams_proxy_id ON upstreams(proxy_id);
CREATE INDEX IF NOT EXISTS idx_upstreams_status ON upstreams(status);
CREATE INDEX IF NOT EXISTS idx_upstreams_deleted_at ON upstreams(deleted_at);
CREATE INDEX IF NOT EXISTS idx_accounts_upstream_id ON accounts(upstream_id);

COMMENT ON TABLE upstreams IS 'Upstream management sites; runtime accounts bind through accounts.upstream_id';
COMMENT ON COLUMN upstreams.credentials IS 'Sensitive upstream management and default API credentials';
COMMENT ON COLUMN accounts.upstream_id IS 'Management binding only; platform scheduling remains isolated';
