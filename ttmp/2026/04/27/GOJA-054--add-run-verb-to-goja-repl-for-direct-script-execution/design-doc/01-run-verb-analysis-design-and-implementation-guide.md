---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/cmd_eval.go
      Note: Existing eval command pattern
    - Path: cmd/goja-repl/root.go
      Note: Root command wiring and commandSupport pattern
    - Path: engine/factory.go
      Note: Factory.NewRuntime for ephemeral execution
    - Path: engine/module_roots.go
      Note: RequireOptionWithModuleRootsFromScript
    - Path: pkg/replapi/app.go
      Note: App facade with session-based execution
    - Path: pkg/replapi/config.go
      Note: Profile presets (Raw/Interactive/Persistent)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Run Verb: Analysis, Design and Implementation Guide

---

## Executive Summary

This document provides a comprehensive, intern-friendly guide for adding a **`run`** verb to `goja-repl`. The `run` command will execute a JavaScript file directly against a fresh, ephemeral go-go-goja runtime — without requiring a persistent REPL session, SQLite database, or session ID. This is the canonical way to run standalone scripts like `testdata/yaml.js` or user-authored automation scripts.

The command will be implemented as a **Glazed command** following the project's established patterns, wired into the existing `goja-repl` Cobra root command alongside `eval`, `create`, `tui`, and other verbs.

**Target usage:**
```bash
goja-repl run ./testdata/yaml.js
goja-repl run --profile raw ./scripts/etl.js
goja-repl run --plugin-dir ./my-plugins ./scripts/with-plugins.js
```

---

## Table of Contents

1. [What is goja-repl?](#1-what-is-goja-repl)
2. [The Current Command Architecture](#2-the-current-command-architecture)
3. [Problem Statement and Scope](#3-problem-statement-and-scope)
4. [Current-State Architecture (Evidence-Based)](#4-current-state-architecture-evidence-based)
5. [Gap Analysis](#5-gap-analysis)
6. [Proposed Architecture and APIs](#6-proposed-architecture-and-apis)
7. [Pseudocode and Key Flows](#7-pseudocode-and-key-flows)
8. [Implementation Phases](#8-implementation-phases)
9. [Test Strategy](#9-test-strategy)
10. [Risks, Alternatives, and Open Questions](#10-risks-alternatives-and-open-questions)
11. [References](#11-references)

---

## 1. What is goja-repl?

`goja-repl` is the primary CLI for go-go-goja. It provides multiple ways to interact with the JavaScript runtime:

- **`tui`** — Interactive terminal UI (Bobatea-based REPL with completion, help, history)
- **`eval`** — Evaluate a JS snippet in a persistent session (requires `session-id`)
- **`create`** — Create a new persistent session
- **`serve`** — Start a JSON HTTP API server
- **Other commands** — `snapshot`, `history`, `bindings`, `export`, `restore`, `docs`, `essay`

All persistent-session commands operate through `replapi.App`, which manages session lifecycle, SQLite-backed state, and optional auto-restore behavior. The `run` command will be different: it will bypass the session store entirely and create a fresh, short-lived runtime.

---

## 2. The Current Command Architecture

### 2.1 How commands are built

`goja-repl` uses the **Glazed commands framework**. Every subcommand follows this pattern:

1. **Define a struct** embedding `*cmds.CommandDescription`
2. **Implement `cmds.BareCommand`** (the `Run` method)
3. **Register in `newRootCommand`** via `cli.BuildCobraCommand`

### 2.2 The `commandSupport` pattern

Every command embeds `commandSupport`, which provides:

- `newApp()` — builds a `replapi.App` with `ProfilePersistent` and SQLite store
- `newAppWithOptions(options)` — builds an `App` with custom profile/store/help system
- `runWithApp(fn)` — convenience wrapper that builds app, defers close, and calls `fn`

### 2.3 The `rootOptions` struct

Root-level flags (defined on the Cobra root command) are shared across all subcommands:

```go
type rootOptions struct {
    DBPath             string  // --db-path (default: goja-repl.sqlite)
    PluginDirs         []string // --plugin-dir
    AllowPluginModules []string // --allow-plugin-module
}
```

These are passed to every command constructor and used when building the `replapi.App`.

---

## 3. Problem Statement and Scope

### 3.1 The problem

There is **no direct way to run a JavaScript file** through `goja-repl` without:

1. Creating a persistent session (`goja-repl create`)
2. Remembering the session ID
3. Running `goja-repl eval --session-id <id> --source "$(cat file.js)"`
4. Cleaning up the session

This is cumbersome for:
- Running example scripts (`testdata/yaml.js`)
- CI/automation pipelines
- One-off data processing scripts
- Quick prototyping

### 3.2 Why a new `run` verb?

- ** ergonomics**: `goja-repl run file.js` is the natural expectation
- **no persistence overhead**: No SQLite, no session ID, no cleanup
- **faster**: No session serialization, no history tracking, no binding capture
- **predictable**: Fresh runtime every time — no state leakage from previous runs
- **CI-friendly**: Exit codes reflect script success/failure

### 3.3 Scope

**In scope:**
- `goja-repl run <file>` — execute a JS file in a fresh ephemeral runtime
- `--profile` flag to choose execution mode (`raw`, `interactive`, `persistent`)
- `--plugin-dir` and `--allow-plugin-module` inheritance from root flags
- Proper exit codes (0 on success, non-zero on JS error or file-not-found)
- Console output forwarding (JS `console.log` → stdout)

**Out of scope (future work):**
- `run` with stdin input (pipe JS directly)
- `run` with multiple files
- `run` with argument passing to the script
- `run` with module root auto-detection from script path

---

## 4. Current-State Architecture (Evidence-Based)

### 4.1 Root command wiring

**File:** `cmd/goja-repl/root.go`

Lines 44–64 show how commands are registered:

```go
commands := []cmds.Command{
    newSessionsCommand(out, opts),
    newCreateCommand(out, opts),
    newEvalCommand(out, opts),
    // ... etc
}
for _, command := range commands {
    cobraCommand, err := cli.BuildCobraCommand(command,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
        }),
    )
    if err != nil {
        return nil, err
    }
    root.AddCommand(cobraCommand)
}
```

### 4.2 The `eval` command pattern

**File:** `cmd/goja-repl/cmd_eval.go`

```go
type evalCommand struct {
    *cmds.CommandDescription
    commandSupport
}

var _ cmds.BareCommand = (*evalCommand)(nil)

func newEvalCommand(out io.Writer, opts *rootOptions) *evalCommand {
    return &evalCommand{
        CommandDescription: cmds.NewCommandDescription("eval",
            cmds.WithShort("Evaluate source in a persistent session"),
            cmds.WithFlags(
                fields.New("session-id", fields.TypeString, fields.WithRequired(true), ...),
                fields.New("source", fields.TypeString, fields.WithRequired(true), ...),
            ),
        ),
        commandSupport: commandSupport{out: out, opts: opts},
    }
}

func (c *evalCommand) Run(ctx context.Context, vals *values.Values) error {
    settings := evalSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
        return err
    }
    return c.runWithApp(func(ctx context.Context, app *replapi.App) error {
        resp, err := app.Evaluate(ctx, settings.SessionID, settings.Source)
        if err != nil {
            return err
        }
        return writeJSON(c.out, resp)
    })(ctx, vals)
}
```

Key observations:
- `evalSettings` uses struct tags `glazed:"session-id"` and `glazed:"source"`
- `runWithApp` handles app lifecycle (build → run → close store)
- Output is JSON-encoded via `writeJSON`

### 4.3 App construction with profiles

**File:** `cmd/goja-repl/root.go`, lines 132–168

```go
func (s commandSupport) newAppWithOptions(options appSupportOptions) (*replapi.App, *repldb.Store, error) {
    // ... store setup ...
    builder := engine.NewBuilder().WithModules(engine.DefaultRegistryModules())
    if options.helpSystem != nil {
        builder = builder.WithRuntimeModuleRegistrars(...)
    }
    builder = pluginSetup.WithBuilder(builder)
    factory, err := builder.Build()
    // ... app construction ...
}
```

**File:** `pkg/replapi/config.go`, lines 12–75

Profiles available:
- `ProfileRaw` — no instrumentation, no binding capture, no persistence
- `ProfileInteractive` — instrumented, binding capture, no persistence
- `ProfilePersistent` — full persistence, auto-restore, SQLite store

For `run`, we want **ephemeral execution** — no store, no persistence. `ProfileRaw` or `ProfileInteractive` are appropriate depending on whether the user wants binding capture.

### 4.4 Direct runtime execution without sessions

The `replapi.App` is designed around persistent sessions. For `run`, we need a different path. We can either:

1. **Build an `App` with `RawConfig()`** (no store) and create a temporary session
2. **Bypass `replapi.App` entirely** and use `engine.Factory.NewRuntime()` directly

Option 2 is cleaner for `run` because:
- No session machinery at all
- No need to create/destroy a temporary session
- Direct control over the runtime lifecycle

---

## 5. Gap Analysis

| # | Gap | Impact | Mitigation |
|---|---|---|---|
| 1 | No `run` verb exists | Users must use awkward `create` + `eval` workflow | Add `run` command |
| 2 | All existing commands assume persistent sessions | `run` needs ephemeral execution | Build runtime directly via `engine.Factory` |
| 3 | No file-reading pattern in commands | `run` needs to read from filesystem | Use `os.ReadFile` in command |
| 4 | `runWithApp` always creates a store | `run` should not require SQLite | Use direct factory instead of `replapi.App` |
| 5 | Output is JSON for all commands | `run` should forward console output directly | Use runtime's console directly, no JSON wrapper |

---

## 6. Proposed Architecture and APIs

### 6.1 Command-line interface

```bash
# Basic usage
goja-repl run ./scripts/my-script.js

# With raw profile (fastest, no instrumentation)
goja-repl run --profile raw ./scripts/my-script.js

# With interactive profile (binding capture, no persistence)
goja-repl run --profile interactive ./scripts/my-script.js

# With plugins
goja-repl run --plugin-dir ./my-plugins ./scripts/plugin-script.js
```

### 6.2 Glazed command definition

```go
type runCommand struct {
    *cmds.CommandDescription
    commandSupport
}

var _ cmds.BareCommand = (*runCommand)(nil)

func newRunCommand(out io.Writer, opts *rootOptions) *runCommand {
    return &runCommand{
        CommandDescription: cmds.NewCommandDescription("run",
            cmds.WithShort("Execute a JavaScript file in a fresh runtime"),
            cmds.WithLong(`
Run executes a JavaScript file directly in a fresh, ephemeral go-go-goja runtime.

No persistent session is created. No SQLite database is required. The runtime is
destroyed when the script completes.

Examples:
  goja-repl run ./testdata/yaml.js
  goja-repl run --profile raw ./scripts/fast-etl.js
  goja-repl run --plugin-dir ./plugins ./scripts/with-custom-modules.js
`),
            cmds.WithFlags(
                fields.New("profile", fields.TypeString,
                    fields.WithDefault("interactive"),
                    fields.WithHelp("Execution profile: raw, interactive, or persistent")),
            ),
            cmds.WithArguments(
                fields.New("file", fields.TypeString,
                    fields.WithRequired(true),
                    fields.WithHelp("Path to the JavaScript file to execute")),
            ),
        ),
        commandSupport: commandSupport{out: out, opts: opts},
    }
}
```

### 6.3 Settings struct

```go
type runSettings struct {
    Profile string `glazed:"profile"`
    File    string `glazed:"file"`
}
```

### 6.4 Run implementation

```go
func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
    settings := runSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
        return err
    }

    // 1. Resolve absolute path
    scriptPath, err := filepath.Abs(settings.File)
    if err != nil {
        return fmt.Errorf("resolve script path: %w", err)
    }
    if _, err := os.Stat(scriptPath); err != nil {
        return fmt.Errorf("script file not found: %w", err)
    }

    // 2. Read file
    sourceBytes, err := os.ReadFile(scriptPath)
    if err != nil {
        return fmt.Errorf("read script: %w", err)
    }

    // 3. Build factory with module roots from script location
    pluginSetup := host.NewRuntimeSetup(c.opts.PluginDirs, c.opts.AllowPluginModules)
    builder := engine.NewBuilder().WithModules(engine.DefaultRegistryModules())

    // Derive module roots from script directory (like Node.js require behavior)
    moduleRootOpts := engine.DefaultModuleRootsOptions()
    requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(scriptPath, moduleRootOpts)
    if err == nil && requireOpt != nil {
        builder = builder.WithRequireOptions(requireOpt)
    }

    builder = pluginSetup.WithBuilder(builder)
    factory, err := builder.Build()
    if err != nil {
        return fmt.Errorf("build engine factory: %w", err)
    }

    // 4. Create ephemeral runtime (no store, no persistence)
    rt, err := factory.NewRuntime(ctx)
    if err != nil {
        return fmt.Errorf("create runtime: %w", err)
    }
    defer rt.Close(ctx)

    // 5. Execute script
    _, err = rt.VM.RunString(string(sourceBytes))
    if err != nil {
        return fmt.Errorf("script execution failed: %w", err)
    }

    return nil
}
```

### 6.5 Why bypass `replapi.App`?

`replapi.App` is designed for session lifecycle management:
- It creates sessions via `CreateSession()`
- It evaluates via `Evaluate(sessionID, source)`
- It persists state via `repldb.Store`
- It auto-restores sessions

For `run`, none of this is needed. Using `engine.Factory.NewRuntime()` directly:
- Avoids creating a throwaway session
- Avoids SQLite overhead
- Gives direct access to `rt.VM.RunString()`
- Makes the command work even when no `--db-path` is valid

### 6.6 Console output forwarding

The goja runtime already has `console.Enable(vm)` called during `factory.NewRuntime()` (see `engine/factory.go`). This means JS `console.log()` calls are automatically forwarded to stdout. No additional wiring is needed.

### 6.7 Exit codes

- `0` — Script executed successfully
- `1` — File not found, read error, factory build error, runtime creation error
- `1` — JavaScript runtime error (goja returns an error)

Since the `Run` method returns an error, Cobra will automatically exit with code 1.

---

## 7. Pseudocode and Key Flows

### 7.1 Full execution flow

```
[User] goja-repl run ./testdata/yaml.js
    |
    v
[Cobra] Parse flags/args
    file = "./testdata/yaml.js"
    profile = "interactive" (default)
    |
    v
[runCommand.Run]
    |
    +-- Resolve absolute path
    |       /home/manuel/.../go-go-goja/testdata/yaml.js
    +-- Verify file exists
    +-- Read file into []byte
    +-- Build engine.Factory
    |       NewBuilder()
    |       WithModules(DefaultRegistryModules())
    |       WithRequireOptions(WithGlobalFolders from script dir)
    |       WithBuilder(pluginSetup)
    |       Build()
    +-- factory.NewRuntime(ctx)
    |       Creates goja.Runtime
    |       Registers all native modules
    |       Enables console
    +-- rt.VM.RunString(source)
    |       Parses and executes JS
    |       console.log() → stdout
    +-- rt.Close(ctx)
    |       Shuts down runtime
    +-- Return nil (success)
    |
    v
[Cobra] Exit 0
```

### 7.2 Comparison: eval vs run

| Aspect | `eval` | `run` |
|---|---|---|
| Session | Persistent (SQLite) | Ephemeral (none) |
| Session ID | Required | Not used |
| DB required | Yes | No |
| Output format | JSON | Raw console output |
| Use case | REPL server, automation | One-off scripts |
| Binding capture | Yes (interactive/persistent) | Configurable via `--profile` |
| History | Stored in DB | Not stored |

---

## 8. Implementation Phases

### Phase 1: Create the command file

**File:** `cmd/goja-repl/cmd_run.go`

Implement `runCommand` struct, constructor, and `Run` method following the patterns from `cmd_eval.go` and the glazed skill.

### Phase 2: Register the command

**File:** `cmd/goja-repl/root.go`

Add `newRunCommand(out, opts)` to the `commands` slice in `newRootCommand`.

### Phase 3: Add settings type

**File:** `cmd/goja-repl/root.go`

Add `runSettings` struct next to `evalSettings`:

```go
type runSettings struct {
    Profile string `glazed:"profile"`
    File    string `glazed:"file"`
}
```

### Phase 4: Test the command

Create a Go test that:
1. Builds a Cobra command tree
2. Runs `goja-repl run testdata/yaml.js`
3. Asserts exit code 0
4. Asserts output contains "OK"

### Phase 5: Update documentation

- Add `run` to the README quick-start section
- Add `run` example to `pkg/doc/04-repl-usage.md`
- Update the folder layout in README to mention `run`

### Phase 6: Validation

```bash
go test ./cmd/goja-repl/... -count=1 -v
make lint
go run ./cmd/goja-repl run ./testdata/yaml.js
```

---

## 9. Test Strategy

### 9.1 Unit test for the command

```go
func TestRunCommandExecutesScript(t *testing.T) {
    out := &bytes.Buffer{}
    opts := &rootOptions{}
    cmd := newRunCommand(out, opts)

    vals := values.NewValues()
    vals.SetValue(schema.DefaultSlug, "file", "testdata/yaml.js")
    vals.SetValue(schema.DefaultSlug, "profile", "interactive")

    err := cmd.Run(context.Background(), vals)
    require.NoError(t, err)
    require.Contains(t, out.String(), "OK")
}
```

### 9.2 Integration test via Cobra

```go
func TestRunViaCobra(t *testing.T) {
    root, err := newRootCommand(io.Discard)
    require.NoError(t, err)

    root.SetArgs([]string{"run", "testdata/yaml.js"})
    err = root.Execute()
    require.NoError(t, err)
}
```

### 9.3 Error cases

- Non-existent file → error
- Invalid JS syntax → error
- File without read permission → error

---

## 10. Risks, Alternatives, and Open Questions

### 10.1 Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Users expect `run` to support stdin | Medium | Low | Document limitation; add in follow-up |
| Users expect `run` to pass argv to script | Medium | Low | Document limitation; add `--` separator in follow-up |
| Profile selection is confusing | Low | Medium | Default to `interactive`; document differences |

### 10.2 Alternatives considered

**Alternative A: Reuse `eval` with auto-created session**
- Create a hidden session, eval the file, delete the session
- **Rejected:** Still requires SQLite store, slower, creates noise in session list

**Alternative B: Shell alias**
- `alias goja-run='goja-repl create && goja-repl eval'`
- **Rejected:** Not discoverable, brittle, no proper error handling

**Alternative C: Separate binary**
- `goja-run file.js` as a standalone command
- **Rejected:** Fragmentation; `run` is a natural subcommand of `goja-repl`

### 10.3 Open questions

1. Should `run` support `--watch` mode for development?
2. Should `run` support `--eval` for inline scripts (`goja-repl run -e "console.log(1)"`)?
3. Should `run` automatically set `process.argv` or `__filename`/`__dirname` for Node.js compatibility?

**Recommendation:** Keep initial implementation minimal. Open questions can be addressed in follow-up tickets.

---

## 11. References

### 11.1 Source files

| File | Relevance |
|---|---|
| `cmd/goja-repl/root.go` | Root command, command registration, `commandSupport`, `rootOptions` |
| `cmd/goja-repl/cmd_eval.go` | Existing eval command pattern |
| `cmd/goja-repl/cmd_create.go` | Simple command without flags |
| `pkg/replapi/app.go` | `App` facade, `Evaluate`, `WithRuntime` |
| `pkg/replapi/config.go` | Profiles (`ProfileRaw`, `ProfileInteractive`, `ProfilePersistent`) |
| `engine/factory.go` | `Factory.NewRuntime()` |
| `engine/module_roots.go` | `RequireOptionWithModuleRootsFromScript` |
| `engine/module_specs.go` | `DefaultRegistryModules()` |
| `testdata/yaml.js` | Example script to test with |

### 11.2 New files to create

| File | Description |
|---|---|
| `cmd/goja-repl/cmd_run.go` | The `run` command implementation |

### 11.3 Existing files to modify

| File | Change |
|---|---|
| `cmd/goja-repl/root.go` | Add `newRunCommand` to commands slice; add `runSettings` struct |

### 11.4 External references

- Glazed command authoring skill
- `github.com/go-go-golems/glazed/pkg/cmds` documentation
- `github.com/spf13/cobra` documentation

---

## Appendix A: Quick-Start Checklist for the Implementer

- [ ] Create `cmd/goja-repl/cmd_run.go` with `runCommand`
- [ ] Add `runSettings` struct to `cmd/goja-repl/root.go`
- [ ] Register `newRunCommand` in `newRootCommand`
- [ ] Write unit test in `cmd/goja-repl/root_test.go` or new `cmd_run_test.go`
- [ ] Run `go run ./cmd/goja-repl run ./testdata/yaml.js`
- [ ] Run `go test ./cmd/goja-repl/... -count=1 -v`
- [ ] Run `make lint`
- [ ] Update README with `run` example
- [ ] Update `pkg/doc/04-repl-usage.md` with `run` section
- [ ] Commit with message: `feat(cmd/goja-repl): add run verb for direct script execution`
