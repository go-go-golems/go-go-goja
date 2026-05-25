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
	cmd := newRunCommand(factory, factory.spec)
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
	cmd, err := buildGlazedCobraCommand(newRunCommand(factory, factory.spec))
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

type runFixtureSettings struct {
	Value string `glazed:"value"`
}

type runFixtureCapability struct{}

func (runFixtureCapability) CapabilityID() string { return "fixture.run" }

func (runFixtureCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
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

func (runFixtureCapability) InitRuntimeFromSections(_ context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
	var settings runFixtureSettings
	if err := vals.DecodeSectionInto("fixture", &settings); err != nil {
		return err
	}
	runtime := handle.Runtime()
	if runtime == nil {
		runtime = goja.New()
	}
	if err := runtime.Set("fixtureValue", settings.Value); err != nil {
		return err
	}
	return nil
}
