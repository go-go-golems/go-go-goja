package validate

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

// Bundle validates all modules in a descriptor bundle.
func Bundle(bundle *spec.Bundle) error {
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}

	for i, module := range bundle.Modules {
		if err := Module(module); err != nil {
			return fmt.Errorf("bundle.modules[%d]: %w", i, err)
		}
	}
	return nil
}

// Module validates a module descriptor.
func Module(module *spec.Module) error {
	if module == nil {
		return fmt.Errorf("module is nil")
	}
	moduleName := strings.TrimSpace(module.Name)
	if moduleName == "" {
		return fmt.Errorf("module name is empty")
	}

	functionNames := map[string]struct{}{}
	for i, fn := range module.Functions {
		fnName := strings.TrimSpace(fn.Name)
		if fnName == "" {
			return fmt.Errorf("module %q function[%d] name is empty", moduleName, i)
		}
		if _, ok := functionNames[fnName]; ok {
			return fmt.Errorf("module %q has duplicate function %q", moduleName, fnName)
		}
		functionNames[fnName] = struct{}{}

		for j, param := range fn.Params {
			paramName := strings.TrimSpace(param.Name)
			if paramName == "" {
				return fmt.Errorf("module %q function %q param[%d] name is empty", moduleName, fnName, j)
			}
			if err := typeRef(param.Type, fmt.Sprintf("module %q function %q param %q", moduleName, fnName, paramName)); err != nil {
				return err
			}
		}

		if err := typeRef(fn.Returns, fmt.Sprintf("module %q function %q return", moduleName, fnName)); err != nil {
			return err
		}
	}
	return nil
}

func typeRef(ref spec.TypeRef, path string) error {
	switch ref.Kind {
	case spec.TypeKindString,
		spec.TypeKindNumber,
		spec.TypeKindBoolean,
		spec.TypeKindAny,
		spec.TypeKindUnknown,
		spec.TypeKindVoid,
		spec.TypeKindNever:
		return nil

	case spec.TypeKindNamed:
		if strings.TrimSpace(ref.Name) == "" {
			return fmt.Errorf("%s named type is empty", path)
		}
		return nil

	case spec.TypeKindArray:
		if ref.Item == nil {
			return fmt.Errorf("%s array item is nil", path)
		}
		return typeRef(*ref.Item, path+"[]")

	case spec.TypeKindUnion:
		if len(ref.Union) == 0 {
			return fmt.Errorf("%s union has no members", path)
		}
		for i := range ref.Union {
			if err := typeRef(ref.Union[i], fmt.Sprintf("%s union[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil

	case spec.TypeKindObject:
		for i, field := range ref.Fields {
			fieldName := strings.TrimSpace(field.Name)
			if fieldName == "" {
				return fmt.Errorf("%s object field[%d] name is empty", path, i)
			}
			if err := typeRef(field.Type, fmt.Sprintf("%s object field %q", path, fieldName)); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("%s has unknown type kind %q", path, ref.Kind)
	}
}
