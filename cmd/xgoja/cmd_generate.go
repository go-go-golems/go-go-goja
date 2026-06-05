package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate"
)

type generateCommand struct {
	*cmds.CommandDescription
	out io.Writer
}

var _ cmds.BareCommand = (*generateCommand)(nil)

type generateSettings struct {
	File         string `glazed:"file"`
	Output       string `glazed:"output"`
	Package      string `glazed:"package"`
	Template     string `glazed:"template"`
	TemplateData bool   `glazed:"template-data"`
	Clean        bool   `glazed:"clean"`
	DryRun       bool   `glazed:"dry-run"`
}

func newGenerateCommand(out io.Writer) *generateCommand {
	return &generateCommand{
		CommandDescription: cmds.NewCommandDescription("generate",
			cmds.WithShort("Generate reusable xgoja runtime package source from an xgoja build spec"),
			cmds.WithLong(`
Generate reads xgoja.yaml and writes source files into an existing Go module.

Generation supports target.kind: package, source, and template. Package mode
writes one reusable runtime package file. Source-fragment mode splits the same
API across spec/providers/embed/bundle files. Template mode renders a caller
provided Go template with the same data contract. Generate does not create
go.mod, run go mod tidy, or compile a binary.

Examples:
  xgoja generate -f xgoja.yaml
  xgoja generate -f xgoja.yaml --output ./internal/xgojaruntime --package xgojaruntime
  xgoja generate -f xgoja.yaml --template ./runtime.go.tmpl --output ./internal/runtime/custom.gen.go
  xgoja generate -f xgoja.yaml --template-data
  xgoja generate -f xgoja.yaml --clean
  xgoja generate -f xgoja.yaml --dry-run
`),
			cmds.WithFlags(
				fields.New("file", fields.TypeString,
					fields.WithDefault("xgoja.yaml"),
					fields.WithShortFlag("f"),
					fields.WithHelp("Path to the xgoja build specification")),
				fields.New("output", fields.TypeString,
					fields.WithShortFlag("o"),
					fields.WithHelp("Override the generated package output directory from target.output")),
				fields.New("package", fields.TypeString,
					fields.WithHelp("Override generated Go package name from target.package or output directory")),
				fields.New("template", fields.TypeString,
					fields.WithHelp("Override custom template path for target.kind template")),
				fields.New("template-data", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Print the JSON template data contract and exit without writing files")),
				fields.New("clean", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Remove known generated xgoja outputs before generating")),
				fields.New("dry-run", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Validate and print the planned generation without writing files")),
			),
		),
		out: out,
	}
}

func (c *generateCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = ctx
	settings := generateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	buildSpec, report, err := buildspec.LoadFile(settings.File)
	if report != nil {
		_, _ = fmt.Fprintf(c.out, "validated %d check(s) for %s\n", len(report.Checks), settings.File)
	}
	if err != nil {
		return err
	}
	kind := strings.TrimSpace(buildSpec.Target.Kind)
	if kind == "" {
		kind = "xgoja"
	}
	if kind != "package" && kind != "source" && kind != "template" {
		return fmt.Errorf("xgoja generate supports target.kind package, source, or template; got %q", kind)
	}
	output := strings.TrimSpace(settings.Output)
	if output == "" {
		output = buildSpec.Target.Output
	}
	if output == "" {
		return fmt.Errorf("generate output directory is required")
	}
	packageName := strings.TrimSpace(settings.Package)
	if packageName == "" {
		packageName = strings.TrimSpace(buildSpec.Target.Package)
	}
	templatePath := strings.TrimSpace(settings.Template)
	if templatePath == "" {
		templatePath = strings.TrimSpace(buildSpec.Target.Template)
	}
	if kind == "template" && templatePath == "" {
		return fmt.Errorf("custom template path is required for target.kind template")
	}
	if templatePath != "" && !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(buildSpec.BaseDir, templatePath)
	}
	dataPackageName := packageName
	if dataPackageName == "" {
		if kind == "template" {
			dataPackageName = generate.InferPackageNameFromDir(filepath.Dir(output))
		} else {
			dataPackageName = generate.InferPackageNameFromDir(output)
		}
	}
	if settings.TemplateData {
		data, err := generate.TemplateDataJSON(buildSpec, dataPackageName)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(c.out, data)
		return err
	}
	if settings.DryRun {
		_, err = fmt.Fprintf(c.out, "xgoja generate dry run ok: name=%s target=%s output=%s package=%s template=%s clean=%v modules=%d packages=%d\n", buildSpec.Name, kind, output, dataPackageName, templatePath, settings.Clean, len(buildSpec.Modules), len(buildSpec.Packages))
		return err
	}
	if settings.Clean {
		if kind == "template" {
			err = generate.CleanGeneratedFile(output)
		} else {
			err = generate.CleanGenerated(output)
		}
		if err != nil {
			return err
		}
	}
	switch kind {
	case "package":
		err = generate.WritePackage(output, buildSpec, generate.PackageOptions{PackageName: packageName})
	case "source":
		err = generate.WriteSourceFragments(output, buildSpec, generate.PackageOptions{PackageName: packageName})
	case "template":
		err = generate.WriteCustomTemplate(output, buildSpec, generate.TemplateOptions{PackageName: packageName, TemplatePath: templatePath})
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.out, "xgoja generate ok: %s\n", output)
	return err
}
