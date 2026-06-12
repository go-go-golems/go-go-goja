package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func WriteAllPlan(dir string, compiled *plan.Plan, opts Options) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	buildSpec := BuildSpecFromPlan(compiled)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbs(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSources(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedAssets(dir, buildSpec); err != nil {
		return err
	}
	files := map[string]string{
		"go.mod":         RenderGoMod(buildSpec, opts),
		"main.go":        RenderMainPlan(compiled),
		"xgoja.gen.json": RenderEmbeddedSpecFromPlan(compiled),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated %s: %w", name, err)
		}
	}
	return nil
}

func WritePackagePlan(dir string, compiled *plan.Plan, opts PackageOptions) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	buildSpec := BuildSpecFromPlan(compiled)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbs(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSources(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedAssets(dir, buildSpec); err != nil {
		return err
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(dir)
	}
	content := RenderPackagePlan(compiled, packageName)
	if err := os.WriteFile(filepath.Join(dir, "xgoja_runtime.gen.go"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write generated package: %w", err)
	}
	return nil
}

func WriteSourceFragmentsPlan(dir string, compiled *plan.Plan, opts PackageOptions) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	buildSpec := BuildSpecFromPlan(compiled)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbs(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSources(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedAssets(dir, buildSpec); err != nil {
		return err
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(dir)
	}
	for name, content := range RenderSourceFragmentsPlan(compiled, packageName) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated source fragment %s: %w", name, err)
		}
	}
	return nil
}

func WriteCustomTemplatePlan(outputFile string, compiled *plan.Plan, opts TemplateOptions) error {
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("custom template output file is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return fmt.Errorf("create custom template output directory: %w", err)
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(filepath.Dir(outputFile))
	}
	content, err := loadCustomTemplate(opts.TemplatePath, packageTemplateDataFromPlan(compiled, packageName))
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write custom template output %s: %w", outputFile, err)
	}
	return nil
}

func TemplateDataJSONFromPlan(compiled *plan.Plan, packageName string) (string, error) {
	if compiled == nil {
		return "", fmt.Errorf("plan is nil")
	}
	data, err := json.MarshalIndent(packageTemplateDataFromPlan(compiled, packageName), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// BuildSpecFromPlan adapts the v2 planner output to the current generator's
// rendering model. It is intentionally contained in the generator package so
// CLI commands consume plan.Plan directly and legacy buildspec loading remains
// isolated to migrate-spec.
func BuildSpecFromPlan(compiled *plan.Plan) *buildspec.BuildSpec {
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
			Env:     cloneStringMapFromPlan(cfg.Go.Env),
		},
		Target:  targetFromPlanArtifacts(cfg.Artifacts),
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
		applyPlanCommand(&out.Commands, &out.CommandProviders, command)
	}
	embeddedSources := embeddedSourceIDsFromPlanArtifacts(cfg.Artifacts)
	embeddedAssets := embeddedAssetIDsFromPlanArtifacts(cfg.Artifacts)
	for _, source := range cfg.Sources {
		switch source.Kind {
		case specv2.SourceKindJSVerbs:
			out.JSVerbs = append(out.JSVerbs, jsVerbSourceFromPlan(source, embeddedSources[source.ID]))
		case specv2.SourceKindAssets:
			out.Assets = append(out.Assets, assetSourceFromPlan(source, embeddedAssets[source.ID]))
		case specv2.SourceKindHelp:
			out.Help.Sources = append(out.Help.Sources, helpSourceFromPlan(source, embeddedSources[source.ID]))
		case specv2.SourceKindScript:
			// Script sources are consumed by run/runtime planning, not by the current generator renderer.
		}
	}
	return out
}

func targetFromPlanArtifacts(artifacts []specv2.ArtifactSpec) buildspec.TargetSpec {
	for _, artifact := range artifacts {
		if artifact.Type == "binary" {
			return buildspec.TargetSpec{Kind: "xgoja", Output: artifact.Output}
		}
		if artifact.Type == "runtime-package" {
			return buildspec.TargetSpec{Kind: "package", Output: artifact.Output, Package: artifact.Package, Import: artifact.Import, Root: artifact.Root, Template: artifact.Template}
		}
		if artifact.Type != "" {
			return buildspec.TargetSpec{Kind: artifact.Type, Output: artifact.Output, Package: artifact.Package, Import: artifact.Import, Root: artifact.Root, Template: artifact.Template}
		}
	}
	return buildspec.TargetSpec{Kind: "xgoja", Output: "dist/xgoja-app"}
}

func applyPlanCommand(commands *buildspec.CommandsSpec, providers *[]buildspec.CommandProviderInstanceSpec, command specv2.CommandSurfaceSpec) {
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

func jsVerbSourceFromPlan(source specv2.SourceSpec, embed bool) buildspec.JSVerbSourceSpec {
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
			out.TypeScript.Define = cloneStringMapFromPlan(source.Compile.Define)
			if source.Compile.Check != nil {
				out.TypeScript.CheckCommand = append([]string(nil), source.Compile.Check.Command...)
			}
		}
	}
	return out
}

func assetSourceFromPlan(source specv2.SourceSpec, embed bool) buildspec.AssetSourceSpec {
	return buildspec.AssetSourceSpec{ID: source.ID, Path: source.From.Dir, Embed: embed}
}

func helpSourceFromPlan(source specv2.SourceSpec, embed bool) buildspec.HelpSourceSpec {
	out := buildspec.HelpSourceSpec{ID: source.ID, Embed: embed}
	if source.From.Provider != nil {
		out.Package = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else {
		out.Path = source.From.Dir
	}
	return out
}

func embeddedSourceIDsFromPlanArtifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
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

func embeddedAssetIDsFromPlanArtifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
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

func cloneStringMapFromPlan(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
