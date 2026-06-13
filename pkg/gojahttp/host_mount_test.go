package gojahttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostRegisterHandlerPreservesRequestPath(t *testing.T) {
	host := NewHost(HostOptions{})
	host.RegisterHandler("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	if rr.Body.String() != "/api/ping" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestHostRegisterHandlerCanStripPrefix(t *testing.T) {
	host := NewHost(HostOptions{})
	host.RegisterHandlerWithOptions("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}), MountOptions{StripPrefix: true})
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	if rr.Body.String() != "/ping" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestHostRegisterHandlerExcludePrefixes(t *testing.T) {
	host := NewHost(HostOptions{})
	host.RegisterHandlerWithOptions("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("root"))
	}), MountOptions{ExcludePrefixes: []string{"/api"}})
	host.RegisterHandler("/api", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("api"))
	}))
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	if rr.Body.String() != "api" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestHostMountedHandlersPrecedeJSRoutes(t *testing.T) {
	host := NewHost(HostOptions{})
	host.RegisterHandler("/api", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("mounted"))
	}))
	host.Register(http.MethodGet, "/api/:id", nil)
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/123", nil))
	if rr.Body.String() != "mounted" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}
