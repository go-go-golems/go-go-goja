package gojahttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/modules/uidsl"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)

func TestSessionCookieIssuedAndReused(t *testing.T) {
	const cookieName = "test_session"
	host := gojahttp.NewHost(gojahttp.HostOptions{
		Dev:      true,
		Renderer: uidsl.RenderAny,
		Sessions: gojahttp.SessionOptions{CookieName: cookieName},
	})
	factory, err := engine.NewBuilder().WithModules(express.NewRegistrar(host), uidsl.NewRegistrar()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	host.SetRuntime(rt.Owner)

	_, err = rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/session", (req, res) => res.json({ id: req.session.id, isNew: req.session.isNew }));
		`)
		return nil, err
	})
	if err != nil {
		t.Fatalf("load script: %v", err)
	}

	first := httptest.NewRecorder()
	host.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/session", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookieName {
		t.Fatalf("expected %s cookie, got %#v", cookieName, cookies)
	}
	if !sessionIDPattern.MatchString(cookies[0].Value) {
		t.Fatalf("invalid session id %q", cookies[0].Value)
	}
	if !strings.Contains(first.Body.String(), `"isNew":true`) || !strings.Contains(first.Body.String(), cookies[0].Value) {
		t.Fatalf("first response missing new session: %s", first.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/session", nil)
	secondReq.AddCookie(cookies[0])
	second := httptest.NewRecorder()
	host.ServeHTTP(second, secondReq)
	if second.Code != http.StatusOK {
		t.Fatalf("second status=%d body=%s", second.Code, second.Body.String())
	}
	if strings.Contains(second.Header().Get("Set-Cookie"), cookieName) {
		t.Fatalf("did not expect replacement session cookie: %s", second.Header().Get("Set-Cookie"))
	}
	if !strings.Contains(second.Body.String(), `"isNew":false`) || !strings.Contains(second.Body.String(), cookies[0].Value) {
		t.Fatalf("second response missing reused session: %s", second.Body.String())
	}
}
