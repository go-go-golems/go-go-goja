package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "goja-perf",
		Short: "Goja performance task runner",
		Long:  "Run and report Goja performance phase tasks using Glazed-defined command flags.",
	}

	phase1TasksCmd, err := newPhase1TasksCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	phase1TasksCobra, err := cli.BuildCobraCommand(phase1TasksCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	phase1RunCmd, err := newPhase1RunCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	phase1RunCobra, err := cli.BuildCobraCommand(phase1RunCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root.AddCommand(phase1TasksCobra)
	root.AddCommand(phase1RunCobra)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
