package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICommandFlow(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "repl.sqlite")

	createOut := &bytes.Buffer{}
	createRoot, err := newRootCommand(createOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	createRoot.SetArgs([]string{"--db-path", dbPath, "create"})
	if err := createRoot.Execute(); err != nil {
		t.Fatalf("execute create: %v", err)
	}

	var createPayload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create output: %v", err)
	}
	if createPayload.Session.ID == "" {
		t.Fatal("expected session id from create command")
	}

	evalOut := &bytes.Buffer{}
	evalRoot, err := newRootCommand(evalOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	evalRoot.SetArgs([]string{"--db-path", dbPath, "eval", "--session-id", createPayload.Session.ID, "--source", "const x = 1; x"})
	if err := evalRoot.Execute(); err != nil {
		t.Fatalf("execute eval: %v", err)
	}

	historyOut := &bytes.Buffer{}
	historyRoot, err := newRootCommand(historyOut)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	historyRoot.SetArgs([]string{"--db-path", dbPath, "history", "--session-id", createPayload.Session.ID})
	if err := historyRoot.Execute(); err != nil {
		t.Fatalf("execute history: %v", err)
	}

	var historyPayload struct {
		History []any `json:"history"`
	}
	if err := json.Unmarshal(historyOut.Bytes(), &historyPayload); err != nil {
		t.Fatalf("decode history output: %v", err)
	}
	if len(historyPayload.History) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(historyPayload.History))
	}
}

func TestTUICommandHelp(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"tui", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute tui help: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "--profile") {
		t.Fatalf("expected tui help to describe --profile, got %q", rendered)
	}
	if !strings.Contains(rendered, "--alt-screen") {
		t.Fatalf("expected tui help to describe --alt-screen, got %q", rendered)
	}
}

func TestEssayCommandHelp(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"essay", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute essay help: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "interactive REPL essay") {
		t.Fatalf("expected essay help to describe the interactive REPL essay, got %q", rendered)
	}
	if !strings.Contains(rendered, "--addr") {
		t.Fatalf("expected essay help to describe --addr, got %q", rendered)
	}
}

func TestRunCommandExecutesScript(t *testing.T) {
	t.Parallel()

	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte("const x = 1 + 1; x;"), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"run", script})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute run: %v", err)
	}
}

func TestRunScriptFileResolvesRelativeRequireFromEntryDirectory(t *testing.T) {
	scriptDir := t.TempDir()
	entryPath := filepath.Join(scriptDir, "entry.js")
	siblingPath := filepath.Join(scriptDir, "sibling.js")
	if err := os.WriteFile(siblingPath, []byte("module.exports = { value: 42 };"), 0644); err != nil {
		t.Fatalf("write sibling script: %v", err)
	}
	entrySource := `
const sibling = require("./sibling");
if (sibling.value !== 42) {
  throw new Error("relative require resolved incorrectly");
}
`
	if err := os.WriteFile(entryPath, []byte(entrySource), 0644); err != nil {
		t.Fatalf("write entry script: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	otherWD := t.TempDir()
	if err := os.Chdir(otherWD); err != nil {
		t.Fatalf("chdir other cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	if err := runScriptFile(context.Background(), runScriptOptions{File: entryPath, UseModuleRoots: true}); err != nil {
		t.Fatalf("run script with sibling require from other cwd: %v", err)
	}
}

func TestRunScriptFileNotFound(t *testing.T) {
	t.Parallel()

	err := runScriptFile(context.Background(), runScriptOptions{
		File: filepath.Join(t.TempDir(), "does-not-exist.js"),
	})
	if err == nil {
		t.Fatal("expected missing file to fail")
	}
	if !strings.Contains(err.Error(), "script file not found") {
		t.Fatalf("expected missing file error, got %v", err)
	}
}

func TestRunScriptFileBadSyntax(t *testing.T) {
	t.Parallel()

	badScript := filepath.Join(t.TempDir(), "bad.js")
	if err := os.WriteFile(badScript, []byte("const x = [bad;"), 0644); err != nil {
		t.Fatalf("write bad script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{File: badScript})
	if err == nil {
		t.Fatal("expected bad syntax to fail")
	}
	if !strings.Contains(err.Error(), "run ") {
		t.Fatalf("expected run error with filename context, got %v", err)
	}
}

func TestRunCommandHelp(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"run", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute run help: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "Execute a JavaScript file") {
		t.Fatalf("expected run help to describe command, got %q", rendered)
	}
	if !strings.Contains(rendered, "file") {
		t.Fatalf("expected run help to describe file argument, got %q", rendered)
	}
}

func TestParseTUIProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{input: "", want: "interactive"},
		{input: "interactive", want: "interactive"},
		{input: "raw", want: "raw"},
		{input: "persistent", want: "persistent"},
	}
	for _, tc := range cases {
		got, err := parseTUIProfile(tc.input)
		if err != nil {
			t.Fatalf("parse profile %q: %v", tc.input, err)
		}
		if string(got) != tc.want {
			t.Fatalf("parse profile %q: expected %q, got %q", tc.input, tc.want, got)
		}
	}
	if _, err := parseTUIProfile("weird"); err == nil {
		t.Fatal("expected invalid profile to fail")
	}
}

func TestRunCommandSafeModeDisablesFs(t *testing.T) {
	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte(`require("fs");`), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{
		File:           script,
		SafeMode:       true,
		UseModuleRoots: true,
	})
	if err == nil {
		t.Fatal("expected fs require to fail in safe mode")
	}
	if !strings.Contains(err.Error(), "Invalid module") {
		t.Fatalf("expected Invalid module error, got %v", err)
	}
}

func TestRunCommandDisableModule(t *testing.T) {
	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte(`require("fs");`), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{
		File:           script,
		DisableModules: []string{"fs"},
		UseModuleRoots: true,
	})
	if err == nil {
		t.Fatal("expected fs require to fail when disabled")
	}
	if !strings.Contains(err.Error(), "Invalid module") {
		t.Fatalf("expected Invalid module error, got %v", err)
	}
}

func TestRunCommandEnableModule(t *testing.T) {
	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte(`require("fs");`), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{
		File:           script,
		EnableModules:  []string{"fs"},
		UseModuleRoots: true,
	})
	if err != nil {
		t.Fatalf("execute run with --enable-module fs: %v", err)
	}
}

func TestRunCommandEnableDatabaseAlias(t *testing.T) {
	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte(`const db = require("db"); if (typeof db.query !== "function") throw new Error("missing query");`), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{
		File:           script,
		EnableModules:  []string{"db"},
		UseModuleRoots: true,
	})
	if err != nil {
		t.Fatalf("execute run with --enable-module db: %v", err)
	}
}

func TestRunCommandEnableModuleOnly(t *testing.T) {
	script := filepath.Join(t.TempDir(), "test.js")
	if err := os.WriteFile(script, []byte(`require("fs"); require("os");`), 0644); err != nil {
		t.Fatalf("write test script: %v", err)
	}

	err := runScriptFile(context.Background(), runScriptOptions{
		File:           script,
		EnableModules:  []string{"fs"},
		UseModuleRoots: true,
	})
	if err == nil {
		t.Fatal("expected os require to fail when not enabled")
	}
	if !strings.Contains(err.Error(), "Invalid module") {
		t.Fatalf("expected Invalid module error, got %v", err)
	}
}
