package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRunCommandIncludesRuntimeProfileModuleSections(t *testing.T) {
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runFixtureCapability{}))
	cmd := newRunCommand(factory, factory.runtimeSpec)
	section, ok := cmd.Description().Schema.Get("fixture")
	if !ok {
		t.Fatal("expected fixture section on run command")
	}
	if section.GetPrefix() != "fixture-" {
		t.Fatalf("fixture prefix = %q", section.GetPrefix())
	}
}

func TestRunCommandInitializesRuntimeFromModuleSections(t *testing.T) {
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runFixtureCapability{}))
	cmd, err := buildGlazedCobraCommand(newRunCommand(factory, factory.runtimeSpec))
	if err != nil {
		t.Fatalf("build cobra command: %v", err)
	}
	script := filepath.Join(t.TempDir(), "check.js")
	if err := os.WriteFile(script, []byte(`if (globalThis.fixtureValue !== "ok") { throw new Error("fixtureValue=" + globalThis.fixtureValue); }`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	cmd.SetArgs([]string{script, "--fixture-value", "ok"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("run command: %v", err)
	}
}

func TestRunCommandRuntimeOverrideInitializesSelectedRuntimeProfile(t *testing.T) {
	factory := newRuntimeOverrideFactory(t)
	cmd, err := buildGlazedCobraCommand(newRunCommand(factory, factory.runtimeSpec))
	if err != nil {
		t.Fatalf("build cobra command: %v", err)
	}
	script := filepath.Join(t.TempDir(), "check.js")
	if err := os.WriteFile(script, []byte(`if (globalThis.fixtureValue !== "override:ok") { throw new Error("fixtureValue=" + globalThis.fixtureValue); }`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	cmd.SetArgs([]string{script, "--runtime", "override", "--fixture-value", "ok"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("run command: %v", err)
	}
}

type runFixtureSettings struct {
	Value string `glazed:"value"`
}

func newRuntimeOverrideFactory(t *testing.T) *RuntimeFactory {
	t.Helper()
	registry := providerapi.NewRegistry()
	if err := registry.Package("defaultpkg",
		providerapi.Module{Name: "mod", NewModuleFactory: noopSectionModule},
		providerapi.WithPackageCapability(prefixedRunFixtureCapability{prefix: "default:"}),
	); err != nil {
		t.Fatalf("register default provider: %v", err)
	}
	if err := registry.Package("overridepkg",
		providerapi.Module{Name: "mod", NewModuleFactory: noopSectionModule},
		providerapi.WithPackageCapability(prefixedRunFixtureCapability{prefix: "override:"}),
	); err != nil {
		t.Fatalf("register override provider: %v", err)
	}
	return NewRuntimeFactory(registry, &RuntimeSpec{
		Commands: CommandsSpec{
			Eval: CommandSpec{Runtime: "default"},
			Run:  CommandSpec{Runtime: "default"},
		},
		Runtimes: map[string]RuntimeProfileSpec{
			"default":  {Modules: []ModuleInstanceSpec{{Package: "defaultpkg", Name: "mod", As: "fixture"}}},
			"override": {Modules: []ModuleInstanceSpec{{Package: "overridepkg", Name: "mod", As: "fixture"}}},
		},
	})
}

type runFixtureCapability struct{}

func (runFixtureCapability) CapabilityID() string { return "fixture.run" }

func (runFixtureCapability) ConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := schema.NewSection(
		"fixture",
		"Fixture",
		schema.WithPrefix("fixture-"),
		schema.WithFields(fields.New("value", fields.TypeString)),
	)
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (runFixtureCapability) InitRuntimeFromSections(_ context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
	return setFixtureValue(vals, handle, "")
}

type prefixedRunFixtureCapability struct{ prefix string }

func (c prefixedRunFixtureCapability) CapabilityID() string { return "fixture.run" }

func (c prefixedRunFixtureCapability) ConfigSections(ctx providerapi.SectionRequest) ([]schema.Section, error) {
	return (runFixtureCapability{}).ConfigSections(ctx)
}

func (c prefixedRunFixtureCapability) InitRuntimeFromSections(_ context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
	return setFixtureValue(vals, handle, c.prefix)
}

func setFixtureValue(vals *values.Values, handle providerapi.RuntimeInitializerHandle, prefix string) error {
	var settings runFixtureSettings
	if err := vals.DecodeSectionInto("fixture", &settings); err != nil {
		return err
	}
	runtime := handle.Runtime()
	if runtime == nil {
		runtime = goja.New()
	}
	if err := runtime.Set("fixtureValue", prefix+settings.Value); err != nil {
		return err
	}
	return nil
}
