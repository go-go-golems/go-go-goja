package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/tsscript"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type runCommand struct {
	*cmds.CommandDescription
	factory     *RuntimeFactory
	runtimePlan *RuntimePlan
	sectionErr  error
}

var _ cmds.BareCommand = (*runCommand)(nil)

type runSettings struct {
	File      string `glazed:"file"`
	KeepAlive bool   `glazed:"keep-alive"`
}

func newRunCommand(factory *RuntimeFactory, runtimePlan *RuntimePlan) cmds.Command {
	moduleSections, _, sectionErr := factory.sectionsForRuntime("run")
	options := []cmds.CommandDescriptionOption{
		cmds.WithShort("Execute a JavaScript file in a generated xgoja runtime"),
		cmds.WithLong(`
Run executes a JavaScript file in a fresh xgoja runtime.

The generated runtime controls which provider modules are available through
require(). The script's directory is added to the module resolution roots so
sibling JavaScript files can be required by relative path.
`),
		cmds.WithArguments(
			fields.New("file", fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("Path to the JavaScript file to execute")),
		),
		cmds.WithFlags(
			fields.New("keep-alive", fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Keep the runtime alive after the script finishes, useful for scripts that register HTTP routes or static mounts")),
		),
	}
	if sectionErr == nil && len(moduleSections) > 0 {
		options = append(options, cmds.WithSections(moduleSections...))
	}
	command, _ := runtimePlan.commandByType("builtin.run")
	return &runCommand{
		CommandDescription: cmds.NewCommandDescription(commandName(command, "run"), options...),
		factory:            factory,
		runtimePlan:        runtimePlan,
		sectionErr:         sectionErr,
	}
}

func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
	if c.sectionErr != nil {
		return c.sectionErr
	}
	settings := runSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	selectedModules, err := c.factory.selectedModuleDescriptors()
	if err != nil {
		return err
	}
	return runScriptFileWithInitializers(ctx, c.factory, settings.File, vals, selectedModules, settings.KeepAlive)
}

func runScriptFileWithInitializers(ctx context.Context, factory *RuntimeFactory, file string, vals *values.Values, selectedModules []providerapi.ModuleDescriptor, keepAlive bool) error {
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
	rt, err := factory.NewRuntimeFromSections(ctx, vals, requireOpt)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}
	defer func() { _ = rt.Close(ctx) }()
	if vals != nil && len(selectedModules) > 0 {
		if err := initRuntimeFromSections(ctx, vals, rt, selectedModules); err != nil {
			return err
		}
	}

	if tsscript.IsTypeScriptPath(scriptPath) {
		if err := runTypeScriptScript(ctx, rt, scriptPath, selectedModules); err != nil {
			return err
		}
	} else {
		_, err = rt.Owner.Call(ctx, "xgoja.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
			_ = vm
			return rt.Require.Require(scriptPath)
		})
		if err != nil {
			return fmt.Errorf("run %s as module: %w", scriptPath, err)
		}
	}
	if keepAlive {
		fmt.Fprintln(os.Stderr, "xgoja run: runtime is alive; press Ctrl-C to stop")
		return waitForKeepAlive(ctx)
	}
	return nil
}

func runTypeScriptScript(ctx context.Context, rt *engine.Runtime, scriptPath string, selectedModules []providerapi.ModuleDescriptor) error {
	artifact, err := tsscript.BundleEntry(scriptPath, tsscript.Options{
		Target:   api.ES2015,
		Format:   api.FormatIIFE,
		Platform: api.PlatformNeutral,
		External: moduleAliases(selectedModules),
	})
	if err != nil {
		return fmt.Errorf("compile TypeScript %s: %w", scriptPath, err)
	}
	_, err = rt.Owner.Call(ctx, "xgoja.run.typescript", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunScript(scriptPath, string(artifact.Code))
	})
	if err != nil {
		return fmt.Errorf("run compiled TypeScript %s: %w", scriptPath, err)
	}
	return nil
}

func moduleAliases(selectedModules []providerapi.ModuleDescriptor) []string {
	if len(selectedModules) == 0 {
		return nil
	}
	out := make([]string, 0, len(selectedModules))
	seen := map[string]struct{}{}
	for _, module := range selectedModules {
		alias := module.As
		if alias == "" {
			alias = module.ModuleID
		}
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	return out
}

func waitForKeepAlive(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-signalCtx.Done()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
