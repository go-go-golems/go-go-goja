package sdk

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
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
			}, MethodSummary("Uppercase helper"), MethodDoc("Uppercase helper"), MethodTags("strings", "uppercase")),
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

	if err := contract.ValidateManifest(manifest, contract.ManifestValidationOptions{
		NamespacePrefix: DefaultNamespace,
	}); err != nil {
		t.Fatalf("shared validate manifest: %v", err)
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
	methods := manifest.GetExports()[1].GetMethodSpecs()
	if len(methods) != 2 || methods[0].GetName() != "upper" || methods[1].GetName() != "lower" {
		t.Fatalf("object methods = %#v", methods)
	}
	if got := methods[0].GetDoc(); got != "Uppercase helper" {
		t.Fatalf("first method doc = %q, want Uppercase helper", got)
	}
	if got := methods[0].GetSummary(); got != "Uppercase helper" {
		t.Fatalf("first method summary = %q, want Uppercase helper", got)
	}
	if got := methods[0].GetTags(); len(got) != 2 || got[0] != "strings" || got[1] != "uppercase" {
		t.Fatalf("first method tags = %#v", got)
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
	if err == nil || !strings.Contains(err.Error(), "must define methods") {
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

func TestEncodeResultNormalizesTypedSlicesAndMaps(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		result, err := encodeResult([]string{"a", "b"})
		if err != nil {
			t.Fatalf("encode []string: %v", err)
		}
		got := result.AsInterface()
		values, ok := got.([]any)
		if !ok {
			t.Fatalf("result type = %T, want []any", got)
		}
		if len(values) != 2 || values[0] != "a" || values[1] != "b" {
			t.Fatalf("result = %#v", values)
		}
	})

	t.Run("int slice", func(t *testing.T) {
		result, err := encodeResult([]int{1, 2, 3})
		if err != nil {
			t.Fatalf("encode []int: %v", err)
		}
		got := result.AsInterface()
		values, ok := got.([]any)
		if !ok {
			t.Fatalf("result type = %T, want []any", got)
		}
		if len(values) != 3 || values[0] != float64(1) || values[1] != float64(2) || values[2] != float64(3) {
			t.Fatalf("result = %#v", values)
		}
	})

	t.Run("string map", func(t *testing.T) {
		result, err := encodeResult(map[string]string{"name": "Manuel"})
		if err != nil {
			t.Fatalf("encode map[string]string: %v", err)
		}
		got := result.AsInterface()
		values, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("result type = %T, want map[string]any", got)
		}
		if values["name"] != "Manuel" {
			t.Fatalf("result = %#v", values)
		}
	})

	t.Run("nested values", func(t *testing.T) {
		result, err := encodeResult(map[string]any{
			"name": "Manuel",
			"tags": []string{"plugins", "goja"},
			"scores": map[string]int{
				"one": 1,
				"two": 2,
			},
		})
		if err != nil {
			t.Fatalf("encode nested values: %v", err)
		}
		got := result.AsInterface()
		values, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("result type = %T, want map[string]any", got)
		}
		tags, ok := values["tags"].([]any)
		if !ok || len(tags) != 2 || tags[0] != "plugins" || tags[1] != "goja" {
			t.Fatalf("tags = %#v", values["tags"])
		}
		scores, ok := values["scores"].(map[string]any)
		if !ok || scores["one"] != float64(1) || scores["two"] != float64(2) {
			t.Fatalf("scores = %#v", values["scores"])
		}
	})
}

func TestEncodeResultRejectsUnsupportedShapes(t *testing.T) {
	_, err := encodeResult(map[int]string{1: "bad"})
	if err == nil || !strings.Contains(err.Error(), "unsupported map key type") {
		t.Fatalf("expected unsupported map key error, got %v", err)
	}

	_, err = encodeResult(func() {})
	if err == nil || !strings.Contains(err.Error(), "unsupported result type") {
		t.Fatalf("expected unsupported result type error, got %v", err)
	}
}
