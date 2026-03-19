package host

import (
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"google.golang.org/protobuf/proto"
)

const RuntimeLoadedModulesContextKey = "hashiplugin.loaded-modules"

type LoadedModuleInfo struct {
	Path     string
	Manifest *contract.ModuleManifest
}

func SnapshotLoadedModules(modules []*LoadedModule) []LoadedModuleInfo {
	if len(modules) == 0 {
		return nil
	}

	out := make([]LoadedModuleInfo, 0, len(modules))
	for _, mod := range modules {
		if mod == nil || mod.Manifest == nil {
			continue
		}
		out = append(out, LoadedModuleInfo{
			Path:     mod.Path,
			Manifest: proto.Clone(mod.Manifest).(*contract.ModuleManifest),
		})
	}
	return out
}
