package repldb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

// Store owns the SQLite database used for durable REPL session persistence.
type Store struct {
	db *sql.DB

	// Test hooks are package-private and nil in production.
	beforeEvaluationCommit func() error
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
	if err := retrySQLiteBusy(ctx, func() error {
		return applyMigrations(ctx, s.db, schemaMigrations(), CurrentSchemaVersion)
	}); err != nil {
		return errors.Wrap(err, "bootstrap repl db: apply migrations")
	}
	if err := retrySQLiteBusy(ctx, func() error {
		var mode string
		if err := s.db.QueryRowContext(ctx, `PRAGMA journal_mode = WAL;`).Scan(&mode); err != nil {
			return err
		}
		if strings.ToLower(mode) != "wal" {
			return fmt.Errorf("journal mode is %q, want wal", mode)
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "bootstrap repl db: configure WAL")
	}
	return nil
}

func retrySQLiteBusy(ctx context.Context, operation func() error) error {
	const maxWait = 5 * time.Second
	deadline := time.Now().Add(maxWait)
	delay := 5 * time.Millisecond
	for {
		err := operation()
		if err == nil || !isSQLiteBusyError(err) {
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < 100*time.Millisecond {
			delay *= 2
		}
	}
}

func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "database schema is locked")
}

func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s_foreign_keys=on&_busy_timeout=5000&_txlock=immediate", path, separator)
}
