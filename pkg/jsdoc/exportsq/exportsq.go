// Package exportsq exports a DocStore into a SQLite database.
package exportsq

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

type Options struct{}

// Write exports store into a temporary sqlite file and streams it to w.
func Write(ctx context.Context, store *model.DocStore, w io.Writer, _ Options) error {
	if store == nil {
		return errors.New("store is nil")
	}
	if w == nil {
		return errors.New("writer is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	tmpDir := os.TempDir()
	f, err := os.CreateTemp(tmpDir, "goja-jsdoc-*.sqlite")
	if err != nil {
		return errors.Wrap(err, "creating temp sqlite file")
	}
	path := f.Name()
	_ = f.Close()
	defer func() { _ = os.Remove(path) }()

	if err := WriteFile(ctx, store, path); err != nil {
		return err
	}

	r, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "opening temp sqlite file")
	}
	defer func() { _ = r.Close() }()

	_, err = io.Copy(w, r)
	return errors.Wrap(err, "streaming sqlite")
}

// WriteFile writes store into a sqlite database file at path.
func WriteFile(ctx context.Context, store *model.DocStore, path string) error {
	if store == nil {
		return errors.New("store is nil")
	}
	if path == "" {
		return errors.New("path is empty")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err, "creating sqlite directory")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return errors.Wrap(err, "opening sqlite")
	}
	defer func() { _ = db.Close() }()

	if err := createSchema(db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "begin tx")
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertPackages(tx, store); err != nil {
		return err
	}
	if err := insertSymbols(tx, store); err != nil {
		return err
	}
	if err := insertExamples(tx, store); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "commit tx")
	}
	return nil
}

func createSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS packages (
			name TEXT PRIMARY KEY,
			title TEXT,
			category TEXT,
			guide TEXT,
			version TEXT,
			description TEXT,
			prose TEXT,
			source_file TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS symbols (
			name TEXT PRIMARY KEY,
			summary TEXT,
			docpage TEXT,
			prose TEXT,
			source_file TEXT,
			line INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS symbol_tags (
			symbol_name TEXT,
			tag TEXT,
			PRIMARY KEY(symbol_name, tag)
		);`,
		`CREATE TABLE IF NOT EXISTS symbol_concepts (
			symbol_name TEXT,
			concept TEXT,
			PRIMARY KEY(symbol_name, concept)
		);`,
		`CREATE TABLE IF NOT EXISTS examples (
			id TEXT PRIMARY KEY,
			title TEXT,
			docpage TEXT,
			body TEXT,
			source_file TEXT,
			line INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS example_symbols (
			example_id TEXT,
			symbol_name TEXT,
			PRIMARY KEY(example_id, symbol_name)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_symbol_tags_symbol_name ON symbol_tags(symbol_name);`,
		`CREATE INDEX IF NOT EXISTS idx_symbol_concepts_symbol_name ON symbol_concepts(symbol_name);`,
		`CREATE INDEX IF NOT EXISTS idx_example_symbols_example_id ON example_symbols(example_id);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return errors.Wrap(err, "creating schema")
		}
	}
	return nil
}

func insertPackages(tx *sql.Tx, store *model.DocStore) error {
	names := make([]string, 0, len(store.ByPackage))
	for name := range store.ByPackage {
		names = append(names, name)
	}
	sort.Strings(names)

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO packages(name, title, category, guide, version, description, prose, source_file) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare packages")
	}
	defer func() { _ = stmt.Close() }()

	for _, name := range names {
		p := store.ByPackage[name]
		if _, err := stmt.Exec(p.Name, p.Title, p.Category, p.Guide, p.Version, p.Description, p.Prose, p.SourceFile); err != nil {
			return errors.Wrap(err, "insert package")
		}
	}
	return nil
}

func insertSymbols(tx *sql.Tx, store *model.DocStore) error {
	names := make([]string, 0, len(store.BySymbol))
	for name := range store.BySymbol {
		names = append(names, name)
	}
	sort.Strings(names)

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO symbols(name, summary, docpage, prose, source_file, line) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare symbols")
	}
	defer func() { _ = stmt.Close() }()

	tagStmt, err := tx.Prepare(`INSERT OR REPLACE INTO symbol_tags(symbol_name, tag) VALUES(?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare symbol_tags")
	}
	defer func() { _ = tagStmt.Close() }()

	conceptStmt, err := tx.Prepare(`INSERT OR REPLACE INTO symbol_concepts(symbol_name, concept) VALUES(?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare symbol_concepts")
	}
	defer func() { _ = conceptStmt.Close() }()

	for _, name := range names {
		s := store.BySymbol[name]
		if _, err := stmt.Exec(s.Name, s.Summary, s.DocPage, s.Prose, s.SourceFile, s.Line); err != nil {
			return errors.Wrap(err, "insert symbol")
		}
		for _, tag := range s.Tags {
			if _, err := tagStmt.Exec(s.Name, tag); err != nil {
				return errors.Wrap(err, "insert symbol tag")
			}
		}
		for _, c := range s.Concepts {
			if _, err := conceptStmt.Exec(s.Name, c); err != nil {
				return errors.Wrap(err, "insert symbol concept")
			}
		}
	}
	return nil
}

func insertExamples(tx *sql.Tx, store *model.DocStore) error {
	ids := make([]string, 0, len(store.ByExample))
	for id := range store.ByExample {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO examples(id, title, docpage, body, source_file, line) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare examples")
	}
	defer func() { _ = stmt.Close() }()

	exSymStmt, err := tx.Prepare(`INSERT OR REPLACE INTO example_symbols(example_id, symbol_name) VALUES(?, ?)`)
	if err != nil {
		return errors.Wrap(err, "prepare example_symbols")
	}
	defer func() { _ = exSymStmt.Close() }()

	for _, id := range ids {
		ex := store.ByExample[id]
		if _, err := stmt.Exec(ex.ID, ex.Title, ex.DocPage, ex.Body, ex.SourceFile, ex.Line); err != nil {
			return errors.Wrap(err, "insert example")
		}

		syms := append([]string(nil), ex.Symbols...)
		sort.Strings(syms)
		for _, sym := range syms {
			if _, err := exSymStmt.Exec(ex.ID, sym); err != nil {
				return errors.Wrap(err, "insert example symbol")
			}
		}
	}
	return nil
}
