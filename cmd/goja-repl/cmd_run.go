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
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
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

Examples:
  goja-repl run ./testdata/yaml.js
  goja-repl --plugin-dir ./plugins run ./scripts/with-custom-modules.js
`),
			cmds.WithArguments(
				fields.New("file", fields.TypeString,
					fields.WithRequired(true),
					fields.WithHelp("Path to the JavaScript file to execute")),
			),
		),
		commandSupport: commandSupport{out: out, opts: opts},
	}
}

type runSettings struct {
	File string `glazed:"file"`
}

type runScriptOptions struct {
	File               string
	PluginDirs         []string
	AllowPluginModules []string
	UseModuleRoots     bool
}

func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := runSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	opts := runScriptOptions{
		File:           settings.File,
		UseModuleRoots: true,
	}
	if c.opts != nil {
		opts.PluginDirs = c.opts.PluginDirs
		opts.AllowPluginModules = c.opts.AllowPluginModules
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

	builder := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareSafe())
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

	factory, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build engine factory: %w", err)
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}
	defer func() { _ = rt.Close(ctx) }()

	_, err = rt.Owner.Call(ctx, "goja-repl.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_ = vm
		return rt.Require.Require(scriptPath)
	})
	if err != nil {
		return fmt.Errorf("run %s as module: %w", scriptPath, err)
	}

	return nil
}
