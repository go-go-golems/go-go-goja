package modules

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"log"
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
	Loader(*goja.Runtime, *goja.Object)
}

// all stores every module that registers itself in its package init().
var all []NativeModule

// Register adds a module implementation to the internal list so that
// EnableAll can later wire every module into a require.Registry.
//
// This function is intended to be called from an `init()` function inside
// each individual module package:
//
//	func init() { modules.Register(&myModule{}) }
func Register(m NativeModule) { // exposed so sub-packages can call it
	all = append(all, m)
}

// EnableAll iterates over every module that has called Register and adds it
// to the provided require.Registry so that it becomes available from JS via
// the standard Node-style `require()` mechanism.
func EnableAll(reg *require.Registry) {
	for _, m := range all {
		log.Printf("modules: registering native module %s", m.Name())
		reg.RegisterNativeModule(m.Name(), m.Loader)
	}
}
