package sdk

import (
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
)

func Serve(mod contract.JSModule) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedServerPluginSets(mod),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}

func ServeModule(name string, opts ...ModuleOption) {
	Serve(MustModule(name, opts...))
}
