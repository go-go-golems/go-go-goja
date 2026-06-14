// Package sqlstore provides a database/sql-backed audit.Store.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
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

// Store persists normalized audit records in SQL.
type Store struct {
	db      *sql.DB
	dialect Dialect
}

// New creates a SQL-backed audit store.
func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("audit/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("audit/sqlstore: unsupported dialect %q", cfg.Dialect)
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
			return fmt.Errorf("apply audit schema: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertAuditRecord(ctx context.Context, record audit.Record) error {
	attrs, err := marshalAttributes(record.Attributes)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.insertQuery(),
		record.Event,
		record.Outcome,
		nullString(record.Reason),
		nullInt(record.StatusCode),
		nullString(record.RouteName),
		record.Method,
		record.Pattern,
		nullString(record.Action),
		nullString(record.ActorID),
		nullString(record.ActorKind),
		nullString(record.TenantID),
		nullString(record.ResourceType),
		nullString(record.ResourceID),
		nullString(record.RequestID),
		nullString(record.IPHash),
		nullString(record.UserAgent),
		string(attrs),
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit record: %w", err)
	}
	return nil
}

// Snapshot returns all stored records in insertion order. It is primarily for
// tests and examples; production callers should query their database directly
// or add app-specific query APIs.
func (s *Store) Snapshot(ctx context.Context) ([]audit.Record, error) {
	rows, err := s.db.QueryContext(ctx, snapshotQuery)
	if err != nil {
		return nil, fmt.Errorf("query audit records: %w", err)
	}
	defer closeRows(rows)
	out := []audit.Record{}
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit records: %w", err)
	}
	return out, nil
}

// QueryByOutcome returns records for operational checks such as denied or
// failed requests. It doubles as executable documentation for common audit
// queries used in deployment runbooks.
func (s *Store) QueryByOutcome(ctx context.Context, outcome string, limit int) ([]audit.Record, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, s.queryByOutcomeQuery(), outcome, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit records by outcome: %w", err)
	}
	defer closeRows(rows)
	out := []audit.Record{}
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit records by outcome: %w", err)
	}
	return out, nil
}

func scanRecord(rows *sql.Rows) (audit.Record, error) {
	var record audit.Record
	var reason sql.NullString
	var statusCode sql.NullInt64
	var routeName sql.NullString
	var action sql.NullString
	var actorID sql.NullString
	var actorKind sql.NullString
	var tenantID sql.NullString
	var resourceType sql.NullString
	var resourceID sql.NullString
	var requestID sql.NullString
	var ipHash sql.NullString
	var userAgent sql.NullString
	var attributesJSON string
	if err := rows.Scan(
		&record.Event,
		&record.Outcome,
		&reason,
		&statusCode,
		&routeName,
		&record.Method,
		&record.Pattern,
		&action,
		&actorID,
		&actorKind,
		&tenantID,
		&resourceType,
		&resourceID,
		&requestID,
		&ipHash,
		&userAgent,
		&attributesJSON,
		&record.CreatedAt,
	); err != nil {
		return audit.Record{}, fmt.Errorf("scan audit record: %w", err)
	}
	record.Reason = reason.String
	if statusCode.Valid {
		record.StatusCode = int(statusCode.Int64)
	}
	record.RouteName = routeName.String
	record.Action = action.String
	record.ActorID = actorID.String
	record.ActorKind = actorKind.String
	record.TenantID = tenantID.String
	record.ResourceType = resourceType.String
	record.ResourceID = resourceID.String
	record.RequestID = requestID.String
	record.IPHash = ipHash.String
	record.UserAgent = userAgent.String
	if err := json.Unmarshal([]byte(attributesJSON), &record.Attributes); err != nil {
		return audit.Record{}, fmt.Errorf("decode audit attributes: %w", err)
	}
	return record, nil
}

const auditColumns = `event, outcome, reason, status_code, route_name, method, pattern, action, actor_id, actor_kind, tenant_id, resource_type, resource_id, request_id, ip_hash, user_agent, attributes_json, created_at`

const (
	insertSQLite           = `INSERT INTO auth_audit_records (` + auditColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	insertPostgres         = `INSERT INTO auth_audit_records (` + auditColumns + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	snapshotQuery          = `SELECT ` + auditColumns + ` FROM auth_audit_records ORDER BY id ASC`
	queryByOutcomeSQLite   = `SELECT ` + auditColumns + ` FROM auth_audit_records WHERE outcome = ? ORDER BY created_at DESC, id DESC LIMIT ?`
	queryByOutcomePostgres = `SELECT ` + auditColumns + ` FROM auth_audit_records WHERE outcome = $1 ORDER BY created_at DESC, id DESC LIMIT $2`
)

func (s *Store) insertQuery() string {
	if s.dialect == DialectPostgres {
		return insertPostgres
	}
	return insertSQLite
}

func (s *Store) queryByOutcomeQuery() string {
	if s.dialect == DialectPostgres {
		return queryByOutcomePostgres
	}
	return queryByOutcomeSQLite
}

func marshalAttributes(attrs map[string]any) ([]byte, error) {
	if attrs == nil {
		attrs = map[string]any{}
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("marshal audit attributes: %w", err)
	}
	return data, nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullInt(value int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(value), Valid: value != 0}
}

func closeRows(rows *sql.Rows) { _ = rows.Close() }

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

var _ audit.Store = (*Store)(nil)
