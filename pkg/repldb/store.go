package repldb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const currentSchemaVersion = "1"

// Store owns the SQLite database used for durable REPL session persistence.
type Store struct {
	db *sql.DB
}

// Open opens or creates a SQLite database at path and ensures the schema exists.
func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("open repl db: path is empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, errors.Wrap(err, "open repl db: creating sqlite directory")
	}

	db, err := sql.Open("sqlite3", sqliteDSN(path))
	if err != nil {
		return nil, errors.Wrap(err, "open repl db: opening sqlite")
	}

	store := &Store{db: db}
	if err := store.bootstrap(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return errors.Wrap(s.db.Close(), "close repl db")
}

// DB exposes the underlying sql.DB for targeted callers and tests.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Store) bootstrap(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("bootstrap repl db: store is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "bootstrap repl db: begin tx")
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil {
		return errors.Wrap(err, "bootstrap repl db: enable foreign keys")
	}
	for _, stmt := range schemaStatements() {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return errors.Wrap(err, "bootstrap repl db: creating schema")
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO repldb_meta(key, value) VALUES('schema_version', ?)`, currentSchemaVersion); err != nil {
		return errors.Wrap(err, "bootstrap repl db: recording schema version")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "bootstrap repl db: commit tx")
	}

	return nil
}

func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL", path, separator)
}
