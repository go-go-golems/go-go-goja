package providerapi

import (
	"context"
	"encoding/json"
	"io/fs"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type ModuleFactory func(ModuleContext) (require.ModuleLoader, error)

type ModuleContext struct {
	Context      context.Context
	Name         string
	As           string
	Config       json.RawMessage
	Host         HostServices
	RuntimeOwner runtimeowner.RuntimeOwner
}

type AssetResolver interface {
	ResolveAsset(id string) (fs.FS, string, bool)
}

type HostServices interface {
	AssetResolver() AssetResolver
}

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
