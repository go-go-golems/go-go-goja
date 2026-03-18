package contract

import (
	"fmt"
	"strings"
)

type ManifestValidationOptions struct {
	NamespacePrefix string
	AllowModules    []string
}

// ValidateManifest enforces shared plugin manifest rules for SDK and host code.
func ValidateManifest(manifest *ModuleManifest, opts ManifestValidationOptions) error {
	if manifest == nil {
		return fmt.Errorf("plugin manifest is nil")
	}

	name := strings.TrimSpace(manifest.GetModuleName())
	if name == "" {
		return fmt.Errorf("plugin manifest module name is empty")
	}

	namespacePrefix := strings.TrimSpace(opts.NamespacePrefix)
	if namespacePrefix != "" && !strings.HasPrefix(name, namespacePrefix) {
		return fmt.Errorf("plugin module %q must use namespace %q", name, namespacePrefix)
	}

	if len(opts.AllowModules) > 0 {
		allowed := false
		for _, candidate := range opts.AllowModules {
			if strings.TrimSpace(candidate) == name {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("plugin module %q is not in the allowlist", name)
		}
	}

	exportNames := map[string]struct{}{}
	for _, exp := range manifest.GetExports() {
		exportName := strings.TrimSpace(exp.GetName())
		if exportName == "" {
			return fmt.Errorf("plugin module %q has an export with empty name", name)
		}
		if _, ok := exportNames[exportName]; ok {
			return fmt.Errorf("plugin module %q has duplicate export %q", name, exportName)
		}
		exportNames[exportName] = struct{}{}

		switch exp.GetKind() {
		case ExportKind_EXPORT_KIND_UNSPECIFIED:
			return fmt.Errorf("plugin module %q export %q has unspecified kind", name, exportName)
		case ExportKind_EXPORT_KIND_FUNCTION:
			if len(exp.GetMethodSpecs()) > 0 {
				return fmt.Errorf("function export %q in module %q must not define methods", exportName, name)
			}
		case ExportKind_EXPORT_KIND_OBJECT:
			if len(exp.GetMethodSpecs()) == 0 {
				return fmt.Errorf("object export %q in module %q must define methods", exportName, name)
			}
			methodNames := map[string]struct{}{}
			for _, method := range exp.GetMethodSpecs() {
				methodName := strings.TrimSpace(method.GetName())
				if methodName == "" {
					return fmt.Errorf("object export %q in module %q has an empty method name", exportName, name)
				}
				if _, ok := methodNames[methodName]; ok {
					return fmt.Errorf("object export %q in module %q has duplicate method %q", exportName, name, methodName)
				}
				methodNames[methodName] = struct{}{}
			}
		default:
			return fmt.Errorf("plugin module %q export %q has unsupported kind %q", name, exportName, exp.GetKind().String())
		}
	}

	return nil
}
