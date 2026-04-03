package replsession

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestServiceRawSessionUsesDirectExecution(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(RawSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.Profile != "raw" {
		t.Fatalf("expected raw profile, got %q", session.Profile)
	}

	resp, err := service.Evaluate(ctx, session.ID, "const x = 1; x")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Rewrite.Mode != "raw" {
		t.Fatalf("expected raw rewrite mode, got %q", resp.Cell.Rewrite.Mode)
	}
	if resp.Cell.Execution.Result != "1" {
		t.Fatalf("expected result 1, got %q", resp.Cell.Execution.Result)
	}
	if resp.Session.BindingCount != 0 {
		t.Fatalf("expected raw mode to avoid binding tracking, got %d", resp.Session.BindingCount)
	}
}

func TestServiceInteractiveSessionTracksBindingsWithoutPersistence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "const x = 1; x")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Rewrite.Mode != "async-iife-with-binding-capture" {
		t.Fatalf("expected instrumented rewrite mode, got %q", resp.Cell.Rewrite.Mode)
	}
	if resp.Session.BindingCount != 1 {
		t.Fatalf("expected one tracked binding, got %d", resp.Session.BindingCount)
	}
	if resp.Session.Profile != "interactive" {
		t.Fatalf("expected interactive profile, got %q", resp.Session.Profile)
	}
}
