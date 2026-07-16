package repldb

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestOpenRealV1FixturePreservesDataAndIsIdempotent(t *testing.T) {
	path := copyV1Fixture(t)
	for openNumber := 1; openNumber <= 2; openNumber++ {
		store, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("open v1 fixture %d: %v", openNumber, err)
		}
		exported, err := store.ExportSession(context.Background(), "fixture-session")
		if err != nil {
			_ = store.Close()
			t.Fatalf("export fixture after open %d: %v", openNumber, err)
		}
		if len(exported.Evaluations) != 1 || exported.Evaluations[0].CellID != 1 {
			_ = store.Close()
			t.Fatalf("fixture evaluations changed after open %d: %#v", openNumber, exported.Evaluations)
		}
		if len(exported.Evaluations[0].ConsoleEvents) != 1 || len(exported.Evaluations[0].BindingVersions) != 1 || len(exported.Evaluations[0].BindingDocs) != 1 {
			_ = store.Close()
			t.Fatalf("fixture child rows changed after open %d: %#v", openNumber, exported.Evaluations[0])
		}
		if !tableExists(t, store.DB(), "session_leases") {
			_ = store.Close()
			t.Fatalf("v1 fixture did not upgrade ownership schema on open %d", openNumber)
		}
		if got := schemaVersionForTest(t, store.DB()); got != CurrentSchemaVersion {
			_ = store.Close()
			t.Fatalf("schema version after open %d = %d, want %d", openNumber, got, CurrentSchemaVersion)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close fixture open %d: %v", openNumber, err)
		}
	}
}

func TestFailedMigrationRollsBackStatementsAndVersion(t *testing.T) {
	path := copyV1Fixture(t)
	db, err := sql.Open("sqlite3", sqliteDSN(path))
	if err != nil {
		t.Fatalf("open raw fixture: %v", err)
	}
	defer func() { _ = db.Close() }()

	migrations := append(schemaMigrations(), migration{
		Version: 3,
		Name:    "injected failing migration",
		Statements: []string{
			`CREATE TABLE phase4_rollback_probe(id INTEGER PRIMARY KEY);`,
			`INSERT INTO table_that_does_not_exist(value) VALUES(1);`,
		},
	})
	if err := applyMigrations(context.Background(), db, migrations, 3); err == nil {
		t.Fatal("expected injected migration failure")
	}
	if tableExists(t, db, "phase4_rollback_probe") {
		t.Fatal("failed migration left its created table behind")
	}
	if got := schemaVersionForTest(t, db); got != 2 {
		t.Fatalf("failed migration changed schema version to %d", got)
	}
}

func TestOpenRejectsDatabaseNewerThanBinary(t *testing.T) {
	path := copyV1Fixture(t)
	db, err := sql.Open("sqlite3", sqliteDSN(path))
	if err != nil {
		t.Fatalf("open raw fixture: %v", err)
	}
	if _, err := db.Exec(`UPDATE repldb_meta SET value = '999' WHERE key = 'schema_version'`); err != nil {
		_ = db.Close()
		t.Fatalf("set future version: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close future database: %v", err)
	}

	store, err := Open(context.Background(), path)
	if store != nil {
		_ = store.Close()
	}
	if !errors.Is(err, ErrDatabaseTooNew) {
		t.Fatalf("expected ErrDatabaseTooNew, got %v", err)
	}
	var versionErr *SchemaVersionError
	if !errors.As(err, &versionErr) || versionErr.Found != 999 || versionErr.Supported != CurrentSchemaVersion {
		t.Fatalf("unexpected typed version error: %#v", err)
	}
}

func TestConcurrentOpenBootstrapsOneConsistentSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.sqlite")
	const openers = 8
	start := make(chan struct{})
	errs := make(chan error, openers)
	var wg sync.WaitGroup
	for range openers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			store, err := Open(context.Background(), path)
			if err == nil {
				err = store.Close()
			}
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent open: %v", err)
		}
	}

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open bootstrapped database: %v", err)
	}
	defer func() { _ = store.Close() }()
	if got := schemaVersionForTest(t, store.DB()); got != CurrentSchemaVersion {
		t.Fatalf("schema version=%d, want %d", got, CurrentSchemaVersion)
	}
	for _, name := range []string{"sessions", "evaluations", "binding_versions", "binding_docs"} {
		if !tableExists(t, store.DB(), name) {
			t.Fatalf("concurrent bootstrap omitted table %q", name)
		}
	}
}

func copyV1Fixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "repl-v1.sqlite"))
	if err != nil {
		t.Fatalf("read v1 fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "repl-v1.sqlite")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("copy v1 fixture: %v", err)
	}
	return path
}

func schemaVersionForTest(t *testing.T, db *sql.DB) int {
	t.Helper()
	version, err := readSchemaVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	return version
}
