// Package sqlstore provides a database/sql-backed OIDC login transaction store.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
)

// Dialect selects SQL placeholder and schema syntax.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// Config controls Store construction. TTL controls how long a transaction can
// be consumed after its recorded creation time. A zero TTL defaults to ten
// minutes, matching the in-memory transaction store.
type Config struct {
	DB      *sql.DB
	Dialect Dialect
	TTL     time.Duration
	Now     func() time.Time
}

// Store persists short-lived, single-use OIDC login transactions.
type Store struct {
	db      *sql.DB
	dialect Dialect
	ttl     time.Duration
	now     func() time.Time
}

var _ keycloakauth.TransactionStore = (*Store)(nil)

// New constructs a transaction store.
func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("keycloakauth/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("keycloakauth/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	if cfg.TTL == 0 {
		cfg.TTL = 10 * time.Minute
	}
	if cfg.TTL <= 0 {
		return nil, fmt.Errorf("keycloakauth/sqlstore: ttl must be positive")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect, ttl: cfg.TTL, now: cfg.Now}, nil
}

// Schema returns the DDL for the configured dialect.
func (s *Store) Schema() string {
	if s.dialect == DialectSQLite {
		return SQLiteSchema
	}
	return PostgresSchema
}

// ApplySchema executes the configured schema. Production deployments should
// run the same DDL through their migration system before host startup.
func (s *Store) ApplySchema(ctx context.Context) error {
	for _, stmt := range splitSQLStatements(s.Schema()) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply oidc transaction schema: %w", err)
		}
	}
	return nil
}

// Put stores a transaction until its creation time plus the configured TTL.
func (s *Store) Put(ctx context.Context, tx keycloakauth.Transaction) error {
	if tx.State == "" || tx.Nonce == "" || tx.PKCEVerifier == "" {
		return fmt.Errorf("oidc transaction state, nonce, and PKCE verifier are required")
	}
	if tx.CreatedAt.IsZero() {
		return fmt.Errorf("oidc transaction created at is required")
	}
	if tx.RedirectURL == "" {
		return fmt.Errorf("oidc transaction redirect URL is required")
	}
	if _, err := s.db.ExecContext(ctx, s.insertQuery(), tx.State, tx.Nonce, tx.PKCEVerifier, tx.RedirectURL, tx.CreatedAt, tx.CreatedAt.Add(s.ttl)); err != nil {
		return fmt.Errorf("store oidc transaction: %w", err)
	}
	return nil
}

// Take atomically deletes and returns one unexpired transaction. The delete is
// the claim operation: concurrent callbacks cannot both redeem the same state.
func (s *Store) Take(ctx context.Context, state string) (keycloakauth.Transaction, error) {
	if state == "" {
		return keycloakauth.Transaction{}, fmt.Errorf("%w", keycloakauth.ErrTransactionUnavailable)
	}
	row := s.db.QueryRowContext(ctx, s.takeQuery(), state, s.now())
	var tx keycloakauth.Transaction
	if err := row.Scan(&tx.Nonce, &tx.PKCEVerifier, &tx.RedirectURL, &tx.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return keycloakauth.Transaction{}, fmt.Errorf("%w", keycloakauth.ErrTransactionUnavailable)
		}
		return keycloakauth.Transaction{}, fmt.Errorf("take oidc transaction: %w", err)
	}
	tx.State = state
	return tx, nil
}

// Cleanup removes expired rows. Correctness never depends on Cleanup because
// Take also filters by expiry; this method only bounds table growth.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, s.cleanupQuery(), s.now())
	if err != nil {
		return 0, fmt.Errorf("cleanup oidc transactions: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cleaned oidc transactions: %w", err)
	}
	return count, nil
}

const (
	insertSQLite    = `INSERT INTO oidc_login_transactions (state, nonce, pkce_verifier, redirect_url, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`
	insertPostgres  = `INSERT INTO oidc_login_transactions (state, nonce, pkce_verifier, redirect_url, created_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`
	takeSQLite      = `DELETE FROM oidc_login_transactions WHERE state = ? AND expires_at > ? RETURNING nonce, pkce_verifier, redirect_url, created_at`
	takePostgres    = `DELETE FROM oidc_login_transactions WHERE state = $1 AND expires_at > $2 RETURNING nonce, pkce_verifier, redirect_url, created_at`
	cleanupSQLite   = `DELETE FROM oidc_login_transactions WHERE expires_at <= ?`
	cleanupPostgres = `DELETE FROM oidc_login_transactions WHERE expires_at <= $1`
)

func (s *Store) insertQuery() string {
	if s.dialect == DialectPostgres {
		return insertPostgres
	}
	return insertSQLite
}

func (s *Store) takeQuery() string {
	if s.dialect == DialectPostgres {
		return takePostgres
	}
	return takeSQLite
}

func (s *Store) cleanupQuery() string {
	if s.dialect == DialectPostgres {
		return cleanupPostgres
	}
	return cleanupSQLite
}

func splitSQLStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if stmt := strings.TrimSpace(part); stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
