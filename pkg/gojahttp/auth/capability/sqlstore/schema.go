package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS auth_capabilities (
    id TEXT PRIMARY KEY,
    purpose TEXT NOT NULL,
    subject_id TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    claims_json TEXT NOT NULL DEFAULT '{}',
    token_hash BLOB NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    single_use BOOLEAN NOT NULL DEFAULT 0,
    used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_capabilities_purpose ON auth_capabilities(purpose);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_subject_id ON auth_capabilities(subject_id);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_resource ON auth_capabilities(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_expires_at ON auth_capabilities(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_revoked_at ON auth_capabilities(revoked_at);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS auth_capabilities (
    id TEXT PRIMARY KEY,
    purpose TEXT NOT NULL,
    subject_id TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    claims_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    single_use BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_capabilities_purpose ON auth_capabilities(purpose);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_subject_id ON auth_capabilities(subject_id);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_resource ON auth_capabilities(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_expires_at ON auth_capabilities(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_capabilities_revoked_at ON auth_capabilities(revoked_at);
`
