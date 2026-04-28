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
    - Path: cmd/goja-repl/cmd_run.go
      Note: run command builder with DefaultRegistryModules
    - Path: cmd/goja-repl/cmd_run.go:run command with DefaultRegistryModules
    - Path: cmd/goja-repl/root.go
      Note: rootOptions and commandSupport shared app construction
    - Path: cmd/goja-repl/root.go:rootOptions and shared app construction
    - Path: engine/module_specs.go
      Note: Engine module spec APIs (DefaultRegistryModules
    - Path: engine/module_specs.go:Engine module spec APIs
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: TUI evaluator Config and runtime builder
    - Path: pkg/repl/evaluators/javascript/evaluator.go:TUI evaluator runtime builder
ExternalSources: []
Summary: 'Implementation diary for GOJA-059: adding granular module enablement flags to goja-repl commands.'
LastUpdated: 2026-04-28T09:00:00-04:00
WhatFor: Chronological record of investigation, design, implementation, and validation
WhenToUse: When resuming work, reviewing decisions, or understanding why something was built a certain way
---





# Diary — GOJA-059: Add granular module enablement flags to goja-repl commands

## Goal

Allow users to selectively enable or disable native Go modules (fs, exec, database, os, yaml, process) when using goja-repl `run`, `tui`, and other commands. The current behavior loads **all** modules unconditionally, which is convenient but not always safe.

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

- Step 2: Design the common Glazed schema for module enablement flags
- Step 3: Implement the schema in `rootOptions` and propagate through all three engine-building paths
- Step 4: Add tests for module filtering
- Step 5: Update documentation

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
