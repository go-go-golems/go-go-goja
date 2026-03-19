package shared

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/structpb"
)

type testModule struct{}

func (testModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "plugin:echo",
		Version:    "v1",
		Exports: []*contract.ExportSpec{
			{
				Name: "ping",
				Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
			},
		},
	}, nil
}

func (testModule) Invoke(_ context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	value := structpb.NewNullValue()
	if len(req.GetArgs()) > 0 {
		value = req.GetArgs()[0]
	}
	return &contract.InvokeResponse{Result: value}, nil
}

func TestJSModulePluginGRPCRoundTrip(t *testing.T) {
	client, server := plugin.TestPluginGRPCConn(t, false, ServerPluginSet(testModule{}))
	defer func() { _ = client.Close() }()
	defer server.Stop()

	raw, err := client.Dispense(ServiceName)
	if err != nil {
		t.Fatalf("dispense plugin: %v", err)
	}

	mod, ok := raw.(contract.JSModule)
	if !ok {
		t.Fatalf("dispensed type = %T, want contract.JSModule", raw)
	}

	manifest, err := mod.Manifest(context.Background())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if manifest.GetModuleName() != "plugin:echo" {
		t.Fatalf("module name = %q, want plugin:echo", manifest.GetModuleName())
	}

	resp, err := mod.Invoke(context.Background(), &contract.InvokeRequest{
		ExportName: "ping",
		Args: []*structpb.Value{
			structpb.NewStringValue("hello"),
		},
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got := resp.GetResult().GetStringValue(); got != "hello" {
		t.Fatalf("result = %q, want hello", got)
	}
}
