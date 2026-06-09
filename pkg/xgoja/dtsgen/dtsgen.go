package dtsgen

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/validate"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type Options struct {
	Strict bool
	Header string
}

type MissingDescriptor struct {
	Package string
	Name    string
	Alias   string
}

type Result struct {
	DTS     string
	Missing []MissingDescriptor
}

func RenderRuntimeSpec(registry *providerapi.ProviderRegistry, runtimeSpec *app.RuntimeSpec, opts Options) (*Result, error) {
	bundle, missing, err := BundleRuntimeSpec(registry, runtimeSpec, opts)
	if err != nil {
		return nil, err
	}
	out, err := render.Bundle(bundle)
	if err != nil {
		return nil, err
	}
	return &Result{DTS: out, Missing: missing}, nil
}

func BundleRuntimeSpec(registry *providerapi.ProviderRegistry, runtimeSpec *app.RuntimeSpec, opts Options) (*spec.Bundle, []MissingDescriptor, error) {
	if registry == nil {
		return nil, nil, fmt.Errorf("provider registry is nil")
	}
	if runtimeSpec == nil {
		return nil, nil, fmt.Errorf("runtime spec is nil")
	}

	seen := map[string]struct{}{}
	modules := make([]*spec.Module, 0, len(runtimeSpec.Modules))
	missing := make([]MissingDescriptor, 0)

	for i, instance := range runtimeSpec.Modules {
		packageID := strings.TrimSpace(instance.Package)
		moduleName := strings.TrimSpace(instance.Name)
		if packageID == "" {
			return nil, nil, fmt.Errorf("runtime modules[%d] package is empty", i)
		}
		if moduleName == "" {
			return nil, nil, fmt.Errorf("runtime modules[%d] name is empty", i)
		}

		providerModule, ok := registry.ResolveModule(packageID, moduleName)
		if !ok {
			return nil, nil, fmt.Errorf("runtime module %s.%s is not registered", packageID, moduleName)
		}
		alias := requireName(instance, providerModule)
		if _, ok := seen[alias]; ok {
			return nil, nil, fmt.Errorf("duplicate require module alias %q", alias)
		}
		seen[alias] = struct{}{}

		if providerModule.TypeScript == nil {
			entry := MissingDescriptor{Package: packageID, Name: moduleName, Alias: alias}
			if opts.Strict {
				return nil, nil, fmt.Errorf("runtime module %s.%s as %q has no TypeScript descriptor", packageID, moduleName, alias)
			}
			missing = append(missing, entry)
			continue
		}

		descriptor := cloneModule(providerModule.TypeScript)
		descriptor.Name = alias
		if err := validate.Module(descriptor); err != nil {
			return nil, nil, fmt.Errorf("runtime module %s.%s as %q TypeScript descriptor: %w", packageID, moduleName, alias, err)
		}
		modules = append(modules, descriptor)
	}

	bundle := &spec.Bundle{HeaderComment: opts.Header, Modules: modules}
	if err := validate.Bundle(bundle); err != nil {
		return nil, nil, err
	}
	return bundle, missing, nil
}

func requireName(instance app.ModuleInstanceSpec, providerModule providerapi.Module) string {
	if alias := strings.TrimSpace(instance.As); alias != "" {
		return alias
	}
	if alias := strings.TrimSpace(providerModule.DefaultAs); alias != "" {
		return alias
	}
	return strings.TrimSpace(instance.Name)
}

func cloneModule(module *spec.Module) *spec.Module {
	if module == nil {
		return nil
	}
	return &spec.Module{
		Name:        module.Name,
		Description: module.Description,
		Functions:   cloneFunctions(module.Functions),
		RawDTS:      append([]string(nil), module.RawDTS...),
	}
}

func cloneFunctions(functions []spec.Function) []spec.Function {
	out := make([]spec.Function, len(functions))
	for i, fn := range functions {
		out[i] = spec.Function{
			Name:        fn.Name,
			Description: fn.Description,
			Params:      cloneParams(fn.Params),
			Returns:     cloneTypeRef(fn.Returns),
		}
	}
	return out
}

func cloneParams(params []spec.Param) []spec.Param {
	out := make([]spec.Param, len(params))
	for i, param := range params {
		out[i] = spec.Param{
			Name:        param.Name,
			Type:        cloneTypeRef(param.Type),
			Optional:    param.Optional,
			Variadic:    param.Variadic,
			Description: param.Description,
		}
	}
	return out
}

func cloneTypeRef(ref spec.TypeRef) spec.TypeRef {
	out := spec.TypeRef{
		Kind:   ref.Kind,
		Name:   ref.Name,
		Union:  cloneTypeRefs(ref.Union),
		Fields: cloneFields(ref.Fields),
	}
	if ref.Item != nil {
		item := cloneTypeRef(*ref.Item)
		out.Item = &item
	}
	return out
}

func cloneTypeRefs(refs []spec.TypeRef) []spec.TypeRef {
	out := make([]spec.TypeRef, len(refs))
	for i := range refs {
		out[i] = cloneTypeRef(refs[i])
	}
	return out
}

func cloneFields(fields []spec.Field) []spec.Field {
	out := make([]spec.Field, len(fields))
	for i, field := range fields {
		out[i] = spec.Field{Name: field.Name, Type: cloneTypeRef(field.Type), Optional: field.Optional}
	}
	return out
}
