package app

import (
	"io"
	"io/fs"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Host struct {
	Providers       *providerapi.Registry
	Spec            *Spec
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

func NewHost(providers *providerapi.Registry, spec *Spec) *Host {
	return NewHostWithOptions(providers, spec, HostOptions{})
}

func NewHostWithOptions(providers *providerapi.Registry, spec *Spec, opts HostOptions) *Host {
	services := HostServices{Assets: NewAssetStore(opts.EmbeddedAssets, spec)}
	middlewaresFunc := opts.MiddlewaresFunc
	if middlewaresFunc == nil {
		middlewaresFunc = MiddlewaresFromSpec(spec)
	}
	return &Host{
		Providers:       providers,
		Spec:            spec,
		Factory:         NewRuntimeFactory(providers, spec, services),
		EmbeddedJSVerbs: opts.EmbeddedJSVerbs,
		EmbeddedHelp:    opts.EmbeddedHelp,
		EmbeddedAssets:  opts.EmbeddedAssets,
		Services:        services,
		Out:             opts.Out,
		MiddlewaresFunc: middlewaresFunc,
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.Spec == nil {
		return
	}
	if err := installRootFramework(root, h.Spec, frameworkOptions{Providers: h.Providers, EmbeddedHelp: h.EmbeddedHelp}); err != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	if h.Spec.Commands.Eval.Enabled {
		h.AttachEval(root)
	}
	if h.Spec.Commands.Run.Enabled {
		h.AttachRun(root)
	}
	if h.Spec.Commands.Repl.Enabled {
		h.AttachRepl(root)
	}
	h.AttachModules(root)
	if h.Spec.Commands.JSVerbs.Enabled {
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
	cmd, err := buildGlazedCobraCommand(newEvalCommand(h.Factory, h.Spec, out), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.Spec.Commands.Eval, "eval"), "Evaluate JavaScript in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRun(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newRunCommand(h.Factory, h.Spec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.Spec.Commands.Run, "run"), "Execute a JavaScript file in a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachRepl(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newTUICommand(h.Factory, h.Spec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub(commandName(h.Spec.Commands.Repl, "repl"), "Run an interactive TUI REPL for a generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachModules(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newModulesCommand(h.Providers, h.Spec), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub("modules", "List provider modules registered in this generated binary", err))
		return
	}
	root.AddCommand(cmd)
}

func (h *Host) AttachVerbs(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	root.AddCommand(newVerbsCommand(h.Providers, h.Factory, h.Spec, h.EmbeddedJSVerbs, h.MiddlewaresFunc))
}
