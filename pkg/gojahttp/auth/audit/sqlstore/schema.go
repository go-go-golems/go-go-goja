package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS auth_audit_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event TEXT NOT NULL,
    outcome TEXT NOT NULL,
    reason TEXT,
    status_code INTEGER,
    route_name TEXT,
    method TEXT NOT NULL DEFAULT '',
    pattern TEXT NOT NULL DEFAULT '',
    action TEXT,
    actor_id TEXT,
    actor_kind TEXT,
    tenant_id TEXT,
    resource_type TEXT,
    resource_id TEXT,
    request_id TEXT,
    ip_hash TEXT,
    user_agent TEXT,
    attributes_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_audit_records_created_at ON auth_audit_records(created_at);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_outcome ON auth_audit_records(outcome);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_event ON auth_audit_records(event);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_actor_id ON auth_audit_records(actor_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_resource ON auth_audit_records(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_tenant_id ON auth_audit_records(tenant_id);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS auth_audit_records (
    id BIGSERIAL PRIMARY KEY,
    event TEXT NOT NULL,
    outcome TEXT NOT NULL,
    reason TEXT,
    status_code INTEGER,
    route_name TEXT,
    method TEXT NOT NULL DEFAULT '',
    pattern TEXT NOT NULL DEFAULT '',
    action TEXT,
    actor_id TEXT,
    actor_kind TEXT,
    tenant_id TEXT,
    resource_type TEXT,
    resource_id TEXT,
    request_id TEXT,
    ip_hash TEXT,
    user_agent TEXT,
    attributes_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_audit_records_created_at ON auth_audit_records(created_at);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_outcome ON auth_audit_records(outcome);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_event ON auth_audit_records(event);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_actor_id ON auth_audit_records(actor_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_resource ON auth_audit_records(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_records_tenant_id ON auth_audit_records(tenant_id);
`
