package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/go-go-goja/pkg/botcli"
	sharedoc "github.com/go-go-golems/go-go-goja/pkg/doc"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "go-go-goja",
		Short:   "Utilities and experiments for go-go-goja",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	if err := logging.AddLoggingSectionToRootCommand(root, "go-go-goja"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	helpSystem := help.NewHelpSystem()
	if err := sharedoc.AddDocToHelpSystem(helpSystem); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load help docs: %v\n", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	root.AddCommand(botcli.NewCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
