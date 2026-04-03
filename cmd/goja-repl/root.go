package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/go-go-goja/engine"
	sharedoc "github.com/go-go-golems/go-go-goja/pkg/doc"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replhttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
		Short: "Persistent JavaScript REPL CLI and JSON server",
		Long:  "Create, evaluate, inspect, export, restore, and serve persistent goja-backed REPL sessions.",
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
		newSnapshotCommand(out, opts),
		newHistoryCommand(out, opts),
		newBindingsCommand(out, opts),
		newDocsCommand(out, opts),
		newExportCommand(out, opts),
		newRestoreCommand(out, opts),
		newServeCommand(out, opts),
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
	if err := sharedoc.AddDocToHelpSystem(helpSystem); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load help docs: %v\n", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	return root, nil
}

type commandSupport struct {
	out  io.Writer
	opts *rootOptions
}

func (s commandSupport) newApp() (*replapi.App, *repldb.Store, error) {
	store, err := repldb.Open(context.Background(), s.opts.DBPath)
	if err != nil {
		return nil, nil, err
	}
	pluginSetup := host.NewRuntimeSetup(s.opts.PluginDirs, s.opts.AllowPluginModules)
	builder := pluginSetup.WithBuilder(engine.NewBuilder().WithModules(engine.DefaultRegistryModules()))
	factory, err := builder.Build()
	if err != nil {
		_ = store.Close()
		return nil, nil, errors.Wrap(err, "build engine factory")
	}
	return replapi.New(factory, store, log.Logger), store, nil
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

type sessionsCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*sessionsCommand)(nil)

func newSessionsCommand(out io.Writer, opts *rootOptions) *sessionsCommand {
	return &sessionsCommand{
		CommandDescription: cmds.NewCommandDescription("sessions",
			cmds.WithShort("List durable REPL sessions"),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *sessionsCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = vals
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	sessions, err := app.ListSessions(ctx)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"sessions": sessions})
}

type createCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*createCommand)(nil)

func newCreateCommand(out io.Writer, opts *rootOptions) *createCommand {
	return &createCommand{
		CommandDescription: cmds.NewCommandDescription("create",
			cmds.WithShort("Create a fresh persistent REPL session"),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *createCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = vals
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	session, err := app.CreateSession(ctx)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"session": session})
}

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

type evalCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*evalCommand)(nil)

func newEvalCommand(out io.Writer, opts *rootOptions) *evalCommand {
	return &evalCommand{
		CommandDescription: cmds.NewCommandDescription("eval",
			cmds.WithShort("Evaluate source in a persistent session"),
			cmds.WithFlags(
				fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id")),
				fields.New("source", fields.TypeString, fields.WithRequired(true), fields.WithHelp("JavaScript source to evaluate")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *evalCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := evalSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	resp, err := app.Evaluate(ctx, settings.SessionID, settings.Source)
	if err != nil {
		return err
	}
	return writeJSON(c.out, resp)
}

type snapshotCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*snapshotCommand)(nil)

func newSnapshotCommand(out io.Writer, opts *rootOptions) *snapshotCommand {
	return &snapshotCommand{
		CommandDescription: cmds.NewCommandDescription("snapshot",
			cmds.WithShort("Load the current live snapshot for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *snapshotCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	summary, err := app.Snapshot(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"session": summary})
}

type historyCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*historyCommand)(nil)

func newHistoryCommand(out io.Writer, opts *rootOptions) *historyCommand {
	return &historyCommand{
		CommandDescription: cmds.NewCommandDescription("history",
			cmds.WithShort("Show persisted evaluation history for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *historyCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	history, err := app.History(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"history": history})
}

type bindingsCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*bindingsCommand)(nil)

func newBindingsCommand(out io.Writer, opts *rootOptions) *bindingsCommand {
	return &bindingsCommand{
		CommandDescription: cmds.NewCommandDescription("bindings",
			cmds.WithShort("Show current bindings for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *bindingsCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	bindings, err := app.Bindings(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"bindings": bindings})
}

type docsCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*docsCommand)(nil)

func newDocsCommand(out io.Writer, opts *rootOptions) *docsCommand {
	return &docsCommand{
		CommandDescription: cmds.NewCommandDescription("docs",
			cmds.WithShort("Show persisted REPL-authored docs for a session"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *docsCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	docs, err := app.Docs(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"docs": docs})
}

type exportCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*exportCommand)(nil)

func newExportCommand(out io.Writer, opts *rootOptions) *exportCommand {
	return &exportCommand{
		CommandDescription: cmds.NewCommandDescription("export",
			cmds.WithShort("Export a session from SQLite"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *exportCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	exported, err := app.Export(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, exported)
}

type restoreCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*restoreCommand)(nil)

func newRestoreCommand(out io.Writer, opts *rootOptions) *restoreCommand {
	return &restoreCommand{
		CommandDescription: cmds.NewCommandDescription("restore",
			cmds.WithShort("Replay-restore a persisted session into a live runtime"),
			cmds.WithFlags(fields.New("session-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Persistent session id"))),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *restoreCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := sessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	summary, err := app.Restore(ctx, settings.SessionID)
	if err != nil {
		return err
	}
	return writeJSON(c.out, map[string]any{"session": summary})
}

type serveCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newServeCommand(out io.Writer, opts *rootOptions) *serveCommand {
	return &serveCommand{
		CommandDescription: cmds.NewCommandDescription("serve",
			cmds.WithShort("Run the persistent REPL JSON server"),
			cmds.WithFlags(
				fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:3090"), fields.WithHelp("Listen address")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	app, store, err := c.newApp()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	handler, err := replhttp.NewHandler(app)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              settings.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(runCtx)
	group.Go(func() error {
		fmt.Fprintf(c.out, "goja-repl server listening on http://%s\n", settings.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})
	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return errors.Wrap(err, "run http server")
	}
	return nil
}
