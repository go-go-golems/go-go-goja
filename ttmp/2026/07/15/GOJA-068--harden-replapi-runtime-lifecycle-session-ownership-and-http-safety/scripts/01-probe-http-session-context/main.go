package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replhttp"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	tmp, err := os.MkdirTemp("", "goja-068-http-context-")
	must(err)
	defer func() { _ = os.RemoveAll(tmp) }()

	store, err := repldb.Open(ctx, filepath.Join(tmp, "repl.sqlite"))
	must(err)
	defer func() { _ = store.Close() }()

	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	must(err)
	app, err := replapi.New(context.Background(), factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	must(err)
	defer func() { _ = app.Close(context.Background()) }()
	handler, err := replhttp.NewHandler(app)
	must(err)

	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/sessions", "application/json", nil)
	must(err)
	defer func() { _ = response.Body.Close() }()

	var payload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	must(json.NewDecoder(response.Body).Decode(&payload))

	// Let net/http return from ServeHTTP and cancel the request context.
	time.Sleep(25 * time.Millisecond)

	var lifetimeErr error
	must(app.WithRuntime(ctx, payload.Session.ID, func(_ context.Context, rt *engine.Runtime) error {
		lifetimeErr = rt.Context().Err()
		return nil
	}))

	fmt.Printf("session=%s runtime_lifetime_error=%v\n", payload.Session.ID, lifetimeErr)
	_ = app.DeleteSession(context.Background(), payload.Session.ID)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
