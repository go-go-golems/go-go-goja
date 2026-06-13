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

func providerImportsFromPlan(compiled *plan.Plan) []providerImport {
	if compiled == nil {
		return nil
	}
	packages := make([]importAliasSeed, 0, len(compiled.Config.Providers))
	for _, provider := range compiled.Config.Providers {
		packages = append(packages, importAliasSeed{ID: provider.ID, Import: provider.Import})
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
	var configFile *app.ConfigFilePlan
	if cfg.App.ConfigFile != nil {
		configFile = &app.ConfigFilePlan{Enabled: cfg.App.ConfigFile.Enabled, Layers: append([]string(nil), cfg.App.ConfigFile.Layers...), FileName: cfg.App.ConfigFile.FileName}
	}
	runtimePlan := app.RuntimePlan{
		Schema: app.RuntimePlanSchema,
		Name:   cfg.Name,
		App: app.AppPlan{
			Name:       cfg.App.Name,
			EnvPrefix:  cfg.App.EnvPrefix,
			ConfigFile: configFile,
		},
		Target: app.TargetPlan{Kind: targetDataFromPlanArtifacts(cfg.Artifacts).Kind, Output: targetOutputFromPlanArtifacts(cfg.Artifacts)},
	}
	for _, provider := range cfg.Providers {
		runtimePlan.Providers = append(runtimePlan.Providers, app.ProviderPlan{ID: provider.ID})
	}
	for _, module := range cfg.Runtime.Modules {
		runtimePlan.Runtime.Modules = append(runtimePlan.Runtime.Modules, app.RuntimeModulePlan{Provider: module.Provider, Name: module.Name, As: module.As, Config: module.Config})
	}
	for _, source := range cfg.Sources {
		runtimePlan.Sources = append(runtimePlan.Sources, runtimeSourceFromPlan(source, paths))
	}
	for _, command := range cfg.Commands {
		runtimePlan.Commands = append(runtimePlan.Commands, runtimeCommandFromPlan(command))
	}
	for _, artifact := range cfg.Artifacts {
		runtimePlan.Artifacts = append(runtimePlan.Artifacts, app.ArtifactPlan{ID: artifact.ID, Type: artifact.Type, Output: artifact.Output, Package: artifact.Package, Import: artifact.Import, Root: artifact.Root, Sources: append([]string(nil), artifact.Sources...), Strict: artifact.Strict})
	}
	data, err := json.MarshalIndent(runtimePlan, "", "  ")
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

func runtimeCommandFromPlan(command specv2.CommandSurfaceSpec) app.CommandPlan {
	return app.CommandPlan{ID: command.ID, Type: command.Type, Name: command.Name, Mount: command.Mount, Provider: command.Provider, Sources: append([]string(nil), command.Sources...), Modules: append([]string(nil), command.Modules...), Config: command.Config, Lazy: command.Lazy}
}

func runtimeSourceFromPlan(source specv2.SourceSpec, paths planEmbeddedPaths) app.SourcePlan {
	embeddedRoot := ""
	switch source.Kind {
	case specv2.SourceKindJSVerbs:
		embeddedRoot = paths.JSVerbRoots[source.ID]
	case specv2.SourceKindHelp:
		embeddedRoot = paths.HelpRoots[source.ID]
	case specv2.SourceKindAssets:
		embeddedRoot = paths.AssetRoots[source.ID]
	case specv2.SourceKindScript:
	}
	out := app.SourcePlan{ID: source.ID, Kind: app.SourceKind(source.Kind), Embed: embeddedRoot != "", Include: append([]string(nil), source.Include...), Exclude: append([]string(nil), source.Exclude...), Extensions: append([]string(nil), source.Extensions...)}
	if source.From.Provider != nil {
		out.Provider = source.From.Provider.Provider
		out.Source = source.From.Provider.Source
	} else if embeddedRoot != "" {
		out.Path = embeddedRoot
	} else {
		out.Path = source.From.Dir
	}
	if strings.EqualFold(source.Language, "typescript") || source.Compile != nil {
		out.TypeScript = &app.TypeScriptPlan{Enabled: strings.EqualFold(source.Language, "typescript"), Bundle: source.Compile != nil && source.Compile.Bundle}
		if source.Compile != nil {
			out.TypeScript.Define = cloneStringMapFromPlan(source.Compile.Define)
			if source.Compile.Check != nil {
				out.TypeScript.CheckCommand = append([]string(nil), source.Compile.Check.Command...)
			}
		}
	}
	return out
}
