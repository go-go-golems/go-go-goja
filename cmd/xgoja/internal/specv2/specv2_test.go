package specv2

import (
	"strings"
	"testing"
)

func TestLoadDataValidSimplifiedTypeScriptJSVerbs(t *testing.T) {
	cfg, err := LoadData([]byte(`schema: xgoja/v2
name: typescript-jsverbs
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: http
      name: express
      as: express
sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - "**/*.ts"
    language: typescript
    compile:
      mode: runtime
      bundle: true
commands:
  - id: run
    type: builtin.run
  - id: verbs
    type: builtin.jsverbs
    sources: [sites]
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs
  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true
`))
	if err != nil {
		t.Fatalf("LoadData returned error: %v", err)
	}
	if cfg.Schema != Schema {
		t.Fatalf("schema = %q, want %q", cfg.Schema, Schema)
	}
	if cfg.Workspace.Mode != "auto" {
		t.Fatalf("workspace mode = %q, want auto", cfg.Workspace.Mode)
	}
	if got := cfg.Providers[0].Register; got != "Register" {
		t.Fatalf("provider register = %q, want Register", got)
	}
	if got := cfg.Sources[0].Compile.Mode; got != "runtime" {
		t.Fatalf("compile mode = %q, want runtime", got)
	}
}

func TestLoadDataRejectsV1WithMigrationDiagnostic(t *testing.T) {
	_, err := LoadData([]byte(`name: old
packages:
  - id: core
    import: github.com/example/core
`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "xgoja migrate-spec") {
		t.Fatalf("error %q does not mention migrate-spec", err.Error())
	}
}

func TestLoadDataRejectsBroadBundlerFields(t *testing.T) {
	_, err := LoadData([]byte(`schema: xgoja/v2
name: bad
sources:
  - id: app
    kind: jsverbs
    from:
      dir: ./verbs
    language: typescript
    compile:
      mode: runtime
      bundle: true
      platform: browser
`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "field platform not found") {
		t.Fatalf("error %q does not mention unknown platform field", err.Error())
	}
}

func TestValidateRejectsDuplicateRuntimeAliases(t *testing.T) {
	cfg := &Config{
		Schema: Schema,
		Name:   "dup",
		Providers: []ProviderSpec{
			{ID: "core", Import: "github.com/example/core", Register: "Register"},
		},
		Runtime: RuntimeSpec{Modules: []RuntimeModuleSpec{
			{Provider: "core", Name: "one", As: "mod"},
			{Provider: "core", Name: "two", As: "mod"},
		}},
	}
	ApplyDefaults(cfg)
	report := Validate(cfg)
	if !report.HasErrors() {
		t.Fatal("expected duplicate alias validation error")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Path == "runtime.modules[1].as" && strings.Contains(check.Message, "duplicate runtime module alias") {
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate alias check not found in %#v", report.Checks)
	}
}

func TestRenderAppliesStableDefaults(t *testing.T) {
	out, err := Render(Config{
		Name: "My App",
		Providers: []ProviderSpec{
			{ID: "core", Import: "github.com/example/core"},
		},
		Artifacts: []ArtifactSpec{
			{ID: "binary", Type: "binary"},
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"schema: xgoja/v2",
		"name: My App",
		"module: xgoja.generated/my-app",
		"mode: auto",
		"register: Register",
		"output: dist/my-app",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, text)
		}
	}
}
