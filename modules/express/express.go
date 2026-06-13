package express

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type Option func(*Registrar)

type StartFunc func(*goja.Runtime) error

type Registrar struct {
	host  *gojahttp.Host
	name  string
	onUse StartFunc
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

func WithOnUse(fn StartFunc) Option {
	return func(r *Registrar) {
		if r != nil {
			r.onUse = fn
		}
	}
}

func (r *Registrar) ID() string { return "express-http" }

func (r *Registrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleRegistrationContext, reg *require.Registry) error {
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

func NewLoader(host *gojahttp.Host, opts ...Option) require.ModuleLoader {
	registrar := NewRegistrar(host, opts...)
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		if host != nil {
			if runtimeServices, ok := runtimebridge.Lookup(vm); ok && runtimeServices.Owner != nil {
				host.SetRuntime(runtimebridgeOwnerAdapter{owner: runtimeServices.Owner})
			}
		}
		registrar.loader(vm, moduleObj)
	}
}

type runtimebridgeOwnerAdapter struct {
	owner runtimebridge.RuntimeOwner
}

func (a runtimebridgeOwnerAdapter) Call(ctx context.Context, op string, fn runtimeowner.CallFunc) (any, error) {
	return a.owner.Call(ctx, op, func(ctx context.Context, vm *goja.Runtime) (any, error) {
		return fn(ctx, vm)
	})
}

func (a runtimebridgeOwnerAdapter) Post(ctx context.Context, op string, fn runtimeowner.PostFunc) error {
	return a.owner.Post(ctx, op, func(ctx context.Context, vm *goja.Runtime) {
		fn(ctx, vm)
	})
}

func (a runtimebridgeOwnerAdapter) WaitIdle(context.Context) error { return nil }
func (a runtimebridgeOwnerAdapter) Shutdown(context.Context) error { return nil }
func (a runtimebridgeOwnerAdapter) IsClosed() bool                 { return false }

func (r *Registrar) loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	_ = exports.Set("app", func() goja.Value { return r.appObject(vm) })
}

type spaOptions struct {
	Index           string
	ExcludePrefixes []string
}

type mountOptions struct {
	StripPrefix     bool
	ExcludePrefixes []string
}

func spaFromAssetsOptions(vm *goja.Runtime, value goja.Value) spaOptions {
	ret := spaOptions{Index: "index.html", ExcludePrefixes: []string{"/api"}}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ret
	}
	obj := value.ToObject(vm)
	if index := obj.Get("index"); index != nil && !goja.IsUndefined(index) && !goja.IsNull(index) {
		ret.Index = index.String()
	}
	if excludes := obj.Get("excludePrefixes"); excludes != nil && !goja.IsUndefined(excludes) && !goja.IsNull(excludes) {
		ret.ExcludePrefixes = nil
		if arr, ok := excludes.Export().([]any); ok {
			for _, exclude := range arr {
				if s, ok := exclude.(string); ok && s != "" {
					ret.ExcludePrefixes = append(ret.ExcludePrefixes, s)
				}
			}
		}
	}
	return ret
}

func mountOptionsFromValue(vm *goja.Runtime, value goja.Value) mountOptions {
	ret := mountOptions{}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ret
	}
	obj := value.ToObject(vm)
	if strip := obj.Get("stripPrefix"); strip != nil && !goja.IsUndefined(strip) && !goja.IsNull(strip) {
		ret.StripPrefix = strip.ToBoolean()
	}
	if excludes := obj.Get("excludePrefixes"); excludes != nil && !goja.IsUndefined(excludes) && !goja.IsNull(excludes) {
		ret.ExcludePrefixes = nil
		if arr, ok := excludes.Export().([]any); ok {
			for _, exclude := range arr {
				if s, ok := exclude.(string); ok && s != "" {
					ret.ExcludePrefixes = append(ret.ExcludePrefixes, s)
				}
			}
		}
	}
	return ret
}

func (r *Registrar) appObject(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	mount := func(prefix string, handlerValue goja.Value, options goja.Value) error {
		if prefix == "" {
			return fmt.Errorf("app.mount requires prefix")
		}
		if err := r.start(vm); err != nil {
			return err
		}
		handler, ok := gojahttp.HTTPHandlerFromValue(handlerValue)
		if !ok {
			return fmt.Errorf("app.mount(%q) requires a Go http.Handler-backed object", prefix)
		}
		opts := mountOptionsFromValue(vm, options)
		r.host.RegisterHandlerWithOptions(prefix, handler, gojahttp.MountOptions{StripPrefix: opts.StripPrefix, ExcludePrefixes: opts.ExcludePrefixes})
		return nil
	}
	_ = obj.Set("mount", mount)
	_ = obj.Set("mountHandler", mount)
	for _, method := range []string{"get", "post", "put", "patch", "delete", "all"} {
		method := method
		_ = obj.Set(method, func(pattern string, handler goja.Value) error {
			fn, ok := goja.AssertFunction(handler)
			if !ok {
				return fmt.Errorf("app.%s(%q) requires a function handler", method, pattern)
			}
			if err := r.start(vm); err != nil {
				return err
			}
			r.host.Register(strings.ToUpper(method), pattern, fn)
			return nil
		})
	}
	_ = obj.Set("static", func(prefix, dir string) error {
		if prefix == "" || dir == "" {
			return fmt.Errorf("app.static requires prefix and directory")
		}
		if err := r.start(vm); err != nil {
			return err
		}
		r.host.RegisterStatic(prefix, dir)
		return nil
	})
	_ = obj.Set("staticFromAssetsModule", func(prefix string, assetsModule goja.Value, root string) error {
		if prefix == "" || root == "" {
			return fmt.Errorf("app.staticFromAssetsModule requires prefix and root")
		}
		if err := r.start(vm); err != nil {
			return err
		}
		handler, err := fsmod.StaticHandlerFromAssetsModule(vm, assetsModule, root)
		if err != nil {
			return err
		}
		r.host.RegisterStaticHandler(prefix, handler)
		return nil
	})
	_ = obj.Set("spaFromAssetsModule", func(prefix string, assetsModule goja.Value, root string, options goja.Value) error {
		if prefix == "" || root == "" {
			return fmt.Errorf("app.spaFromAssetsModule requires prefix and root")
		}
		if err := r.start(vm); err != nil {
			return err
		}
		spaOptions := spaFromAssetsOptions(vm, options)
		handler, err := fsmod.SPAHandlerFromAssetsModule(vm, assetsModule, root, spaOptions.Index)
		if err != nil {
			return err
		}
		r.host.RegisterStaticHandlerWithOptions(prefix, handler, spaOptions.ExcludePrefixes)
		return nil
	})
	_ = obj.Set("listen", func() error { return r.start(vm) })
	return obj
}

func (r *Registrar) start(vm *goja.Runtime) error {
	if r.onUse == nil {
		return nil
	}
	return r.onUse(vm)
}
