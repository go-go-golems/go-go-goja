package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

	registry, err := scanVerbSource(providerapi.NewProviderRegistry(), nil, JSVerbSourceSpec{
		ID:         "local",
		Path:       dir,
		Extensions: []string{".ts"},
		TypeScript: &TypeScriptSpec{Enabled: true, Bundle: true, Target: "es2015", Format: "cjs", Platform: "neutral"},
	})
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
