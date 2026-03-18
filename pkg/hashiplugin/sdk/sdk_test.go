package sdk

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNewModuleBuildsManifestCompatibleWithHostValidation(t *testing.T) {
	mod, err := NewModule(
		"plugin:greeter",
		Version("v1"),
		Doc("Example greeter plugin"),
		Capabilities("strings", "examples"),
		Function("greet", func(context.Context, *Call) (any, error) {
			return "hello", nil
		}, ExportDoc("Greets a user")),
		Object("strings",
			ObjectDoc("String helpers"),
			Method("upper", func(_ context.Context, call *Call) (any, error) {
				return strings.ToUpper(call.StringDefault(0, "")), nil
			}, ExportDoc("Uppercase helper")),
			Method("lower", func(_ context.Context, call *Call) (any, error) {
				return strings.ToLower(call.StringDefault(0, "")), nil
			}),
		),
	)
	if err != nil {
		t.Fatalf("new module: %v", err)
	}

	manifest, err := mod.Manifest(context.Background())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	if err := host.ValidateManifest(host.Config{}, manifest); err != nil {
		t.Fatalf("host validate manifest: %v", err)
	}
	if got := manifest.GetVersion(); got != "v1" {
		t.Fatalf("version = %q, want v1", got)
	}
	if got := manifest.GetDoc(); got != "Example greeter plugin" {
		t.Fatalf("doc = %q", got)
	}
	if got := len(manifest.GetExports()); got != 2 {
		t.Fatalf("len(exports) = %d, want 2", got)
	}
	if got := manifest.GetExports()[0].GetName(); got != "greet" {
		t.Fatalf("first export = %q, want greet", got)
	}
	if got := manifest.GetExports()[1].GetMethods(); len(got) != 2 || got[0] != "upper" || got[1] != "lower" {
		t.Fatalf("object methods = %#v", got)
	}
}

func TestNewModuleRejectsDuplicateExport(t *testing.T) {
	_, err := NewModule(
		"plugin:dup",
		Function("ping", func(context.Context, *Call) (any, error) { return nil, nil }),
		Function("ping", func(context.Context, *Call) (any, error) { return nil, nil }),
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate export") {
		t.Fatalf("expected duplicate export error, got %v", err)
	}
}

func TestNewModuleRejectsObjectWithoutMethods(t *testing.T) {
	_, err := NewModule("plugin:bad", Object("math"))
	if err == nil || !strings.Contains(err.Error(), "must define at least one method") {
		t.Fatalf("expected missing methods error, got %v", err)
	}
}

func TestModuleInvokeFunctionAndObjectMethod(t *testing.T) {
	mod := MustModule(
		"plugin:greeter",
		Function("greet", func(_ context.Context, call *Call) (any, error) {
			return "hello, " + call.StringDefault(0, "world"), nil
		}),
		Object("math",
			Method("add", func(_ context.Context, call *Call) (any, error) {
				a, err := call.Float64(0)
				if err != nil {
					return nil, err
				}
				b, err := call.Float64(1)
				if err != nil {
					return nil, err
				}
				return a + b, nil
			}),
		),
	)

	resp, err := mod.Invoke(context.Background(), &contract.InvokeRequest{
		ExportName: "greet",
		Args: []*structpb.Value{
			structpb.NewStringValue("Manuel"),
		},
	})
	if err != nil {
		t.Fatalf("invoke greet: %v", err)
	}
	if got := resp.GetResult().GetStringValue(); got != "hello, Manuel" {
		t.Fatalf("greet result = %q", got)
	}

	resp, err = mod.Invoke(context.Background(), &contract.InvokeRequest{
		ExportName: "math",
		MethodName: "add",
		Args: []*structpb.Value{
			structpb.NewNumberValue(2),
			structpb.NewNumberValue(3),
		},
	})
	if err != nil {
		t.Fatalf("invoke math.add: %v", err)
	}
	if got := resp.GetResult().GetNumberValue(); got != 5 {
		t.Fatalf("math.add = %v, want 5", got)
	}
}

func TestModuleInvokeUnknownExportAndBadResult(t *testing.T) {
	mod := MustModule(
		"plugin:bad",
		Function("oops", func(context.Context, *Call) (any, error) {
			return make(chan int), nil
		}),
	)

	_, err := mod.Invoke(context.Background(), &contract.InvokeRequest{ExportName: "missing"})
	if err == nil || !strings.Contains(err.Error(), "unsupported export") {
		t.Fatalf("expected unsupported export error, got %v", err)
	}

	_, err = mod.Invoke(context.Background(), &contract.InvokeRequest{ExportName: "oops"})
	if err == nil || !strings.Contains(err.Error(), "encode result") {
		t.Fatalf("expected encode result error, got %v", err)
	}
}

func TestModuleBuiltThroughSDKWorksOverSharedGRPCTransport(t *testing.T) {
	mod := MustModule(
		"plugin:echo",
		Function("ping", func(_ context.Context, call *Call) (any, error) {
			if call.Len() == 0 {
				return nil, nil
			}
			return call.Value(0)
		}),
	)

	client, server := plugin.TestPluginGRPCConn(t, false, shared.ServerPluginSet(mod))
	defer func() { _ = client.Close() }()
	defer server.Stop()

	raw, err := client.Dispense(shared.ServiceName)
	if err != nil {
		t.Fatalf("dispense plugin: %v", err)
	}

	remote, ok := raw.(contract.JSModule)
	if !ok {
		t.Fatalf("dispensed type = %T, want contract.JSModule", raw)
	}

	manifest, err := remote.Manifest(context.Background())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if got := manifest.GetModuleName(); got != "plugin:echo" {
		t.Fatalf("manifest module = %q", got)
	}

	resp, err := remote.Invoke(context.Background(), &contract.InvokeRequest{
		ExportName: "ping",
		Args: []*structpb.Value{
			structpb.NewStringValue("hello"),
		},
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got := resp.GetResult().GetStringValue(); got != "hello" {
		t.Fatalf("result = %q", got)
	}
}
