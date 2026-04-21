package fuzz

import (
	"context"
	"path/filepath"
	"testing"
)

func FuzzPersistenceRoundTrip(f *testing.F) {
	for _, triple := range PersistenceSeeds() {
		f.Add(triple[0], triple[1], triple[2])
	}

	f.Fuzz(func(t *testing.T, seed, restore, continuation string) {
		ctx := context.Background()

		// Phase 1: Seed a persistent session.
		app1, store1 := newPersistentApp(t)

		session, err := app1.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = store1.Close()
					t.Fatalf("panic in seed phase source=%q: %v", truncate(seed, 60), r)
				}
			}()
			_, _, _, _ = safeEval(ctx, app1, session.ID, seed)
		}()

		sessionID := session.ID
		_ = store1.Close()

		// Phase 2: Restore in a new app and verify.
		app2, store2 := newPersistentApp(t)
		defer func() { _ = store2.Close() }()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic in restore phase sessionID=%s: %v", sessionID, r)
				}
			}()

			snapshot, err := app2.Snapshot(ctx, sessionID)
			if err != nil {
				return // session may not restore cleanly from invalid seed
			}

			// Invariant: restored session preserves the ID.
			if snapshot.ID != sessionID {
				t.Fatalf("restored ID mismatch: got %q, want %q", snapshot.ID, sessionID)
			}

			// Evaluate a second input on the restored session.
			_, _, _, _ = safeEval(ctx, app2, sessionID, restore)

			// And a third.
			_, _, _, _ = safeEval(ctx, app2, sessionID, continuation)
		}()
	})
}

// TestFuzzPersistenceBasicRoundTrip tests a simple persist-restore-continue cycle.
func TestFuzzPersistenceBasicRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "repl.sqlite")

	// Phase 1: Seed.
	app1, store1 := newPersistentAppAtPath(t, dbPath)
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	_, _, resp1, err := safeEval(ctx, app1, session.ID, "const x = 42")
	if err != nil {
		t.Fatalf("seed evaluate: %v", err)
	}
	if resp1.Cell.Execution.Status != "ok" {
		t.Fatalf("seed status: %q", resp1.Cell.Execution.Status)
	}
	_ = store1.Close()

	// Phase 2: Restore in new app on same DB.
	app2, store2 := newPersistentAppAtPath(t, dbPath)
	defer func() { _ = store2.Close() }()

	snapshot, err := app2.Snapshot(ctx, session.ID)
	if err != nil {
		t.Fatalf("restore snapshot: %v", err)
	}
	if snapshot.CellCount != 1 {
		t.Fatalf("expected 1 restored cell, got %d", snapshot.CellCount)
	}
	if snapshot.BindingCount != 1 {
		t.Fatalf("expected 1 restored binding, got %d", snapshot.BindingCount)
	}

	// Phase 3: Continue.
	_, _, resp3, err := safeEval(ctx, app2, session.ID, "x + 1")
	if err != nil {
		t.Fatalf("continue evaluate: %v", err)
	}
	if resp3.Cell.Execution.Result != "43" {
		t.Fatalf("expected 43, got %q", resp3.Cell.Execution.Result)
	}
}

// TestFuzzPersistenceHistoryPreserved verifies history is readable after restore.
func TestFuzzPersistenceHistoryPreserved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "repl.sqlite")

	app1, store1 := newPersistentAppAtPath(t, dbPath)
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	sources := []string{"const x = 1", "x + 1", "const y = 2"}
	for _, src := range sources {
		_, _, _, _ = safeEval(ctx, app1, session.ID, src)
	}
	_ = store1.Close()

	// Check history via new app.
	app2, store2 := newPersistentAppAtPath(t, dbPath)
	defer func() { _ = store2.Close() }()

	history, err := app2.History(ctx, session.ID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(history))
	}
}

// TestFuzzPersistenceDelete verifies deleted sessions can't be restored.
func TestFuzzPersistenceDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	app1, store1 := newPersistentApp(t)
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	_, _, _, _ = safeEval(ctx, app1, session.ID, "const x = 1")

	if err := app1.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = store1.Close()

	app2, store2 := newPersistentApp(t)
	defer func() { _ = store2.Close() }()

	sessions, err := app2.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == session.ID {
			t.Fatal("deleted session should not appear in list")
		}
	}
}
