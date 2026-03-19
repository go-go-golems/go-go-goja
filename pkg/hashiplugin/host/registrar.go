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
	if err := validateUniqueLoadedModules(loaded); err != nil {
		closeLoaded(loaded)
		if cfg.Report != nil {
			cfg.Report.SetError(err)
		}
		return err
	}
	for _, mod := range loaded {
		if err := RegisterModule(reg, mod, runtimeContext(ctx)); err != nil {
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

func runtimeContext(ctx *engine.RuntimeModuleContext) context.Context {
	if ctx == nil || ctx.Context == nil {
		return context.Background()
	}
	return ctx.Context
}

func validateUniqueLoadedModules(loaded []*LoadedModule) error {
	if len(loaded) < 2 {
		return nil
	}

	seen := map[string]string{}
	for _, mod := range loaded {
		if mod == nil {
			continue
		}
		name := mod.RequireName()
		if name == "" {
			continue
		}
		if firstPath, ok := seen[name]; ok {
			return fmt.Errorf("duplicate plugin module %q discovered at %q and %q", name, firstPath, mod.Path)
		}
		seen[name] = mod.Path
	}
	return nil
}
