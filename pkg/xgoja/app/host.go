package app

import (
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

type Host struct {
	Providers *providerapi.Registry
	Spec      *Spec
	Factory   *RuntimeFactory
}

func NewHost(providers *providerapi.Registry, spec *Spec) *Host {
	return &Host{
		Providers: providers,
		Spec:      spec,
		Factory:   NewRuntimeFactory(providers, spec),
	}
}

func (h *Host) AttachDefaultCommands(root *cobra.Command) {
	if root == nil || h == nil || h.Spec == nil {
		return
	}
	h.AttachEval(root)
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
	root.AddCommand(newModulesCommand(h.Providers, h.Spec))
}

func (h *Host) AttachVerbs(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	root.AddCommand(newVerbsCommand(h.Factory, h.Spec))
}
