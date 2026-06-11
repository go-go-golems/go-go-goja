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
go:
  imports:
    - import: github.com/lib/pq
      alias: _
      version: v1.10.9
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
modules:
  - package: core
    name: fs
  - package: core
    name: yaml
    as: yml
commands:
  eval:
    enabled: true
help:
  sources:
    - id: provider-docs
      package: core
      source: docs
jsverbs:
  - id: local
    path: ./verbs
    embed: true
    extensions: [.ts]
    typescript:
      enabled: true
      bundle: true
assets:
  - id: app-assets
    path: ./assets
    embed: true
    description: test assets
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}

	buildSpec, report, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("load valid build spec: %v", err)
	}
	if report == nil || report.HasErrors() {
		t.Fatalf("expected non-error report, got %#v", report)
	}
	if buildSpec.Name != "webrepl" {
		t.Fatalf("name = %q", buildSpec.Name)
	}
	if buildSpec.AppName != "webrepl-dev" {
		t.Fatalf("appName = %q", buildSpec.AppName)
	}
	if buildSpec.EnvPrefix != "WEBREPL_DEV" {
		t.Fatalf("envPrefix = %q", buildSpec.EnvPrefix)
	}
	if len(buildSpec.Go.Imports) != 1 || buildSpec.Go.Imports[0].Import != "github.com/lib/pq" || buildSpec.Go.Imports[0].Alias != "_" || buildSpec.Go.Imports[0].Version != "v1.10.9" {
		t.Fatalf("go imports = %#v", buildSpec.Go.Imports)
	}
	if buildSpec.JSVerbs[0].TypeScript == nil || !buildSpec.JSVerbs[0].TypeScript.Enabled || buildSpec.JSVerbs[0].TypeScript.Target != "es2015" || buildSpec.JSVerbs[0].TypeScript.Format != "cjs" || buildSpec.JSVerbs[0].TypeScript.Platform != "neutral" {
		t.Fatalf("typescript defaults = %#v", buildSpec.JSVerbs[0].TypeScript)
	}
	if buildSpec.Target.Kind != "xgoja" {
		t.Fatalf("default target kind = %q", buildSpec.Target.Kind)
	}
	if buildSpec.Commands.Eval.Name != "eval" {
		t.Fatalf("default eval command name = %q", buildSpec.Commands.Eval.Name)
	}
	if buildSpec.Modules[1].Alias() != "yml" {
		t.Fatalf("module alias = %q", buildSpec.Modules[1].Alias())
	}
	if len(buildSpec.Help.Sources) != 1 || buildSpec.Help.Sources[0].Source != "docs" {
		t.Fatalf("help sources = %#v", buildSpec.Help.Sources)
	}
	if len(buildSpec.Assets) != 1 || buildSpec.Assets[0].ID != "app-assets" || buildSpec.Assets[0].Description != "test assets" {
		t.Fatalf("assets = %#v", buildSpec.Assets)
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
		t.Fatalf("write build spec: %v", err)
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
modules:
  - package: core
    name: fs
    as: same
  - package: core
    name: yaml
    as: same
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
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
modules:
  - package: core
    name: fs
jsverbs:
  - id: missing
    path: ./missing
    embed: true
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}

	_, _, err := LoadFile(specPath)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing path error, got %v", err)
	}
}

func TestLoadFileDefaultsGeneratedModulePath(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: My Fixture_App.v2
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
modules:
  - package: core
    name: fs
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}

	buildSpec, report, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("load build spec: %v", err)
	}
	if report == nil || report.HasErrors() {
		t.Fatalf("expected non-error report, got %#v", report)
	}
	if buildSpec.Go.Module != "xgoja.generated/my-fixture-app-v2" {
		t.Fatalf("go.module = %q, want xgoja.generated/my-fixture-app-v2", buildSpec.Go.Module)
	}
}

func TestLoadFilePreservesExplicitModulePath(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: checked-in-host
go:
  module: github.com/acme/project/cmd/checked-in-host
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
modules:
  - package: core
    name: fs
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}

	buildSpec, report, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("load build spec: %v", err)
	}
	if report == nil || report.HasErrors() {
		t.Fatalf("expected non-error report, got %#v", report)
	}
	if buildSpec.Go.Module != "github.com/acme/project/cmd/checked-in-host" {
		t.Fatalf("go.module = %q, want explicit module path", buildSpec.Go.Module)
	}
}

func TestLoadFileAppliesConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: config-test
appName: config-test
configFile:
  enabled: true
  layers:
    - cwd
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
modules:
  - package: core
    name: fs
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}

	buildSpec, report, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("load build spec: %v", err)
	}
	if report == nil || report.HasErrors() {
		t.Fatalf("expected non-error report, got %#v", report)
	}
	if buildSpec.ConfigFile == nil || !buildSpec.ConfigFile.Enabled {
		t.Fatalf("config not enabled")
	}
	if buildSpec.ConfigFile.FileName != "config.yaml" {
		t.Fatalf("configFile.FileName = %q, want config.yaml", buildSpec.ConfigFile.FileName)
	}
}
