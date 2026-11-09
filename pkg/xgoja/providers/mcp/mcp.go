// Package mcp is an xgoja provider whose command set serves the generated
// binary's runtime as an MCP eval server (pkg/replmcp).
//
//	providers:
//	  - id: mcp
//	    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/mcp
//	commands:
//	  - { id: mcp, type: provider.command-set, provider: mcp, name: mcp, mount: mcp }
//
// produces "<binary> mcp serve" (stdio by default, --http ADDR for streamable
// HTTP). Sessions use the binary's selected runtime modules.
package mcp

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net"
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
	"github.com/go-go-golems/go-go-goja/pkg/replmcp"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerutil"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// PackageID is the provider package id.
const PackageID = "go-go-goja-mcp"

// CommandSetName is the name of the command set provider.
const CommandSetName = "mcp"

// Register registers the mcp command set provider.
func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID,
		providerapi.CommandSetProvider{
			Name:         CommandSetName,
			DefaultMount: "mcp",
			Description:  "Serve the runtime as an MCP eval server (eval, create_session, list_sessions, history, bindings)",
			ConfigSchema: json.RawMessage(configSchema),
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				return newCommandSet(ctx)
			},
		},
	)
}

const configSchema = `{
  "type": "object",
  "properties": {
    "name": {"type": "string", "description": "MCP implementation name (default: the command mount)"},
    "version": {"type": "string"},
    "instructions": {"type": "string"},
    "evalTimeoutMs": {"type": "integer", "minimum": 0},
    "maxResultChars": {"type": "integer", "minimum": 0},
    "maxImages": {"type": "integer", "minimum": 0},
    "maxImageBytes": {"type": "integer", "minimum": 0}
  },
  "additionalProperties": false
}`

// Config is the command-set config from xgoja.yaml (commands[].config).
type Config struct {
	Name           string `json:"name,omitempty"`
	Version        string `json:"version,omitempty"`
	Instructions   string `json:"instructions,omitempty"`
	EvalTimeoutMS  int    `json:"evalTimeoutMs,omitempty"`
	MaxResultChars int    `json:"maxResultChars,omitempty"`
	MaxImages      int    `json:"maxImages,omitempty"`
	MaxImageBytes  int    `json:"maxImageBytes,omitempty"`
}

func decodeConfig(raw json.RawMessage) (Config, error) {
	cfg := Config{}
	if len(raw) == 0 || string(raw) == "null" {
		return cfg, nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "decode mcp command config")
	}
	return cfg, nil
}

type serveSettings struct {
	HTTP          string `glazed:"http"`
	AllowRemote   bool   `glazed:"allow-remote"`
	EvalTimeoutMS int    `glazed:"eval-timeout-ms"`
}

type serveCommand struct {
	*cmds.CommandDescription
	commandCtx providerapi.CommandSetContext
	config     Config
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	if ctx.RuntimeFactory == nil {
		return nil, errors.New("mcp command requires a runtime factory")
	}
	cfg, err := decodeConfig(ctx.Config)
	if err != nil {
		return nil, err
	}
	moduleSections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{CommandProviderID: ctx.Name}, nil)
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription("serve",
		cmds.WithShort("Serve the runtime as an MCP eval server (stdio by default)"),
		cmds.WithLong(`Serve an MCP server with the tools eval, create_session, list_sessions, history and bindings.
Sessions are in-memory and use this binary's runtime modules. MCP is spoken over stdin/stdout
unless --http is given (loopback only unless --allow-remote).`),
		cmds.WithFlags(
			fields.New("http", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Serve MCP streamable HTTP on this address instead of stdio")),
			fields.New("allow-remote", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Acknowledge that a non-loopback --http listener has no built-in authentication")),
			fields.New("eval-timeout-ms", fields.TypeInteger, fields.WithDefault(cfg.EvalTimeoutMS), fields.WithHelp("Per-cell evaluation timeout in milliseconds (0 keeps the default)")),
		),
	)
	if len(moduleSections) > 0 {
		seen := map[string]string{}
		if desc.Schema != nil {
			desc.Schema.ForEach(func(slug string, _ schema.Section) { seen[slug] = "command schema" })
		}
		collected := []schema.Section{}
		if err := providerutil.AppendUniqueSections(&collected, seen, moduleSections, "mcp runtime modules"); err != nil {
			return nil, err
		}
		desc.SetSections(collected...)
	}
	return &providerapi.CommandSet{Commands: []cmds.Command{&serveCommand{CommandDescription: desc, commandCtx: ctx, config: cfg}}}, nil
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) (retErr error) {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if settings.HTTP != "" {
		if err := validateListenAddress(settings.HTTP, settings.AllowRemote); err != nil {
			return err
		}
	}
	server, app, err := NewServer(ctx, c.commandCtx, c.config, settings.EvalTimeoutMS, vals)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		retErr = stderrors.Join(retErr, app.Close(closeCtx))
	}()

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if settings.HTTP == "" {
		if err := replmcp.ServeStdio(runCtx, server); err != nil && !errors.Is(err, context.Canceled) {
			return errors.Wrap(err, "serve mcp over stdio")
		}
		return nil
	}
	return serveHTTP(runCtx, server, settings.HTTP)
}

// NewServer builds the replapi app and MCP server for one command run.
// Sessions get runtimes from commandCtx.RuntimeFactory with the parsed module
// config sections in vals (which may be nil). The caller closes the app.
func NewServer(ctx context.Context, commandCtx providerapi.CommandSetContext, cfg Config, evalTimeoutMS int, vals *values.Values) (*mcpsdk.Server, *replapi.App, error) {
	factory := commandCtx.RuntimeFactory
	if factory == nil {
		return nil, nil, errors.New("mcp command requires a runtime factory")
	}
	selected := commandCtx.SelectedModules
	newRuntime := func(startup context.Context, lifetime context.Context) (*engine.Runtime, error) {
		// The xgoja factory binds the runtime lifetime to the context it gets.
		rt, err := factory.NewRuntimeFromSections(lifetime, vals)
		if err != nil {
			return nil, err
		}
		if len(selected) > 0 {
			if err := providerutil.InitRuntimeFromSections(startup, vals, runtimeHandle{rt: rt}, selected); err != nil {
				_ = rt.Close(context.WithoutCancel(startup))
				return nil, err
			}
		}
		return rt, nil
	}
	appOptions := []replapi.Option{
		replapi.WithProfile(replapi.ProfileInteractive),
		replapi.WithResultProcessor(replsession.DefaultResultProcessor{FullText: true}),
	}
	if evalTimeoutMS > 0 {
		policy := replsession.InteractiveSessionOptions().Policy
		policy.Eval.TimeoutMS = int64(evalTimeoutMS)
		appOptions = append(appOptions, replapi.WithDefaultSessionPolicy(policy))
	}
	app, err := replapi.NewWithRuntimeConstructor(ctx, newRuntime, zerolog.Nop(), appOptions...)
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = strings.TrimSpace(commandCtx.Mount)
	}
	opts := []replmcp.Option{
		replmcp.WithName(name),
		replmcp.WithVersion(cfg.Version),
		replmcp.WithLimits(replmcp.Limits{MaxResultChars: cfg.MaxResultChars, MaxImages: cfg.MaxImages, MaxImageBytes: cfg.MaxImageBytes}),
	}
	if cfg.Instructions != "" {
		opts = append(opts, replmcp.WithInstructions(cfg.Instructions))
	}
	server, err := replmcp.NewServer(app, opts...)
	if err != nil {
		_ = app.Close(context.WithoutCancel(ctx))
		return nil, nil, err
	}
	return server, app, nil
}

type runtimeHandle struct {
	rt *engine.Runtime
}

func (h runtimeHandle) EngineRuntime() *engine.Runtime { return h.rt }

func (h runtimeHandle) Close(ctx context.Context) error {
	if h.rt == nil {
		return nil
	}
	return h.rt.Close(ctx)
}

func serveHTTP(ctx context.Context, server *mcpsdk.Server, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           replmcp.NewStreamableHTTPHandler(server, nil),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		fmt.Fprintf(os.Stderr, "MCP server listening on http://%s\n", addr)
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

func validateListenAddress(addr string, allowRemote bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.Wrapf(err, "invalid listen address %q", addr)
	}
	host = strings.TrimSpace(host)
	ip := net.ParseIP(host)
	loopback := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
	if !loopback && !allowRemote {
		return errors.Errorf("refusing non-loopback listen address %q without --allow-remote; the MCP server has no built-in authentication", addr)
	}
	return nil
}
