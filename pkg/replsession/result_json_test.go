package replsession

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestServiceRawResultJSONForUndefinedDeclarationResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(RawSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `let x = 1`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Result != "undefined" {
		t.Fatalf("expected undefined result, got %q", resp.Cell.Execution.Result)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"kind":"undefined"`) {
		t.Fatalf("expected undefined metadata envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"result":null`) {
		t.Fatalf("expected null result placeholder, got %q", resp.Cell.Execution.ResultJSON)
	}
}

func TestServiceRawResultJSONForPromisePreviewWithoutAwait(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(RawSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `Promise.resolve(7)`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.HasPrefix(resp.Cell.Execution.Result, "Promise {") {
		t.Fatalf("expected promise preview, got %q", resp.Cell.Execution.Result)
	}
	if resp.Cell.Execution.ResultJSON == "" {
		t.Fatal("expected non-empty result JSON for promise preview")
	}
}

func TestServiceInstrumentedResultJSONForFinalExpression(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "const x = 40; x + 2")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok, got %q: %s", resp.Cell.Execution.Status, resp.Cell.Execution.Error)
	}
	if resp.Cell.Execution.ResultJSON != `{"result":42}` {
		t.Fatalf("expected result JSON envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
}

func TestServiceInstrumentedResultJSONForObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `const obj = { slug: "a", count: 2 }; obj`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.ResultJSON != `{"result":{"slug":"a","count":2}}` {
		t.Fatalf("expected object result JSON envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
}

func TestServiceInstrumentedResultJSONForFunctionValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `function f() { return 1; } f`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"kind":"function"`) {
		t.Fatalf("expected function metadata envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"result":null`) {
		t.Fatalf("expected null result placeholder, got %q", resp.Cell.Execution.ResultJSON)
	}
}

func TestServiceInstrumentedResultJSONForUndefinedValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `undefined`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"kind":"undefined"`) {
		t.Fatalf("expected undefined metadata envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, `"result":null`) {
		t.Fatalf("expected null result placeholder, got %q", resp.Cell.Execution.ResultJSON)
	}
}

func TestServiceInstrumentedResultJSONReportsSerializationError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, `const x = {}; x.self = x; x`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.Contains(resp.Cell.Execution.ResultJSON, "error") || !strings.Contains(resp.Cell.Execution.ResultJSON, "JSON") {
		t.Fatalf("expected serialization error envelope, got %q", resp.Cell.Execution.ResultJSON)
	}
}
