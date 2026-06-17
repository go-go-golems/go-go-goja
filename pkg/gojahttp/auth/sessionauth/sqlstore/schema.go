package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    keycloak_sub TEXT,
    email TEXT,
    email_verified BOOLEAN NOT NULL DEFAULT 0,
    tenant_ids_json TEXT NOT NULL DEFAULT '[]',
    csrf_token TEXT NOT NULL,
    mfa_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    last_seen_at TIMESTAMP NOT NULL,
    idle_expires_at TIMESTAMP NOT NULL,
    absolute_expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    claims_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_keycloak_sub ON auth_sessions(keycloak_sub);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_idle_expires_at ON auth_sessions(idle_expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_absolute_expires_at ON auth_sessions(absolute_expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_revoked_at ON auth_sessions(revoked_at);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    keycloak_sub TEXT,
    email TEXT,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    tenant_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    csrf_token TEXT NOT NULL,
    mfa_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    idle_expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    claims_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_keycloak_sub ON auth_sessions(keycloak_sub);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_idle_expires_at ON auth_sessions(idle_expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_absolute_expires_at ON auth_sessions(absolute_expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_revoked_at ON auth_sessions(revoked_at);
`
