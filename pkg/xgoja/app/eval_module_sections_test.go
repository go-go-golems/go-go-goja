package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestEvalCommandIncludesRuntimeProfileModuleSections(t *testing.T) {
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runFixtureCapability{}))
	cmd := newEvalCommand(factory, factory.runtimeSpec, nil)
	section, ok := cmd.Description().Schema.Get("fixture")
	if !ok {
		t.Fatal("expected fixture section on eval command")
	}
	if section.GetPrefix() != "fixture-" {
		t.Fatalf("fixture prefix = %q", section.GetPrefix())
	}
}

func TestEvalCommandInitializesRuntimeFromModuleSections(t *testing.T) {
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runFixtureCapability{}))
	out := &bytes.Buffer{}
	cmd, err := buildGlazedCobraCommand(newEvalCommand(factory, factory.runtimeSpec, out))
	if err != nil {
		t.Fatalf("build cobra command: %v", err)
	}
	cmd.SetArgs([]string{`globalThis.fixtureValue`, "--fixture-value", "ok"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("eval command: %v", err)
	}
	if got := out.String(); got != "ok\n" {
		t.Fatalf("eval output = %q", got)
	}
}

func TestEvalCommandRuntimeOverrideInitializesSelectedRuntimeProfile(t *testing.T) {
	factory := newRuntimeOverrideFactory(t)
	out := &bytes.Buffer{}
	cmd, err := buildGlazedCobraCommand(newEvalCommand(factory, factory.runtimeSpec, out))
	if err != nil {
		t.Fatalf("build cobra command: %v", err)
	}
	cmd.SetArgs([]string{`globalThis.fixtureValue`, "--runtime", "override", "--fixture-value", "ok"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("eval command: %v", err)
	}
	if got := out.String(); got != "override:ok\n" {
		t.Fatalf("eval output = %q", got)
	}
}
