package app

import (
	"bytes"
	"context"
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
    "jsverbs": {"enabled": true, "runtime": "repl", "name": "verbs"}
  },
  "jsverbs": [{"id": "local", "path": "./verbs", "embed": true}]
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

func TestGeneratedRootVerbsCommandListsSources(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, SpecJSON: fixtureSpecJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute verbs: %v", err)
	}
	if got := out.String(); got != "local\n" {
		t.Fatalf("verbs output = %q", got)
	}
}
