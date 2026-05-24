package main

import (
	"context"
	"debug/buildinfo"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

type inspectCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*inspectCommand)(nil)

type inspectSettings struct {
	Binary string `glazed:"binary"`
}

func newInspectCommand() *inspectCommand {
	sections, err := commandSections()
	if err != nil {
		panic(err)
	}
	return &inspectCommand{CommandDescription: cmds.NewCommandDescription("inspect",
		cmds.WithShort("Inspect Go build information from an installed binary"),
		cmds.WithLong(`
Inspect reads Go build metadata from a binary. This is diagnostic evidence only:
it cannot prove that an arbitrary installed binary can be extended in process.

Examples:
  xgoja inspect ./dist/webrepl
  xgoja inspect $(which goja-repl) --output json
`),
		cmds.WithArguments(
			fields.New("binary", fields.TypeString,
				fields.WithRequired(true),
				fields.WithIsArgument(true),
				fields.WithHelp("Path to the Go binary to inspect")),
		),
		cmds.WithSections(sections...),
	)}
}

func (c *inspectCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := inspectSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(settings.Binary)
	if err != nil {
		return fmt.Errorf("read build info from %s: %w", settings.Binary, err)
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("binary", settings.Binary),
		types.MRP("go_version", info.GoVersion),
		types.MRP("main_module", info.Main.Path),
		types.MRP("main_version", info.Main.Version),
		types.MRP("dependency_count", len(info.Deps)),
		types.MRP("extension_status", "not guaranteed from build metadata; prefer adapter rebuild"),
	))
}
