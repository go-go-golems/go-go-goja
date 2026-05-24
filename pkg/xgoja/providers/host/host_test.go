package host

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegisterHostProvider(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	for _, name := range []string{"fs", "node:fs", "exec", "database", "db"} {
		if _, ok := registry.ResolveModule(PackageID, name); !ok {
			t.Fatalf("expected host module %q", name)
		}
	}
}

func TestFSRequiresExplicitAllow(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "fs")
	if !ok {
		t.Fatal("expected fs module")
	}
	_, err := mod.New(providerapi.ModuleContext{Context: context.Background(), Name: "fs", As: "fs"})
	if err == nil || !strings.Contains(err.Error(), "config.allow=true") {
		t.Fatalf("expected allow error, got %v", err)
	}
}

func TestExecAllowedCommands(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "exec")
	if !ok {
		t.Fatal("expected exec module")
	}
	cfg, err := json.Marshal(ExecConfig{Allow: true, AllowedCommands: []string{"echo"}})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	loader, err := mod.New(providerapi.ModuleContext{Context: context.Background(), Name: "exec", As: "exec", Config: cfg})
	if err != nil {
		t.Fatalf("new exec loader: %v", err)
	}
	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	run, ok := goja.AssertFunction(exports.Get("run"))
	if !ok {
		t.Fatal("exec.run is not a function")
	}
	if _, err := run(goja.Undefined(), vm.ToValue("sh"), vm.ToValue([]string{"-c", "echo bad"})); err == nil {
		t.Fatal("expected disallowed command error")
	}
}
