package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/go-go-goja/engine"
	sharedoc "github.com/go-go-golems/go-go-goja/pkg/doc"
	docaccessruntime "github.com/go-go-golems/go-go-goja/pkg/docaccess/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	DBPath             string
	PluginDirs         []string
	AllowPluginModules []string
}

func newRootCommand(out io.Writer) (*cobra.Command, error) {
	opts := &rootOptions{}

	root := &cobra.Command{
		Use:   "goja-repl",
		Short: "JavaScript REPL CLI, TUI, and JSON server",
		Long:  "Create, evaluate, inspect, export, restore, serve, and interact with goja-backed REPL sessions.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	root.SetOut(out)
	root.PersistentFlags().StringVar(&opts.DBPath, "db-path", "goja-repl.sqlite", "SQLite path for persistent REPL state")
	root.PersistentFlags().StringSliceVar(&opts.PluginDirs, "plugin-dir", nil, fmt.Sprintf("plugin directory (defaults to %s/... when omitted)", host.DefaultDiscoveryRoot()))
	root.PersistentFlags().StringSliceVar(&opts.AllowPluginModules, "allow-plugin-module", nil, "allow only the listed plugin module names")
	if err := logging.AddLoggingSectionToRootCommand(root, "goja-repl"); err != nil {
		return nil, err
	}
	setDefaultFlagValue(root, "log-level", "error")
	setDefaultFlagValue(root, "log-format", "text")

	commands := []cmds.Command{
		newSessionsCommand(out, opts),
		newCreateCommand(out, opts),
		newEvalCommand(out, opts),
		newRunCommand(out, opts),
		newSnapshotCommand(out, opts),
		newHistoryCommand(out, opts),
		newBindingsCommand(out, opts),
		newDocsCommand(out, opts),
		newExportCommand(out, opts),
		newRestoreCommand(out, opts),
		newServeCommand(out, opts),
		newEssayCommand(out, opts),
		newTUICommand(out, opts),
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

	helpSystem, err := newSharedHelpSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load help docs: %v\n", err)
		helpSystem = help.NewHelpSystem()
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	return root, nil
}

// commandSupport provides shared app construction for CLI commands.
type commandSupport struct {
	out  io.Writer
	opts *rootOptions
}

// appSupportOptions configures how newAppWithOptions builds the app.
type appSupportOptions struct {
	profile    replapi.Profile
	withStore  bool
	helpSystem *help.HelpSystem
}

func (s commandSupport) newApp() (*replapi.App, *repldb.Store, error) {
	return s.newAppWithOptions(appSupportOptions{
		profile:   replapi.ProfilePersistent,
		withStore: true,
	})
}

func (s commandSupport) newAppWithOptions(options appSupportOptions) (*replapi.App, *repldb.Store, error) {
	var store *repldb.Store
	var err error
	if options.withStore {
		store, err = repldb.Open(context.Background(), s.opts.DBPath)
		if err != nil {
			return nil, nil, err
		}
	}
	pluginSetup := host.NewRuntimeSetup(s.opts.PluginDirs, s.opts.AllowPluginModules)
	builder := engine.NewBuilder().WithModules(engine.DefaultRegistryModules())
	if options.helpSystem != nil {
		builder = builder.WithRuntimeModuleRegistrars(docaccessruntime.NewRegistrar(docaccessruntime.Config{
			HelpSources: []docaccessruntime.HelpSource{{
				ID:      "default-help",
				Title:   "Default Help",
				Summary: "Embedded REPL help pages",
				System:  options.helpSystem,
			}},
		}))
	}
	builder = pluginSetup.WithBuilder(builder)
	factory, err := builder.Build()
	if err != nil {
		if store != nil {
			_ = store.Close()
		}
		return nil, nil, errors.Wrap(err, "build engine factory")
	}
	appOpts := []replapi.Option{replapi.WithProfile(options.profile)}
	if store != nil {
		appOpts = append(appOpts, replapi.WithStore(store))
	}
	app, err := replapi.New(factory, log.Logger, appOpts...)
	if err != nil {
		if store != nil {
			_ = store.Close()
		}
		return nil, nil, err
	}
	return app, store, nil
}

func newSharedHelpSystem() (*help.HelpSystem, error) {
	helpSystem := help.NewHelpSystem()
	if err := sharedoc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	return helpSystem, nil
}

func writeJSON(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func setDefaultFlagValue(root *cobra.Command, name string, value string) {
	flag := root.PersistentFlags().Lookup(name)
	if flag == nil {
		return
	}
	flag.DefValue = value
	_ = flag.Value.Set(value)
}

// --- Shared settings types used by multiple commands ---

type sessionSettings struct {
	SessionID string `glazed:"session-id"`
}

type evalSettings struct {
	SessionID string `glazed:"session-id"`
	Source    string `glazed:"source"`
}

type serveSettings struct {
	Addr string `glazed:"addr"`
}

// runWithApp is a convenience wrapper for the common pattern:
// build app, defer close, call fn.
func (s commandSupport) runWithApp(fn func(ctx context.Context, app *replapi.App) error) func(ctx context.Context, vals *values.Values) error {
	return func(ctx context.Context, vals *values.Values) error {
		_ = vals
		app, store, err := s.newApp()
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()
		return fn(ctx, app)
	}
}
