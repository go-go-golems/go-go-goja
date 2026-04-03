package replapi

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestAppRestoresPersistedSessionAndContinuesEvaluation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	factory := newTestFactory(t)
	app1, err := New(factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new app1: %v", err)
	}
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := app1.Evaluate(ctx, session.ID, "const x = 1; x"); err != nil {
		t.Fatalf("first evaluate: %v", err)
	}

	// Simulate a new process by creating a fresh app against the same store.
	app2, err := New(factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new app2: %v", err)
	}
	snapshot, err := app2.Snapshot(ctx, session.ID)
	if err != nil {
		t.Fatalf("restore snapshot: %v", err)
	}
	if snapshot.CellCount != 1 {
		t.Fatalf("expected restored cell count 1, got %d", snapshot.CellCount)
	}

	resp, err := app2.Evaluate(ctx, session.ID, "x + 1")
	if err != nil {
		t.Fatalf("second evaluate: %v", err)
	}
	if resp.Cell.Execution.Result != "2" {
		t.Fatalf("expected result 2, got %q", resp.Cell.Execution.Result)
	}

	history, err := app2.History(ctx, session.ID)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected history length 2, got %d", len(history))
	}
}

func TestAppRawProfileDoesNotRequireStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	factory := newTestFactory(t)
	app, err := New(factory, zerolog.Nop(), WithProfile(ProfileRaw))
	if err != nil {
		t.Fatalf("new raw app: %v", err)
	}

	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.Profile != string(ProfileRaw) {
		t.Fatalf("expected raw profile, got %q", session.Profile)
	}

	resp, err := app.Evaluate(ctx, session.ID, "const x = 1; x")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Result != "1" {
		t.Fatalf("expected result 1, got %q", resp.Cell.Execution.Result)
	}
	if resp.Session.BindingCount != 0 {
		t.Fatalf("expected no tracked bindings in raw mode, got %d", resp.Session.BindingCount)
	}
	if resp.Cell.Rewrite.Mode != "raw" {
		t.Fatalf("expected raw rewrite mode, got %q", resp.Cell.Rewrite.Mode)
	}
}

func TestAppPersistentProfileRequiresStore(t *testing.T) {
	t.Parallel()

	factory := newTestFactory(t)
	if _, err := New(factory, zerolog.Nop(), WithProfile(ProfilePersistent)); err == nil {
		t.Fatal("expected persistent profile without store to fail")
	}
}

func TestAppSessionOverrideCanDropFromPersistentToRaw(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	factory := newTestFactory(t)
	app, err := New(factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	rawProfile := ProfileRaw
	session, err := app.CreateSessionWithOptions(ctx, SessionOptions{Profile: &rawProfile})
	if err != nil {
		t.Fatalf("create raw session override: %v", err)
	}
	if session.Profile != string(ProfileRaw) {
		t.Fatalf("expected raw profile, got %q", session.Profile)
	}

	if _, err := app.Evaluate(ctx, session.ID, "const x = 1; x"); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	sessions, err := app.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected raw override session to avoid durable storage, got %d persisted sessions", len(sessions))
	}
}

func newTestFactory(t *testing.T) *engine.Factory {
	t.Helper()

	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func openTestStore(t *testing.T) *repldb.Store {
	t.Helper()

	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
