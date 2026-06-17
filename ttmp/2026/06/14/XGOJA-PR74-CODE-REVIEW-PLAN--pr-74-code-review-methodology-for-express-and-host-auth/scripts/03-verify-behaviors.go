// Verification harness for PR 74 code review. Runs against the module's
// exported APIs only; does not modify production code. Invoke from the
// go-go-goja repo root:
//
//   GOFLAGS=-buildvcs=false go run ./ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/03-verify-behaviors.go
//
//go:build ignore

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
)

type authenticatorFunc func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error)

func (f authenticatorFunc) Authenticate(ctx context.Context, req *http.Request, session *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	return f(ctx, req, session, spec)
}

func plannedRuntime(host *gojahttp.Host, script string) (goja.Callable, func()) {
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		panic(err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		panic(err)
	}
	host.SetRuntime(rt.Owner)
	ret, err := rt.Owner.Call(context.Background(), "verify", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(script)
	})
	if err != nil {
		panic(err)
	}
	fn, ok := goja.AssertFunction(ret.(goja.Value))
	if !ok {
		panic(fmt.Sprintf("script did not return function: %T", ret))
	}
	return fn, func() { _ = rt.Close(context.Background()) }
}

func register(host *gojahttp.Host, plan gojahttp.RoutePlan) goja.Callable {
	handler, closeRT := plannedRuntime(host, `(function(_ctx, res) { res.send("ran"); })`)
	_ = closeRT
	if err := host.RegisterPlanned(plan, handler); err != nil {
		panic(err)
	}
	return handler
}

func main() {
	// --- Finding: authenticator returns (nil, nil) -> status code ---
	fmt.Println("== behavior A: authenticator returns (nil actor, nil error) ==")
	{
		host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return nil, nil // bug-prone contract: success but no actor
			}),
			Authorizer: nil, // never reached
		}})
		register(host, gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"})
		rr := httptest.NewRecorder()
		host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
		fmt.Printf("  (nil,nil) -> status=%d body=%q\n", rr.Code, rr.Body.String())
	}

	// --- Finding: authenticator returns generic error vs sentinel ---
	fmt.Println("== behavior B: authenticator returns generic error ==")
	{
		host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return nil, fmt.Errorf("internal lookup blew up")
			}),
		}})
		register(host, gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"})
		rr := httptest.NewRecorder()
		host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
		fmt.Printf("  generic err -> status=%d body=%q (dev mode leaks: %v)\n", rr.Code, rr.Body.String(), contains(rr.Body.String(), "blew up"))
	}

	// --- Finding: authenticator returns generic error (PRODUCTION, dev=false) ---
	fmt.Println("== behavior B2: generic error in production (dev=false) ==")
	{
		host := gojahttp.NewHost(gojahttp.HostOptions{Dev: false, Auth: gojahttp.AuthOptions{
			Authenticator: authenticatorFunc(func(context.Context, *http.Request, *gojahttp.SessionDTO, gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
				return nil, fmt.Errorf("internal lookup blew up")
			}),
		}})
		register(host, gojahttp.RoutePlan{Method: "GET", Pattern: "/me", Security: gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser}, Action: "user.self.read"})
		rr := httptest.NewRecorder()
		host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/me", nil))
		fmt.Printf("  prod generic err -> status=%d body=%q (leaks internals: %v)\n", rr.Code, rr.Body.String(), contains(rr.Body.String(), "blew up"))
	}

	// --- Finding: audit over-redaction of capabilityId ---
	fmt.Println("== behavior C: audit RedactMap over-redaction ==")
	{
		ev := gojahttp.AuditEvent{
			Event:    "capability.redeemed",
			Outcome:  "completed",
			Action:   "org.member.invite",
			Reason:   "",
			Actor:    &gojahttp.Actor{ID: "u1", Kind: "user"},
			Resource: &gojahttp.ResourceRef{Type: "org", ID: "o1"},
			Attributes: map[string]any{
				"capabilityId": "cap_123",
				"purpose":      "invite",
				"subjectId":    "u1",
				"sessionId":    "sess_xyz",
			},
		}
		mem := &audit.MemoryStore{}
		sink := audit.Sink{Store: mem}
		if err := sink.RecordAudit(context.Background(), ev); err != nil {
			panic(err)
		}
		rec := mem.Snapshot()[0]
		fmt.Printf("  normalized record attributes = %#v\n", rec.Attributes)
	}

	// --- Finding: capability single-use + hashing ---
	fmt.Println("== behavior D: capability token hashing + single-use ==")
	{
		store := capability.NewMemoryStore()
		svc := capability.Service{Store: store, Now: func() time.Time { return time.Unix(0, 0) }}
		res, err := svc.Issue(context.Background(), capability.IssueSpec{
			Purpose:   "invite",
			SubjectID: "u1",
			TTL:       3600,
			SingleUse: true,
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("  raw token returned once: present=%v (len=%d)\n", res.Token != "", len(res.Token))
		fmt.Printf("  stored capability TokenHash redacted in result: %v\n", res.Capability.TokenHash == nil)
		c1, err := svc.Redeem(context.Background(), "invite", res.Token)
		fmt.Printf("  first redeem: ok=%v id=%s err=%v\n", c1 != nil, func() string { if c1 != nil { return c1.ID }; return "" }(), err)
		c2, err := svc.Redeem(context.Background(), "invite", res.Token)
		fmt.Printf("  second redeem (single-use): ok=%v err=%v\n", c2 != nil, err)
	}

	fmt.Println("\nALL CHECKS COMPLETE")
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
