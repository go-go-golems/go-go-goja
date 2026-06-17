package gojahttp

import "testing"

func TestValidateRoutePlanRequiresSecurityMode(t *testing.T) {
	_, err := ValidateRoutePlan(RoutePlan{Method: "GET", Pattern: "/admin"})
	if err == nil {
		t.Fatal("expected missing security mode error")
	}
}

func TestValidateRoutePlanPublic(t *testing.T) {
	plan, err := ValidateRoutePlan(RoutePlan{Method: "get", Pattern: "healthz", Security: SecuritySpec{Mode: SecurityModePublic}, Action: "ignored"})
	if err != nil {
		t.Fatalf("ValidateRoutePlan: %v", err)
	}
	if plan.Method != "GET" || plan.Pattern != "/healthz" {
		t.Fatalf("normalized plan = %#v", plan)
	}
	if plan.Security.Required {
		t.Fatalf("public route should not be required auth: %#v", plan.Security)
	}
}

func TestValidateRoutePlanUserRequiresAllowAction(t *testing.T) {
	_, err := ValidateRoutePlan(RoutePlan{Method: "GET", Pattern: "/me", Security: SecuritySpec{Mode: SecurityModeUser}})
	if err == nil {
		t.Fatal("expected missing action error")
	}
}

func TestValidateRoutePlanResourceParamValidation(t *testing.T) {
	_, err := ValidateRoutePlan(RoutePlan{
		Method:   "PATCH",
		Pattern:  "/projects/:projectId",
		Security: SecuritySpec{Mode: SecurityModeUser},
		Action:   "project.update",
		Resources: []ResourceSpec{{
			Type: "project",
			ID:   ValueSource{Kind: ValueSourceParam, Key: "id"},
		}},
	})
	if err == nil {
		t.Fatal("expected missing route parameter error")
	}
}

func TestValidateRoutePlanResourceDefaultsNameAndTenantParam(t *testing.T) {
	plan, err := ValidateRoutePlan(RoutePlan{
		Method:   "PATCH",
		Pattern:  "/orgs/:orgId/projects/:projectId",
		Security: SecuritySpec{Mode: SecurityModeUser},
		Action:   "project.update",
		Resources: []ResourceSpec{{
			Type:   "project",
			ID:     ValueSource{Kind: ValueSourceParam, Key: "projectId"},
			Tenant: &ValueSource{Kind: ValueSourceParam, Key: "orgId"},
		}},
	})
	if err != nil {
		t.Fatalf("ValidateRoutePlan: %v", err)
	}
	if got := plan.Resources[0].Name; got != "project" {
		t.Fatalf("resource name = %q", got)
	}
	if plan.Resources[0].Tenant == nil || plan.Resources[0].Tenant.Key != "orgId" {
		t.Fatalf("tenant source = %#v", plan.Resources[0].Tenant)
	}
}

func TestRegisterPlannedStoresPlanOnMatchedRoute(t *testing.T) {
	host := NewHost(HostOptions{})
	plan := RoutePlan{Method: "GET", Pattern: "/healthz", Security: SecuritySpec{Mode: SecurityModePublic}}
	if err := host.RegisterPlanned(plan, nil); err != nil {
		t.Fatalf("RegisterPlanned: %v", err)
	}
	route, _, ok := host.registry.Match("GET", "/healthz")
	if !ok {
		t.Fatal("expected planned route match")
	}
	if route.Plan == nil {
		t.Fatal("matched route missing plan")
	}
	if route.Plan.Security.Mode != SecurityModePublic {
		t.Fatalf("plan security = %#v", route.Plan.Security)
	}
}
