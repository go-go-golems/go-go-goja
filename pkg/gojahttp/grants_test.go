package gojahttp_test

import (
	"reflect"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestGrantSetNormalizeDeduplicatesAndSorts(t *testing.T) {
	set, err := gojahttp.NewGrantSet(
		gojahttp.Grant{Action: " project.read ", TenantID: " o1 "},
		gojahttp.Grant{Action: "project.update", ResourceType: "project", ResourceID: "p1"},
		gojahttp.Grant{Action: "project.read", TenantID: "o1"},
	)
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	got := set.ScopeStrings()
	want := []string{"resource:project:p1:project.update", "tenant:o1:project.read"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scopes = %#v, want %#v", got, want)
	}
}

func TestGrantSetNormalizeRejectsInvalidGrants(t *testing.T) {
	if _, err := gojahttp.NewGrantSet(gojahttp.Grant{}); err == nil {
		t.Fatal("expected missing action error")
	}
	if _, err := gojahttp.NewGrantSet(gojahttp.Grant{Action: "project.read", ResourceID: "p1"}); err == nil {
		t.Fatal("expected resource id without type error")
	}
}

func TestGrantSetAllowsByActionTenantAndResource(t *testing.T) {
	set, err := gojahttp.NewGrantSet(
		gojahttp.Grant{Action: "project.read", TenantID: "o1"},
		gojahttp.Grant{Action: "project.update", TenantID: "o1", ResourceType: "project", ResourceID: "p1"},
	)
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	if !set.AllowsResource("project.read", "o1", "project", "p2") {
		t.Fatal("expected tenant-scoped read to allow any project in tenant")
	}
	if set.AllowsResource("project.read", "o2", "project", "p2") {
		t.Fatal("tenant-scoped read should deny other tenant")
	}
	if !set.AllowsResource("project.update", "o1", "project", "p1") {
		t.Fatal("expected resource-scoped update to allow matching project")
	}
	if set.AllowsResource("project.update", "o1", "project", "p2") {
		t.Fatal("resource-scoped update should deny other project")
	}
}

func TestGrantSetWildcardAllowsAnyAction(t *testing.T) {
	set, err := gojahttp.NewGrantSet(gojahttp.Grant{Action: "*", TenantID: "o1"})
	if err != nil {
		t.Fatalf("NewGrantSet: %v", err)
	}
	if !set.Allows("anything.do", &gojahttp.ResourceRef{Type: "report", ID: "r1", TenantID: "o1"}) {
		t.Fatal("expected wildcard tenant grant to allow action")
	}
	if set.Allows("anything.do", &gojahttp.ResourceRef{Type: "report", ID: "r1", TenantID: "o2"}) {
		t.Fatal("wildcard tenant grant should still enforce tenant")
	}
}

func TestGrantSetIntersectNarrowsWildcards(t *testing.T) {
	requested, err := gojahttp.NewGrantSet(
		gojahttp.Grant{Action: "project.read", TenantID: "o1"},
		gojahttp.Grant{Action: "project.update", TenantID: "o1", ResourceType: "project"},
	)
	if err != nil {
		t.Fatalf("requested grants: %v", err)
	}
	approved, err := gojahttp.NewGrantSet(
		gojahttp.Grant{Action: "*", TenantID: "o1", ResourceType: "project", ResourceID: "p1"},
	)
	if err != nil {
		t.Fatalf("approved grants: %v", err)
	}
	got, err := requested.Intersect(approved)
	if err != nil {
		t.Fatalf("Intersect: %v", err)
	}
	want := []string{"tenant:o1:resource:project:p1:project.read", "tenant:o1:resource:project:p1:project.update"}
	if !reflect.DeepEqual(got.ScopeStrings(), want) {
		t.Fatalf("intersection = %#v, want %#v", got.ScopeStrings(), want)
	}
}

func TestGrantSetIntersectRejectsDisjointGrants(t *testing.T) {
	requested, err := gojahttp.NewGrantSet(gojahttp.Grant{Action: "project.read", TenantID: "o1"})
	if err != nil {
		t.Fatalf("requested grants: %v", err)
	}
	approved, err := gojahttp.NewGrantSet(gojahttp.Grant{Action: "project.read", TenantID: "o2"})
	if err != nil {
		t.Fatalf("approved grants: %v", err)
	}
	got, err := requested.Intersect(approved)
	if err != nil {
		t.Fatalf("Intersect: %v", err)
	}
	if len(got.Grants) != 0 {
		t.Fatalf("expected empty intersection, got %#v", got.ScopeStrings())
	}
}
