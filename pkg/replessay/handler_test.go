package replessay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func TestHandlerServesMeetSessionBootstrapAndHTML(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	bootstrapReq := httptest.NewRequest(http.MethodGet, MeetSessionBootstrapPath, nil)
	bootstrapRes := httptest.NewRecorder()
	handler.ServeHTTP(bootstrapRes, bootstrapReq)
	if bootstrapRes.Code != http.StatusOK {
		t.Fatalf("expected 200 bootstrap, got %d", bootstrapRes.Code)
	}

	var payload BootstrapResponse
	if err := json.NewDecoder(bootstrapRes.Body).Decode(&payload); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	if payload.Section.ID != "meet-a-session" {
		t.Fatalf("expected section id meet-a-session, got %q", payload.Section.ID)
	}
	if payload.DefaultView.Profile != "persistent" {
		t.Fatalf("expected persistent default profile, got %q", payload.DefaultView.Profile)
	}
	if len(payload.Section.Panels) == 0 {
		t.Fatal("expected bootstrap to describe panels")
	}

	pageReq := httptest.NewRequest(http.MethodGet, MeetSessionPagePath, nil)
	pageRes := httptest.NewRecorder()
	handler.ServeHTTP(pageRes, pageReq)
	if pageRes.Code != http.StatusOK {
		t.Fatalf("expected 200 page, got %d", pageRes.Code)
	}
	body := pageRes.Body.String()
	if !strings.Contains(body, "Meet a Session") && !strings.Contains(body, "GOJA-043 REPL Essay") {
		t.Fatalf("expected page to contain essay shell markers, got %q", body)
	}
}

func TestHandlerServesLaterSectionBootstraps(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	endpoints := []string{
		ProfilesBootstrapPath,
		CodeFlowBootstrapPath,
		PersistenceBootstrapPath,
		TimeoutBootstrapPath,
	}
	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", endpoint, res.Code)
		}
	}
}

func TestHandlerArticleScopedSessionLifecycle(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	createReq := httptest.NewRequest(http.MethodPost, MeetSessionCreatePath, nil)
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
		t.Fatal("expected session id from article create route")
	}

	snapshotReq := httptest.NewRequest(http.MethodGet, meetSessionSnapshotPrefix+createPayload.Session.ID, nil)
	snapshotRes := httptest.NewRecorder()
	handler.ServeHTTP(snapshotRes, snapshotReq)
	if snapshotRes.Code != http.StatusOK {
		t.Fatalf("expected 200 snapshot, got %d", snapshotRes.Code)
	}

	var snapshotPayload struct {
		Session struct {
			ID      string `json:"id"`
			Profile string `json:"profile"`
		} `json:"session"`
	}
	if err := json.NewDecoder(snapshotRes.Body).Decode(&snapshotPayload); err != nil {
		t.Fatalf("decode snapshot payload: %v", err)
	}
	if snapshotPayload.Session.ID != createPayload.Session.ID {
		t.Fatalf("expected snapshot id %q, got %q", createPayload.Session.ID, snapshotPayload.Session.ID)
	}
	if snapshotPayload.Session.Profile != "persistent" {
		t.Fatalf("expected persistent profile, got %q", snapshotPayload.Session.Profile)
	}
}

func TestHandlerProfileSectionCreatesSelectedProfile(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	bootstrapReq := httptest.NewRequest(http.MethodGet, ProfilesBootstrapPath, nil)
	bootstrapRes := httptest.NewRecorder()
	handler.ServeHTTP(bootstrapRes, bootstrapReq)
	if bootstrapRes.Code != http.StatusOK {
		t.Fatalf("expected 200 bootstrap, got %d", bootstrapRes.Code)
	}

	var bootstrapPayload ProfileSectionResponse
	if err := json.NewDecoder(bootstrapRes.Body).Decode(&bootstrapPayload); err != nil {
		t.Fatalf("decode profile bootstrap: %v", err)
	}
	if len(bootstrapPayload.Profiles) != 3 {
		t.Fatalf("expected 3 profile specs, got %d", len(bootstrapPayload.Profiles))
	}

	createReq := httptest.NewRequest(http.MethodPost, ProfilesCreatePath, strings.NewReader(`{"profile":"raw"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d", createRes.Code)
	}

	var createPayload struct {
		Session struct {
			ID      string `json:"id"`
			Profile string `json:"profile"`
		} `json:"session"`
	}
	if err := json.NewDecoder(createRes.Body).Decode(&createPayload); err != nil {
		t.Fatalf("decode profile create payload: %v", err)
	}
	if createPayload.Session.Profile != "raw" {
		t.Fatalf("expected raw profile, got %q", createPayload.Session.Profile)
	}
	if createPayload.Session.ID == "" {
		t.Fatal("expected created session id")
	}
}

func TestHandlerCodeFlowEvaluatesLiveSession(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	createReq := httptest.NewRequest(http.MethodPost, CodeFlowCreatePath, strings.NewReader(`{"profile":"interactive"}`))
	createReq.Header.Set("Content-Type", "application/json")
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
		t.Fatalf("decode code-flow create payload: %v", err)
	}

	evalReq := httptest.NewRequest(
		http.MethodPost,
		codeFlowSnapshotPrefix+createPayload.Session.ID+codeFlowEvaluatePathSuffix,
		strings.NewReader(`{"source":"const x = 1; x"}`),
	)
	evalReq.Header.Set("Content-Type", "application/json")
	evalRes := httptest.NewRecorder()
	handler.ServeHTTP(evalRes, evalReq)
	if evalRes.Code != http.StatusOK {
		t.Fatalf("expected 200 evaluate, got %d", evalRes.Code)
	}

	var payload struct {
		Session struct {
			CellCount int `json:"cellCount"`
		} `json:"session"`
		Cell struct {
			Rewrite struct {
				TransformedSource string `json:"transformedSource"`
			} `json:"rewrite"`
			Execution struct {
				Status string `json:"status"`
			} `json:"execution"`
		} `json:"cell"`
	}
	if err := json.NewDecoder(evalRes.Body).Decode(&payload); err != nil {
		t.Fatalf("decode evaluate payload: %v", err)
	}
	if payload.Cell.Execution.Status != "ok" {
		t.Fatalf("expected ok execution status, got %q", payload.Cell.Execution.Status)
	}
	if payload.Cell.Rewrite.TransformedSource == "" {
		t.Fatal("expected transformed source in live evaluate response")
	}
	if payload.Session.CellCount != 1 {
		t.Fatalf("expected session cell count 1 after evaluation, got %d", payload.Session.CellCount)
	}
}

func TestHandlerRedirectsStaticEssayPrefix(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/static/essay", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301 static prefix redirect, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/static/essay/" {
		t.Fatalf("expected static prefix redirect location /static/essay/, got %q", location)
	}
}

func newTestHandler(t *testing.T) http.Handler {
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
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new essay handler: %v", err)
	}
	return handler
}
