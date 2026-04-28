package fuzz

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func newFactory(t *testing.T) *engine.Factory {
	t.Helper()
	factory, err := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func newRawApp(t *testing.T) *replapi.App {
	t.Helper()
	app, err := replapi.New(newFactory(t), zerolog.Nop(), replapi.WithProfile(replapi.ProfileRaw))
	if err != nil {
		t.Fatalf("create raw app: %v", err)
	}
	return app
}

func newInteractiveApp(t *testing.T) *replapi.App {
	t.Helper()
	app, err := replapi.New(newFactory(t), zerolog.Nop(), replapi.WithProfile(replapi.ProfileInteractive))
	if err != nil {
		t.Fatalf("create interactive app: %v", err)
	}
	return app
}

// newPersistentApp creates a persistent app. Callers should use openStore to
// get a store at the same path for restore verification.
func newPersistentApp(t *testing.T) (*replapi.App, *repldb.Store) {
	t.Helper()
	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app, err := replapi.New(newFactory(t), zerolog.Nop(),
		replapi.WithProfile(replapi.ProfilePersistent),
		replapi.WithStore(store),
	)
	if err != nil {
		t.Fatalf("create persistent app: %v", err)
	}
	return app, store
}

// newPersistentAppAtPath creates a persistent app backed by a specific SQLite file.
func newPersistentAppAtPath(t *testing.T, dbPath string) (*replapi.App, *repldb.Store) {
	t.Helper()
	store, err := repldb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open store at %s: %v", dbPath, err)
	}
	app, err := replapi.New(newFactory(t), zerolog.Nop(),
		replapi.WithProfile(replapi.ProfilePersistent),
		replapi.WithStore(store),
	)
	if err != nil {
		t.Fatalf("create persistent app: %v", err)
	}
	return app, store
}

// safeEval runs Evaluate with panic recovery and a timeout.
// Returns (panicked, panicValue, response, error).
func safeEval(ctx context.Context, app *replapi.App, sessionID, source string) (bool, any, *replsession.EvaluateResponse, error) {
	var didPanic bool
	var panicVal any
	defer func() {
		if r := recover(); r != nil {
			didPanic = true
			panicVal = r
		}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := app.Evaluate(timeoutCtx, sessionID, source)
	return didPanic, panicVal, resp, err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
