package repldb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenBootstrapsSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "repl.sqlite")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	for _, tableName := range []string{
		"repldb_meta",
		"sessions",
		"evaluations",
		"console_events",
		"bindings",
		"binding_versions",
		"binding_docs",
	} {
		if !tableExists(t, store.DB(), tableName) {
			t.Fatalf("expected table %q to exist", tableName)
		}
	}

	var version string
	err = store.DB().QueryRowContext(ctx, `SELECT value FROM repldb_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("expected schema version %q, got %q", currentSchemaVersion, version)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	store, err := Open(context.Background(), "")
	if err == nil {
		_ = store.Close()
		t.Fatal("expected error for empty path")
	}
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&name)
	switch err {
	case nil:
		return name == tableName
	case sql.ErrNoRows:
		return false
	default:
		t.Fatalf("query sqlite_master for %q: %v", tableName, err)
		return false
	}
}
