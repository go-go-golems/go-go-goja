package engine

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

func TestNodeJSPrimitivesDefaultGlobalsAndRequires(t *testing.T) {
	factory, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "buffer global", code: `Buffer.from("abc").toString()`, want: "abc"},
		{name: "url global", code: `new URL("https://example.com/path").hostname`, want: "example.com"},
		{name: "url search params global", code: `new URLSearchParams("a=1").get("a")`, want: "1"},
		{name: "require buffer", code: `require("buffer").Buffer.from("xyz").toString()`, want: "xyz"},
		{name: "require url", code: `require("url").URL === URL ? "yes" : "no"`, want: "yes"},
		{name: "require util", code: `require("util").format("%s:%d", "x", 3)`, want: "x:3"},
		{name: "require process", code: `typeof require("process").env`, want: "object"},
		{name: "process global absent by default", code: `typeof process`, want: "undefined"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ret, err := rt.Owner.Call(context.Background(), "nodejs-primitives", func(_ context.Context, vm *goja.Runtime) (any, error) {
				value, runErr := vm.RunString(tc.code)
				if runErr != nil {
					return nil, runErr
				}
				return value.String(), nil
			})
			if err != nil {
				t.Fatalf("run %s: %v", tc.code, err)
			}
			if ret != tc.want {
				t.Fatalf("%s = %v, want %q", tc.code, ret, tc.want)
			}
		})
	}
}

func TestProcessEnvInitializerInstallsProcessGlobal(t *testing.T) {
	factory, err := NewBuilder().WithRuntimeInitializers(ProcessEnv()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "process-env", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`typeof process === "object" && typeof process.env === "object" ? "ok" : "missing"`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run process env smoke: %v", err)
	}
	if ret != "ok" {
		t.Fatalf("process global status = %v, want ok", ret)
	}
}
