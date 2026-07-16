package repldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSessionLeaseAcquireRenewTakeoverReleaseAndEpochs(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, time.July, 15, 21, 0, 0, 0, time.UTC)
	createLeaseTestSession(t, store, now)

	leaseA, err := store.AcquireSessionLease(ctx, "lease-session", "owner-a", now, time.Minute)
	if err != nil {
		t.Fatalf("acquire absent lease: %v", err)
	}
	if leaseA.Epoch != 1 || leaseA.OwnerID != "owner-a" {
		t.Fatalf("unexpected first lease: %#v", leaseA)
	}
	if _, err := store.AcquireSessionLease(ctx, "lease-session", "owner-b", now.Add(10*time.Second), time.Minute); !errors.Is(err, ErrSessionOwned) {
		t.Fatalf("expected active-owner conflict, got %v", err)
	}

	renewedA, err := store.AcquireSessionLease(ctx, "lease-session", "owner-a", now.Add(20*time.Second), time.Minute)
	if err != nil {
		t.Fatalf("same-owner acquire: %v", err)
	}
	if renewedA.Epoch != leaseA.Epoch || !renewedA.LeaseUntil.After(leaseA.LeaseUntil) {
		t.Fatalf("same owner should renew without epoch change: before=%#v after=%#v", leaseA, renewedA)
	}

	takeoverAt := renewedA.LeaseUntil.Add(time.Nanosecond)
	leaseB, err := store.AcquireSessionLease(ctx, "lease-session", "owner-b", takeoverAt, time.Minute)
	if err != nil {
		t.Fatalf("take over expired lease: %v", err)
	}
	if leaseB.Epoch != 2 || leaseB.OwnerID != "owner-b" {
		t.Fatalf("expected owner-b epoch 2, got %#v", leaseB)
	}
	if _, err := store.RenewSessionLease(ctx, renewedA, takeoverAt, time.Minute); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected stale renewal rejection, got %v", err)
	}
	if err := store.ReleaseSessionLease(ctx, renewedA); err != nil {
		t.Fatalf("stale release should be harmless: %v", err)
	}
	if err := store.ReleaseSessionLease(ctx, leaseB); err != nil {
		t.Fatalf("release current lease: %v", err)
	}
	leaseC, err := store.AcquireSessionLease(ctx, "lease-session", "owner-c", takeoverAt.Add(time.Second), time.Minute)
	if err != nil {
		t.Fatalf("acquire released lease: %v", err)
	}
	if leaseC.Epoch != 3 {
		t.Fatalf("release must preserve monotonic epoch, got %#v", leaseC)
	}
}

func TestSimultaneousLeaseAcquireHasExactlyOneOwner(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, time.July, 15, 21, 30, 0, 0, time.UTC)
	createLeaseTestSession(t, store, now)

	const contenders = 8
	start := make(chan struct{})
	results := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		ownerID := fmt.Sprintf("owner-%d", i)
		go func() {
			<-start
			_, err := store.AcquireSessionLease(ctx, "lease-session", ownerID, now, time.Minute)
			results <- err
		}()
	}
	close(start)
	successes := 0
	conflicts := 0
	for i := 0; i < contenders; i++ {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrSessionOwned):
			conflicts++
		default:
			t.Fatalf("unexpected acquire result: %v", err)
		}
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("successes=%d conflicts=%d, want 1/%d", successes, conflicts, contenders-1)
	}
}

func TestPersistEvaluationFencedRejectsExpiryStaleEpochAndWrongHead(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, time.July, 15, 22, 0, 0, 0, time.UTC)
	createLeaseTestSession(t, store, now)

	leaseA, err := store.AcquireSessionLease(ctx, "lease-session", "owner-a", now, time.Minute)
	if err != nil {
		t.Fatalf("acquire owner-a: %v", err)
	}
	record1 := leaseTestEvaluation(1, now.Add(time.Second))
	fenceA1 := WriteFence{OwnerID: leaseA.OwnerID, Epoch: leaseA.Epoch, ExpectedCellID: 1}
	if err := store.PersistEvaluationFenced(ctx, record1, fenceA1, now.Add(time.Second)); err != nil {
		t.Fatalf("fenced cell 1: %v", err)
	}
	if err := store.PersistEvaluationFenced(ctx, record1, fenceA1, now.Add(2*time.Second)); !errors.Is(err, ErrWriteConflict) {
		t.Fatalf("expected durable-head conflict, got %v", err)
	}
	if err := store.PersistEvaluationFenced(ctx, leaseTestEvaluation(2, now.Add(2*time.Minute)), WriteFence{OwnerID: leaseA.OwnerID, Epoch: leaseA.Epoch, ExpectedCellID: 2}, now.Add(2*time.Minute)); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected expired fence rejection, got %v", err)
	}

	leaseB, err := store.AcquireSessionLease(ctx, "lease-session", "owner-b", now.Add(2*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("owner-b takeover: %v", err)
	}
	if err := store.PersistEvaluationFenced(ctx, leaseTestEvaluation(2, now.Add(2*time.Minute)), WriteFence{OwnerID: leaseA.OwnerID, Epoch: leaseA.Epoch, ExpectedCellID: 2}, now.Add(2*time.Minute)); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected stale epoch rejection, got %v", err)
	}
	if err := store.PersistEvaluationFenced(ctx, leaseTestEvaluation(2, now.Add(2*time.Minute)), WriteFence{OwnerID: leaseB.OwnerID, Epoch: leaseB.Epoch, ExpectedCellID: 2}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("owner-b fenced cell 2: %v", err)
	}

	evaluations, err := store.LoadEvaluations(ctx, "lease-session")
	if err != nil {
		t.Fatalf("load evaluations: %v", err)
	}
	if len(evaluations) != 2 || evaluations[0].CellID != 1 || evaluations[1].CellID != 2 {
		t.Fatalf("unexpected fenced history: %#v", evaluations)
	}
}

func createLeaseTestSession(t *testing.T, store *Store, now time.Time) {
	t.Helper()
	if err := store.CreateSession(context.Background(), SessionRecord{SessionID: "lease-session", CreatedAt: now, UpdatedAt: now, EngineKind: "goja"}); err != nil {
		t.Fatalf("create lease test session: %v", err)
	}
}

func leaseTestEvaluation(cellID int, createdAt time.Time) EvaluationRecord {
	return EvaluationRecord{
		SessionID:         "lease-session",
		CellID:            cellID,
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
