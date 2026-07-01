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

func TestProtoJSONHandlerSessionLifecycle(t *testing.T) {
	t.Parallel()

	handler, err := NewProtoJSONHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}

	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil))
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
	handler.ServeHTTP(evalRes, httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sessionID+"/evaluate", evalBody))
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
	handler.ServeHTTP(historyRes, httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/history", nil))
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

func TestProtoJSONHandlerRejectsUnknownEvaluateFields(t *testing.T) {
	t.Parallel()

	handler, err := NewProtoJSONHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil))
	var createPayload replapiv1.CreateSessionResponse
	if err := pbconv.UnmarshalOptions.Unmarshal(createRes.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	evalRes := httptest.NewRecorder()
	handler.ServeHTTP(evalRes, httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+createPayload.GetSession().GetId()+"/evaluate", strings.NewReader(`{"source":"1","surprise":true}`)))
	if evalRes.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", evalRes.Code, evalRes.Body.String())
	}
	if !strings.Contains(evalRes.Body.String(), "unknown field") {
		t.Fatalf("expected unknown field error, got %s", evalRes.Body.String())
	}
}

func TestProtoJSONHandlerSessionNotFound(t *testing.T) {
	t.Parallel()

	handler, err := NewProtoJSONHandler(newTestApp(t))
	if err != nil {
		t.Fatalf("new proto handler: %v", err)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/sessions/missing", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}
