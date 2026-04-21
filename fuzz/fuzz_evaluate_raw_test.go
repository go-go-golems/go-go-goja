package fuzz

import (
	"context"
	"testing"
)

func FuzzEvaluateRaw(f *testing.F) {
	for _, seed := range SeedsMinimal {
		f.Add(seed)
	}
	for _, seed := range SeedsErrorInputs {
		f.Add(seed)
	}
	for _, seed := range SeedsUnicode {
		f.Add(seed)
	}
	for _, seed := range SeedsTypeCoercion {
		f.Add(seed)
	}
	for _, seed := range SeedsDeepNesting {
		f.Add(seed)
	}
	for _, seed := range SeedsObjectEdgeCases {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		app := newRawApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, panicVal, resp, _ := safeEval(ctx, app, session.ID, source)
		if panicked {
			t.Fatalf("panic on raw evaluate source=%q: %v", truncate(source, 100), panicVal)
		}

		// Invariant: response is never nil when no panic occurs.
		if resp == nil {
			t.Fatalf("nil response for source=%q", truncate(source, 100))
		}

		// Invariant: Cell is always populated.
		if resp.Cell == nil {
			t.Fatalf("nil cell for source=%q", truncate(source, 100))
		}

		// Invariant: status is one of the known values.
		switch resp.Cell.Execution.Status {
		case "ok", "runtime-error", "timeout", "empty-source":
			// valid
		default:
			t.Fatalf("unknown status %q for source=%q", resp.Cell.Execution.Status, truncate(source, 100))
		}

		// Invariant: raw mode never uses instrumented rewrite.
		// Empty source short-circuits to mode "none" instead of "raw".
		if resp.Cell.Rewrite.Mode == "async-iife-with-binding-capture" {
			t.Fatalf("raw mode produced rewrite mode=%q for source=%q", resp.Cell.Rewrite.Mode, truncate(source, 100))
		}
		if resp.Cell.Rewrite.Mode == "none" && resp.Cell.Execution.Status != "empty-source" {
			t.Fatalf("raw mode produced unexpected mode %q for source=%q", resp.Cell.Rewrite.Mode, truncate(source, 100))
		}
		_ = resp.Session
	})
}

// TestFuzzEvaluateRawSeeds runs every seed through the raw path once.
func TestFuzzEvaluateRawSeeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newRawApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	for _, seed := range AllSeeds() {
		func(source string) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on source=%q: %v", truncate(source, 80), r)
				}
			}()
			panicked, _, _, _ := safeEval(ctx, app, session.ID, source)
			if panicked {
				t.Fatalf("panic on source=%q", truncate(source, 80))
			}
		}(seed)
	}

	snap, err := app.Snapshot(ctx, session.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	// Invariant: raw mode does not track bindings.
	if snap.BindingCount != 0 {
		t.Fatalf("raw mode should have 0 bindings, got %d", snap.BindingCount)
	}
	_ = snap.CellCount
}

// TestFuzzEvaluateRawRewriteMode verifies raw mode never uses instrumented rewrite.
func TestFuzzEvaluateRawRewriteMode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newRawApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	sources := []string{
		"const x = 1; x",
		"function f() {}; f()",
		"class A {}; new A()",
	}
	for _, source := range sources {
		_, _, resp, err := safeEval(ctx, app, session.ID, source)
		if err != nil {
			t.Fatalf("evaluate %q: %v", source, err)
		}
		if resp.Cell.Rewrite.Mode != "raw" {
			t.Errorf("expected raw rewrite mode for %q, got %q", source, resp.Cell.Rewrite.Mode)
		}
		// Invariant: raw mode reports zero tracked bindings.
		if resp.Session.BindingCount != 0 {
			t.Errorf("raw mode should have 0 bindings, got %d for %q", resp.Session.BindingCount, source)
		}
	}
}

// TestFuzzEvaluateRawTimeout verifies infinite loops hit the timeout.
func TestFuzzEvaluateRawTimeout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newRawApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// This should time out, not hang.
	_, _, resp, _ := safeEval(ctx, app, session.ID, "while(true){}")
	if resp == nil || resp.Cell == nil {
		t.Fatal("expected non-nil response for timeout")
	}
	if resp.Cell.Execution.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", resp.Cell.Execution.Status)
	}
}

// TestFuzzEvaluateRawSessionState verifies session accumulates cells.
func TestFuzzEvaluateRawSessionState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newRawApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	for i, src := range []string{"1", "2", "3"} {
		_, _, resp, _ := safeEval(ctx, app, session.ID, src)
		if resp == nil {
			t.Fatalf("nil response at index %d", i)
		}
		if resp.Session.CellCount != i+1 {
			t.Fatalf("expected cell count %d, got %d", i+1, resp.Session.CellCount)
		}
	}
}
