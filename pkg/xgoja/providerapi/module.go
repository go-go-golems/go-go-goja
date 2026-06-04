package providerapi

import (
	"context"
	"encoding/json"
	"io/fs"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// ModuleSetupContext is passed to a provider module while it creates the
// CommonJS module loader for one selected runtime module instance.
type ModuleSetupContext struct {
	Context      context.Context
	Name         string
	As           string
	Config       json.RawMessage
	Host         HostServices
	RuntimeOwner runtimeowner.RuntimeOwner
	AddCloser    func(func(context.Context) error) error
}

type AssetResolver interface {
	ResolveAsset(id string) (fs.FS, string, bool)
}

type HostServices interface {
	AssetResolver() AssetResolver
}

// HostServiceLookup is implemented by host service bags that carry arbitrary
// provider-contributed services. xgoja core treats service values as opaque;
// provider packages define stable keys and typed payloads for their own use.
type HostServiceLookup interface {
	HostService(key string) (any, bool)
	HostServiceValues(key string) []any
}

// Module describes a provider-owned native module that xgoja can select into a
// runtime profile. NewModuleFactory creates the CommonJS loader during runtime
// setup for each selected module instance.
type Module struct {
	Name             string
	DefaultAs        string
	Description      string
	ConfigSchema     json.RawMessage
	NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
}

func (m Module) applyToPackage(pkg *Package) error {
	return pkg.addModule(m)
}
