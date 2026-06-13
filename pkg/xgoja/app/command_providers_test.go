package app

import (
	"context"
	"os"
	"path/filepath"
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

func TestHostAttachCommandProvidersPassesSelectedModules(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.Module{Name: "mod", NewModuleFactory: noopSectionModule},
		providerapi.CommandSetProvider{
			Name:         "tools",
			DefaultMount: "tools",
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
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
	runtimePlan := &RuntimePlan{
		Modules: []RuntimeModulePlan{{Package: "fixture", Name: "mod"}},
		CommandProviders: []CommandProviderInstanceSpec{{
			ID:      "fixture-tools",
			Package: "fixture",
			Name:    "tools",
			Mount:   "fixture",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, runtimePlan).AttachDefaultCommands(root)
	root.SetArgs([]string{"fixture", "ping"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute command provider command: %v", err)
	}
}

func TestJSVerbSourcesOnlyExposeExplicitVerbsButKeepHelpersLoadable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "site.js"), []byte(`
__package__({ name: "site" });
__verb__("start", { name: "start", short: "Start", output: "text" });
function start() { return require("./helper.js").helper(); }
`), 0o644); err != nil {
		t.Fatalf("write site jsverb: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "helper.js"), []byte(`
function helper() { return "helper"; }
function helperThatMustNotBecomeACommand() { return "hidden"; }
module.exports = { helper };
`), 0o644); err != nil {
		t.Fatalf("write helper module: %v", err)
	}

	registry, err := scanVerbSource(providerapi.NewProviderRegistry(), nil, SourcePlan{ID: "local", Path: dir}, nil)
	if err != nil {
		t.Fatalf("scan source: %v", err)
	}
	verbs := registry.Verbs()
	if len(verbs) != 1 || verbs[0].FullPath() != "site start" {
		t.Fatalf("verbs = %#v, want only explicit site start", verbs)
	}
	if _, err := registry.RequireLoader()("/helper.js"); err != nil {
		t.Fatalf("registry loader lost helper module: %v", err)
	}
}

func TestHostAttachCommandProvidersProvidesJSVerbSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tools.js"), []byte(`
__package__({ name: "tools" });
__verb__("hello", { name: "hello", short: "Hello", output: "text" });
function hello() { return "hello"; }
`), 0o644); err != nil {
		t.Fatalf("write jsverb: %v", err)
	}
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture",
		providerapi.CommandSetProvider{
			Name: "tools",
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				if ctx.JSVerbs == nil {
					t.Fatal("expected jsverb source set")
				}
				sources := ctx.JSVerbs.ListJSVerbSources()
				if len(sources) != 1 || sources[0].ID != "local" {
					t.Fatalf("sources = %#v", sources)
				}
				registries, err := ctx.JSVerbs.ScanAllJSVerbSources()
				if err != nil {
					t.Fatalf("scan all jsverb sources: %v", err)
				}
				if len(registries) != 1 || len(registries[0].Verbs()) != 1 {
					t.Fatalf("registries = %#v", registries)
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
	runtimePlan := &RuntimePlan{
		JSVerbs: []SourcePlan{{ID: "local", Path: dir}},
		CommandProviders: []CommandProviderInstanceSpec{{
			ID:      "fixture-tools",
			Package: "fixture",
			Name:    "tools",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, runtimePlan).AttachDefaultCommands(root)
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
	runtimePlan := &RuntimePlan{
		Modules: []RuntimeModulePlan{{Package: "fixture", Name: "mod"}},
		CommandProviders: []CommandProviderInstanceSpec{{
			ID:      "fixture-tools",
			Package: "fixture",
			Name:    "tools",
			Mount:   "fixture",
		}},
	}
	root := &cobra.Command{Use: "test"}
	NewHost(registry, runtimePlan).AttachDefaultCommands(root)
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
