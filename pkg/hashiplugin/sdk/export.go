package sdk

import (
	"fmt"
	"slices"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

type ExportOption func(*exportConfig) error
type ObjectOption func(*objectConfig) error
type MethodOption func(*methodConfig) error

type exportConfig struct {
	doc string
}

type methodConfig struct {
	doc     string
	summary string
	tags    []string
}

type objectConfig struct {
	doc     string
	methods []*methodDefinition
}

type exportDefinition struct {
	name    string
	kind    contract.ExportKind
	doc     string
	handler Handler
	methods []*methodDefinition
}

type methodDefinition struct {
	name    string
	doc     string
	summary string
	tags    []string
	handler Handler
}

func ExportDoc(doc string) ExportOption {
	return func(cfg *exportConfig) error {
		cfg.doc = strings.TrimSpace(doc)
		return nil
	}
}

func ObjectDoc(doc string) ObjectOption {
	return func(cfg *objectConfig) error {
		cfg.doc = strings.TrimSpace(doc)
		return nil
	}
}

func MethodDoc(doc string) MethodOption {
	return func(cfg *methodConfig) error {
		cfg.doc = strings.TrimSpace(doc)
		return nil
	}
}

func MethodSummary(summary string) MethodOption {
	return func(cfg *methodConfig) error {
		cfg.summary = strings.TrimSpace(summary)
		return nil
	}
}

func MethodTags(tags ...string) MethodOption {
	return func(cfg *methodConfig) error {
		cfg.tags = append(cfg.tags, tags...)
		return nil
	}
}

func Function(name string, fn Handler, opts ...ExportOption) ModuleOption {
	return func(def *moduleDefinition) error {
		cfg := exportConfig{}
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("sdk function %q option %d is nil", name, i)
			}
			if err := opt(&cfg); err != nil {
				return err
			}
		}
		def.exports = append(def.exports, &exportDefinition{
			name:    strings.TrimSpace(name),
			kind:    contract.ExportKind_EXPORT_KIND_FUNCTION,
			doc:     cfg.doc,
			handler: fn,
		})
		return nil
	}
}

func Object(name string, opts ...ObjectOption) ModuleOption {
	return func(def *moduleDefinition) error {
		cfg := objectConfig{}
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("sdk object %q option %d is nil", name, i)
			}
			if err := opt(&cfg); err != nil {
				return err
			}
		}
		def.exports = append(def.exports, &exportDefinition{
			name:    strings.TrimSpace(name),
			kind:    contract.ExportKind_EXPORT_KIND_OBJECT,
			doc:     cfg.doc,
			methods: cfg.methods,
		})
		return nil
	}
}

func Method(name string, fn Handler, opts ...MethodOption) ObjectOption {
	return func(cfg *objectConfig) error {
		methodCfg := methodConfig{}
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("sdk method %q option %d is nil", name, i)
			}
			if err := opt(&methodCfg); err != nil {
				return err
			}
		}
		method := &methodDefinition{
			name:    strings.TrimSpace(name),
			doc:     methodCfg.doc,
			summary: methodCfg.summary,
			tags:    normalizeStrings(methodCfg.tags),
			handler: fn,
		}
		if len(cfg.methods) > 0 {
			names := make([]string, 0, len(cfg.methods))
			for _, existing := range cfg.methods {
				names = append(names, existing.name)
			}
			if slices.Contains(names, method.name) {
				return fmt.Errorf("sdk object method %q is duplicated", method.name)
			}
		}
		cfg.methods = append(cfg.methods, method)
		return nil
	}
}
