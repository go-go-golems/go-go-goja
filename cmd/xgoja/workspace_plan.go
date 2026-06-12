package main

import (
	"path"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
)

func goModulePlanForBuildSpec(buildSpec *buildspec.BuildSpec) (*workspace.Plan, error) {
	if buildSpec == nil {
		return nil, nil
	}
	requirements := []workspace.Requirement{}
	if (buildSpec.Target.Kind == "adapter" || buildSpec.Target.Kind == "cobra") && strings.TrimSpace(buildSpec.Target.Import) != "" {
		requirements = append(requirements, workspace.Requirement{
			ModulePath: providerModulePathForWorkspace(buildSpec.Target.Import),
			Version:    buildSpec.Target.Version,
			RequiredBy: []string{"target:" + buildSpec.Target.Kind},
		})
	}
	for _, pkg := range buildSpec.Packages {
		requirements = append(requirements, workspace.Requirement{
			ModulePath:      providerModulePathForWorkspace(pkg.Import),
			Version:         pkg.Version,
			ExplicitReplace: pkg.Replace,
			RequiredBy:      []string{"package:" + pkg.ID},
		})
	}
	for _, imp := range buildSpec.Go.Imports {
		modulePath := strings.TrimSpace(imp.Module)
		if modulePath == "" {
			modulePath = providerModulePathForWorkspace(imp.Import)
		}
		requirements = append(requirements, workspace.Requirement{
			ModulePath: modulePath,
			Version:    imp.Version,
			RequiredBy: []string{"go.import:" + imp.Import},
		})
	}
	return workspace.Resolve(requirements, workspace.Options{
		Spec:     workspace.Spec{Mode: workspace.ModeAuto},
		StartDir: buildSpec.BaseDir,
	})
}

func providerModulePathForWorkspace(importPath string) string {
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
