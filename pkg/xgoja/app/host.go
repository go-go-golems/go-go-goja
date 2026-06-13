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
	RuntimePlan     *RuntimePlan
	Factory         *RuntimeFactory
	EmbeddedJSVerbs fs.FS
	EmbeddedHelp    fs.FS
	EmbeddedAssets  fs.FS
	Services        HostServices
	Out             io.Writer
	MiddlewaresFunc cli.CobraMiddlewaresFunc
}

type HostOptions struct {
	EmbeddedJSVerbs   fs.FS
	EmbeddedHelp      fs.FS
	EmbeddedAssets    fs.FS
	Out               io.Writer
	MiddlewaresFunc   cli.CobraMiddlewaresFunc
	ConfigureServices func(*HostServices)
}

func NewHost(providers *providerapi.ProviderRegistry, runtimePlan *RuntimePlan) *Host {
	return NewHostWithOptions(providers, runtimePlan, HostOptions{})
}

func NewHostWithOptions(providers *providerapi.ProviderRegistry, runtimePlan *RuntimePlan, opts HostOptions) *Host {
	services := HostServices{Assets: NewAssetStore(opts.EmbeddedAssets, runtimePlan)}
	if opts.ConfigureServices != nil {
		opts.ConfigureServices(&services)
	}
	middlewaresFunc := opts.MiddlewaresFunc
	if middlewaresFunc == nil {
		middlewaresFunc = MiddlewaresFromSpec(runtimePlan)
	}
	return &Host{
		Providers:       providers,
		RuntimePlan:     runtimePlan,
		Factory:         NewRuntimeFactory(providers, runtimePlan, services),
		EmbeddedJSVerbs: opts.EmbeddedJSVerbs,
		EmbeddedHelp:    opts.EmbeddedHelp,
		EmbeddedAssets:  opts.EmbeddedAssets,
		Services:        services,
		Out:             opts.Out,
		MiddlewaresFunc: middlewaresFunc,
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.RuntimePlan == nil {
		return
	}
	if err := installRootFramework(root, h.RuntimePlan, frameworkOptions{Providers: h.Providers, EmbeddedHelp: h.EmbeddedHelp}); err != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	if _, ok := h.RuntimePlan.commandByType("builtin.eval"); ok {
		h.AttachEval(root)
	}
	if _, ok := h.RuntimePlan.commandByType("builtin.run"); ok {
		h.AttachRun(root)
	}
	if _, ok := h.RuntimePlan.commandByType("builtin.repl"); ok {
		h.AttachRepl(root)
	}
	h.AttachModules(root)
	h.AttachSelectedModules(root)
	h.AttachTypes(root)
	if _, ok := h.RuntimePlan.commandByType("builtin.jsverbs"); ok {
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
	cmd, err := buildGlazedCobraCommand(newEvalCommand(h.Factory, h.RuntimePlan, out), h.MiddlewaresFunc)
	if err != nil {
		command, _ := h.RuntimePlan.commandByType("builtin.eval")
		root.AddCommand(commandErrorStub(commandName(command, "eval"), "Evaluate JavaScript in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRun(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newRunCommand(h.Factory, h.RuntimePlan), h.MiddlewaresFunc)
	if err != nil {
		command, _ := h.RuntimePlan.commandByType("builtin.run")
		root.AddCommand(commandErrorStub(commandName(command, "run"), "Execute a JavaScript file in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRepl(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newTUICommand(h.Factory, h.RuntimePlan), h.MiddlewaresFunc)
	if err != nil {
		command, _ := h.RuntimePlan.commandByType("builtin.repl")
		root.AddCommand(commandErrorStub(commandName(command, "repl"), "Run an interactive TUI REPL for a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachModules(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newModulesCommand(h.Providers, h.RuntimePlan), h.MiddlewaresFunc)
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
	cmd, err := buildGlazedCobraCommand(newSelectedModulesCommand(h.RuntimePlan), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub("selected-modules", "List require() modules selected for this generated runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachVerbs(root *cobra.Command) {
	if root == nil || h == nil || h.RuntimePlan == nil {
		return
	}
	jsverbsCommand, _ := h.RuntimePlan.commandByType("builtin.jsverbs")
	if commandMount(jsverbsCommand) == "root" {
		cmds, err := buildVerbCommands(h.Providers, h.Factory, h.RuntimePlan, h.EmbeddedJSVerbs)
		if err != nil {
			root.AddCommand(commandErrorStub(commandName(jsverbsCommand, "verbs"), "Run configured JavaScript verb commands", err))
			return
		}
		middlewaresFunc := h.MiddlewaresFunc
		if middlewaresFunc == nil {
			middlewaresFunc = cli.CobraCommandDefaultMiddlewares
		}
		if err := cli.AddCommandsToRootCommand(root, cmds, nil, cli.WithParserConfig(cli.CobraParserConfig{MiddlewaresFunc: middlewaresFunc})); err != nil {
			root.AddCommand(commandErrorStub(commandName(jsverbsCommand, "verbs"), "Run configured JavaScript verb commands", err))
		}
		return
	}
	root.AddCommand(newVerbsCommand(h.Providers, h.Factory, h.RuntimePlan, h.EmbeddedJSVerbs, h.MiddlewaresFunc))
}
