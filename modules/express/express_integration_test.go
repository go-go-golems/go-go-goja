package express

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules/uidsl"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestExpressRouteReturnsHTMLNode(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	factory, err := engine.NewBuilder().WithRuntimeModuleRegistrars(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const ui = require("ui.dsl");
			const app = express.app();
			app.get("/hello/:name", (req, res) => ui.h1("Hello " + req.params.name));
		`)
		return nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/hello/Goja", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%s", ct)
	}
	if !strings.Contains(rr.Body.String(), "<h1>Hello Goja</h1>") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressPostJSONEcho(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	factory, err := engine.NewBuilder().WithRuntimeModuleRegistrars(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.post("/echo", (req, res) => res.status(201).json({ title: req.body.title }));
		`)
		return nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"title":"Card"}`))
	req.Header.Set("Content-Type", "application/json")
	host.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"title":"Card"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressRouteAwaitsReturnedPromise(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	factory, err := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("timer")).WithRuntimeModuleRegistrars(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const ui = require("ui.dsl");
			const timer = require("timer");
			const app = express.app();
			app.get("/async-return", async (req, res) => {
			  await timer.sleep(5);
			  return ui.h1("async " + req.query.name);
			});
		`)
		return nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	buf := httptest.NewRecorder()
	host.ServeHTTP(buf, httptest.NewRequest(http.MethodGet, "/async-return?name=route", nil))
	if buf.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", buf.Code, buf.Body.String())
	}
	if !strings.Contains(buf.Body.String(), "<h1>async route</h1>") {
		t.Fatalf("body=%s", buf.Body.String())
	}
	if strings.Contains(buf.Body.String(), "Promise") {
		t.Fatalf("promise was rendered instead of awaited: %s", buf.Body.String())
	}
}

func TestExpressRouteAwaitsPromiseThatSendsResponse(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	factory, err := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("timer")).WithRuntimeModuleRegistrars(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const timer = require("timer");
			const app = express.app();
			app.get("/async-send", async (req, res) => {
			  await timer.sleep(5);
			  res.json({ ok: true, name: req.query.name });
			});
		`)
		return nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/async-send?name=response", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) || !strings.Contains(rr.Body.String(), `"name":"response"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHeadFallsBackToGetWithoutBody(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny})
	factory, err := engine.NewBuilder().WithRuntimeModuleRegistrars(NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)
	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/hello", (req, res) => res.type("text/plain").send("hello body"));
		`)
		return nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodHead, "/hello", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty HEAD body, got %q", rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("content-type=%s", ct)
	}
}
