package gojahttp

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

const hiddenHTTPHandlerKey = "__go_go_goja_http_handler"

// HandlerRef is the shared cross-module ABI for JavaScript-visible values that
// carry a Go http.Handler. Producers attach it to an object with AttachHTTPHandler;
// HTTP modules unwrap it with HTTPHandlerFromValue.
type HandlerRef struct {
	Handler http.Handler
}

// AttachHTTPHandler attaches handler as a hidden, non-enumerable Go reference on
// obj. The JavaScript object remains an ordinary object publicly, while Go-backed
// HTTP modules can recover the handler through HTTPHandlerFromValue.
func AttachHTTPHandler(vm *goja.Runtime, obj *goja.Object, handler http.Handler) error {
	if vm == nil {
		return fmt.Errorf("gojahttp: nil runtime")
	}
	if obj == nil {
		return fmt.Errorf("gojahttp: nil object")
	}
	if handler == nil {
		return fmt.Errorf("gojahttp: nil handler")
	}
	ref := &HandlerRef{Handler: handler}
	value := vm.ToValue(ref)
	if err := obj.Set(hiddenHTTPHandlerKey, value); err != nil {
		return fmt.Errorf("gojahttp: attach handler ref: %w", err)
	}
	return obj.DefineDataProperty(hiddenHTTPHandlerKey, value, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
}

// HTTPHandlerFromValue extracts a Go http.Handler hidden on a JavaScript value.
func HTTPHandlerFromValue(value goja.Value) (http.Handler, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, false
	}
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, false
	}
	raw := obj.Get(hiddenHTTPHandlerKey)
	if raw == nil || goja.IsUndefined(raw) || goja.IsNull(raw) {
		return nil, false
	}
	ref, ok := raw.Export().(*HandlerRef)
	if !ok || ref == nil || ref.Handler == nil {
		return nil, false
	}
	return ref.Handler, true
}
