package buildspec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileValidSpec(t *testing.T) {
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.Mkdir(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	assetsDir := filepath.Join(dir, "assets")
	if err := os.Mkdir(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: webrepl
appName: webrepl-dev
envPrefix: WEBREPL_DEV
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
runtimes:
  repl:
    modules:
      - package: core
        name: fs
      - package: core
        name: yaml
        as: yml
commands:
  eval:
    enabled: true
    runtime: repl
help:
  sources:
    - id: provider-docs
      package: core
      source: docs
jsverbs:
  - id: local
    path: ./verbs
    embed: true
assets:
  - id: app-assets
    path: ./assets
    embed: true
    description: test assets
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	spec, report, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("load valid spec: %v", err)
	}
	if report == nil || report.HasErrors() {
		t.Fatalf("expected non-error report, got %#v", report)
	}
	if spec.Name != "webrepl" {
		t.Fatalf("name = %q", spec.Name)
	}
	if spec.AppName != "webrepl-dev" {
		t.Fatalf("appName = %q", spec.AppName)
	}
	if spec.EnvPrefix != "WEBREPL_DEV" {
		t.Fatalf("envPrefix = %q", spec.EnvPrefix)
	}
	if spec.Target.Kind != "xgoja" {
		t.Fatalf("default target kind = %q", spec.Target.Kind)
	}
	if spec.Commands.Eval.Name != "eval" {
		t.Fatalf("default eval command name = %q", spec.Commands.Eval.Name)
	}
	if spec.Runtimes["repl"].Modules[1].Alias() != "yml" {
		t.Fatalf("module alias = %q", spec.Runtimes["repl"].Modules[1].Alias())
	}
	if len(spec.Help.Sources) != 1 || spec.Help.Sources[0].Source != "docs" {
		t.Fatalf("help sources = %#v", spec.Help.Sources)
	}
	if len(spec.Assets) != 1 || spec.Assets[0].ID != "app-assets" || spec.Assets[0].Description != "test assets" {
		t.Fatalf("assets = %#v", spec.Assets)
	}
}

func TestLoadFileRejectsUnsupportedAssetFilters(t *testing.T) {
	dir := t.TempDir()
	assetsDir := filepath.Join(dir, "assets")
	if err := os.Mkdir(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: bad-assets
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
runtimes:
  main:
    modules:
      - package: core
        name: fs
assets:
  - id: app-assets
    path: ./assets
    embed: true
    include:
      - public/**
    exclude:
      - secrets/**
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	_, report, err := LoadFile(specPath)
	if err == nil {
		t.Fatal("expected unsupported asset filter validation error")
	}
	if report == nil || !report.HasErrors() {
		t.Fatalf("expected error report, got %#v", report)
	}
	if !strings.Contains(err.Error(), "assets[0].include") || !strings.Contains(err.Error(), "assets[0].exclude") {
		t.Fatalf("expected include/exclude errors, got %v", err)
	}
}

func TestLoadFileDuplicateAlias(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: bad
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
runtimes:
  main:
    modules:
      - package: core
        name: fs
        as: same
      - package: core
        name: yaml
        as: same
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	_, report, err := LoadFile(specPath)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if report == nil || !report.HasErrors() {
		t.Fatalf("expected error report, got %#v", report)
	}
	if !strings.Contains(err.Error(), "duplicate alias") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestLoadFileMissingEmbeddedVerbPath(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: bad
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
runtimes:
  main:
    modules:
      - package: core
        name: fs
jsverbs:
  - id: missing
    path: ./missing
    embed: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	_, _, err := LoadFile(specPath)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing path error, got %v", err)
	}
}
