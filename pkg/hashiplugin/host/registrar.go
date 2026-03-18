package host

import (
	"context"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
)

// Registrar discovers, validates, and registers plugin-backed runtime modules.
type Registrar struct {
	config Config
}

// NewRegistrar creates a runtime-scoped module registrar for HashiCorp plugins.
func NewRegistrar(config Config) *Registrar {
	return &Registrar{config: config}
}

func (r *Registrar) ID() string {
	return "hashiplugin-registrar"
}

func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}

	cfg := r.config.withDefaults()
	paths, err := Discover(cfg)
	if err != nil {
		if cfg.Report != nil {
			cfg.Report.SetError(err)
		}
		return err
	}
	if cfg.Report != nil {
		cfg.Report.SetCandidates(paths)
	}
	if len(paths) == 0 {
		return nil
	}

	loaded, err := LoadModules(cfg, paths)
	if err != nil {
		if cfg.Report != nil {
			cfg.Report.SetError(err)
		}
		return err
	}
	for _, mod := range loaded {
		if err := RegisterModule(reg, mod); err != nil {
			closeLoaded(loaded)
			if cfg.Report != nil {
				cfg.Report.SetError(err)
			}
			return err
		}
		if cfg.Report != nil {
			cfg.Report.AddLoaded(mod)
		}
	}
	if ctx != nil {
		ctx.SetValue(RuntimeLoadedModulesContextKey, SnapshotLoadedModules(loaded))
	}
	if ctx != nil && ctx.AddCloser != nil {
		if err := ctx.AddCloser(func(context.Context) error {
			closeLoaded(loaded)
			return nil
		}); err != nil {
			closeLoaded(loaded)
			return err
		}
	}
	return nil
}
