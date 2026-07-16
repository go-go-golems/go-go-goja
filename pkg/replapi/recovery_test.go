package replapi

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func TestRecoverSessionDiscardsSuspectVMAndRestoresDurableHead(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	app, err := New(
		ctx,
		newTestFactory(t),
		zerolog.Nop(),
		WithProfile(ProfilePersistent),
		WithStore(store),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() { _ = app.Close(context.Background()) }()

	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := app.Evaluate(ctx, session.ID, `let x = 1; x`); err != nil {
		t.Fatalf("seed durable cell: %v", err)
	}

	if _, err := store.DB().ExecContext(ctx, `
		CREATE TRIGGER fail_cell_two
		BEFORE INSERT ON evaluations
		WHEN NEW.session_id = '`+session.ID+`' AND NEW.cell_id = 2
		BEGIN
			SELECT RAISE(FAIL, 'injected cell 2 failure');
		END;
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	response, err := app.Evaluate(ctx, session.ID, `x = 99; x`)
	if !errors.Is(err, replsession.ErrCommitFailed) || response == nil || response.Cell.Execution.Result != "99" {
		t.Fatalf("expected executed-but-uncommitted cell, got response=%#v err=%v", response, err)
	}

	recovered, err := app.RecoverSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if recovered.CellCount != 1 {
		t.Fatalf("expected recovery from one-cell durable head, got %d", recovered.CellCount)
	}
	var restoredX string
	if err := app.WithRuntime(ctx, session.ID, func(_ context.Context, runtime *engine.Runtime) error {
		restoredX = runtime.VM.Get("x").String()
		return nil
	}); err != nil {
		t.Fatalf("inspect recovered runtime: %v", err)
	}
	if restoredX != "1" {
		t.Fatalf("expected uncommitted x=99 to be discarded, got x=%s", restoredX)
	}

	if _, err := store.DB().ExecContext(ctx, `DROP TRIGGER fail_cell_two`); err != nil {
		t.Fatalf("drop failure trigger: %v", err)
	}
	next, err := app.Evaluate(ctx, session.ID, `x += 1; x`)
	if err != nil {
		t.Fatalf("evaluate after recovery: %v", err)
	}
	if next.Cell.ID != 2 || next.Cell.Execution.Result != "2" {
		t.Fatalf("expected replacement durable cell 2, got %#v", next.Cell)
	}
	history, err := app.History(ctx, session.ID)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 2 || history[0].CellID != 1 || history[1].CellID != 2 {
		t.Fatalf("expected contiguous durable history [1 2], got %#v", history)
	}
}
