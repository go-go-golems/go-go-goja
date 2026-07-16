package replhttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
)

// TestHardeningHTTPSessionRuntimeOutlivesCreateRequest is the red regression
// for GOJA-068 P0.1. It uses a real net/http server because direct ServeHTTP
// tests do not model request-context cancellation after the handler returns.
func TestHardeningHTTPSessionRuntimeOutlivesCreateRequest(t *testing.T) {
	app := newTestApp(t)
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	response, err := http.Post(server.URL+"/api/sessions", "application/json", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		t.Fatalf("read create response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close create response: %v", closeErr)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d: %s", response.StatusCode, body)
	}

	var payload replapiv1.CreateSessionResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	sessionID := payload.GetSession().GetId()
	if sessionID == "" {
		t.Fatal("expected non-empty session id")
	}

	// Give net/http a bounded window to cancel the request context after
	// ServeHTTP returns. A correctly app-owned runtime remains active through
	// the entire observation window.
	deadline := time.Now().Add(100 * time.Millisecond)
	for {
		var lifetimeErr error
		if err := app.WithRuntime(context.Background(), sessionID, func(_ context.Context, rt *engine.Runtime) error {
			lifetimeErr = rt.Context().Err()
			return nil
		}); err != nil {
			t.Fatalf("inspect runtime lifetime: %v", err)
		}
		if lifetimeErr != nil {
			t.Fatalf("runtime lifetime was canceled with create request: %v", lifetimeErr)
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
}
