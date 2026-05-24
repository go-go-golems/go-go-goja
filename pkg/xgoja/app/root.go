package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Options struct {
	Providers       *providerapi.Registry
	SpecJSON        string
	Out             io.Writer
	EmbeddedJSVerbs fs.FS
}

func NewRootCommand(opts Options) (*cobra.Command, error) {
	if opts.Providers == nil {
		return nil, fmt.Errorf("providers registry is required")
	}
	spec := &Spec{}
	if err := json.Unmarshal([]byte(opts.SpecJSON), spec); err != nil {
		return nil, fmt.Errorf("decode embedded xgoja spec: %w", err)
	}
	host := NewHostWithOptions(opts.Providers, spec, HostOptions{EmbeddedJSVerbs: opts.EmbeddedJSVerbs})
	root := &cobra.Command{
		Use:   spec.Name,
		Short: "Generated xgoja binary",
	}
	if opts.Out != nil {
		root.SetOut(opts.Out)
	}
	host.AttachDefaultCommands(root)
	return root, nil
}

func newEvalCommand(factory *RuntimeFactory, spec *Spec) *cobra.Command {
	profile := firstRuntime(spec)
	cmdName := commandName(spec.Commands.Eval, "eval")
	cmd := &cobra.Command{
		Use:   cmdName + " [source]",
		Short: "Evaluate JavaScript in a generated xgoja runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := factory.NewRuntime(cmd.Context(), profile)
			if err != nil {
				return err
			}
			defer func() { _ = rt.Close(context.Background()) }()
			ret, err := rt.Owner.Call(cmd.Context(), "xgoja.eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
				value, err := vm.RunString(args[0])
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
				fmt.Fprintln(cmd.OutOrStdout(), ret)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "runtime", profile, "Runtime profile to use")
	return cmd
}

type modulesCommand struct {
	*cmds.CommandDescription
	providers *providerapi.Registry
}

var _ cmds.GlazeCommand = (*modulesCommand)(nil)

func newModulesCommand(providers *providerapi.Registry, spec *Spec) cmds.Command {
	_ = spec
	return &modulesCommand{
		CommandDescription: cmds.NewCommandDescription("modules",
			cmds.WithShort("List provider modules registered in this generated binary"),
			cmds.WithLong("List provider modules compiled into this generated xgoja binary."),
		),
		providers: providers,
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
			if err := gp.AddRow(ctx, types.NewRow(
				types.MRP("package", pkg.ID),
				types.MRP("module", name),
				types.MRP("require", fmt.Sprintf("%s.%s", pkg.ID, name)),
			)); err != nil {
				return err
			}
		}
	}
	return nil
}

func newVerbsCommand(providers *providerapi.Registry, factory *RuntimeFactory, spec *Spec, embeddedJSVerbs fs.FS) *cobra.Command {
	root := &cobra.Command{
		Use:   commandName(spec.Commands.JSVerbs, "verbs"),
		Short: "Run configured JavaScript verb commands",
	}
	mounted, err := buildVerbCommands(providers, factory, spec, embeddedJSVerbs)
	if err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
		return root
	}
	list := &cobra.Command{
		Use:   "sources",
		Short: "List configured JavaScript verb sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, source := range spec.JSVerbs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", source.ID)
			}
			return nil
		},
	}
	root.AddCommand(list)
	if err := glazedcli.AddCommandsToRootCommand(root, mounted, nil, glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
		MiddlewaresFunc: glazedcli.CobraCommandDefaultMiddlewares,
	})); err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	return root
}

func buildVerbCommands(providers *providerapi.Registry, factory *RuntimeFactory, spec *Spec, embeddedJSVerbs fs.FS) ([]cmds.Command, error) {
	commands := []cmds.Command{}
	for _, source := range spec.JSVerbs {
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
				rt, err := factory.NewRuntime(ctx, spec.Commands.JSVerbs.Runtime, require.WithLoader(registry.RequireLoader()))
				if err != nil {
					return nil, err
				}
				defer func() { _ = rt.Close(context.Background()) }()
				return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

func scanVerbSource(providers *providerapi.Registry, embeddedJSVerbs fs.FS, source JSVerbSourceSpec) (*jsverbs.Registry, error) {
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
		registry, err := jsverbs.ScanFS(providerSource.FS, providerSource.Root)
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
		registry, err := jsverbs.ScanFS(embeddedJSVerbs, source.Path)
		if err != nil {
			return nil, fmt.Errorf("scan embedded jsverb source %s: %w", source.ID, err)
		}
		return registry, nil
	}
	registry, err := jsverbs.ScanDir(source.Path)
	if err != nil {
		return nil, fmt.Errorf("scan jsverb source %s: %w", source.ID, err)
	}
	return registry, nil
}

func firstRuntime(spec *Spec) string {
	if spec.Commands.Eval.Enabled && spec.Commands.Eval.Runtime != "" {
		return spec.Commands.Eval.Runtime
	}
	names := make([]string, 0, len(spec.Runtimes))
	for name := range spec.Runtimes {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "main"
	}
	return names[0]
}

func commandName(command CommandSpec, fallback string) string {
	if command.Name != "" {
		return command.Name
	}
	return fallback
}
