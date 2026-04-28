---
Title: Diary
Ticket: GOJA-059
Status: active
Topics:
    - backend
    - cli
    - security
    - go-go-golems
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/engine/module_specs.go:Engine module spec APIs
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/root.go:rootOptions and commandSupport shared app construction
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/cmd/goja-repl/cmd_run.go:run command with DefaultRegistryModules
    - /home/manuel/workspaces/2026-04-28/add-run-verb/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go:TUI evaluator runtime builder
ExternalSources: []
Summary: "Implementation diary for GOJA-059: adding granular module enablement via a ModuleMiddleware pipeline to goja-repl commands."
LastUpdated: 2026-04-28T09:15:00-04:00
WhatFor: "Chronological record of investigation, design, implementation, and validation"
WhenToUse: "When resuming work, reviewing decisions, or understanding why something was built a certain way"
---

# Diary — GOJA-059: Add granular module enablement flags to goja-repl commands

## Goal

Allow users to selectively enable or disable native Go modules (fs, exec, database, os, yaml, process) when using goja-repl `run`, `tui`, and other commands. The current behavior loads **all** modules unconditionally.

## Step 1: Investigation — Current module loading behavior

The user asked for the ability to load "dangerous" modules into the repl tui and run command. I first investigated whether these modules are already available.

### Prompt Context

**User prompt (verbatim):** "Ok, I want to be able to load some of the 'dangerous' modules into the repl tui and the run command, for example db/fs/exec, etc... How do I do that? I basically want to be able to open a sqlite db and write to disk from the repl or from a script to test things. Create a new docmgr ticket about loading modules for the run/tui methods (and other goja-repl verbs, tbh. Maybe we can make a common glazed schema for that and handle it as a base part of the framework, for jsverbs too for example)"

**Assistant interpretation:** The user wants CLI flags to control which native Go modules are loaded when running JS through goja-repl. They want this as a reusable pattern across all commands.

**Inferred user intent:** Make module loading granular and controllable from the CLI, not just all-or-nothing. This improves security and gives users explicit control over the sandbox boundary.

### What I did

- Read `engine/module_specs.go` to understand module registration APIs:
  - `DefaultRegistryModules()` — loads ALL modules (safe + dangerous)
  - `DataOnlyDefaultRegistryModules()` — loads only safe modules (crypto, events, path, time, timer)
  - `DefaultRegistryModulesNamed(names...)` — loads only named modules
  - `DefaultRegistryModule(name)` — loads a single module
- Read `cmd/goja-repl/cmd_run.go` — confirms `run` uses `DefaultRegistryModules()`
- Read `cmd/goja-repl/root.go` — confirms `newAppWithOptions` (used by tui, eval, create, etc.) also uses `DefaultRegistryModules()`
- Read `pkg/repl/evaluators/javascript/evaluator.go` — confirms TUI evaluator also uses `DefaultRegistryModules()`
- Created test scripts in `/tmp/` and ran them with `go run ./cmd/goja-repl run ...`:
  - `test-fs.js` — `require("fs")` ✅ works (writes/reads `/tmp/goja-test.txt`)
  - `test-db.js` — `require("database")` ✅ works
  - `test-exec.js` — `require("exec")` ✅ works
- Checked `goja-repl tui --help` — no module-related flags exist

### What worked

- The dangerous modules (fs, db, exec, os, yaml) are **already loaded** by both `run` and `tui` today.
- The engine already has all the APIs needed for granular loading.
- Test scripts confirmed modules are functional.

### What didn't work

- There is **no way to selectively disable** modules. It's all-or-nothing.
- There is **no way to run in safe mode** (data-only modules) from the CLI.
- The `--safe-mode` concept exists in the engine (`DataOnlyDefaultRegistryModules`) but is not exposed to users.

### What I learned

- The engine is already well-designed for sandboxing — `DataOnlyDefaultRegistryModules()` exists, `DefaultRegistryModulesNamed()` exists.
- The gap is purely in **CLI flag plumbing** and **propagation to all code paths** that build engines.
- There are **three independent code paths** that build engines:
  1. `commandSupport.newAppWithOptions()` — for replapi-backed commands (tui, eval, create, etc.)
  2. `runScriptFile()` — for the `run` command (standalone, no replapi)
  3. `javascript.New()` — for the TUI evaluator (standalone, no replapi)
- All three currently hardcode `DefaultRegistryModules()`.

### What was tricky to build

N/A — this was investigation only.

### What warrants a second pair of eyes

N/A — investigation only.

### What should be done in the future

- Step 2: Design the module enablement API
- Step 3: Implement the API in the engine
- Step 4: Wire through CLI flags to all three engine-building paths
- Step 5: Add tests for module filtering
- Step 6: Update documentation

### Code review instructions

N/A — no code changes yet.

### Technical details

**Available modules:**
```
crypto, node:crypto    → safe (data-only)
events, node:events    → safe (data-only)
path, node:path        → safe (data-only)
time                   → safe (data-only)
timer                  → safe (data-only)
fs, node:fs            → dangerous (filesystem)
os, node:os            → dangerous (system info)
exec                   → dangerous (subprocess)
database               → dangerous (SQLite)
yaml                   → dangerous (file I/O via fs helpers)
process, node:process  → dangerous (env vars)
```

**Engine builder patterns:**
```go
// Current (all modules)
engine.NewBuilder().WithModules(engine.DefaultRegistryModules())

// Safe mode (data only)
engine.NewBuilder().WithModules(engine.DataOnlyDefaultRegistryModules())

// Named modules
engine.NewBuilder().WithModules(engine.DefaultRegistryModulesNamed("fs", "path"))
```

## Step 2: Design — From Strategy Enums to Middleware Pipeline

### Prompt Context

**User prompt (verbatim):** "the whole set of engine APIs is a bit baroque and could be unified, right? The engine.DefaultRegistryModules estc..."

**Assistant interpretation:** The user wants a cleaner, unified API for module selection instead of the current grab-bag of four functions.

**Inferred user intent:** Replace `DefaultRegistryModules()`, `DataOnlyDefaultRegistryModules()`, `DefaultRegistryModulesNamed()`, and `DefaultRegistryModule()` with a single, composable mechanism.

### What I did

- Proposed a `ModuleSelection` struct with a `SelectionStrategy` enum (All/Safe/Only/Except)
- User pushed back: "can we chain multiple selections? Could ProcessModule and ProcessEnv just be two modules too?"
- Proposed a `ModuleFilter` pipeline (pure functions over `[]string`)
- User pushed for the `f(next Handler) Handler` pattern for explicit control flow
- Landed on `ModuleMiddleware func(next ModuleSelector) ModuleSelector`
- Updated the design doc with the full middleware pipeline approach

### What worked

- The `f(next) Handler` pattern maps cleanly to module selection:
  - **Override middlewares** (Safe, Only) don't call next → replace entire selection
  - **Transform middlewares** (Exclude, Add, Custom) call next → modify result
- `Pipeline()` helper composes middlewares left-to-right
- Old API can be deprecated as thin wrappers
- CLI flags map 1:1 to middlewares:
  - `--safe-mode` → `MiddlewareSafe()`
  - `--enable-module fs,db` → `MiddlewareOnly("fs", "database")`
  - `--disable-module fs` → `MiddlewareExclude("fs")`

### What didn't work

- Initial `ModuleSelection` enum approach was rejected as not composable enough
- `ModuleFilter` pipeline was close but lacked explicit control flow
- The `f(next Handler) Handler` pattern was the right abstraction

### What I learned

- The middleware pattern gives users **full control flow**: short-circuit, pre/post transform, arbitrary composition
- Order matters: `Pipeline(Safe, Add("fs"))` is different from `Pipeline(Add("fs"), Safe)`
- `ProcessModule` and `ProcessEnv` **cannot** be folded into module middleware because:
  - `ProcessModule` is a `ModuleSpec` (factory-time registration)
  - `ProcessEnv` is a `RuntimeInitializer` (per-runtime global variable)
  - They stay orthogonal via `WithProcess()` builder method

### What was tricky to build

N/A — design discussion, no code yet.

### What warrants a second pair of eyes

- The `Pipeline` composition order: first middleware executes first, wrapping subsequent ones. This is standard HTTP middleware behavior but could be surprising.
- Deprecation strategy: old functions will be thin wrappers calling `UseModuleMiddleware`. Need to ensure no behavioral changes.

### What should be done in the future

- Step 3: Implement `engine/module_middleware.go` with core types and built-ins
- Step 4: Add `UseModuleMiddleware` to `FactoryBuilder`
- Step 5: Deprecate old API family in `module_specs.go`
- Step 6: Wire CLI flags through all three engine-building paths
- Step 7: Write tests

### Code review instructions

N/A — design only.

### Technical details

**Core types:**
```go
type ModuleSelector func(available []string) []string
type ModuleMiddleware func(next ModuleSelector) ModuleSelector
```

**Built-in middlewares:**
```go
func MiddlewareSafe() ModuleMiddleware       // override: data-only
func MiddlewareOnly(names...) ModuleMiddleware // override: whitelist
func MiddlewareExclude(names...) ModuleMiddleware // transform: blacklist
func MiddlewareAdd(names...) ModuleMiddleware     // transform: append
func MiddlewareCustom(fn) ModuleMiddleware        // transform: arbitrary
```

**Usage:**
```go
// Safe + fs, no yaml
engine.NewBuilder().UseModuleMiddleware(Pipeline(
    MiddlewareSafe(),
    MiddlewareAdd("fs"),
    MiddlewareExclude("yaml"),
))

// All except exec
engine.NewBuilder().UseModuleMiddleware(MiddlewareExclude("exec"))

// Only fs and db
engine.NewBuilder().UseModuleMiddleware(MiddlewareOnly("fs", "database"))
```
