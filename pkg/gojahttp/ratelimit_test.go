package gojahttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestValidateRoutePlanNormalizesRateLimits(t *testing.T) {
	plan, err := gojahttp.ValidateRoutePlan(gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/orgs/:orgId/projects",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic},
		RateLimits: []gojahttp.RateLimitSpec{{
			Policy: " public.projects ",
			Limit:  10,
			Window: time.Minute,
			KeyParts: []gojahttp.RateLimitKeyPart{
				{Kind: gojahttp.RateLimitKeyRoute},
				{Kind: gojahttp.RateLimitKeyTenantParam, Key: "orgId"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("ValidateRoutePlan: %v", err)
	}
	if got := plan.RateLimits[0].Policy; got != "public.projects" {
		t.Fatalf("policy = %q", got)
	}
	if got := plan.RateLimits[0].KeyParts[1].Key; got != "orgId" {
		t.Fatalf("key = %q", got)
	}
}

func TestValidateRoutePlanRejectsInvalidRateLimitParam(t *testing.T) {
	_, err := gojahttp.ValidateRoutePlan(gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/orgs/:orgId/projects",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic},
		RateLimits: []gojahttp.RateLimitSpec{{
			Policy:   "bad",
			Limit:    1,
			Window:   time.Minute,
			KeyParts: []gojahttp.RateLimitKeyPart{{Kind: gojahttp.RateLimitKeyParam, Key: "missing"}},
		}},
	})
	if err == nil {
		t.Fatal("expected unknown route parameter error")
	}
}

func TestEnforcerPreAuthRateLimitDeniesPublicRoute(t *testing.T) {
	limiter := gojahttp.NewMemoryRateLimiter()
	enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{Auth: gojahttp.AuthOptions{RateLimiter: limiter}})
	plan := &gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/public",
		Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic},
		RateLimits: []gojahttp.RateLimitSpec{{
			Policy:   "public.read",
			Limit:    1,
			Window:   time.Minute,
			KeyParts: []gojahttp.RateLimitKeyPart{{Kind: gojahttp.RateLimitKeyRoute}, {Kind: gojahttp.RateLimitKeyIP}},
		}},
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/public", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		dto, err := enforcer.Request(httptest.NewRecorder(), req, nil)
		if err != nil {
			t.Fatalf("Request: %v", err)
		}
		_, status, err := enforcer.Enforce(context.Background(), req, dto, plan)
		if i == 0 && (err != nil || status != 0) {
			t.Fatalf("first Enforce status=%d err=%v", status, err)
		}
		if i == 1 {
			if !errors.Is(err, gojahttp.ErrRateLimited) || status != http.StatusTooManyRequests {
				t.Fatalf("second Enforce status=%d err=%v", status, err)
			}
		}
	}
}

func TestEnforcerPostAuthRateLimitDoesNotChargeDeniedRequest(t *testing.T) {
	limiter := gojahttp.NewMemoryRateLimiter()
	allowed := false
	enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{Auth: gojahttp.AuthOptions{
		RateLimiter: limiter,
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Resources: resolverFunc(func(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
			return &gojahttp.ResourceRef{ID: req.ID, Type: req.Spec.Type}, nil
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: allowed}, nil
		}),
	}})
	plan := &gojahttp.RoutePlan{
		Method:   http.MethodPost,
		Pattern:  "/projects/:projectID/run",
		Security: gojahttp.User().Required(),
		Resources: []gojahttp.ResourceSpec{
			gojahttp.Resource("project").IDFromParam("projectID").MustExist(),
		},
		Action: "project.run",
		RateLimits: []gojahttp.RateLimitSpec{{
			Policy:   "project.write",
			Limit:    1,
			Window:   time.Minute,
			KeyParts: []gojahttp.RateLimitKeyPart{{Kind: gojahttp.RateLimitKeyResource, Key: "project"}},
		}},
	}
	enforce := func() (int, error) {
		req := httptest.NewRequest(http.MethodPost, "/projects/p1/run", nil)
		dto, err := enforcer.Request(httptest.NewRecorder(), req, map[string]string{"projectID": "p1"})
		if err != nil {
			t.Fatal(err)
		}
		_, status, err := enforcer.Enforce(context.Background(), req, dto, plan)
		return status, err
	}
	if status, err := enforce(); status != http.StatusForbidden || !errors.Is(err, gojahttp.ErrForbidden) {
		t.Fatalf("denied request status=%d err=%v", status, err)
	}
	allowed = true
	if status, err := enforce(); status != 0 || err != nil {
		t.Fatalf("authorized request was charged by prior denial: status=%d err=%v", status, err)
	}
}

func TestEnforcerPostAuthRateLimitUsesActor(t *testing.T) {
	limiter := gojahttp.NewMemoryRateLimiter()
	enforcer := gojahttp.NewEnforcer(gojahttp.EnforcerOptions{Auth: gojahttp.AuthOptions{
		RateLimiter: limiter,
		Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
			return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
		}),
		Authorizer: authorizerFunc(func(context.Context, gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
			return gojahttp.AuthorizationDecision{Allowed: true}, nil
		}),
	}})
	plan := &gojahttp.RoutePlan{
		Method:   http.MethodPost,
		Pattern:  "/me/run",
		Security: gojahttp.User().Required(),
		Action:   "job.run",
		RateLimits: []gojahttp.RateLimitSpec{{
			Policy:   "actor.write",
			Limit:    1,
			Window:   time.Minute,
			KeyParts: []gojahttp.RateLimitKeyPart{{Kind: gojahttp.RateLimitKeyActor}, {Kind: gojahttp.RateLimitKeyRoute}},
		}},
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/me/run", nil)
		dto, err := enforcer.Request(httptest.NewRecorder(), req, nil)
		if err != nil {
			t.Fatalf("Request: %v", err)
		}
		_, status, err := enforcer.Enforce(context.Background(), req, dto, plan)
		if i == 0 && (err != nil || status != 0) {
			t.Fatalf("first Enforce status=%d err=%v", status, err)
		}
		if i == 1 {
			if !errors.Is(err, gojahttp.ErrRateLimited) || status != http.StatusTooManyRequests {
				t.Fatalf("second Enforce status=%d err=%v", status, err)
			}
		}
	}
}
