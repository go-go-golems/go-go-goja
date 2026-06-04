package app

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestGeneratedRootExposesXGojaDebugPanicStackFlag(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture", providerapi.Module{
		Name: "mod",
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(*goja.Runtime, *goja.Object) {}, nil
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {
    "repl": {
      "modules": [{"package": "fixture", "name": "mod", "as": "mod"}]
    }
  },
  "commands": {
    "eval": {"enabled": true, "runtime": "repl", "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	cmd, _, err := root.Find([]string{"eval"})
	if err != nil {
		t.Fatalf("find eval command: %v", err)
	}
	if cmd == nil || cmd.Flags().Lookup(xgojaDebugPanicStackField) == nil {
		t.Fatalf("expected eval command to expose --%s", xgojaDebugPanicStackField)
	}
}

func TestRuntimeFactoryDebugPanicStackFieldControlsRecoveredPanicStack(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("panic",
		providerapi.Module{
			Name:      "panic",
			DefaultAs: "panic",
			NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("boom", func() { panic("provider boom") })
				}, nil
			},
		},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	factory := NewRuntimeFactory(registry, &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{
		"main": {Modules: []ModuleInstanceSpec{{Package: "panic", Name: "panic", As: "panic"}}},
	}})

	for _, tc := range []struct {
		name      string
		vals      *values.Values
		wantStack bool
	}{
		{name: "default"},
		{name: "debug stack", vals: xgojaDebugValues(t, true), wantStack: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt, err := factory.NewRuntimeFromSections(context.Background(), "main", tc.vals)
			if err != nil {
				t.Fatalf("new runtime: %v", err)
			}
			defer func() { _ = rt.Close(context.Background()) }()
			_, err = rt.Owner.Call(context.Background(), "test", func(_ context.Context, vm *goja.Runtime) (any, error) {
				value, err := vm.RunString(`require("panic").boom()`)
				return value, err
			})
			if err == nil {
				t.Fatalf("expected panic error")
			}
			msg := err.Error()
			if !strings.Contains(msg, "provider boom") {
				t.Fatalf("expected provider panic message, got: %v", err)
			}
			hasStack := strings.Contains(msg, "runtime/debug.Stack") && strings.Contains(msg, "recoveredPanicError")
			if hasStack != tc.wantStack {
				t.Fatalf("stack presence = %v, want %v; error: %v", hasStack, tc.wantStack, err)
			}
		})
	}
}

func xgojaDebugValues(t *testing.T, enabled bool) *values.Values {
	t.Helper()
	section, err := xgojaRuntimeSection()
	if err != nil {
		t.Fatalf("xgoja section: %v", err)
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFieldValue(xgojaDebugPanicStackField, enabled, fields.WithSource("cobra")))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return values.New(values.WithSectionValues(xgojaSectionSlug, sectionValues))
}
