package engine

import (
	"fmt"
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
	VM      *goja.Runtime
	Require *require.RequireModule
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner
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
// go-go-goja/modules.DefaultRegistry. This is explicit and opt-in.
func DefaultRegistryModules() ModuleSpec {
	return defaultRegistryModulesSpec{}
}
