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
}

type packageTemplateData struct {
	PackageName       string
	SpecJSON          string
	HasEmbedded       bool
	HasEmbeddedJSVerb bool
	HasEmbeddedHelp   bool
	HasEmbeddedAssets bool
	ProviderImports   []providerImport
}

type providerImport struct {
	Alias    string
	Import   string
	Register string
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
	}
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
