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
	if resp.Cell.Rewrite.Mode != "raw" {
		t.Fatalf("expected rewrite mode raw, got %q", resp.Cell.Rewrite.Mode)
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
