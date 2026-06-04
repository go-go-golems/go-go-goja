package app

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRuntimeFactoryCollectsHostServiceContributionsBeforeModuleSetup(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	capability := &hostServiceCapability{key: "demo", value: "from-capability"}
	seen := ""
	if err := registry.Package("fixture",
		providerapi.Module{
			Name:      "mod",
			DefaultAs: "mod",
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				lookup, ok := ctx.Host.(providerapi.HostServiceLookup)
				if !ok {
					t.Fatalf("host does not implement HostServiceLookup")
				}
				value, ok := lookup.HostService("demo")
				if !ok {
					t.Fatalf("missing demo host service")
				}
				seen, _ = value.(string)
				return func(*goja.Runtime, *goja.Object) {}, nil
			},
		},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	runtimeSpec := &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}}}}}
	factory := NewRuntimeFactory(registry, runtimeSpec, HostServices{})
	rt, err := factory.NewRuntimeFromSections(context.Background(), "main", values.New())
	if err != nil {
		t.Fatalf("NewRuntimeFromSections: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if seen != "from-capability" {
		t.Fatalf("seen host service = %q", seen)
	}
	if capability.calls != 1 {
		t.Fatalf("capability calls = %d, want 1", capability.calls)
	}
}

func TestHostServiceContributionsDedupeSamePackageCapability(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	capability := &hostServiceCapability{key: "demo", value: "once"}
	module := func(name string) providerapi.Module {
		return providerapi.Module{Name: name, DefaultAs: name, NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(*goja.Runtime, *goja.Object) {}, nil
		}}
	}
	if err := registry.Package("fixture",
		module("first"),
		module("second"),
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	runtimeSpec := &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "first"}, {Package: "fixture", Name: "second"}}}}}
	factory := NewRuntimeFactory(registry, runtimeSpec, HostServices{})
	rt, err := factory.NewRuntimeFromSections(context.Background(), "main", values.New())
	if err != nil {
		t.Fatalf("NewRuntimeFromSections: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if capability.calls != 1 {
		t.Fatalf("capability calls = %d, want 1", capability.calls)
	}
}

func TestHostServiceContributionClosersRunOnRuntimeClose(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	closed := 0
	capability := &hostServiceCapability{key: "demo", value: "from-capability", closer: func(context.Context) error {
		closed++
		return nil
	}}
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(*goja.Runtime, *goja.Object) {}, nil
		}},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	runtimeSpec := &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}}}}}
	factory := NewRuntimeFactory(registry, runtimeSpec, HostServices{})
	rt, err := factory.NewRuntimeFromSections(context.Background(), "main", values.New())
	if err != nil {
		t.Fatalf("NewRuntimeFromSections: %v", err)
	}
	if closed != 0 {
		t.Fatalf("closer ran before runtime close")
	}
	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if closed != 1 {
		t.Fatalf("closed = %d, want 1", closed)
	}
}

func TestHostServiceContributionClosersRunOnRuntimeSetupFailure(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	closed := 0
	capability := &hostServiceCapability{key: "demo", value: "from-capability", closer: func(context.Context) error {
		closed++
		return nil
	}}
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return nil, assertErr("setup failed")
		}},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	runtimeSpec := &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}}}}}
	factory := NewRuntimeFactory(registry, runtimeSpec, HostServices{})
	_, err := factory.NewRuntimeFromSections(context.Background(), "main", values.New())
	if err == nil || !strings.Contains(err.Error(), "setup failed") {
		t.Fatalf("expected setup failure, got %v", err)
	}
	if closed != 1 {
		t.Fatalf("closed = %d, want 1", closed)
	}
}

func TestHostServiceContributionErrorsAreWrapped(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	capability := &hostServiceCapability{err: assertErr("boom")}
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(*goja.Runtime, *goja.Object) {}, nil
		}},
		providerapi.WithPackageCapability(capability),
	); err != nil {
		t.Fatalf("register package: %v", err)
	}
	runtimeSpec := &RuntimeSpec{Runtimes: map[string]RuntimeProfileSpec{"main": {Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}}}}}
	factory := NewRuntimeFactory(registry, runtimeSpec, HostServices{})
	_, err := factory.NewRuntimeFromSections(context.Background(), "main", values.New())
	if err == nil || !strings.Contains(err.Error(), "contribute host services for fixture capability host-service") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestHostServicesReturnsMultipleValues(t *testing.T) {
	host := HostServices{Services: map[string][]any{"demo": {"a", "b"}}}
	values := host.HostServiceValues("demo")
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("HostServiceValues = %#v", values)
	}
	value, ok := host.HostService("demo")
	if !ok {
		t.Fatalf("HostService missing")
	}
	multi, ok := value.([]any)
	if !ok || len(multi) != 2 {
		t.Fatalf("HostService = %#v", value)
	}
}

type hostServiceCapability struct {
	key    string
	value  any
	calls  int
	err    error
	closer func(context.Context) error
}

func (c *hostServiceCapability) CapabilityID() string { return "host-service" }
func (c *hostServiceCapability) ContributeHostServices(_ context.Context, req providerapi.HostServiceContributionRequest, sink providerapi.HostServiceSink) error {
	c.calls++
	if req.RuntimeProfile != "main" {
		return assertErr("unexpected profile")
	}
	if c.err != nil {
		return c.err
	}
	if err := sink.AddHostService(c.key, c.value); err != nil {
		return err
	}
	if c.closer != nil {
		return sink.AddCloser(c.closer)
	}
	return nil
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
