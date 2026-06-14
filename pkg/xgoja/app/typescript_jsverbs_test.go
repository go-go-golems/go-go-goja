package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestScanVerbSourceTypeScriptScansAndInvokesBundledVerb(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.ts"), []byte(`
		export function message(name: string): string { return "hello " + name }
	`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sites.ts"), []byte(`
		import { message } from "./helper"
		__package__({ name: "sites" })
		__verb__("demo", { name: "demo", output: "text" })
		function demo(): string { return message("goja") }
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := scanVerbSource(providerapi.NewProviderRegistry(), nil, SourcePlan{
		ID:         "local",
		Path:       dir,
		Extensions: []string{".ts"},
		TypeScript: &TypeScriptPlan{Enabled: true, Bundle: true, Target: "es2015", Format: "cjs", Platform: "neutral"},
	}, nil)
	if err != nil {
		t.Fatalf("scanVerbSource() error = %v", err)
	}
	verb, ok := registry.Verb("sites demo")
	if !ok {
		t.Fatalf("expected sites demo verb, got %#v", registry.Verbs())
	}

	factory, err := engine.NewRuntimeFactoryBuilder().WithRequireOptions(noderequire.WithLoader(registry.RequireLoader())).Build()
	if err != nil {
		t.Fatalf("build runtime factory: %v", err)
	}
	runtime, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = runtime.Close(context.Background()) }()

	got, err := registry.InvokeInRuntime(context.Background(), runtime, verb, values.New())
	if err != nil {
		t.Fatalf("InvokeInRuntime() error = %v", err)
	}
	if got != "hello goja" {
		t.Fatalf("InvokeInRuntime() = %#v, want hello goja", got)
	}
}

func TestScanVerbSourceTypeScriptUsesTypeScriptExtensionsByDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sites.ts"), []byte(`
		__package__({ name: "sites" })
		__verb__("demo", { name: "demo", output: "text" })
		function demo(): string { return "hello default ts extensions" }
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := scanVerbSource(providerapi.NewProviderRegistry(), nil, SourcePlan{
		ID:         "local",
		Path:       dir,
		TypeScript: &TypeScriptPlan{Enabled: true, Bundle: true, Target: "es2015", Format: "cjs", Platform: "neutral"},
	}, nil)
	if err != nil {
		t.Fatalf("scanVerbSource() error = %v", err)
	}
	if _, ok := registry.Verb("sites demo"); !ok {
		t.Fatalf("expected sites demo verb without explicit extensions, got %#v", registry.Verbs())
	}
}

func TestSourceGraphRuntimeAliasesUseOnlySelectedRuntimeAliases(t *testing.T) {
	providers := providerapi.NewProviderRegistry()
	if err := providers.Package("fixture", providerapi.Module{Name: "secret", DefaultAs: "secret", NewModuleFactory: noopAppModuleFactory}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	selected := sourceGraphRuntimeAliases([]string{"allowed"})
	if !containsString(selected, "allowed") {
		t.Fatalf("selected runtime aliases = %#v, want allowed", selected)
	}
	if containsString(selected, "secret") {
		t.Fatalf("selected runtime aliases leaked unselected provider module: %#v", selected)
	}

	all := allProviderRuntimeAliases(providers)
	if !containsString(all, "secret") {
		t.Fatalf("all provider runtime aliases = %#v, want secret", all)
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func TestScanVerbSourceTypeScriptProviderFSBundlesHelperImport(t *testing.T) {
	providers := providerapi.NewProviderRegistry()
	if err := providers.Package("fixture", providerapi.VerbSource{
		Name: "sites",
		FS: fstest.MapFS{
			"verbs/helper.ts": {Data: []byte(`export function message(name: string): string { return "hello " + name }`)},
			"verbs/sites.ts": {Data: []byte(`
				import { message } from "./helper"
				__package__({ name: "sites" })
				__verb__("demo", { name: "demo", output: "text" })
				function demo(): string { return message("provider") }
			`)},
		},
		Root: "verbs",
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	registry, err := scanVerbSource(providers, nil, SourcePlan{
		ID:         "provider-sites",
		Provider:   "fixture",
		Source:     "sites",
		Extensions: []string{".ts"},
		TypeScript: &TypeScriptPlan{Enabled: true, Bundle: true, Target: "es2015", Format: "cjs", Platform: "neutral"},
	}, nil)
	if err != nil {
		t.Fatalf("scanVerbSource() error = %v", err)
	}
	verb, ok := registry.Verb("sites demo")
	if !ok {
		t.Fatalf("expected sites demo verb, got %#v", registry.Verbs())
	}

	factory, err := engine.NewRuntimeFactoryBuilder().WithRequireOptions(noderequire.WithLoader(registry.RequireLoader())).Build()
	if err != nil {
		t.Fatalf("build runtime factory: %v", err)
	}
	runtime, err := factory.NewRuntime(engine.WithStartupContext(context.Background()), engine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = runtime.Close(context.Background()) }()

	got, err := registry.InvokeInRuntime(context.Background(), runtime, verb, values.New())
	if err != nil {
		t.Fatalf("InvokeInRuntime() error = %v", err)
	}
	if got != "hello provider" {
		t.Fatalf("InvokeInRuntime() = %#v, want hello provider", got)
	}
}
