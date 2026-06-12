package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func TestRootHelp(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"xgoja", "build", "generate", "gen-dts", "doctor", "inspect", "list-modules", "migrate-spec"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected help to contain %q, got %q", want, rendered)
		}
	}
}

func TestBundledHelpTopic(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"help", "user-guide"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help topic: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"xgoja user guide and buildspec reference", "Runtime filesystem source", "Provider-shipped source"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected bundled help to contain %q, got %q", want, rendered)
		}
	}
}

func TestBuildCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	workDir := filepath.Join(t.TempDir(), "work")
	root.SetArgs([]string{"build", "-f", specPath, "--output", "./dist/fixture", "--work-dir", workDir, "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute build: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{
		"generated build workspace",
		"generated module: xgoja.generated/fixture",
		"xgoja builds from the generated module root",
		"release note: if you check this generated host into a repository as a nested Go module",
		"xgoja dry run ok",
		"./dist/fixture",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected build output to contain %q, got %q", want, rendered)
		}
	}
	for _, name := range []string{"go.mod", "main.go", "xgoja.gen.json"} {
		if _, err := os.Stat(filepath.Join(workDir, name)); err != nil {
			t.Fatalf("expected generated %s: %v", name, err)
		}
	}
}

func TestGenerateCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "xgoja_runtime.gen.go")); err != nil {
		t.Fatalf("expected generated package: %v", err)
	}
	if !strings.Contains(out.String(), "xgoja generate ok") {
		t.Fatalf("expected generate output, got %q", out.String())
	}
}

func TestGenerateCommandLetsGeneratorSanitizeInferredPackageName(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpecWithoutPackageName(t)
	outputDir := filepath.Join(t.TempDir(), "xgoja-runtime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "xgoja_runtime.gen.go"))
	if err != nil {
		t.Fatalf("read generated package: %v", err)
	}
	if !strings.Contains(string(data), "package xgoja_runtime") {
		t.Fatalf("expected sanitized inferred package name, got:\n%s", data)
	}
}

func TestGenerateCommandPrintsTemplateData(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpecWithoutPackageName(t)
	outputDir := filepath.Join(t.TempDir(), "xgoja-runtime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--template-data"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate template-data: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{`"PackageName": "xgoja_runtime"`, `"ProviderImports"`, `"SpecJSON"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected template data to contain %s, got %s", want, rendered)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, "xgoja_runtime.gen.go")); !os.IsNotExist(err) {
		t.Fatalf("template-data should not write output, stat err=%v", err)
	}
}

func TestGenerateCommandCleanRemovesKnownGeneratedFiles(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeSourceSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	if err := os.MkdirAll(filepath.Join(outputDir, "xgoja_embed"), 0o755); err != nil {
		t.Fatalf("mkdir stale embed: %v", err)
	}
	staleGenerated := filepath.Join(outputDir, "xgoja_runtime.gen.go")
	if err := os.WriteFile(staleGenerated, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale generated file: %v", err)
	}
	keep := filepath.Join(outputDir, "keep.go")
	if err := os.WriteFile(keep, []byte("package keep"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime", "--clean"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate clean source: %v", err)
	}
	if _, err := os.Stat(staleGenerated); !os.IsNotExist(err) {
		t.Fatalf("expected stale generated file removed, err=%v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("expected non-generated file preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "spec.gen.go")); err != nil {
		t.Fatalf("expected regenerated source fragment: %v", err)
	}
}

func TestGenerateCommandWritesSourceFragments(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeSourceSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate source: %v", err)
	}
	for _, name := range []string{"spec.gen.go", "providers.gen.go", "bundle.gen.go"} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected source fragment %s: %v", name, err)
		}
	}
}

func TestGenerateCommandWritesCustomTemplate(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTemplateSpec(t)
	outputPath := filepath.Join(t.TempDir(), "custom.gen.go")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputPath, "--package", "customruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate template: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read custom output: %v", err)
	}
	if !strings.Contains(string(data), "package customruntime") || !strings.Contains(string(data), "const ProviderCount = 1") {
		t.Fatalf("unexpected custom output: %s", data)
	}
}

func TestBuildCommandLoadsV2SpecDryRun(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeV2Spec(t)
	root.SetArgs([]string{"build", "-f", specPath, "--dry-run", "--keep-work"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute v2 build dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "validated xgoja/v2 plan") {
		t.Fatalf("expected v2 validation output, got %q", out.String())
	}
}

func TestBuildCommandBuildsBinary(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeBuildableSpec(t)
	outputPath := filepath.Join(t.TempDir(), "fixture")
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"build", "-f", specPath, "--output", outputPath, "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute build: %v", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output binary: %v", err)
	}
}

func TestDoctorCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"doctor", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
}

func TestDoctorCommandLoadsV2Spec(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeV2Spec(t)
	root.SetArgs([]string{"doctor", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
}

func TestMigrateSpecCommandWritesOutput(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	outputPath := filepath.Join(t.TempDir(), "xgoja.v2.yaml")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--out", outputPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read migrated spec: %v", err)
	}
	text := string(data)
	for _, want := range []string{"schema: xgoja/v2", "providers:", "runtime:", "artifacts:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected migrated spec to contain %q, got:\n%s", want, text)
		}
	}
	if !strings.Contains(out.String(), "wrote migrated xgoja/v2 spec") {
		t.Fatalf("expected migrate output, got %q", out.String())
	}
}

func TestMigrateSpecCommandInPlaceBackup(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--in-place", "--backup"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec in-place: %v", err)
	}
	if _, err := os.Stat(specPath + ".bak"); err != nil {
		t.Fatalf("expected backup: %v", err)
	}
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read in-place spec: %v", err)
	}
	if !strings.Contains(string(data), "schema: xgoja/v2") {
		t.Fatalf("expected v2 schema after in-place migration, got:\n%s", data)
	}
}

func TestMigrateSpecCommandCheckAlreadyV2(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	rendered, err := specv2.Render(specv2.Config{
		Name: "already-v2",
		Providers: []specv2.ProviderSpec{{
			ID:     "core",
			Import: "github.com/example/core",
		}},
		Artifacts: []specv2.ArtifactSpec{{
			ID:   "binary",
			Type: "binary",
		}},
	})
	if err != nil {
		t.Fatalf("render v2 spec: %v", err)
	}
	specPath := writeFile(t, "xgoja.v2.yaml", string(rendered)+"\n")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec check: %v", err)
	}
	if !strings.Contains(out.String(), "already in rendered xgoja/v2 form") {
		t.Fatalf("expected check output, got %q", out.String())
	}
}

func TestMigrateSpecCommandPrintsWarnings(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeFile(t, "xgoja.yaml", `name: warnings
appName: warnings
packages:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
modules:
  - package: http
    name: express
commands:
  jsverbs:
    enabled: true
jsverbs:
  - id: local-sites
    path: ./verbs
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      external:
        - express
`)
	outputPath := filepath.Join(t.TempDir(), "xgoja.v2.yaml")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--out", outputPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec warnings: %v", err)
	}
	if !strings.Contains(out.String(), "warning:") || !strings.Contains(out.String(), "runtime module alias") {
		t.Fatalf("expected warning output, got %q", out.String())
	}
}

func TestInspectCommandReadsCurrentBinary(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"inspect", os.Args[0], "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute inspect: %v", err)
	}
}

func TestGenDTSCommandLoadsV2Spec(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTypedCoreV2Spec(t)
	outputPath := filepath.Join(t.TempDir(), "xgoja-modules.d.ts")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"gen-dts", "-f", specPath, "--out", outputPath, "--strict", "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute v2 gen-dts: %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated dts: %v", err)
	}
	if !strings.Contains(string(data), `declare module "path:typed"`) {
		t.Fatalf("expected aliased path declaration, got:\n%s", data)
	}
	if !strings.Contains(out.String(), "validated xgoja/v2 plan") {
		t.Fatalf("expected v2 validation output, got %q", out.String())
	}
}

func TestGenDTSCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTypedCoreSpec(t)
	outputPath := filepath.Join(t.TempDir(), "xgoja-modules.d.ts")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"gen-dts", "-f", specPath, "--out", outputPath, "--strict", "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute gen-dts: %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated dts: %v", err)
	}
	if !strings.Contains(string(data), `declare module "path:typed"`) {
		t.Fatalf("expected aliased path declaration, got:\n%s", data)
	}
}

func TestListModulesCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"list-modules", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute list-modules: %v", err)
	}
}

func writePackageSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture-package
target:
  kind: package
  output: internal/xgojaruntime
  package: xgojaruntime
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
modules:
  - package: fixture
    name: hello
    as: hello
`), 0o644); err != nil {
		t.Fatalf("write package spec: %v", err)
	}
	return specPath
}

func writePackageSpecWithoutPackageName(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture-package

target:
  kind: package
  output: internal/xgoja-runtime
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
modules:
  - package: fixture
    name: hello
    as: hello
`), 0o644); err != nil {
		t.Fatalf("write package spec without package name: %v", err)
	}
	return specPath
}

func writeSourceSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture-source
target:
  kind: source
  output: internal/xgojaruntime
  package: xgojaruntime
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
modules:
  - package: fixture
    name: hello
    as: hello
`), 0o644); err != nil {
		t.Fatalf("write source spec: %v", err)
	}
	return specPath
}

func writeTemplateSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "runtime.go.tmpl")
	if err := os.WriteFile(templatePath, []byte(`// Code generated by custom xgoja template; DO NOT EDIT.
package {{ .PackageName }}

const ProviderCount = {{ len .ProviderImports }}
`), 0o644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture-template
target:
  kind: template
  output: internal/xgojaruntime/custom.gen.go
  package: customruntime
  template: runtime.go.tmpl
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
modules:
  - package: fixture
    name: hello
    as: hello
`), 0o644); err != nil {
		t.Fatalf("write template spec: %v", err)
	}
	return specPath
}

func writeBuildableSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
modules:
  - package: fixture
    name: hello
    as: hello
commands:
  repl:
    enabled: true
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}
	return specPath
}

func writeTypedCoreV2Spec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: typed-core
go:
  module: xgoja.generated/typed-core
  version: "1.26"
workspace:
  mode: off
providers:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
    register: Register
runtime:
  modules:
    - provider: go-go-goja-core
      name: path
      as: path:typed
commands:
  - id: eval
    type: builtin.eval
    name: eval
artifacts:
  - id: declarations
    type: dts
    output: xgoja-modules.d.ts
    strict: true
`), 0o644); err != nil {
		t.Fatalf("write typed core v2 spec: %v", err)
	}
	return specPath
}

func writeTypedCoreSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: typed-core
packages:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
modules:
  - package: go-go-goja-core
    name: path
    as: path:typed
commands:
  eval:
    enabled: true
`), 0o644); err != nil {
		t.Fatalf("write typed core spec: %v", err)
	}
	return specPath
}

func writeV2Spec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.Mkdir(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(verbsDir, "site.js"), []byte(`__package__({ name: "sites" })`), 0o644); err != nil {
		t.Fatalf("write verb: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: fixture
go:
  module: xgoja.generated/fixture
  version: "1.26"
workspace:
  mode: off
providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
sources:
  - id: local
    kind: jsverbs
    from:
      dir: ./verbs
    extensions: [.js]
commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [local]
artifacts:
  - id: bin
    type: binary
    output: dist/fixture
    sources: [local]
`), 0o644); err != nil {
		t.Fatalf("write v2 spec: %v", err)
	}
	return specPath
}

func writeValidSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.Mkdir(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: fixture
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/xgoja
modules:
  - package: core
    name: fs
commands:
  repl:
    enabled: true
jsverbs:
  - id: local
    path: ./verbs
    embed: true
`), 0o644); err != nil {
		t.Fatalf("write build spec: %v", err)
	}
	return specPath
}

func writeFile(t *testing.T, name string, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
