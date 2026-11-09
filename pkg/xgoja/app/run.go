package app

import (
	"context"
	"fmt"
	"io"
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
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent/scriptresult"
	"github.com/go-go-golems/go-go-goja/pkg/tsscript"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type runCommand struct {
	*cmds.CommandDescription
	factory     *RuntimeFactory
	runtimePlan *RuntimePlan
	out         io.Writer
	sectionErr  error
}

var _ cmds.BareCommand = (*runCommand)(nil)

type runSettings struct {
	File       string `glazed:"file"`
	KeepAlive  bool   `glazed:"keep-alive"`
	ContentDir string `glazed:"content-dir"`
}

func newRunCommand(factory *RuntimeFactory, runtimePlan *RuntimePlan, out io.Writer) cmds.Command {
	moduleSections, _, sectionErr := factory.sectionsForRuntime("run")
	options := []cmds.CommandDescriptionOption{
		cmds.WithShort("Execute a JavaScript file in a generated xgoja runtime"),
		cmds.WithLong(`
Run executes a JavaScript file in a fresh xgoja runtime.

The generated runtime controls which provider modules are available through
require(). The script's directory is added to the module resolution roots so
sibling JavaScript files can be required by relative path.

If module.exports (awaited when it is a promise) holds result content, for
example images returned by a module, it is printed with "[content N: ...]"
placeholders; --content-dir DIR renders the content and writes it to
DIR/content-N.<ext>, printing the paths. A TypeScript script's bundle value is
treated the same way.
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
			fields.New(scriptresult.ContentDirFlag, fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp(scriptresult.ContentDirHelp)),
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
		out:                out,
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
	out := c.out
	if out == nil {
		out = os.Stdout
	}
	return runScriptFileWithInitializers(ctx, c.factory, settings.File, vals, selectedModules, settings.KeepAlive, out, scriptresult.Options{ContentDir: settings.ContentDir})
}

func runScriptFileWithInitializers(ctx context.Context, factory *RuntimeFactory, file string, vals *values.Values, selectedModules []providerapi.ModuleDescriptor, keepAlive bool, out io.Writer, content scriptresult.Options) error {
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

	var result goja.Value
	if tsscript.IsTypeScriptPath(scriptPath) {
		result, err = runTypeScriptScript(ctx, rt, scriptPath, selectedModules)
		if err != nil {
			return err
		}
	} else {
		ret, err := rt.Owner.Call(ctx, "xgoja.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
			_ = vm
			return rt.Require.Require(scriptPath)
		})
		if err != nil {
			return fmt.Errorf("run %s as module: %w", scriptPath, err)
		}
		result, _ = ret.(goja.Value)
	}
	if err := reportScriptResult(ctx, rt, result, out, content); err != nil {
		return err
	}
	if keepAlive {
		fmt.Fprintln(os.Stderr, "xgoja run: runtime is alive; press Ctrl-C to stop")
		return waitForKeepAlive(ctx)
	}
	return nil
}

// reportScriptResult prints the script's result (module.exports) only when it
// holds result content; scripts without content print nothing, as before.
func reportScriptResult(ctx context.Context, rt *engine.Runtime, result goja.Value, out io.Writer, content scriptresult.Options) error {
	if result == nil {
		return nil
	}
	settledValue, err := scriptresult.Settle(ctx, rt.Owner, result)
	if err != nil {
		return err
	}
	report, err := scriptresult.Build(ctx, rt.Owner, settledValue, content)
	if err != nil {
		return err
	}
	return report.Fprint(out)
}

func runTypeScriptScript(ctx context.Context, rt *engine.Runtime, scriptPath string, selectedModules []providerapi.ModuleDescriptor) (goja.Value, error) {
	artifact, err := tsscript.BundleEntry(scriptPath, tsscript.Options{
		Target:   api.ES2015,
		Format:   api.FormatIIFE,
		Platform: api.PlatformNeutral,
		External: moduleAliases(selectedModules),
	})
	if err != nil {
		return nil, fmt.Errorf("compile TypeScript %s: %w", scriptPath, err)
	}
	ret, err := rt.Owner.Call(ctx, "xgoja.run.typescript", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunScript(scriptPath, string(artifact.Code))
	})
	if err != nil {
		return nil, fmt.Errorf("run compiled TypeScript %s: %w", scriptPath, err)
	}
	value, _ := ret.(goja.Value)
	return value, nil
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
