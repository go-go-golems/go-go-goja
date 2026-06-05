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
	File    string `glazed:"file"`
	Output  string `glazed:"output"`
	Package string `glazed:"package"`
	DryRun  bool   `glazed:"dry-run"`
}

func newGenerateCommand(out io.Writer) *generateCommand {
	return &generateCommand{
		CommandDescription: cmds.NewCommandDescription("generate",
			cmds.WithShort("Generate reusable xgoja runtime package source from an xgoja build spec"),
			cmds.WithLong(`
Generate reads xgoja.yaml and writes source files into an existing Go module.

The first generation mode supports target.kind: package. It writes a reusable
runtime package containing provider registration, embedded runtime spec JSON,
optional embedded resource filesystems, NewBundle, NewRuntime, and command
attachment helpers. It does not create go.mod, run go mod tidy, or compile a
binary.

Examples:
  xgoja generate -f xgoja.yaml
  xgoja generate -f xgoja.yaml --output ./internal/xgojaruntime --package xgojaruntime
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
	if kind != "package" {
		return fmt.Errorf("xgoja generate currently supports target.kind package, got %q", kind)
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
	if packageName == "" {
		packageName = filepath.Base(filepath.Clean(output))
	}
	if settings.DryRun {
		_, err = fmt.Fprintf(c.out, "xgoja generate dry run ok: name=%s target=%s output=%s package=%s modules=%d packages=%d\n", buildSpec.Name, kind, output, packageName, len(buildSpec.Modules), len(buildSpec.Packages))
		return err
	}
	if err := generate.WritePackage(output, buildSpec, generate.PackageOptions{PackageName: packageName}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.out, "xgoja generate ok: %s\n", output)
	return err
}
