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
	SpecJSON         string
	HasEmbedded      bool
	NeedsContext     bool
	HasTargetImport  bool
	TargetKind       string
	TargetImport     string
	TargetRoot       string
	HostConstruction string
	RootConstruction string
	ProviderImports  []providerImport
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

func mainTemplateDataFromSpec(spec *buildspec.Spec) mainTemplateData {
	aliases := importAliases(spec.Packages)
	hasEmbedded := hasEmbeddedJSVerbSources(spec)
	providers := make([]providerImport, 0, len(spec.Packages))
	for _, pkg := range spec.Packages {
		providers = append(providers, providerImport{
			Alias:    aliases[pkg.ID],
			Import:   pkg.Import,
			Register: pkg.Register,
		})
	}

	rootFn := strings.TrimSpace(spec.Target.Root)
	if rootFn == "" {
		rootFn = "NewRootCommand"
	}

	data := mainTemplateData{
		SpecJSON:        escapeRawString(RenderEmbeddedSpec(spec)),
		HasEmbedded:     hasEmbedded,
		NeedsContext:    spec.Target.Kind == "adapter",
		HasTargetImport: spec.Target.Kind == "adapter" || spec.Target.Kind == "cobra",
		TargetKind:      spec.Target.Kind,
		TargetImport:    spec.Target.Import,
		TargetRoot:      rootFn,
		ProviderImports: providers,
	}
	if hasEmbedded {
		data.HostConstruction = "host := app.NewHostWithOptions(registry, spec, app.HostOptions{EmbeddedJSVerbs: embeddedJSVerbs})"
		data.RootConstruction = "root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON, EmbeddedJSVerbs: embeddedJSVerbs})"
	} else {
		data.HostConstruction = "host := app.NewHost(registry, spec)"
		data.RootConstruction = "root, err := app.NewRootCommand(app.Options{Providers: registry, SpecJSON: embeddedSpecJSON})"
	}
	return data
}
