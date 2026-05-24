package app

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

func buildGlazedCobraCommand(command cmds.Command) (*cobra.Command, error) {
	return cli.BuildCobraCommand(command,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
}

func commandErrorStub(use string, short string, err error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return err
		},
	}
}
