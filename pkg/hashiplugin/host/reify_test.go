package host

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

type contextAwareModule struct{}

func (contextAwareModule) Manifest(context.Context) (*contract.ModuleManifest, error) {
	return &contract.ModuleManifest{
		ModuleName: "plugin:test:ctx",
		Exports: []*contract.ExportSpec{{
			Name: "ping",
			Kind: contract.ExportKind_EXPORT_KIND_FUNCTION,
		}},
	}, nil
}

func (contextAwareModule) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, nil
}

func TestRegisterModuleUsesRuntimeContextForInvocation(t *testing.T) {
	reg := require.NewRegistry()
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RegisterModule(reg, &LoadedModule{
		Path:     "/tmp/plugin:test:ctx",
		Manifest: mustManifest(t, contextAwareModule{}),
		Module:   contextAwareModule{},
	}, canceledCtx)
	if err != nil {
		t.Fatalf("register module: %v", err)
	}

	vm := goja.New()
	req := reg.Enable(vm)
	mod, err := req.Require("plugin:test:ctx")
	if err != nil {
		t.Fatalf("require plugin:test:ctx: %v", err)
	}

	fn, ok := goja.AssertFunction(mod.ToObject(vm).Get("ping"))
	if !ok {
		t.Fatalf("ping export is not a function")
	}

	_, err = fn(goja.Undefined())
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func mustManifest(t *testing.T, mod contract.JSModule) *contract.ModuleManifest {
	t.Helper()
	manifest, err := mod.Manifest(context.Background())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	return manifest
}
