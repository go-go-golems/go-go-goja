package gojahttp

import "testing"

func TestRegistryMatchesParamsInOrder(t *testing.T) {
	r := NewRegistry()
	r.Add("GET", "/cards/:id/move", nil)
	_, params, ok := r.Match("GET", "/cards/42/move")
	if !ok {
		t.Fatal("expected route match")
	}
	if params["id"] != "42" {
		t.Fatalf("expected id param, got %#v", params)
	}
}

func TestRegistryMethodAndWildcard(t *testing.T) {
	r := NewRegistry()
	r.Add("ALL", "/health", nil)
	if _, _, ok := r.Match("POST", "/health"); !ok {
		t.Fatal("expected ALL match")
	}
	if _, _, ok := r.Match("GET", "/missing"); ok {
		t.Fatal("unexpected missing match")
	}
}
