package fuzz

import (
	"context"
	"testing"
)

func FuzzSessionSequence(f *testing.F) {
	for _, pair := range SequenceSeeds() {
		f.Add(pair[0], pair[1])
	}

	f.Fuzz(func(t *testing.T, first, second string) {
		ctx := context.Background()
		app := newInteractiveApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		// Evaluate first input — errors are fine, panics are not.
		panicked, panicVal, _, _ := safeEval(ctx, app, session.ID, first)
		if panicked {
			t.Fatalf("panic on first evaluate first=%q: %v", truncate(first, 60), panicVal)
		}

		// Evaluate second input.
		panicked, panicVal, resp, _ := safeEval(ctx, app, session.ID, second)
		if panicked {
			t.Fatalf("panic on second evaluate first=%q second=%q: %v",
				truncate(first, 40), truncate(second, 40), panicVal)
		}

		// Invariant: session has at least 2 cells (even if errors).
		if resp != nil && resp.Session.CellCount < 2 {
			t.Fatalf("expected >= 2 cells after two evaluations, got %d", resp.Session.CellCount)
		}
	})
}

// FuzzSessionSequenceRaw is the same as FuzzSessionSequence but with raw mode.
func FuzzSessionSequenceRaw(f *testing.F) {
	for _, pair := range SequenceSeeds() {
		f.Add(pair[0], pair[1])
	}

	f.Fuzz(func(t *testing.T, first, second string) {
		ctx := context.Background()
		app := newRawApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, panicVal, _, _ := safeEval(ctx, app, session.ID, first)
		if panicked {
			t.Fatalf("panic on raw first first=%q: %v", truncate(first, 60), panicVal)
		}

		panicked, panicVal, _, _ = safeEval(ctx, app, session.ID, second)
		if panicked {
			t.Fatalf("panic on raw second first=%q second=%q: %v",
				truncate(first, 40), truncate(second, 40), panicVal)
		}
	})
}

// TestFuzzSessionSequenceState verifies state survives across cells.
func TestFuzzSessionSequenceState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Declare a variable.
	_, _, resp, err := safeEval(ctx, app, session.ID, "const x = 42")
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("declare status: %q", resp.Cell.Execution.Status)
	}

	// Use it in the next cell.
	_, _, resp, err = safeEval(ctx, app, session.ID, "x + 1")
	if err != nil {
		t.Fatalf("use: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("use status: %q", resp.Cell.Execution.Status)
	}
	if resp.Cell.Execution.Result != "43" {
		t.Fatalf("expected 43, got %q", resp.Cell.Execution.Result)
	}

	// Invariant: binding count is 1 (just "x").
	if resp.Session.BindingCount != 1 {
		t.Fatalf("expected 1 binding, got %d", resp.Session.BindingCount)
	}
}

// TestFuzzSessionSequenceRedeclare verifies re-declaring a variable updates the binding.
func TestFuzzSessionSequenceRedeclare(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, _, resp1, _ := safeEval(ctx, app, session.ID, "const x = 1")
	if resp1.Cell.Execution.Status != "ok" {
		t.Fatalf("first: %q", resp1.Cell.Execution.Status)
	}

	_, _, resp2, _ := safeEval(ctx, app, session.ID, "const x = 2")
	if resp2.Cell.Execution.Status != "ok" {
		t.Fatalf("second: %q", resp2.Cell.Execution.Status)
	}

	// The value should now be 2.
	_, _, resp3, _ := safeEval(ctx, app, session.ID, "x")
	if resp3.Cell.Execution.Result != "2" {
		t.Fatalf("expected 2, got %q", resp3.Cell.Execution.Result)
	}

	// Invariant: still only 1 binding (x), not 2.
	if resp3.Session.BindingCount != 1 {
		t.Fatalf("expected 1 binding after re-declare, got %d", resp3.Session.BindingCount)
	}
}

// TestFuzzSessionSequenceErrorRecovery verifies session works after an error.
func TestFuzzSessionSequenceErrorRecovery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app := newInteractiveApp(t)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Declare something.
	_, _, _, _ = safeEval(ctx, app, session.ID, "const x = 10")

	// Cause an error.
	_, _, resp2, _ := safeEval(ctx, app, session.ID, "throw new Error('boom')")
	if resp2.Cell.Execution.Status != "runtime-error" {
		t.Fatalf("expected runtime-error, got %q", resp2.Cell.Execution.Status)
	}

	// Session should still work.
	_, _, resp3, _ := safeEval(ctx, app, session.ID, "x + 5")
	if resp3.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok after error recovery, got %q", resp3.Cell.Execution.Status)
	}
	if resp3.Cell.Execution.Result != "15" {
		t.Fatalf("expected 15, got %q", resp3.Cell.Execution.Result)
	}
}
