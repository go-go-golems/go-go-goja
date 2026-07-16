package replhttp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func TestHandlerConfigHasBoundedDefaults(t *testing.T) {
	config := DefaultHandlerConfig()
	if config.MaxRequestBodyBytes <= 0 || config.MaxSourceBytes <= 0 {
		t.Fatalf("expected positive limits, got body=%d source=%d", config.MaxRequestBodyBytes, config.MaxSourceBytes)
	}
	if int64(config.MaxSourceBytes) >= config.MaxRequestBodyBytes {
		t.Fatalf("source limit must leave room for JSON framing: body=%d source=%d", config.MaxRequestBodyBytes, config.MaxSourceBytes)
	}
	app := newTestApp(t)
	for _, options := range [][]HandlerOption{
		{WithMaxRequestBodyBytes(0)},
		{WithMaxSourceBytes(0)},
		{WithMaxRequestBodyBytes(8), WithMaxSourceBytes(9)},
	} {
		if _, err := NewHandler(app, options...); err == nil {
			t.Fatalf("expected invalid limits %#v to fail", options)
		}
	}
}

func TestSuccessResponsesCarryRequestAndSecurityHeaders(t *testing.T) {
	app := newTestApp(t)
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	request.Header.Set(RequestIDHeader, "success-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get(RequestIDHeader) != "success-request" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("missing success security/request headers: %#v", response.Header())
	}
}

func TestEvaluateRequestValidationAndSecurityHeaders(t *testing.T) {
	app := newTestApp(t)
	handler, err := NewHandler(app, WithMaxRequestBodyBytes(256), WithMaxSourceBytes(4))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	sessionID := createHTTPSession(t, server.URL)

	tests := []struct {
		name        string
		contentType string
		body        string
		status      int
		code        string
	}{
		{name: "content type required", body: `{"schemaVersion":1,"source":"1"}`, status: http.StatusBadRequest, code: "invalid_content_type"},
		{name: "wrong content type", contentType: "text/plain", body: `{"schemaVersion":1,"source":"1"}`, status: http.StatusBadRequest, code: "invalid_content_type"},
		{name: "missing version", contentType: "application/json", body: `{"source":"1"}`, status: http.StatusBadRequest, code: "unsupported_schema_version"},
		{name: "future version", contentType: "application/json; charset=utf-8", body: `{"schemaVersion":2,"source":"1"}`, status: http.StatusBadRequest, code: "unsupported_schema_version"},
		{name: "unknown field", contentType: "application/json", body: `{"schemaVersion":1,"source":"1","surprise":true}`, status: http.StatusBadRequest, code: "invalid_argument"},
		{name: "source too large", contentType: "application/json", body: `{"schemaVersion":1,"source":"12345"}`, status: http.StatusRequestEntityTooLarge, code: "source_too_large"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, server.URL+"/api/sessions/"+sessionID+"/evaluate", strings.NewReader(test.body))
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}
			req.Header.Set(RequestIDHeader, "validation-request")
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != test.status {
				t.Fatalf("status=%d, want %d: %s", response.StatusCode, test.status, body)
			}
			payload := decodeHTTPError(t, body)
			if payload.GetCode() != test.code || payload.GetRequestId() != "validation-request" {
				t.Fatalf("unexpected error payload: %#v", &payload)
			}
			if response.Header.Get("X-Content-Type-Options") != "nosniff" || response.Header.Get(RequestIDHeader) != "validation-request" {
				t.Fatalf("missing security/request headers: %#v", response.Header)
			}
		})
	}
}

func TestEvaluateRequestBodyLimitUsesHTTP413(t *testing.T) {
	app := newTestApp(t)
	handler, err := NewHandler(app, WithMaxRequestBodyBytes(64), WithMaxSourceBytes(16))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	sessionID := createHTTPSession(t, server.URL)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/sessions/"+sessionID+"/evaluate", strings.NewReader(strings.Repeat("x", 65)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Force chunked transfer so this exercises MaxBytesReader rather than only
	// the Content-Length fast rejection.
	req.ContentLength = -1
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	payload := decodeHTTPError(t, body)
	if response.StatusCode != http.StatusRequestEntityTooLarge || payload.GetCode() != "request_too_large" {
		t.Fatalf("expected 413 request_too_large, got %d %#v", response.StatusCode, &payload)
	}
}

func TestTypedTransportErrorMappings(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "owned", err: repldb.ErrSessionOwned, status: http.StatusConflict, code: "session_owned"},
		{name: "lease lost", err: repldb.ErrLeaseLost, status: http.StatusConflict, code: "session_not_writable"},
		{name: "write conflict", err: repldb.ErrWriteConflict, status: http.StatusConflict, code: "session_not_writable"},
		{name: "degraded", err: replsession.ErrSessionDegraded, status: http.StatusConflict, code: "session_not_writable"},
		{name: "fenced", err: replsession.ErrSessionFenced, status: http.StatusConflict, code: "session_not_writable"},
		{name: "commit", err: replsession.ErrCommitFailed, status: http.StatusServiceUnavailable, code: "persistence_unavailable"},
		{name: "closing", err: replapi.ErrAppClosing, status: http.StatusServiceUnavailable, code: "service_shutting_down"},
		{name: "canceled", err: context.Canceled, status: http.StatusServiceUnavailable, code: "service_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapTransportError(test.err)
			if mapped.status != test.status || mapped.code != test.code {
				t.Fatalf("mapped %#v to status=%d code=%q, want %d %q", test.err, mapped.status, mapped.code, test.status, test.code)
			}
		})
	}
}

func TestHTTPStatusMappingAndInternalRedaction(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		app := newTestApp(t)
		handler, err := NewHandler(app)
		if err != nil {
			t.Fatalf("new handler: %v", err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/sessions/missing", nil))
		payload := decodeHTTPError(t, response.Body.Bytes())
		if response.Code != http.StatusNotFound || payload.GetCode() != "session_not_found" {
			t.Fatalf("expected not found mapping, got %d %#v", response.Code, &payload)
		}
	})

	t.Run("app closed", func(t *testing.T) {
		app := newTestApp(t)
		handler, err := NewHandler(app)
		if err != nil {
			t.Fatalf("new handler: %v", err)
		}
		if err := app.Close(context.Background()); err != nil {
			t.Fatalf("close app: %v", err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/sessions", nil))
		payload := decodeHTTPError(t, response.Body.Bytes())
		if response.Code != http.StatusServiceUnavailable || payload.GetCode() != "service_shutting_down" {
			t.Fatalf("expected shutdown mapping, got %d %#v", response.Code, &payload)
		}
	})

	t.Run("session owned", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "owned.sqlite")
		appA, storeA := newPersistentHTTPTestApp(t, dbPath)
		appB, storeB := newPersistentHTTPTestApp(t, dbPath)
		t.Cleanup(func() {
			_ = appB.Close(context.Background())
			_ = appA.Close(context.Background())
			_ = storeB.Close()
			_ = storeA.Close()
		})
		summary, err := appA.CreateSession(context.Background())
		if err != nil {
			t.Fatalf("create owned session: %v", err)
		}
		handler, err := NewHandler(appB)
		if err != nil {
			t.Fatalf("new handler: %v", err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/sessions/"+summary.ID, nil))
		payload := decodeHTTPError(t, response.Body.Bytes())
		if response.Code != http.StatusConflict || payload.GetCode() != "session_owned" {
			t.Fatalf("expected ownership mapping, got %d %#v", response.Code, &payload)
		}
	})

	t.Run("database details redacted but logged", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "closed.sqlite")
		app, store := newPersistentHTTPTestApp(t, dbPath)
		t.Cleanup(func() { _ = app.Close(context.Background()) })
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
		var logs bytes.Buffer
		handler, err := NewHandler(app, WithHandlerLogger(zerolog.New(&logs)))
		if err != nil {
			t.Fatalf("new handler: %v", err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/sessions", nil))
		payload := decodeHTTPError(t, response.Body.Bytes())
		if response.Code != http.StatusInternalServerError || payload.GetCode() != "internal" || payload.GetMessage() != "internal server error" {
			t.Fatalf("expected redacted internal mapping, got %d %#v", response.Code, &payload)
		}
		if strings.Contains(response.Body.String(), "sql") || strings.Contains(response.Body.String(), "closed") {
			t.Fatalf("client response leaked internal database detail: %s", response.Body.String())
		}
		if !strings.Contains(logs.String(), "database is closed") {
			t.Fatalf("expected server diagnostics to retain error detail, got %s", logs.String())
		}
	})
}

func TestCanceledHTTPRequestDoesNotPoisonSession(t *testing.T) {
	app := newTestApp(t)
	handler, err := NewHandler(app)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	sessionID := createHTTPSession(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/sessions/"+sessionID+"/evaluate", strings.NewReader(`{"schemaVersion":1,"source":"for (;;) {}"}`))
	if err != nil {
		t.Fatalf("new canceled request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err == nil {
		_ = response.Body.Close()
		t.Fatal("expected client request deadline to expire")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		response, body := evaluateHTTP(t, server.URL, sessionID, "1 + 1")
		if response.StatusCode == http.StatusOK {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("session did not recover after request cancellation: %d %s", response.StatusCode, body)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func createHTTPSession(t *testing.T, baseURL string) string {
	t.Helper()
	response, err := http.Post(baseURL+"/api/sessions", "application/json", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d: %s", response.StatusCode, body)
	}
	var payload replapiv1.CreateSessionResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return payload.GetSession().GetId()
}

func evaluateHTTP(t *testing.T, baseURL string, sessionID string, source string) (*http.Response, []byte) {
	t.Helper()
	body, err := pbconv.MarshalJSON(&replapiv1.EvaluateRequest{SchemaVersion: pbconv.SchemaVersion, Source: source})
	if err != nil {
		t.Fatalf("marshal evaluate request: %v", err)
	}
	response, err := http.Post(baseURL+"/api/sessions/"+sessionID+"/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	responseBody, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	return response, responseBody
}

func decodeHTTPError(t *testing.T, body []byte) *replapiv1.ErrorResponse {
	t.Helper()
	payload := &replapiv1.ErrorResponse{}
	if err := pbconv.UnmarshalOptions.Unmarshal(body, payload); err != nil {
		t.Fatalf("decode error response %q: %v", body, err)
	}
	return payload
}

func newPersistentHTTPTestApp(t *testing.T, dbPath string) (*replapi.App, *repldb.Store) {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	store, err := repldb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app, err := replapi.New(context.Background(), factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	if err != nil {
		_ = store.Close()
		t.Fatalf("new app: %v", err)
	}
	return app, store
}
