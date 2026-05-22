package main

import (
	"context"
	"fmt"
	"io"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type buildCommand struct {
	*cmds.CommandDescription
	out io.Writer
}

var _ cmds.BareCommand = (*buildCommand)(nil)

type buildSettings struct {
	File     string `glazed:"file"`
	Output   string `glazed:"output"`
	WorkDir  string `glazed:"work-dir"`
	KeepWork bool   `glazed:"keep-work"`
	DryRun   bool   `glazed:"dry-run"`
}

func newBuildCommand(out io.Writer) *buildCommand {
	return &buildCommand{
		CommandDescription: cmds.NewCommandDescription("build",
			cmds.WithShort("Build a custom goja binary from an xgoja spec"),
			cmds.WithLong(`
Build reads xgoja.yaml, generates a temporary Go program that imports selected
module provider packages, and compiles a custom binary.

Phase 1 wires the Glazed command surface. Spec validation and generation are
implemented in the next tasks.

Examples:
  xgoja build -f xgoja.yaml
  xgoja build -f examples/webrepl/xgoja.yaml --output ./dist/webrepl
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
			),
		),
		out: out,
	}
}

func (c *buildCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = ctx
	settings := buildSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	_, err := fmt.Fprintf(c.out, "xgoja build is wired; generation is not implemented yet (file=%s, output=%s, dry-run=%t)\n", settings.File, settings.Output, settings.DryRun)
	return err
}
