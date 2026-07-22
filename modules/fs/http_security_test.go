package fs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHTTPHandlerFallbackDoesNotReflectRequestInput(t *testing.T) {
	const indexDocument = `<!doctype html><html><body><main>trusted application shell</main></body></html>`
	backend := NewReadOnlyFSBackend(FSMount{
		FS: fstest.MapFS{
			"assets/index.html": &fstest.MapFile{Data: []byte(indexDocument), Mode: 0o444},
		},
		Root:  "assets",
		Mount: "/",
	})
	handler := &spaHTTPHandler{
		backend:   backend,
		root:      "/",
		indexPath: "/index.html",
		fileServer: http.FileServer(http.FS(&readOnlyHTTPFS{
			backend: backend,
			root:    "/",
		})),
	}

	const attackerInput = `<script>alert(1)</script>`
	req := httptest.NewRequest(http.MethodGet, "https://example.test/%3Cscript%3Ealert%281%29%3C%2Fscript%3E", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != indexDocument {
		t.Fatalf("body = %q, want trusted index document", got)
	}
	if strings.Contains(recorder.Body.String(), attackerInput) {
		t.Fatalf("response reflected request input: %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
}
