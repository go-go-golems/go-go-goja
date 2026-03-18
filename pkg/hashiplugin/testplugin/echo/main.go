package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/structpb"
)

type echoModule struct{}

func (echoModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "plugin:echo",
		Version:    "v1",
		Exports: []*contract.ExportSpec{
			{
				Name: "ping",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
			{
				Name: "pid",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
			{
				Name:    "math",
				Kind:    contract.ExportKind_EXPORT_KIND_OBJECT,
				Methods: []string{"add"},
			},
		},
	}, nil
}

func (echoModule) Invoke(_ context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	switch req.GetExportName() {
	case "ping":
		if len(req.GetArgs()) == 0 {
			return &contract.InvokeResponse{Result: structpb.NewNullValue()}, nil
		}
		return &contract.InvokeResponse{Result: req.GetArgs()[0]}, nil
	case "pid":
		value, err := structpb.NewValue(float64(os.Getpid()))
		if err != nil {
			return nil, err
		}
		return &contract.InvokeResponse{Result: value}, nil
	case "math":
		if req.GetMethodName() != "add" {
			return nil, fmt.Errorf("unsupported method %q", req.GetMethodName())
		}
		if len(req.GetArgs()) != 2 {
			return nil, fmt.Errorf("math.add expects 2 arguments")
		}
		sum := req.GetArgs()[0].GetNumberValue() + req.GetArgs()[1].GetNumberValue()
		value, err := structpb.NewValue(sum)
		if err != nil {
			return nil, err
		}
		return &contract.InvokeResponse{Result: value}, nil
	default:
		return nil, fmt.Errorf("unsupported export %q", req.GetExportName())
	}
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedServerPluginSets(echoModule{}),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}
