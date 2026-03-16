package jsverbs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/engine"
)

func (r *Registry) invoke(ctx context.Context, verb *VerbSpec, parsedValues *values.Values) (interface{}, error) {
	factory, err := engine.NewBuilder().
		WithRequireOptions(require.WithLoader(r.sourceLoader)).
		WithModules(engine.DefaultRegistryModules()).
		Build()
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

	args, err := buildArguments(parsedValues, verb, r.RootDir)
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

func buildArguments(parsedValues *values.Values, verb *VerbSpec, rootDir string) ([]interface{}, error) {
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

	args := make([]interface{}, 0, len(verb.Params))
	for _, param := range verb.Params {
		field := verb.Field(param.Name)
		if field == nil {
			field = inferFieldFromParam(param)
		}
		if field != nil {
			switch bind := strings.TrimSpace(field.Bind); bind {
			case "all":
				args = append(args, allValues)
				continue
			case "context":
				args = append(args, map[string]interface{}{
					"verb":       verb.FullPath(),
					"function":   verb.FunctionName,
					"module":     verb.File.ModulePath,
					"sourceFile": verb.File.AbsPath,
					"rootDir":    rootDir,
					"values":     allValues,
					"sections":   sectionValues,
				})
				continue
			case "":
			default:
				slug := cleanCommandWord(bind)
				args = append(args, cloneMap(sectionValues[slug]))
				continue
			}
		}
		if param.Kind != ParameterIdentifier && param.Kind != ParameterUnknown {
			return nil, fmt.Errorf("%s parameter %q requires a bind", verb.SourceRef(), param.Name)
		}
		sectionSlug := schema.DefaultSlug
		if field != nil && field.Section != "" {
			sectionSlug = field.Section
		}
		value := sectionValues[sectionSlug][param.Name]
		if param.Rest {
			rv := reflect.ValueOf(value)
			if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
				for i := 0; i < rv.Len(); i++ {
					args = append(args, rv.Index(i).Interface())
				}
				continue
			}
		}
		args = append(args, value)
	}
	return args, nil
}

func (r *Registry) sourceLoader(path string) ([]byte, error) {
	absPath, err := resolveModulePath(r.RootDir, path)
	if err != nil {
		return nil, require.ModuleFileDoesNotExistError
	}
	src, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, require.ModuleFileDoesNotExistError
		}
		return nil, err
	}
	return []byte(r.injectOverlay(path, absPath, string(src))), nil
}

func (r *Registry) injectOverlay(moduleKey, absPath, source string) string {
	file := r.filesByAbs[absPath]
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

func resolveModulePath(rootDir, modulePath string) (string, error) {
	modulePath = filepath.FromSlash(strings.TrimSpace(modulePath))
	if modulePath == "" {
		return "", fmt.Errorf("module path is empty")
	}
	candidates := []string{}
	if filepath.IsAbs(modulePath) {
		candidates = append(candidates, modulePath)
	} else {
		candidates = append(candidates, filepath.Join(rootDir, modulePath))
	}

	expanded := []string{}
	for _, candidate := range candidates {
		expanded = append(expanded, candidate)
		expanded = append(expanded, candidate+".js", candidate+".cjs")
		expanded = append(expanded, filepath.Join(candidate, "index.js"))
		expanded = append(expanded, filepath.Join(candidate, "index.cjs"))
	}
	for _, candidate := range expanded {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("module %s not found", modulePath)
}

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
