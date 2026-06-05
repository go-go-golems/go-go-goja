package buildspec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Validate(buildSpec *BuildSpec) *Report {
	report := &Report{}
	if buildSpec == nil {
		report.AddError("buildSpec", "", "BuildSpec is nil")
		return report
	}

	validateName(report, buildSpec)
	validateAppSettings(report, buildSpec)
	validateConfig(report, buildSpec)
	validateTarget(report, buildSpec)
	packageIDs := validatePackages(report, buildSpec)
	validateModules(report, buildSpec.Modules, packageIDs)
	validateCommands(report, buildSpec)
	validateCommandProviders(report, buildSpec, packageIDs)
	validateJSVerbs(report, buildSpec, packageIDs)
	validateHelp(report, buildSpec, packageIDs)
	validateAssets(report, buildSpec)

	return report
}

func validateName(report *Report, buildSpec *BuildSpec) {
	if strings.TrimSpace(buildSpec.Name) == "" {
		report.AddError("name", "name", "name is required")
		return
	}
	report.AddOK("name", "name", fmt.Sprintf("BuildSpec name is %q", buildSpec.Name))
}

func validateAppSettings(report *Report, buildSpec *BuildSpec) {
	appName := strings.TrimSpace(buildSpec.AppName)
	if appName != "" {
		report.AddOK("app-name", "appName", appName)
	}
	envPrefix := strings.TrimSpace(buildSpec.EnvPrefix)
	if envPrefix == "" {
		return
	}
	if !isShellSafeEnvPrefix(envPrefix) {
		report.AddError("env-prefix", "envPrefix", "envPrefix must match [A-Z][A-Z0-9_]*")
		return
	}
	report.AddOK("env-prefix", "envPrefix", envPrefix)
}

func isShellSafeEnvPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	for i, r := range prefix {
		switch {
		case r >= 'A' && r <= 'Z':
			continue
		case r >= '0' && r <= '9' && i > 0:
			continue
		case r == '_' && i > 0:
			continue
		default:
			return false
		}
	}
	return true
}

func validateConfig(report *Report, buildSpec *BuildSpec) {
	if buildSpec.ConfigFile == nil || !buildSpec.ConfigFile.Enabled {
		report.AddOK("config", "config", "config not enabled")
		return
	}
	if usesAppScopedConfigLayer(buildSpec.ConfigFile.Layers) {
		if strings.TrimSpace(buildSpec.AppName) == "" {
			report.AddError("config-app-name", "config", "config layers system, xdg, and home require appName to be set")
		} else {
			report.AddOK("config-app-name", "config", "appName is set for app-scoped config discovery")
		}
	}
	if len(buildSpec.ConfigFile.Layers) == 0 {
		report.AddError("config-layers", "config.layers", "config.enabled requires at least one layer")
		return
	}
	for i, layer := range buildSpec.ConfigFile.Layers {
		layer = strings.TrimSpace(layer)
		if !isKnownConfigLayer(layer) {
			report.AddError("config-layer", fmt.Sprintf("config.layers[%d]", i), fmt.Sprintf("unknown config layer %q", layer))
		} else {
			report.AddOK("config-layer", fmt.Sprintf("config.layers[%d]", i), layer)
		}
	}
	report.AddOK("config", "config", fmt.Sprintf("%d config layer(s) declared", len(buildSpec.ConfigFile.Layers)))
}

var knownConfigLayers = map[string]bool{
	"system":   true,
	"xdg":      true,
	"home":     true,
	"git-root": true,
	"cwd":      true,
	"explicit": true,
}

func isKnownConfigLayer(layer string) bool {
	return knownConfigLayers[layer]
}

func usesAppScopedConfigLayer(layers []string) bool {
	for _, layer := range layers {
		switch strings.TrimSpace(layer) {
		case "system", "xdg", "home":
			return true
		}
	}
	return false
}

func validateTarget(report *Report, buildSpec *BuildSpec) {
	kind := strings.TrimSpace(buildSpec.Target.Kind)
	switch kind {
	case "xgoja", "adapter", "cobra", "package":
		report.AddOK("target-kind", "target.kind", fmt.Sprintf("target kind %q is supported", kind))
	default:
		report.AddError("target-kind", "target.kind", fmt.Sprintf("unsupported target kind %q", kind))
	}
	if strings.TrimSpace(buildSpec.Target.Output) == "" {
		report.AddError("target-output", "target.output", "target output is required")
	} else {
		report.AddOK("target-output", "target.output", buildSpec.Target.Output)
	}
	if kind == "adapter" || kind == "cobra" {
		if strings.TrimSpace(buildSpec.Target.Import) == "" {
			report.AddError("target-import", "target.import", fmt.Sprintf("target import is required for %s mode", kind))
		} else {
			report.AddOK("target-import", "target.import", buildSpec.Target.Import)
		}
	}
	if kind == "cobra" {
		if strings.TrimSpace(buildSpec.Target.Root) == "" {
			report.AddError("target-root", "target.root", "target root function is required for cobra mode")
		} else {
			report.AddOK("target-root", "target.root", buildSpec.Target.Root)
		}
	}
	if kind == "package" && strings.TrimSpace(buildSpec.Target.Package) != "" {
		if sanitized := sanitizeGoPackageName(buildSpec.Target.Package); sanitized != strings.TrimSpace(buildSpec.Target.Package) {
			report.AddError("target-package", "target.package", fmt.Sprintf("target package must be a valid Go package identifier; suggested value %q", sanitized))
		} else {
			report.AddOK("target-package", "target.package", buildSpec.Target.Package)
		}
	}
}

func sanitizeGoPackageName(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for i, r := range value {
		valid := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "xgojaruntime"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "xgoja_" + out
	}
	return out
}

func validatePackages(report *Report, buildSpec *BuildSpec) map[string]PackageSpec {
	ids := map[string]PackageSpec{}
	if len(buildSpec.Packages) == 0 {
		report.AddError("packages", "packages", "at least one provider package is required")
		return ids
	}
	for i, pkg := range buildSpec.Packages {
		path := fmt.Sprintf("packages[%d]", i)
		id := strings.TrimSpace(pkg.ID)
		if id == "" {
			report.AddError("package-id", path+".id", "package id is required")
			continue
		}
		if _, ok := ids[id]; ok {
			report.AddError("package-id", path+".id", fmt.Sprintf("duplicate package id %q", id))
			continue
		}
		ids[id] = pkg
		if strings.TrimSpace(pkg.Import) == "" {
			report.AddError("package-import", path+".import", fmt.Sprintf("package %q import is required", id))
		} else {
			report.AddOK("package-import", path+".import", pkg.Import)
		}
		if strings.TrimSpace(pkg.Replace) != "" {
			if err := requireExistingPath(buildSpec.BaseDir, pkg.Replace); err != nil {
				report.AddError("package-replace", path+".replace", err.Error())
			} else {
				report.AddOK("package-replace", path+".replace", pkg.Replace)
			}
		}
	}
	report.AddOK("packages", "packages", fmt.Sprintf("%d package(s) declared", len(ids)))
	return ids
}

func validateModules(report *Report, modules []ModuleInstanceSpec, packageIDs map[string]PackageSpec) {
	if len(modules) == 0 {
		report.AddError("modules", "modules", "at least one module is required")
		return
	}
	aliases := map[string]string{}
	for i, mod := range modules {
		modPath := fmt.Sprintf("modules[%d]", i)
		if strings.TrimSpace(mod.Package) == "" {
			report.AddError("module-package", modPath+".package", "module package is required")
		} else if _, ok := packageIDs[mod.Package]; !ok {
			report.AddError("module-package", modPath+".package", fmt.Sprintf("unknown package id %q", mod.Package))
		}
		if strings.TrimSpace(mod.Name) == "" {
			report.AddError("module-name", modPath+".name", "module name is required")
		}
		alias := strings.TrimSpace(mod.Alias())
		if alias == "" {
			report.AddError("module-alias", modPath+".as", "module alias resolves to empty")
			continue
		}
		if prev, ok := aliases[alias]; ok {
			report.AddError("module-alias", modPath+".as", fmt.Sprintf("duplicate alias %q already used by %s", alias, prev))
			continue
		}
		aliases[alias] = mod.Ref()
	}
	report.AddOK("modules", "modules", fmt.Sprintf("%d module(s) selected", len(modules)))
}

func validateCommands(report *Report, buildSpec *BuildSpec) {
	validateCommand(report, "commands.eval", buildSpec.Commands.Eval)
	validateCommand(report, "commands.run", buildSpec.Commands.Run)
	validateCommand(report, "commands.repl", buildSpec.Commands.Repl)
	validateCommand(report, "commands.jsverbs", buildSpec.Commands.JSVerbs)
	validateJSVerbCommandMount(report, buildSpec.Commands.JSVerbs)
}

func validateJSVerbCommandMount(report *Report, command CommandSpec) {
	mount := strings.TrimSpace(command.Mount)
	if mount == "" {
		return
	}
	switch strings.ToLower(mount) {
	case "root", "/", ".":
		report.AddOK("command-mount", "commands.jsverbs.mount", "root")
	default:
		report.AddError("command-mount", "commands.jsverbs.mount", fmt.Sprintf("unsupported jsverbs mount %q; supported values are root, /, and .", mount))
	}
}

func validateCommand(report *Report, path string, command CommandSpec) {
	if !command.Enabled {
		report.AddOK("command", path, "command disabled")
		return
	}
	report.AddOK("command", path, "command enabled")
}

func validateCommandProviders(report *Report, buildSpec *BuildSpec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, provider := range buildSpec.CommandProviders {
		path := fmt.Sprintf("commandProviders[%d]", i)
		id := strings.TrimSpace(provider.ID)
		if id == "" {
			report.AddError("command-provider-id", path+".id", "command provider id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("command-provider-id", path+".id", fmt.Sprintf("duplicate command provider id %q", id))
		} else {
			ids[id] = struct{}{}
		}
		if strings.TrimSpace(provider.Package) == "" {
			report.AddError("command-provider-package", path+".package", "command provider package is required")
		} else if _, ok := packageIDs[provider.Package]; !ok {
			report.AddError("command-provider-package", path+".package", fmt.Sprintf("unknown package id %q", provider.Package))
		}
		if strings.TrimSpace(provider.Name) == "" {
			report.AddError("command-provider-name", path+".name", "command provider name is required")
		}
	}
	if len(buildSpec.CommandProviders) > 0 {
		report.AddOK("command-providers", "commandProviders", fmt.Sprintf("%d command provider(s) declared", len(buildSpec.CommandProviders)))
	}
}

func validateJSVerbs(report *Report, buildSpec *BuildSpec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, source := range buildSpec.JSVerbs {
		path := fmt.Sprintf("jsverbs[%d]", i)
		id := strings.TrimSpace(source.ID)
		if id == "" {
			report.AddError("jsverb-id", path+".id", "jsverb source id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("jsverb-id", path+".id", fmt.Sprintf("duplicate jsverb source id %q", id))
		} else {
			ids[id] = struct{}{}
		}
		if strings.TrimSpace(source.Package) != "" || strings.TrimSpace(source.Source) != "" {
			if strings.TrimSpace(source.Package) == "" || strings.TrimSpace(source.Source) == "" {
				report.AddError("jsverb-provider-source", path, "provider jsverb sources require both package and source")
				continue
			}
			if _, ok := packageIDs[source.Package]; !ok {
				report.AddError("jsverb-provider-source", path+".package", fmt.Sprintf("unknown package id %q", source.Package))
			} else {
				report.AddOK("jsverb-provider-source", path, fmt.Sprintf("provider source %s.%s", source.Package, source.Source))
			}
			continue
		}
		if strings.TrimSpace(source.Path) == "" {
			report.AddError("jsverb-path", path+".path", "filesystem jsverb source requires path")
			continue
		}
		if source.Embed {
			if err := requireExistingPath(buildSpec.BaseDir, source.Path); err != nil {
				report.AddError("jsverb-path", path+".path", err.Error())
			} else {
				report.AddOK("jsverb-path", path+".path", source.Path)
			}
		} else {
			report.AddOK("jsverb-path", path+".path", "runtime filesystem source")
		}
	}
}

func validateHelp(report *Report, buildSpec *BuildSpec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, source := range buildSpec.Help.Sources {
		path := fmt.Sprintf("help.sources[%d]", i)
		id := strings.TrimSpace(source.ID)
		if id == "" {
			report.AddError("help-source-id", path+".id", "help source id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("help-source-id", path+".id", fmt.Sprintf("duplicate help source id %q", id))
		} else {
			ids[id] = struct{}{}
		}

		hasProvider := strings.TrimSpace(source.Package) != "" || strings.TrimSpace(source.Source) != ""
		hasPath := strings.TrimSpace(source.Path) != ""
		if hasProvider && hasPath {
			report.AddError("help-source-shape", path, "help source cannot combine provider source and filesystem path")
			continue
		}
		if hasProvider {
			if strings.TrimSpace(source.Package) == "" || strings.TrimSpace(source.Source) == "" {
				report.AddError("help-provider-source", path, "provider help sources require both package and source")
				continue
			}
			if _, ok := packageIDs[source.Package]; !ok {
				report.AddError("help-provider-source", path+".package", fmt.Sprintf("unknown package id %q", source.Package))
			} else {
				report.AddOK("help-provider-source", path, fmt.Sprintf("provider source %s.%s", source.Package, source.Source))
			}
			continue
		}
		if !hasPath {
			report.AddError("help-path", path+".path", "filesystem help source requires path")
			continue
		}
		if source.Embed {
			if err := requireExistingDir(buildSpec.BaseDir, source.Path); err != nil {
				report.AddError("help-path", path+".path", err.Error())
			} else {
				report.AddOK("help-path", path+".path", source.Path)
			}
		} else {
			report.AddOK("help-path", path+".path", "runtime filesystem source")
		}
	}
}

func validateAssets(report *Report, buildSpec *BuildSpec) {
	ids := map[string]struct{}{}
	for i, source := range buildSpec.Assets {
		path := fmt.Sprintf("assets[%d]", i)
		id := strings.TrimSpace(source.ID)
		if id == "" {
			report.AddError("asset-id", path+".id", "asset id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("asset-id", path+".id", fmt.Sprintf("duplicate asset id %q", id))
		} else {
			ids[id] = struct{}{}
		}

		if strings.TrimSpace(source.Path) == "" {
			report.AddError("asset-path", path+".path", "asset source requires path")
			continue
		}
		if !source.Embed {
			report.AddError("asset-embed", path+".embed", "asset sources currently require embed: true")
			continue
		}
		if err := requireExistingDir(buildSpec.BaseDir, source.Path); err != nil {
			report.AddError("asset-path", path+".path", err.Error())
		} else {
			report.AddOK("asset-path", path+".path", source.Path)
		}
	}
}

func requireExistingPath(baseDir, rawPath string) error {
	_, _, err := existingPathInfo(baseDir, rawPath)
	return err
}

func requireExistingDir(baseDir, rawPath string) error {
	path, info, err := existingPathInfo(baseDir, rawPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: not a directory", path)
	}
	return nil
}

func existingPathInfo(baseDir, rawPath string) (string, os.FileInfo, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", nil, fmt.Errorf("path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil, fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", path, err)
	}
	return path, info, nil
}
