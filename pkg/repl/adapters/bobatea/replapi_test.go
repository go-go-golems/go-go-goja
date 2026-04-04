package bobatea

import (
	"context"
	"path/filepath"
	"testing"

	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestREPLAPIAdapterEvaluateStreamResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := newAdapterTestApp(t, replapi.ProfileInteractive, nil)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	adapter, err := NewREPLAPIAdapter(app, session.ID)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	var events []bobarepl.Event
	err = adapter.EvaluateStream(ctx, "2 + 3", func(ev bobarepl.Event) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("evaluate stream: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != bobarepl.EventResultMarkdown {
		t.Fatalf("expected result markdown event, got %q", events[0].Kind)
	}
	if got := events[0].Props["markdown"]; got != "5" {
		t.Fatalf("expected markdown result 5, got %#v", got)
	}
}

func TestREPLAPIAdapterEvaluateStreamConsoleAndError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := newAdapterTestApp(t, replapi.ProfileInteractive, nil)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	adapter, err := NewREPLAPIAdapter(app, session.ID)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	var consoleEvents []bobarepl.Event
	err = adapter.EvaluateStream(ctx, "console.log('hello'); 7", func(ev bobarepl.Event) {
		consoleEvents = append(consoleEvents, ev)
	})
	if err != nil {
		t.Fatalf("evaluate stream console: %v", err)
	}
	if len(consoleEvents) != 2 {
		t.Fatalf("expected 2 events, got %d", len(consoleEvents))
	}
	if consoleEvents[0].Kind != bobarepl.EventStdout {
		t.Fatalf("expected stdout event, got %q", consoleEvents[0].Kind)
	}
	if got := consoleEvents[0].Props["text"]; got != "\"hello\"" {
		t.Fatalf("expected stdout text %q, got %#v", "\"hello\"", got)
	}
	if consoleEvents[1].Kind != bobarepl.EventResultMarkdown {
		t.Fatalf("expected result markdown event, got %q", consoleEvents[1].Kind)
	}

	var errorEvents []bobarepl.Event
	err = adapter.EvaluateStream(ctx, "throw new Error('boom')", func(ev bobarepl.Event) {
		errorEvents = append(errorEvents, ev)
	})
	if err != nil {
		t.Fatalf("evaluate stream error: %v", err)
	}
	if len(errorEvents) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(errorEvents))
	}
	if errorEvents[0].Kind != bobarepl.EventStderr {
		t.Fatalf("expected stderr event, got %q", errorEvents[0].Kind)
	}
}

func TestREPLAPIAdapterSessionPersistsAcrossEvaluations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := newAdapterTestApp(t, replapi.ProfileInteractive, nil)
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	adapter, err := NewREPLAPIAdapter(app, session.ID)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	err = adapter.EvaluateStream(ctx, "const x = 41", func(ev bobarepl.Event) {})
	if err != nil {
		t.Fatalf("seed evaluate stream: %v", err)
	}

	var events []bobarepl.Event
	err = adapter.EvaluateStream(ctx, "x + 1", func(ev bobarepl.Event) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("second evaluate stream: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if got := events[0].Props["markdown"]; got != "42" {
		t.Fatalf("expected markdown result 42, got %#v", got)
	}
}

func TestAppWithRuntimeAutoRestoresPersistentSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openAdapterTestStore(t)
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	factory := newAdapterTestFactory(t)
	app1, err := replapi.New(factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	if err != nil {
		t.Fatalf("new app1: %v", err)
	}
	session, err := app1.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := app1.Evaluate(ctx, session.ID, "const x = 41"); err != nil {
		t.Fatalf("seed evaluate: %v", err)
	}

	app2, err := replapi.New(factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	if err != nil {
		t.Fatalf("new app2: %v", err)
	}

	var preview string
	err = app2.WithRuntime(ctx, session.ID, func(rt *engine.Runtime) error {
		preview = rt.VM.Get("x").String()
		return nil
	})
	if err != nil {
		t.Fatalf("with runtime: %v", err)
	}
	if preview != "41" {
		t.Fatalf("expected restored runtime x=41, got %q", preview)
	}
}

func newAdapterTestFactory(t *testing.T) *engine.Factory {
	t.Helper()

	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func openAdapterTestStore(t *testing.T) *repldb.Store {
	t.Helper()

	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func newAdapterTestApp(t *testing.T, profile replapi.Profile, store *repldb.Store) *replapi.App {
	t.Helper()

	factory := newAdapterTestFactory(t)
	opts := []replapi.Option{replapi.WithProfile(profile)}
	if store != nil {
		opts = append(opts, replapi.WithStore(store))
	}
	app, err := replapi.New(factory, zerolog.Nop(), opts...)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
