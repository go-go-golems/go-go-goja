package main

import (
	"context"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
)

type invalidModule struct{}

func (invalidModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "echo",
		Version:    "v1",
		Exports: []*contract.ExportSpec{
			{
				Name: "ping",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
		},
	}, nil
}

func (invalidModule) Invoke(context.Context, *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	return &contract.InvokeResponse{}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedServerPluginSets(invalidModule{}),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}
