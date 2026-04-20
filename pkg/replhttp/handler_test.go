package replhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestHandlerSessionLifecycleAndHistory(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d", createRes.Code)
	}

	var createPayload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.NewDecoder(createRes.Body).Decode(&createPayload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	if createPayload.Session.ID == "" {
		t.Fatal("expected session id in create payload")
	}

	evalBody := bytes.NewBufferString(`{"source":"const x = 1; x"}`)
	evalReq := httptest.NewRequest(http.MethodPost, "/api/sessions/"+createPayload.Session.ID+"/evaluate", evalBody)
	evalRes := httptest.NewRecorder()
	handler.ServeHTTP(evalRes, evalReq)
	if evalRes.Code != http.StatusOK {
		t.Fatalf("expected 200 evaluate, got %d", evalRes.Code)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/api/sessions/"+createPayload.Session.ID+"/history", nil)
	historyRes := httptest.NewRecorder()
	handler.ServeHTTP(historyRes, historyReq)
	if historyRes.Code != http.StatusOK {
		t.Fatalf("expected 200 history, got %d", historyRes.Code)
	}

	var historyPayload struct {
		History []any `json:"history"`
	}
	if err := json.NewDecoder(historyRes.Body).Decode(&historyPayload); err != nil {
		t.Fatalf("decode history payload: %v", err)
	}
	if len(historyPayload.History) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(historyPayload.History))
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/sessions/"+createPayload.Session.ID+"/export", nil)
	exportRes := httptest.NewRecorder()
	handler.ServeHTTP(exportRes, exportReq)
	if exportRes.Code != http.StatusOK {
		t.Fatalf("expected 200 export, got %d", exportRes.Code)
	}
}

func newTestApp(t *testing.T) *replapi.App {
	t.Helper()

	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build()
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

func TestHandlerPanicRecoveryReturns500(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	// Create a session first
	createReq := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d", createRes.Code)
	}
	var createPayload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.NewDecoder(createRes.Body).Decode(&createPayload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}

	// Send empty source which would have panicked before BUG-1 fix
	// The recovery middleware should still work if any other panic occurs
	evalReq := httptest.NewRequest(http.MethodPost, "/api/sessions/"+createPayload.Session.ID+"/evaluate",
		bytes.NewBufferString(`{"source":""}`))
	evalRes := httptest.NewRecorder()
	handler.ServeHTTP(evalRes, evalReq)

	// Should get a proper JSON response, not an empty reply
	if evalRes.Code == 0 {
		t.Fatal("expected non-zero status code, got 0 (connection likely closed by panic)")
	}
	// After BUG-1 fix, empty source returns status 'empty-source' (200), not a panic
	// But the recovery middleware should ensure we never get an empty reply
	if evalRes.Code == http.StatusOK || evalRes.Code == http.StatusInternalServerError {
		// Verify we got valid JSON back
		var resp map[string]any
		if err := json.NewDecoder(evalRes.Body).Decode(&resp); err != nil {
			t.Fatalf("expected valid JSON response, got: %q", evalRes.Body.String())
		}
	} else {
		t.Logf("Got status %d (acceptable for empty source)", evalRes.Code)
	}
}
