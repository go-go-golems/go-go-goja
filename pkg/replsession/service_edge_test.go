package replsession

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestServiceEmptySourceReturnsGracefully(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(openPersistenceTestStore(t)))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "")
	if err != nil {
		t.Fatalf("evaluate empty source: %v", err)
	}
	if resp.Cell.Execution.Status != "empty-source" {
		t.Fatalf("expected status empty-source, got %q", resp.Cell.Execution.Status)
	}
	if resp.Cell.Execution.Result != "undefined" {
		t.Fatalf("expected result undefined, got %q", resp.Cell.Execution.Result)
	}
	if resp.Cell.Rewrite.Mode != "none" {
		t.Fatalf("expected rewrite mode none, got %q", resp.Cell.Rewrite.Mode)
	}
}

func TestServiceWhitespaceSourceReturnsGracefully(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(openPersistenceTestStore(t)))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	for _, src := range []string{"   ", "\t\n", "  \n  "} {
		resp, err := service.Evaluate(ctx, session.ID, src)
		if err != nil {
			t.Fatalf("evaluate whitespace source %q: %v", src, err)
		}
		if resp.Cell.Execution.Status != "empty-source" {
			t.Fatalf("source %q: expected status empty-source, got %q", src, resp.Cell.Execution.Status)
		}
	}
}

func TestServiceSessionUsableAfterEmptySource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(openPersistenceTestStore(t)))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Empty source should not corrupt the session
	_, err = service.Evaluate(ctx, session.ID, "")
	if err != nil {
		t.Fatalf("evaluate empty: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "const x = 1; x")
	if err != nil {
		t.Fatalf("evaluate after empty: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok after empty-source, got %q", resp.Cell.Execution.Status)
	}
	if resp.Cell.Execution.Result != "1" {
		t.Fatalf("expected result 1, got %q", resp.Cell.Execution.Result)
	}
	if resp.Session.BindingCount != 1 {
		t.Fatalf("expected 1 binding, got %d", resp.Session.BindingCount)
	}
}

func TestServiceEmptySourceDoesNotPanicWithInteractiveOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Interactive mode triggers jsparse.Analyze which is where the panic was
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// This used to panic — verify it no longer does
	resp, err := service.Evaluate(ctx, session.ID, "")
	if err != nil {
		t.Fatalf("evaluate empty source in interactive mode: %v", err)
	}
	if resp.Cell.Execution.Status != "empty-source" {
		t.Fatalf("expected empty-source, got %q", resp.Cell.Execution.Status)
	}

	// Verify whitespace too
	resp2, err := service.Evaluate(ctx, session.ID, "   ")
	if err != nil {
		t.Fatalf("evaluate whitespace in interactive mode: %v", err)
	}
	if resp2.Cell.Execution.Status != "empty-source" {
		t.Fatalf("expected empty-source for whitespace, got %q", resp2.Cell.Execution.Status)
	}

	// Session still works
	resp3, err := service.Evaluate(ctx, session.ID, "1 + 1")
	if err != nil {
		t.Fatalf("evaluate after empties: %v", err)
	}
	if resp3.Cell.Execution.Result != "2" {
		t.Fatalf("expected 2, got %q", resp3.Cell.Execution.Result)
	}
}

func TestServiceThrowNewErrorPreservesMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "throw new Error('boom')")
	if err != nil {
		t.Fatalf("evaluate throw: %v", err)
	}
	if resp.Cell.Execution.Status != "runtime-error" {
		t.Fatalf("expected runtime-error, got %q", resp.Cell.Execution.Status)
	}
	if !strings.Contains(resp.Cell.Execution.Error, "boom") {
		t.Fatalf("expected error to contain 'boom', got %q", resp.Cell.Execution.Error)
	}
}

func TestServiceThrowStringPreservesMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "throw 'simple string'")
	if err != nil {
		t.Fatalf("evaluate throw string: %v", err)
	}
	if resp.Cell.Execution.Status != "runtime-error" {
		t.Fatalf("expected runtime-error, got %q", resp.Cell.Execution.Status)
	}
	if !strings.Contains(resp.Cell.Execution.Error, "simple string") {
		t.Fatalf("expected error to contain 'simple string', got %q", resp.Cell.Execution.Error)
	}
}

func TestServiceInstrumentedAwaitExpression(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Interactive mode uses instrumented execution with top-level await support
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "await Promise.resolve(99)")
	if err != nil {
		t.Fatalf("evaluate await: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok, got %q: %s", resp.Cell.Execution.Status, resp.Cell.Execution.Error)
	}
	if resp.Cell.Execution.Result != "99" {
		t.Fatalf("expected 99, got %q", resp.Cell.Execution.Result)
	}
	if !resp.Cell.Execution.Awaited {
		t.Fatal("expected Awaited=true")
	}
}

func TestServiceInstrumentedAwaitChained(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// First cell: declare a value normally
	_, err = service.Evaluate(ctx, session.ID, "const a = 10")
	if err != nil {
		t.Fatalf("evaluate const: %v", err)
	}

	// Second cell: use await with a reference to the binding
	resp, err := service.Evaluate(ctx, session.ID, "await Promise.resolve(a + 5)")
	if err != nil {
		t.Fatalf("evaluate await: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok, got %q: %s", resp.Cell.Execution.Status, resp.Cell.Execution.Error)
	}
	if resp.Cell.Execution.Result != "15" {
		t.Fatalf("expected 15, got %q", resp.Cell.Execution.Result)
	}
}

func TestServiceFunctionSourceMappingIsPopulated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "function greet(name) { return 'Hello, ' + name; }; greet('World')")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok, got %q", resp.Cell.Execution.Status)
	}
	if resp.Session.BindingCount != 1 {
		t.Fatalf("expected 1 binding, got %d", resp.Session.BindingCount)
	}
	b := resp.Session.Bindings[0]
	if b.Kind != "function" {
		t.Fatalf("expected binding kind=function, got %q", b.Kind)
	}
	if b.Runtime.FunctionMapping == nil {
		t.Fatal("expected FunctionMapping to be populated, got nil")
	}
	if b.Runtime.FunctionMapping.Name != "greet" {
		t.Fatalf("expected mapping name=greet, got %q", b.Runtime.FunctionMapping.Name)
	}
}
