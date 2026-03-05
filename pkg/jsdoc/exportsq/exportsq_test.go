package exportsq

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestWriteFile_SQLiteSchemaAndCounts(t *testing.T) {
	store := model.NewDocStore()
	store.AddFile(&model.FileDoc{
		FilePath: "test.js",
		Package:  &model.Package{Name: "pkg"},
		Symbols: []*model.SymbolDoc{
			{Name: "fn", Tags: []string{"t1", "t2"}, Concepts: []string{"c1"}},
		},
		Examples: []*model.Example{
			{ID: "ex1", Symbols: []string{"fn"}},
		},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "out.sqlite")
	if err := WriteFile(context.Background(), store, path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected sqlite file to exist: %v", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	assertCount(t, db, "packages", 1)
	assertCount(t, db, "symbols", 1)
	assertCount(t, db, "symbol_tags", 2)
	assertCount(t, db, "symbol_concepts", 1)
	assertCount(t, db, "examples", 1)
	assertCount(t, db, "example_symbols", 1)
}

func assertCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	row := db.QueryRow("SELECT COUNT(*) FROM " + table)
	var got int
	if err := row.Scan(&got); err != nil {
		t.Fatalf("scan count(%s): %v", table, err)
	}
	if got != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, got)
	}
}
