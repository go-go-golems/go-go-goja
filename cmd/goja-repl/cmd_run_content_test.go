package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

type runTestLook struct{}

func (runTestLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: []byte("png")}}, nil
}

type runLookInit struct{}

func (runLookInit) ID() string { return "goja-repl-run-test-look" }

func (runLookInit) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("look", func() runTestLook { return runTestLook{} })
}

func TestRunScriptReportsContent(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "look.js")
	if err := os.WriteFile(script, []byte(`module.exports = { img: look() };`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := runScriptFile(context.Background(), runScriptOptions{File: script, SafeMode: true, Out: out, initializers: []engine.RuntimeInitializer{runLookInit{}}}); err != nil {
		t.Fatalf("run without dir: %v", err)
	}
	if !strings.HasPrefix(out.String(), "{img: [content 1]}\n# 1 content item(s) not written") {
		t.Fatalf("output = %q", out.String())
	}

	out.Reset()
	contentDir := filepath.Join(dir, "content")
	if err := runScriptFile(context.Background(), runScriptOptions{File: script, SafeMode: true, Out: out, ContentDir: contentDir, initializers: []engine.RuntimeInitializer{runLookInit{}}}); err != nil {
		t.Fatalf("run with dir: %v", err)
	}
	path := filepath.Join(contentDir, "content-1.png")
	if data, err := os.ReadFile(path); err != nil || string(data) != "png" || !strings.Contains(out.String(), "# content 1: "+path) {
		t.Fatalf("file=%q err=%v output=%q", data, err, out.String())
	}
}

func TestRunHelpDocumentsContentDir(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"run", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--content-dir") {
		t.Fatalf("run help missing --content-dir:\n%s", out.String())
	}
}
