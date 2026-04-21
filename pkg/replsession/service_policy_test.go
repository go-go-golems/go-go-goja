package replsession

import (
	"context"
	"strings"
	"testing"
	"time"

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

func TestServiceRawSessionAwaitExpressionWorksButDeclarationDoesNot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	opts := RawSessionOptions()
	opts.Policy.Eval.SupportTopLevelAwait = true
	opts.Policy.Eval.TimeoutMS = int64((250 * time.Millisecond) / time.Millisecond)

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(opts))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	expressionResp, err := service.Evaluate(ctx, session.ID, "await Promise.resolve(9)")
	if err != nil {
		t.Fatalf("evaluate await expression: %v", err)
	}
	if expressionResp.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok status for await expression, got %q", expressionResp.Cell.Execution.Status)
	}
	if expressionResp.Cell.Execution.Result != "9" {
		t.Fatalf("expected await expression result 9, got %q", expressionResp.Cell.Execution.Result)
	}
	if !expressionResp.Cell.Execution.Awaited {
		t.Fatal("expected await expression to be marked awaited")
	}

	declarationResp, err := service.Evaluate(ctx, session.ID, "const x = await Promise.resolve(3); x")
	if err != nil {
		t.Fatalf("evaluate await declaration: %v", err)
	}
	if declarationResp.Cell.Execution.Status != "runtime-error" {
		t.Fatalf("expected runtime-error for declaration-style await, got %q", declarationResp.Cell.Execution.Status)
	}
	if declarationResp.Cell.Execution.Error == "" {
		t.Fatal("expected declaration-style await to return an error message")
	}
}

func TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	opts := RawSessionOptions()
	opts.Policy.Eval.SupportTopLevelAwait = true
	opts.Policy.Eval.TimeoutMS = 20

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(opts))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "await new Promise(() => {})")
	if err != nil {
		t.Fatalf("evaluate pending await: %v", err)
	}
	if resp.Cell.Execution.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", resp.Cell.Execution.Status)
	}
	if !resp.Cell.Execution.Awaited {
		t.Fatal("expected pending await to be marked awaited")
	}
}

func TestServiceRawSyncRunawayTimeoutKeepsSessionUsable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	opts := RawSessionOptions()
	opts.Policy.Eval.TimeoutMS = 20

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(opts))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "while (true) {}")
	if err != nil {
		t.Fatalf("evaluate runaway loop: %v", err)
	}
	if resp.Cell.Execution.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", resp.Cell.Execution.Status)
	}
	if !strings.Contains(resp.Cell.Execution.Error, ErrEvaluationTimeout.Error()) {
		t.Fatalf("expected timeout error to mention %q, got %q", ErrEvaluationTimeout.Error(), resp.Cell.Execution.Error)
	}

	next, err := service.Evaluate(ctx, session.ID, "1 + 1")
	if err != nil {
		t.Fatalf("evaluate after timeout: %v", err)
	}
	if next.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok status after timeout, got %q", next.Cell.Execution.Status)
	}
	if next.Cell.Execution.Result != "2" {
		t.Fatalf("expected result 2 after timeout, got %q", next.Cell.Execution.Result)
	}
}

func TestServiceInteractiveSyncRunawayTimeoutKeepsSessionUsable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	opts := InteractiveSessionOptions()
	opts.Policy.Eval.TimeoutMS = 20

	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(opts))

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, err := service.Evaluate(ctx, session.ID, "while (true) {}")
	if err != nil {
		t.Fatalf("evaluate runaway loop: %v", err)
	}
	if resp.Cell.Execution.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", resp.Cell.Execution.Status)
	}
	if !strings.Contains(resp.Cell.Execution.Error, ErrEvaluationTimeout.Error()) {
		t.Fatalf("expected timeout error to mention %q, got %q", ErrEvaluationTimeout.Error(), resp.Cell.Execution.Error)
	}

	next, err := service.Evaluate(ctx, session.ID, "const x = 41; x + 1")
	if err != nil {
		t.Fatalf("evaluate after timeout: %v", err)
	}
	if next.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok status after timeout, got %q", next.Cell.Execution.Status)
	}
	if next.Cell.Execution.Result != "42" {
		t.Fatalf("expected result 42 after timeout, got %q", next.Cell.Execution.Result)
	}
	if next.Session.BindingCount != 1 {
		t.Fatalf("expected one tracked binding after recovery, got %d", next.Session.BindingCount)
	}
}
