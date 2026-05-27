package jsverbscli

//glazedclilint:file-ignore legacy list command exposes output flags pending row processor migration

import (
	"context"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
)

type listCommand struct {
	*cmds.CommandDescription
	discovered []DiscoveredVerb
}

func newListCommand(discovered []DiscoveredVerb) (*cobra.Command, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := glazedcli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	cmd := &listCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list",
			cmds.WithShort("List discovered JavaScript verbs"),
			cmds.WithLong(`List discovered JavaScript verbs as structured rows.

Each row includes the mounted verb path, repository provenance, source file,
function name, output mode, repository source, and repository root/embedded
location. Use Glazed output flags such as --output json, --output yaml,
--fields, and --sort-by to process the list in scripts.`),
			cmds.WithSections(glazedSection, commandSettingsSection),
		),
		discovered: discovered,
	}
	return glazedcli.BuildCobraCommandFromCommand(cmd, glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
		ShortHelpSections: []string{schema.DefaultSlug, settings.GlazedSlug},
		MiddlewaresFunc:   glazedcli.CobraCommandDefaultMiddlewares,
	}))
}

func (c *listCommand) RunIntoGlazeProcessor(ctx context.Context, _ *values.Values, gp middlewares.Processor) error {
	for _, item := range c.discovered {
		verb := item.Verb
		if verb == nil || verb.File == nil {
			continue
		}
		row := types.NewRow(
			types.MRP("path", verb.FullPath()),
			types.MRP("repository", item.Repository.Repository.Name),
			types.MRP("file", verb.File.RelPath),
			types.MRP("function", verb.FunctionName),
			types.MRP("output_mode", verb.OutputMode),
			types.MRP("source", item.Repository.Repository.Source),
			types.MRP("location", describeRepository(item.Repository)),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}
