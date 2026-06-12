package generate

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
)

const xgojaRuntimeModule = "github.com/go-go-golems/go-go-goja"

func RenderGoMod(buildSpec *buildspec.BuildSpec, opts Options) string {
	defaults := defaultOptions()
	if opts.XGojaModuleVersion == "" {
		opts.XGojaModuleVersion = defaults.XGojaModuleVersion
	}
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n", buildSpec.Go.Module)
	fmt.Fprintf(&b, "go %s\n\n", buildSpec.Go.Version)

	requires := map[string]string{xgojaRuntimeModule: opts.XGojaModuleVersion}
	if (buildSpec.Target.Kind == "adapter" || buildSpec.Target.Kind == "cobra") && strings.TrimSpace(buildSpec.Target.Version) != "" {
		requires[providerModulePath(buildSpec.Target.Import)] = buildSpec.Target.Version
	}
	for _, pkg := range buildSpec.Packages {
		version := strings.TrimSpace(pkg.Version)
		if version == "" {
			continue
		}
		requires[providerModulePath(pkg.Import)] = version
	}
	for _, imp := range buildSpec.Go.Imports {
		version := strings.TrimSpace(imp.Version)
		if version == "" {
			continue
		}
		modulePath := strings.TrimSpace(imp.Module)
		if modulePath == "" {
			modulePath = providerModulePath(imp.Import)
		}
		if modulePath != "" {
			requires[modulePath] = version
		}
	}
	for _, module := range plannedGoModules(opts.GoModules) {
		version := strings.TrimSpace(module.Version)
		if version == "" && strings.TrimSpace(module.LocalDir) != "" {
			version = "v0.0.0"
		}
		if version != "" {
			requires[module.ModulePath] = version
		}
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
	for _, pkg := range buildSpec.Packages {
		if strings.TrimSpace(pkg.Replace) != "" {
			replaces[providerModulePath(pkg.Import)] = resolveReplacePath(buildSpec.BaseDir, pkg.Replace)
		}
	}
	for _, module := range plannedGoModules(opts.GoModules) {
		if strings.TrimSpace(module.LocalDir) != "" {
			replaces[module.ModulePath] = module.LocalDir
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

func plannedGoModules(plan *workspace.Plan) []workspace.GoModulePlan {
	if plan == nil {
		return nil
	}
	return append([]workspace.GoModulePlan(nil), plan.Modules...)
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

func resolveReplacePath(baseDir, replace string) string {
	replace = strings.TrimSpace(replace)
	if replace == "" || filepath.IsAbs(replace) || strings.TrimSpace(baseDir) == "" {
		return replace
	}
	return filepath.Join(baseDir, replace)
}

func filepathSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
