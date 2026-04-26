package engine

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
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
	for _, name := range expandDefaultRegistryModuleNames(s.names) {
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

var defaultRegistryModuleAliases = map[string][]string{
	"crypto": {"node:crypto"},
	"events": {"node:events"},
	"fs":     {"node:fs"},
	"os":     {"node:os"},
	"path":   {"node:path"},
}

func expandDefaultRegistryModuleNames(names []string) []string {
	ret := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	add := func(rawName string) {
		name := strings.TrimSpace(rawName)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		ret = append(ret, name)
	}
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		add(name)
		for _, alias := range defaultRegistryModuleAliases[name] {
			add(alias)
		}
	}
	return ret
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

var dataOnlyDefaultRegistryModuleNames = []string{"crypto", "node:crypto", "events", "node:events", "path", "node:path", "time", "timer"}

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

func processObject(vm *goja.Runtime) *goja.Object {
	process := vm.NewObject()
	env := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	_ = process.Set("env", env)
	return process
}

func processModuleLoader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	_ = exports.Set("env", processObject(vm).Get("env"))
}

type processModuleSpec struct{}

func (processModuleSpec) ID() string { return "native:process" }

func (processModuleSpec) Register(reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	reg.RegisterNativeModule("process", processModuleLoader)
	reg.RegisterNativeModule("node:process", processModuleLoader)
	return nil
}

// ProcessModule returns a ModuleSpec that registers require("process") and
// require("node:process") for this runtime factory only. It is opt-in because
// process.env exposes host environment variables.
func ProcessModule() ModuleSpec {
	return processModuleSpec{}
}

type processEnvInitializer struct{}

func (p processEnvInitializer) ID() string {
	return "process-env"
}

func (p processEnvInitializer) InitRuntime(ctx *RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return fmt.Errorf("runtime context or VM is nil")
	}
	if ctx.Require != nil {
		if processValue, err := ctx.Require.Require("process"); err == nil {
			return ctx.VM.Set("process", processValue)
		}
	}
	return ctx.VM.Set("process", processObject(ctx.VM))
}

// ProcessEnv returns a runtime initializer that installs the global process
// object. It is opt-in because process.env exposes host environment variables.
// Use ProcessModule() as well when scripts should also be able to call
// require("process").
func ProcessEnv() RuntimeInitializer {
	return processEnvInitializer{}
}
