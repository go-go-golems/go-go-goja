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
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/devauth"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8788", "listen address for manual server mode")
	script := flag.String("script", "examples/xgoja/18-express-auth-host/scripts/server.js", "JavaScript route script")
	smoke := flag.Bool("smoke", false, "run an in-process smoke test instead of listening")
	flag.Parse()

	if err := run(context.Background(), *listen, *script, *smoke); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, listen, script string, smoke bool) error {
	kit := devauth.New(devauth.Config{})
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, RejectRawRoutes: true, Auth: kit.AuthOptions()})
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareOnly("timer")).
		WithModules(express.NewRegistrar(host)).
		Build()
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
	mux := buildMux(host, kit)
	if smoke {
		return runSmoke(mux, kit)
	}
	log.Printf("serving auth demo on http://%s", listen)
	log.Printf("login: curl -i -X POST -H 'Content-Type: application/json' -d '{\"username\":\"%s\",\"password\":\"%s\"}' http://%s/auth/dev/login", devauth.DefaultUsername, devauth.DefaultPassword, listen)
	log.Printf("then call protected routes with the returned cookie and X-CSRF-Token")
	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}

func buildMux(host http.Handler, kit *devauth.Kit) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /auth/dev/login", kit.LoginHandler())
	mux.Handle("POST /auth/dev/logout", kit.LogoutHandler())
	mux.Handle("GET /auth/dev/session", kit.SessionHandler())
	mux.Handle("/", host)
	return mux
}

func runSmoke(handler http.Handler, kit *devauth.Kit) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := server.Client()
	client.Jar = jar

	if err := checkRequest(client, server.URL, smokeCheck{name: "public health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK, wantBody: `"ok":true`}); err != nil {
		return err
	}
	if err := checkRequest(client, server.URL, smokeCheck{name: "me before login", method: http.MethodGet, path: "/me", wantStatus: http.StatusUnauthorized}); err != nil {
		return err
	}
	if err := login(client, server.URL, "wrong@example.test", "bad-password", http.StatusUnauthorized); err != nil {
		return err
	}
	csrf, err := loginAndCSRF(client, server.URL)
	if err != nil {
		return err
	}
	checks := []smokeCheck{
		{name: "async return", method: http.MethodGet, path: "/async-return?name=dev", wantStatus: http.StatusOK, wantBody: `async return dev`},
		{name: "async send", method: http.MethodGet, path: "/async-send?name=dev", wantStatus: http.StatusOK, wantBody: `"mode":"send"`},
		{name: "session after login", method: http.MethodGet, path: "/auth/dev/session", wantStatus: http.StatusOK, wantBody: `"id":"u1"`},
		{name: "me after login", method: http.MethodGet, path: "/me", wantStatus: http.StatusOK, wantBody: `"id":"u1"`},
		{name: "project missing csrf", method: http.MethodPatch, path: "/orgs/o1/projects/p1", wantStatus: http.StatusForbidden},
		{name: "project update", method: http.MethodPatch, path: "/orgs/o1/projects/p1", csrf: csrf, wantStatus: http.StatusOK, wantBody: `"updated":"p1"`},
		{name: "project missing", method: http.MethodPatch, path: "/orgs/o1/projects/missing", csrf: csrf, wantStatus: http.StatusNotFound},
		{name: "logout", method: http.MethodPost, path: "/auth/dev/logout", csrf: csrf, wantStatus: http.StatusNoContent},
		{name: "me after logout", method: http.MethodGet, path: "/me", wantStatus: http.StatusUnauthorized},
	}
	for _, check := range checks {
		if err := checkRequest(client, server.URL, check); err != nil {
			return err
		}
	}
	if kit.AuditCount() == 0 {
		return errors.New("expected audit events")
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"auditEvents": kit.AuditCount(), "status": "PASS"}); err != nil {
		return err
	}
	return nil
}

type smokeCheck struct {
	name       string
	method     string
	path       string
	csrf       string
	wantStatus int
	wantBody   string
}

func loginAndCSRF(client *http.Client, baseURL string) (string, error) {
	var response struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := loginInto(client, baseURL, devauth.DefaultUsername, devauth.DefaultPassword, http.StatusOK, &response); err != nil {
		return "", err
	}
	if response.CSRFToken == "" {
		return "", errors.New("login response missing csrfToken")
	}
	return response.CSRFToken, nil
}

func login(client *http.Client, baseURL, username, password string, wantStatus int) error {
	return loginInto(client, baseURL, username, password, wantStatus, nil)
}

func loginInto(client *http.Client, baseURL, username, password string, wantStatus int, out any) error {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/dev/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("login: read body: %w", err)
	}
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("login: status=%d body=%s want=%d", resp.StatusCode, string(data), wantStatus)
	}
	fmt.Printf("ok %-24s %d\n", loginName(wantStatus), resp.StatusCode)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("login: decode body: %w", err)
		}
	}
	return nil
}

func loginName(status int) string {
	if status == http.StatusOK {
		return "login"
	}
	return "bad login"
}

func checkRequest(client *http.Client, baseURL string, check smokeCheck) error {
	req, err := http.NewRequest(check.method, baseURL+check.path, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	if check.csrf != "" {
		req.Header.Set("X-CSRF-Token", check.csrf)
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
	return nil
}
