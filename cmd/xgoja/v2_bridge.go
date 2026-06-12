package main

import (
	"os"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func loadBuildSpecOrV2Plan(file string) (*buildspec.BuildSpec, *plan.Plan, bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, false, err
	}
	kind, _, err := specv2.DetectSchema(data)
	if err != nil {
		return nil, nil, false, err
	}
	if kind != specv2.SchemaKindV2 {
		buildSpec, _, err := buildspec.LoadFile(file)
		return buildSpec, nil, false, err
	}
	cfg, err := specv2.LoadFile(file)
	if err != nil {
		return nil, nil, true, err
	}
	compiled, err := plan.Compile(plan.Options{Config: *cfg, Providers: syntheticProviderRegistryFromV2(cfg), StartDir: cfg.BaseDir})
	if err != nil {
		return nil, nil, true, err
	}
	return buildSpecFromV2Plan(compiled), compiled, true, nil
}

func buildSpecFromV2Plan(compiled *plan.Plan) *buildspec.BuildSpec {
	cfg := compiled.Config
	out := &buildspec.BuildSpec{
		Name:      cfg.Name,
		AppName:   cfg.App.Name,
		EnvPrefix: cfg.App.EnvPrefix,
		Go: buildspec.GoSpec{
			Version: cfg.Go.Version,
			Module:  cfg.Go.Module,
			Tags:    append([]string(nil), cfg.Go.Tags...),
			LDFlags: append([]string(nil), cfg.Go.LDFlags...),
			Env:     cloneStringMapV2(cfg.Go.Env),
		},
		Target:  targetFromV2Artifacts(cfg.Artifacts),
		BaseDir: cfg.BaseDir,
	}
	if cfg.App.ConfigFile != nil {
		out.ConfigFile = &buildspec.ConfigFileSpec{Enabled: cfg.App.ConfigFile.Enabled, Layers: append([]string(nil), cfg.App.ConfigFile.Layers...), FileName: cfg.App.ConfigFile.FileName}
	}
	for _, goImport := range cfg.Go.Imports {
		out.Go.Imports = append(out.Go.Imports, buildspec.GoImportSpec{Import: goImport.Import, Alias: goImport.Alias, Module: goImport.Module, Version: goImport.Version})
	}
	for _, provider := range cfg.Providers {
		out.Packages = append(out.Packages, buildspec.PackageSpec{ID: provider.ID, Import: provider.Import, Version: provider.Module.Version, Register: provider.Register, Replace: provider.Module.Replace})
	}
	for _, module := range cfg.Runtime.Modules {
		out.Modules = append(out.Modules, buildspec.ModuleInstanceSpec{Package: module.Provider, Name: module.Name, As: module.As, Config: module.Config})
	}
	for _, command := range cfg.Commands {
		applyV2Command(&out.Commands, &out.CommandProviders, command)
	}
	embeddedSources := embeddedSourceIDsFromV2Artifacts(cfg.Artifacts)
	embeddedAssets := embeddedAssetIDsFromV2Artifacts(cfg.Artifacts)
	for _, source := range cfg.Sources {
		switch source.Kind {
		case specv2.SourceKindJSVerbs:
			out.JSVerbs = append(out.JSVerbs, jsVerbSourceFromV2(source, embeddedSources[source.ID]))
		case specv2.SourceKindAssets:
			out.Assets = append(out.Assets, assetSourceFromV2(source, embeddedAssets[source.ID]))
		case specv2.SourceKindHelp:
			out.Help.Sources = append(out.Help.Sources, helpSourceFromV2(source, embeddedSources[source.ID]))
		case specv2.SourceKindScript:
			// Script sources are consumed by run/runtime planning, not by the current buildspec bridge.
		}
	}
	return out
}

func targetFromV2Artifacts(artifacts []specv2.ArtifactSpec) buildspec.TargetSpec {
	for _, artifact := range artifacts {
		if artifact.Type == "binary" {
			return buildspec.TargetSpec{Kind: "xgoja", Output: artifact.Output}
		}
		if artifact.Type != "" {
			return buildspec.TargetSpec{Kind: artifact.Type, Output: artifact.Output, Package: artifact.Package, Import: artifact.Import, Root: artifact.Root, Template: artifact.Template}
		}
	}
	return buildspec.TargetSpec{Kind: "xgoja", Output: "dist/xgoja-app"}
}

func applyV2Command(commands *buildspec.CommandsSpec, providers *[]buildspec.CommandProviderInstanceSpec, command specv2.CommandSurfaceSpec) {
	spec := buildspec.CommandSpec{Enabled: true, Name: command.Name, Mount: command.Mount}
	switch command.Type {
	case "builtin.eval":
		commands.Eval = spec
	case "builtin.run":
		commands.Run = spec
	case "builtin.repl":
		commands.Repl = spec
	case "builtin.jsverbs":
		commands.JSVerbs = spec
	case "provider.command-set":
		*providers = append(*providers, buildspec.CommandProviderInstanceSpec{ID: command.ID, Package: command.Provider, Name: command.Name, Mount: command.Mount, Modules: append([]string(nil), command.Modules...), Config: command.Config, Lazy: command.Lazy})
	}
}

func jsVerbSourceFromV2(source specv2.SourceSpec, embed bool) buildspec.JSVerbSourceSpec {
	out := buildspec.JSVerbSourceSpec{ID: source.ID, Embed: embed, Include: append([]string(nil), source.Include...), Exclude: append([]string(nil), source.Exclude...), Extensions: append([]string(nil), source.Extensions...)}
	if source.From.Provider != nil {
		out.Package = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else {
		out.Path = source.From.Dir
	}
	if strings.EqualFold(source.Language, "typescript") || source.Compile != nil {
		out.TypeScript = &buildspec.TypeScriptSpec{Enabled: strings.EqualFold(source.Language, "typescript"), Bundle: source.Compile != nil && source.Compile.Bundle}
		if source.Compile != nil {
			out.TypeScript.Define = cloneStringMapV2(source.Compile.Define)
			if source.Compile.Check != nil {
				out.TypeScript.CheckCommand = append([]string(nil), source.Compile.Check.Command...)
			}
		}
	}
	return out
}

func assetSourceFromV2(source specv2.SourceSpec, embed bool) buildspec.AssetSourceSpec {
	return buildspec.AssetSourceSpec{ID: source.ID, Path: source.From.Dir, Embed: embed}
}

func helpSourceFromV2(source specv2.SourceSpec, embed bool) buildspec.HelpSourceSpec {
	out := buildspec.HelpSourceSpec{ID: source.ID, Embed: embed}
	if source.From.Provider != nil {
		out.Package = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else {
		out.Path = source.From.Dir
	}
	return out
}

func embeddedSourceIDsFromV2Artifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
	out := map[string]bool{}
	for _, artifact := range artifacts {
		switch artifact.Type {
		case "binary", "runtime-package", "source", "template", "adapter", "cobra":
			for _, sourceID := range artifact.Sources {
				if strings.TrimSpace(sourceID) != "" {
					out[sourceID] = true
				}
			}
		}
	}
	return out
}

func embeddedAssetIDsFromV2Artifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
	out := map[string]bool{}
	for _, artifact := range artifacts {
		if artifact.Type != "embedded-assets" {
			continue
		}
		for _, sourceID := range artifact.Sources {
			if strings.TrimSpace(sourceID) != "" {
				out[sourceID] = true
			}
		}
	}
	return out
}

func cloneStringMapV2(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
