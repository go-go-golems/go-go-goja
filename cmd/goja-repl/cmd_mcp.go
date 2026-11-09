package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replmcp"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type mcpCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*mcpCommand)(nil)

type mcpSettings struct {
	HTTP          string `glazed:"http"`
	AllowRemote   bool   `glazed:"allow-remote"`
	Persistent    bool   `glazed:"persistent"`
	EvalTimeoutMS int    `glazed:"eval-timeout-ms"`
}

func newMCPCommand(out io.Writer, opts *rootOptions) *mcpCommand {
	return &mcpCommand{
		CommandDescription: cmds.NewCommandDescription("mcp",
			cmds.WithShort("Serve REPL sessions as an MCP server (stdio by default)"),
			cmds.WithLong(`Serve an MCP server with the tools eval, create_session, list_sessions, history and bindings.

The server speaks MCP over stdin/stdout unless --http is given. Because an eval tool over a
runtime with fs or exec is a shell, the server loads only the data-only (safe) modules unless
modules are enabled explicitly with --enable-module; --disable-module removes modules from
that set. Plugins are loaded only from explicit --plugin-dir directories.`),
			cmds.WithFlags(
				fields.New("http", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Serve MCP streamable HTTP on this address (for example 127.0.0.1:3091) instead of stdio")),
				fields.New("allow-remote", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Acknowledge that a non-loopback --http listener has no built-in authentication")),
				fields.New("persistent", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Use persistent sessions stored in --db-path instead of in-memory sessions")),
				fields.New("eval-timeout-ms", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Per-cell evaluation timeout in milliseconds (0 keeps the profile default)")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

// mcpModuleMiddleware defaults to safe modules: explicit --enable-module
// selects exactly those modules, --disable-module subtracts from the result.
func mcpModuleMiddleware(opts *rootOptions) engine.ModuleMiddleware {
	base := engine.MiddlewareSafe()
	if opts != nil && len(opts.EnableModules) > 0 && !opts.SafeMode {
		base = engine.MiddlewareOnly(opts.EnableModules...)
	}
	if opts != nil && len(opts.DisableModules) > 0 {
		return engine.Pipeline(engine.MiddlewareExclude(opts.DisableModules...), base)
	}
	return base
}

func (c *mcpCommand) Run(ctx context.Context, vals *values.Values) (retErr error) {
	settings := mcpSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if settings.HTTP != "" {
		if err := validateServeListenAddress(settings.HTTP, settings.AllowRemote); err != nil {
			return err
		}
	}
	server, app, store, err := c.newMCPServer(ctx, settings)
	if err != nil {
		return err
	}
	defer func() {
		retErr = stderrors.Join(retErr, errors.Wrap(closeAppAndStore(app, store), "close mcp server resources"))
	}()

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if settings.HTTP == "" {
		if err := replmcp.ServeStdio(runCtx, server); err != nil && !errors.Is(err, context.Canceled) {
			return errors.Wrap(err, "serve mcp over stdio")
		}
		return nil
	}
	return c.serveHTTP(runCtx, server, settings.HTTP)
}

func (c *mcpCommand) newMCPServer(ctx context.Context, settings mcpSettings) (*mcp.Server, *replapi.App, *repldb.Store, error) {
	profile := replapi.ProfileInteractive
	sessionOptions := replsession.InteractiveSessionOptions()
	if settings.Persistent {
		profile = replapi.ProfilePersistent
		sessionOptions = replsession.PersistentSessionOptions()
	}
	appOptions := []replapi.Option{
		replapi.WithResultProcessor(replsession.DefaultResultProcessor{FullText: true}),
	}
	if settings.EvalTimeoutMS > 0 {
		policy := sessionOptions.Policy
		policy.Eval.TimeoutMS = int64(settings.EvalTimeoutMS)
		appOptions = append(appOptions, replapi.WithDefaultSessionPolicy(policy))
	}
	app, store, err := c.newAppWithOptions(ctx, appSupportOptions{
		profile:             profile,
		withStore:           settings.Persistent,
		moduleMiddleware:    mcpModuleMiddleware(c.opts),
		explicitPluginsOnly: true,
		appOptions:          appOptions,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	server, err := replmcp.NewServer(app, replmcp.WithName("goja-repl"))
	if err != nil {
		return nil, nil, nil, stderrors.Join(err, closeAppAndStore(app, store))
	}
	return server, app, store, nil
}

func (c *mcpCommand) serveHTTP(ctx context.Context, server *mcp.Server, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           replmcp.NewStreamableHTTPHandler(server, nil),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		fmt.Fprintf(c.out, "goja-repl MCP server listening on http://%s\n", strings.TrimSpace(addr))
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
		return errors.Wrap(err, "run mcp http server")
	}
	return nil
}
