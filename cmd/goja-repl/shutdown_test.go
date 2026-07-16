package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestCloseAppAndStoreReleasesLeaseBeforeClosingSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shutdown.sqlite")
	appA, storeA := newShutdownTestApp(t, dbPath)
	session, err := appA.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := closeAppAndStore(appA, storeA); err != nil {
		t.Fatalf("close first app/store: %v", err)
	}

	appB, storeB := newShutdownTestApp(t, dbPath)
	if _, err := appB.Restore(context.Background(), session.ID); err != nil {
		t.Fatalf("restore after orderly lease release: %v", err)
	}
	if err := closeAppAndStore(appB, storeB); err != nil {
		t.Fatalf("close second app/store: %v", err)
	}
}

func TestCloseAppAndStoreReportsAppShutdownFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "closed-early.sqlite")
	app, store := newShutdownTestApp(t, dbPath)
	if _, err := app.CreateSession(context.Background()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store early: %v", err)
	}

	err := closeAppAndStore(app, store)
	if err == nil {
		t.Fatal("expected app lease-release failure after premature store close")
	}
	if !strings.Contains(err.Error(), "close repl app") || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("expected contextual shutdown diagnostics, got %v", err)
	}
}

func newShutdownTestApp(t *testing.T, dbPath string) (*replapi.App, *repldb.Store) {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	store, err := repldb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app, err := replapi.New(context.Background(), factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	if err != nil {
		_ = store.Close()
		t.Fatalf("new app: %v", err)
	}
	return app, store
}
