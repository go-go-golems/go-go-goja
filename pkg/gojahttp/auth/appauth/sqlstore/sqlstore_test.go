package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/appauthtest"
)

func TestSQLiteStoreContract(t *testing.T) {
	appauthtest.RunStoreContract(t, func(tb testing.TB) appauthtest.Harness {
		tb.Helper()
		store := newSQLiteStore(tb)
		return appauthtest.Harness{
			Users:       store,
			Memberships: store,
			Resources:   store,
			AddUser: func(user appauth.User) {
				tb.Helper()
				if err := store.AddUser(context.Background(), user); err != nil {
					tb.Fatalf("add user: %v", err)
				}
			},
			AddMember: func(membership appauth.Membership) {
				tb.Helper()
				if err := store.AddMembership(context.Background(), membership); err != nil {
					tb.Fatalf("add membership: %v", err)
				}
			},
			AddResource: func(resource appauth.Resource) {
				tb.Helper()
				if err := store.AddResource(context.Background(), resource); err != nil {
					tb.Fatalf("add resource: %v", err)
				}
			},
		}
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

func newSQLiteStore(tb testing.TB) *sqlstore.Store {
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
}
