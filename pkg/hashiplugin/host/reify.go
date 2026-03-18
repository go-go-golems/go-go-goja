package host

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"google.golang.org/protobuf/types/known/structpb"
)

// RegisterModule reifies a loaded plugin into a native CommonJS module.
func RegisterModule(reg *require.Registry, loaded *LoadedModule) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	if loaded == nil || loaded.Manifest == nil {
		return fmt.Errorf("loaded plugin module is nil")
	}

	requireName := loaded.RequireName()
	if requireName == "" {
		return fmt.Errorf("loaded plugin module has empty require name")
	}

	reg.RegisterNativeModule(requireName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		for _, exp := range loaded.Manifest.GetExports() {
			exp := exp
			switch exp.GetKind() {
			case contract.ExportKind_EXPORT_KIND_UNSPECIFIED:
				continue
			case contract.ExportKind_EXPORT_KIND_FUNCTION:
				modules.SetExport(exports, requireName, exp.GetName(), func(call goja.FunctionCall) goja.Value {
					return invokeExport(vm, loaded, exp.GetName(), "", call)
				})
			case contract.ExportKind_EXPORT_KIND_OBJECT:
				obj := vm.NewObject()
				for _, method := range exp.GetMethodSpecs() {
					methodName := method.GetName()
					modules.SetExport(obj, requireName, methodName, func(call goja.FunctionCall) goja.Value {
						return invokeExport(vm, loaded, exp.GetName(), methodName, call)
					})
				}
				modules.SetExport(exports, requireName, exp.GetName(), obj)
			}
		}
	})

	return nil
}

func invokeExport(vm *goja.Runtime, loaded *LoadedModule, exportName, methodName string, call goja.FunctionCall) goja.Value {
	args, err := exportArgs(call.Arguments)
	if err != nil {
		panic(vm.NewGoError(err))
	}

	resp, err := loaded.Invoke(context.Background(), &contract.InvokeRequest{
		ExportName: exportName,
		MethodName: methodName,
		Args:       args,
	})
	if err != nil {
		panic(vm.NewGoError(err))
	}

	if resp == nil || resp.Result == nil {
		return vm.ToValue(nil)
	}
	return vm.ToValue(resp.Result.AsInterface())
}

func exportArgs(args []goja.Value) ([]*structpb.Value, error) {
	out := make([]*structpb.Value, 0, len(args))
	for i, arg := range args {
		value, err := structpb.NewValue(arg.Export())
		if err != nil {
			return nil, fmt.Errorf("convert argument %d: %w", i, err)
		}
		out = append(out, value)
	}
	return out, nil
}
