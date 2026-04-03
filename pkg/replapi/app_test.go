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
	app1 := New(factory, store, zerolog.Nop())
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := app1.Evaluate(ctx, session.ID, "const x = 1; x"); err != nil {
		t.Fatalf("first evaluate: %v", err)
	}

	// Simulate a new process by creating a fresh app against the same store.
	app2 := New(factory, store, zerolog.Nop())
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
