package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/engine"
)

type runCommand struct {
	*cmds.CommandDescription
	factory *RuntimeFactory
	spec    *Spec
}

var _ cmds.BareCommand = (*runCommand)(nil)

type runSettings struct {
	File    string `glazed:"file"`
	Runtime string `glazed:"runtime"`
}

func newRunCommand(factory *RuntimeFactory, spec *Spec) cmds.Command {
	profile := commandRuntime(spec.Commands.Run, firstRuntime(spec))
	return &runCommand{
		CommandDescription: cmds.NewCommandDescription(commandName(spec.Commands.Run, "run"),
			cmds.WithShort("Execute a JavaScript file in a generated xgoja runtime"),
			cmds.WithLong(`
Run executes a JavaScript file in a fresh xgoja runtime.

The runtime profile controls which provider modules are available through
require(). The script's directory is added to the module resolution roots so
sibling JavaScript files can be required by relative path.
`),
			cmds.WithArguments(
				fields.New("file", fields.TypeString,
					fields.WithRequired(true),
					fields.WithHelp("Path to the JavaScript file to execute")),
			),
			cmds.WithFlags(
				fields.New("runtime", fields.TypeString,
					fields.WithDefault(profile),
					fields.WithHelp("Runtime profile to use")),
			),
		),
		factory: factory,
		spec:    spec,
	}
}

func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := runSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return runScriptFile(ctx, c.factory, settings.Runtime, settings.File)
}

func runScriptFile(ctx context.Context, factory *RuntimeFactory, profile string, file string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if factory == nil {
		return fmt.Errorf("runtime factory is required")
	}
	if file == "" {
		return fmt.Errorf("script file is required")
	}
	scriptPath, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("resolve script path: %w", err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script file not found %q: %w", scriptPath, err)
	}
	requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
	if err != nil {
		return fmt.Errorf("resolve module roots from script %q: %w", scriptPath, err)
	}
	rt, err := factory.NewRuntime(ctx, profile, requireOpt)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}
	defer func() { _ = rt.Close(ctx) }()

	_, err = rt.Owner.Call(ctx, "xgoja.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_ = vm
		return rt.Require.Require(scriptPath)
	})
	if err != nil {
		return fmt.Errorf("run %s as module: %w", scriptPath, err)
	}
	return nil
}

func commandRuntime(command CommandSpec, fallback string) string {
	if command.Runtime != "" {
		return command.Runtime
	}
	return fallback
}
