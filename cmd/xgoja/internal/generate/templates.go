package generate

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
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

type providerImport struct {
	Alias    string
	Import   string
	Register string
}

func renderMainTemplate(data mainTemplateData) (string, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/main.go.tmpl")
	if err != nil {
		return "", fmt.Errorf("parse main template: %w", err)
	}
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, "main.go.tmpl", data); err != nil {
		return "", fmt.Errorf("execute main template: %w", err)
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return "", fmt.Errorf("format generated main.go: %w\n%s", err, b.String())
	}
	return string(formatted), nil
}

func mainTemplateDataFromSpec(buildSpec *buildspec.BuildSpec) mainTemplateData {
	aliases := importAliases(buildSpec.Packages)
	hasEmbeddedJSVerb := hasEmbeddedJSVerbSources(buildSpec)
	hasEmbeddedHelp := hasEmbeddedHelpSources(buildSpec)
	hasEmbeddedAssets := hasEmbeddedAssetSources(buildSpec)
	hasEmbedded := hasEmbeddedJSVerb || hasEmbeddedHelp || hasEmbeddedAssets
	providers := make([]providerImport, 0, len(buildSpec.Packages))
	for _, pkg := range buildSpec.Packages {
		providers = append(providers, providerImport{
			Alias:    aliases[pkg.ID],
			Import:   pkg.Import,
			Register: pkg.Register,
		})
	}

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
		ProviderImports:   providers,
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
