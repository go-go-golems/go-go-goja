---
Title: Module Enablement Design
Ticket: GOJA-059
Status: active
Topics:
    - backend
    - cli
    - security
    - go-go-golems
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/module_specs.go:Engine module spec APIs (DefaultRegistryModules, DataOnlyDefaultRegistryModules, DefaultRegistryModulesNamed)
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/root.go:rootOptions and commandSupport (shared app construction)
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/cmd_run.go:run command builder uses DefaultRegistryModules()
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go:TUI evaluator Config and runtime builder
ExternalSources: []
Summary: "Design for adding granular module enablement via a ModuleMiddleware pipeline to all goja-repl commands, deprecating the baroque DefaultRegistryModules / DefaultRegistryModulesNamed / DataOnlyDefaultRegistryModules / DefaultRegistryModule API family in favor of a single UseModuleMiddleware builder method."
LastUpdated: 2026-04-28T09:15:00-04:00
WhatFor: "Reference design for implementing module security controls in goja-repl"
WhenToUse: "When adding flags, updating builder logic, or wiring module filtering into commands"
---

# Module Enablement Design (GOJA-059)

## Goal

Allow users to selectively enable or disable native Go modules when running JS through goja-repl. Today, `run` and `tui` unconditionally load **all** modules via `engine.DefaultRegistryModules()`. We will replace this with a composable `ModuleMiddleware` pipeline that gives fine-grained control over the module sandbox.

## Current State

### Module categories

| Category | Modules | Risk |
|----------|---------|------|
| Data-only (safe) | crypto, events, path, time, timer | Low — no host filesystem or process access |
| Host-access (dangerous) | fs, os, exec, database, yaml | High — filesystem, subprocess, DB, env |
| Process exposure | process, node:process | Medium — exposes `process.env` |

### Where modules are loaded

| Command | File | Builder call |
|---------|------|-------------|
| run | `cmd/goja-repl/cmd_run.go:94` | `engine.DefaultRegistryModules()` |
| tui, eval, create, etc. | `cmd/goja-repl/root.go:123` | `engine.DefaultRegistryModules()` |
| replapi.App (sessions) | `cmd/goja-repl/root.go:123` | `engine.DefaultRegistryModules()` |
| TUI evaluator | `pkg/repl/evaluators/javascript/evaluator.go:117` | `engine.DefaultRegistryModules()` |

All paths currently use the same unconditional `DefaultRegistryModules()` spec.

### The old API family (being deprecated)

```go
engine.DefaultRegistryModules()                          // all modules
engine.DataOnlyDefaultRegistryModules()                  // safe only
engine.DefaultRegistryModulesNamed("fs", "os")          // named only
engine.DefaultRegistryModule("fs")                       // single module
```

This is baroque: four functions for four selection strategies, none composable.

## Proposed Design: ModuleMiddleware Pipeline

### Core types

```go
// ModuleSelector chooses which modules to register from the full set of available names.
type ModuleSelector func(available []string) []string

// ModuleMiddleware wraps a selector. The standard pattern is f(next) returns a new selector.
// This gives explicit control flow: each middleware decides whether to call next,
// short-circuit, modify before/after, etc.
type ModuleMiddleware func(next ModuleSelector) ModuleSelector
```

### Base selector

```go
// SelectAll is the identity selector — includes all available modules.
func SelectAll(available []string) []string { return available }
```

### Built-in middlewares

**Override middlewares** (do NOT call next — replace the entire selection):

```go
// MiddlewareSafe returns only data-safe modules.
func MiddlewareSafe() ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        return func(available []string) []string {
            return intersect(available, safeModuleNames)  // doesn't call next
        }
    }
}

// MiddlewareOnly returns only the named modules.
func MiddlewareOnly(names ...string) ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        return func(available []string) []string {
            return intersect(available, names)  // doesn't call next
        }
    }
}
```

**Transform middlewares** (call next first, then modify the result):

```go
// MiddlewareExclude calls next, then removes named modules.
func MiddlewareExclude(names ...string) ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        return func(available []string) []string {
            selected := next(available)
            return filterOut(selected, names)
        }
    }
}

// MiddlewareAdd calls next, then appends named modules.
func MiddlewareAdd(names ...string) ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        return func(available []string) []string {
            selected := next(available)
            return append(selected, intersect(available, names)...)
        }
    }
}

// MiddlewareCustom calls next, then applies an arbitrary transformation.
func MiddlewareCustom(fn func(selected []string) []string) ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        return func(available []string) []string {
            return fn(next(available))
        }
    }
}
```

### Pipeline helper

```go
// Pipeline composes middlewares left-to-right: the first middleware in the list
// executes first, wrapping the subsequent ones.
func Pipeline(mws ...ModuleMiddleware) ModuleMiddleware {
    return func(next ModuleSelector) ModuleSelector {
        handler := next
        for i := len(mws) - 1; i >= 0; i-- {
            handler = mws[i](handler)
        }
        return handler
    }
}
```

### Builder integration

```go
func (b *FactoryBuilder) UseModuleMiddleware(mw ...ModuleMiddleware) *FactoryBuilder {
    b.assertMutable()
    b.moduleMiddlewares = append(b.moduleMiddlewares, mw...)
    return b
}
```

In `Build()`:

```go
selector := SelectAll
for i := len(b.moduleMiddlewares) - 1; i >= 0; i-- {
    selector = b.moduleMiddlewares[i](selector)
}
selected := selector(allRegisteredNames)

for _, name := range selected {
    specs = append(specs, DefaultRegistryModule(name))
}
```

### Usage examples

```go
// Safe mode (data-only modules)
engine.NewBuilder().UseModuleMiddleware(MiddlewareSafe())

// All except exec and os
engine.NewBuilder().UseModuleMiddleware(MiddlewareExclude("exec", "os"))

// Only fs and database
engine.NewBuilder().UseModuleMiddleware(MiddlewareOnly("fs", "database"))

// Safe + fs, no yaml (pipeline, order matters!)
engine.NewBuilder().UseModuleMiddleware(Pipeline(
    MiddlewareSafe(),
    MiddlewareAdd("fs"),
    MiddlewareExclude("yaml"),
))

// Custom transformation
engine.NewBuilder().UseModuleMiddleware(MiddlewareCustom(func(selected []string) []string {
    sort.Strings(selected)
    return unique(selected)
}))
```

## CLI flag mapping

| Flag | Middleware equivalent |
|------|----------------------|
| `--safe-mode` | `MiddlewareSafe()` |
| `--enable-module fs,db` | `MiddlewareOnly("fs", "database")` |
| `--disable-module fs` | `MiddlewareExclude("fs")` |
| (default) | none (SelectAll) |

### Validation rules

- `--enable-module` and `--disable-module` are mutually exclusive
- `--safe-mode` is mutually exclusive with both `--enable-module` and `--disable-module`
- Module names are validated against `modules.ListDefaultModules()` + known aliases

## Common Glazed schema

Add to `rootOptions` (shared by all commands via `commandSupport`):

```go
type rootOptions struct {
    DBPath             string
    PluginDirs         []string
    AllowPluginModules []string
    
    // NEW: module enablement
    EnableModules      []string  // --enable-module  (whitelist → MiddlewareOnly)
    DisableModules     []string  // --disable-module (blacklist → MiddlewareExclude)
    SafeMode           bool      // --safe-mode      (→ MiddlewareSafe)
}
```

Glazed field definitions:

```go
fields.New("enable-module", fields.TypeStringList,
    fields.WithHelp("Enable only these native modules (comma-separated). Implies all others are disabled."),
)
fields.New("disable-module", fields.TypeStringList,
    fields.WithHelp("Disable these native modules (comma-separated). All others remain enabled."),
)
fields.New("safe-mode", fields.TypeBool,
    fields.WithHelp("Load only data-only modules (crypto, events, path, time, timer). Disables fs, os, exec, database, yaml, process."),
    fields.WithDefault(false),
)
```

## Propagation paths

### Path A: CLI commands (run, tui, eval, create, etc.)

All commands go through `commandSupport.newAppWithOptions()` which builds the engine.

```go
func (s commandSupport) moduleMiddleware() ModuleMiddleware {
    if s.opts.SafeMode {
        return MiddlewareSafe()
    }
    if len(s.opts.EnableModules) > 0 {
        return MiddlewareOnly(s.opts.EnableModules...)
    }
    if len(s.opts.DisableModules) > 0 {
        return MiddlewareExclude(s.opts.DisableModules...)
    }
    return nil  // SelectAll (default)
}

func (s commandSupport) newAppWithOptions(options appSupportOptions) (*replapi.App, *repldb.Store, error) {
    builder := engine.NewBuilder().
        UseModuleMiddleware(s.moduleMiddleware())
    // ...
}
```

### Path B: run command (bypasses replapi.App)

`cmd_run.go` builds its own engine:

```go
func runScriptFile(ctx context.Context, opts runScriptOptions) error {
    builder := engine.NewBuilder().
        UseModuleMiddleware(opts.moduleMiddleware())
    // ...
}
```

### Path C: TUI evaluator

The evaluator Config needs the same fields:

```go
type Config struct {
    EnableModules      []string
    DisableModules     []string
    SafeMode           bool
    // ... existing fields
}
```

And in `New()`:

```go
mw := buildMiddlewareFromConfig(config)
builder := ggjengine.NewBuilder().UseModuleMiddleware(mw)
```

### Process stays orthogonal

`ProcessModule` and `ProcessEnv` remain separate concerns:

```go
engine.NewBuilder().
    UseModuleMiddleware(Pipeline(MiddlewareSafe(), MiddlewareAdd("fs"))).
    WithProcess()  // adds process module + global.process
```

## Deprecation plan

The old API family is deprecated but retained as thin wrappers:

| Old API | New equivalent |
|---------|---------------|
| `DefaultRegistryModules()` | `UseModuleMiddleware(nil)` (or omit) |
| `DataOnlyDefaultRegistryModules()` | `UseModuleMiddleware(MiddlewareSafe())` |
| `DefaultRegistryModulesNamed(names...)` | `UseModuleMiddleware(MiddlewareOnly(names...))` |
| `DefaultRegistryModule(name)` | `UseModuleMiddleware(MiddlewareOnly(name))` |

Mark old functions with `// Deprecated: Use UseModuleMiddleware instead.`

## Files to modify

| File | Change |
|------|--------|
| `engine/module_middleware.go` | NEW: ModuleSelector, ModuleMiddleware, built-ins, Pipeline |
| `engine/factory.go` | Add `UseModuleMiddleware`, integrate into Build |
| `engine/module_specs.go` | Deprecate old API family, add wrappers |
| `cmd/goja-repl/root.go` | Add module flags to rootOptions; add `moduleMiddleware()` helper; wire into newAppWithOptions |
| `cmd/goja-repl/cmd_run.go` | Add module fields to runScriptOptions; wire into runScriptFile |
| `pkg/repl/evaluators/javascript/evaluator.go` | Add module fields to Config; wire into builder |
| `cmd/goja-repl/root_test.go` | Add tests for module flag behavior |
| `pkg/doc/04-repl-usage.md` | Document module security model |

## Test plan

1. **Unit**: Middleware composition (Safe → Add → Exclude, pipeline order)
2. **Unit**: Middleware edge cases (empty names, unknown names, deduplication)
3. **Integration**: `go run ./cmd/goja-repl run --safe-mode ./testdata/yaml.js` → expect `yaml` require to fail
4. **Integration**: `go run ./cmd/goja-repl run --disable-module fs ./script-using-fs.js` → expect fs require to fail
5. **Integration**: `go run ./cmd/goja-repl run --enable-module fs,path ./script.js` → only fs and path available
6. **TUI smoke**: Start TUI with `--safe-mode`, verify `require("fs")` throws

## Open questions

1. Should `--safe-mode` also disable plugins? (Probably yes — plugins are arbitrary Go code)
2. Should we add a `--no-modules` flag to disable everything except core JS? (Nice-to-have; could be `MiddlewareOnly()` with no names)
3. How does this interact with `--allow-plugin-module`? (Orthogonal — plugin filtering stays separate)

## Key insight

The middleware pattern decouples **selection strategy** from **implementation**. Users can compose strategies arbitrarily: whitelist after blacklist, add then remove, or write entirely custom logic. The engine no longer needs a growing family of `DefaultRegistry*` functions — one method, infinite combinations.
