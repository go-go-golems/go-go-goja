package replhttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
)

func TestHandlerSessionLifecycle(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}

	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/sessions", nil))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d: %s", createRes.Code, createRes.Body.String())
	}
	var createPayload replapiv1.CreateSessionResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(createRes.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	sessionID := createPayload.GetSession().GetId()
	if sessionID == "" || createPayload.GetSchemaVersion() != pbconv.SchemaVersion {
		t.Fatalf("bad create response: %#v", &createPayload)
	}

	evalBody := bytes.NewBufferString(`{"schemaVersion":1,"source":"const x = 1; x"}`)
	evalRes := httptest.NewRecorder()
	evalReq := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/evaluate", evalBody)
	evalReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(evalRes, evalReq)
	if evalRes.Code != http.StatusOK {
		t.Fatalf("expected 200 evaluate, got %d: %s", evalRes.Code, evalRes.Body.String())
	}
	var evalPayload replapiv1.EvaluateResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(evalRes.Body.Bytes(), &evalPayload); err != nil {
		t.Fatalf("decode evaluate response: %v", err)
	}
	if evalPayload.GetCell().GetExecution().GetStatus() != "ok" {
		t.Fatalf("bad evaluate response: %#v", &evalPayload)
	}

	historyRes := httptest.NewRecorder()
	handler.ServeHTTP(historyRes, httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID+"/history", nil))
	if historyRes.Code != http.StatusOK {
		t.Fatalf("expected 200 history, got %d: %s", historyRes.Code, historyRes.Body.String())
	}
	var historyPayload replapiv1.HistoryResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(historyRes.Body.Bytes(), &historyPayload); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(historyPayload.GetHistory()) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(historyPayload.GetHistory()))
	}
}

func TestHandlerRejectsUnknownEvaluateFields(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/sessions", nil))
	var createPayload replapiv1.CreateSessionResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(createRes.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	evalRes := httptest.NewRecorder()
	evalReq := httptest.NewRequest(http.MethodPost, "/api/sessions/"+createPayload.GetSession().GetId()+"/evaluate", strings.NewReader(`{"schemaVersion":1,"source":"1","surprise":true}`))
	evalReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(evalRes, evalReq)
	if evalRes.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", evalRes.Code, evalRes.Body.String())
	}
	var errorPayload replapiv1.ErrorResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(evalRes.Body.Bytes(), &errorPayload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errorPayload.GetCode() != "invalid_argument" || strings.Contains(errorPayload.GetMessage(), "unknown field") {
		t.Fatalf("expected stable redacted invalid_argument error, got %#v", &errorPayload)
	}
}

func TestHandlerSessionNotFound(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/sessions/missing", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}
