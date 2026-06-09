package app

import (
	"io"
	"io/fs"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Host struct {
	Providers       *providerapi.ProviderRegistry
	RuntimeSpec     *RuntimeSpec
	Factory         *RuntimeFactory
	EmbeddedJSVerbs fs.FS
	EmbeddedHelp    fs.FS
	EmbeddedAssets  fs.FS
	Services        HostServices
	Out             io.Writer
	MiddlewaresFunc cli.CobraMiddlewaresFunc
}

type HostOptions struct {
	EmbeddedJSVerbs fs.FS
	EmbeddedHelp    fs.FS
	EmbeddedAssets  fs.FS
	Out             io.Writer
	MiddlewaresFunc cli.CobraMiddlewaresFunc
}

func NewHost(providers *providerapi.ProviderRegistry, runtimeSpec *RuntimeSpec) *Host {
	return NewHostWithOptions(providers, runtimeSpec, HostOptions{})
}

func NewHostWithOptions(providers *providerapi.ProviderRegistry, runtimeSpec *RuntimeSpec, opts HostOptions) *Host {
	services := HostServices{Assets: NewAssetStore(opts.EmbeddedAssets, runtimeSpec)}
	middlewaresFunc := opts.MiddlewaresFunc
	if middlewaresFunc == nil {
		middlewaresFunc = MiddlewaresFromSpec(runtimeSpec)
	}
	return &Host{
		Providers:       providers,
		RuntimeSpec:     runtimeSpec,
		Factory:         NewRuntimeFactory(providers, runtimeSpec, services),
		EmbeddedJSVerbs: opts.EmbeddedJSVerbs,
		EmbeddedHelp:    opts.EmbeddedHelp,
		EmbeddedAssets:  opts.EmbeddedAssets,
		Services:        services,
		Out:             opts.Out,
		MiddlewaresFunc: middlewaresFunc,
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.RuntimeSpec == nil {
		return
	}
	if err := installRootFramework(root, h.RuntimeSpec, frameworkOptions{Providers: h.Providers, EmbeddedHelp: h.EmbeddedHelp}); err != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	if h.RuntimeSpec.Commands.Eval.Enabled {
		h.AttachEval(root)
	}
	if h.RuntimeSpec.Commands.Run.Enabled {
		h.AttachRun(root)
	}
	if h.RuntimeSpec.Commands.Repl.Enabled {
		h.AttachRepl(root)
	}
	h.AttachModules(root)
	h.AttachSelectedModules(root)
	if h.RuntimeSpec.Commands.JSVerbs.Enabled {
		h.AttachVerbs(root)
	}
	h.AttachCommandProviders(root)
}

func (h *Host) AttachEval(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	out := h.Out
	if out == nil {
		out = root.OutOrStdout()
	}
	cmd, err := buildGlazedCobraCommand(newEvalCommand(h.Factory, h.RuntimeSpec, out), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.RuntimeSpec.Commands.Eval, "eval"), "Evaluate JavaScript in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRun(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newRunCommand(h.Factory, h.RuntimeSpec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.RuntimeSpec.Commands.Run, "run"), "Execute a JavaScript file in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRepl(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newTUICommand(h.Factory, h.RuntimeSpec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.RuntimeSpec.Commands.Repl, "repl"), "Run an interactive TUI REPL for a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachModules(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newModulesCommand(h.Providers, h.RuntimeSpec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub("modules", "List provider modules compiled into this generated binary", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachSelectedModules(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newSelectedModulesCommand(h.RuntimeSpec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub("selected-modules", "List require() modules selected for this generated runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachVerbs(root *cobra.Command) {
	if root == nil || h == nil || h.RuntimeSpec == nil {
		return
	}
	if commandMount(h.RuntimeSpec.Commands.JSVerbs) == "root" {
		cmds, err := buildVerbCommands(h.Providers, h.Factory, h.RuntimeSpec, h.EmbeddedJSVerbs)
		if err != nil {
			root.AddCommand(commandErrorStub(commandName(h.RuntimeSpec.Commands.JSVerbs, "verbs"), "Run configured JavaScript verb commands", err))
			return
		}
		middlewaresFunc := h.MiddlewaresFunc
		if middlewaresFunc == nil {
			middlewaresFunc = cli.CobraCommandDefaultMiddlewares
		}
		if err := cli.AddCommandsToRootCommand(root, cmds, nil, cli.WithParserConfig(cli.CobraParserConfig{MiddlewaresFunc: middlewaresFunc})); err != nil {
			root.AddCommand(commandErrorStub(commandName(h.RuntimeSpec.Commands.JSVerbs, "verbs"), "Run configured JavaScript verb commands", err))
		}
		return
	}
	root.AddCommand(newVerbsCommand(h.Providers, h.Factory, h.RuntimeSpec, h.EmbeddedJSVerbs, h.MiddlewaresFunc))
}
