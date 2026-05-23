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
	for _, want := range []string{"xgoja", "build", "doctor", "inspect", "list-modules"} {
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
	root.SetArgs([]string{"help", "buildspec"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help topic: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"xgoja buildspec reference", "Runtime filesystem source", "Provider-shipped source"} {
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
	root.SetArgs([]string{"list-modules", "-f", specPath, "--profile", "repl", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute list-modules: %v", err)
	}
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
runtimes:
  repl:
    modules:
      - package: fixture
        name: hello
        as: hello
commands:
  repl:
    enabled: true
    runtime: repl
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
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
runtimes:
  repl:
    modules:
      - package: core
        name: fs
commands:
  repl:
    enabled: true
    runtime: repl
jsverbs:
  - id: local
    path: ./verbs
    embed: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return specPath
}
