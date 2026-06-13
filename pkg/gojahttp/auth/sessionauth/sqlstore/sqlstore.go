// Package sqlstore provides a database/sql-backed sessionauth.Store.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// Dialect selects SQL placeholder and schema syntax.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// Config controls Store construction.
type Config struct {
	DB      *sql.DB
	Dialect Dialect
	Now     func() time.Time
}

// Store persists opaque app sessions in SQL.
type Store struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

// New creates a SQL-backed session store.
func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("sessionauth/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("sessionauth/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect, now: cfg.Now}, nil
}

// Schema returns the DDL for the configured dialect.
func (s *Store) Schema() string {
	if s.dialect == DialectSQLite {
		return SQLiteSchema
	}
	return PostgresSchema
}

// ApplySchema executes the configured schema. It is intended for tests,
// examples, and simple migrations; production hosts can run the same DDL with
// their migration tool of choice.
func (s *Store) ApplySchema(ctx context.Context) error {
	for _, stmt := range splitSQLStatements(s.Schema()) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply sessionauth schema: %w", err)
		}
	}
	return nil
}

func (s *Store) Create(ctx context.Context, session sessionauth.Session) error {
	if session.ID == "" {
		return fmt.Errorf("session id is required")
	}
	return s.insert(ctx, s.db, session)
}

func (s *Store) Get(ctx context.Context, id string) (*sessionauth.Session, error) {
	return s.scan(ctx, s.db, id)
}

func (s *Store) Touch(ctx context.Context, id string, now time.Time, idleExpiresAt time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE auth_sessions SET last_seen_at = `+s.bind(1)+`, idle_expires_at = `+s.bind(2)+` WHERE id = `+s.bind(3),
		now, idleExpiresAt, id,
	)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return requireAffected(res, sessionauth.ErrInvalidCookie)
}

func (s *Store) Rotate(ctx context.Context, oldID string, next sessionauth.Session) error {
	if next.ID == "" {
		return fmt.Errorf("next session id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rotate session: %w", err)
	}
	defer rollback(tx)
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE id = `+s.bind(1), oldID); err != nil {
		return fmt.Errorf("delete old session: %w", err)
	}
	if err := s.insert(ctx, tx, next); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rotate session: %w", err)
	}
	return nil
}

func (s *Store) Revoke(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = `+s.bind(1)+` WHERE id = `+s.bind(2), s.now(), id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Store) insert(ctx context.Context, exec sqlExecer, session sessionauth.Session) error {
	tenantIDs, err := marshalJSON(session.TenantIDs)
	if err != nil {
		return err
	}
	claims, err := marshalJSON(session.Claims)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `INSERT INTO auth_sessions (`+
		`id, user_id, keycloak_sub, email, email_verified, tenant_ids_json, csrf_token, mfa_at, created_at, last_seen_at, idle_expires_at, absolute_expires_at, revoked_at, claims_json`+
		`) VALUES (`+s.bindList(14)+`)`,
		session.ID,
		session.UserID,
		nullString(session.KeycloakSub),
		nullString(session.Email),
		session.EmailVerified,
		string(tenantIDs),
		session.CSRFToken,
		nullTime(session.MFAAt),
		session.CreatedAt,
		session.LastSeenAt,
		session.IdleExpiresAt,
		session.AbsoluteExpiresAt,
		nullTime(session.RevokedAt),
		string(claims),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) scan(ctx context.Context, queryer sqlQueryer, id string) (*sessionauth.Session, error) {
	row := queryer.QueryRowContext(ctx, `SELECT `+
		`id, user_id, keycloak_sub, email, email_verified, tenant_ids_json, csrf_token, mfa_at, created_at, last_seen_at, idle_expires_at, absolute_expires_at, revoked_at, claims_json `+
		`FROM auth_sessions WHERE id = `+s.bind(1), id)
	var session sessionauth.Session
	var keycloakSub sql.NullString
	var email sql.NullString
	var tenantIDsJSON string
	var claimsJSON string
	var mfaAt sql.NullTime
	var revokedAt sql.NullTime
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&keycloakSub,
		&email,
		&session.EmailVerified,
		&tenantIDsJSON,
		&session.CSRFToken,
		&mfaAt,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.IdleExpiresAt,
		&session.AbsoluteExpiresAt,
		&revokedAt,
		&claimsJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sessionauth.ErrInvalidCookie
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	session.KeycloakSub = keycloakSub.String
	session.Email = email.String
	if mfaAt.Valid {
		session.MFAAt = &mfaAt.Time
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	if err := json.Unmarshal([]byte(tenantIDsJSON), &session.TenantIDs); err != nil {
		return nil, fmt.Errorf("decode tenant ids: %w", err)
	}
	if err := json.Unmarshal([]byte(claimsJSON), &session.Claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	return &session, nil
}

func (s *Store) bindList(count int) string {
	parts := make([]string, count)
	for i := range count {
		parts[i] = s.bind(i + 1)
	}
	return strings.Join(parts, ", ")
}

func (s *Store) bind(n int) string {
	if s.dialect == DialectPostgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type sqlQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func marshalJSON(value any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal session json: %w", err)
	}
	return data, nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func requireAffected(res sql.Result, missing error) error {
	count, err := res.RowsAffected()
	if err != nil {
		return nil
	}
	if count == 0 {
		return missing
	}
	return nil
}

func rollback(tx *sql.Tx) { _ = tx.Rollback() }

func splitSQLStatements(schema string) []string {
	pieces := strings.Split(schema, ";")
	out := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		stmt := strings.TrimSpace(piece)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
