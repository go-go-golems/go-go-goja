package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	jsdocdoc "github.com/go-go-golems/go-go-goja/cmd/goja-jsdoc/doc"
)

func main() {
	root := &cobra.Command{
		Use:   "goja-jsdoc",
		Short: "JavaScript doc extraction and browser (jsdocex migrated into go-go-goja)",
		Long: `Extract documentation metadata from JavaScript sources using sentinel patterns
and optionally serve a web UI + JSON API with live reload.`,
	}

	extractCmd, err := newExtractCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	extractCobra, err := cli.BuildCobraCommand(extractCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root.AddCommand(extractCobra)

	serveCmd, err := newServeCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	serveCobra, err := cli.BuildCobraCommand(serveCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root.AddCommand(serveCobra)

	exportCmd, err := newExportCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	exportCobra, err := cli.BuildCobraCommand(exportCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root.AddCommand(exportCobra)

	helpSystem := help.NewHelpSystem()
	if err := jsdocdoc.AddDocToHelpSystem(helpSystem); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load help docs: %v\n", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
