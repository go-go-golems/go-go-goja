package gojahttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dop251/goja"
)

func TestAttachHTTPHandlerHiddenRef(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("mounted"))
	})
	if err := AttachHTTPHandler(vm, obj, handler); err != nil {
		t.Fatalf("attach handler: %v", err)
	}
	extracted, ok := HTTPHandlerFromValue(obj)
	if !ok {
		t.Fatalf("expected handler ref to extract")
	}
	rr := httptest.NewRecorder()
	extracted.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Body.String() != "mounted" {
		t.Fatalf("handler body = %q", rr.Body.String())
	}
	if err := vm.Set("obj", obj); err != nil {
		t.Fatalf("set obj: %v", err)
	}
	value, err := vm.RunString(`JSON.stringify(Object.keys(obj))`)
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if value.String() != "[]" {
		t.Fatalf("hidden handler key should not be enumerable, keys=%s", value.String())
	}
}

func TestHTTPHandlerFromValueRejectsPlainObject(t *testing.T) {
	vm := goja.New()
	if handler, ok := HTTPHandlerFromValue(vm.NewObject()); ok || handler != nil {
		t.Fatalf("plain object unexpectedly extracted handler")
	}
	if handler, ok := HTTPHandlerFromValue(goja.Undefined()); ok || handler != nil {
		t.Fatalf("undefined unexpectedly extracted handler")
	}
}
