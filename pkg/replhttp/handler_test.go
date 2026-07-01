package replhttp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func newTestApp(t *testing.T) *replapi.App {
	t.Helper()

	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	store, err := repldb.Open(context.Background(), filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	app, err := replapi.New(
		factory,
		zerolog.Nop(),
		replapi.WithProfile(replapi.ProfilePersistent),
		replapi.WithStore(store),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
