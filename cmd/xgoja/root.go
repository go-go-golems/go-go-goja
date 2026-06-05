package main

import (
	"io"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/doc"
	"github.com/spf13/cobra"
)

func newRootCommand(out io.Writer) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "xgoja",
		Short: "Build custom goja binaries from declarative module specs",
		Long: `xgoja builds custom goja-powered binaries by generating a Go program
that imports selected module provider packages and compiling it with the normal Go toolchain.

The first implementation is intentionally staged. The CLI shape is available first;
buildspec validation and code generation are added in follow-up phases.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	root.SetOut(out)

	if err := logging.AddLoggingSectionToRootCommand(root, "xgoja"); err != nil {
		return nil, err
	}
	setDefaultFlagValue(root, "log-level", "error")
	setDefaultFlagValue(root, "log-format", "text")

	commands := []cmds.Command{
		newBuildCommand(out),
		newGenerateCommand(out),
		newDoctorCommand(),
		newInspectCommand(),
		newListModulesCommand(),
	}
	for _, command := range commands {
		cobraCommand, err := cli.BuildCobraCommand(command,
			cli.WithParserConfig(cli.CobraParserConfig{
				ShortHelpSections: []string{schema.DefaultSlug},
				MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
			}),
		)
		if err != nil {
			return nil, err
		}
		root.AddCommand(cobraCommand)
	}

	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	return root, nil
}

func commandSections() ([]schema.Section, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	return []schema.Section{glazedSection, commandSettingsSection}, nil
}

func setDefaultFlagValue(root *cobra.Command, name string, value string) {
	flag := root.PersistentFlags().Lookup(name)
	if flag == nil {
		return
	}
	flag.DefValue = value
	_ = flag.Value.Set(value)
}
