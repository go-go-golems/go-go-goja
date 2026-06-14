---
Title: "xgoja overview"
Slug: overview
Short: "How xgoja builds custom goja-powered binaries from provider packages and a top-level module set."
Topics:
- xgoja
- goja
- code-generation
- providers
- cli
Commands:
- xgoja
- xgoja build
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

`xgoja` builds custom goja-powered command-line programs by generating a normal Go program and compiling it with the Go toolchain. The generated program imports selected provider packages, registers their native modules, embeds a v2-native `app.RuntimePlan`, and exposes commands such as `eval`, `run`, rich `repl`, `modules`, configured JavaScript verbs, and provider command sets.

The design is intentionally compile-time oriented. Go-backed native modules are selected before the generated binary is built. JavaScript code can still be loaded at runtime or embedded into the generated binary, but the Go packages that implement native `require()` modules are ordinary Go imports in the generated source.

## Core model

The build has two separate decisions.

1. **Build-time provider selection** decides which provider packages are compiled into the binary.
2. **Runtime plan selection** decides which compiled-in modules, sources, commands, and artifacts are active.

A provider package contributes named modules, optional command sets, and optional source descriptors. `runtime.modules` selects provider modules by provider ID and module name, then assigns the `require()` alias used by JavaScript. `commands[]` selects built-in command surfaces and provider command sets.

```text
xgoja.yaml
  -> generated go.mod + main.go
  -> provider Register functions
  -> embedded app.RuntimePlan
  -> go build
  -> generated binary
  -> module set
  -> goja runtime + require modules
```

## Provider packages

A provider package exports a registration function, usually named `Register`, that receives a `*providerapi.ProviderRegistry`.

```go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("fixture",
        providerapi.Module{
            Name:      "hello",
            DefaultAs: "hello",
            NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
                return newHelloModule(ctx)
            },
        },
    )
}
```

The generated program imports this package and calls the registration function at startup. If the package is not listed in `providers:`, it is not part of the generated binary.

## Module selections

The runtime module list selects the modules that should be registered for runtime-backed command invocations.

```yaml
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
```

The `as` field controls the JavaScript module name:

```js
const hello = require("hello")
```

The runtime module list makes the JavaScript capability surface explicit. A provider package can be compiled into the binary without exposing every module it registers; only modules listed in `runtime.modules` are registered for generated runtime-backed commands.

## Generated commands

A pure xgoja generated binary currently provides these command families:

- a `commands[]` entry with `type: builtin.eval` evaluates a JavaScript string in the generated runtime.
- a `commands[]` entry with `type: builtin.run` executes a JavaScript file with script-local module resolution.
- a `commands[]` entry with `type: builtin.repl` starts an interactive Bubble Tea REPL for the generated runtime.
- `modules` lists provider modules registered in the binary.
- a `commands[]` entry with `type: builtin.jsverbs` mounts JavaScript functions as Glazed/Cobra commands.

Provider packages can also contribute command families. For example, the HTTP provider can mount `serve` commands that run JavaScript verbs as long-lived Express site setup functions.

Existing applications can also use xgoja in target modes that attach generated commands to a Cobra root, delegate integration to an adapter package, or generate an importable runtime package with `xgoja generate`.

## When to use xgoja

Use `xgoja` when a CLI needs JavaScript scripting with Go-backed native modules, and those native modules should be selected through a declarative build rather than manually edited into a custom `main.go`.

Do not use `xgoja` as a dynamic Go plugin loader. The reliable extension boundary is source generation followed by a normal Go build.

## See also

- `xgoja-v2-reference` for the YAML reference.
- `user-guide` for the current native v2 workflow.
- `migrating-to-xgoja-v2` for converting legacy specs.
- `examples/xgoja` in the repository for runnable v2 projects.
