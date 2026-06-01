package buildspec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Validate(spec *Spec) *Report {
	report := &Report{}
	if spec == nil {
		report.AddError("spec", "", "spec is nil")
		return report
	}

	validateName(report, spec)
	validateTarget(report, spec)
	packageIDs := validatePackages(report, spec)
	validateRuntimes(report, spec, packageIDs)
	validateCommands(report, spec)
	validateCommandProviders(report, spec, packageIDs)
	validateJSVerbs(report, spec, packageIDs)
	validateHelp(report, spec, packageIDs)
	validateAssets(report, spec)

	return report
}

func validateName(report *Report, spec *Spec) {
	if strings.TrimSpace(spec.Name) == "" {
		report.AddError("name", "name", "name is required")
		return
	}
	report.AddOK("name", "name", fmt.Sprintf("spec name is %q", spec.Name))
}

func validateTarget(report *Report, spec *Spec) {
	kind := strings.TrimSpace(spec.Target.Kind)
	switch kind {
	case "xgoja", "adapter", "cobra":
		report.AddOK("target-kind", "target.kind", fmt.Sprintf("target kind %q is supported", kind))
	default:
		report.AddError("target-kind", "target.kind", fmt.Sprintf("unsupported target kind %q", kind))
	}
	if strings.TrimSpace(spec.Target.Output) == "" {
		report.AddError("target-output", "target.output", "target output is required")
	} else {
		report.AddOK("target-output", "target.output", spec.Target.Output)
	}
	if kind == "adapter" || kind == "cobra" {
		if strings.TrimSpace(spec.Target.Import) == "" {
			report.AddError("target-import", "target.import", fmt.Sprintf("target import is required for %s mode", kind))
		} else {
			report.AddOK("target-import", "target.import", spec.Target.Import)
		}
	}
	if kind == "cobra" {
		if strings.TrimSpace(spec.Target.Root) == "" {
			report.AddError("target-root", "target.root", "target root function is required for cobra mode")
		} else {
			report.AddOK("target-root", "target.root", spec.Target.Root)
		}
	}
}

func validatePackages(report *Report, spec *Spec) map[string]PackageSpec {
	ids := map[string]PackageSpec{}
	if len(spec.Packages) == 0 {
		report.AddError("packages", "packages", "at least one provider package is required")
		return ids
	}
	for i, pkg := range spec.Packages {
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
			if err := requireExistingPath(spec.BaseDir, pkg.Replace); err != nil {
				report.AddError("package-replace", path+".replace", err.Error())
			} else {
				report.AddOK("package-replace", path+".replace", pkg.Replace)
			}
		}
	}
	report.AddOK("packages", "packages", fmt.Sprintf("%d package(s) declared", len(ids)))
	return ids
}

func validateRuntimes(report *Report, spec *Spec, packageIDs map[string]PackageSpec) {
	if len(spec.Runtimes) == 0 {
		report.AddError("runtimes", "runtimes", "at least one runtime profile is required")
		return
	}
	for runtimeName, runtime := range spec.Runtimes {
		path := "runtimes." + runtimeName
		if strings.TrimSpace(runtimeName) == "" {
			report.AddError("runtime-name", "runtimes", "runtime profile name is empty")
			continue
		}
		if len(runtime.Modules) == 0 {
			report.AddError("runtime-modules", path+".modules", "runtime must select at least one module")
			continue
		}
		aliases := map[string]string{}
		for i, mod := range runtime.Modules {
			modPath := fmt.Sprintf("%s.modules[%d]", path, i)
			if strings.TrimSpace(mod.Package) == "" {
				report.AddError("runtime-module-package", modPath+".package", "module package is required")
			} else if _, ok := packageIDs[mod.Package]; !ok {
				report.AddError("runtime-module-package", modPath+".package", fmt.Sprintf("unknown package id %q", mod.Package))
			}
			if strings.TrimSpace(mod.Name) == "" {
				report.AddError("runtime-module-name", modPath+".name", "module name is required")
			}
			alias := strings.TrimSpace(mod.Alias())
			if alias == "" {
				report.AddError("runtime-module-alias", modPath+".as", "module alias resolves to empty")
				continue
			}
			if prev, ok := aliases[alias]; ok {
				report.AddError("runtime-module-alias", modPath+".as", fmt.Sprintf("duplicate alias %q already used by %s", alias, prev))
				continue
			}
			aliases[alias] = mod.Ref()
		}
		report.AddOK("runtime", path, fmt.Sprintf("%d module(s) selected", len(runtime.Modules)))
	}
}

func validateCommands(report *Report, spec *Spec) {
	validateCommandRuntime(report, "commands.eval", spec.Commands.Eval, spec.Runtimes)
	validateCommandRuntime(report, "commands.run", spec.Commands.Run, spec.Runtimes)
	validateCommandRuntime(report, "commands.repl", spec.Commands.Repl, spec.Runtimes)
	validateCommandRuntime(report, "commands.jsverbs", spec.Commands.JSVerbs, spec.Runtimes)
}

func validateCommandRuntime(report *Report, path string, command CommandSpec, runtimes map[string]Runtime) {
	if !command.Enabled {
		report.AddOK("command", path, "command disabled")
		return
	}
	if strings.TrimSpace(command.Runtime) == "" {
		report.AddError("command-runtime", path+".runtime", "enabled command requires a runtime profile")
		return
	}
	if _, ok := runtimes[command.Runtime]; !ok {
		report.AddError("command-runtime", path+".runtime", fmt.Sprintf("unknown runtime profile %q", command.Runtime))
		return
	}
	report.AddOK("command-runtime", path+".runtime", command.Runtime)
}

func validateCommandProviders(report *Report, spec *Spec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, provider := range spec.CommandProviders {
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
		if strings.TrimSpace(provider.RuntimeProfile) != "" {
			if _, ok := spec.Runtimes[provider.RuntimeProfile]; !ok {
				report.AddError("command-provider-runtime", path+".runtimeProfile", fmt.Sprintf("unknown runtime profile %q", provider.RuntimeProfile))
			} else {
				report.AddOK("command-provider-runtime", path+".runtimeProfile", provider.RuntimeProfile)
			}
		}
	}
	if len(spec.CommandProviders) > 0 {
		report.AddOK("command-providers", "commandProviders", fmt.Sprintf("%d command provider(s) declared", len(spec.CommandProviders)))
	}
}

func validateJSVerbs(report *Report, spec *Spec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, source := range spec.JSVerbs {
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
			if err := requireExistingPath(spec.BaseDir, source.Path); err != nil {
				report.AddError("jsverb-path", path+".path", err.Error())
			} else {
				report.AddOK("jsverb-path", path+".path", source.Path)
			}
		} else {
			report.AddOK("jsverb-path", path+".path", "runtime filesystem source")
		}
	}
}

func validateHelp(report *Report, spec *Spec, packageIDs map[string]PackageSpec) {
	ids := map[string]struct{}{}
	for i, source := range spec.Help.Sources {
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
			if err := requireExistingDir(spec.BaseDir, source.Path); err != nil {
				report.AddError("help-path", path+".path", err.Error())
			} else {
				report.AddOK("help-path", path+".path", source.Path)
			}
		} else {
			report.AddOK("help-path", path+".path", "runtime filesystem source")
		}
	}
}

func validateAssets(report *Report, spec *Spec) {
	ids := map[string]struct{}{}
	for i, source := range spec.Assets {
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
		if err := requireExistingDir(spec.BaseDir, source.Path); err != nil {
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
