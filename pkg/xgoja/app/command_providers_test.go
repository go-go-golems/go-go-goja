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

func TestHostAttachCommandProvidersMountsGlazedCommand(t *testing.T) {
	called := false
	registry := providerapi.NewRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", New: noopSectionModule},
		providerapi.CommandSetProvider{
			Name:         "tools",
			DefaultMount: "tools",
			New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				if len(ctx.SelectedModules) != 1 {
					t.Fatalf("selected modules = %#v", ctx.SelectedModules)
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
	spec := &Spec{
		Runtimes: map[string]Runtime{"main": {Modules: []ModuleInstance{{Package: "fixture", Name: "mod"}}}},
		CommandProviders: []CommandProviderInstance{{
			ID:             "fixture-tools",
			Package:        "fixture",
			Name:           "tools",
			Mount:          "fixture",
			RuntimeProfile: "main",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, spec).AttachDefaultCommands(root)
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
