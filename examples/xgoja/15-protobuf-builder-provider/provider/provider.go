package provider

import (
	"github.com/dop251/goja_nodejs/require"
	taskpb "github.com/go-go-golems/go-go-goja/examples/xgoja/15-protobuf-builder-provider/proto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const (
	PackageID  = "protobuf-builder-example"
	ModuleName = "examples.xgoja.protobuf.v1"
)

// Register exposes the generated protobuf builder module as an xgoja provider
// module. xgoja hosts can select ModuleName and receive both the Goja runtime
// loader and the generated TypeScript declaration module.
func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:        ModuleName,
		Description: "Generated Goja protobuf builders for the xgoja protobuf example schema",
		TypeScript:  taskpb.GojaBuilderFileTaskProtoTypeScriptModule(ModuleName),
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return taskpb.NewGojaBuilderFileTaskProtoLoader(ModuleName), nil
		},
	})
}
