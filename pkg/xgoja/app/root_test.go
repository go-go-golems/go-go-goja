package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider"
)

const fixtureSpecJSON = `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {
    "repl": {
      "modules": [{"package": "fixture", "name": "hello", "as": "hello"}]
    }
  },
  "commands": {
    "repl": {"enabled": true, "runtime": "repl", "name": "repl"},
    "jsverbs": {"enabled": false}
  }
}`

func TestGeneratedRootEvalUsesProviderModule(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: fixtureSpecJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"repl", `require("hello").greet("intern")`})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "hello intern\n" {
		t.Fatalf("eval output = %q", got)
	}
}

func TestGeneratedRootRespectsConfiguredReplCommandName(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {
    "repl": {
      "modules": [{"package": "fixture", "name": "hello", "as": "hello"}]
    }
  },
  "commands": {
    "repl": {"enabled": true, "runtime": "repl", "name": "runjs"},
    "jsverbs": {"enabled": false}
  }
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"runjs", `require("hello").greet("intern")`})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute configured repl command: %v", err)
	}
	if got := out.String(); got != "hello intern\n" {
		t.Fatalf("configured repl output = %q", got)
	}
}

func TestGeneratedRootRespectsDisabledReplCommand(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {
    "repl": {
      "modules": [{"package": "fixture", "name": "hello", "as": "hello"}]
    }
  },
  "commands": {
    "repl": {"enabled": false, "runtime": "repl", "name": "runjs"},
    "jsverbs": {"enabled": false}
  }
}`
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	for _, cmd := range root.Commands() {
		if cmd.Name() == "runjs" || cmd.Name() == "eval" {
			t.Fatalf("repl command %q attached despite commands.repl.enabled=false", cmd.Name())
		}
	}
}

func TestGeneratedRootModulesCommand(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: fixtureSpecJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"modules"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute modules: %v", err)
	}
	if got := out.String(); got != "fixture.hello\nfixture.owner-check\n" {
		t.Fatalf("modules output = %q", got)
	}
}

func TestRuntimeFactoryDoesNotExposeImplicitEngineModules(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	spec := &Spec{}
	if err := json.Unmarshal([]byte(fixtureSpecJSON), spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	rt, err := NewRuntimeFactory(registry, spec).NewRuntime(context.Background(), "repl")
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	if _, err := rt.Require.Require("hello"); err != nil {
		t.Fatalf("require hello: %v", err)
	}
	if _, err := rt.Require.Require("path"); err == nil {
		t.Fatalf("require path succeeded, want xgoja spec-selected modules only")
	}
}

func TestGeneratedRootMountsProviderJSVerbs(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {"repl": {"modules": [{"package": "fixture", "name": "hello", "as": "hello"}, {"package": "fixture", "name": "owner-check", "as": "owner-check"}]}},
  "commands": {"repl": {"enabled": true, "runtime": "repl", "name": "repl"}, "jsverbs": {"enabled": true, "runtime": "repl", "name": "verbs"}},
  "jsverbs": [{"id": "provider", "package": "fixture", "source": "verbs"}]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "provider-greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute provider verb: %v", err)
	}

	root, err = NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root for owner verb: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "owner-ping"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute provider owner verb: %v", err)
	}
}

func TestGeneratedRootMountsEmbeddedJSVerbs(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	embedded := fstest.MapFS{
		"xgoja_embed/jsverbs/local/tools.js": &fstest.MapFile{Data: []byte(`
__package__({ name: "tools" })
__verb__("embeddedGreet", {
  name: "embedded-greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function embeddedGreet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
`)},
	}
	specJSON := `{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {"repl": {"modules": [{"package": "fixture", "name": "hello", "as": "hello"}]}},
  "commands": {"repl": {"enabled": true, "runtime": "repl", "name": "repl"}, "jsverbs": {"enabled": true, "runtime": "repl", "name": "verbs"}},
  "jsverbs": [{"id": "local", "path": "xgoja_embed/jsverbs/local", "embed": true}]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out, EmbeddedJSVerbs: embedded})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "embedded-greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute embedded verb: %v", err)
	}
}

func TestGeneratedRootMountsFilesystemJSVerbs(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	verbsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(verbsDir, "tools.js"), []byte(`
__package__({ name: "tools" })
__verb__("greet", {
  name: "greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function greet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
`), 0o644); err != nil {
		t.Fatalf("write verb: %v", err)
	}
	specJSON := fmt.Sprintf(`{
  "name": "fixture",
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "packages": [{"id": "fixture"}],
  "runtimes": {"repl": {"modules": [{"package": "fixture", "name": "hello", "as": "hello"}]}},
  "commands": {"repl": {"enabled": true, "runtime": "repl", "name": "repl"}, "jsverbs": {"enabled": true, "runtime": "repl", "name": "verbs"}},
  "jsverbs": [{"id": "local", "path": %q, "embed": false}]
}`, verbsDir)
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute mounted verb: %v", err)
	}
	// The Glazed writer command currently writes through the framework output path
	// rather than this root's bytes.Buffer. Successful execution proves the
	// mounted command scanned, built, created an xgoja runtime, and invoked JS.
}
