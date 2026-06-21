package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS auth_program_agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    owner_user_id TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    disabled_at TIMESTAMP NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    policy_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS auth_program_api_tokens (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    subject_user_id TEXT NOT NULL DEFAULT '',
    token_hash BLOB NOT NULL,
    token_prefix TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    grants_json TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_auth_program_agents_owner ON auth_program_agents(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_tenant ON auth_program_agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_disabled_at ON auth_program_agents(disabled_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_created_at ON auth_program_agents(created_at, id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_prefix ON auth_program_api_tokens(token_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_agent ON auth_program_api_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_subject ON auth_program_api_tokens(subject_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_revoked_at ON auth_program_api_tokens(revoked_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_created_at ON auth_program_api_tokens(created_at, id);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS auth_program_agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    owner_user_id TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    disabled_at TIMESTAMPTZ NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    policy_json JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE IF NOT EXISTS auth_program_api_tokens (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    subject_user_id TEXT NOT NULL DEFAULT '',
    token_hash BYTEA NOT NULL,
    token_prefix TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    last_used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    grants_json JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_auth_program_agents_owner ON auth_program_agents(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_tenant ON auth_program_agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_disabled_at ON auth_program_agents(disabled_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_created_at ON auth_program_agents(created_at, id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_prefix ON auth_program_api_tokens(token_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_agent ON auth_program_api_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_subject ON auth_program_api_tokens(subject_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_revoked_at ON auth_program_api_tokens(revoked_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_created_at ON auth_program_api_tokens(created_at, id);
`
