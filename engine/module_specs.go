package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/process"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// ModuleSpec is a static registration unit applied at factory build time.
type ModuleSpec interface {
	ID() string
	Register(reg *require.Registry) error
}

// RuntimeInitializer is a per-runtime initialization hook executed after VM and
// require setup.
type RuntimeInitializer interface {
	ID() string
	InitRuntime(ctx *RuntimeContext) error
}

// RuntimeContext exposes runtime-scoped objects to initializers.
type RuntimeContext struct {
	Context context.Context
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner
	Values  map[string]any
}

func (ctx *RuntimeContext) SetValue(key string, value any) {
	if ctx == nil || key == "" {
		return
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	ctx.Values[key] = value
}

func (ctx *RuntimeContext) Value(key string) (any, bool) {
	if ctx == nil || ctx.Values == nil || key == "" {
		return nil, false
	}
	value, ok := ctx.Values[key]
	return value, ok
}

// NativeModuleSpec registers a single native module loader.
type NativeModuleSpec struct {
	ModuleID   string
	ModuleName string
	Loader     require.ModuleLoader
}

func (s NativeModuleSpec) ID() string {
	if strings.TrimSpace(s.ModuleID) != "" {
		return strings.TrimSpace(s.ModuleID)
	}
	return "native:" + strings.TrimSpace(s.ModuleName)
}

func (s NativeModuleSpec) Register(reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	name := strings.TrimSpace(s.ModuleName)
	if name == "" {
		return fmt.Errorf("module name is empty")
	}
	if s.Loader == nil {
		return fmt.Errorf("native module %q loader is nil", name)
	}
	reg.RegisterNativeModule(name, s.Loader)
	return nil
}

type defaultRegistryModulesSpec struct{}

func (s defaultRegistryModulesSpec) ID() string {
	return "default-registry-modules"
}

func (s defaultRegistryModulesSpec) Register(reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	modules.EnableAll(reg)
	return nil
}

// DefaultRegistryModules returns a ModuleSpec that registers every module from
// go-go-goja/modules.DefaultRegistry. Prefer DefaultRegistryModule(name) or
// DefaultRegistryModulesNamed(...) when embedding untrusted or semi-trusted
// JavaScript, because the full default registry includes host-access modules
// such as fs, os, exec, and database.
func DefaultRegistryModules() ModuleSpec {
	return defaultRegistryModulesSpec{}
}

type namedDefaultRegistryModulesSpec struct {
	id    string
	names []string
}

func (s namedDefaultRegistryModulesSpec) ID() string {
	if strings.TrimSpace(s.id) != "" {
		return strings.TrimSpace(s.id)
	}
	return "default-registry-modules:" + strings.Join(s.names, ",")
}

func (s namedDefaultRegistryModulesSpec) Register(reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	for _, rawName := range s.names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			return fmt.Errorf("default registry module name is empty")
		}
		mod := modules.GetModule(name)
		if mod == nil {
			return fmt.Errorf("default registry module %q is not registered", name)
		}
		reg.RegisterNativeModule(mod.Name(), mod.Loader)
	}
	return nil
}

// DefaultRegistryModule returns a ModuleSpec that registers one module from
// modules.DefaultRegistry by its JavaScript require() name.
func DefaultRegistryModule(name string) ModuleSpec {
	name = strings.TrimSpace(name)
	return namedDefaultRegistryModulesSpec{
		id:    "default-registry-module:" + name,
		names: []string{name},
	}
}

// DefaultRegistryModulesNamed returns a ModuleSpec that registers only the
// named modules from modules.DefaultRegistry. Use this for granular sandbox
// composition instead of DefaultRegistryModules().
func DefaultRegistryModulesNamed(names ...string) ModuleSpec {
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			trimmed = append(trimmed, strings.TrimSpace(name))
		}
	}
	return namedDefaultRegistryModulesSpec{
		id:    "default-registry-modules:" + strings.Join(trimmed, ","),
		names: trimmed,
	}
}

var dataOnlyDefaultRegistryModuleNames = []string{"crypto", "path", "time", "timer"}

// DataOnlyDefaultRegistryModules returns the non-host-filesystem/non-process
// primitives that are installed automatically for every engine runtime.
func DataOnlyDefaultRegistryModules() ModuleSpec {
	return namedDefaultRegistryModulesSpec{
		id:    "data-only-default-registry-modules",
		names: append([]string(nil), dataOnlyDefaultRegistryModuleNames...),
	}
}

// DataOnlyDefaultRegistryModuleNames returns a copy of the module names that
// are installed automatically for every engine runtime.
func DataOnlyDefaultRegistryModuleNames() []string {
	return append([]string(nil), dataOnlyDefaultRegistryModuleNames...)
}

type processEnvInitializer struct{}

func (p processEnvInitializer) ID() string {
	return "process-env"
}

func (p processEnvInitializer) InitRuntime(ctx *RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return fmt.Errorf("runtime context or VM is nil")
	}
	process.Enable(ctx.VM)
	return nil
}

// ProcessEnv returns a runtime initializer that installs the global process
// object. It is opt-in because goja_nodejs/process exposes the host
// environment via process.env.
func ProcessEnv() RuntimeInitializer {
	return processEnvInitializer{}
}
