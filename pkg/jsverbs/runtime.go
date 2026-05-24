package jsverbs

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/engine"
)

func (r *Registry) invoke(ctx context.Context, verb *VerbSpec, parsedValues *values.Values) (interface{}, error) {
	builder := engine.NewBuilder().
		WithRequireOptions(require.WithLoader(r.sourceLoader))
	if r.ModuleMiddleware != nil {
		builder = builder.UseModuleMiddleware(r.ModuleMiddleware)
	}
	factory, err := builder.Build()
	if err != nil {
		return nil, err
	}

	runtime, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = runtime.Close(context.Background())
	}()

	return r.InvokeInRuntime(ctx, runtime, verb, parsedValues)
}

// RequireLoader exposes the scanned-source overlay loader so host applications
// can compose their own engine runtime while still using jsverbs module capture.
func (r *Registry) RequireLoader() func(modulePath string) ([]byte, error) {
	return r.sourceLoader
}

// InvokeInRuntime invokes a verb inside an already-live caller-owned runtime.
// Unlike the default Commands()/invoke() path, it does not create or close the runtime.
func (r *Registry) InvokeInRuntime(ctx context.Context, runtime *engine.Runtime, verb *VerbSpec, parsedValues *values.Values) (interface{}, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	if runtime == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	if verb == nil {
		return nil, fmt.Errorf("verb is nil")
	}
	plan, err := buildVerbBindingPlan(r, verb)
	if err != nil {
		return nil, err
	}
	args, err := buildArguments(parsedValues, plan, r.RootDir)
	if err != nil {
		return nil, err
	}

	ret, err := runtime.Owner.Call(ctx, "jsverbs.invoke", func(_ context.Context, vm *goja.Runtime) (interface{}, error) {
		if _, err := runtime.Require.Require(verb.File.ModulePath); err != nil {
			return nil, err
		}
		registryValue := vm.Get("__glazedVerbRegistry")
		if registryValue == nil || goja.IsUndefined(registryValue) || goja.IsNull(registryValue) {
			return nil, fmt.Errorf("js verb registry not initialized")
		}
		registryObject := registryValue.ToObject(vm)
		entryValue := registryObject.Get(verb.File.ModulePath)
		if entryValue == nil || goja.IsUndefined(entryValue) || goja.IsNull(entryValue) {
			return nil, fmt.Errorf("js verb module entry missing for %s", verb.File.ModulePath)
		}
		entryObject := entryValue.ToObject(vm)
		fnValue := entryObject.Get(verb.FunctionName)
		fn, ok := goja.AssertFunction(fnValue)
		if !ok {
			return nil, fmt.Errorf("js function %s not captured for %s", verb.FunctionName, verb.File.RelPath)
		}
		jsArgs := make([]goja.Value, 0, len(args))
		for _, arg := range args {
			jsArgs = append(jsArgs, vm.ToValue(arg))
		}
		result, err := fn(goja.Undefined(), jsArgs...)
		if err != nil {
			return nil, err
		}
		if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
			return nil, nil
		}
		if promise, ok := result.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return result.Export(), nil
	})
	if err != nil {
		return nil, err
	}

	if promise, ok := ret.(*goja.Promise); ok {
		return waitForPromise(ctx, runtime, promise)
	}
	return ret, nil
}

func buildArguments(parsedValues *values.Values, plan *VerbBindingPlan, rootDir string) ([]interface{}, error) {
	sectionValues := map[string]map[string]interface{}{}
	if parsedValues == nil {
		parsedValues = values.New()
	}
	allValues := parsedValues.GetDataMap()

	parsedValues.ForEach(func(slug string, value *values.SectionValues) {
		sectionMap := map[string]interface{}{}
		value.Fields.ForEach(func(_ string, fieldValue *fields.FieldValue) {
			if fieldValue == nil || fieldValue.Definition == nil {
				return
			}
			sectionMap[fieldValue.Definition.Name] = fieldValue.Value
		})
		sectionValues[slug] = sectionMap
	})

	args := make([]interface{}, 0, len(plan.Parameters))
	for _, binding := range plan.Parameters {
		switch binding.Mode {
		case BindingModeAll:
			args = append(args, allValues)
			continue
		case BindingModeContext:
			args = append(args, map[string]interface{}{
				"verb":       plan.Verb.FullPath(),
				"function":   plan.Verb.FunctionName,
				"module":     plan.Verb.File.ModulePath,
				"sourceFile": fileSourcePath(plan.Verb.File),
				"rootDir":    rootDir,
				"values":     allValues,
				"sections":   sectionValues,
			})
			continue
		case BindingModeSection:
			args = append(args, cloneMap(sectionValues[binding.SectionSlug]))
			continue
		case BindingModePositional:
			value := sectionValues[binding.SectionSlug][binding.Field.Name]
			if binding.Param.Rest {
				rv := reflect.ValueOf(value)
				if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
					for i := 0; i < rv.Len(); i++ {
						args = append(args, rv.Index(i).Interface())
					}
					continue
				}
			}
			args = append(args, value)
		default:
			return nil, fmt.Errorf("%s parameter %q has unsupported binding mode %q", plan.Verb.SourceRef(), binding.Param.Name, binding.Mode)
		}
	}
	return args, nil
}

func (r *Registry) sourceLoader(modulePath string) ([]byte, error) {
	file := r.filesByModule[modulePath]
	if file == nil {
		return nil, require.ModuleFileDoesNotExistError
	}
	return []byte(r.injectOverlay(modulePath, file, string(file.Source))), nil
}

func (r *Registry) injectOverlay(moduleKey string, file *FileSpec, source string) string {
	functionNames := []string{}
	if file != nil {
		for _, fn := range file.Functions {
			functionNames = append(functionNames, fn.Name)
		}
		sort.Strings(functionNames)
	}

	var suffix strings.Builder
	suffix.WriteString("\n")
	suffix.WriteString(`globalThis.__glazedVerbRegistry = globalThis.__glazedVerbRegistry || {};` + "\n")
	suffix.WriteString(`globalThis.__glazedVerbRegistry["`)
	suffix.WriteString(moduleKey)
	suffix.WriteString(`"] = {`)
	for i, name := range functionNames {
		if i > 0 {
			suffix.WriteString(",")
		}
		suffix.WriteString(name)
		suffix.WriteString(`: typeof `)
		suffix.WriteString(name)
		suffix.WriteString(` === "function" ? `)
		suffix.WriteString(name)
		suffix.WriteString(` : undefined`)
	}
	suffix.WriteString("};\n")

	prelude := strings.Join([]string{
		`globalThis.__glazedVerbRegistry = globalThis.__glazedVerbRegistry || {};`,
		`globalThis.__package__ = globalThis.__package__ || function() {};`,
		`globalThis.__section__ = globalThis.__section__ || function() {};`,
		`globalThis.__verb__ = globalThis.__verb__ || function() {};`,
		`globalThis.doc = globalThis.doc || function() { return ""; };`,
		"",
	}, "\n")

	return injectPrelude(source, prelude) + suffix.String()
}

func injectPrelude(source, prelude string) string {
	trimmed := strings.TrimLeft(source, "\ufeff \t\r\n")
	if strings.HasPrefix(trimmed, `"use strict";`) || strings.HasPrefix(trimmed, `'use strict';`) {
		if idx := strings.Index(source, "\n"); idx >= 0 {
			return source[:idx+1] + prelude + source[idx+1:]
		}
	}
	return prelude + source
}

// waitForPromise intentionally uses polling for v1. It is simple, explicit, and
// good enough for the current prototype, but should not be mistaken for the final
// async bridge design.
func waitForPromise(ctx context.Context, runtime *engine.Runtime, promise *goja.Promise) (interface{}, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, err := runtime.Owner.Call(ctx, "jsverbs.promise-state", func(_ context.Context, vm *goja.Runtime) (interface{}, error) {
			return promiseSnapshot{
				State:  promise.State(),
				Result: promise.Result(),
			}, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %s", valueString(snapshot.Result))
		case goja.PromiseStateFulfilled:
			if snapshot.Result == nil || goja.IsUndefined(snapshot.Result) || goja.IsNull(snapshot.Result) {
				return nil, nil
			}
			return snapshot.Result.Export(), nil
		}
	}
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

func valueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	return value.String()
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func fileSourcePath(file *FileSpec) string {
	if file == nil {
		return ""
	}
	if file.AbsPath != "" {
		return file.AbsPath
	}
	return file.RelPath
}
