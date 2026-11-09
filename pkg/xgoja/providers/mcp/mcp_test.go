package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const lookPackageID = "mcp-test-look"

type testLook struct{ label string }

func (l *testLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: []byte("png:" + l.label), Caption: l.label}}, nil
}

func (l *testLook) ResultPlaceholder() string { return "look " + l.label }

func registerLookProvider(registry *providerapi.ProviderRegistry) error {
	return registry.Package(lookPackageID, providerapi.Module{
		Name:      "look",
		DefaultAs: "look",
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(vm *goja.Runtime, module *goja.Object) {
				exports := module.Get("exports").(*goja.Object)
				_ = exports.Set("look", func(label string) *testLook { return &testLook{label: label} })
			}, nil
		},
	})
}

func newTestCommandContext(t *testing.T, config json.RawMessage) providerapi.CommandSetContext {
	t.Helper()
	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register mcp: %v", err)
	}
	if err := registerLookProvider(providers); err != nil {
		t.Fatalf("register look: %v", err)
	}
	plan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: lookPackageID, Name: "look", As: "look"}}}}
	return providerapi.CommandSetContext{
		Context:        context.Background(),
		PackageID:      PackageID,
		Name:           CommandSetName,
		Mount:          "mcp",
		Config:         config,
		Providers:      providers,
		RuntimeFactory: app.NewRuntimeFactory(providers, plan, app.HostServices{}),
	}
}

func TestRegister(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register: %v", err)
	}
	provider, ok := registry.ResolveCommandSetProvider(PackageID, CommandSetName)
	if !ok {
		t.Fatal("expected mcp command set provider")
	}
	if provider.DefaultMount != "mcp" || provider.NewCommandSet == nil || len(provider.ConfigSchema) == 0 {
		t.Fatalf("provider = %#v", provider)
	}
	if !json.Valid(provider.ConfigSchema) {
		t.Fatalf("config schema is not JSON: %s", provider.ConfigSchema)
	}
}

func TestCommandSetBuildsServeCommand(t *testing.T) {
	ctx := newTestCommandContext(t, json.RawMessage(`{"evalTimeoutMs": 12000}`))
	provider, _ := ctx.Providers.ResolveCommandSetProvider(PackageID, CommandSetName)
	set, err := provider.NewCommandSet(ctx)
	if err != nil {
		t.Fatalf("new command set: %v", err)
	}
	if len(set.Commands) != 1 || set.Commands[0].Description().Name != "serve" {
		t.Fatalf("commands = %#v", set.Commands)
	}
	flags := set.Commands[0].Description().GetDefaultFlags()
	for _, name := range []string{"http", "allow-remote", "eval-timeout-ms"} {
		if _, ok := flags.Get(name); !ok {
			t.Fatalf("missing flag %s", name)
		}
	}
	if def, _ := flags.Get("eval-timeout-ms"); def.Default == nil || *def.Default != 12000 {
		t.Fatalf("eval-timeout-ms default = %#v", def.Default)
	}
}

func TestCommandSetRejectsBadConfigAndMissingFactory(t *testing.T) {
	ctx := newTestCommandContext(t, json.RawMessage(`{"unknown": true}`))
	if _, err := newCommandSet(ctx); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
	ctx = newTestCommandContext(t, nil)
	ctx.RuntimeFactory = nil
	if _, err := newCommandSet(ctx); err == nil {
		t.Fatal("expected missing runtime factory error")
	}
}

func TestNewServerEvaluatesWithSelectedModules(t *testing.T) {
	bg := context.Background()
	commandCtx := newTestCommandContext(t, nil)
	server, replApp, err := NewServer(bg, commandCtx, Config{Name: "cv-agent"}, 0, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(func() { _ = replApp.Close(bg) })

	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(bg, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "xgoja-mcp-test"}, nil)
	session, err := client.Connect(bg, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		_ = serverSession.Wait()
	})
	if name := session.InitializeResult().ServerInfo.Name; name != "cv-agent" {
		t.Fatalf("server name = %q", name)
	}

	result, err := session.CallTool(bg, &mcpsdk.CallToolParams{Name: "eval", Arguments: map[string]any{
		"code": `const { look } = require("look"); ({ before: look("before"), n: 1 })`,
	}})
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	text := result.Content[0].(*mcpsdk.TextContent).Text
	if result.IsError || !strings.HasPrefix(text, "{before: [content 1: look before], n: 1}") {
		t.Fatalf("eval: isError=%v text=%q", result.IsError, text)
	}
	if len(result.Content) != 2 {
		t.Fatalf("content = %#v", result.Content)
	}
	if image, ok := result.Content[1].(*mcpsdk.ImageContent); !ok || string(image.Data) != "png:before" {
		t.Fatalf("image = %#v", result.Content[1])
	}

	result, err = session.CallTool(bg, &mcpsdk.CallToolParams{Name: "eval", Arguments: map[string]any{"code": `typeof look`}})
	if err != nil || result.Content[0].(*mcpsdk.TextContent).Text != `"function"` {
		t.Fatalf("binding did not persist: err=%v result=%#v", err, result)
	}
}

func TestValidateListenAddress(t *testing.T) {
	if err := validateListenAddress("127.0.0.1:0", false); err != nil {
		t.Fatalf("loopback: %v", err)
	}
	if err := validateListenAddress("0.0.0.0:3000", false); err == nil || !strings.Contains(err.Error(), "--allow-remote") {
		t.Fatalf("remote without ack: %v", err)
	}
	if err := validateListenAddress("0.0.0.0:3000", true); err != nil {
		t.Fatalf("remote with ack: %v", err)
	}
}
