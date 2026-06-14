package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS auth_app_users (
    id TEXT PRIMARY KEY,
    keycloak_sub TEXT UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    email_verified BOOLEAN NOT NULL DEFAULT 0,
    disabled_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS auth_app_tenants (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    disabled_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS auth_app_memberships (
    user_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    role TEXT NOT NULL,
    revoked_at TIMESTAMP NULL,
    PRIMARY KEY (user_id, tenant_id, role)
);

CREATE TABLE IF NOT EXISTS auth_app_resources (
    type TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    owner_id TEXT NOT NULL DEFAULT '',
    claims_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (type, id)
);

CREATE INDEX IF NOT EXISTS idx_auth_app_users_keycloak_sub ON auth_app_users(keycloak_sub);
CREATE INDEX IF NOT EXISTS idx_auth_app_users_disabled_at ON auth_app_users(disabled_at);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_user ON auth_app_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_tenant ON auth_app_memberships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_revoked_at ON auth_app_memberships(revoked_at);
CREATE INDEX IF NOT EXISTS idx_auth_app_resources_tenant ON auth_app_resources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_resources_owner ON auth_app_resources(owner_id);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS auth_app_users (
    id TEXT PRIMARY KEY,
    keycloak_sub TEXT UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    email_verified BOOLEAN NOT NULL DEFAULT false,
    disabled_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS auth_app_tenants (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    disabled_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS auth_app_memberships (
    user_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    role TEXT NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    PRIMARY KEY (user_id, tenant_id, role)
);

CREATE TABLE IF NOT EXISTS auth_app_resources (
    type TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    owner_id TEXT NOT NULL DEFAULT '',
    claims_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (type, id)
);

CREATE INDEX IF NOT EXISTS idx_auth_app_users_keycloak_sub ON auth_app_users(keycloak_sub);
CREATE INDEX IF NOT EXISTS idx_auth_app_users_disabled_at ON auth_app_users(disabled_at);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_user ON auth_app_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_tenant ON auth_app_memberships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_memberships_revoked_at ON auth_app_memberships(revoked_at);
CREATE INDEX IF NOT EXISTS idx_auth_app_resources_tenant ON auth_app_resources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_app_resources_owner ON auth_app_resources(owner_id);
`
