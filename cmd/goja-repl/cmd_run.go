package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent/scriptresult"
)

type runCommand struct {
	*cmds.CommandDescription
	commandSupport
}

var _ cmds.BareCommand = (*runCommand)(nil)

func newRunCommand(out io.Writer, opts *rootOptions) *runCommand {
	return &runCommand{
		CommandDescription: cmds.NewCommandDescription("run",
			cmds.WithShort("Execute a JavaScript file in a fresh runtime"),
			cmds.WithLong(`
Run executes a JavaScript file directly in a fresh, ephemeral go-go-goja runtime.

No persistent session is created. No SQLite database is required. The runtime is
destroyed when the script completes.

If module.exports (awaited when it is a promise) holds result content, such as
images returned by a module, it is printed with "[content N: ...]" placeholders;
--content-dir DIR renders the content and writes DIR/content-N.<ext>.

Examples:
  goja-repl --enable-module yaml run ./examples/goja-repl/scripts/yaml.js
  goja-repl --plugin-dir ./plugins run ./scripts/with-custom-modules.js
`),
			cmds.WithArguments(
				fields.New("file", fields.TypeString,
					fields.WithRequired(true),
					fields.WithHelp("Path to the JavaScript file to execute")),
			),
			cmds.WithFlags(
				fields.New(scriptresult.ContentDirFlag, fields.TypeString,
					fields.WithDefault(""),
					fields.WithHelp(scriptresult.ContentDirHelp)),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

type runSettings struct {
	File       string `glazed:"file"`
	ContentDir string `glazed:"content-dir"`
}

type runScriptOptions struct {
	File               string
	PluginDirs         []string
	AllowPluginModules []string
	EnableModules      []string
	DisableModules     []string
	SafeMode           bool
	UseModuleRoots     bool
	// Out receives the result report (placeholders, written paths); nil discards it.
	Out        io.Writer
	ContentDir string
	// initializers are extra runtime initializers (tests install Go globals).
	initializers []engine.RuntimeInitializer
}

func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := runSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	opts := runScriptOptions{
		File:           settings.File,
		UseModuleRoots: true,
		Out:            c.out,
		ContentDir:     settings.ContentDir,
	}
	if c.opts != nil {
		opts.PluginDirs = c.opts.PluginDirs
		opts.AllowPluginModules = c.opts.AllowPluginModules
		opts.EnableModules = c.opts.EnableModules
		opts.DisableModules = c.opts.DisableModules
		opts.SafeMode = c.opts.SafeMode
	}
	return runScriptFile(ctx, opts)
}

func runScriptFile(ctx context.Context, opts runScriptOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.File == "" {
		return fmt.Errorf("script file is required")
	}

	scriptPath, err := filepath.Abs(opts.File)
	if err != nil {
		return fmt.Errorf("resolve script path: %w", err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script file not found %q: %w", scriptPath, err)
	}

	builder := engine.NewRuntimeFactoryBuilder()
	if opts.SafeMode {
		builder = builder.UseModuleMiddleware(engine.MiddlewareSafe())
	} else if len(opts.EnableModules) > 0 {
		builder = builder.UseModuleMiddleware(engine.MiddlewareOnly(opts.EnableModules...))
	} else if len(opts.DisableModules) > 0 {
		builder = builder.UseModuleMiddleware(engine.MiddlewareExclude(opts.DisableModules...))
	}
	if opts.UseModuleRoots {
		requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
		if err != nil {
			return fmt.Errorf("resolve module roots from script %q: %w", scriptPath, err)
		}
		if requireOpt != nil {
			builder = builder.WithRequireOptions(requireOpt)
		}
	}

	pluginSetup := host.NewRuntimeSetup(opts.PluginDirs, opts.AllowPluginModules)
	builder = pluginSetup.WithBuilder(builder)
	if len(opts.initializers) > 0 {
		builder = builder.WithRuntimeInitializers(opts.initializers...)
	}

	factory, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build engine factory: %w", err)
	}

	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}
	defer func() { _ = rt.Close(ctx) }()

	ret, err := rt.Owner.Call(ctx, "goja-repl.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_ = vm
		return rt.Require.Require(scriptPath)
	})
	if err != nil {
		return fmt.Errorf("run %s as module: %w", scriptPath, err)
	}
	exports, _ := ret.(goja.Value)
	if exports == nil {
		return nil
	}
	exports, err = scriptresult.Settle(ctx, rt.Owner, exports)
	if err != nil {
		return fmt.Errorf("run %s: %w", scriptPath, err)
	}
	report, err := scriptresult.Build(ctx, rt.Owner, exports, scriptresult.Options{ContentDir: opts.ContentDir})
	if err != nil {
		return err
	}
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	return report.Fprint(out)
}
