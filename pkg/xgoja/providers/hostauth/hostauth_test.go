package hostauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	hostauthsvc "github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegisterHostAuthProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "auth")
	if !ok {
		t.Fatal("expected auth module")
	}
	if mod.DefaultAs != "auth" {
		t.Fatalf("default alias = %q, want auth", mod.DefaultAs)
	}
}

func TestAuthModuleRequiresHostAuthServices(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "auth")
	if !ok {
		t.Fatal("expected auth module")
	}
	_, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "auth", As: "auth"})
	if err == nil || !strings.Contains(err.Error(), "requires hostauth services") {
		t.Fatalf("expected missing hostauth services error, got %v", err)
	}
}

func TestAuthAuditQueryFromJavaScript(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	store := &audit.MemoryStore{}
	now := time.Date(2026, 6, 17, 13, 0, 0, 0, time.UTC)
	for _, record := range []audit.Record{
		{Event: "old denied", Outcome: "denied", TenantID: "o1", ActorID: "u1", ResourceType: "project", ResourceID: "p1", CreatedAt: now},
		{Event: "new denied", Outcome: "denied", TenantID: "o1", ActorID: "u2", ResourceType: "project", ResourceID: "p2", CreatedAt: now.Add(time.Second)},
		{Event: "other tenant", Outcome: "denied", TenantID: "o2", ActorID: "u3", ResourceType: "project", ResourceID: "p3", CreatedAt: now.Add(2 * time.Second)},
	} {
		if err := store.InsertAuditRecord(context.Background(), record); err != nil {
			t.Fatalf("insert %s: %v", record.Event, err)
		}
	}

	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(hostauthsvc.ServicesKey, &hostauthsvc.Services{AuditStore: store}); err != nil {
		t.Fatalf("set hostauth services: %v", err)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{
		Provider: PackageID,
		Name:     "auth",
		As:       "auth",
		Config: map[string]any{
			"audit": map[string]any{"maxLimit": 1},
		},
	}}}}
	factory := app.NewRuntimeFactory(registry, runtimePlan, hostServices)
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "hostauth.test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const auth = require("auth");
			const records = auth.audit.query({ tenantId: "o1", outcome: "denied", limit: 99 });
			JSON.stringify({ count: records.length, event: records[0].event, tenantId: records[0].tenantId, resourceId: records[0].resourceId });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run auth module: %v", err)
	}
	state := ret.(string)
	for _, want := range []string{`"count":1`, `"event":"new denied"`, `"tenantId":"o1"`, `"resourceId":"p2"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("state missing %s: %s", want, state)
		}
	}
}
