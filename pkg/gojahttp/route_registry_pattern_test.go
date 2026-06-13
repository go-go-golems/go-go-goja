package gojahttp

import (
	"net/http"
	"testing"
)

func TestRouteRegistryCapturesNamedParams(t *testing.T) {
	registry := NewRegistry()
	registry.Add(http.MethodGet, "/users/:id/messages/:messageID", nil)
	_, params, ok := registry.Match(http.MethodGet, "/users/u-1/messages/m-2")
	if !ok {
		t.Fatalf("expected route to match")
	}
	if params["id"] != "u-1" || params["messageID"] != "m-2" {
		t.Fatalf("params = %#v", params)
	}
}

func TestRouteRegistryWildcardMatchesRemainderWithoutCapture(t *testing.T) {
	registry := NewRegistry()
	registry.Add(http.MethodGet, "/assets/*", nil)
	_, params, ok := registry.Match(http.MethodGet, "/assets/js/app.js")
	if !ok {
		t.Fatalf("expected wildcard route to match")
	}
	if len(params) != 0 {
		t.Fatalf("wildcard currently matches without capturing splat, params=%#v", params)
	}
}

func TestRouteRegistryWildcardMustBeSegment(t *testing.T) {
	if _, ok := matchPattern("/assets/*.js", "/assets/app.js"); ok {
		t.Fatalf("wildcard inside a segment should not match")
	}
}
