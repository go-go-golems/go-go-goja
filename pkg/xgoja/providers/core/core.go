package core

import (
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/crypto"
	_ "github.com/go-go-golems/go-go-goja/modules/events"
	_ "github.com/go-go-golems/go-go-goja/modules/path"
	_ "github.com/go-go-golems/go-go-goja/modules/time"
	_ "github.com/go-go-golems/go-go-goja/modules/timer"
	_ "github.com/go-go-golems/go-go-goja/modules/yaml"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "go-go-goja-core"

var coreModuleNames = []string{
	"path",
	"node:path",
	"yaml",
	"crypto",
	"node:crypto",
	"time",
	"timer",
	"events",
	"node:events",
}

// Register exposes the data-oriented first-party go-go-goja modules as an
// xgoja provider package. These modules do not intentionally perform host
// filesystem, process, or database operations.
func Register(registry *providerapi.ProviderRegistry) error {
	entries := make([]providerapi.Entry, 0, len(coreModuleNames))
	for _, name := range coreModuleNames {
		mod := modules.GetModule(name)
		if mod == nil {
			return fmt.Errorf("core module %q is not registered", name)
		}
		entries = append(entries, nativeModuleEntry(mod))
	}
	return registry.Package(PackageID, entries...)
}

func nativeModuleEntry(mod modules.NativeModule) providerapi.Module {
	return providerapi.Module{
		Name:        mod.Name(),
		DefaultAs:   mod.Name(),
		Description: mod.Doc(),
		TypeScript:  nativeModuleTypeScript(mod),
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return mod.Loader, nil
		},
	}
}

func nativeModuleTypeScript(mod modules.NativeModule) *spec.Module {
	declarer, ok := mod.(modules.TypeScriptDeclarer)
	if !ok {
		return nil
	}
	return declarer.TypeScriptModule()
}
