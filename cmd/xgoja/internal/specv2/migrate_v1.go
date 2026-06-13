package specv2

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/migratebuildspec"
)

type MigrationWarning struct {
	Path    string
	Message string
}

func (w MigrationWarning) String() string {
	if strings.TrimSpace(w.Path) == "" {
		return w.Message
	}
	return w.Path + ": " + w.Message
}

type MigrationResult struct {
	Config   Config
	Warnings []MigrationWarning
}

func MigrateV1(buildSpec *migratebuildspec.BuildSpec) MigrationResult {
	if buildSpec == nil {
		return MigrationResult{Config: Config{Schema: Schema}}
	}

	result := MigrationResult{Config: Config{
		Schema: Schema,
		Name:   buildSpec.Name,
		App: AppSpec{
			Name:      firstNonEmpty(buildSpec.AppName, buildSpec.Name),
			EnvPrefix: buildSpec.EnvPrefix,
		},
		Go: GoSpec{
			Module:  buildSpec.Go.Module,
			Version: buildSpec.Go.Version,
			Tags:    cloneSlice(buildSpec.Go.Tags),
			LDFlags: cloneSlice(buildSpec.Go.LDFlags),
			Env:     cloneMap(buildSpec.Go.Env),
			Imports: migrateGoImports(buildSpec.Go.Imports),
		},
		Workspace: WorkspaceSpec{Mode: "auto"},
	}}
	if buildSpec.ConfigFile != nil {
		result.Config.App.ConfigFile = &ConfigFileSpec{
			Enabled:  buildSpec.ConfigFile.Enabled,
			Layers:   cloneSlice(buildSpec.ConfigFile.Layers),
			FileName: buildSpec.ConfigFile.FileName,
		}
	}

	result.Config.Providers = migratePackages(buildSpec.Packages, &result)
	result.Config.Runtime.Modules = migrateRuntimeModules(buildSpec.Modules)
	result.Config.Sources = migrateSources(buildSpec, &result)
	result.Config.Commands = migrateCommands(buildSpec)
	result.Config.Artifacts = migrateArtifacts(buildSpec, &result)

	ApplyDefaults(&result.Config)
	return result
}

func migrateGoImports(imports []migratebuildspec.GoImportSpec) []GoImportSpec {
	out := make([]GoImportSpec, 0, len(imports))
	for _, imp := range imports {
		out = append(out, GoImportSpec{
			Import:  imp.Import,
			Alias:   imp.Alias,
			Module:  imp.Module,
			Version: imp.Version,
		})
	}
	return out
}

func migratePackages(packages []migratebuildspec.PackageSpec, result *MigrationResult) []ProviderSpec {
	out := make([]ProviderSpec, 0, len(packages))
	for i, pkg := range packages {
		provider := ProviderSpec{
			ID:       pkg.ID,
			Import:   pkg.Import,
			Register: pkg.Register,
			Module: ProviderModuleSpec{
				Version: pkg.Version,
			},
		}
		if strings.TrimSpace(pkg.Replace) != "" {
			provider.Module.Replace = pkg.Replace
			result.warn(fmt.Sprintf("packages[%d].replace", i), "migrated as provider.module.replace; prefer workspace.mode=auto when this replacement is already covered by go.work")
		}
		out = append(out, provider)
	}
	return out
}

func migrateRuntimeModules(modules []migratebuildspec.ModuleInstanceSpec) []RuntimeModuleSpec {
	out := make([]RuntimeModuleSpec, 0, len(modules))
	for _, module := range modules {
		out = append(out, RuntimeModuleSpec{
			Provider: module.Package,
			Name:     module.Name,
			As:       module.As,
			Config:   cloneAnyMap(module.Config),
		})
	}
	return out
}

func migrateSources(buildSpec *migratebuildspec.BuildSpec, result *MigrationResult) []SourceSpec {
	out := []SourceSpec{}
	for i, source := range buildSpec.JSVerbs {
		id := firstNonEmpty(source.ID, fmt.Sprintf("jsverbs-%d", i+1))
		migrated := SourceSpec{
			ID:         id,
			Kind:       SourceKindJSVerbs,
			From:       migrateSourceFrom(source.Path, source.Package, source.Source),
			Include:    cloneSlice(source.Include),
			Exclude:    cloneSlice(source.Exclude),
			Extensions: cloneSlice(source.Extensions),
		}
		if source.TypeScript != nil && source.TypeScript.Enabled {
			migrated.Language = "typescript"
			migrated.Compile = migrateTypeScript(source.TypeScript, buildSpec.Modules, fmt.Sprintf("jsverbs[%d].typescript", i), result)
		} else {
			migrated.Language = inferLanguageFromExtensions(source.Extensions)
		}
		out = append(out, migrated)
	}
	for i, source := range buildSpec.Help.Sources {
		out = append(out, SourceSpec{
			ID:   firstNonEmpty(source.ID, fmt.Sprintf("help-%d", i+1)),
			Kind: SourceKindHelp,
			From: migrateSourceFrom(source.Path, source.Package, source.Source),
		})
	}
	for i, asset := range buildSpec.Assets {
		out = append(out, SourceSpec{
			ID:   firstNonEmpty(asset.ID, fmt.Sprintf("assets-%d", i+1)),
			Kind: SourceKindAssets,
			From: SourceFromSpec{Dir: asset.Path},
		})
	}
	return out
}

func migrateSourceFrom(path, provider, source string) SourceFromSpec {
	if strings.TrimSpace(provider) != "" || strings.TrimSpace(source) != "" {
		return SourceFromSpec{Provider: &ProviderSourceRef{Provider: provider, Source: source}}
	}
	return SourceFromSpec{Dir: path}
}

func migrateTypeScript(ts *migratebuildspec.TypeScriptSpec, modules []migratebuildspec.ModuleInstanceSpec, path string, result *MigrationResult) *CompileSpec {
	compile := &CompileSpec{
		Mode:   "runtime",
		Bundle: ts.Bundle,
		Define: cloneStringMap(ts.Define),
	}
	if len(ts.CheckCommand) > 0 {
		compile.Check = &CompileCheckSpec{Command: cloneSlice(ts.CheckCommand)}
	}
	aliases := map[string]bool{}
	for _, module := range modules {
		aliases[module.Alias()] = true
	}
	for _, external := range ts.External {
		external = strings.TrimSpace(external)
		if external == "" {
			continue
		}
		if aliases[external] {
			result.warn(path+".external", fmt.Sprintf("runtime module alias %q is derived automatically in v2", external))
			continue
		}
		result.warn(path+".external", fmt.Sprintf("non-runtime external %q was not migrated because v2 MVP has no ordinary externals field", external))
	}
	for _, field := range []struct{ name, value string }{
		{"target", ts.Target},
		{"format", ts.Format},
		{"platform", ts.Platform},
		{"tsconfig", ts.Tsconfig},
		{"sourcemap", ts.Sourcemap},
	} {
		if strings.TrimSpace(field.value) != "" {
			result.warn(path+"."+field.name, fmt.Sprintf("%s is an xgoja-owned compiler profile detail in v2 and was not migrated", field.name))
		}
	}
	return compile
}

func migrateCommands(buildSpec *migratebuildspec.BuildSpec) []CommandSurfaceSpec {
	out := []CommandSurfaceSpec{}
	if buildSpec.Commands.Eval.Enabled {
		out = append(out, migrateBuiltinCommand("eval", "builtin.eval", buildSpec.Commands.Eval, nil))
	}
	if buildSpec.Commands.Run.Enabled {
		out = append(out, migrateBuiltinCommand("run", "builtin.run", buildSpec.Commands.Run, nil))
	}
	if buildSpec.Commands.Repl.Enabled {
		out = append(out, migrateBuiltinCommand("repl", "builtin.repl", buildSpec.Commands.Repl, nil))
	}
	if buildSpec.Commands.JSVerbs.Enabled {
		out = append(out, migrateBuiltinCommand("jsverbs", "builtin.jsverbs", buildSpec.Commands.JSVerbs, jsverbSourceIDs(buildSpec.JSVerbs)))
	}
	for i, provider := range buildSpec.CommandProviders {
		id := firstNonEmpty(provider.ID, fmt.Sprintf("command-provider-%d", i+1))
		out = append(out, CommandSurfaceSpec{
			ID:       id,
			Type:     "provider.command-set",
			Provider: provider.Package,
			Name:     provider.Name,
			Mount:    provider.Mount,
			Modules:  cloneSlice(provider.Modules),
			Config:   cloneAnyMap(provider.Config),
			Lazy:     provider.Lazy,
			Sources:  jsverbSourceIDs(buildSpec.JSVerbs),
		})
	}
	return out
}

func migrateBuiltinCommand(defaultID, commandType string, command migratebuildspec.CommandSpec, sources []string) CommandSurfaceSpec {
	return CommandSurfaceSpec{
		ID:      firstNonEmpty(command.Name, defaultID),
		Type:    commandType,
		Name:    command.Name,
		Mount:   command.Mount,
		Sources: cloneSlice(sources),
	}
}

func jsverbSourceIDs(sources []migratebuildspec.JSVerbSourceSpec) []string {
	ids := make([]string, 0, len(sources))
	for i, source := range sources {
		ids = append(ids, firstNonEmpty(source.ID, fmt.Sprintf("jsverbs-%d", i+1)))
	}
	return ids
}

func migrateArtifacts(buildSpec *migratebuildspec.BuildSpec, result *MigrationResult) []ArtifactSpec {
	out := []ArtifactSpec{}
	embeddedExecutableSources := embeddedExecutableSourceIDs(buildSpec, result)
	target := buildSpec.Target
	switch strings.TrimSpace(target.Kind) {
	case "", "xgoja":
		out = append(out, ArtifactSpec{ID: "binary", Type: "binary", Output: target.Output, Sources: embeddedExecutableSources})
	case "package":
		out = append(out, ArtifactSpec{ID: "runtime-package", Type: "runtime-package", Output: target.Output, Package: target.Package, Sources: embeddedExecutableSources})
	case "adapter", "cobra", "source", "template":
		out = append(out, ArtifactSpec{ID: target.Kind, Type: target.Kind, Output: target.Output, Package: target.Package, Import: target.Import, Root: target.Root, Template: target.Template, Sources: embeddedExecutableSources})
	}
	assetSources := []string{}
	for i, asset := range buildSpec.Assets {
		if asset.Embed {
			assetSources = append(assetSources, firstNonEmpty(asset.ID, fmt.Sprintf("assets-%d", i+1)))
		}
	}
	if len(assetSources) > 0 {
		out = append(out, ArtifactSpec{ID: "embedded-assets", Type: "embedded-assets", Sources: assetSources})
	}
	return out
}

func embeddedExecutableSourceIDs(buildSpec *migratebuildspec.BuildSpec, result *MigrationResult) []string {
	out := []string{}
	for i, source := range buildSpec.JSVerbs {
		if source.Embed {
			out = append(out, firstNonEmpty(source.ID, fmt.Sprintf("jsverbs-%d", i+1)))
			result.warn(fmt.Sprintf("jsverbs[%d].embed", i), "embedded jsverb source is represented as an artifact source dependency in v2")
		}
	}
	for i, source := range buildSpec.Help.Sources {
		if source.Embed {
			out = append(out, firstNonEmpty(source.ID, fmt.Sprintf("help-%d", i+1)))
			result.warn(fmt.Sprintf("help.sources[%d].embed", i), "embedded help source is represented as an artifact source dependency in v2")
		}
	}
	return out
}

func (r *MigrationResult) warn(path, message string) {
	r.Warnings = append(r.Warnings, MigrationWarning{Path: path, Message: message})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cloneSlice[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneMap(in map[string]string) map[string]string { return cloneStringMap(in) }

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
