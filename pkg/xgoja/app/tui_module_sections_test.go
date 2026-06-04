package app

import (
	"context"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestTUICommandIncludesRuntimeProfileModuleSections(t *testing.T) {
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runFixtureCapability{}))
	cmd := newTUICommand(factory, factory.runtimeSpec)
	section, ok := cmd.Description().Schema.Get("fixture")
	if !ok {
		t.Fatal("expected fixture section on repl command")
	}
	if section.GetPrefix() != "fixture-" {
		t.Fatalf("fixture prefix = %q", section.GetPrefix())
	}
}

func TestNewXGojaTUIEvaluatorInitializesRuntimeFromModuleSections(t *testing.T) {
	called := false
	factory := newSectionTestFactory(t, providerapi.WithPackageCapability(runtimeInitCapability{
		id: "tui-init",
		fn: func(context.Context, *values.Values, providerapi.RuntimeInitializerHandle) error {
			called = true
			return nil
		},
	}))
	descriptors, err := factory.selectedModuleDescriptors("main")
	if err != nil {
		t.Fatalf("selected descriptors: %v", err)
	}
	adapter, err := newXGojaTUIEvaluator(context.Background(), factory, "main", values.New(), descriptors)
	if err != nil {
		t.Fatalf("new TUI evaluator: %v", err)
	}
	defer func() { _ = adapter.Close() }()
	if !called {
		t.Fatal("expected runtime initializer to be called")
	}
}
