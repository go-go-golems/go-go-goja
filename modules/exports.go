package modules

import (
	"log"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/calllog"
)

// SetExport attaches a Go function to module exports with JS->Go call logging.
func SetExport(vm *goja.Runtime, exports *goja.Object, moduleName, name string, fn interface{}) {
	wrapped := calllog.WrapGoFunction(vm, moduleName, name, fn)
	if err := exports.Set(name, wrapped); err != nil {
		log.Printf("%s: failed to set %s function: %v", moduleName, name, err)
	}
}
