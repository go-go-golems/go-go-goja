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
Summary: "Design for adding granular module enablement flags (--enable-module, --disable-module, --safe-mode) to all goja-repl commands, with a common Glazed schema shared across run, tui, eval, create, and future jsverbs."
LastUpdated: 2026-04-28T09:00:00-04:00
WhatFor: "Reference design for implementing module security controls in goja-repl"
WhenToUse: "When adding flags, updating builder logic, or wiring module filtering into commands"
---

# Module Enablement Design (GOJA-059)

## Goal

Allow users to selectively enable or disable native Go modules when running JS through goja-repl. Today, `run` and `tui` unconditionally load **all** modules via `engine.DefaultRegistryModules()`. We want:

- `--enable-module fs,exec` — load only specific modules (whitelist)
- `--disable-module fs,exec` — load all except specific modules (blacklist)
- `--safe-mode` — load only `DataOnlyDefaultRegistryModules()` (crypto, events, path, time, timer)
- Default behavior stays as-is (all modules) for backward compatibility

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

### Engine APIs already available

```go
engine.DefaultRegistryModules()                          // all modules
engine.DataOnlyDefaultRegistryModules()                  // safe only
engine.DefaultRegistryModulesNamed("fs", "os")          // named only
engine.DefaultRegistryModule("fs")                       // single module
engine.ProcessModule()                                   // process require()
engine.ProcessEnv()                                      // global process
```

## Proposed Design

### 1. Common Glazed schema for module enablement

Add to `rootOptions` (shared by all commands via `commandSupport`):

```go
type rootOptions struct {
    DBPath             string
    PluginDirs         []string
    AllowPluginModules []string
    
    // NEW: module enablement
    EnableModules      []string  // --enable-module  (whitelist)
    DisableModules     []string  // --disable-module (blacklist)
    SafeMode           bool      // --safe-mode
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

**Validation rules:**
- `--enable-module` and `--disable-module` are mutually exclusive
- `--safe-mode` is mutually exclusive with both `--enable-module` and `--disable-module`
- Module names are validated against `modules.ListDefaultModules()` + known aliases

### 2. Module selection logic

```go
func selectModules(enable, disable []string, safeMode bool) []engine.ModuleSpec {
    if safeMode {
        return []engine.ModuleSpec{engine.DataOnlyDefaultRegistryModules()}
    }
    if len(enable) > 0 {
        return []engine.ModuleSpec{engine.DefaultRegistryModulesNamed(enable...)}
    }
    if len(disable) > 0 {
        all := allModuleNames()
        allowed := filterOut(all, disable)
        return []engine.ModuleSpec{engine.DefaultRegistryModulesNamed(allowed...)}
    }
    return []engine.ModuleSpec{engine.DefaultRegistryModules()}
}
```

### 3. Propagation paths

**Path A: CLI commands (run, tui, eval, create, etc.)**

All commands go through `commandSupport.newAppWithOptions()` which builds the engine. We change:

```go
// root.go
func (s commandSupport) moduleSpecs() []engine.ModuleSpec {
    return selectModules(s.opts.EnableModules, s.opts.DisableModules, s.opts.SafeMode)
}

func (s commandSupport) newAppWithOptions(options appSupportOptions) (*replapi.App, *repldb.Store, error) {
    // ...
    builder := engine.NewBuilder().WithModules(s.moduleSpecs()...)
    // ...
}
```

**Path B: run command (bypasses replapi.App)**

`cmd_run.go` builds its own engine. Change:

```go
func runScriptFile(ctx context.Context, opts runScriptOptions) error {
    moduleSpecs := selectModules(opts.EnableModules, opts.DisableModules, opts.SafeMode)
    builder := engine.NewBuilder().WithModules(moduleSpecs...)
    // ...
}
```

**Path C: TUI evaluator**

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
moduleSpecs := selectModules(config.EnableModules, config.DisableModules, config.SafeMode)
builder := ggjengine.NewBuilder().WithModules(moduleSpecs...)
```

### 4. jsverbs integration (future)

The `jsverbs` subsystem (`pkg/jsverbs`) currently invokes JS through a caller-provided runtime. It does not control module loading. However, the registry could accept module enablement settings:

```go
type RegistryConfig struct {
    EnableModules  []string
    DisableModules []string
    SafeMode       bool
}
```

When `Registry.Commands()` builds command descriptions, it could inject the module flags into each verb's Glazed parameters, and the invoker would pass them to the runtime builder.

**This is out of scope for GOJA-059** — noted as follow-up.

## Files to modify

| File | Change |
|------|--------|
| `cmd/goja-repl/root.go` | Add module flags to rootOptions; add moduleSpecs() helper; wire into newAppWithOptions |
| `cmd/goja-repl/cmd_run.go` | Add module fields to runScriptOptions; wire into runScriptFile |
| `pkg/repl/evaluators/javascript/evaluator.go` | Add module fields to Config; wire into builder |
| `engine/module_specs.go` | Add `AllModuleNames()` helper or expand existing APIs |
| `cmd/goja-repl/root_test.go` | Add tests for module flag behavior |
| `pkg/doc/04-repl-usage.md` | Document module security model |

## Test plan

1. **Unit**: `selectModules()` with all combinations of enable/disable/safe/none
2. **Integration**: `go run ./cmd/goja-repl run --safe-mode ./testdata/yaml.js` → expect `yaml` require to fail
3. **Integration**: `go run ./cmd/goja-repl run --disable-module fs ./script-using-fs.js` → expect fs require to fail
4. **Integration**: `go run ./cmd/goja-repl run --enable-module fs,path ./script.js` → only fs and path available
5. **TUI smoke**: Start TUI with `--safe-mode`, verify `require("fs")` throws

## Open questions

1. Should `--safe-mode` also disable plugins? (Probably yes — plugins are arbitrary Go code)
2. Should we add a `--no-modules` flag to disable everything except core JS? (Nice-to-have)
3. How does this interact with `--allow-plugin-module`? (Orthogonal — plugin filtering stays separate)
