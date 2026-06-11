package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRunScriptFileWithInitializersRunsTypeScriptEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.ts"), []byte(`export const name: string = "goja"`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.ts")
	if err := os.WriteFile(entry, []byte(`
		import { name } from "./helper"
		globalThis.result = "hello " + name
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	factory := NewRuntimeFactory(providerapi.NewProviderRegistry(), &RuntimeSpec{})
	if err := runScriptFileWithInitializers(context.Background(), factory, entry, nil, nil, false); err != nil {
		t.Fatalf("runScriptFileWithInitializers() error = %v", err)
	}
}

func TestModuleAliasesDeduplicatesSelectedModuleAliases(t *testing.T) {
	aliases := moduleAliases([]providerapi.ModuleDescriptor{
		{ModuleID: "fs", As: "fs:assets"},
		{ModuleID: "fs", As: "fs:assets"},
		{ModuleID: "path"},
	})
	if got, want := aliases, []string{"fs:assets", "path"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("moduleAliases() = %#v, want %#v", got, want)
	}
}
