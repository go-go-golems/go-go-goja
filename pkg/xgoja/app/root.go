package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Options struct {
	Providers       *providerapi.ProviderRegistry
	SpecJSON        string
	Out             io.Writer
	EmbeddedJSVerbs fs.FS
	EmbeddedHelp    fs.FS
	EmbeddedAssets  fs.FS
	MiddlewaresFunc glazedcli.CobraMiddlewaresFunc
}

func NewRootCommand(opts Options) (*cobra.Command, error) {
	if opts.Providers == nil {
		return nil, fmt.Errorf("providers registry is required")
	}
	runtimeSpec := &RuntimeSpec{}
	if err := json.Unmarshal([]byte(opts.SpecJSON), runtimeSpec); err != nil {
		return nil, fmt.Errorf("decode embedded xgoja runtime spec: %w", err)
	}
	host := NewHostWithOptions(opts.Providers, runtimeSpec, HostOptions{EmbeddedJSVerbs: opts.EmbeddedJSVerbs, EmbeddedHelp: opts.EmbeddedHelp, EmbeddedAssets: opts.EmbeddedAssets, Out: opts.Out, MiddlewaresFunc: opts.MiddlewaresFunc})
	root := &cobra.Command{
		Use:   runtimeSpec.Name,
		Short: "Generated xgoja binary",
	}
	if opts.Out != nil {
		root.SetOut(opts.Out)
	}
	host.AttachDefaultCommands(root)
	return root, nil
}

type evalCommand struct {
	*cmds.CommandDescription
	factory    *RuntimeFactory
	out        io.Writer
	sectionErr error
}

var _ cmds.BareCommand = (*evalCommand)(nil)

type evalSettings struct {
	Source string `glazed:"source"`
}

func newEvalCommand(factory *RuntimeFactory, runtimeSpec *RuntimeSpec, out io.Writer) cmds.Command {
	moduleSections, _, sectionErr := factory.sectionsForRuntime("eval")
	options := []cmds.CommandDescriptionOption{
		cmds.WithShort("Evaluate JavaScript in a generated xgoja runtime"),
		cmds.WithLong(`
Evaluate executes a JavaScript source string in a fresh xgoja runtime and prints
non-null, non-undefined results.

The generated runtime controls which provider modules are available through
require(). Provider modules may add Glazed sections; those sections are parsed
before evaluation and runtime initializers run before the JavaScript source.
`),
		cmds.WithArguments(
			fields.New("source", fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("JavaScript source to evaluate")),
		),
	}
	if sectionErr == nil && len(moduleSections) > 0 {
		options = append(options, cmds.WithSections(moduleSections...))
	}
	return &evalCommand{
		CommandDescription: cmds.NewCommandDescription(commandName(runtimeSpec.Commands.Eval, "eval"), options...),
		factory:            factory,
		out:                out,
		sectionErr:         sectionErr,
	}
}

func (c *evalCommand) Run(ctx context.Context, vals *values.Values) error {
	if c.sectionErr != nil {
		return c.sectionErr
	}
	settings := evalSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	selectedModules, err := c.factory.selectedModuleDescriptors()
	if err != nil {
		return err
	}
	return evalSourceWithInitializers(ctx, c.factory, settings.Source, vals, selectedModules, c.out)
}

func evalSourceWithInitializers(ctx context.Context, factory *RuntimeFactory, source string, vals *values.Values, selectedModules []providerapi.ModuleDescriptor, out io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if factory == nil {
		return fmt.Errorf("runtime factory is required")
	}
	rt, err := factory.NewRuntimeFromSections(ctx, vals)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if vals != nil && len(selectedModules) > 0 {
		if err := initRuntimeFromSections(ctx, vals, rt, selectedModules); err != nil {
			return err
		}
	}
	ret, err := rt.Owner.Call(ctx, "xgoja.eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(source)
		if err != nil {
			return nil, err
		}
		if value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
			return value.Export(), nil
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	if ret != nil {
		if out == nil {
			out = io.Discard
		}
		fmt.Fprintln(out, ret)
	}
	return nil
}

type modulesCommand struct {
	*cmds.CommandDescription
	providers *providerapi.ProviderRegistry
}

type selectedModulesCommand struct {
	*cmds.CommandDescription
	runtimeSpec *RuntimeSpec
}

var _ cmds.GlazeCommand = (*modulesCommand)(nil)
var _ cmds.GlazeCommand = (*selectedModulesCommand)(nil)

func newModulesCommand(providers *providerapi.ProviderRegistry, runtimeSpec *RuntimeSpec) cmds.Command {
	_ = runtimeSpec
	return &modulesCommand{
		CommandDescription: cmds.NewCommandDescription("modules",
			cmds.WithShort("List provider modules compiled into this generated binary"),
			cmds.WithLong("List provider modules compiled into this generated xgoja binary. This is a provider catalog, not the selected require() aliases for this runtime. Use selected-modules for runtime aliases."),
		),
		providers: providers,
	}
}

func newSelectedModulesCommand(runtimeSpec *RuntimeSpec) cmds.Command {
	return &selectedModulesCommand{
		CommandDescription: cmds.NewCommandDescription("selected-modules",
			cmds.WithShort("List require() modules selected for this generated runtime"),
			cmds.WithLong("List the provider modules selected into this generated xgoja runtime, including the actual CommonJS require() alias and static module config."),
		),
		runtimeSpec: runtimeSpec,
	}
}

func (c *modulesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	_ = vals
	if c.providers == nil {
		return fmt.Errorf("providers registry is required")
	}
	for _, pkg := range c.providers.Packages() {
		names := make([]string, 0, len(pkg.Modules))
		for name := range pkg.Modules {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			providerRef := fmt.Sprintf("%s.%s", pkg.ID, name)
			if err := gp.AddRow(ctx, types.NewRow(
				types.MRP("package", pkg.ID),
				types.MRP("module", name),
				types.MRP("provider_ref", providerRef),
			)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *selectedModulesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	_ = vals
	if c.runtimeSpec == nil {
		return fmt.Errorf("runtime spec is required")
	}
	for _, mod := range c.runtimeSpec.Modules {
		config := "{}"
		if len(mod.Config) > 0 {
			data, err := json.Marshal(mod.Config)
			if err != nil {
				return fmt.Errorf("marshal config for %s.%s: %w", mod.Package, mod.Name, err)
			}
			config = string(data)
		}
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("package", mod.Package),
			types.MRP("module", mod.Name),
			types.MRP("alias", mod.Alias()),
			types.MRP("provider_ref", fmt.Sprintf("%s.%s", mod.Package, mod.Name)),
			types.MRP("config", config),
		)); err != nil {
			return err
		}
	}
	return nil
}

func newVerbsCommand(providers *providerapi.ProviderRegistry, factory *RuntimeFactory, runtimeSpec *RuntimeSpec, embeddedJSVerbs fs.FS, middlewaresFunc glazedcli.CobraMiddlewaresFunc) *cobra.Command {
	root := &cobra.Command{
		Use:   commandName(runtimeSpec.Commands.JSVerbs, "verbs"),
		Short: "Run configured JavaScript verb commands",
	}
	mounted, err := buildVerbCommands(providers, factory, runtimeSpec, embeddedJSVerbs)
	if err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
		return root
	}
	list := &cobra.Command{
		Use:   "sources",
		Short: "List configured JavaScript verb sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, source := range runtimeSpec.JSVerbs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", source.ID)
			}
			return nil
		},
	}
	root.AddCommand(list)
	if middlewaresFunc == nil {
		middlewaresFunc = glazedcli.CobraCommandDefaultMiddlewares
	}
	if err := glazedcli.AddCommandsToRootCommand(root, mounted, nil, glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
		MiddlewaresFunc: middlewaresFunc,
	})); err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	return root
}

func buildVerbCommands(providers *providerapi.ProviderRegistry, factory *RuntimeFactory, runtimeSpec *RuntimeSpec, embeddedJSVerbs fs.FS) ([]cmds.Command, error) {
	moduleSections, selectedModules, err := factory.sectionsForRuntime("jsverbs")
	if err != nil {
		return nil, err
	}
	commands := []cmds.Command{}
	for _, source := range runtimeSpec.JSVerbs {
		registry, err := scanVerbSource(providers, embeddedJSVerbs, source)
		if err != nil {
			return nil, err
		}
		if registry == nil {
			continue
		}
		for _, verb := range registry.Verbs() {
			verb := verb
			registry := registry
			cmd, err := registry.CommandForVerbWithInvoker(verb, func(ctx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
				rt, err := factory.NewRuntimeFromSections(ctx, parsedValues, require.WithLoader(registry.RequireLoader()))
				if err != nil {
					return nil, err
				}
				defer func() { _ = rt.Close(context.Background()) }()
				if len(selectedModules) > 0 {
					if err := initRuntimeFromSections(ctx, parsedValues, rt, selectedModules); err != nil {
						return nil, err
					}
				}
				return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			if len(moduleSections) > 0 {
				if err := addSectionsToCommandDescription(cmd.Description(), moduleSections, "jsverbs runtime"); err != nil {
					return nil, err
				}
			}
			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

func scanVerbSource(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, source JSVerbSourceSpec) (*jsverbs.Registry, error) {
	scanOptions := jsVerbScanOptions(source)
	if source.Package != "" || source.Source != "" {
		if providers == nil {
			return nil, fmt.Errorf("scan jsverb source %s: providers registry is required", source.ID)
		}
		providerSource, ok := providers.ResolveVerbSource(source.Package, source.Source)
		if !ok {
			return nil, fmt.Errorf("scan jsverb source %s: unknown provider verb source %s.%s", source.ID, source.Package, source.Source)
		}
		if providerSource.FS == nil {
			return nil, fmt.Errorf("scan jsverb source %s: provider verb source %s.%s has no filesystem", source.ID, source.Package, source.Source)
		}
		registry, err := jsverbs.ScanFS(providerSource.FS, providerSource.Root, scanOptions)
		if err != nil {
			return nil, fmt.Errorf("scan provider jsverb source %s (%s.%s): %w", source.ID, source.Package, source.Source, err)
		}
		return registry, nil
	}
	if source.Path == "" {
		return nil, nil
	}
	if source.Embed {
		if embeddedJSVerbs == nil {
			return nil, fmt.Errorf("scan jsverb source %s: embedded jsverbs filesystem is not configured", source.ID)
		}
		registry, err := jsverbs.ScanFS(embeddedJSVerbs, source.Path, scanOptions)
		if err != nil {
			return nil, fmt.Errorf("scan embedded jsverb source %s: %w", source.ID, err)
		}
		return registry, nil
	}
	registry, err := jsverbs.ScanDir(source.Path, scanOptions)
	if err != nil {
		return nil, fmt.Errorf("scan jsverb source %s: %w", source.ID, err)
	}
	return registry, nil
}

func jsVerbScanOptions(source JSVerbSourceSpec) jsverbs.ScanOptions {
	options := jsverbs.DefaultScanOptions()
	if len(source.Extensions) > 0 {
		options.Extensions = append([]string(nil), source.Extensions...)
	}
	options.Include = append([]string(nil), source.Include...)
	options.Exclude = append([]string(nil), source.Exclude...)
	return options
}

func commandName(command CommandSpec, fallback string) string {
	if command.Name != "" {
		return command.Name
	}
	return fallback
}

func commandMount(command CommandSpec) string {
	switch strings.ToLower(strings.TrimSpace(command.Mount)) {
	case "root", "/", ".":
		return "root"
	default:
		return ""
	}
}
