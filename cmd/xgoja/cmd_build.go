package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildexec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate"
)

type buildCommand struct {
	*cmds.CommandDescription
	out io.Writer
}

var _ cmds.BareCommand = (*buildCommand)(nil)

type buildSettings struct {
	File         string `glazed:"file"`
	Output       string `glazed:"output"`
	WorkDir      string `glazed:"work-dir"`
	KeepWork     bool   `glazed:"keep-work"`
	DryRun       bool   `glazed:"dry-run"`
	XGojaVersion string `glazed:"xgoja-version"`
	XGojaReplace string `glazed:"xgoja-replace"`
}

func newBuildCommand(out io.Writer) *buildCommand {
	return &buildCommand{
		CommandDescription: cmds.NewCommandDescription("build",
			cmds.WithShort("Build a custom goja binary from an xgoja spec"),
			cmds.WithLong(`
Build reads xgoja.yaml, generates a temporary Go program that imports selected
module provider packages, and compiles a custom binary.

By default, generated go.mod requires the go-go-goja module version recorded in
this xgoja binary. When developing from a local checkout, pass --xgoja-replace
to point generated builds at that checkout.

Examples:
  xgoja build -f xgoja.yaml
  xgoja build -f examples/xgoja/provider-shipped-jsverbs/xgoja.yaml --output ./dist/provider
  xgoja build -f xgoja.yaml --xgoja-replace /path/to/go-go-goja --keep-work
  xgoja build -f xgoja.yaml --dry-run --keep-work
`),
			cmds.WithFlags(
				fields.New("file", fields.TypeString,
					fields.WithDefault("xgoja.yaml"),
					fields.WithShortFlag("f"),
					fields.WithHelp("Path to the xgoja build specification")),
				fields.New("output", fields.TypeString,
					fields.WithHelp("Override the output binary path from the spec")),
				fields.New("work-dir", fields.TypeString,
					fields.WithHelp("Directory for generated build files; defaults to a temporary directory")),
				fields.New("keep-work", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Keep the generated build directory after completion or failure")),
				fields.New("dry-run", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Validate and print the planned build without compiling")),
				fields.New("xgoja-version", fields.TypeString,
					fields.WithDefault(defaultXGojaModuleVersion()),
					fields.WithHelp("go-go-goja module version required by generated go.mod when --xgoja-replace is not set")),
				fields.New("xgoja-replace", fields.TypeString,
					fields.WithHelp("Optional local replacement path for github.com/go-go-golems/go-go-goja in generated go.mod")),
			),
		),
		out: out,
	}
}

func (c *buildCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := buildSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	spec, report, err := buildspec.LoadFile(settings.File)
	if report != nil {
		_, _ = fmt.Fprintf(c.out, "validated %d check(s) for %s\n", len(report.Checks), settings.File)
	}
	if err != nil {
		return err
	}
	output := settings.Output
	if output == "" {
		output = spec.Target.Output
	}
	workDir := settings.WorkDir
	cleanup := func() {}
	if workDir == "" {
		tmp, err := os.MkdirTemp("", "xgoja-build-*")
		if err != nil {
			return fmt.Errorf("create temporary build directory: %w", err)
		}
		workDir = tmp
		if !settings.KeepWork {
			cleanup = func() { _ = os.RemoveAll(tmp) }
		}
	}
	defer cleanup()

	if err := generate.WriteAll(workDir, spec, generate.Options{XGojaModuleVersion: settings.XGojaVersion, XGojaReplace: settings.XGojaReplace}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.out, "generated build workspace: %s\n", workDir)
	if settings.DryRun {
		_, err = fmt.Fprintf(c.out, "xgoja dry run ok: name=%s target=%s output=%s runtimes=%d packages=%d\n", spec.Name, spec.Target.Kind, output, len(spec.Runtimes), len(spec.Packages))
		return err
	}

	if _, err := buildexec.GoModTidy(ctx, workDir); err != nil {
		return err
	}
	outputPath, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if _, err := buildexec.GoBuild(ctx, workDir, outputPath, spec.Go.Tags, spec.Go.LDFlags); err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.out, "xgoja build ok: %s\n", outputPath)
	return err
}

func defaultXGojaModuleVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if isModuleVersion(info.Main.Version) {
			return info.Main.Version
		}
		for _, dep := range info.Deps {
			if dep.Path == "github.com/go-go-golems/go-go-goja" && isModuleVersion(dep.Version) {
				return dep.Version
			}
		}
	}
	return "v0.0.0"
}

func isModuleVersion(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "v") && len(value) > 1
}
