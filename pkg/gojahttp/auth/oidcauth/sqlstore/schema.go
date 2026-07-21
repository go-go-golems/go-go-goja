package sqlstore

const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS oidc_login_transactions (
    state TEXT PRIMARY KEY,
    nonce TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    redirect_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oidc_login_transactions_expires_at
    ON oidc_login_transactions(expires_at);
`

const PostgresSchema = `
CREATE TABLE IF NOT EXISTS oidc_login_transactions (
    state TEXT PRIMARY KEY,
    nonce TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    redirect_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oidc_login_transactions_expires_at
    ON oidc_login_transactions(expires_at);
`
