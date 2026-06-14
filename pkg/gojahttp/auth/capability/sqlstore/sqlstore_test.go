package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/capabilitytest"
)

func TestSQLiteStoreContract(t *testing.T) {
	capabilitytest.RunStoreContract(t, func(tb testing.TB) capability.Store {
		tb.Helper()
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			tb.Fatalf("open sqlite: %v", err)
		}
		db.SetMaxOpenConns(1)
		tb.Cleanup(func() { _ = db.Close() })
		store, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: sqlstore.DialectSQLite})
		if err != nil {
			tb.Fatalf("new store: %v", err)
		}
		if err := store.ApplySchema(context.Background()); err != nil {
			tb.Fatalf("apply schema: %v", err)
		}
		return store
	})
}

func TestNewValidation(t *testing.T) {
	if _, err := sqlstore.New(sqlstore.Config{}); err == nil {
		t.Fatalf("expected missing db error")
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: "bogus"}); err == nil {
		t.Fatalf("expected unsupported dialect error")
	}
}
