package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type contentTestLook struct{ name string }

func (l *contentTestLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/jpeg", Data: []byte("jpeg:" + l.name), Caption: l.name}}, nil
}

func (l *contentTestLook) ResultPlaceholder() string { return "look " + l.name }

func newContentTestFactory(t *testing.T) *RuntimeFactory {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("contentfixture", providerapi.Module{
		Name: "look",
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return func(vm *goja.Runtime, module *goja.Object) {
				exports := module.Get("exports").(*goja.Object)
				_ = exports.Set("look", func(name string) *contentTestLook { return &contentTestLook{name: name} })
			}, nil
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return NewRuntimeFactory(registry, &RuntimePlan{
		Runtime: RuntimeSection{Modules: []RuntimeModulePlan{{Provider: "contentfixture", Name: "look", As: "look"}}},
	})
}

func runEvalCommand(t *testing.T, factory *RuntimeFactory, args ...string) string {
	t.Helper()
	out := &bytes.Buffer{}
	cmd, err := buildGlazedCobraCommand(newEvalCommand(factory, factory.runtimePlan, out))
	if err != nil {
		t.Fatalf("build eval command: %v", err)
	}
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("eval %v: %v", args, err)
	}
	return out.String()
}

func TestEvalPrintsContentPlaceholdersWithoutContentDir(t *testing.T) {
	factory := newContentTestFactory(t)
	got := runEvalCommand(t, factory, `({ n: 1, img: require("look").look("a") })`)
	want := "{n: 1, img: [content 1: look a]}\n# 1 content item(s) not written; pass --content-dir DIR to write them\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if got := runEvalCommand(t, factory, `40 + 2`); got != "42\n" {
		t.Fatalf("plain output = %q", got)
	}
	if got := runEvalCommand(t, factory, `Promise.resolve("awaited")`); got != "awaited\n" {
		t.Fatalf("promise output = %q", got)
	}
}

func TestEvalWritesContentDir(t *testing.T) {
	factory := newContentTestFactory(t)
	dir := filepath.Join(t.TempDir(), "out")
	got := runEvalCommand(t, factory, `(async () => [require("look").look("a"), require("look").look("b")])()`, "--content-dir", dir)
	for _, name := range []string{"content-1.jpg", "content-2.jpg"} {
		path := filepath.Join(dir, name)
		if !strings.Contains(got, path) {
			t.Fatalf("output %q does not name %s", got, path)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(dir, "content-2.jpg"))
	if string(data) != "jpeg:b" || !strings.HasPrefix(got, "[[content 1: look a], [content 2: look b]]\n") {
		t.Fatalf("output = %q data = %q", got, data)
	}
}

func TestRunReportsModuleExportsContent(t *testing.T) {
	factory := newContentTestFactory(t)
	dir := t.TempDir()
	script := filepath.Join(dir, "script.js")
	if err := os.WriteFile(script, []byte(`
const { look } = require("look");
module.exports = (async () => ({ stats: { mean: 0.5 }, after: look("after") }))();
`), 0o644); err != nil {
		t.Fatal(err)
	}
	contentDir := filepath.Join(dir, "looks")
	out := &bytes.Buffer{}
	cmd, err := buildGlazedCobraCommand(newRunCommand(factory, factory.runtimePlan, out))
	if err != nil {
		t.Fatalf("build run command: %v", err)
	}
	cmd.SetArgs([]string{script, "--content-dir", contentDir})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "{stats: {mean: 0.5}, after: [content 1: look after]}\n# content 1 after: " + filepath.Join(contentDir, "content-1.jpg") + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}

	plain := filepath.Join(dir, "plain.js")
	if err := os.WriteFile(plain, []byte(`module.exports = { a: 1 };`), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	cmd, _ = buildGlazedCobraCommand(newRunCommand(factory, factory.runtimePlan, out))
	cmd.SetArgs([]string{plain})
	if err := cmd.ExecuteContext(context.Background()); err != nil || out.Len() != 0 {
		t.Fatalf("plain run: err=%v output=%q", err, out.String())
	}
}
