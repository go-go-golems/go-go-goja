package engine

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

func TestDataOnlyModulesAreEnabledByDefault(t *testing.T) {
	factory, err := NewBuilder().UseModuleMiddleware(MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	tests := map[string]string{
		`require("path").join("a", "b")`:                                          "a/b",
		`require("node:path").join("a", "b")`:                                     "a/b",
		`typeof require("time").now()`:                                            "number",
		`require("crypto").createHash("sha256").update("abc").digest("hex")`:      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		`require("node:crypto").createHash("sha256").update("abc").digest("hex")`: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		`typeof require("events").EventEmitter`:                                   "function",
		`typeof require("node:events").EventEmitter`:                              "function",
		`typeof require("timer").sleep`:                                           "function",
	}
	for code, want := range tests {
		ret, err := rt.Owner.Call(context.Background(), "data-only-default", func(_ context.Context, vm *goja.Runtime) (any, error) {
			value, runErr := vm.RunString(code)
			if runErr != nil {
				return nil, runErr
			}
			return value.String(), nil
		})
		if err != nil {
			t.Fatalf("run %s: %v", code, err)
		}
		if ret != want {
			t.Fatalf("%s = %v, want %q", code, ret, want)
		}
	}
}

func TestDefaultBuilderEnablesAllDefaultRegistryModules(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "default-modules-present", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			function canRequire(name) {
				try { require(name); return true; } catch (e) { return false; }
			}
			JSON.stringify({ fs: canRequire("fs"), nodeFs: canRequire("node:fs"), os: canRequire("os"), nodeOs: canRequire("node:os"), exec: canRequire("exec"), database: canRequire("database"), db: canRequire("db") });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run default module smoke: %v", err)
	}
	if ret != `{"fs":true,"nodeFs":true,"os":true,"nodeOs":true,"exec":true,"database":true,"db":true}` {
		t.Fatalf("default module availability = %v", ret)
	}
}

func TestSafeMiddlewareRestrictsHostAccessModules(t *testing.T) {
	factory, err := NewBuilder().UseModuleMiddleware(MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "host-modules-absent", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			function canRequire(name) {
				try { require(name); return true; } catch (e) { return false; }
			}
			JSON.stringify({ fs: canRequire("fs"), nodeFs: canRequire("node:fs"), os: canRequire("os"), nodeOs: canRequire("node:os"), exec: canRequire("exec"), database: canRequire("database"), db: canRequire("db") });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run host absent smoke: %v", err)
	}
	if ret != `{"fs":false,"nodeFs":false,"os":false,"nodeOs":false,"exec":false,"database":false,"db":false}` {
		t.Fatalf("host module availability = %v", ret)
	}
}

//nolint:staticcheck // Tests the deprecated DefaultRegistryModule function directly.
func TestDefaultRegistryModuleEnablesOneHostModule(t *testing.T) {
	factory, err := NewBuilder().WithModules(DefaultRegistryModule("fs")).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "single-module", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			function canRequire(name) {
				try { require(name); return true; } catch (e) { return false; }
			}
			JSON.stringify({ fs: canRequire("fs"), nodeFs: canRequire("node:fs"), os: canRequire("os"), nodeOs: canRequire("node:os") });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run single-module smoke: %v", err)
	}
	if ret != `{"fs":true,"nodeFs":true,"os":false,"nodeOs":false}` {
		t.Fatalf("single module availability = %v", ret)
	}
}

//nolint:staticcheck // Tests the deprecated DefaultRegistryModulesNamed function directly.
func TestDefaultRegistryModulesNamedEnablesSelectedHostModules(t *testing.T) {
	factory, err := NewBuilder().WithModules(DefaultRegistryModulesNamed("fs", "os")).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "named-modules", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			function canRequire(name) {
				try { require(name); return true; } catch (e) { return false; }
			}
			JSON.stringify({ fs: canRequire("fs"), nodeFs: canRequire("node:fs"), os: canRequire("os"), nodeOs: canRequire("node:os"), exec: canRequire("exec") });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run named-modules smoke: %v", err)
	}
	if ret != `{"fs":true,"nodeFs":true,"os":true,"nodeOs":true,"exec":false}` {
		t.Fatalf("named module availability = %v", ret)
	}
}
