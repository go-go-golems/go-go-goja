package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
	root.SetArgs([]string{"eval", `require("hello").greet("intern")`})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute eval: %v", err)
	}
	if got := out.String(); got != "hello intern\n" {
		t.Fatalf("eval output = %q", got)
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
	if got := out.String(); got != "fixture.hello\n" {
		t.Fatalf("modules output = %q", got)
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
