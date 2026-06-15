package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type demoAuth struct{}

func (demoAuth) Authenticate(_ context.Context, req *http.Request, _ *gojahttp.SessionDTO, _ gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	userID := strings.TrimSpace(req.Header.Get("X-Demo-User"))
	if userID == "" {
		return nil, gojahttp.ErrUnauthenticated
	}
	return &gojahttp.Actor{ID: userID, Kind: "user", TenantIDs: []string{"demo"}}, nil
}

func (demoAuth) ResolveResource(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	if req.ID == "missing" {
		return nil, gojahttp.ErrNotFound
	}
	return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID, TenantID: "demo"}, nil
}

func (demoAuth) Authorize(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	if req.Actor == nil {
		return gojahttp.AuthorizationDecision{}, gojahttp.ErrUnauthenticated
	}
	if req.Action == "project.read" && req.Resource != nil && req.Resource.ID != "forbidden" {
		return gojahttp.AuthorizationDecision{Allowed: true}, nil
	}
	return gojahttp.AuthorizationDecision{Allowed: false, Reason: "demo policy denied the request"}, nil
}

func newHandler() (http.Handler, error) {
	auth := demoAuth{}
	host := gojahttp.NewHost(gojahttp.HostOptions{
		Dev: true,
		Auth: gojahttp.AuthOptions{
			Authenticator: auth,
			Resources:     auth,
			Authorizer:    auth,
		},
	})
	app := gojahttp.NewApp(host)
	if err := app.Get("/healthz").Public().HandleJSON(func(context.Context, *gojahttp.SecureContext) (any, error) {
		return map[string]bool{"ok": true}, nil
	}); err != nil {
		return nil, err
	}
	if err := app.Get("/projects/:projectID").
		Auth(gojahttp.User().Required()).
		Resource(gojahttp.Resource("project").IDFromParam("projectID").MustExist()).
		Audit("project.read").
		Allow("project.read").
		HandleJSON(projectResponse); err != nil {
		return nil, err
	}

	middlewareRoute, err := gojahttp.PlannedMiddleware(gojahttp.MiddlewareOptions{
		Auth: gojahttp.AuthOptions{
			Authenticator: auth,
			Resources:     auth,
			Authorizer:    auth,
		},
		ParamFunc: func(r *http.Request, name string) string { return r.PathValue(name) },
	}, gojahttp.RoutePlan{
		Method:   http.MethodGet,
		Pattern:  "/middleware/projects/:projectID",
		Security: gojahttp.User().Required(),
		Resources: []gojahttp.ResourceSpec{
			gojahttp.Resource("project").IDFromParam("projectID").MustExist(),
		},
		Action: "project.read",
		Audit:  gojahttp.AuditSpec{Event: "project.read.middleware"},
	}, func(ctx context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, _ *http.Request) error {
		value, err := projectResponse(ctx, sec)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(value)
	})
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/", host)
	mux.Handle("GET /middleware/projects/{projectID}", middlewareRoute)
	return mux, nil
}

func projectResponse(_ context.Context, sec *gojahttp.SecureContext) (any, error) {
	return map[string]any{
		"actor":   sec.Actor.ID,
		"project": sec.Resource.ID,
		"action":  sec.Plan.Action,
	}, nil
}

func main() {
	listen := flag.String("listen", "127.0.0.1:18810", "HTTP listen address")
	smoke := flag.Bool("smoke", false, "run smoke checks and exit")
	flag.Parse()

	handler, err := newHandler()
	if err != nil {
		log.Fatal(err)
	}
	if *smoke {
		if err := runSmoke(handler); err != nil {
			log.Fatal(err)
		}
		fmt.Println("smoke ok")
		return
	}
	log.Printf("listening on http://%s", *listen)
	log.Fatal(http.ListenAndServe(*listen, handler))
}

func runSmoke(handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	checks := []struct {
		name   string
		path   string
		user   string
		status int
		body   string
	}{
		{name: "public", path: "/healthz", status: http.StatusOK, body: `"ok":true`},
		{name: "unauthenticated", path: "/projects/p1", status: http.StatusUnauthorized},
		{name: "host route", path: "/projects/p1", user: "alice", status: http.StatusOK, body: `"project":"p1"`},
		{name: "middleware route", path: "/middleware/projects/p2", user: "alice", status: http.StatusOK, body: `"project":"p2"`},
		{name: "forbidden", path: "/projects/forbidden", user: "alice", status: http.StatusForbidden},
	}
	for _, check := range checks {
		req, err := http.NewRequest(http.MethodGet, server.URL+check.path, nil)
		if err != nil {
			return err
		}
		if check.user != "" {
			req.Header.Set("X-Demo-User", check.user)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		body, readErr := io.ReadAll(res.Body)
		closeErr := res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("%s: read body: %w", check.name, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("%s: close body: %w", check.name, closeErr)
		}
		if res.StatusCode != check.status {
			return fmt.Errorf("%s: status=%d body=%s", check.name, res.StatusCode, string(body))
		}
		if check.body != "" && !strings.Contains(string(body), check.body) {
			return fmt.Errorf("%s: body %q does not contain %q", check.name, string(body), check.body)
		}
	}
	return nil
}
