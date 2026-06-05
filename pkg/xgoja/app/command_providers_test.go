package app

import (
	"context"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

func TestApplyMountToCommandsDoesNotMutateProviderDescriptions(t *testing.T) {
	original := &fixtureBareCommand{
		CommandDescription: cmds.NewCommandDescription("ping", cmds.WithShort("Ping")),
		run:                func(context.Context, *values.Values) error { return nil },
	}

	mounted := applyMountToCommands([]cmds.Command{original}, "fixture")
	if len(original.Description().Parents) != 0 {
		t.Fatalf("original parents mutated: %#v", original.Description().Parents)
	}
	if len(mounted) != 1 {
		t.Fatalf("mounted commands = %d", len(mounted))
	}
	if mounted[0] == original {
		t.Fatal("expected mounted wrapper, got original command")
	}
	if got := mounted[0].Description().Parents; len(got) != 1 || got[0] != "fixture" {
		t.Fatalf("mounted parents = %#v", got)
	}
	if _, ok := mounted[0].(cmds.BareCommand); !ok {
		t.Fatalf("mounted command lost BareCommand interface: %T", mounted[0])
	}
}

func TestHostAttachCommandProvidersUsesDefaultRuntimeProfileContext(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: noopSectionModule},
		providerapi.CommandSetProvider{
			Name:         "tools",
			DefaultMount: "tools",
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				if ctx.RuntimeProfile != defaultRuntimeProfile {
					t.Fatalf("runtime profile = %q", ctx.RuntimeProfile)
				}
				if len(ctx.SelectedModules) != 1 || ctx.SelectedModules[0].ModuleID != "mod" {
					t.Fatalf("selected modules = %#v", ctx.SelectedModules)
				}
				runtime, err := ctx.RuntimeFactory.NewRuntime(ctx.Context)
				if err != nil {
					t.Fatalf("new runtime from defaulted command provider profile: %v", err)
				}
				if runtime == nil || runtime.VM == nil {
					t.Fatalf("runtime = %#v", runtime)
				}
				if err := runtime.Close(ctx.Context); err != nil {
					t.Fatalf("close runtime: %v", err)
				}
				return &providerapi.CommandSet{Commands: []cmds.Command{&fixtureBareCommand{
					CommandDescription: cmds.NewCommandDescription("ping", cmds.WithShort("Ping")),
					run:                func(context.Context, *values.Values) error { return nil },
				}}}, nil
			},
		},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	runtimeSpec := &RuntimeSpec{
		Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}},
		CommandProviders: []CommandProviderInstanceSpec{{
			ID:      "fixture-tools",
			Package: "fixture",
			Name:    "tools",
			Mount:   "fixture",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, runtimeSpec).AttachDefaultCommands(root)
	root.SetArgs([]string{"fixture", "ping"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute command provider command: %v", err)
	}
}

func TestHostAttachCommandProvidersMountsGlazedCommand(t *testing.T) {
	called := false
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: noopSectionModule},
		providerapi.CommandSetProvider{
			Name:         "tools",
			DefaultMount: "tools",
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				if len(ctx.SelectedModules) != 1 {
					t.Fatalf("selected modules = %#v", ctx.SelectedModules)
				}
				if ctx.RuntimeFactory == nil {
					t.Fatal("expected typed runtime factory")
				}
				runtime, err := ctx.RuntimeFactory.NewRuntime(ctx.Context)
				if err != nil {
					t.Fatalf("new runtime from typed runtime factory: %v", err)
				}
				if runtime == nil || runtime.VM == nil {
					t.Fatalf("runtime = %#v", runtime)
				}
				if err := runtime.Close(ctx.Context); err != nil {
					t.Fatalf("close runtime: %v", err)
				}
				return &providerapi.CommandSet{Commands: []cmds.Command{&fixtureBareCommand{
					CommandDescription: cmds.NewCommandDescription(
						"ping",
						cmds.WithShort("Ping fixture command provider"),
						cmds.WithFlags(fields.New("message", fields.TypeString)),
					),
					run: func(context.Context, *values.Values) error {
						called = true
						return nil
					},
				}}}, nil
			},
		},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	runtimeSpec := &RuntimeSpec{
		Modules: []ModuleInstanceSpec{{Package: "fixture", Name: "mod"}},
		CommandProviders: []CommandProviderInstanceSpec{{
			ID:      "fixture-tools",
			Package: "fixture",
			Name:    "tools",
			Mount:   "fixture",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, runtimeSpec).AttachDefaultCommands(root)
	root.SetArgs([]string{"fixture", "ping", "--message", "hello"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute command provider command: %v", err)
	}
	if !called {
		t.Fatal("expected fixture command provider command to run")
	}
}

type fixtureBareCommand struct {
	*cmds.CommandDescription
	run func(context.Context, *values.Values) error
}

var _ cmds.BareCommand = (*fixtureBareCommand)(nil)

func (c *fixtureBareCommand) Run(ctx context.Context, vals *values.Values) error {
	return c.run(ctx, vals)
}
