package fuzz

import (
	"context"
	"testing"
)

func FuzzEvaluateInstrumented(f *testing.F) {
	for _, seed := range SeedsMinimal {
		f.Add(seed)
	}
	for _, seed := range SeedsDeclarations {
		f.Add(seed)
	}
	for _, seed := range SeedsExpressions {
		f.Add(seed)
	}
	for _, seed := range SeedsMixed {
		f.Add(seed)
	}
	for _, seed := range SeedsAsync {
		f.Add(seed)
	}
	for _, seed := range SeedsErrorInputs {
		f.Add(seed)
	}
	for _, seed := range SeedsUnicode {
		f.Add(seed)
	}
	for _, seed := range SeedsObjectEdgeCases {
		f.Add(seed)
	}
	for _, seed := range SeedsConsoleCapture {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		app := newInteractiveApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, panicVal, resp, _ := safeEval(ctx, app, session.ID, source)
		if panicked {
			t.Fatalf("panic on instrumented evaluate source=%q: %v", truncate(source, 100), panicVal)
		}

		// Invariant: response is never nil.
		if resp == nil || resp.Cell == nil {
			t.Fatalf("nil response for source=%q", truncate(source, 100))
		}

		cell := resp.Cell

		// Invariant: status is a known value.
		switch cell.Execution.Status {
		case "ok", "runtime-error", "timeout", "parse-error", "empty-source", "helper-error":
			// valid
		default:
			t.Fatalf("unknown status %q for source=%q", cell.Execution.Status, truncate(source, 100))
		}

		// Invariant: mode is instrumented (not raw).
		if cell.Rewrite.Mode == "raw" {
			t.Fatalf("interactive mode produced raw rewrite for source=%q", truncate(source, 100))
		}

		// Invariant: Cell ID is always >= 1.
		if cell.ID < 1 {
			t.Fatalf("cell ID %d < 1 for source=%q", cell.ID, truncate(source, 100))
		}

		// Invariant: Runtime report fields are populated in instrumented mode.
		if cell.Rewrite.Mode == "async-iife-with-binding-capture" {
			if cell.Runtime.BeforeGlobals == nil {
				t.Fatalf("nil BeforeGlobals for source=%q", truncate(source, 100))
			}
			if cell.Runtime.AfterGlobals == nil {
				t.Fatalf("nil AfterGlobals for source=%q", truncate(source, 100))
			}
		}

		// Invariant: successful evaluations have a non-negative duration.
		if cell.Execution.DurationMS < 0 {
			t.Fatalf("negative duration %d for source=%q", cell.Execution.DurationMS, truncate(source, 100))
		}

		// Invariant: Console events are captured in interactive mode.
		if cell.Execution.Console == nil {
			t.Fatalf("nil console slice for source=%q", truncate(source, 100))
		}
	})
}

// TestFuzzInstrumentedSeeds runs every seed through the instrumented path.
func TestFuzzInstrumentedSeeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	for _, seed := range AllSeeds() {
		func(source string) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic for source=%q: %v", truncate(source, 80), r)
				}
			}()
			panicked, _, _, _ := safeEval(ctx, app, session.ID, source)
			if panicked {
				t.Fatalf("panic for source=%q", truncate(source, 80))
			}
		}(seed)
	}
}

// TestFuzzInstrumentedBindingCapture verifies declarations are captured as bindings.
func TestFuzzInstrumentedBindingCapture(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	cases := []struct {
		source   string
		wantBind string
	}{
		{"const x = 1", "x"},
		{"function f() { return 1 }", "f"},
		{"class A {}", "A"},
	}

	for _, tc := range cases {
		_, _, resp, err := safeEval(ctx, app, session.ID, tc.source)
		if err != nil {
			t.Fatalf("evaluate %q: %v", tc.source, err)
		}
		if resp.Cell.Execution.Status != "ok" {
			t.Fatalf("expected ok for %q, got %q", tc.source, resp.Cell.Execution.Status)
		}

		// Check that the binding appears in the response.
		found := false
		for _, name := range resp.Cell.Runtime.NewBindings {
			if name == tc.wantBind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("binding %q not found in NewBindings for %q (got %v)",
				tc.wantBind, tc.source, resp.Cell.Runtime.NewBindings)
		}

		// Check session-level binding tracking.
		found = false
		for _, b := range resp.Session.Bindings {
			if b.Name == tc.wantBind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("binding %q not in session bindings for %q", tc.wantBind, tc.source)
		}
	}
}

// TestFuzzInstrumentedConsoleCapture verifies console events are captured.
func TestFuzzInstrumentedConsoleCapture(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, _, resp, err := safeEval(ctx, app, session.ID, "console.log('hello'); 42")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok, got %q", resp.Cell.Execution.Status)
	}
	if len(resp.Cell.Execution.Console) != 1 {
		t.Fatalf("expected 1 console event, got %d", len(resp.Cell.Execution.Console))
	}
	if resp.Cell.Execution.Console[0].Kind != "log" {
		t.Fatalf("expected console event kind 'log', got %q", resp.Cell.Execution.Console[0].Kind)
	}
}

// TestFuzzInstrumentedCellCountMonotonic verifies cell IDs increase.
func TestFuzzInstrumentedCellCountMonotonic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var lastID int
	for _, src := range []string{"1", "2", "3", "const x = 1", "x"} {
		_, _, resp, _ := safeEval(ctx, app, session.ID, src)
		if resp.Cell.ID <= lastID {
			t.Fatalf("cell ID %d not monotonic after %d for source=%q", resp.Cell.ID, lastID, src)
		}
		lastID = resp.Cell.ID
	}
}
