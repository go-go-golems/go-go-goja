package host

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

// ValidateManifest enforces the v1 plugin manifest rules.
func ValidateManifest(cfg Config, manifest *contract.ModuleManifest) error {
	cfg = cfg.withDefaults()
	if manifest == nil {
		return fmt.Errorf("plugin manifest is nil")
	}

	name := strings.TrimSpace(manifest.GetModuleName())
	if name == "" {
		return fmt.Errorf("plugin manifest module name is empty")
	}
	if !strings.HasPrefix(name, cfg.Namespace) {
		return fmt.Errorf("plugin module %q must use namespace %q", name, cfg.Namespace)
	}
	if len(cfg.AllowModules) > 0 {
		allowed := false
		for _, candidate := range cfg.AllowModules {
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
		case contract.ExportKind_EXPORT_KIND_UNSPECIFIED:
			return fmt.Errorf("plugin module %q export %q has unspecified kind", name, exportName)
		case contract.ExportKind_EXPORT_KIND_FUNCTION:
			if len(exp.GetMethods()) > 0 {
				return fmt.Errorf("function export %q in module %q must not define methods", exportName, name)
			}
		case contract.ExportKind_EXPORT_KIND_OBJECT:
			if len(exp.GetMethods()) == 0 {
				return fmt.Errorf("object export %q in module %q must define methods", exportName, name)
			}
			methodNames := map[string]struct{}{}
			for _, method := range exp.GetMethods() {
				method = strings.TrimSpace(method)
				if method == "" {
					return fmt.Errorf("object export %q in module %q has an empty method name", exportName, name)
				}
				if _, ok := methodNames[method]; ok {
					return fmt.Errorf("object export %q in module %q has duplicate method %q", exportName, name, method)
				}
				methodNames[method] = struct{}{}
			}
		default:
			return fmt.Errorf("plugin module %q export %q has unsupported kind %q", name, exportName, exp.GetKind().String())
		}
	}

	return nil
}
