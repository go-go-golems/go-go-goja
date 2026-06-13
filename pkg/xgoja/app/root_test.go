package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider"
)

const fixtureRuntimePlanJSON = `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    }
  ]
}`

func TestGeneratedRootEvalUsesProviderModule(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: fixtureRuntimePlanJSON, Out: out})
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

func TestGeneratedRootRespectsConfiguredReplCommandName(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "runjs"
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
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
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  }
}`
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	for _, cmd := range root.Commands() {
		if cmd.Name() == "runjs" || cmd.Name() == "eval" {
			t.Fatalf("eval command %q attached despite commands.eval.enabled=false", cmd.Name())
		}
	}
}

func TestGeneratedRootInstallsHelpAndLogging(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: fixtureRuntimePlanJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	if root.PersistentPreRunE == nil {
		t.Fatalf("expected generated root to initialize logging with PersistentPreRunE")
	}
	if root.PersistentFlags().Lookup("log-level") == nil {
		t.Fatalf("expected generated root to expose glazed logging flags")
	}
	root.SetArgs([]string{"help", "runtime-overview"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute generated help: %v", err)
	}
	if got := out.String(); !bytes.Contains([]byte(got), []byte("generated xgoja runtime overview")) {
		t.Fatalf("expected generated help topic, got %q", got)
	}
}

func TestGeneratedRootLoadsProviderHelpSource(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	docs := fstest.MapFS{
		"topics/01-fixture.md": {Data: []byte(`---
Title: Fixture JavaScript API reference
Slug: fixture-js-api-reference
Short: Fixture provider docs.
Topics:
- fixture
Commands:
- fixture
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Fixture provider help body.
`)},
	}
	if err := registry.Package("fixture",
		providerapi.Module{Name: "hello", NewModuleFactory: noopAppModuleFactory},
		providerapi.HelpSource{Name: "docs", FS: docs, Root: "."},
	); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "sources": [
    {
      "id": "fixture-docs",
      "source": "docs",
      "kind": "help",
      "provider": "fixture"
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"help", "fixture-js-api-reference"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	if got := out.String(); !bytes.Contains([]byte(got), []byte("Fixture JavaScript API reference")) || !bytes.Contains([]byte(got), []byte("Fixture provider help body")) {
		t.Fatalf("expected provider help topic, got %q", got)
	}
}

func TestGeneratedRootLoadsEmbeddedLocalHelpSource(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	embeddedHelp := fstest.MapFS{
		"xgoja_embed/help/local/topics/01-local.md": {Data: []byte(`---
Title: Local generated help
Slug: local-generated-help
Short: Local generated docs.
Topics:
- local
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Local generated help body.
`)},
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "sources": [
    {
      "id": "local",
      "path": "xgoja_embed/help/local",
      "embed": true,
      "kind": "help"
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out, EmbeddedHelp: embeddedHelp})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"help", "local-generated-help"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	if got := out.String(); !bytes.Contains([]byte(got), []byte("Local generated help")) || !bytes.Contains([]byte(got), []byte("Local generated help body")) {
		t.Fatalf("expected embedded local help topic, got %q", got)
	}
}

func TestGeneratedRootReportsMissingProviderHelpSource(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "sources": [
    {
      "id": "missing",
      "source": "missing",
      "kind": "help",
      "provider": "fixture"
    }
  ]
}`
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON})
	if err != nil {
		t.Fatalf("new root should defer framework errors to execution, got %v", err)
	}
	root.SetArgs([]string{"help", "runtime-overview"})
	err = root.ExecuteContext(context.Background())
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("unknown provider help source fixture.missing")) {
		t.Fatalf("expected missing provider help source error, got %v", err)
	}
}

func TestGeneratedRootTUIHelp(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "commands": [
    {
      "id": "repl",
      "type": "builtin.repl",
      "name": "repl"
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"repl", "--help"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute repl help: %v", err)
	}
	for _, want := range []string{"interactive TUI REPL", "--alt-screen"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("expected REPL help to contain %q, got %q", want, out.String())
		}
	}
}

func TestGeneratedRootRunCommandExecutesScriptFile(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.js"), []byte(`exports.name = "intern"`), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	script := filepath.Join(dir, "main.js")
	if err := os.WriteFile(script, []byte(`
const helper = require("./helper")
const hello = require("hello")
if (hello.greet(helper.name) !== "hello intern") {
  throw new Error("unexpected greeting")
}
`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    },
    {
      "id": "run",
      "type": "builtin.run",
      "name": "run"
    }
  ]
}`
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"run", script})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute run: %v", err)
	}
}

func TestGeneratedRootModulesCommand(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: fixtureRuntimePlanJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"modules"})
	got := captureStdout(t, func() {
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("execute modules: %v", err)
		}
	})
	for _, want := range []string{"fixture", "hello", "owner-check", "fixture.hello", "fixture.owner-check"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("modules output should contain %q, got %q", want, got)
		}
	}
}

func TestGeneratedRootSelectedModulesCommand(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      },
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello:custom",
        "config": {
          "message": "hi"
        }
      }
    ]
  },
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    }
  ]
}`
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"selected-modules", "--output", "json"})
	got := captureStdout(t, func() {
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("execute selected-modules: %v", err)
		}
	})
	for _, want := range []string{`"alias": "hello"`, `"alias": "hello:custom"`, `"provider_ref": "fixture.hello"`, `"config": "{\"message\":\"hi\"}"`} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("selected-modules json should contain %q, got %q", want, got)
		}
	}
}

func TestRuntimeFactoryDoesNotExposeImplicitEngineModules(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	runtimePlan := &RuntimePlan{}
	if err := json.Unmarshal([]byte(fixtureRuntimePlanJSON), runtimePlan); err != nil {
		t.Fatalf("parse runtime plan: %v", err)
	}
	rt, err := NewRuntimeFactory(registry, runtimePlan).NewRuntime(context.Background())
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
		t.Fatalf("require path succeeded, want xgoja runtime plan-selected modules only")
	}
}

func TestGeneratedRootMountsProviderJSVerbs(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := testprovider.Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	specJSON := `{
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      },
      {
        "provider": "fixture",
        "name": "owner-check",
        "as": "owner-check"
      }
    ]
  },
  "sources": [
    {
      "id": "provider",
      "source": "verbs",
      "kind": "jsverbs",
      "provider": "fixture"
    }
  ],
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    },
    {
      "id": "jsverbs",
      "type": "builtin.jsverbs",
      "name": "verbs",
      "sources": [
        "provider"
      ]
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "provider-greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute provider verb: %v", err)
	}

	root, err = NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
	if err != nil {
		t.Fatalf("new root for owner verb: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "owner-ping"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute provider owner verb: %v", err)
	}
}

func TestGeneratedRootMountsEmbeddedJSVerbs(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "sources": [
    {
      "id": "local",
      "path": "xgoja_embed/jsverbs/local",
      "embed": true,
      "kind": "jsverbs"
    }
  ],
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    },
    {
      "id": "jsverbs",
      "type": "builtin.jsverbs",
      "name": "verbs",
      "sources": [
        "local"
      ]
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out, EmbeddedJSVerbs: embedded})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"verbs", "tools", "embedded-greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute embedded verb: %v", err)
	}
}

func TestGeneratedRootMountsEmbeddedJSVerbsAtRoot(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {
    "name": "fixture"
  },
  "target": {
    "kind": "xgoja",
    "output": "dist/fixture"
  },
  "providers": [
    {
      "id": "fixture"
    }
  ],
  "runtime": {
    "modules": [
      {
        "provider": "fixture",
        "name": "hello",
        "as": "hello"
      }
    ]
  },
  "sources": [
    {
      "id": "local",
      "path": "xgoja_embed/jsverbs/local",
      "embed": true,
      "kind": "jsverbs"
    }
  ],
  "commands": [
    {
      "id": "eval",
      "type": "builtin.eval",
      "name": "eval"
    },
    {
      "id": "jsverbs",
      "type": "builtin.jsverbs",
      "name": "verbs",
      "mount": "root",
      "sources": [
        "local"
      ]
    }
  ]
}`
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out, EmbeddedJSVerbs: embedded})
	if err != nil {
		t.Fatalf("new root: %v", err)
	}
	root.SetArgs([]string{"tools", "embedded-greet", "--name", "intern"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute root-mounted embedded verb: %v", err)
	}
}

func TestGeneratedRootMountsFilesystemJSVerbs(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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
  "schema": "xgoja/runtime/v2",
  "name": "fixture",
  "app": {"name": "fixture"},
  "target": {"kind": "xgoja", "output": "dist/fixture"},
  "providers": [{"id": "fixture"}],
  "runtime": {"modules": [{"provider": "fixture", "name": "hello", "as": "hello"}]},
  "sources": [{"id": "local", "kind": "jsverbs", "path": %q, "embed": false}],
  "commands": [
    {"id": "eval", "type": "builtin.eval", "name": "eval"},
    {"id": "jsverbs", "type": "builtin.jsverbs", "name": "verbs", "sources": ["local"]}
  ]
}`, verbsDir)
	out := &bytes.Buffer{}
	root, err := NewRootCommand(Options{Providers: registry, RuntimePlanJSON: specJSON, Out: out})
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(data)
}

func noopAppModuleFactory(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
	return func(vm *goja.Runtime, module *goja.Object) {}, nil
}
