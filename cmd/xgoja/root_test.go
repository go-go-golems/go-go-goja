package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, want := range []string{"xgoja", "build", "generate", "doctor", "inspect", "list-modules"} {
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
	if !strings.Contains(rendered, "xgoja dry run ok") || !strings.Contains(rendered, "./dist/fixture") || !strings.Contains(rendered, "generated build workspace") {
		t.Fatalf("expected build output to mention dry-run plan, got %q", rendered)
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
