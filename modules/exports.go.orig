package modules

import (
	"log"

	"github.com/dop251/goja"
)

// SetExport attaches a Go value to module exports.
func SetExport(vm *goja.Runtime, exports *goja.Object, moduleName, name string, fn interface{}) {
	_ = vm
	if err := exports.Set(name, fn); err != nil {
		log.Printf("%s: failed to set %s function: %v", moduleName, name, err)
	}
}
