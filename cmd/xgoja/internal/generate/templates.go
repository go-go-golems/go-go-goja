package generate

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type mainTemplateData struct {
	SpecJSON          string
	HasEmbedded       bool
	HasEmbeddedJSVerb bool
	HasEmbeddedHelp   bool
	HasEmbeddedAssets bool
	NeedsContext      bool
	HasTargetImport   bool
	TargetKind        string
	TargetImport      string
	TargetRoot        string
	HostConstruction  string
	RootConstruction  string
	ProviderImports   []providerImport
	ExtraImports      []extraImport
}

type packageTemplateData struct {
	PackageName       string
	SpecJSON          string
	HasEmbedded       bool
	HasEmbeddedJSVerb bool
	HasEmbeddedHelp   bool
	HasEmbeddedAssets bool
	ProviderImports   []providerImport
	ExtraImports      []extraImport
}

type dtsGenTemplateData struct {
	SpecJSON        string
	ProviderImports []providerImport
	ExtraImports    []extraImport
	Strict          bool
}

type providerImport struct {
	Alias    string
	Import   string
	Register string
}

type extraImport struct {
	Alias       string
	AliasPrefix string
	Import      string
}

func renderMainTemplate(data mainTemplateData) (string, error) {
	return renderTemplate("main.go.tmpl", data, "generated main.go")
}

func renderPackageTemplate(data packageTemplateData) (string, error) {
	return renderTemplate("runtime_package.go.tmpl", data, "generated runtime package")
}

func renderSpecFragmentTemplate(data packageTemplateData) (string, error) {
	return renderTemplate("spec_fragment.go.tmpl", data, "generated spec fragment")
}

func renderProvidersFragmentTemplate(data packageTemplateData) (string, error) {
	return renderTemplate("providers_fragment.go.tmpl", data, "generated providers fragment")
}

func renderEmbedFragmentTemplate(data packageTemplateData) (string, error) {
	return renderTemplate("embed_fragment.go.tmpl", data, "generated embed fragment")
}

func renderBundleFragmentTemplate(data packageTemplateData) (string, error) {
	return renderTemplate("bundle_fragment.go.tmpl", data, "generated bundle fragment")
}

func renderDTSGenMainTemplate(data dtsGenTemplateData) (string, error) {
	return renderTemplate("dtsgen_main.go.tmpl", data, "generated dtsgen main.go")
}

func renderCustomTemplate(path string, data packageTemplateData) (string, error) {
	tmpl, err := template.New(filepathBase(path)).Funcs(templateFuncs()).ParseFiles(path)
	if err != nil {
		return "", fmt.Errorf("parse custom template %s: %w", path, err)
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute custom template %s: %w", path, err)
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return "", fmt.Errorf("format custom template output %s: %w\n%s", path, err, b.String())
	}
	return string(formatted), nil
}

func renderTemplate(name string, data any, label string) (string, error) {
	tmpl, err := template.New(name).Funcs(templateFuncs()).ParseFS(templateFS, "templates/"+name)
	if err != nil {
		return "", fmt.Errorf("parse %s template: %w", name, err)
	}
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, name, data); err != nil {
		return "", fmt.Errorf("execute %s template: %w", name, err)
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return "", fmt.Errorf("format %s: %w\n%s", label, err, b.String())
	}
	return string(formatted), nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"quote":     strconv.Quote,
		"rawString": escapeRawString,
		"json": func(v any) (string, error) {
			data, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}
}

func filepathBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "custom.tmpl"
	}
	idx := strings.LastIndexAny(path, `/\\`)
	if idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func loadCustomTemplate(path string, data packageTemplateData) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("custom template path is required")
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return renderCustomTemplate(path, data)
}

func mainTemplateDataFromPlan(compiled *plan.Plan) mainTemplateData {
	cfg := compiled.Config
	paths := embeddedPlanPaths(cfg)
	hasEmbeddedJSVerb := len(paths.JSVerbRoots) > 0
	hasEmbeddedHelp := len(paths.HelpRoots) > 0
	hasEmbeddedAssets := len(paths.AssetRoots) > 0
	hasEmbedded := hasEmbeddedJSVerb || hasEmbeddedHelp || hasEmbeddedAssets
	target := targetDataFromPlanArtifacts(cfg.Artifacts)
	rootFn := strings.TrimSpace(target.Root)
	if rootFn == "" {
		rootFn = "NewRootCommand"
	}
	data := mainTemplateData{
		SpecJSON:          escapeRawString(RenderEmbeddedSpecFromPlan(compiled)),
		HasEmbedded:       hasEmbedded,
		HasEmbeddedJSVerb: hasEmbeddedJSVerb,
		HasEmbeddedHelp:   hasEmbeddedHelp,
		HasEmbeddedAssets: hasEmbeddedAssets,
		NeedsContext:      target.Kind == "adapter",
		HasTargetImport:   target.Kind == "adapter" || target.Kind == "cobra",
		TargetKind:        target.Kind,
		TargetImport:      target.Import,
		TargetRoot:        rootFn,
		ProviderImports:   providerImportsFromPlan(compiled),
		ExtraImports:      extraImportsFromPlan(compiled),
	}
	embeddedJSVerbArg := "nil"
	if hasEmbeddedJSVerb {
		embeddedJSVerbArg = "embeddedJSVerbs"
	}
	embeddedHelpArg := "nil"
	if hasEmbeddedHelp {
		embeddedHelpArg = "embeddedHelp"
	}
	embeddedAssetsArg := "nil"
	if hasEmbeddedAssets {
		embeddedAssetsArg = "embeddedAssets"
	}
	if hasEmbedded {
		data.HostConstruction = fmt.Sprintf("host := app.NewHostWithOptions(registry, buildSpec, app.HostOptions{EmbeddedJSVerbs: %s, EmbeddedHelp: %s, EmbeddedAssets: %s})", embeddedJSVerbArg, embeddedHelpArg, embeddedAssetsArg)
		data.RootConstruction = fmt.Sprintf("root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON, EmbeddedJSVerbs: %s, EmbeddedHelp: %s, EmbeddedAssets: %s})", embeddedJSVerbArg, embeddedHelpArg, embeddedAssetsArg)
	} else {
		data.HostConstruction = "host := app.NewHost(registry, buildSpec)"
		data.RootConstruction = "root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON})"
	}
	return data
}

func dtsGenTemplateDataFromPlan(compiled *plan.Plan, strict bool) dtsGenTemplateData {
	return dtsGenTemplateData{
		SpecJSON:        escapeRawString(RenderEmbeddedSpecFromPlan(compiled)),
		ProviderImports: providerImportsFromPlan(compiled),
		ExtraImports:    extraImportsFromPlan(compiled),
		Strict:          strict,
	}
}

func packageTemplateDataFromPlan(compiled *plan.Plan, packageName string) packageTemplateData {
	paths := embeddedPlanPaths(compiled.Config)
	hasEmbeddedJSVerb := len(paths.JSVerbRoots) > 0
	hasEmbeddedHelp := len(paths.HelpRoots) > 0
	hasEmbeddedAssets := len(paths.AssetRoots) > 0
	return packageTemplateData{
		PackageName:       packageName,
		SpecJSON:          escapeRawString(RenderEmbeddedSpecFromPlan(compiled)),
		HasEmbedded:       hasEmbeddedJSVerb || hasEmbeddedHelp || hasEmbeddedAssets,
		HasEmbeddedJSVerb: hasEmbeddedJSVerb,
		HasEmbeddedHelp:   hasEmbeddedHelp,
		HasEmbeddedAssets: hasEmbeddedAssets,
		ProviderImports:   providerImportsFromPlan(compiled),
		ExtraImports:      extraImportsFromPlan(compiled),
	}
}

func mainTemplateDataFromSpec(buildSpec *buildspec.BuildSpec) mainTemplateData {
	hasEmbeddedJSVerb := hasEmbeddedJSVerbSources(buildSpec)
	hasEmbeddedHelp := hasEmbeddedHelpSources(buildSpec)
	hasEmbeddedAssets := hasEmbeddedAssetSources(buildSpec)
	hasEmbedded := hasEmbeddedJSVerb || hasEmbeddedHelp || hasEmbeddedAssets

	rootFn := strings.TrimSpace(buildSpec.Target.Root)
	if rootFn == "" {
		rootFn = "NewRootCommand"
	}

	data := mainTemplateData{
		SpecJSON:          escapeRawString(RenderEmbeddedSpec(buildSpec)),
		HasEmbedded:       hasEmbedded,
		HasEmbeddedJSVerb: hasEmbeddedJSVerb,
		HasEmbeddedHelp:   hasEmbeddedHelp,
		HasEmbeddedAssets: hasEmbeddedAssets,
		NeedsContext:      buildSpec.Target.Kind == "adapter",
		HasTargetImport:   buildSpec.Target.Kind == "adapter" || buildSpec.Target.Kind == "cobra",
		TargetKind:        buildSpec.Target.Kind,
		TargetImport:      buildSpec.Target.Import,
		TargetRoot:        rootFn,
		ProviderImports:   providerImportsFromSpec(buildSpec),
		ExtraImports:      extraImportsFromSpec(buildSpec),
	}
	embeddedJSVerbArg := "nil"
	if hasEmbeddedJSVerb {
		embeddedJSVerbArg = "embeddedJSVerbs"
	}
	embeddedHelpArg := "nil"
	if hasEmbeddedHelp {
		embeddedHelpArg = "embeddedHelp"
	}
	embeddedAssetsArg := "nil"
	if hasEmbeddedAssets {
		embeddedAssetsArg = "embeddedAssets"
	}
	if hasEmbedded {
		data.HostConstruction = fmt.Sprintf("host := app.NewHostWithOptions(registry, buildSpec, app.HostOptions{EmbeddedJSVerbs: %s, EmbeddedHelp: %s, EmbeddedAssets: %s})", embeddedJSVerbArg, embeddedHelpArg, embeddedAssetsArg)
		data.RootConstruction = fmt.Sprintf("root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON, EmbeddedJSVerbs: %s, EmbeddedHelp: %s, EmbeddedAssets: %s})", embeddedJSVerbArg, embeddedHelpArg, embeddedAssetsArg)
	} else {
		data.HostConstruction = "host := app.NewHost(registry, buildSpec)"
		data.RootConstruction = "root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON})"
	}
	return data
}

func dtsGenTemplateDataFromSpec(buildSpec *buildspec.BuildSpec, strict bool) dtsGenTemplateData {
	return dtsGenTemplateData{
		SpecJSON:        escapeRawString(RenderEmbeddedSpec(buildSpec)),
		ProviderImports: providerImportsFromSpec(buildSpec),
		ExtraImports:    extraImportsFromSpec(buildSpec),
		Strict:          strict,
	}
}

func packageTemplateDataFromSpec(buildSpec *buildspec.BuildSpec, packageName string) packageTemplateData {
	hasEmbeddedJSVerb := hasEmbeddedJSVerbSources(buildSpec)
	hasEmbeddedHelp := hasEmbeddedHelpSources(buildSpec)
	hasEmbeddedAssets := hasEmbeddedAssetSources(buildSpec)
	return packageTemplateData{
		PackageName:       packageName,
		SpecJSON:          escapeRawString(RenderEmbeddedSpec(buildSpec)),
		HasEmbedded:       hasEmbeddedJSVerb || hasEmbeddedHelp || hasEmbeddedAssets,
		HasEmbeddedJSVerb: hasEmbeddedJSVerb,
		HasEmbeddedHelp:   hasEmbeddedHelp,
		HasEmbeddedAssets: hasEmbeddedAssets,
		ProviderImports:   providerImportsFromSpec(buildSpec),
		ExtraImports:      extraImportsFromSpec(buildSpec),
	}
}

func extraImportsFromSpec(buildSpec *buildspec.BuildSpec) []extraImport {
	if buildSpec == nil {
		return nil
	}
	seen := map[string]struct{}{}
	ret := make([]extraImport, 0, len(buildSpec.Go.Imports))
	for _, imp := range buildSpec.Go.Imports {
		importPath := strings.TrimSpace(imp.Import)
		if importPath == "" {
			continue
		}
		alias := strings.TrimSpace(imp.Alias)
		key := alias + "\x00" + importPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		aliasPrefix := ""
		if alias != "" {
			aliasPrefix = alias + " "
		}
		ret = append(ret, extraImport{Alias: alias, AliasPrefix: aliasPrefix, Import: importPath})
	}
	return ret
}

func providerImportsFromSpec(buildSpec *buildspec.BuildSpec) []providerImport {
	aliases := importAliases(buildSpec.Packages)
	providers := make([]providerImport, 0, len(buildSpec.Packages))
	for _, pkg := range buildSpec.Packages {
		providers = append(providers, providerImport{
			Alias:    aliases[pkg.ID],
			Import:   pkg.Import,
			Register: pkg.Register,
		})
	}
	return providers
}

func providerImportsFromPlan(compiled *plan.Plan) []providerImport {
	if compiled == nil {
		return nil
	}
	packages := make([]buildspec.PackageSpec, 0, len(compiled.Config.Providers))
	for _, provider := range compiled.Config.Providers {
		packages = append(packages, buildspec.PackageSpec{ID: provider.ID, Import: provider.Import, Register: provider.Register})
	}
	aliases := importAliases(packages)
	providers := make([]providerImport, 0, len(compiled.Config.Providers))
	for _, provider := range compiled.Config.Providers {
		providers = append(providers, providerImport{Alias: aliases[provider.ID], Import: provider.Import, Register: provider.Register})
	}
	return providers
}

func extraImportsFromPlan(compiled *plan.Plan) []extraImport {
	if compiled == nil {
		return nil
	}
	seen := map[string]struct{}{}
	ret := make([]extraImport, 0, len(compiled.Config.Go.Imports))
	for _, imp := range compiled.Config.Go.Imports {
		importPath := strings.TrimSpace(imp.Import)
		if importPath == "" {
			continue
		}
		alias := strings.TrimSpace(imp.Alias)
		key := alias + "\x00" + importPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		aliasPrefix := ""
		if alias != "" {
			aliasPrefix = alias + " "
		}
		ret = append(ret, extraImport{Alias: alias, AliasPrefix: aliasPrefix, Import: importPath})
	}
	return ret
}

type planTargetData struct {
	Kind   string
	Import string
	Root   string
}

func targetDataFromPlanArtifacts(artifacts []specv2.ArtifactSpec) planTargetData {
	for _, artifact := range artifacts {
		if artifact.Type == "binary" {
			return planTargetData{Kind: "xgoja"}
		}
		if artifact.Type == "runtime-package" {
			return planTargetData{Kind: "package", Import: artifact.Import, Root: artifact.Root}
		}
		if artifact.Type != "" {
			return planTargetData{Kind: artifact.Type, Import: artifact.Import, Root: artifact.Root}
		}
	}
	return planTargetData{Kind: "xgoja"}
}

type planEmbeddedPaths struct {
	JSVerbRoots map[string]string
	HelpRoots   map[string]string
	AssetRoots  map[string]string
}

func embeddedPlanPaths(cfg specv2.Config) planEmbeddedPaths {
	embeddedSources := embeddedSourceIDsFromPlanArtifacts(cfg.Artifacts)
	embeddedAssets := embeddedAssetIDsFromPlanArtifacts(cfg.Artifacts)
	return planEmbeddedPaths{
		JSVerbRoots: embeddedPlanRoots(cfg.Sources, specv2.SourceKindJSVerbs, embeddedSources, "xgoja_embed/jsverbs"),
		HelpRoots:   embeddedPlanRoots(cfg.Sources, specv2.SourceKindHelp, embeddedSources, "xgoja_embed/help"),
		AssetRoots:  embeddedPlanRoots(cfg.Sources, specv2.SourceKindAssets, embeddedAssets, "xgoja_embed/assets"),
	}
}

func embeddedPlanRoots(sources []specv2.SourceSpec, kind specv2.SourceKind, embedded map[string]bool, prefix string) map[string]string {
	roots := map[string]string{}
	used := map[string]struct{}{}
	for _, source := range sources {
		if source.Kind != kind || !embedded[source.ID] || strings.TrimSpace(source.From.Dir) == "" || source.From.Provider != nil {
			continue
		}
		base := sanitizeIdentifier(source.ID)
		if base == "" {
			base = "source"
		}
		name := base
		for suffix := 2; ; suffix++ {
			if _, ok := used[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		used[name] = struct{}{}
		roots[source.ID] = prefix + "/" + name
	}
	return roots
}

func RenderEmbeddedSpecFromPlan(compiled *plan.Plan) string {
	if compiled == nil {
		return "null\n"
	}
	cfg := compiled.Config
	paths := embeddedPlanPaths(cfg)
	var configFile *app.ConfigFileSpec
	if cfg.App.ConfigFile != nil {
		configFile = &app.ConfigFileSpec{Enabled: cfg.App.ConfigFile.Enabled, Layers: append([]string(nil), cfg.App.ConfigFile.Layers...), FileName: cfg.App.ConfigFile.FileName}
	}
	helpSpec := &app.HelpSpec{}
	runtimeSpec := app.RuntimeSpec{
		Name:       cfg.Name,
		AppName:    cfg.App.Name,
		EnvPrefix:  cfg.App.EnvPrefix,
		ConfigFile: configFile,
		Target:     app.TargetSpec{Kind: targetDataFromPlanArtifacts(cfg.Artifacts).Kind, Output: targetOutputFromPlanArtifacts(cfg.Artifacts)},
	}
	for _, provider := range cfg.Providers {
		runtimeSpec.Packages = append(runtimeSpec.Packages, app.PackageSpec{ID: provider.ID})
	}
	for _, module := range cfg.Runtime.Modules {
		runtimeSpec.Modules = append(runtimeSpec.Modules, app.ModuleInstanceSpec{Package: module.Provider, Name: module.Name, As: module.As, Config: module.Config})
	}
	for _, command := range cfg.Commands {
		applyPlanRuntimeCommand(&runtimeSpec.Commands, &runtimeSpec.CommandProviders, command)
	}
	for _, source := range cfg.Sources {
		switch source.Kind {
		case specv2.SourceKindJSVerbs:
			runtimeSpec.JSVerbs = append(runtimeSpec.JSVerbs, runtimeJSVerbSourceFromPlan(source, paths.JSVerbRoots[source.ID]))
		case specv2.SourceKindHelp:
			helpSpec.Sources = append(helpSpec.Sources, runtimeHelpSourceFromPlan(source, paths.HelpRoots[source.ID]))
		case specv2.SourceKindAssets:
			runtimeSpec.Assets = append(runtimeSpec.Assets, runtimeAssetSourceFromPlan(source, paths.AssetRoots[source.ID]))
		case specv2.SourceKindScript:
		}
	}
	payload := struct {
		Name             string                            `json:"name"`
		AppName          string                            `json:"appName,omitempty"`
		EnvPrefix        string                            `json:"envPrefix,omitempty"`
		ConfigFile       *app.ConfigFileSpec               `json:"configFile,omitempty"`
		Target           app.TargetSpec                    `json:"target"`
		Packages         []app.PackageSpec                 `json:"packages"`
		Modules          []app.ModuleInstanceSpec          `json:"modules"`
		Commands         app.CommandsSpec                  `json:"commands"`
		CommandProviders []app.CommandProviderInstanceSpec `json:"commandProviders,omitempty"`
		JSVerbs          []app.JSVerbSourceSpec            `json:"jsverbs,omitempty"`
		Help             *app.HelpSpec                     `json:"help,omitempty"`
		Assets           []app.AssetSourceSpec             `json:"assets,omitempty"`
	}{
		Name:             runtimeSpec.Name,
		AppName:          runtimeSpec.AppName,
		EnvPrefix:        runtimeSpec.EnvPrefix,
		ConfigFile:       runtimeSpec.ConfigFile,
		Target:           runtimeSpec.Target,
		Packages:         runtimeSpec.Packages,
		Modules:          runtimeSpec.Modules,
		Commands:         runtimeSpec.Commands,
		CommandProviders: runtimeSpec.CommandProviders,
		JSVerbs:          runtimeSpec.JSVerbs,
		Assets:           runtimeSpec.Assets,
	}
	if len(helpSpec.Sources) > 0 {
		payload.Help = helpSpec
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func targetOutputFromPlanArtifacts(artifacts []specv2.ArtifactSpec) string {
	for _, artifact := range artifacts {
		if artifact.Type == "binary" || artifact.Type == "runtime-package" || artifact.Type == "adapter" || artifact.Type == "cobra" || artifact.Type == "source" || artifact.Type == "template" {
			return artifact.Output
		}
	}
	return "dist/xgoja-app"
}

func applyPlanRuntimeCommand(commands *app.CommandsSpec, providers *[]app.CommandProviderInstanceSpec, command specv2.CommandSurfaceSpec) {
	spec := app.CommandSpec{Enabled: true, Name: command.Name, Mount: command.Mount}
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
		*providers = append(*providers, app.CommandProviderInstanceSpec{ID: command.ID, Package: command.Provider, Name: command.Name, Mount: command.Mount, Modules: append([]string(nil), command.Modules...), Config: command.Config, Lazy: command.Lazy})
	}
}

func runtimeJSVerbSourceFromPlan(source specv2.SourceSpec, embeddedRoot string) app.JSVerbSourceSpec {
	out := app.JSVerbSourceSpec{ID: source.ID, Embed: embeddedRoot != "", Include: append([]string(nil), source.Include...), Exclude: append([]string(nil), source.Exclude...), Extensions: append([]string(nil), source.Extensions...)}
	if source.From.Provider != nil {
		out.Package = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else if embeddedRoot != "" {
		out.Path = embeddedRoot
	} else {
		out.Path = source.From.Dir
	}
	if strings.EqualFold(source.Language, "typescript") || source.Compile != nil {
		out.TypeScript = &app.TypeScriptSpec{Enabled: strings.EqualFold(source.Language, "typescript"), Bundle: source.Compile != nil && source.Compile.Bundle}
		if source.Compile != nil {
			out.TypeScript.Define = cloneStringMapFromPlan(source.Compile.Define)
			if source.Compile.Check != nil {
				out.TypeScript.CheckCommand = append([]string(nil), source.Compile.Check.Command...)
			}
		}
	}
	return out
}

func runtimeHelpSourceFromPlan(source specv2.SourceSpec, embeddedRoot string) app.HelpSourceSpec {
	out := app.HelpSourceSpec{ID: source.ID, Embed: embeddedRoot != ""}
	if source.From.Provider != nil {
		out.Package = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else if embeddedRoot != "" {
		out.Path = embeddedRoot
	} else {
		out.Path = source.From.Dir
	}
	return out
}

func runtimeAssetSourceFromPlan(source specv2.SourceSpec, embeddedRoot string) app.AssetSourceSpec {
	path := source.From.Dir
	if embeddedRoot != "" {
		path = embeddedRoot
	}
	return app.AssetSourceSpec{ID: source.ID, Path: path, Embed: embeddedRoot != ""}
}
