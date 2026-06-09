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

func TestRegistryRoutesReturnsCopySafeDescriptors(t *testing.T) {
	r := NewRegistry()
	r.Add("get", "cards/:id", nil)
	r.Add("POST", "/cards", nil)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Fatalf("Routes len = %d", len(routes))
	}
	if routes[0] != (RouteDescriptor{Method: "GET", Pattern: "/cards/:id"}) {
		t.Fatalf("first route = %#v", routes[0])
	}
	if routes[1] != (RouteDescriptor{Method: "POST", Pattern: "/cards"}) {
		t.Fatalf("second route = %#v", routes[1])
	}

	routes[0].Method = "PATCH"
	again := r.Routes()
	if again[0].Method != "GET" {
		t.Fatalf("Routes returned mutable backing storage: %#v", again)
	}
}

func TestHostRoutesDelegatesToRegistry(t *testing.T) {
	host := NewHost(HostOptions{})
	host.Register("GET", "/hello", nil)
	routes := host.Routes()
	if len(routes) != 1 || routes[0] != (RouteDescriptor{Method: "GET", Pattern: "/hello"}) {
		t.Fatalf("Routes = %#v", routes)
	}
}
