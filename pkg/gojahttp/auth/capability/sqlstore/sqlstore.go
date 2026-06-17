// Package sqlstore provides a database/sql-backed capability.Store.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
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
}

// Store persists bearer capability token hashes in SQL.
type Store struct {
	db      *sql.DB
	dialect Dialect
}

// New creates a SQL-backed capability store.
func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("capability/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("capability/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect}, nil
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
			return fmt.Errorf("apply capability schema: %w", err)
		}
	}
	return nil
}

func (s *Store) Create(ctx context.Context, record capability.Capability) error {
	if record.ID == "" || len(record.TokenHash) == 0 {
		return fmt.Errorf("capability id and token hash are required")
	}
	claims, err := marshalClaims(record.Claims)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.insertQuery(),
		record.ID,
		record.Purpose,
		record.SubjectID,
		record.ResourceType,
		record.ResourceID,
		string(claims),
		cloneBytes(record.TokenHash),
		record.ExpiresAt,
		record.SingleUse,
		nullTime(record.UsedAt),
		nullTime(record.RevokedAt),
		record.CreatedBy,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create capability: %w", err)
	}
	return nil
}

func (s *Store) Lookup(ctx context.Context, tokenHash []byte, purpose string, now time.Time) (*capability.Capability, error) {
	record, err := s.scanByHash(ctx, s.db, tokenHash)
	if err != nil {
		return nil, err
	}
	if err := validateCapability(record, purpose, now); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Store) Redeem(ctx context.Context, tokenHash []byte, purpose string, now time.Time) (*capability.Capability, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin redeem capability: %w", err)
	}
	defer rollback(tx)

	record, err := s.scanByHash(ctx, tx, tokenHash)
	if err != nil {
		return nil, err
	}
	if err := validateCapability(record, purpose, now); err != nil {
		return nil, err
	}
	if record.SingleUse {
		res, err := tx.ExecContext(ctx, s.markUsedQuery(), now, record.ID)
		if err != nil {
			return nil, fmt.Errorf("mark capability used: %w", err)
		}
		if err := requireAffected(res, capability.ErrUsed); err != nil {
			return nil, err
		}
		usedAt := now
		record.UsedAt = &usedAt
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit redeem capability: %w", err)
	}
	return record, nil
}

func validateCapability(record *capability.Capability, purpose string, now time.Time) error {
	if record.Purpose != purpose {
		return capability.ErrWrongPurpose
	}
	if record.RevokedAt != nil {
		return capability.ErrRevoked
	}
	if !record.ExpiresAt.IsZero() && now.After(record.ExpiresAt) {
		return capability.ErrExpired
	}
	if record.SingleUse && record.UsedAt != nil {
		return capability.ErrUsed
	}
	return nil
}

func (s *Store) Revoke(ctx context.Context, id string, now time.Time) error {
	res, err := s.db.ExecContext(ctx, s.revokeQuery(), now, id)
	if err != nil {
		return fmt.Errorf("revoke capability: %w", err)
	}
	return requireAffected(res, capability.ErrNotFound)
}

func (s *Store) ByID(ctx context.Context, id string) (*capability.Capability, error) {
	row := s.db.QueryRowContext(ctx, s.getByIDQuery(), id)
	return scanCapability(row)
}

func (s *Store) scanByHash(ctx context.Context, queryer sqlQueryer, tokenHash []byte) (*capability.Capability, error) {
	row := queryer.QueryRowContext(ctx, s.getByHashQuery(), cloneBytes(tokenHash))
	return scanCapability(row)
}

func scanCapability(row scanner) (*capability.Capability, error) {
	var record capability.Capability
	var claimsJSON string
	var usedAt sql.NullTime
	var revokedAt sql.NullTime
	if err := row.Scan(
		&record.ID,
		&record.Purpose,
		&record.SubjectID,
		&record.ResourceType,
		&record.ResourceID,
		&claimsJSON,
		&record.TokenHash,
		&record.ExpiresAt,
		&record.SingleUse,
		&usedAt,
		&revokedAt,
		&record.CreatedBy,
		&record.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, capability.ErrNotFound
		}
		return nil, fmt.Errorf("scan capability: %w", err)
	}
	if usedAt.Valid {
		record.UsedAt = &usedAt.Time
	}
	if revokedAt.Valid {
		record.RevokedAt = &revokedAt.Time
	}
	if err := json.Unmarshal([]byte(claimsJSON), &record.Claims); err != nil {
		return nil, fmt.Errorf("decode capability claims: %w", err)
	}
	record.TokenHash = cloneBytes(record.TokenHash)
	return &record, nil
}

const capabilityColumns = `id, purpose, subject_id, resource_type, resource_id, claims_json, token_hash, expires_at, single_use, used_at, revoked_at, created_by, created_at`

const (
	insertSQLite      = `INSERT INTO auth_capabilities (` + capabilityColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	insertPostgres    = `INSERT INTO auth_capabilities (` + capabilityColumns + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	getByIDSQLite     = `SELECT ` + capabilityColumns + ` FROM auth_capabilities WHERE id = ?`
	getByIDPostgres   = `SELECT ` + capabilityColumns + ` FROM auth_capabilities WHERE id = $1`
	getByHashSQLite   = `SELECT ` + capabilityColumns + ` FROM auth_capabilities WHERE token_hash = ?`
	getByHashPostgres = `SELECT ` + capabilityColumns + ` FROM auth_capabilities WHERE token_hash = $1`
	markUsedSQLite    = `UPDATE auth_capabilities SET used_at = ? WHERE id = ? AND used_at IS NULL`
	markUsedPostgres  = `UPDATE auth_capabilities SET used_at = $1 WHERE id = $2 AND used_at IS NULL`
	revokeSQLite      = `UPDATE auth_capabilities SET revoked_at = ? WHERE id = ?`
	revokePostgres    = `UPDATE auth_capabilities SET revoked_at = $1 WHERE id = $2`
)

func (s *Store) insertQuery() string {
	if s.dialect == DialectPostgres {
		return insertPostgres
	}
	return insertSQLite
}

func (s *Store) getByIDQuery() string {
	if s.dialect == DialectPostgres {
		return getByIDPostgres
	}
	return getByIDSQLite
}

func (s *Store) getByHashQuery() string {
	if s.dialect == DialectPostgres {
		return getByHashPostgres
	}
	return getByHashSQLite
}

func (s *Store) markUsedQuery() string {
	if s.dialect == DialectPostgres {
		return markUsedPostgres
	}
	return markUsedSQLite
}

func (s *Store) revokeQuery() string {
	if s.dialect == DialectPostgres {
		return revokePostgres
	}
	return revokeSQLite
}

type sqlQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type scanner interface {
	Scan(dest ...any) error
}

func marshalClaims(claims map[string]string) ([]byte, error) {
	if claims == nil {
		claims = map[string]string{}
	}
	data, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshal capability claims: %w", err)
	}
	return data, nil
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func cloneBytes(in []byte) []byte {
	if in == nil {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
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
