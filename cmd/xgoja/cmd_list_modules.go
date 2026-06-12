package main

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate"
)

type listModulesCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*listModulesCommand)(nil)

type listModulesSettings struct {
	File string `glazed:"file"`
}

func newListModulesCommand() *listModulesCommand {
	sections, err := commandSections()
	if err != nil {
		panic(err)
	}
	return &listModulesCommand{CommandDescription: cmds.NewCommandDescription("list-modules",
		cmds.WithShort("List modules selected by an xgoja build spec"),
		cmds.WithLong(`
List modules shows the require() modules selected by an xgoja build spec. Phase
1 wires the command; static build spec parsing is implemented in the buildspec task.

Examples:
  xgoja list-modules -f xgoja.yaml
  xgoja list-modules -f xgoja.yaml --output table
`),
		cmds.WithFlags(
			fields.New("file", fields.TypeString,
				fields.WithDefault("xgoja.yaml"),
				fields.WithShortFlag("f"),
				fields.WithHelp("Path to the xgoja build specification")),
		),
		cmds.WithSections(sections...),
	)}
}

func (c *listModulesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := listModulesSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	compiledPlan, err := loadV2Plan(settings.File)
	if err != nil {
		return err
	}
	buildSpec := generate.BuildSpecFromPlan(compiledPlan)
	for _, mod := range buildSpec.Modules {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("file", settings.File),
			types.MRP("package", mod.Package),
			types.MRP("module", mod.Name),
			types.MRP("alias", mod.Alias()),
		)); addErr != nil {
			return addErr
		}
	}
	return nil
}
