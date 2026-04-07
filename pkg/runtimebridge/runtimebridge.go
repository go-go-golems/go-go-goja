package runtimebridge

import (
	"context"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// Bindings exposes runtime-owned scheduling primitives for modules that need
// async owner-thread settlement.
type Bindings struct {
	Context context.Context
	Loop    *eventloop.EventLoop
	Owner   runtimeowner.Runner
}

var bindingsByVM sync.Map

// Store registers runtime bindings for a concrete VM.
func Store(vm *goja.Runtime, bindings Bindings) {
	if vm == nil {
		return
	}
	bindingsByVM.Store(vm, bindings)
}

// Lookup returns the bindings registered for a concrete VM.
func Lookup(vm *goja.Runtime) (Bindings, bool) {
	if vm == nil {
		return Bindings{}, false
	}
	value, ok := bindingsByVM.Load(vm)
	if !ok {
		return Bindings{}, false
	}
	bindings, ok := value.(Bindings)
	if !ok {
		return Bindings{}, false
	}
	return bindings, true
}

// Delete removes runtime bindings for a concrete VM.
func Delete(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	bindingsByVM.Delete(vm)
}
