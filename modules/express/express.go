package express

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type Option func(*Registrar)

type Registrar struct {
	host *gojahttp.Host
	name string
}

func NewRegistrar(host *gojahttp.Host, opts ...Option) *Registrar {
	r := &Registrar{host: host, name: "express"}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func WithName(name string) Option {
	return func(r *Registrar) {
		if r != nil && name != "" {
			r.name = name
		}
	}
}

func (r *Registrar) ID() string { return "express-http" }

func (r *Registrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	if r.host == nil {
		return fmt.Errorf("express registrar requires host")
	}
	r.host.SetRuntime(ctx.Owner)
	name := r.name
	if name == "" {
		name = "express"
	}
	reg.RegisterNativeModule(name, r.loader)
	return nil
}

func (r *Registrar) loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	_ = exports.Set("app", func() goja.Value { return r.appObject(vm) })
}

func (r *Registrar) appObject(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	for _, method := range []string{"get", "post", "put", "patch", "delete", "all"} {
		method := method
		_ = obj.Set(method, func(pattern string, handler goja.Value) error {
			fn, ok := goja.AssertFunction(handler)
			if !ok {
				return fmt.Errorf("app.%s(%q) requires a function handler", method, pattern)
			}
			r.host.Register(strings.ToUpper(method), pattern, fn)
			return nil
		})
	}
	_ = obj.Set("static", func(prefix, dir string) error {
		if prefix == "" || dir == "" {
			return fmt.Errorf("app.static requires prefix and directory")
		}
		r.host.RegisterStatic(prefix, dir)
		return nil
	})
	return obj
}
