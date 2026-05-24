package app

import (
	"io/fs"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Host struct {
	Providers       *providerapi.Registry
	Spec            *Spec
	Factory         *RuntimeFactory
	EmbeddedJSVerbs fs.FS
}

type HostOptions struct {
	EmbeddedJSVerbs fs.FS
}

func NewHost(providers *providerapi.Registry, spec *Spec) *Host {
	return NewHostWithOptions(providers, spec, HostOptions{})
}

func NewHostWithOptions(providers *providerapi.Registry, spec *Spec, opts HostOptions) *Host {
	return &Host{
		Providers:       providers,
		Spec:            spec,
		Factory:         NewRuntimeFactory(providers, spec),
		EmbeddedJSVerbs: opts.EmbeddedJSVerbs,
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.Spec == nil {
		return
	}
	if err := installRootFramework(root, h.Spec); err != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	if h.Spec.Commands.Repl.Enabled {
		h.AttachEval(root)
	}
	h.AttachModules(root)
	if h.Spec.Commands.JSVerbs.Enabled {
		h.AttachVerbs(root)
	}
}

func (h *Host) AttachEval(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	root.AddCommand(newEvalCommand(h.Factory, h.Spec))
}

func (h *Host) AttachModules(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	cmd, err := buildGlazedCobraCommand(newModulesCommand(h.Providers, h.Spec))
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
	root.AddCommand(newVerbsCommand(h.Providers, h.Factory, h.Spec, h.EmbeddedJSVerbs))
}
