package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider"
)

func TestDefaultEnvPrefixNormalizesAppName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "hyphen", in: "my-app", want: "MY_APP"},
		{name: "dots spaces underscores", in: " my.app_name dev ", want: "MY_APP_NAME_DEV"},
		{name: "repeated separators", in: "my---app", want: "MY_APP"},
		{name: "leading digit", in: "123-app", want: "APP_123_APP"},
		{name: "empty", in: " -- ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultEnvPrefix(tt.in); got != tt.want {
				t.Fatalf("DefaultEnvPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGeneratedRootReadsModuleSectionFromDerivedEnvPrefix(t *testing.T) {
	t.Setenv("ENV_FIXTURE_FIXTURE_VALUE", "from-env")
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "appName": "env-fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "from-env\n" {
		t.Fatalf("eval output = %q, want from-env", got)
	}
}

func TestGeneratedRootReadsModuleSectionFromExplicitEnvPrefix(t *testing.T) {
	t.Setenv("XGOJA_TEST_FIXTURE_VALUE", "explicit-env")
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "envPrefix": "XGOJA_TEST",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "explicit-env\n" {
		t.Fatalf("eval output = %q, want explicit-env", got)
	}
}

func TestGeneratedRootKeepsDefaultMiddlewaresWithoutAppSettings(t *testing.T) {
	t.Setenv("FIXTURE_FIXTURE_VALUE", "ignored")
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: fixtureSpecJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "fixtureValue === ''"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "true\n" {
		t.Fatalf("eval output = %q, want true", got)
	}
}

func TestGeneratedRootReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("fixture:\n  value: from-config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	chdirForTest(t, dir)

	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "appName": "config-fixture",
  "configFile": {"enabled": true, "layers": ["cwd"], "fileName": "config.yaml"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "from-config\n" {
		t.Fatalf("eval output = %q, want from-config", got)
	}
}

func TestConfigPrecedenceEnvBeatsConfig(t *testing.T) {
	t.Setenv("CONFIG_FIXTURE_FIXTURE_VALUE", "from-env")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("fixture:\n  value: from-config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	chdirForTest(t, dir)

	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "appName": "config-fixture",
  "envPrefix": "CONFIG_FIXTURE",
  "configFile": {"enabled": true, "layers": ["cwd"], "fileName": "config.yaml"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "from-env\n" {
		t.Fatalf("eval output = %q, want from-env", got)
	}
}

func TestConfigPrecedenceFlagBeatsEnv(t *testing.T) {
	t.Setenv("CONFIG_FIXTURE_FIXTURE_VALUE", "from-env")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("fixture:\n  value: from-config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	chdirForTest(t, dir)

	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "appName": "config-fixture",
  "envPrefix": "CONFIG_FIXTURE",
  "configFile": {"enabled": true, "layers": ["cwd"], "fileName": "config.yaml"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "--fixture-value", "from-flag", "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "from-flag\n" {
		t.Fatalf("eval output = %q, want from-flag", got)
	}
}

func TestExplicitConfigFileRequiresExplicitLayer(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "explicit.yaml")
	if err := os.WriteFile(configPath, []byte("fixture:\n  value: from-explicit\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "appName": "config-fixture",
  "configFile": {"enabled": true, "layers": ["cwd"], "fileName": "config.yaml"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "--config-file", configPath, "fixtureValue === ''"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "true\n" {
		t.Fatalf("eval output = %q, want true", got)
	}
}

func TestExplicitConfigFileLoadsWithExplicitLayer(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "explicit.yaml")
	if err := os.WriteFile(configPath, []byte("fixture:\n  value: from-explicit\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "configFile": {"enabled": true, "layers": ["explicit"], "fileName": "config.yaml"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "modules": [{"package": "fixture", "name": "hello", "as": "hello"}],
  "commands": {
    "eval": {"enabled": true, "name": "eval"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"eval", "--config-file", configPath, "fixtureValue"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "from-explicit\n" {
		t.Fatalf("eval output = %q, want from-explicit", got)
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("restore wd: %v", err)
		}
	})
}
