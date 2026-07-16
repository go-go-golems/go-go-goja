package replsession

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestCommitFailureReturnsCellAndExactRetryDoesNotRerunJavaScript(t *testing.T) {
	ctx := context.Background()
	store := &retryPersistence{failCell: 2, failuresRemaining: 2}
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithPersistence(store))
	defer func() { _ = service.Close(context.Background()) }()

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Evaluate(ctx, session.ID, `let x = 1; x`); err != nil {
		t.Fatalf("cell 1: %v", err)
	}

	failedResponse, err := service.Evaluate(ctx, session.ID, `x += 1; x`)
	if !errors.Is(err, ErrCommitFailed) {
		t.Fatalf("expected ErrCommitFailed, got %v", err)
	}
	if failedResponse == nil || failedResponse.Cell == nil || failedResponse.Cell.ID != 2 || failedResponse.Cell.Execution.Result != "2" {
		t.Fatalf("expected executed cell 2 response, got %#v", failedResponse)
	}
	var commitErr *CommitError
	if !errors.As(err, &commitErr) || commitErr.SessionID != session.ID || commitErr.CellID != 2 || commitErr.PersistenceCause() == nil {
		t.Fatalf("unexpected typed commit error: %#v", err)
	}
	if failedResponse.Session.CellCount != 1 {
		t.Fatalf("uncommitted cell must not be published in history, got %d", failedResponse.Session.CellCount)
	}

	health, err := service.SessionHealth(ctx, session.ID)
	if err != nil || health != SessionHealthDegraded {
		t.Fatalf("expected degraded health, got health=%q err=%v", health, err)
	}
	if response, err := service.Evaluate(ctx, session.ID, `x += 100; x`); !errors.Is(err, ErrSessionDegraded) || response != nil {
		t.Fatalf("expected pre-execution degraded rejection, got response=%#v err=%v", response, err)
	}

	retryResponse, err := service.RetryPendingCommit(ctx, session.ID)
	if !errors.Is(err, ErrCommitFailed) || retryResponse == nil || retryResponse.Cell.ID != 2 {
		t.Fatalf("expected first exact retry to fail with cell response, got response=%#v err=%v", retryResponse, err)
	}
	if len(store.attempts) != 3 || !reflect.DeepEqual(store.attempts[1], store.attempts[2]) {
		t.Fatalf("retry changed retained record\nfirst: %#v\nretry: %#v", store.attempts[1], store.attempts[2])
	}

	retryResponse, err = service.RetryPendingCommit(ctx, session.ID)
	if err != nil {
		t.Fatalf("successful exact retry: %v", err)
	}
	if retryResponse.Session.CellCount != 2 || retryResponse.Cell.ID != 2 {
		t.Fatalf("expected committed cell 2 publication, got %#v", retryResponse)
	}
	if len(store.attempts) != 4 || !reflect.DeepEqual(store.attempts[1], store.attempts[3]) {
		t.Fatal("successful retry did not reuse the exact original record")
	}

	var runtimeX string
	if err := service.WithRuntime(ctx, session.ID, func(_ context.Context, runtime *engine.Runtime) error {
		runtimeX = runtime.VM.Get("x").String()
		return nil
	}); err != nil {
		t.Fatalf("inspect runtime: %v", err)
	}
	if runtimeX != "2" {
		t.Fatalf("retry reran JavaScript; expected x=2, got %s", runtimeX)
	}

	third, err := service.Evaluate(ctx, session.ID, `x += 1; x`)
	if err != nil {
		t.Fatalf("cell 3 after retry: %v", err)
	}
	if third.Cell.ID != 3 || third.Cell.Execution.Result != "3" {
		t.Fatalf("expected contiguous cell 3 result, got %#v", third.Cell)
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(store.persisted, want) {
		t.Fatalf("expected durable IDs %v, got %v", want, store.persisted)
	}
}

func TestFencedSessionRejectsJavaScriptBeforeExecution(t *testing.T) {
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop())
	defer func() { _ = service.Close(context.Background()) }()
	session, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	state, err := service.getSession(session.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	op, err := state.beginOperation(context.Background())
	if err != nil {
		t.Fatalf("begin operation: %v", err)
	}
	state.health = SessionHealthFenced
	state.healthCause = errors.New("test fence loss")
	op.Release()

	response, err := service.Evaluate(context.Background(), session.ID, `globalThis.mustNotRun = true`)
	if !errors.Is(err, ErrSessionFenced) || response != nil {
		t.Fatalf("expected fenced rejection, got response=%#v err=%v", response, err)
	}
}

type retryPersistence struct {
	failCell          int
	failuresRemaining int
	attempts          []repldb.EvaluationRecord
	persisted         []int
}

func (*retryPersistence) CreateSession(context.Context, repldb.SessionRecord) error { return nil }
func (*retryPersistence) DeleteSession(context.Context, string, time.Time) error    { return nil }

func (p *retryPersistence) PersistEvaluation(_ context.Context, record repldb.EvaluationRecord) error {
	p.attempts = append(p.attempts, record)
	if record.CellID == p.failCell && p.failuresRemaining > 0 {
		p.failuresRemaining--
		return errors.New("injected append failure")
	}
	p.persisted = append(p.persisted, record.CellID)
	return nil
}
