package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	report := &testReport{}

	// --- Bootstrap ---
	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: build factory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(os.TempDir(), "goja-repl-lowlevel-test.sqlite")
	store, err := repldb.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: open store: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = store.Close()
		_ = os.Remove(dbPath)
	}()

	service := replsession.NewService(factory, zerolog.Nop(),
		replsession.WithPersistence(store),
	)

	fmt.Println("=== Low-Level replsession Smoke Tests ===")
	fmt.Println()

	// ================================================================
	// T01: Empty source (reproduce BUG-1 directly)
	// ================================================================
	report.run("T01: Empty source crashes service", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		// This should panic or return an error — we want to see which
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("    PANIC: %v\n", r)
			}
		}()
		_, err = service.Evaluate(ctx, session.ID, "")
		return err
	})

	// ================================================================
	// T02: Whitespace-only source
	// ================================================================
	report.run("T02: Whitespace-only source crashes service", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("    PANIC: %v\n", r)
			}
		}()
		_, err = service.Evaluate(ctx, session.ID, "   ")
		return err
	})

	// ================================================================
	// T03: Basic binding lifecycle
	// ================================================================
	report.run("T03: Basic binding lifecycle", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "const x = 42; x")
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok, got %s", resp.Cell.Execution.Status)
		}
		if resp.Cell.Execution.Result != "42" {
			return fmt.Errorf("expected 42, got %q", resp.Cell.Execution.Result)
		}
		if resp.Session.BindingCount != 1 {
			return fmt.Errorf("expected 1 binding, got %d", resp.Session.BindingCount)
		}
		b := resp.Session.Bindings[0]
		if b.Name != "x" || b.Kind != "const" || b.Origin != "declared-top-level" {
			return fmt.Errorf("unexpected binding: name=%q kind=%q origin=%q", b.Name, b.Kind, b.Origin)
		}
		if b.Runtime.ValueKind != "number" || b.Runtime.Preview != "42" {
			return fmt.Errorf("unexpected runtime view: kind=%q preview=%q", b.Runtime.ValueKind, b.Runtime.Preview)
		}
		return nil
	})

	// ================================================================
	// T04: Binding persistence across cells
	// ================================================================
	report.run("T04: Binding cross-cell reference", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "const a = 10")
		if err != nil {
			return fmt.Errorf("cell 1: %w", err)
		}
		resp, err := service.Evaluate(ctx, session.ID, "a + 5")
		if err != nil {
			return fmt.Errorf("cell 2: %w", err)
		}
		if resp.Cell.Execution.Result != "15" {
			return fmt.Errorf("expected 15, got %q", resp.Cell.Execution.Result)
		}
		return nil
	})

	// ================================================================
	// T05: Rewrite report contents
	// ================================================================
	report.run("T05: Rewrite report captures last expression", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "const y = 100; y + 1")
		if err != nil {
			return err
		}
		rw := resp.Cell.Rewrite
		if rw.Mode != "async-iife-with-binding-capture" {
			return fmt.Errorf("unexpected mode: %q", rw.Mode)
		}
		if !rw.CapturedLastExpr {
			return fmt.Errorf("expected CapturedLastExpr=true")
		}
		if rw.FinalExpressionSrc != "y + 1" {
			return fmt.Errorf("expected finalExpressionSrc='y + 1', got %q", rw.FinalExpressionSrc)
		}
		if len(rw.DeclaredNames) != 1 || rw.DeclaredNames[0] != "y" {
			return fmt.Errorf("expected declaredNames=[y], got %v", rw.DeclaredNames)
		}
		if !strings.Contains(rw.TransformedSource, "async function") {
			return fmt.Errorf("transformed source doesn't contain async wrapper")
		}
		return nil
	})

	// ================================================================
	// T06: Static analysis report
	// ================================================================
	report.run("T06: Static analysis report", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "const z = 1; console.log(z)")
		if err != nil {
			return err
		}
		s := resp.Cell.Static
		if len(s.TopLevelBindings) != 1 {
			return fmt.Errorf("expected 1 top-level binding, got %d", len(s.TopLevelBindings))
		}
		if s.TopLevelBindings[0].Name != "z" {
			return fmt.Errorf("expected binding z, got %q", s.TopLevelBindings[0].Name)
		}
		if s.ASTNodeCount == 0 {
			return fmt.Errorf("expected AST nodes, got 0")
		}
		if s.CSTNodeCount == 0 {
			return fmt.Errorf("expected CST nodes, got 0")
		}
		if s.Scope == nil {
			return fmt.Errorf("expected scope tree")
		}
		return nil
	})

	// ================================================================
	// T07: Console capture
	// ================================================================
	report.run("T07: Console capture", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, `console.log("hello", 42); console.warn("danger"); 1`)
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok, got %s", resp.Cell.Execution.Status)
		}
		if len(resp.Cell.Execution.Console) != 2 {
			return fmt.Errorf("expected 2 console events, got %d (value: %v)", len(resp.Cell.Execution.Console), resp.Cell.Execution.Console)
		}
		if resp.Cell.Execution.Console[0].Kind != "log" {
			return fmt.Errorf("expected first event kind=log, got %q", resp.Cell.Execution.Console[0].Kind)
		}
		if resp.Cell.Execution.Console[1].Kind != "warn" {
			return fmt.Errorf("expected second event kind=warn, got %q", resp.Cell.Execution.Console[1].Kind)
		}
		return nil
	})

	// ================================================================
	// T08: Class binding tracking
	// ================================================================
	report.run("T08: Class binding with prototype chain", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, `class Point { constructor(x, y) { this.x = x; this.y = y; } toString() { return "(" + this.x + "," + this.y + ")"; } }; new Point(1, 2).toString()`)
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok, got %s", resp.Cell.Execution.Status)
		}
		if resp.Session.BindingCount != 1 {
			return fmt.Errorf("expected 1 binding, got %d", resp.Session.BindingCount)
		}
		b := resp.Session.Bindings[0]
		if b.Kind != "class" {
			return fmt.Errorf("expected binding kind=class, got %q", b.Kind)
		}
		if b.Static == nil || len(b.Static.Members) == 0 {
			return fmt.Errorf("expected class members in static view")
		}
		if b.Runtime.ValueKind != "function" {
			return fmt.Errorf("expected runtime valueKind=function (class constructor), got %q", b.Runtime.ValueKind)
		}
		return nil
	})

	// ================================================================
	// T09: Function binding with source mapping
	// ================================================================
	report.run("T09: Function binding source mapping", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, `function greet(name) { return "Hello, " + name; }; greet("World")`)
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok, got %s", resp.Cell.Execution.Status)
		}
		b := resp.Session.Bindings[0]
		if b.Kind != "function" {
			return fmt.Errorf("expected binding kind=function, got %q", b.Kind)
		}
		if b.Static == nil {
			return fmt.Errorf("expected static view")
		}
		if len(b.Static.Parameters) != 1 || b.Static.Parameters[0] != "name" {
			return fmt.Errorf("expected parameters=[name], got %v", b.Static.Parameters)
		}
		if b.Runtime.FunctionMapping == nil {
			return fmt.Errorf("expected function source mapping")
		}
		if b.Runtime.FunctionMapping.Name != "greet" {
			return fmt.Errorf("expected mapping name=greet, got %q", b.Runtime.FunctionMapping.Name)
		}
		return nil
	})

	// ================================================================
	// T10: Destructuring binding capture
	// ================================================================
	report.run("T10: Array destructuring", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, `const [p, q] = [10, 20]; p + q`)
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Result != "30" {
			return fmt.Errorf("expected 30, got %q", resp.Cell.Execution.Result)
		}
		if resp.Session.BindingCount != 2 {
			return fmt.Errorf("expected 2 bindings (p, q), got %d", resp.Session.BindingCount)
		}
		return nil
	})

	// ================================================================
	// T11: Syntax error recovery
	// ================================================================
	report.run("T11: Syntax error recovery", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "const valid = 1; valid")
		if err != nil {
			return fmt.Errorf("valid eval: %w", err)
		}
		resp, err := service.Evaluate(ctx, session.ID, "const = broken{")
		if err != nil {
			return fmt.Errorf("syntax error eval returned error: %w", err)
		}
		if resp.Cell.Execution.Status != "parse-error" {
			return fmt.Errorf("expected parse-error, got %q", resp.Cell.Execution.Status)
		}
		// Session should still be usable
		resp2, err := service.Evaluate(ctx, session.ID, "valid + 10")
		if err != nil {
			return fmt.Errorf("post-error eval: %w", err)
		}
		if resp2.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok after error recovery, got %q", resp2.Cell.Execution.Status)
		}
		if resp2.Cell.Execution.Result != "11" {
			return fmt.Errorf("expected 11, got %q", resp2.Cell.Execution.Result)
		}
		return nil
	})

	// ================================================================
	// T12: Runtime error
	// ================================================================
	report.run("T12: Runtime error", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "throw new Error('boom')")
		if err != nil {
			return fmt.Errorf("runtime error eval returned Go error: %w", err)
		}
		if resp.Cell.Execution.Status != "runtime-error" {
			return fmt.Errorf("expected runtime-error, got %q", resp.Cell.Execution.Status)
		}
		if !strings.Contains(resp.Cell.Execution.Error, "boom") {
			return fmt.Errorf("expected error to contain 'boom', got %q", resp.Cell.Execution.Error)
		}
		return nil
	})

	// ================================================================
	// T13: let re-declaration across cells
	// ================================================================
	report.run("T13: let re-declaration across cells", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "let counter = 0; counter")
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "let counter = 5; counter")
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Result != "5" {
			return fmt.Errorf("expected 5 after re-declaration, got %q", resp.Cell.Execution.Result)
		}
		// Check the binding was updated
		b := resp.Session.Bindings[0]
		if b.LastUpdatedCell != 2 {
			return fmt.Errorf("expected LastUpdatedCell=2, got %d", b.LastUpdatedCell)
		}
		return nil
	})

	// ================================================================
	// T14: Persistence round-trip
	// ================================================================
	report.run("T14: Persistence round-trip (SQLite)", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "const persistMe = 42; persistMe")
		if err != nil {
			return err
		}
		// Verify persisted in SQLite
		var count int
		err = store.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM evaluations WHERE session_id = ?`, session.ID,
		).Scan(&count)
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("expected 1 evaluation row, got %d", count)
		}
		// Verify binding version persisted
		err = store.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM binding_versions bv JOIN bindings b ON b.binding_id = bv.binding_id WHERE b.session_id = ?`,
			session.ID,
		).Scan(&count)
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("expected 1 binding version row, got %d", count)
		}
		return nil
	})

	// ================================================================
	// T15: Global diff tracking (new, update, remove)
	// ================================================================
	report.run("T15: Global diff tracking", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		// Cell 1: declare
		resp1, err := service.Evaluate(ctx, session.ID, "const g1 = 1; g1")
		if err != nil {
			return err
		}
		if len(resp1.Cell.Runtime.NewBindings) != 1 || resp1.Cell.Runtime.NewBindings[0] != "g1" {
			return fmt.Errorf("cell 1: expected newBindings=[g1], got %v", resp1.Cell.Runtime.NewBindings)
		}
		// Cell 2: update (re-declare same name)
		resp2, err := service.Evaluate(ctx, session.ID, "const g1 = 2; g1")
		if err != nil {
			return err
		}
		if len(resp2.Cell.Runtime.UpdatedBindings) != 1 || resp2.Cell.Runtime.UpdatedBindings[0] != "g1" {
			return fmt.Errorf("cell 2: expected updatedBindings=[g1], got %v", resp2.Cell.Runtime.UpdatedBindings)
		}
		return nil
	})

	// ================================================================
	// T16: Unresolved identifiers in static analysis
	// ================================================================
	report.run("T16: Unresolved identifiers in static analysis", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "unknownVar + 1")
		if err != nil {
			return err
		}
		if len(resp.Cell.Static.Unresolved) == 0 {
			return fmt.Errorf("expected unresolved identifier for unknownVar")
		}
		if resp.Cell.Static.Unresolved[0].Snippet != "unknownVar" {
			return fmt.Errorf("expected unresolved snippet=unknownVar, got %q", resp.Cell.Static.Unresolved[0].Snippet)
		}
		return nil
	})

	// ================================================================
	// T17: Timeout on infinite loop (raw mode)
	// ================================================================
	report.run("T17: Timeout on infinite loop (raw mode)", func() error {
		rawFactory, _ := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
		opts := replsession.RawSessionOptions()
		opts.Policy.Eval.TimeoutMS = 50
		svc := replsession.NewService(rawFactory, zerolog.Nop(),
			replsession.WithDefaultSessionOptions(opts),
		)
		session, err := svc.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := svc.Evaluate(ctx, session.ID, "while (true) {}")
		if err != nil {
			return fmt.Errorf("timeout eval returned Go error: %w", err)
		}
		if resp.Cell.Execution.Status != "timeout" {
			return fmt.Errorf("expected timeout, got %q", resp.Cell.Execution.Status)
		}
		if !strings.Contains(resp.Cell.Execution.Error, "timed out") {
			return fmt.Errorf("expected timeout in error, got %q", resp.Cell.Execution.Error)
		}
		// Session should be usable after timeout
		next, err := svc.Evaluate(ctx, session.ID, "1 + 1")
		if err != nil {
			return fmt.Errorf("post-timeout eval: %w", err)
		}
		if next.Cell.Execution.Result != "2" {
			return fmt.Errorf("expected 2 after timeout, got %q", next.Cell.Execution.Result)
		}
		return nil
	})

	// ================================================================
	// T18: Await expression (interactive mode)
	// ================================================================
	report.run("T18: Await expression (interactive mode)", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "await Promise.resolve(99)")
		if err != nil {
			return err
		}
		if resp.Cell.Execution.Status != "ok" {
			return fmt.Errorf("expected ok, got %q", resp.Cell.Execution.Status)
		}
		if resp.Cell.Execution.Result != "99" {
			return fmt.Errorf("expected 99, got %q", resp.Cell.Execution.Result)
		}
		if !resp.Cell.Execution.Awaited {
			return fmt.Errorf("expected Awaited=true")
		}
		return nil
	})

	// ================================================================
	// T19: Scope view in static analysis
	// ================================================================
	report.run("T19: Scope view in static analysis", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, "const a = 1; const b = 2; if (true) { const c = 3; }")
		if err != nil {
			return err
		}
		scope := resp.Cell.Static.Scope
		if scope == nil {
			return fmt.Errorf("expected scope tree")
		}
		if scope.Kind != "global" {
			return fmt.Errorf("expected root scope kind=global, got %q", scope.Kind)
		}
		if len(scope.Bindings) != 2 {
			return fmt.Errorf("expected 2 root bindings (a, b), got %d: %v", len(scope.Bindings), scope.Bindings)
		}
		if len(scope.Children) == 0 {
			return fmt.Errorf("expected at least one child scope (block)")
		}
		return nil
	})

	// ================================================================
	// T20: Delete session and verify persistence
	// ================================================================
	report.run("T20: Delete session persistence", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "const del = 1; del")
		if err != nil {
			return err
		}
		err = service.DeleteSession(ctx, session.ID)
		if err != nil {
			return err
		}
		// Verify soft-deleted in SQLite
		var deletedAt *string
		err = store.DB().QueryRowContext(ctx,
			`SELECT deleted_at FROM sessions WHERE session_id = ?`, session.ID,
		).Scan(&deletedAt)
		if err != nil {
			return err
		}
		if deletedAt == nil || *deletedAt == "" {
			return fmt.Errorf("expected deleted_at to be set")
		}
		// Verify in-memory session is gone
		_, err = service.Snapshot(ctx, session.ID)
		if err == nil {
			return fmt.Errorf("expected ErrSessionNotFound after delete")
		}
		return nil
	})

	// ================================================================
	// T21: Snapshot includes globals
	// ================================================================
	report.run("T21: Snapshot includes current globals", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		_, err = service.Evaluate(ctx, session.ID, "const glob1 = 'test'; glob1")
		if err != nil {
			return err
		}
		snap, err := service.Snapshot(ctx, session.ID)
		if err != nil {
			return err
		}
		if len(snap.CurrentGlobals) == 0 {
			return fmt.Errorf("expected CurrentGlobals to be populated")
		}
		found := false
		for _, g := range snap.CurrentGlobals {
			if g.Name == "glob1" {
				found = true
				if g.Kind != "string" {
					return fmt.Errorf("expected glob1 kind=string, got %q", g.Kind)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("glob1 not found in CurrentGlobals")
		}
		return nil
	})

	// ================================================================
	// T22: Prototype chain inspection
	// ================================================================
	report.run("T22: Prototype chain inspection", func() error {
		session, err := service.CreateSession(ctx)
		if err != nil {
			return err
		}
		resp, err := service.Evaluate(ctx, session.ID, `const obj = {a: 1, b: 2}; obj`)
		if err != nil {
			return err
		}
		if len(resp.Session.Bindings) == 0 {
			return fmt.Errorf("expected at least one binding")
		}
		b := resp.Session.Bindings[0]
		if b.Runtime.ValueKind != "object" {
			return fmt.Errorf("expected valueKind=object, got %q", b.Runtime.ValueKind)
		}
		if len(b.Runtime.OwnProperties) == 0 {
			return fmt.Errorf("expected ownProperties to be populated")
		}
		if len(b.Runtime.PrototypeChain) == 0 {
			return fmt.Errorf("expected prototypeChain to be populated")
		}
		return nil
	})

	fmt.Println()
	report.printSummary()
}

// ================================================================
// Minimal test harness
// ================================================================

type testReport struct {
	passed int
	failed int
	errors []string
}

func (r *testReport) run(name string, fn func() error) {
	fmt.Printf("  %-60s", name)
	err := fn()
	if err != nil {
		r.failed++
		r.errors = append(r.errors, name)
		fmt.Printf("FAIL\n       %v\n", err)
	} else {
		r.passed++
		fmt.Println("PASS")
	}
}

func (r *testReport) printSummary() {
	fmt.Println()
	fmt.Printf("=== Results: %d passed, %d failed ===\n", r.passed, r.failed)
	if len(r.errors) > 0 {
		fmt.Println("Failures:")
		for _, name := range r.errors {
			fmt.Printf("  - %s\n", name)
		}
	}
}
