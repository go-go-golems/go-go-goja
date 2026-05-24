package providerapi

import (
	"context"
	"encoding/json"

	"github.com/dop251/goja_nodejs/require"
)

type ModuleFactory func(ModuleContext) (require.ModuleLoader, error)

type ModuleContext struct {
	Context context.Context
	Name    string
	As      string
	Config  json.RawMessage
	Host    HostServices
}

type HostServices interface{}

type Module struct {
	Name         string
	DefaultAs    string
	Description  string
	ConfigSchema json.RawMessage
	New          ModuleFactory
}

func (m Module) applyToPackage(pkg *Package) error {
	return pkg.addModule(m)
}
