package express

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/uidsl"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestExpressPlannedBuilderSupportsRateLimit(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true, Renderer: uidsl.RenderAny, Auth: gojahttp.AuthOptions{RateLimiter: gojahttp.NewMemoryRateLimiter()}})
	rt := newExpressAuthRuntime(t, host)
	runExpressAuthScript(t, rt, `
		const express = require("express");
		const app = express.app();
		app.get("/limited")
		  .public()
		  .rateLimit(express.rateLimit("public.limited").perMinute(1).byIP().byRoute())
		  .handle((_ctx, res) => res.json({ ok: true }));
	`)
	routes := host.Routes()
	if len(routes) != 1 || routes[0].RateLimitPolicies != "public.limited" {
		t.Fatalf("route descriptor = %#v", routes)
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "192.0.2.50:1234"
		rr := httptest.NewRecorder()
		host.ServeHTTP(rr, req)
		if i == 0 && rr.Code != http.StatusOK {
			t.Fatalf("first status=%d body=%s", rr.Code, rr.Body.String())
		}
		if i == 1 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("second status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestExpressRateLimitRejectsPlainObject(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	rt := newExpressAuthRuntime(t, host)
	_, err := rt.Owner.Call(context.Background(), "load-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const app = express.app();
			app.get("/bad")
			  .public()
			  .rateLimit({ policy: "bad", limit: 1 })
			  .handle(() => "bad");
		`)
		return nil, err
	})
	if err == nil {
		t.Fatal("expected plain rate limit object to be rejected")
	}
	if !strings.Contains(err.Error(), "express.rateLimit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
