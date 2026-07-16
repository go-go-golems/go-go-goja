package repldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// CurrentSchemaVersion is the newest durable schema understood by this binary.
const CurrentSchemaVersion = 2

const currentSchemaVersion = "2"

var (
	// ErrDatabaseTooNew prevents an older binary from relabeling a newer database.
	ErrDatabaseTooNew = errors.New("repldb: database schema is newer than this binary")
	// ErrInvalidSchemaVersion identifies missing, malformed, or unsupported migration metadata.
	ErrInvalidSchemaVersion = errors.New("repldb: invalid schema version")
)

// SchemaVersionError reports the stored and supported versions.
type SchemaVersionError struct {
	Found     int
	Supported int
	Cause     error
}

func (e *SchemaVersionError) Error() string {
	if e == nil {
		return ErrInvalidSchemaVersion.Error()
	}
	return fmt.Sprintf("repldb: schema version %d (supported through %d): %v", e.Found, e.Supported, e.Cause)
}

func (e *SchemaVersionError) Unwrap() error {
	if e != nil && e.Cause != nil {
		return e.Cause
	}
	return ErrInvalidSchemaVersion
}

type migration struct {
	Version    int
	Name       string
	Statements []string
}

func schemaMigrations() []migration {
	return []migration{
		{
			Version:    1,
			Name:       "initial repl session journal",
			Statements: schemaStatements(),
		},
		{
			Version: 2,
			Name:    "per-session ownership leases",
			Statements: []string{
				`CREATE TABLE session_leases (
					session_id TEXT PRIMARY KEY,
					owner_id TEXT NOT NULL,
					epoch INTEGER NOT NULL CHECK(epoch > 0),
					lease_until TEXT NOT NULL,
					updated_at TEXT NOT NULL,
					FOREIGN KEY(session_id) REFERENCES sessions(session_id)
				);`,
				`CREATE INDEX idx_session_leases_until ON session_leases(lease_until);`,
			},
		},
	}
}

func applyMigrations(ctx context.Context, db *sql.DB, migrations []migration, target int) error {
	if db == nil {
		return fmt.Errorf("migrate repl db: database is nil")
	}
	if err := validateMigrations(migrations, target); err != nil {
		return err
	}
	for {
		version, err := readSchemaVersion(ctx, db)
		if err != nil {
			return err
		}
		if version > target {
			return &SchemaVersionError{Found: version, Supported: target, Cause: ErrDatabaseTooNew}
		}
		if version == target {
			return nil
		}
		next, ok := migrationForVersion(migrations, version+1)
		if !ok {
			return &SchemaVersionError{Found: version, Supported: target, Cause: ErrInvalidSchemaVersion}
		}
		if err := applyMigration(ctx, db, next, target); err != nil {
			return err
		}
	}
}

func applyMigration(ctx context.Context, db *sql.DB, next migration, target int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate repl db to v%d: begin transaction: %w", next.Version, err)
	}
	defer func() { _ = tx.Rollback() }()

	// Re-read under the immediate transaction. Another opener may have completed
	// this migration while this connection waited for the SQLite write lock.
	version, err := readSchemaVersionQuery(ctx, tx)
	if err != nil {
		return err
	}
	if version > target {
		return &SchemaVersionError{Found: version, Supported: target, Cause: ErrDatabaseTooNew}
	}
	if version >= next.Version {
		return tx.Commit()
	}
	if version != next.Version-1 {
		return &SchemaVersionError{Found: version, Supported: target, Cause: ErrInvalidSchemaVersion}
	}

	for index, statement := range next.Statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate repl db to v%d (%s), statement %d: %w", next.Version, next.Name, index+1, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO repldb_meta(key, value) VALUES('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, strconv.Itoa(next.Version)); err != nil {
		return fmt.Errorf("migrate repl db to v%d: record schema version: %w", next.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate repl db to v%d: commit transaction: %w", next.Version, err)
	}
	return nil
}

func validateMigrations(migrations []migration, target int) error {
	if target < 0 {
		return fmt.Errorf("%w: negative target %d", ErrInvalidSchemaVersion, target)
	}
	seen := map[int]struct{}{}
	lastVersion := 0
	for _, item := range migrations {
		if item.Version <= lastVersion {
			return fmt.Errorf("%w: migrations are not strictly ordered at v%d", ErrInvalidSchemaVersion, item.Version)
		}
		lastVersion = item.Version
		if item.Version <= 0 || item.Version > target {
			continue
		}
		if _, exists := seen[item.Version]; exists {
			return fmt.Errorf("%w: duplicate migration v%d", ErrInvalidSchemaVersion, item.Version)
		}
		seen[item.Version] = struct{}{}
	}
	for expected := 1; expected <= target; expected++ {
		if _, ok := seen[expected]; !ok {
			return fmt.Errorf("%w: missing migration v%d", ErrInvalidSchemaVersion, expected)
		}
	}
	return nil
}

func migrationForVersion(migrations []migration, version int) (migration, bool) {
	for _, item := range migrations {
		if item.Version == version {
			return item, true
		}
	}
	return migration{}, false
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	return readSchemaVersionQuery(ctx, db)
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readSchemaVersionQuery(ctx context.Context, queryer queryRower) (int, error) {
	var metaTable string
	err := queryer.QueryRowContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = 'repldb_meta'
	`).Scan(&metaTable)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read repl db schema version: inspect metadata table: %w", err)
	}

	var raw string
	err = queryer.QueryRowContext(ctx, `SELECT value FROM repldb_meta WHERE key = 'schema_version'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, &SchemaVersionError{Found: 0, Supported: CurrentSchemaVersion, Cause: ErrInvalidSchemaVersion}
	}
	if err != nil {
		return 0, fmt.Errorf("read repl db schema version: %w", err)
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version < 0 {
		return 0, &SchemaVersionError{Found: 0, Supported: CurrentSchemaVersion, Cause: ErrInvalidSchemaVersion}
	}
	return version, nil
}
