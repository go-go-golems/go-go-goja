package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	sharedoc "github.com/go-go-golems/go-go-goja/pkg/doc"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

const defaultExampleDir = "testdata/jsverbs"

// listCommand implements cmds.GlazeCommand and emits discovered verbs as structured rows.
type listCommand struct {
	*cmds.CommandDescription
	registry *jsverbs.Registry
}

func (c *listCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	for _, verb := range c.registry.Verbs() {
		row := types.NewRow(
			types.MRP("path", verb.FullPath()),
			types.MRP("source", verb.SourceRef()),
			types.MRP("output_mode", verb.OutputMode),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	dir := discoverDirectory(os.Args[1:])

	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := registerExampleSharedSections(dir, registry); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	commands, err := registry.Commands()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root := &cobra.Command{
		Use:   "jsverbs-example",
		Short: "Expose scanned JavaScript functions as Glazed commands",
		Long: fmt.Sprintf(
			"Scan %s for .js/.cjs files, infer verbs from top-level functions, and run them through goja + Glazed.",
			registry.RootDir,
		),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	root.PersistentFlags().StringP("dir", "d", dir, "Directory scanned before command registration")
	if err := logging.AddLoggingSectionToRootCommand(root, "jsverbs-example"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	setDefaultFlagValue(root, "log-level", "error")
	setDefaultFlagValue(root, "log-format", "text")

	listDesc := cmds.NewCommandDescription(
		"list",
		cmds.WithShort("List discovered JS verbs"),
		cmds.WithLong("Emit all discovered jsverbs as a structured table."),
	)
	listCmd := &listCommand{
		CommandDescription: listDesc,
		registry:           registry,
	}
	allCommands := append([]cmds.Command{listCmd}, commands...)

	if err := cli.AddCommandsToRootCommand(
		root,
		allCommands,
		nil,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug, schema.GlobalDefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	helpSystem := help.NewHelpSystem()
	if err := sharedoc.AddDocToHelpSystem(helpSystem); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load help docs: %v\n", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func discoverDirectory(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dir" || arg == "-d":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "--dir="):
			return strings.TrimPrefix(arg, "--dir=")
		}
	}
	return defaultExampleDir
}

func setDefaultFlagValue(root *cobra.Command, name string, value string) {
	flag := root.PersistentFlags().Lookup(name)
	if flag == nil {
		return
	}
	flag.DefValue = value
	_ = flag.Value.Set(value)
}

func registerExampleSharedSections(dir string, registry *jsverbs.Registry) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}
	if filepath.Base(filepath.Clean(dir)) != "registry-shared" {
		return nil
	}

	return registry.AddSharedSection(&jsverbs.SectionSpec{
		Slug:        "filters",
		Title:       "Registry Filters",
		Description: "Example host-registered shared filter flags for jsverbs-example.",
		Fields: map[string]*jsverbs.FieldSpec{
			"state": {
				Type:    "choice",
				Choices: []string{"open", "closed"},
				Default: "open",
				Help:    "Issue state",
			},
			"labels": {
				Type: "stringList",
				Help: "Labels to filter on",
			},
		},
	})
}
