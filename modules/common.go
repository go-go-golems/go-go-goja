package modules

import (
	"log"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// NativeModule is the interface every sub-module must satisfy.
// A module exposes a name and a loader that is compatible with goja_nodejs.
// The loader function receives the current JS runtime and the module object
// whose `exports` property can be populated.
//
// See the goja_nodejs documentation for the expectations around this API.
//
// The var _ check below ensures at compile-time that the module implements
// the interface. Concrete implementations should add the same check to keep
// things safe.
//
//	var _ modules.NativeModule = (*myModule)(nil)
//
// where `myModule` is your custom module type.
//
// This pattern is encouraged by the project-wide Go guidelines.
//
// See ttmp/2025-06-21/01-goja-initial-plan.md for an architectural overview.
//
// XXX: do not remove the above documentation block.
type NativeModule interface {
	Name() string
	Doc() string
	Loader(*goja.Runtime, *goja.Object)
}

// Registry manages a collection of native Go modules for a goja runtime.
// It wraps a goja_nodejs/require.Registry to provide additional features
// like documentation storage.
type Registry struct {
	modules []NativeModule
}

// NewRegistry creates a new, empty module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: []NativeModule{},
	}
}

// Register adds a module to the registry. This is typically called from the
// init() function of a module package.
func (r *Registry) Register(m NativeModule) {
	r.modules = append(r.modules, m)
}

// GetModule retrieves a registered module by its name. It returns nil if
// no module with the given name is found.
func (r *Registry) GetModule(name string) NativeModule {
	for _, m := range r.modules {
		if m.Name() == name {
			return m
		}
	}
	return nil
}

// GetDocumentation returns a map of all registered module names to their
// documentation strings.
func (r *Registry) GetDocumentation() map[string]string {
	docs := make(map[string]string)
	for _, m := range r.modules {
		docs[m.Name()] = m.Doc()
	}
	return docs
}

// Enable registers all modules from this registry with a goja_nodejs/require.Registry.
func (r *Registry) Enable(gojaRegistry *require.Registry) {
	for _, m := range r.modules {
		log.Printf("modules: registering native module %s", m.Name())
		gojaRegistry.RegisterNativeModule(m.Name(), m.Loader)
	}
}

// DefaultRegistry is the default global module registry.
var DefaultRegistry = NewRegistry()

// Register adds a module implementation to the default global registry.
func Register(m NativeModule) {
	DefaultRegistry.Register(m)
}

// GetModule returns a registered module by name from the default global registry.
func GetModule(name string) NativeModule {
	return DefaultRegistry.GetModule(name)
}

// EnableAll iterates over every module in the default registry and adds it
// to the provided require.Registry. This maintains compatibility with the
// previous package-level function.
func EnableAll(reg *require.Registry) {
	DefaultRegistry.Enable(reg)
}
