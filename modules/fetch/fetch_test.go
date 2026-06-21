package fetch_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	_ "github.com/go-go-golems/go-go-goja/modules/fetch"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
)

func TestLowLevelFetchJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" || r.URL.Query().Get("name") != "goja" {
			t.Fatalf("request url = %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"goja"}`))
	}))
	defer server.Close()

	rt := newRuntime(t)
	url := strconv.Quote(server.URL + "/status?name=goja")
	_, err := rt.Owner.Call(context.Background(), "fetch.low-level.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			globalThis.__fetchSmoke = { done: false };
			(async () => {
				const fetch = require("fetch");
				const response = await fetch.fetch(` + url + `, { headers: { Accept: "application/json" } });
				const body = await response.json();
				globalThis.__fetchSmoke = { done: true, error: "", status: response.status, ok: response.ok, name: body.name };
			})().catch(e => { globalThis.__fetchSmoke = { done: true, error: String(e) }; });
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	state := requireFetchState(t, rt)
	for _, want := range []string{`"error":""`, `"status":200`, `"ok":true`, `"name":"goja"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("state missing %s: %s", want, state)
		}
	}
}

func TestClientBearerFromFile(t *testing.T) {
	const token = "secret-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.Path != "/agent/reports/daily" || r.URL.Query().Get("format") != "short" {
			t.Fatalf("request url = %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reportId":"daily","auth":"apiToken"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token.json")
	if err := os.WriteFile(tokenFile, []byte(`{"token":{"value":"`+token+`"}}`), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	rt := newRuntime(t)
	script := fmt.Sprintf(`
		globalThis.__fetchSmoke = { done: false };
		(async () => {
			const fetch = require("fetch");
			const client = fetch.client()
				.baseUrl(%s)
				.auth(fetch.auth.bearer().fromFile(%s).jsonPath("token.value"))
				.acceptJson()
				.expectJson();
			const body = await client.get("/agent/reports/daily").query("format", "short").run();
			globalThis.__fetchSmoke = { done: true, error: "", reportId: body.reportId, auth: body.auth };
		})().catch(e => { globalThis.__fetchSmoke = { done: true, error: String(e), status: e.status || 0 }; });
	`, strconv.Quote(server.URL), strconv.Quote(tokenFile))
	_, err := rt.Owner.Call(context.Background(), "fetch.client.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(script)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	state := requireFetchState(t, rt)
	for _, want := range []string{`"error":""`, `"reportId":"daily"`, `"auth":"apiToken"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("state missing %s: %s", want, state)
		}
	}
}

func TestClientAuthRejectsPlainObject(t *testing.T) {
	rt := newRuntime(t)
	ret, err := rt.Owner.Call(context.Background(), "fetch.auth-plain", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const fetch = require("fetch");
			let message = "";
			try { fetch.client().auth({ type: "bearer", token: "secret" }); }
			catch (e) { message = String(e); }
			message;
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(ret.(string), "fetch.auth") {
		t.Fatalf("error did not explain auth builder rejection: %s", ret.(string))
	}
	if strings.Contains(ret.(string), "secret") {
		t.Fatalf("error leaked token: %s", ret.(string))
	}
}

func newRuntime(t *testing.T) *engine.Runtime {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareOnly("fetch")).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}

func requireFetchState(t *testing.T, rt *engine.Runtime) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := readFetchState(t, rt)
		if strings.Contains(state, `"done":true`) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("fetch state not done: %s", readFetchState(t, rt))
	return ""
}

func readFetchState(t *testing.T, rt *engine.Runtime) string {
	t.Helper()
	ret, err := rt.Owner.Call(context.Background(), "fetch.state", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`JSON.stringify(globalThis.__fetchSmoke || { done: false })`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	return ret.(string)
}
