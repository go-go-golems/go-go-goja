package generate

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

const xgojaRuntimeModule = "github.com/go-go-golems/go-go-goja"

func RenderGoMod(spec *buildspec.Spec, opts Options) string {
	defaults := defaultOptions()
	if opts.XGojaModuleVersion == "" {
		opts.XGojaModuleVersion = defaults.XGojaModuleVersion
	}
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n", spec.Go.Module)
	fmt.Fprintf(&b, "go %s\n\n", spec.Go.Version)

	requires := map[string]string{xgojaRuntimeModule: opts.XGojaModuleVersion}
	if (spec.Target.Kind == "adapter" || spec.Target.Kind == "cobra") && strings.TrimSpace(spec.Target.Version) != "" {
		requires[providerModulePath(spec.Target.Import)] = spec.Target.Version
	}
	for _, pkg := range spec.Packages {
		version := strings.TrimSpace(pkg.Version)
		if version == "" {
			continue
		}
		requires[providerModulePath(pkg.Import)] = version
	}
	keys := make([]string, 0, len(requires))
	for k := range requires {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteString("require (\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "\t%s %s\n", k, requires[k])
	}
	b.WriteString(")\n")

	replaces := map[string]string{}
	if strings.TrimSpace(opts.XGojaReplace) != "" {
		replaces[xgojaRuntimeModule] = opts.XGojaReplace
	}
	for _, pkg := range spec.Packages {
		if strings.TrimSpace(pkg.Replace) != "" {
			replaces[providerModulePath(pkg.Import)] = pkg.Replace
		}
	}
	if len(replaces) > 0 {
		keys = keys[:0]
		for k := range replaces {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "replace %s => %s\n", k, filepathSlash(replaces[k]))
		}
	}
	return b.String()
}

func providerModulePath(importPath string) string {
	importPath = strings.Trim(path.Clean(strings.TrimSpace(importPath)), "/")
	if importPath == "." {
		return ""
	}
	for _, marker := range []string{"/pkg/", "/cmd/", "/internal/"} {
		if idx := strings.Index(importPath, marker); idx >= 0 {
			return importPath[:idx]
		}
	}
	if strings.HasSuffix(importPath, "/xgoja") {
		return path.Dir(importPath)
	}
	return importPath
}

func filepathSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
