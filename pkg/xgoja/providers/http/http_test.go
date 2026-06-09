package http

import (
	"context"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegister(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := registry.ResolveModule(PackageID, "express"); !ok {
		t.Fatal("expected express module")
	}
	if _, ok := registry.ResolveCommandSetProvider(PackageID, "serve"); !ok {
		t.Fatal("expected serve command provider")
	}
	caps, ok := registry.ResolvePackageCapabilities(PackageID)
	if !ok || len(caps) != 1 {
		t.Fatalf("capabilities = %#v ok=%v", caps, ok)
	}
}

func TestCapabilityProvidesHTTPSection(t *testing.T) {
	capability := newHTTPCapability()
	sections, err := capability.GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) != 1 || sections[0].GetSlug() != "http" {
		t.Fatalf("sections = %#v", sections)
	}
	if sections[0].GetPrefix() != "http-" {
		t.Fatalf("prefix = %q", sections[0].GetPrefix())
	}
}

func TestCapabilityRejectsNilRuntimeInitializerHandle(t *testing.T) {
	capability := newHTTPCapability()
	if err := capability.InitRuntimeFromSections(context.Background(), nil, nil); err == nil {
		t.Fatal("expected nil runtime handle error")
	}
}

func TestCapabilityDisablesHTTPWhenValuesAreNil(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	if err := capability.InitRuntimeFromSections(context.Background(), nil, testRuntimeInitializerHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.settings.Enabled {
		t.Fatalf("expected nil values to keep HTTP disabled, got %#v", entry.settings)
	}
}

func TestCapabilityEnablesHTTPByDefaultWhenValuesArePresent(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	vals := httpValues(t, nil)
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeInitializerHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if !entry.settings.Enabled || entry.settings.Listen != "127.0.0.1:8787" {
		t.Fatalf("settings = %#v", entry.settings)
	}
}

func TestCapabilityAllowsExplicitHTTPDisable(t *testing.T) {
	capability := newHTTPCapability()
	vm := goja.New()
	vals := httpValues(t, map[string]any{"enabled": false, "listen": "127.0.0.1:9999"})
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeInitializerHandle{vm: vm}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}
	entry := capability.entry(vm)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.settings.Enabled || entry.settings.Listen != "127.0.0.1:9999" {
		t.Fatalf("settings = %#v", entry.settings)
	}
}

func TestExternalHostServiceValidation(t *testing.T) {
	capability := newHTTPCapability()
	if _, err := capability.newExpressLoader(app.HostServices{Services: map[string][]any{HostServiceKey: {"bad"}}}); err == nil || !strings.Contains(err.Error(), "must be ExternalHostService") {
		t.Fatalf("expected wrong type error, got %v", err)
	}
	if _, err := capability.newExpressLoader(app.HostServices{Services: map[string][]any{HostServiceKey: {ExternalHostService{}}}}); err == nil || !strings.Contains(err.Error(), "nil Host") {
		t.Fatalf("expected nil host error, got %v", err)
	}
}

func TestExpressProviderRegistersIntoExternalHost(t *testing.T) {
	jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	runtimeSpec := &app.RuntimeSpec{Modules: []app.ModuleInstanceSpec{{Package: PackageID, Name: "express", As: "express"}}}
	host := app.NewHostWithOptions(registry, runtimeSpec, app.HostOptions{ConfigureServices: func(services *app.HostServices) {
		if err := services.SetHostService(HostServiceKey, ExternalHostService{Host: jsHost, OwnsListen: false}); err != nil {
			t.Fatalf("SetHostService: %v", err)
		}
	}})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	_, err = rt.Owner.Call(context.Background(), "register external route", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`require("express").app().get("/hello/:name", (req, res) => res.json({ hello: req.params.name }))`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("register route: %v", err)
	}
	routes := jsHost.Routes()
	if len(routes) != 1 || routes[0].Method != "GET" || routes[0].Pattern != "/hello/:name" {
		t.Fatalf("external host routes = %#v", routes)
	}

	rr := httptest.NewRecorder()
	jsHost.ServeHTTP(rr, httptest.NewRequest(stdhttp.MethodGet, "/hello/goja", nil))
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"hello":"goja"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestExpressExternalHostDoesNotBindConfiguredHTTPPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve listen address: %v", err)
	}
	defer func() { _ = listener.Close() }()

	capability := newHTTPCapability()
	jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	services := app.HostServices{}
	if err := services.SetHostService(HostServiceKey, ExternalHostService{Host: jsHost, OwnsListen: false}); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	loader, err := capability.newExpressLoader(services)
	if err != nil {
		t.Fatalf("new express loader: %v", err)
	}
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(engine.NativeModuleRegistrar{ModuleName: "express", Loader: loader}).Build()
	if err != nil {
		t.Fatalf("build runtime factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	vals := httpValues(t, map[string]any{"enabled": true, "listen": listener.Addr().String()})
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeInitializerHandle{rt: rt}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}

	_, err = rt.Owner.Call(context.Background(), "register external route", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`require("express").app().get("/external", (_req, res) => res.json({ ok: true }))`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("external route registration should not bind occupied port: %v", err)
	}

	rr := httptest.NewRecorder()
	jsHost.ServeHTTP(rr, httptest.NewRequest(stdhttp.MethodGet, "/external", nil))
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestExpressRequireDoesNotBindHTTPPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve listen address: %v", err)
	}
	defer func() { _ = listener.Close() }()

	capability := newHTTPCapability()
	factory, err := engine.NewRuntimeFactoryBuilder().WithModules(engine.NativeModuleRegistrar{ModuleName: "express", Loader: capability.NewExpressLoader()}).Build()
	if err != nil {
		t.Fatalf("build runtime factory: %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	vals := httpValues(t, map[string]any{"enabled": true, "listen": listener.Addr().String()})
	if err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeInitializerHandle{rt: rt}); err != nil {
		t.Fatalf("init runtime: %v", err)
	}

	_, err = rt.Owner.Call(context.Background(), "require express", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`require("express")`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("require express should not bind occupied port: %v", err)
	}

	_, err = rt.Owner.Call(context.Background(), "register route", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`require("express").app().get("/healthz", (_req, res) => res.json({ ok: true }))`)
		return nil, runErr
	})
	if err == nil {
		t.Fatal("expected route registration to report occupied port")
	}
	if !strings.Contains(err.Error(), "listen on ") {
		t.Fatalf("expected listen error after route registration, got %v", err)
	}
}

func TestCapabilityStartReportsPortConflictsSynchronously(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve listen address: %v", err)
	}
	defer func() { _ = listener.Close() }()

	capability := newHTTPCapability()
	vm := goja.New()
	entry := capability.entry(vm)
	entry.mu.Lock()
	entry.settings = settings{Enabled: true, Listen: listener.Addr().String()}
	entry.mu.Unlock()

	err = capability.start(vm, entry)
	if err == nil {
		t.Fatal("expected port conflict error")
	}
	if !strings.Contains(err.Error(), "listen on ") {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func httpValues(t *testing.T, overrides map[string]any) *values.Values {
	t.Helper()
	capability := newHTTPCapability()
	sections, err := capability.GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	section := sections[0]
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		if definition.Default != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: *definition.Default})
		}
	}
	for name, value := range overrides {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			t.Fatalf("unknown field %q", name)
		}
		fieldValues.Set(name, &fields.FieldValue{Definition: definition, Value: value})
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return values.New(values.WithSectionValues("http", sectionValues))
}

type testRuntimeInitializerHandle struct {
	vm *goja.Runtime
	rt *engine.Runtime
}

func (h testRuntimeInitializerHandle) EngineRuntime() *engine.Runtime {
	if h.rt != nil {
		return h.rt
	}
	return &engine.Runtime{VM: h.vm}
}
func (h testRuntimeInitializerHandle) Close(context.Context) error { return nil }
