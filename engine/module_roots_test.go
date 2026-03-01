package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveModuleRootsFromScript_DefaultOptions(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "js", "extractor", "main.js")

	roots, err := ResolveModuleRootsFromScript(scriptPath, DefaultModuleRootsOptions())
	if err != nil {
		t.Fatalf("resolve module roots: %v", err)
	}

	scriptDir := filepath.Dir(scriptPath)
	parentDir := filepath.Dir(scriptDir)
	expected := []string{
		scriptDir,
		parentDir,
		filepath.Join(scriptDir, "node_modules"),
		filepath.Join(parentDir, "node_modules"),
	}

	if len(roots) != len(expected) {
		t.Fatalf("roots length = %d, want %d (%v)", len(roots), len(expected), roots)
	}
	for i := range expected {
		if roots[i] != expected[i] {
			t.Fatalf("roots[%d] = %q, want %q", i, roots[i], expected[i])
		}
	}
}

func TestResolveModuleRootsFromScript_ExtraFoldersDeduplicated(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "js", "extractor", "main.js")
	scriptDir := filepath.Dir(scriptPath)
	parentDir := filepath.Dir(scriptDir)

	roots, err := ResolveModuleRootsFromScript(scriptPath, ModuleRootsOptions{
		IncludeScriptDir: true,
		IncludeParentDir: true,
		ExtraFolders: []string{
			scriptDir,
			parentDir,
			filepath.Join(tmpDir, "extras"),
		},
	})
	if err != nil {
		t.Fatalf("resolve module roots: %v", err)
	}

	if len(roots) != 3 {
		t.Fatalf("roots length = %d, want 3 (%v)", len(roots), roots)
	}
	if roots[2] != filepath.Join(tmpDir, "extras") {
		t.Fatalf("roots[2] = %q, want %q", roots[2], filepath.Join(tmpDir, "extras"))
	}
}

func TestWithModuleRootsFromScript_ResolvesNestedRequire(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "js")
	scriptDir := filepath.Join(projectDir, "extractor")
	libDir := filepath.Join(projectDir, "lib")

	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("mkdir scriptDir: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir libDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "answer.js"), []byte("module.exports = { value: 42 };"), 0o644); err != nil {
		t.Fatalf("write answer.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "entry.js"), []byte("const a = require('lib/answer.js'); module.exports = { ok: a.value };"), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	factory, err := NewBuilder(
		WithModuleRootsFromScript(filepath.Join(scriptDir, "entry.js"), DefaultModuleRootsOptions()),
	).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	val, err := rt.Require.Require("entry.js")
	if err != nil {
		t.Fatalf("require entry.js: %v", err)
	}
	obj := val.ToObject(rt.VM)
	if got := obj.Get("ok").ToInteger(); got != 42 {
		t.Fatalf("ok = %d, want 42", got)
	}
}
