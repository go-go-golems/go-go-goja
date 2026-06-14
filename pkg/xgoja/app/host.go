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
	SourceRegistry  *SourceRegistry
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
	sourceRegistry := NewSourceRegistryWithRuntimeAliases(providers, opts.EmbeddedJSVerbs, runtimePlan.allSources(), runtimePlanModuleAliases(runtimePlan.runtimeModules()))
	services := HostServices{Assets: NewAssetStoreFromSources(opts.EmbeddedAssets, sourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindAssets))}
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
		SourceRegistry:  sourceRegistry,
		Out:             opts.Out,
		MiddlewaresFunc: middlewaresFunc,
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.RuntimePlan == nil {
		return
	}
	if err := installRootFramework(root, h.RuntimePlan, frameworkOptions{Providers: h.Providers, SourceRegistry: h.SourceRegistry, EmbeddedHelp: h.EmbeddedHelp}); err != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	for _, command := range h.RuntimePlan.runtimeCommands() {
		h.AttachCommandPlan(root, command)
	}
	h.AttachModules(root)
	h.AttachSelectedModules(root)
	h.AttachTypes(root)
}

func (h *Host) AttachCommandPlan(root *cobra.Command, command CommandPlan) {
	if root == nil || h == nil {
		return
	}
	switch command.Type {
	case "builtin.eval":
		h.AttachEval(root)
	case "builtin.run":
		h.AttachRun(root)
	case "builtin.repl":
		h.AttachRepl(root)
	case "builtin.jsverbs":
		h.attachVerbCommandPlan(root, command)
	case "provider.command-set":
		h.AttachCommandProvider(root, command)
	}
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
	h.attachVerbCommandPlan(root, jsverbsCommand)
}

func (h *Host) attachVerbCommandPlan(root *cobra.Command, jsverbsCommand CommandPlan) {
	if root == nil || h == nil || h.RuntimePlan == nil {
		return
	}
	sourceRegistry := h.SourceRegistry.Scoped(jsverbsCommand.Sources)
	if commandMount(jsverbsCommand) == "root" {
		cmds, err := buildVerbCommands(sourceRegistry, h.Factory, h.RuntimePlan)
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
	root.AddCommand(newVerbsCommand(sourceRegistry, h.Factory, h.RuntimePlan, jsverbsCommand, h.MiddlewaresFunc))
}
