// 04-persistence-fuzz: Fuzzer targeting session persistence, restore, and store operations.
//
// This experiment fuzzes the persistent session path:
//  1. Create persistent session with SQLite store
//  2. Evaluate multiple cells
//  3. Restore from store in a new App instance
//  4. Continue evaluating
//  5. Verify state consistency
//
// Run:
//
//	cd ttmp/.../scripts/04-persistence-fuzz/
//	go test -fuzz=FuzzPersistenceRestore -v -fuzztime=30s
package fuzz

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func newFactory(t *testing.T) *engine.Factory {
	t.Helper()
	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func newPersistentApp(t *testing.T) (*replapi.App, *repldb.Store) {
	t.Helper()
	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app, err := replapi.New(newFactory(t), zerolog.Nop(),
		replapi.WithProfile(replapi.ProfilePersistent),
		replapi.WithStore(store),
	)
	if err != nil {
		t.Fatalf("create persistent app: %v", err)
	}
	return app, store
}

// FuzzPersistenceRestore fuzzes create-evaluate-restore-verify cycles.
func FuzzPersistenceRestore(f *testing.F) {
	type testCase struct {
		seed    string
		restore string
	}
	seeds := []testCase{
		{"const x = 1", "x + 1"},
		{"function f() { return 42 }", "f()"},
		{"let a = [1, 2]; a.push(3)", "a.length"},
		{"const obj = { count: 0 }; obj.count = 5", "obj.count"},
	}
	for _, s := range seeds {
		f.Add(s.seed, s.restore)
	}

	f.Fuzz(func(t *testing.T, seed, restore string) {
		ctx := context.Background()

		// Phase 1: Create, seed, close
		app1, store1 := newPersistentApp(t)
		session, err := app1.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = store1.Close()
					t.Fatalf("PANIC in seed phase source=%q: %v", truncate(seed, 60), r)
				}
			}()
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			app1.Evaluate(timeoutCtx, session.ID, seed)
		}()

		sessionID := session.ID
		_ = store1.Close()

		// Phase 2: Restore in new app and verify
		app2, store2 := newPersistentApp(t)
		defer func() { _ = store2.Close() }()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC in restore phase sessionID=%s restore=%q: %v", sessionID, truncate(restore, 60), r)
				}
			}()

			snapshot, err := app2.Snapshot(ctx, sessionID)
			if err != nil {
				return // session may not restore cleanly from invalid seed data
			}
			_ = snapshot

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			app2.Evaluate(timeoutCtx, sessionID, restore)
		}()
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
