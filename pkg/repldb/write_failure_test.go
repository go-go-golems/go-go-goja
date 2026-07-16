package repldb

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestPersistEvaluationRollsBackInsertChildAndCommitFailures(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	createdAt := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	if err := store.CreateSession(ctx, SessionRecord{
		SessionID:  "failure-session",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
		EngineKind: "goja",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	record := failureTestEvaluation(createdAt)
	record.BindingVersions = []BindingVersionRecord{{Name: "", CellID: 1, CreatedAt: createdAt}}
	if err := store.PersistEvaluation(ctx, record); err == nil {
		t.Fatal("expected child-row failure")
	}
	assertEvaluationCount(t, store, 0)
	assertBindingCount(t, store, 0)

	record.BindingVersions = nil
	store.beforeEvaluationCommit = func() error { return errors.New("injected commit failure") }
	if err := store.PersistEvaluation(ctx, record); err == nil {
		t.Fatal("expected commit-stage failure")
	}
	assertEvaluationCount(t, store, 0)
	assertBindingCount(t, store, 0)

	store.beforeEvaluationCommit = nil
	if err := store.PersistEvaluation(ctx, record); err != nil {
		t.Fatalf("persist after rollback: %v", err)
	}
	assertEvaluationCount(t, store, 1)

	if err := store.PersistEvaluation(ctx, record); err == nil {
		t.Fatal("expected duplicate evaluation insert failure")
	}
	assertEvaluationCount(t, store, 1)
}

func failureTestEvaluation(createdAt time.Time) EvaluationRecord {
	return EvaluationRecord{
		SessionID:         "failure-session",
		CellID:            1,
		CreatedAt:         createdAt,
		RawSource:         "1 + 1",
		RewrittenSource:   "1 + 1",
		OK:                true,
		ResultJSON:        json.RawMessage(`{"result":2}`),
		AnalysisJSON:      json.RawMessage(`{}`),
		GlobalsBeforeJSON: json.RawMessage(`[]`),
		GlobalsAfterJSON:  json.RawMessage(`[]`),
	}
}

func assertEvaluationCount(t *testing.T, store *Store, want int) {
	t.Helper()
	var got int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM evaluations`).Scan(&got); err != nil {
		t.Fatalf("count evaluations: %v", err)
	}
	if got != want {
		t.Fatalf("evaluation count=%d, want %d", got, want)
	}
}

func assertBindingCount(t *testing.T, store *Store, want int) {
	t.Helper()
	var got int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM bindings`).Scan(&got); err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if got != want {
		t.Fatalf("binding count=%d, want %d", got, want)
	}
}
