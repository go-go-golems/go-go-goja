package fuzz

import (
	"context"
	"testing"
)

// FuzzGlobalSnapshot produces unusual values, then verifies that
// snapshot/binding introspection never panics.
func FuzzGlobalSnapshot(f *testing.F) {
	for _, seed := range SeedsObjectEdgeCases {
		f.Add(seed)
	}
	for _, seed := range SeedsTypeCoercion {
		f.Add(seed)
	}
	for _, seed := range SeedsStressSerialization {
		f.Add(seed)
	}
	for _, seed := range SeedsExpressions {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		app := newInteractiveApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		// Evaluate the source (which produces some value).
		panicked, panicVal, evalResp, _ := safeEval(ctx, app, session.ID, source)
		if panicked {
			t.Fatalf("panic on evaluate source=%q: %v", truncate(source, 100), panicVal)
		}
		if evalResp == nil {
			return
		}

		// Now snapshot and verify it doesn't panic.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on snapshot after source=%q: %v", truncate(source, 100), r)
				}
			}()

			snapshot, err := app.Snapshot(ctx, session.ID)
			if err != nil {
				return
			}

			// Invariant: snapshot is never nil on success.
			if snapshot == nil {
				t.Fatalf("nil snapshot for source=%q", truncate(source, 100))
			}

			// Invariant: bindings list is never nil.
			if snapshot.Bindings == nil {
				t.Fatalf("nil bindings slice for source=%q", truncate(source, 100))
			}

			// Invariant: history is never nil.
			if snapshot.History == nil {
				t.Fatalf("nil history for source=%q", truncate(source, 100))
			}

			// Invariant: current globals are populated.
			if snapshot.CurrentGlobals == nil {
				t.Fatalf("nil current globals for source=%q", truncate(source, 100))
			}

			// Invariant: each binding has a non-empty name.
			for _, b := range snapshot.Bindings {
				if b.Name == "" {
					t.Fatalf("binding with empty name for source=%q", truncate(source, 100))
				}
				// Invariant: runtime view is populated.
				if b.Runtime.ValueKind == "" {
					t.Fatalf("binding %q has empty ValueKind for source=%q", b.Name, truncate(source, 100))
				}
			}
		}()

		// Verify Bindings() endpoint doesn't panic either.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on Bindings() after source=%q: %v", truncate(source, 100), r)
				}
			}()
			_, _ = app.Bindings(ctx, session.ID)
		}()
	})
}

// TestFuzzGlobalSnapshotBasic verifies snapshot after simple evaluations.
func TestFuzzGlobalSnapshotBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, _, _, _ = safeEval(ctx, app, session.ID, "const x = { a: 1, b: 'hello' }")
	_, _, _, _ = safeEval(ctx, app, session.ID, "function f() { return 42 }")

	snapshot, err := app.Snapshot(ctx, session.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.BindingCount != 2 {
		t.Fatalf("expected 2 bindings, got %d", snapshot.BindingCount)
	}

	bindings, err := app.Bindings(ctx, session.ID)
	if err != nil {
		t.Fatalf("bindings: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings from endpoint, got %d", len(bindings))
	}

	// Verify each binding has a value kind.
	for _, b := range bindings {
		if b.Runtime.ValueKind == "" {
			t.Errorf("binding %q has empty ValueKind", b.Name)
		}
		if b.Runtime.Preview == "" {
			t.Errorf("binding %q has empty Preview", b.Name)
		}
	}
}

// TestFuzzGlobalSnapshotUnusualValues verifies snapshot handles unusual types.
func TestFuzzGlobalSnapshotUnusualValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []string{
		"const a = Object.create(null)",
		"const b = new Proxy({}, {})",
		"const c = Symbol('test')",
		"const d = new Map()",
		"const e = new Set()",
		"const f = /regex/gi",
		"const g = new Date()",
		"const h = new Error('err')",
		"const i = undefined",
		"const j = null",
	}

	for _, src := range cases {
		func(source string) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic for source=%q: %v", source, r)
				}
			}()

			app := newInteractiveApp(t)
			session, _ := app.CreateSession(ctx)
			_, _, _, _ = safeEval(ctx, app, session.ID, source)
			_, err := app.Snapshot(ctx, session.ID)
			if err != nil {
				t.Fatalf("snapshot failed for %q: %v", source, err)
			}
		}(src)
	}
}
