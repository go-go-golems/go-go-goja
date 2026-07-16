package replapi

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

// TestHardeningPersistentSessionRejectsSecondLiveOwner is the red regression
// for GOJA-068 P0.5. The ownership mechanism selected in Phase 5 may be a
// lease or an explicitly exclusive database owner, but a second App must not
// publish another writable runtime for the same durable session.
func TestHardeningPersistentSessionRejectsSecondLiveOwner(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	factory := newTestFactory(t)

	newPersistentApp := func() *App {
		app, err := New(context.Background(), factory, zerolog.Nop(), WithProfile(ProfilePersistent), WithStore(store))
		if err != nil {
			t.Fatalf("new persistent app: %v", err)
		}
		return app
	}

	appA := newPersistentApp()
	defer func() { _ = appA.Close(context.Background()) }()
	session, err := appA.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := appA.Evaluate(ctx, session.ID, `const owner = "seed"; owner`); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	appB := newPersistentApp()
	defer func() { _ = appB.Close(context.Background()) }()
	if _, err := appB.Snapshot(ctx, session.ID); err == nil {
		_ = appB.DeleteSession(context.Background(), session.ID)
		_ = appA.DeleteSession(context.Background(), session.ID)
		t.Fatal("expected second App to be rejected before publishing a writable runtime")
	}

	_ = appA.DeleteSession(context.Background(), session.ID)
}
