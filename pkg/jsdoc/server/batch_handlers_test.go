package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

func TestBatchExtract(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte(`__doc__({"name":"fn","summary":"hello"})`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := New(model.NewDocStore(), dir, "127.0.0.1", 0)
	h := s.Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/batch/extract", strings.NewReader(`{"inputs":[{"path":"a.js"}]}`))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var resp struct {
		Store model.DocStore `json:"store"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := len(resp.Store.BySymbol); got != 1 {
		t.Fatalf("expected 1 symbol, got %d", got)
	}
}

func TestBatchExport_Markdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte(`__doc__({"name":"fn","summary":"hello"})`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := New(model.NewDocStore(), dir, "127.0.0.1", 0)
	h := s.Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/batch/export", strings.NewReader(`{
		"inputs":[{"path":"a.js"}],
		"format":"markdown",
		"options":{"tocDepth":2}
	}`))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/markdown") {
		t.Fatalf("expected markdown content-type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "## Packages") {
		t.Fatalf("expected markdown packages section, got:\n%s", rr.Body.String())
	}
}

func TestBatchExport_SQLiteHeadersAndBody(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte(`__doc__({"name":"fn","summary":"hello"})`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := New(model.NewDocStore(), dir, "127.0.0.1", 0)
	h := s.Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/batch/export", strings.NewReader(`{"inputs":[{"path":"a.js"}],"format":"sqlite"}`))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("expected application/octet-stream, got %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "docs.sqlite") {
		t.Fatalf("expected content-disposition to include docs.sqlite, got %q", cd)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected non-empty sqlite body")
	}
}

func TestBatchPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	s := New(model.NewDocStore(), dir, "127.0.0.1", 0)
	h := s.Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/batch/extract", strings.NewReader(`{"inputs":[{"path":"../a.js"}]}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
