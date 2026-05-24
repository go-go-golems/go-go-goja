---
Title: "xgoja overview"
Slug: overview
Short: "How xgoja builds custom goja-powered binaries from provider packages and runtime profiles."
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

`xgoja` builds custom goja-powered command-line programs by generating a normal Go program and compiling it with the Go toolchain. The generated program imports selected provider packages, registers their native modules, embeds a normalized runtime specification, and exposes commands such as the configured `repl`/evaluation command, `modules`, and configured JavaScript verbs.

The design is intentionally compile-time oriented. Go-backed native modules are selected before the generated binary is built. JavaScript code can still be loaded at runtime or embedded into the generated binary, but the Go packages that implement native `require()` modules are ordinary Go imports in the generated source.

## Core model

The build has two separate decisions.

1. **Build-time package selection** decides which provider packages are compiled into the binary.
2. **Runtime profile selection** decides which compiled-in modules are available to one command invocation.

A provider package contributes named modules and optional JavaScript verb sources. A runtime profile selects provider modules by package ID and module name, then assigns the `require()` alias used by JavaScript.

```text
xgoja.yaml
  -> generated go.mod + main.go
  -> provider Register functions
  -> embedded runtime spec
  -> go build
  -> generated binary
  -> runtime profile
  -> goja runtime + require modules
```

## Provider packages

A provider package exports a registration function, usually named `Register`, that receives a `*providerapi.Registry`.

```go
func Register(registry *providerapi.Registry) error {
    return registry.Package("fixture",
        providerapi.Module{
            Name:      "hello",
            DefaultAs: "hello",
            New:       newHelloModule,
        },
    )
}
```

The generated program imports this package and calls the registration function at startup. If the package is not listed in `packages:`, it is not part of the generated binary.

## Runtime profiles

A runtime profile lists the modules that should be registered for a command invocation.

```yaml
runtimes:
  repl:
    modules:
      - package: fixture
        name: hello
        as: hello
```

The `as` field controls the JavaScript module name:

```js
const hello = require("hello")
```

Profiles make the capability surface explicit. A module compiled into the binary is not available to a command unless the command uses a profile that selects it.

## Generated commands

A pure xgoja generated binary currently provides these command families:

- the configured `commands.repl.name` command evaluates a JavaScript string in a selected runtime profile.
- `modules` lists provider modules registered in the binary.
- the configured `jsverbs` command mounts JavaScript functions as Glazed/Cobra commands when enabled.

Existing applications can also use xgoja in target modes that attach these generated commands to a Cobra root or delegate integration to an adapter package.

## When to use xgoja

Use `xgoja` when a CLI needs JavaScript scripting with Go-backed native modules, and those native modules should be selected through a declarative build rather than manually edited into a custom `main.go`.

Do not use `xgoja` as a dynamic Go plugin loader. The reliable extension boundary is source generation followed by a normal Go build.

## See also

- `buildspec` for the YAML reference.
- `tutorial` for an end-to-end build flow.
