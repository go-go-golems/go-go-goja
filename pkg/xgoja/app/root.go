package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Options struct {
	Providers *providerapi.Registry
	SpecJSON  string
	Out       io.Writer
}

func NewRootCommand(opts Options) (*cobra.Command, error) {
	if opts.Providers == nil {
		return nil, fmt.Errorf("providers registry is required")
	}
	spec := &Spec{}
	if err := json.Unmarshal([]byte(opts.SpecJSON), spec); err != nil {
		return nil, fmt.Errorf("decode embedded xgoja spec: %w", err)
	}
	host := NewHost(opts.Providers, spec)
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
	cmd := &cobra.Command{
		Use:   "eval [source]",
		Short: "Evaluate JavaScript in a generated xgoja runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := factory.NewRuntime(cmd.Context(), profile)
			if err != nil {
				return err
			}
			value, err := rt.VM.RunString(args[0])
			if err != nil {
				return err
			}
			var ret any
			if value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
				ret = value.Export()
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

func newModulesCommand(providers *providerapi.Registry, spec *Spec) *cobra.Command {
	return &cobra.Command{
		Use:   "modules",
		Short: "List provider modules registered in this generated binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = spec
			for _, pkg := range providers.Packages() {
				names := make([]string, 0, len(pkg.Modules))
				for name := range pkg.Modules {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					fmt.Fprintf(cmd.OutOrStdout(), "%s.%s\n", pkg.ID, name)
				}
			}
			return nil
		},
	}
}

func newVerbsCommand(factory *RuntimeFactory, spec *Spec) *cobra.Command {
	root := &cobra.Command{
		Use:   commandName(spec.Commands.JSVerbs, "verbs"),
		Short: "Run configured JavaScript verb commands",
	}
	mounted, err := buildVerbCommands(factory, spec)
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

func buildVerbCommands(factory *RuntimeFactory, spec *Spec) ([]cmds.Command, error) {
	commands := []cmds.Command{}
	for _, source := range spec.JSVerbs {
		if source.Path == "" {
			continue
		}
		registry, err := jsverbs.ScanDir(source.Path)
		if err != nil {
			return nil, fmt.Errorf("scan jsverb source %s: %w", source.ID, err)
		}
		for _, verb := range registry.Verbs() {
			verb := verb
			registry := registry
			cmd, err := registry.CommandForVerbWithInvoker(verb, func(ctx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
				rt, err := factory.NewRuntime(ctx, spec.Commands.JSVerbs.Runtime, require.WithLoader(registry.RequireLoader()))
				if err != nil {
					return nil, err
				}
				return registry.InvokeInGojaRuntime(ctx, rt.VM, rt.Require, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

func firstRuntime(spec *Spec) string {
	if spec.Commands.Repl.Enabled && spec.Commands.Repl.Runtime != "" {
		return spec.Commands.Repl.Runtime
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
