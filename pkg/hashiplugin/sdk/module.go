package sdk

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"google.golang.org/protobuf/proto"
)

const DefaultNamespace = "plugin:"

type Handler func(context.Context, *Call) (any, error)

type ModuleOption func(*moduleDefinition) error

type moduleDefinition struct {
	name         string
	version      string
	doc          string
	capabilities []string
	exports      []*exportDefinition
}

type Module struct {
	manifest *contract.ModuleManifest
	dispatch map[dispatchKey]Handler
}

var _ contract.JSModule = (*Module)(nil)

func NewModule(name string, opts ...ModuleOption) (*Module, error) {
	def := &moduleDefinition{
		name: strings.TrimSpace(name),
	}
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("sdk module %q option %d is nil", def.name, i)
		}
		if err := opt(def); err != nil {
			return nil, err
		}
	}
	if err := validateModuleDefinition(def); err != nil {
		return nil, err
	}

	manifest := buildManifest(def)
	dispatch, err := buildDispatchTable(def)
	if err != nil {
		return nil, err
	}

	return &Module{
		manifest: manifest,
		dispatch: dispatch,
	}, nil
}

func MustModule(name string, opts ...ModuleOption) *Module {
	mod, err := NewModule(name, opts...)
	if err != nil {
		panic(err)
	}
	return mod
}

func Version(v string) ModuleOption {
	return func(def *moduleDefinition) error {
		def.version = strings.TrimSpace(v)
		return nil
	}
}

func Doc(doc string) ModuleOption {
	return func(def *moduleDefinition) error {
		def.doc = strings.TrimSpace(doc)
		return nil
	}
}

func Capabilities(values ...string) ModuleOption {
	return func(def *moduleDefinition) error {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if slices.Contains(def.capabilities, value) {
				continue
			}
			def.capabilities = append(def.capabilities, value)
		}
		return nil
	}
}

func (m *Module) Manifest(context.Context) (*contract.ModuleManifest, error) {
	if m == nil || m.manifest == nil {
		return nil, fmt.Errorf("sdk module manifest is nil")
	}
	return proto.Clone(m.manifest).(*contract.ModuleManifest), nil
}

func validateModuleDefinition(def *moduleDefinition) error {
	if def == nil {
		return fmt.Errorf("sdk module definition is nil")
	}
	for _, exp := range def.exports {
		if exp == nil {
			return fmt.Errorf("sdk module %q contains a nil export", def.name)
		}
		exp.name = strings.TrimSpace(exp.name)

		switch exp.kind {
		case contract.ExportKind_EXPORT_KIND_UNSPECIFIED:
			// Shared manifest validation below owns kind-shape validation.
		case contract.ExportKind_EXPORT_KIND_FUNCTION:
			if exp.handler == nil {
				return fmt.Errorf("sdk function export %q in module %q has nil handler", exp.name, def.name)
			}
			if len(exp.methods) > 0 {
				return fmt.Errorf("sdk function export %q in module %q must not define methods", exp.name, def.name)
			}
		case contract.ExportKind_EXPORT_KIND_OBJECT:
			for _, method := range exp.methods {
				if method == nil {
					return fmt.Errorf("sdk object export %q in module %q contains a nil method", exp.name, def.name)
				}
				method.name = strings.TrimSpace(method.name)
				if method.handler == nil {
					return fmt.Errorf("sdk method %q in export %q module %q has nil handler", method.name, exp.name, def.name)
				}
			}
		default:
			return fmt.Errorf("sdk export %q in module %q has unsupported kind %q", exp.name, def.name, exp.kind.String())
		}
	}

	def.capabilities = normalizeStrings(def.capabilities)
	if err := contract.ValidateManifest(buildManifest(def), contract.ManifestValidationOptions{
		NamespacePrefix: DefaultNamespace,
	}); err != nil {
		return fmt.Errorf("sdk module definition invalid: %w", err)
	}
	return nil
}

func buildManifest(def *moduleDefinition) *contract.ModuleManifest {
	manifest := &contract.ModuleManifest{
		ModuleName:   def.name,
		Version:      def.version,
		Exports:      make([]*contract.ExportSpec, 0, len(def.exports)),
		Capabilities: append([]string(nil), def.capabilities...),
		Doc:          def.doc,
	}
	for _, exp := range def.exports {
		spec := &contract.ExportSpec{
			Name: exp.name,
			Kind: exp.kind,
			Doc:  exp.doc,
		}
		if exp.kind == contract.ExportKind_EXPORT_KIND_OBJECT {
			methods := make([]*contract.MethodSpec, 0, len(exp.methods))
			for _, method := range exp.methods {
				methods = append(methods, &contract.MethodSpec{
					Name:    method.name,
					Summary: method.summary,
					Doc:     method.doc,
					Tags:    append([]string(nil), method.tags...),
				})
			}
			spec.MethodSpecs = methods
		}
		manifest.Exports = append(manifest.Exports, spec)
	}
	return manifest
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}
