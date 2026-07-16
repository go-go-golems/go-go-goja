package replapi

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func TestAppsReceiveDistinctDefaultOwnerIDs(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	factory := newTestFactory(t)
	newDefaultApp := func() *App {
		app, err := New(ctx, factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
		if err != nil {
			t.Fatalf("new app: %v", err)
		}
		return app
	}
	appA := newDefaultApp()
	defer func() { _ = appA.Close(context.Background()) }()
	appB := newDefaultApp()
	defer func() { _ = appB.Close(context.Background()) }()
	sessionA, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create appA session: %v", err)
	}
	sessionB, err := appB.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create appB session: %v", err)
	}
	var ownerA, ownerB string
	if err := store.DB().QueryRow(`SELECT owner_id FROM session_leases WHERE session_id = ?`, sessionA.ID).Scan(&ownerA); err != nil {
		t.Fatalf("load appA owner: %v", err)
	}
	if err := store.DB().QueryRow(`SELECT owner_id FROM session_leases WHERE session_id = ?`, sessionB.ID).Scan(&ownerB); err != nil {
		t.Fatalf("load appB owner: %v", err)
	}
	if ownerA == "" || ownerB == "" || ownerA == ownerB {
		t.Fatalf("expected distinct non-empty owner IDs, got %q and %q", ownerA, ownerB)
	}
	if ownerA != appA.OwnerID() || ownerB != appB.OwnerID() {
		t.Fatalf("diagnostic owner IDs disagree with durable leases: app=%q/%q db=%q/%q", appA.OwnerID(), appB.OwnerID(), ownerA, ownerB)
	}
}

func TestStoreOnlyDeleteCannotBypassAnotherLiveOwner(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	factory := newTestFactory(t)
	appA, err := New(ctx, factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new appA: %v", err)
	}
	defer func() { _ = appA.Close(context.Background()) }()
	session, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := appA.UnloadSession(ctx, session.ID); err != nil {
		t.Fatalf("unload appA session: %v", err)
	}

	appB, err := New(ctx, factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new appB: %v", err)
	}
	if _, err := appB.Restore(ctx, session.ID); err != nil {
		t.Fatalf("restore appB: %v", err)
	}
	if err := appA.DeleteSession(ctx, session.ID); !errors.Is(err, repldb.ErrSessionOwned) {
		t.Fatalf("expected delete ownership rejection, got %v", err)
	}
	if _, err := store.LoadSession(ctx, session.ID); err != nil {
		t.Fatalf("rejected delete hid session: %v", err)
	}
	if err := appB.Close(ctx); err != nil {
		t.Fatalf("close appB: %v", err)
	}
	if err := appA.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("delete after owner release: %v", err)
	}
}

func TestDeleteSoftDeletesSessionAndReleasesLease(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	app, err := New(ctx, newTestFactory(t), zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := app.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	defer func() { _ = app.Close(context.Background()) }()

	var deletedAt, leaseUntil string
	if err := store.DB().QueryRow(`
		SELECT s.deleted_at, l.lease_until
		FROM sessions s JOIN session_leases l USING(session_id)
		WHERE s.session_id = ?
	`, session.ID).Scan(&deletedAt, &leaseUntil); err != nil {
		t.Fatalf("load deleted ownership rows: %v", err)
	}
	if deletedAt == "" || leaseUntil != time.Unix(0, 0).UTC().Format(time.RFC3339Nano) {
		t.Fatalf("expected soft delete plus expired lease, deleted_at=%q lease_until=%q", deletedAt, leaseUntil)
	}
	if _, err := app.Restore(ctx, session.ID); !errors.Is(err, repldb.ErrSessionNotFound) {
		t.Fatalf("deleted session must remain hidden, got %v", err)
	}
}

func TestPersistentLeaseRejectsSecondOwnerAndFencesExpiredOwnerAfterTakeover(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	clock := newFakeClock(time.Date(2026, time.July, 15, 23, 0, 0, 0, time.UTC))
	factory := newTestFactory(t)
	newApp := func(owner string) *App {
		app, err := New(ctx, factory, zerolog.Nop(),
			WithProfile(ProfilePersistent),
			WithStore(store),
			withOwnerIDForTest(owner),
			WithClock(clock),
			WithLeaseTTL(time.Minute),
		)
		if err != nil {
			t.Fatalf("new app %s: %v", owner, err)
		}
		return app
	}

	appA := newApp("owner-a")
	defer func() { _ = appA.Close(context.Background()) }()
	session, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := appA.Evaluate(ctx, session.ID, `let x = 1; x`); err != nil {
		t.Fatalf("seed cell: %v", err)
	}

	appB := newApp("owner-b")
	defer func() { _ = appB.Close(context.Background()) }()
	if _, err := appB.Snapshot(ctx, session.ID); !errors.Is(err, repldb.ErrSessionOwned) {
		t.Fatalf("expected active owner rejection, got %v", err)
	}

	clock.Advance(2 * time.Minute)
	restored, err := appB.Snapshot(ctx, session.ID)
	if err != nil {
		t.Fatalf("take over expired lease: %v", err)
	}
	if restored.CellCount != 1 {
		t.Fatalf("expected one restored cell, got %d", restored.CellCount)
	}

	if response, err := appA.Evaluate(ctx, session.ID, `x = 99; x`); !errors.Is(err, replsession.ErrSessionFenced) || response != nil {
		t.Fatalf("expected stale owner fenced before JavaScript, got response=%#v err=%v", response, err)
	}
	health, err := appA.SessionHealth(ctx, session.ID)
	if err != nil || health != replsession.SessionHealthFenced {
		t.Fatalf("expected fenced appA health, got health=%q err=%v", health, err)
	}
	history, err := appB.History(ctx, session.ID)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("stale owner changed durable history: %#v", history)
	}

	if err := appB.Close(ctx); err != nil {
		t.Fatalf("close takeover owner: %v", err)
	}
	recovered, err := appA.RecoverSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("recover fenced owner after release: %v", err)
	}
	if recovered.CellCount != 1 {
		t.Fatalf("expected recovery from durable head, got %d cells", recovered.CellCount)
	}
	next, err := appA.Evaluate(ctx, session.ID, `x += 1; x`)
	if err != nil {
		t.Fatalf("evaluate after ownership recovery: %v", err)
	}
	if next.Cell.ID != 2 || next.Cell.Execution.Result != "2" {
		t.Fatalf("unexpected recovered continuation: %#v", next.Cell)
	}
}

func TestStaleLiveOwnerCannotDeleteSessionAfterLeaseTakeover(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	clock := newFakeClock(time.Date(2026, time.July, 16, 11, 0, 0, 0, time.UTC))
	factory := newTestFactory(t)
	newApp := func(owner string) *App {
		app, err := New(ctx, factory, zerolog.Nop(),
			WithProfile(ProfilePersistent),
			WithStore(store),
			withOwnerIDForTest(owner),
			WithClock(clock),
			WithLeaseTTL(time.Minute),
		)
		if err != nil {
			t.Fatalf("new app %s: %v", owner, err)
		}
		return app
	}

	appA := newApp("delete-owner-a")
	defer func() { _ = appA.Close(context.Background()) }()
	session, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := appA.Evaluate(ctx, session.ID, `const durable = 1; durable`); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	clock.Advance(2 * time.Minute)
	appB := newApp("delete-owner-b")
	defer func() { _ = appB.Close(context.Background()) }()
	if _, err := appB.Snapshot(ctx, session.ID); err != nil {
		t.Fatalf("take over expired session: %v", err)
	}

	if err := appA.DeleteSession(ctx, session.ID); !errors.Is(err, replsession.ErrSessionFenced) {
		t.Fatalf("expected stale delete to be fenced, got %v", err)
	}
	if _, err := store.LoadSession(ctx, session.ID); err != nil {
		t.Fatalf("stale owner soft-deleted current owner's session: %v", err)
	}
	if snapshot, err := appB.Snapshot(ctx, session.ID); err != nil || snapshot.CellCount != 1 {
		t.Fatalf("current owner lost session after stale delete: snapshot=%#v err=%v", snapshot, err)
	}
}

func TestRestoreHeartbeatPreventsTakeoverDuringLongReplay(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	factory := newTestFactory(t)
	newOwnedApp := func(owner string) *App {
		app, err := New(ctx, factory, zerolog.Nop(),
			WithProfile(ProfilePersistent),
			WithStore(store),
			withOwnerIDForTest(owner),
			WithLeaseTTL(90*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("new app %s: %v", owner, err)
		}
		return app
	}

	appA := newOwnedApp("replay-owner-a")
	session, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := appA.Evaluate(ctx, session.ID, `const replayStart = Date.now(); while (Date.now() - replayStart < 350) {} replayStart`); err != nil {
		t.Fatalf("seed slow replay cell: %v", err)
	}
	if err := appA.Close(ctx); err != nil {
		t.Fatalf("close seed owner: %v", err)
	}

	appB := newOwnedApp("replay-owner-b")
	defer func() { _ = appB.Close(context.Background()) }()
	restoreDone := make(chan error, 1)
	go func() {
		_, err := appB.Snapshot(ctx, session.ID)
		restoreDone <- err
	}()
	waitForLeaseOwner(t, store, session.ID, "replay-owner-b")
	time.Sleep(180 * time.Millisecond) // Twice the TTL while replay is still busy.

	appC := newOwnedApp("replay-owner-c")
	defer func() { _ = appC.Close(context.Background()) }()
	if _, err := appC.Snapshot(ctx, session.ID); !errors.Is(err, repldb.ErrSessionOwned) {
		t.Fatalf("expected replay heartbeat to retain ownership, got %v", err)
	}
	if err := <-restoreDone; err != nil {
		t.Fatalf("slow restore: %v", err)
	}
}

func waitForLeaseOwner(t *testing.T, store *repldb.Store, sessionID string, ownerID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var got string
		err := store.DB().QueryRow(`SELECT owner_id FROM session_leases WHERE session_id = ?`, sessionID).Scan(&got)
		if err == nil && got == ownerID {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("lease owner did not become %q; last owner=%q err=%v", ownerID, got, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestLeaseHeartbeatCoversEvaluationLongerThanTTL(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	app, err := New(ctx, newTestFactory(t), zerolog.Nop(),
		WithProfile(ProfilePersistent),
		WithStore(store),
		withOwnerIDForTest("heartbeat-owner"),
		WithLeaseTTL(90*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() { _ = app.Close(context.Background()) }()
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	response, err := app.Evaluate(ctx, session.ID, `const startedAt = Date.now(); while (Date.now() - startedAt < 240) {} 42`)
	if err != nil {
		t.Fatalf("long evaluation should retain lease: %v", err)
	}
	if response.Cell.Execution.Status != "ok" || response.Cell.ID != 1 {
		t.Fatalf("unexpected long evaluation response: %#v", response.Cell)
	}
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock { return &fakeClock{now: now.UTC()} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(delta time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(delta)
	c.mu.Unlock()
}
