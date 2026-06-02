package app

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

func buildGlazedCobraCommand(command cmds.Command, middlewaresFuncs ...cli.CobraMiddlewaresFunc) (*cobra.Command, error) {
	middlewaresFunc := cli.CobraCommandDefaultMiddlewares
	if len(middlewaresFuncs) > 0 && middlewaresFuncs[0] != nil {
		middlewaresFunc = middlewaresFuncs[0]
	}
	return cli.BuildCobraCommand(command,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   middlewaresFunc,
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
