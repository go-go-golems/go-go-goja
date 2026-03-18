package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/structpb"
)

type greeterModule struct{}

func (greeterModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "plugin:greeter",
		Version:    "v1",
		Exports: []*contract.ExportSpec{
			{
				Name: "greet",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
			{
				Name:    "strings",
				Kind:    contract.ExportKind_EXPORT_KIND_OBJECT,
				Methods: []string{"upper", "lower"},
			},
			{
				Name:    "meta",
				Kind:    contract.ExportKind_EXPORT_KIND_OBJECT,
				Methods: []string{"pid"},
			},
		},
	}, nil
}

func (greeterModule) Invoke(_ context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	switch req.GetExportName() {
	case "greet":
		name := "world"
		if len(req.GetArgs()) > 0 && req.GetArgs()[0] != nil {
			if candidate := strings.TrimSpace(req.GetArgs()[0].GetStringValue()); candidate != "" {
				name = candidate
			}
		}
		return newStringResponse(fmt.Sprintf("hello, %s", name))
	case "strings":
		return invokeStrings(req)
	case "meta":
		if req.GetMethodName() != "pid" {
			return nil, fmt.Errorf("unsupported meta method %q", req.GetMethodName())
		}
		return newNumberResponse(float64(os.Getpid()))
	default:
		return nil, fmt.Errorf("unsupported export %q", req.GetExportName())
	}
}

func invokeStrings(req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	if len(req.GetArgs()) == 0 {
		return newStringResponse("")
	}
	input := req.GetArgs()[0].GetStringValue()

	switch req.GetMethodName() {
	case "upper":
		return newStringResponse(strings.ToUpper(input))
	case "lower":
		return newStringResponse(strings.ToLower(input))
	default:
		return nil, fmt.Errorf("unsupported strings method %q", req.GetMethodName())
	}
}

func newStringResponse(value string) (*contract.InvokeResponse, error) {
	result, err := structpb.NewValue(value)
	if err != nil {
		return nil, err
	}
	return &contract.InvokeResponse{Result: result}, nil
}

func newNumberResponse(value float64) (*contract.InvokeResponse, error) {
	result, err := structpb.NewValue(value)
	if err != nil {
		return nil, err
	}
	return &contract.InvokeResponse{Result: result}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  shared.Handshake,
		VersionedPlugins: shared.VersionedServerPluginSets(greeterModule{}),
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}
