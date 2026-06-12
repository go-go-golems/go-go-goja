package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type demoAuthServices struct {
	mu     sync.Mutex
	audits []gojahttp.AuditEvent
}

func (s *demoAuthServices) Authenticate(_ context.Context, req *http.Request, _ *gojahttp.SessionDTO, _ gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	if req.Header.Get("Authorization") != "Bearer demo-user" {
		return nil, gojahttp.ErrUnauthenticated
	}
	return &gojahttp.Actor{ID: "u1", Kind: "user", TenantIDs: []string{"o1"}}, nil
}

func (s *demoAuthServices) ResolveResource(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	if req.Spec.Type != "project" || req.ID != "p1" || req.TenantID != "o1" {
		return nil, gojahttp.ErrNotFound
	}
	return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID, TenantID: req.TenantID}, nil
}

func (s *demoAuthServices) Authorize(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	if req.Actor == nil {
		return gojahttp.AuthorizationDecision{}, nil
	}
	switch req.Action {
	case "user.self.read":
		return gojahttp.AuthorizationDecision{Allowed: true}, nil
	case "project.update":
		allowed := req.Resource != nil && req.Resource.ID == "p1" && req.Resource.TenantID == "o1"
		return gojahttp.AuthorizationDecision{Allowed: allowed, Reason: reasonIf(!allowed, "project not allowed")}, nil
	default:
		return gojahttp.AuthorizationDecision{Allowed: false, Reason: "unknown action"}, nil
	}
}

func (s *demoAuthServices) VerifyCSRF(_ context.Context, req gojahttp.CSRFRequest) error {
	if req.HTTPRequest.Header.Get("X-CSRF-Token") != "demo-csrf" {
		return errors.New("missing or invalid X-CSRF-Token")
	}
	return nil
}

func (s *demoAuthServices) RecordAudit(_ context.Context, event gojahttp.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, event)
	log.Printf("audit event=%s outcome=%s actor=%s action=%s status=%d reason=%s", event.Event, event.Outcome, actorID(event.Actor), event.Action, event.StatusCode, event.Reason)
	return nil
}

func (s *demoAuthServices) auditCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.audits)
}

func reasonIf(cond bool, reason string) string {
	if cond {
		return reason
	}
	return ""
}

func actorID(actor *gojahttp.Actor) string {
	if actor == nil {
		return ""
	}
	return actor.ID
}

func main() {
	listen := flag.String("listen", "127.0.0.1:8788", "listen address for manual server mode")
	script := flag.String("script", "examples/xgoja/16-express-auth-host/scripts/server.js", "JavaScript route script")
	smoke := flag.Bool("smoke", false, "run an in-process smoke test instead of listening")
	flag.Parse()

	if err := run(context.Background(), *listen, *script, *smoke); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, listen, script string, smoke bool) error {
	services := &demoAuthServices{}
	host := gojahttp.NewHost(gojahttp.HostOptions{
		Dev:             true,
		RejectRawRoutes: true,
		Auth: gojahttp.AuthOptions{
			Authenticator: services,
			Resources:     services,
			Authorizer:    services,
			CSRF:          services,
			Audit:         services,
		},
	})
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(express.NewRegistrar(host)).Build()
	if err != nil {
		return err
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(ctx) }()
	host.SetRuntime(rt.Owner)
	data, err := os.ReadFile(script)
	if err != nil {
		return err
	}
	if _, err := rt.Owner.Call(ctx, "load-auth-example", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(string(data))
		return nil, runErr
	}); err != nil {
		return err
	}
	if smoke {
		return runSmoke(host, services)
	}
	log.Printf("serving auth demo on http://%s", listen)
	log.Printf("try: curl -i http://%s/healthz", listen)
	log.Printf("try: curl -i -H 'Authorization: Bearer demo-user' http://%s/me", listen)
	log.Printf("try: curl -i -X PATCH -H 'Authorization: Bearer demo-user' -H 'X-CSRF-Token: demo-csrf' http://%s/orgs/o1/projects/p1", listen)
	return http.ListenAndServe(listen, host)
}

func runSmoke(host http.Handler, services *demoAuthServices) error {
	server := httptest.NewServer(host)
	defer server.Close()
	checks := []struct {
		name       string
		method     string
		path       string
		auth       bool
		csrf       bool
		wantStatus int
		wantBody   string
	}{
		{name: "public health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK, wantBody: `"ok":true`},
		{name: "me unauthenticated", method: http.MethodGet, path: "/me", wantStatus: http.StatusUnauthorized},
		{name: "me authenticated", method: http.MethodGet, path: "/me", auth: true, wantStatus: http.StatusOK, wantBody: `"id":"u1"`},
		{name: "project missing csrf", method: http.MethodPatch, path: "/orgs/o1/projects/p1", auth: true, wantStatus: http.StatusForbidden},
		{name: "project update", method: http.MethodPatch, path: "/orgs/o1/projects/p1", auth: true, csrf: true, wantStatus: http.StatusOK, wantBody: `"updated":"p1"`},
		{name: "project missing", method: http.MethodPatch, path: "/orgs/o1/projects/missing", auth: true, csrf: true, wantStatus: http.StatusNotFound},
	}
	client := server.Client()
	for _, check := range checks {
		req, err := http.NewRequest(check.method, server.URL+check.path, bytes.NewReader(nil))
		if err != nil {
			return err
		}
		if check.auth {
			req.Header.Set("Authorization", "Bearer demo-user")
		}
		if check.csrf {
			req.Header.Set("X-CSRF-Token", "demo-csrf")
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("%s: read body: %w", check.name, readErr)
		}
		if resp.StatusCode != check.wantStatus {
			return fmt.Errorf("%s: status=%d body=%s", check.name, resp.StatusCode, string(body))
		}
		if check.wantBody != "" && !bytes.Contains(body, []byte(check.wantBody)) {
			return fmt.Errorf("%s: body=%s missing %s", check.name, string(body), check.wantBody)
		}
		fmt.Printf("ok %-24s %d\n", check.name, resp.StatusCode)
	}
	if services.auditCount() == 0 {
		return errors.New("expected audit events")
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"auditEvents": services.auditCount(), "status": "PASS"}); err != nil {
		return err
	}
	return nil
}
