package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth"
)

func TestSQLiteStoreConsumesTransactionOnce(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	store := newSQLiteStore(t, func() time.Time { return now })
	tx := transaction(now)
	if err := store.Put(context.Background(), tx); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := store.Take(context.Background(), tx.State)
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	if got != tx {
		t.Fatalf("transaction = %#v, want %#v", got, tx)
	}
	_, err = store.Take(context.Background(), tx.State)
	if !errors.Is(err, keycloakauth.ErrTransactionUnavailable) {
		t.Fatalf("second Take error = %v, want ErrTransactionUnavailable", err)
	}
}

func TestSQLiteStoreRejectsExpiredTransactionWithoutCleanup(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 10, 0, 0, time.UTC)
	store := newSQLiteStore(t, func() time.Time { return now })
	tx := transaction(now.Add(-11 * time.Minute))
	if err := store.Put(context.Background(), tx); err != nil {
		t.Fatalf("Put: %v", err)
	}
	_, err := store.Take(context.Background(), tx.State)
	if !errors.Is(err, keycloakauth.ErrTransactionUnavailable) {
		t.Fatalf("Take expired error = %v, want ErrTransactionUnavailable", err)
	}
	count, err := store.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if count != 1 {
		t.Fatalf("Cleanup count = %d, want 1", count)
	}
}

func TestSQLiteStoreConcurrentTakeHasOneWinner(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	store := newSQLiteStore(t, func() time.Time { return now })
	tx := transaction(now)
	if err := store.Put(context.Background(), tx); err != nil {
		t.Fatalf("Put: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Take(context.Background(), tx.State)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, keycloakauth.ErrTransactionUnavailable) {
			t.Fatalf("Take error = %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent takes = %d, want 1", successes)
	}
}

func TestSQLiteStoreSurvivesStoreReopen(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "transactions.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{DB: db, Dialect: DialectSQLite, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	tx := transaction(now)
	if err := store.Put(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err = New(Config{DB: db, Dialect: DialectSQLite, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Take(context.Background(), tx.State)
	if err != nil {
		t.Fatalf("Take after reopen: %v", err)
	}
	if got != tx {
		t.Fatalf("transaction after reopen = %#v, want %#v", got, tx)
	}
}

func TestPostgresSchemaAndQueries(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectPostgres})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(store.Schema(), "TIMESTAMPTZ") {
		t.Fatalf("postgres schema = %s", store.Schema())
	}
	if !strings.Contains(store.takeQuery(), "$1") || !strings.Contains(store.takeQuery(), "$2") {
		t.Fatalf("postgres take query = %s", store.takeQuery())
	}
}

func newSQLiteStore(t testing.TB, now func() time.Time) *Store {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:keycloakauth-transaction-store?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(Config{DB: db, Dialect: DialectSQLite, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func transaction(createdAt time.Time) keycloakauth.Transaction {
	return keycloakauth.Transaction{State: "state-1", Nonce: "nonce-1", PKCEVerifier: "verifier-1", CreatedAt: createdAt, RedirectURL: "/after"}
}
