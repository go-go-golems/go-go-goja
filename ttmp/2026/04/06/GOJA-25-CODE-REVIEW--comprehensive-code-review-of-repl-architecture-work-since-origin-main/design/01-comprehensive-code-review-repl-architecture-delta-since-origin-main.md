---
Title: Comprehensive Code Review REPL Architecture Delta Since Origin Main
Ticket: GOJA-25-CODE-REVIEW
Status: active
Topics:
    - code-review
    - architecture
    - repl
    - persistent-repl
    - sqlite
    - tui
    - cli
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/replsession/service.go
      Note: Core session kernel
    - Path: pkg/replsession/types.go
      Note: Transport-neutral DTOs
    - Path: pkg/replsession/policy.go
      Note: Session policy model
    - Path: pkg/repldb/store.go
      Note: SQLite store bootstrap
    - Path: pkg/replapi/app.go
      Note: Application facade
    - Path: cmd/goja-repl/root.go
      Note: Unified CLI entry point
    - Path: cmd/goja-repl/tui.go
      Note: Bubble Tea TUI subcommand
    - Path: pkg/repl/adapters/bobatea/replapi.go
      Note: Bobatea evaluator adapter
ExternalSources: []
Summary: Full code review and architectural analysis of the REPL work since origin/main, covering the session kernel, persistence layer, HTTP transport, profile system, TUI unification, and all remaining gaps and issues.
LastUpdated: 2026-04-06T16:00:00-04:00
WhatFor: Use this document to understand every part of the new REPL architecture, find confusing code, deprecated paths, unused code, naming issues, and to onboard as a new intern.
WhenToUse: Read this when you need to understand what the REPL system does, how its packages relate, where the rough edges are, and what to clean up next.
---

# Comprehensive Code Review: REPL Architecture Delta Since Origin/Main

## 0. How to Read This Document

This document is written for a **new intern** joining the project. It assumes you know Go and JavaScript but have never seen this codebase. Each section explains:

- **What** the thing is and why it exists
- **Where** the code lives (file paths and line numbers)
- **How** the pieces connect (with diagrams and pseudocode)
- **What is confusing, deprecated, unused, or poorly named**

Read it top-to-bottom the first time. After that, use the section headers as a reference index.

The diff under review spans **88 files, +14,149 lines, -1,167 lines** across five implementation tickets (GOJA-20 through GOJA-24). It represents a major architectural shift: from three separate REPL prototypes to one unified persistent REPL system.

---

## 1. What Is This Project?

**go-go-goja** is a Go library that embeds a JavaScript runtime (goja, a pure-Go ECMAScript 5.1+ interpreter) and provides tools around it:

- A module system so JavaScript code can `require()` native Go packages
- Static analysis of JavaScript (AST parsing, scope resolution, tree-sitter CST)
- Runtime introspection (object inspection, prototype chain walking, function-to-source mapping)
- An interactive REPL (Read-Eval-Print Loop) that lets humans and LLM agents evaluate JavaScript incrementally

The work under review transforms the REPL from **three separate prototypes** into **one unified system**:

```
BEFORE (origin/main)                     AFTER (this branch)
========================                 ========================
cmd/js-repl/    (Bubble Tea TUI)         cmd/goja-repl/
cmd/repl/       (line REPL)                ├── create, eval, snapshot
cmd/web-repl/   (HTTP + browser UI)         ├── history, bindings, docs
pkg/webrepl/    (prototype session)         ├── export, restore
pkg/repl/evaluators/javascript/             ├── serve (JSON server)
  (monolithic evaluator)                    └── tui (Bubble Tea UI)
                                         pkg/replsession/  (session kernel)
                                         pkg/repldb/       (SQLite persistence)
                                         pkg/replhttp/     (JSON transport)
                                         pkg/replapi/      (application facade)
```

### Why this matters

Before this work, the repository had three independent REPL entrypoints that did not share session management, persistence, evaluation logic, or binding tracking. An LLM agent using one could not resume a session in another. A human using the TUI could not later inspect their work through the CLI. This unification creates one **session kernel** that all surfaces share.

---

## 2. Architecture Overview

### 2.1 Layer diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    cmd/goja-repl (unified CLI)                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐  │
│  │ create   │ │ eval     │ │ serve    │ │ tui      │ │ export  │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └─────────┘  │
└───────────────────────────┬─────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        pkg/replapi (App facade)                     │
│                                                                     │
│  Combines:                                                          │
│  - live session management (replsession.Service)                   │
│  - optional durable store (repldb.Store)                           │
│  - auto-restore behavior                                           │
│  - profile-based configuration                                     │
│                                                                     │
│  App exposes: CreateSession, Evaluate, Snapshot, Restore,          │
│  DeleteSession, ListSessions, History, Export, Bindings, Docs      │
└──────────┬────────────────────────────────────┬─────────────────────┘
           │                                    │
           ▼                                    ▼
┌────────────────────────┐      ┌──────────────────────────────────────┐
│  pkg/replsession       │      │  pkg/repldb                          │
│  (session kernel)      │      │  (SQLite persistence)                │
│                        │      │                                      │
│  Service               │      │  Store                               │
│  ├── CreateSession     │      │  ├── CreateSession (write)           │
│  ├── Evaluate          │      │  ├── PersistEvaluation (write)       │
│  ├── Snapshot          │      │  ├── DeleteSession (write)           │
│  ├── DeleteSession     │      │  ├── ListSessions (read)             │
│  ├── RestoreSession    │      │  ├── LoadEvaluations (read)          │
│  └── WithRuntime       │      │  ├── ExportSession (read)            │
│                        │      │  └── LoadReplaySource (read)         │
│  sessionState          │      │                                      │
│  ├── runtime           │      │  Tables:                             │
│  ├── bindings          │      │  ├── sessions                        │
│  ├── cells             │      │  ├── evaluations                     │
│  └── consoleSink       │      │  ├── console_events                  │
│                        │      │  ├── bindings                        │
│  Evaluation pipeline:  │      │  ├── binding_versions                │
│  1. Parse/analyze      │      │  └── binding_docs                    │
│  2. Rewrite (async IIFE│      │                                      │
│     binding capture)   │      └──────────────────────────────────────┘
│  3. Snapshot globals   │
│  4. Execute in goja    │               ┌──────────────────────────┐
│  5. Snapshot globals   │               │  pkg/replhttp            │
│  6. Diff globals       │               │  (JSON HTTP transport)   │
│  7. Update bindings    │               │                          │
│  8. Persist to DB      │               │  NewHandler(app)         │
│  9. Return report      │               │  ├── POST /api/sessions  │
└──────────┬─────────────┘               │  ├── GET  /sessions/{id} │
           │                             │  ├── DEL  /sessions/{id} │
           │                             │  ├── POST .../evaluate   │
           ▼                             │  ├── GET  .../history    │
┌────────────────────────┐               │  ├── GET  .../bindings   │
│  engine/               │               │  ├── GET  .../docs       │
│  ├── Factory           │               │  ├── GET  .../export     │
│  ├── Runtime           │               │  └── POST .../restore    │
│  └── RuntimeOwner      │               └──────────────────────────┘
│     (serialized VM)    │
└────────────────────────┘
```

### 2.2 Package dependency flow

Dependencies flow **downward** only. No package imports something above it.

```
cmd/goja-repl
    │
    ├──▶ pkg/replapi ──▶ pkg/replsession ──▶ engine/
    │                  ──▶ pkg/repldb      ──▶ sqlite3
    │                  ──▶ pkg/replhttp
    │
    └──▶ pkg/repl/adapters/bobatea ──▶ pkg/replapi
                                    ──▶ pkg/repl/evaluators/javascript
                                          ──▶ pkg/jsparse
                                          ──▶ pkg/docaccess
                                          ──▶ pkg/inspector
```

---

## 3. Package-by-Package Deep Dive

### 3.1 pkg/replsession/ — The Session Kernel

This is the heart of the system. It manages live JavaScript REPL sessions.

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `service.go` | ~1633 | Session lifecycle, evaluation pipeline, binding bookkeeping, persistence |
| `types.go` | ~301 | All transport-neutral result types (CellReport, SessionSummary, etc.) |
| `policy.go` | ~173 | Configuration model (EvalPolicy, ObservePolicy, PersistPolicy) |
| `rewrite.go` | ~428 | Source-to-source transformation (async IIFE with binding capture) |

#### 3.1.1 The Service struct

```go
type Service struct {
    mu                 sync.RWMutex          // protects sessions map
    factory            *engine.Factory       // creates goja runtimes
    logger             zerolog.Logger
    store              Persistence           // optional durable store
    nextID             uint64                // auto-increment for session IDs
    sessions           map[string]*sessionState
    defaultSessionOpts SessionOptions        // from profile
}
```

**How it works:**

1. `NewService(factory, logger, opts...)` creates the service with a runtime factory and optional persistence.
2. `CreateSessionWithOptions(ctx, opts)` allocates a fresh goja runtime via `factory.NewRuntime(ctx)`, installs console capture and doc sentinels, optionally persists to SQLite.
3. `Evaluate(ctx, sessionID, source)` runs the full evaluation pipeline (see section 4).
4. `DeleteSession(ctx, sessionID)` removes the session and closes its runtime.
5. `RestoreSession(ctx, opts, history)` replays stored cells to rebuild a live session.

#### 3.1.2 The sessionState struct

```go
type sessionState struct {
    id          string
    profile     string
    policy      SessionPolicy
    createdAt   time.Time
    runtime     *engine.Runtime       // owns the goja VM
    logger      zerolog.Logger
    mu          sync.Mutex            // serializes cell evaluation
    nextCellID  int
    cells       []*cellState
    bindings    map[string]*bindingState
    consoleSink []ConsoleEvent
    ignored     map[string]struct{}   // sentinel names like __doc__
}
```

**Key invariant:** All goja VM access goes through `runtime.Owner.Call(ctx, opName, fn)`. The `sessionState.mu` mutex serializes evaluation. This prevents data races because the goja VM is not goroutine-safe.

#### 3.1.3 The bindingState struct

```go
type bindingState struct {
    Name            string
    Kind            jsparse.BindingKind  // var, let, const, function, class
    Origin          string               // "declared-top-level" or "runtime-global-diff"
    DeclaredInCell  int
    LastUpdatedCell int
    DeclaredLine    int
    DeclaredSnippet string
    Static          *BindingStaticView   // parser-derived metadata
    Runtime         BindingRuntimeView   // runtime inspection results
}
```

Bindings are discovered through three paths:
1. **Persisted-by-wrap** — names captured by the async IIFE return object
2. **Runtime-global-diff added** — new names that appeared after evaluation (leaked globals)
3. **Runtime-global-diff updated** — existing names whose values changed

---

### 3.2 pkg/repldb/ — SQLite Persistence

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `store.go` | ~92 | DB open/bootstrap/close |
| `schema.go` | ~89 | CREATE TABLE statements |
| `types.go` | ~71 | Durable record types |
| `write.go` | ~335 | Transactional writes |
| `read.go` | ~359 | Queries and export |

#### 3.2.1 Database schema

```
sessions ──┬──< evaluations ──┬──< console_events
           │                  ├──< binding_versions
           │                  └──< binding_docs
           └──< bindings ──┬──< binding_versions
                           └──< binding_docs
```

The schema uses `AUTOINCREMENT` integer primary keys for `evaluations`, `bindings`, `binding_versions`, `binding_docs`, and `console_events`. Sessions use `session_id TEXT PRIMARY KEY`.

#### 3.2.2 Write path

`PersistEvaluation` writes everything in a single SQLite transaction:

```
BeginTx
  ├── INSERT evaluation row → evaluation_id
  ├── INSERT console_events (batch prepared statement)
  ├── For each binding version:
  │     ├── INSERT OR IGNORE into bindings (ensure parent row)
  │     └── INSERT into binding_versions
  ├── For each binding doc:
  │     ├── INSERT OR IGNORE into bindings (ensure parent row)
  │     └── INSERT into binding_docs
  └── UPDATE sessions SET updated_at
Commit
```

**Good:** Transactional writes prevent partial state.

**Issue:** The `ensureBindingTx` function uses `INSERT OR IGNORE` then `SELECT` to find the binding ID. This works but requires two round trips per binding. An `INSERT ... ON CONFLICT DO UPDATE ... RETURNING binding_id` (upsert with returning) would be more efficient for sessions with many bindings per cell.

---

### 3.3 pkg/replapi/ — Application Facade

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `app.go` | ~218 | Facade combining session kernel + store |
| `config.go` | ~183 | Profile model and configuration |

The `App` is the public API surface that CLI commands and the HTTP handler consume. It provides:

- `CreateSession` / `CreateSessionWithOptions` — allocate sessions
- `Evaluate` — run one cell
- `Snapshot` — read current session state
- `Restore` — replay a persisted session
- `DeleteSession` — remove a session
- `ListSessions` / `History` / `Export` / `Bindings` / `Docs` — query
- `WithRuntime` — inspect the live goja runtime under session ownership

**The auto-restore pattern** (in `ensureLiveSession`):

```go
func (a *App) ensureLiveSession(ctx, sessionID) (*SessionSummary, error) {
    summary, err := a.service.Snapshot(ctx, sessionID)
    if err == nil {
        return summary, nil  // already live
    }
    if !errors.Is(err, ErrSessionNotFound) {
        return nil, err      // real error
    }
    if !a.config.AutoRestore || a.store == nil {
        return nil, err      // can't restore
    }
    return a.Restore(ctx, sessionID)  // auto-replay from SQLite
}
```

This means every `Evaluate`, `Snapshot`, `Bindings` call transparently restores the session if it isn't live yet. This is a deliberate design choice so that `goja-repl eval --session demo` works even after a server restart.

---

### 3.4 pkg/replhttp/ — JSON HTTP Transport

**Key file:** `handler.go` (~175 lines)

This is a thin JSON transport over `replapi.App`. It uses Go's standard `http.ServeMux` with manual path parsing.

**Routes:**

| Method | Path | Action |
|--------|------|--------|
| `GET` | `/api/sessions` | List sessions |
| `POST` | `/api/sessions` | Create session |
| `GET` | `/api/sessions/{id}` | Snapshot |
| `DELETE` | `/api/sessions/{id}` | Delete |
| `POST` | `/api/sessions/{id}/evaluate` | Evaluate cell |
| `POST` | `/api/sessions/{id}/restore` | Restore from DB |
| `GET` | `/api/sessions/{id}/history` | Cell history |
| `GET` | `/api/sessions/{id}/bindings` | Current bindings |
| `GET` | `/api/sessions/{id}/docs` | REPL-authored docs |
| `GET` | `/api/sessions/{id}/export` | Full export |

---

### 3.5 cmd/goja-repl/ — Unified CLI

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `root.go` | ~567 | Command definitions, factory wiring |
| `tui.go` | ~173 | Bubble Tea TUI subcommand |
| `main.go` | ~17 | Entry point |
| `root_test.go` | ~114 | Command-level tests |

The CLI uses the **Glazed** framework for command definitions. Each subcommand (`create`, `eval`, `serve`, `tui`, etc.) is a `cmds.BareCommand` implementation.

**Factory wiring** (in `newAppWithOptions`):

```
1. Open SQLite store (if needed)
2. Set up plugin discovery (hashiplugin/host)
3. Build engine.Factory with:
   - DefaultRegistryModules (fs, exec, database)
   - Optional docaccess runtime registrar
   - Plugin registrars
4. Create replapi.App with:
   - Profile (raw, interactive, persistent)
   - Store (if persistent)
5. Return (app, store, error)
```

---

### 3.6 pkg/repl/adapters/bobatea/ — Bubble Tea Adapters

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `replapi.go` | ~184 | Evaluator adapter for replapi-backed sessions |
| `runtime_assistance.go` | ~114 | Assistance-only adapter for self-owned runtimes |

The `REPLAPIAdapter` implements four Bobatea interfaces:
- `bobarepl.Evaluator` — evaluate source, return timeline events
- `bobarepl.InputCompleter` — provide auto-completion
- `bobarepl.HelpBarProvider` — contextual help bar
- `bobarepl.HelpDrawerProvider` — full help drawer

The `RuntimeAssistance` implements the same three assistance interfaces (without evaluation) for callers like `smalltalk-inspector` that own their own runtime.

---

### 3.7 pkg/repl/evaluators/javascript/ — Shared Assistance Provider

**Key files:**

| File | Lines | Role |
|------|-------|------|
| `evaluator.go` | ~615 (after -592) | Legacy evaluator, now delegates to Assistance |
| `assistance.go` | ~676 | Shared completion/help provider |
| `docs_resolver.go` | ~16 | Docs hub accessor |

The `Assistance` struct was extracted from the old monolithic evaluator. It provides:
- `CompleteInput` — tree-sitter parsing → candidate ranking → runtime augmentation
- `GetHelpBar` — contextual help summaries for the current cursor position
- `GetHelpDrawer` — full documentation for the symbol under the cursor

---

## 4. The Evaluation Pipeline — Step by Step

When you submit `const x = 42; x + 1` to a session, here is exactly what happens:

```
Step 1: PARSE
  jsparse.Analyze("<repl-cell-1>", source, nil)
  → AnalysisResult{Program, Index, Resolution}
  → BindingKind, ScopeRecords, Unresolved identifiers

Step 2: BUILD STATIC REPORT (optional)
  buildStaticReport(analysis, cstRoot, 512, 512)
  → Diagnostics, TopLevelBindings, ScopeView, AST rows, CST rows

Step 3: REWRITE
  buildRewrite(source, analysis, cellID=1)
  → Wrap in async IIFE:
     (async function () {
       let __ggg_repl_last_1__;
       const x = 42;
       __ggg_repl_last_1__ = (x + 1);
       return {
         "__ggg_repl_bindings_1__": {
           "x": (typeof x === "undefined" ? undefined : x)
         },
         "__ggg_repl_last_1__": (typeof __ggg_repl_last_1__ === "undefined"
           ? undefined : __ggg_repl_last_1__)
       };
     })()

Step 4: SNAPSHOT BEFORE
  iterate vm.GlobalObject().Keys()
  skip builtins (IsBuiltinGlobal)
  skip ignored (__doc__, __package__, etc.)
  → map[string]GlobalStateView

Step 5: EXECUTE
  runtime.Owner.Call(ctx, "replsession.run-string", func(vm) {
    return vm.RunString(rewrittenSource)
  })
  → goja.Value (a Promise)

Step 6: AWAIT PROMISE
  poll promise.State() every 5ms
  → Fulfilled → promise.Result()

Step 7: PERSIST WRAPPED RETURN
  Extract bindings object → mirror onto vm.GlobalObject()
  Extract last expression value
  → persistedNames=["x"], lastValue="43"

Step 8: SNAPSHOT AFTER
  Same as Step 4

Step 9: DIFF
  compare before vs after
  → added=["x"], diffs, leaked globals

Step 10: UPDATE BINDINGS
  upsertDeclaredBinding(analysis, "x", cellID=1)
  refreshBindingRuntimeDetails()
  → inspect goja values, prototype chains, function→source mappings

Step 11: PERSIST TO DB (if enabled)
  PersistEvaluation → SQLite transaction
  → evaluation row + console events + binding versions + docs

Step 12: RETURN
  EvaluateResponse{Session, Cell}
```

### Why the rewrite is needed

Goja evaluates code in global scope. `let` and `const` are block-scoped, so they don't persist between `RunString()` calls. The async IIFE wrapper captures them and mirrors them back onto the global object:

```javascript
// User writes: let x = 1
// After IIFE resolves, we do: vm.Set("x", value)
// In next cell, user writes: x + 1  → works because x is on the global object
```

---

## 5. The Policy System

The behavior of each session is controlled by a `SessionPolicy` with three sub-policies:

```go
type SessionPolicy struct {
    Eval    EvalPolicy    // how source is executed
    Observe ObservePolicy // what analysis/introspection is done
    Persist PersistPolicy // what gets written to SQLite
}
```

### EvalPolicy

| Mode | Behavior |
|------|----------|
| `instrumented` | Full async-IIFE rewrite with binding capture (default for interactive/persistent) |
| `raw` | Direct `vm.RunString()` with optional top-level await wrapping |

### ObservePolicy

| Flag | Effect |
|------|--------|
| `StaticAnalysis` | Parse with jsparse + tree-sitter, build static report |
| `RuntimeSnapshot` | Snapshot globals before and after each cell |
| `BindingTracking` | Track binding metadata, refresh runtime details |
| `ConsoleCapture` | Replace console object to capture output |
| `JSDocExtraction` | Install __doc__/__package__ sentinels, extract docs |

### PersistPolicy

| Flag | Effect |
|------|--------|
| `Enabled` | Master switch |
| `Sessions` | Write session row to SQLite |
| `Evaluations` | Write evaluation rows |
| `BindingVersions` | Write binding version rows with export snapshots |
| `BindingDocs` | Write extracted JSDoc metadata |

### Preset profiles

```go
// Raw: near-straight goja, no rewriting, no analysis
RawSessionOptions()

// Interactive: full analysis, binding capture, console, docs, NO persistence
InteractiveSessionOptions()

// Persistent: everything in Interactive PLUS full SQLite persistence
PersistentSessionOptions()
```

---

## 6. Code Review Findings

### 6.1 CRITICAL: service.go is a 1633-line monolith

**File:** `pkg/replsession/service.go`

This single file contains:
- Session lifecycle (CreateSession, DeleteSession, RestoreSession)
- The entire evaluation pipeline (Evaluate, evaluateInstrumented, evaluateRaw)
- Global snapshot/diff logic
- Binding bookkeeping (upsert, refresh, classify)
- Console capture installation
- Doc sentinel installation
- Persistence record assembly
- Promise polling
- Static report helpers (buildStaticReport was moved to rewrite.go, but service still has the evaluation orchestration)

**Recommendation:** Split into at least:

```
pkg/replsession/
  service.go           # Service struct, CreateSession, DeleteSession
  evaluate.go          # Evaluate, evaluateInstrumented, evaluateRaw
  snapshot.go          # snapshotGlobals, diffGlobals, refreshBindingRuntimeDetails
  bindings.go          # upsertDeclaredBinding, upsertRuntimeDiscoveredBinding, classifyBindingExport
  console.go           # installConsoleCapture, formatConsoleMessage
  persistence.go       # persistCell, bindingPersistenceRecords, extractBindingDocs
  restore.go           # RestoreSession
  helpers.go           # runtimeValueKind, gojaValuePreview, dedupeSortedStrings, etc.
```

The design doc for GOJA-20 actually recommended this split. It was not done during implementation.

### 6.2 CONFUSING: Two overlapping types for "session options"

**Files:** `pkg/replapi/config.go` and `pkg/replsession/policy.go`

There are two different `SessionOptions` types:

- `replsession.SessionOptions` — the internal session configuration
- `replapi.SessionOptions` — the app-level override struct

Both have `ID`, `CreatedAt`, `Profile`, and `Policy` fields, but they are different types. The `replapi.SessionOptions` has `Profile *Profile` and `Policy *SessionPolicy` as pointers (optional overrides), while `replsession.SessionOptions` has `Profile string` and `Policy SessionPolicy` as values.

**This is confusing** because:
- The same concept (session options) has two different shapes
- Callers must understand which one to use where
- `resolveSessionOptions` in config.go converts from replapi.SessionOptions to replsession.SessionOptions

**Recommendation:** Consider renaming one to `SessionOverrides` or `CreateSessionRequest` to distinguish "I want to override these defaults" from "these are the resolved settings".

### 6.3 CONFUSING: Duplicate Persistence interface

**File:** `pkg/replsession/service.go`

```go
type Persistence interface {
    CreateSession(ctx context.Context, record repldb.SessionRecord) error
    DeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error
    PersistEvaluation(ctx context.Context, record repldb.EvaluationRecord) error
}
```

The design doc for GOJA-20 recommended a broader `Store` interface. Instead, `replsession` defines its own narrow `Persistence` interface while `repldb.Store` is a concrete struct with many more methods. The `replapi.App` holds `*repldb.Store` directly and passes it through `WithPersistence(store)` which wraps it as the `Persistence` interface.

**Issue:** The indirection is reasonable (dependency inversion), but the naming is confusing. `replsession.Persistence` is not the same as `repldb.Store`, and there is no adapter/wrapper visible — `*repldb.Store` happens to satisfy the `Persistence` interface because it has the right method signatures, but this duck-typing is not documented anywhere.

**Recommendation:** Add a comment on the `Persistence` interface explaining that `*repldb.Store` satisfies it. Or create an explicit adapter.

### 6.4 DEPRECATED: pkg/repl/evaluators/javascript/evaluator.go still exists

**File:** `pkg/repl/evaluators/javascript/evaluator.go`

This is the old monolithic evaluator. After the extraction of `Assistance`, it was trimmed by 592 lines but it still exists and still:
- Owns its own `goja.Runtime`
- Has its own console capture
- Has its own evaluation loop
- Has its own promise handling

**Current consumers:**
- The old `jsadapter` in `pkg/repl/adapters/bobatea/javascript.go` (if it still exists)
- Potentially other callers not in the diff

**Recommendation:** Add a deprecation comment at the top of the file. Plan a follow-up ticket to migrate remaining consumers and delete it.

### 6.5 UNUSED: The `ignored` map is populated but barely checked

**File:** `pkg/replsession/service.go`

```go
type sessionState struct {
    // ...
    ignored     map[string]struct{}
}
```

The `ignored` map is populated in `installDocSentinels` with `__doc__`, `__package__`, `__example__`, and `doc`. It is checked in:
- `snapshotGlobals` — skips these names in global snapshots
- `bindingVersionRecord` — skips bindings for ignored names

**Subtle issue:** If a user writes `const doc = "hello"`, the name `doc` would be ignored because the sentinel installer puts `"doc"` in the ignored map. This is a correctness issue: `doc` is a common variable name that should not be silently suppressed from globals/bindings just because the doc template tag uses the same name.

**Recommendation:** Use a more specific sentinel prefix like `__ggg_doc__` instead of `doc` to avoid colliding with user variable names. Or track ignored names with a different mechanism that doesn't suppress user-declared bindings.

### 6.6 NAMING: `__ggg_repl_last_N__` and `__ggg_repl_bindings_N__` conventions

**File:** `pkg/replsession/rewrite.go`

The rewrite uses helper names like `__ggg_repl_last_1__` and `__ggg_repl_bindings_1__`. The `__ggg_` prefix is a reasonable namespace, but:
- These names appear in global snapshots (they're created inside the IIFE scope so they shouldn't leak, but `__ggg_repl_last_N__` is declared with `let` inside the IIFE)
- The naming convention is undocumented outside code comments

**Recommendation:** Document the naming convention in a comment block at the top of `rewrite.go`.

### 6.7 PERFORMANCE: Promise polling is busy-wait

**File:** `pkg/replsession/service.go`, `waitPromise` method

```go
func (s *sessionState) waitPromise(ctx context.Context, promise *goja.Promise) (goja.Value, error) {
    for {
        // ... check state ...
        case goja.PromiseStatePending:
            time.Sleep(5 * time.Millisecond)
            continue
        // ...
    }
}
```

This busy-waits with 5ms sleeps. For fast promises, this adds up to 5ms latency. For infinite loops, this hangs forever.

**Issues:**
- No timeout — a `while(true){}` cell hangs the session forever
- No context cancellation check — `ctx.Done()` is not checked in the loop
- CPU waste from polling

**Recommendation:**
1. Add `ctx.Done()` check in the loop
2. Add a configurable evaluation timeout (e.g., `WithEvalTimeout(30 * time.Second)`)
3. Use `vm.Interrupt("timeout")` to break infinite loops
4. Consider using goja's event loop integration for more efficient promise resolution

### 6.8 NAMING: `replapi` vs `replsession` package naming

The package names are confusing:
- `pkg/replapi` — is this the "API" in the sense of HTTP API? No, it's an application facade.
- `pkg/replsession` — is this about sessions? Yes, but it also handles evaluation, rewriting, snapshots, etc.

**Recommendation:** Consider clearer names:
- `pkg/replapp` instead of `pkg/replapi`
- `pkg/replcore` or `pkg/replkernel` instead of `pkg/replsession`

This is a naming refactor only and could be deferred.

### 6.9 MISSING: No evaluation timeout

**File:** `pkg/replsession/service.go`

There is no timeout on evaluation. A user can submit `while(true){}` and it will hang the session forever, consuming the goroutine. The design docs (GOJA-20, section 8.2) explicitly called this out as a high-priority gap.

**Recommendation:** Add `EvalTimeout time.Duration` to `EvalPolicy`. In `evaluateInstrumented` and `evaluateRaw`, wrap execution in a `context.WithTimeout` and call `vm.Interrupt("evaluation timeout")` when the context expires.

### 6.10 MISSING: No session TTL or cleanup

Sessions live forever until explicitly deleted. If a server creates 1000 sessions and nobody deletes them, the process runs out of memory.

**Recommendation:** Add a background goroutine that reaps sessions older than a configurable TTL.

### 6.11 MISSING: No CORS headers in HTTP handler

**File:** `pkg/replhttp/handler.go`

The JSON server has no CORS headers. This means browser-based clients cannot call the API from a different origin.

**Recommendation:** Add CORS middleware (at least `Access-Control-Allow-Origin: *` for development).

### 6.12 MISSING: No authentication or rate limiting

**File:** `pkg/replhttp/handler.go`

Any network client can create unlimited sessions, evaluate arbitrary code, and read all session data.

**Recommendation:** For production use, add API key authentication and rate limiting. For development, document that the server is intended for localhost use only.

### 6.13 ISSUE: repldb/types.go has suspicious comment

**File:** `pkg/repldb/types.go`

The first few lines of this file contain what appears to be garbled or copy-pasted text in comments:

```go
// SessionRecord is the durable representation of a REPL session.
type SessionRecord struct { ... }

type SessionRecord struct { ... }  // duplicate declaration?
```

Looking at the actual file content more carefully, there are some unusual struct declarations and comments that look like they may have been generated or pasted incorrectly. The `BindingVersionRecord` and `BindingDocRecord` structs at the bottom lack clear documentation.

**Recommendation:** Clean up comments in types.go. Add doc comments to all exported types.

### 6.14 ISSUE: Empty shell scripts in ticket directories

The diff adds several `.sh` files with zero bytes:

```
ttmp/.../scripts/repro-tmux-goja-035.sh
ttmp/.../scripts/analyze_bobatea_goja_boundary.sh
ttmp/.../scripts/widget_reuse_matrix.sh
ttmp/.../scripts/run-exp01.sh
ttmp/.../scripts/run-exp02.sh
ttmp/.../scripts/run-exp03.sh
ttmp/.../scripts/list-origin-main-review-surface.sh
ttmp/.../scripts/review-handoff-context.sh
```

These are empty placeholder scripts from earlier tickets. They serve no purpose in the current codebase.

**Recommendation:** Either populate them with useful content or delete them.

### 6.15 ISSUE: Hardcoded limits are package-level constants

**File:** `pkg/replsession/service.go`

```go
const (
    defaultASTRowLimit         = 512
    defaultCSTRowLimit         = 512
    defaultOwnPropertyLimit    = 20
    defaultPrototypeLevelLimit = 4
    defaultPrototypePropLimit  = 12
)
```

These are hardcoded and not configurable per-session. A session evaluating a 10,000-line script will still only get 512 AST rows.

**Recommendation:** Move these into `ObservePolicy` or `SessionOptions` so they can be tuned per-session.

### 6.16 ISSUE: Duplicate error type definitions

**Files:** `pkg/replsession/service.go` and `pkg/repldb/read.go`

```go
// pkg/replsession/service.go
var ErrSessionNotFound = errors.New("replsession: session not found")

// pkg/repldb/read.go
var ErrSessionNotFound = errors.New("repldb: session not found")
```

Two packages define the same semantic error with different messages. The HTTP handler maps both:

```go
func statusForError(err error) int {
    switch {
    case errors.Is(err, replsession.ErrSessionNotFound),
         errors.Is(err, repldb.ErrSessionNotFound):
        return http.StatusNotFound
    // ...
    }
}
```

**Recommendation:** Define one canonical `ErrSessionNotFound` in a shared location (e.g., `pkg/replapi/errors.go`) and have both packages reference it. Or use error codes with `errors.As`.

### 6.17 ISSUE: RestoreSession uses temporary Service

**File:** `pkg/replsession/service.go`, `RestoreSession`

```go
func (s *Service) RestoreSession(ctx context.Context, opts SessionOptions, history []string) (*SessionSummary, error) {
    // Creates a temporary service for replay
    tmpService := NewService(s.factory, s.logger, WithDefaultSessionOptions(replayOpts))
    tmpSummary, err := tmpService.CreateSessionWithOptions(ctx, replayOpts)
    // ... replay all cells ...
    // Transfer session state to the real service
    delete(tmpService.sessions, tmpSummary.ID)
    s.sessions[resolved.ID] = tmpState
}
```

This creates a whole temporary `Service` just to replay cells. It works but is wasteful — it creates a full service with its own sessions map just to hold one session temporarily.

**Recommendation:** Add an internal `createSessionState` method that creates a bare `sessionState` without registering it in the map, then replay against that state directly.

### 6.18 ISSUE: Inconsistent JSON field naming

**Files:** `pkg/replsession/types.go`

Most types use camelCase JSON tags consistently (`createdAt`, `cellId`, `durationMs`, `hadSideEffects`). However:

- `HelperError` serializes as `helperError` (fine)
- `HadSideFX` serializes as `hadSideEffects` (abbreviation FX → full word, inconsistent with the Go field name)
- `DurationMS` serializes as `durationMs` (MS → Ms, inconsistent)

**Recommendation:** Rename Go fields for consistency:
- `HadSideFX` → `HadSideEffects`
- `DurationMS` → `DurationMs`

### 6.19 ISSUE: The `buildRewrite` function for raw mode has no tests

**File:** `pkg/replsession/service.go`, `buildRewriteReport`

The raw-mode rewrite path (when `policy.Eval.Mode == "raw"`) includes top-level await wrapping logic:

```go
func wrapTopLevelAwaitExpression(source string) (string, bool) {
    trimmed := strings.TrimSpace(source)
    if strings.HasPrefix(trimmed, "await ") {
        return "(async () => { return " + trimmed + "; })()", true
    }
    return source, false
}
```

This is a simplistic heuristic that only handles code starting with `await `. It would miss patterns like:
```javascript
const x = await fetch('/api')
```

**Recommendation:** Add tests for the raw-mode rewrite. Consider improving the await detection to handle more patterns.

### 6.20 DESIGN: Persistence is synchronous inside Evaluate

**File:** `pkg/replsession/service.go`, `persistCell`

The `persistCell` method writes to SQLite synchronously while holding the session mutex:

```go
func (s *Service) Evaluate(ctx context.Context, sessionID string, source string) (*EvaluateResponse, error) {
    state.mu.Lock()
    defer state.mu.Unlock()
    // ... evaluation ...
    if err := s.persistCell(ctx, state, cell); err != nil {
        return nil, err
    }
    return &EvaluateResponse{...}, nil
}
```

This means the session is locked (no concurrent evaluations possible, which is correct for goja) but also blocks on SQLite I/O. For fast evaluations (sub-millisecond), the SQLite write might dominate the latency.

**Recommendation:** This is acceptable for now. If latency becomes an issue, consider async persistence (write to a channel, drain in a background goroutine).

### 6.21 ISSUE: No WAL mode for SQLite

**File:** `pkg/repldb/store.go`

The design docs recommended WAL mode and a busy timeout, but the implementation does neither:

```go
func (s *Store) bootstrap(ctx context.Context) error {
    tx, err := s.db.BeginTx(ctx, nil)
    // ...
    tx.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)
    // No PRAGMA journal_mode=WAL
    // No PRAGMA busy_timeout
}
```

**Recommendation:** Add:
```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
```

### 6.22 ISSUE: HTTP handler does not validate content-type

**File:** `pkg/replhttp/handler.go`

The evaluate endpoint accepts any content type:
```go
var req replsession.EvaluateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeJSONErrorMessage(w, http.StatusBadRequest, "invalid JSON body")
    return
}
```

It should check `Content-Type: application/json`.

### 6.23 ISSUE: SQLite stores full JSON blobs for reports

**File:** `pkg/repldb/schema.go`

The `evaluations` table stores `result_json`, `analysis_json`, `globals_before_json`, `globals_after_json` as full JSON text columns. For a session with 1000 cells, each containing full AST/CST rows, this could grow to hundreds of megabytes.

**Recommendation:** Add a configuration option to prune heavyweight fields (AST rows, CST rows) from persisted reports, or store them in a compressed form.

### 6.24 GOOD: Clean separation of concerns

Despite the issues above, the architectural separation is genuinely good:

1. **Session kernel** (`replsession`) is transport-neutral — it knows nothing about HTTP or CLI
2. **Persistence** (`repldb`) is behind a narrow interface — `replsession` only sees `Persistence`
3. **Application facade** (`replapi`) composes kernel + store correctly
4. **HTTP transport** (`replhttp`) is thin — it just maps routes to `replapi.App` calls
5. **TUI** (`cmd/goja-repl tui`) uses the same `replapi.App` as the CLI and server

### 6.25 GOOD: Comprehensive test coverage

The diff includes tests for:
- `pkg/repldb/store_test.go` — schema bootstrap, session CRUD, evaluation persistence
- `pkg/replsession/service_persistence_test.go` — end-to-end persistence through the live service
- `pkg/replsession/service_policy_test.go` — policy normalization
- `pkg/replapi/app_test.go` — app facade including auto-restore
- `pkg/replhttp/handler_test.go` — HTTP endpoint tests
- `pkg/repl/adapters/bobatea/replapi_test.go` — Bobatea adapter tests
- `pkg/repl/adapters/bobatea/runtime_assistance_test.go` — assistance-only adapter tests
- `cmd/goja-repl/root_test.go` — command-level tests

### 6.26 GOOD: Provenance tracking

Every response includes `ProvenanceRecord` entries that explain how each section was obtained:

```go
type ProvenanceRecord struct {
    Section string   `json:"section"`
    Source  string   `json:"source"`
    Notes   []string `json:"notes,omitempty"`
}
```

This is excellent for debugging and for LLM agents that need to understand what information is available.

### 6.27 GOOD: The policy/profile system

The three-tier policy system (`EvalPolicy` + `ObservePolicy` + `PersistPolicy`) with named profiles (`raw`, `interactive`, `persistent`) is well-designed. It allows:
- Simple preset selection (just pick a profile name)
- Fine-grained override when needed
- Normalization that fills in sensible defaults

---

## 7. Summary of Actionable Items

### Must fix (correctness/safety)

| # | Issue | File | Severity |
|---|-------|------|----------|
| 1 | No evaluation timeout — `while(true){}` hangs forever | `service.go` | High |
| 2 | `doc` in ignored map collides with user variable name | `service.go` | High |
| 3 | No context cancellation in promise polling | `service.go` | Medium |
| 4 | Duplicate `ErrSessionNotFound` in two packages | `service.go`, `read.go` | Medium |

### Should fix (code quality)

| # | Issue | File | Severity |
|---|-------|------|----------|
| 5 | `service.go` is 1633 lines — needs splitting | `service.go` | Medium |
| 6 | Two `SessionOptions` types with similar names | `config.go`, `policy.go` | Medium |
| 7 | `Persistence` interface duck-typing undocumented | `service.go` | Low |
| 8 | No SQLite WAL mode or busy timeout | `store.go` | Medium |
| 9 | Inconsistent Go field naming (`HadSideFX`, `DurationMS`) | `types.go` | Low |
| 10 | Hardcoded limits, not per-session configurable | `service.go` | Low |
| 11 | RestoreSession creates wasteful temporary Service | `service.go` | Low |

### Should clean up (deprecated/unused)

| # | Issue | File | Severity |
|---|-------|------|----------|
| 12 | `pkg/repl/evaluators/javascript/evaluator.go` still exists | `evaluator.go` | Medium |
| 13 | Empty `.sh` files in ticket directories | `ttmp/...` | Low |
| 14 | Suspicious comments in `repldb/types.go` | `types.go` | Low |

### Nice to have (ergonomics)

| # | Issue | File | Severity |
|---|-------|------|----------|
| 15 | No CORS headers in HTTP handler | `handler.go` | Low |
| 16 | No authentication or rate limiting | `handler.go` | Low |
| 17 | HTTP handler does not validate Content-Type | `handler.go` | Low |
| 18 | Full JSON blobs stored without pruning | `schema.go` | Low |
| 19 | Package naming (`replapi` vs `replsession`) unclear | — | Low |
| 20 | Raw-mode `wrapTopLevelAwaitExpression` is simplistic | `service.go` | Low |
| 21 | No session TTL/cleanup mechanism | `service.go` | Medium |

---

## 8. Recommendations for Next Steps

### Immediate (before merging to main)

1. **Add evaluation timeout** with `vm.Interrupt()` and `context.WithTimeout`
2. **Rename the `doc` sentinel** to avoid colliding with user variable names
3. **Add context cancellation** to the promise polling loop
4. **Add WAL mode** to SQLite bootstrap

### Short-term (first cleanup ticket after merge)

5. **Split `service.go`** into focused files
6. **Add deprecation notice** to `pkg/repl/evaluators/javascript/evaluator.go`
7. **Consolidate error types** into one canonical location
8. **Delete empty `.sh` files**

### Medium-term (architecture improvements)

9. **Add session TTL** and background cleanup
10. **Add CORS and Content-Type validation** to HTTP handler
11. **Make limits configurable** per-session
12. **Add report pruning** for large persisted sessions

---

## 9. Testing Validation Commands

```bash
# Run all tests
go test ./...

# Run focused tests for the new packages
go test ./pkg/replsession/... ./pkg/repldb/... ./pkg/replapi/... ./pkg/replhttp/...

# Run adapter tests
go test ./pkg/repl/adapters/bobatea/...

# Run CLI tests
go test ./cmd/goja-repl/...

# Run linter
golangci-lint run ./...

# Smoke test the TUI
go run ./cmd/goja-repl tui --alt-screen=false

# Test the server
go run ./cmd/goja-repl serve &
curl -X POST http://localhost:3090/api/sessions
curl -X POST http://localhost:3090/api/sessions/$ID/evaluate -d '{"source":"1+1"}'
```

---

## 10. Glossary for New Interns

| Term | Definition |
|------|-----------|
| **goja** | A pure-Go ECMAScript 5.1+ JavaScript interpreter. Not V8, not Node.js. |
| **goja.Runtime** | The Go type representing one JavaScript VM. NOT goroutine-safe. |
| **runtimeowner.Runner** | Serializes access to one `goja.Runtime` via `Call(ctx, name, fn)`. |
| **engine.Factory** | Creates pre-configured `Runtime` instances with modules registered. |
| **IIFE** | Immediately Invoked Function Expression: `(function(){ ... })()` |
| **Binding** | A named JavaScript value tracked by the REPL session (variable, function, class). |
| **Cell** | One unit of code submitted for evaluation (like a Jupyter notebook cell). |
| **Session** | A named, long-lived goja runtime with persistent bindings and history. |
| **Rewrite** | The source-to-source transformation that wraps code in an async IIFE. |
| **Global diff** | Comparison of non-builtin global properties before and after evaluation. |
| **Profile** | A named preset of session policies: `raw`, `interactive`, or `persistent`. |
| **Policy** | The full behavioral configuration for one session: eval + observe + persist. |
| **CST** | Concrete Syntax Tree from tree-sitter (preserves whitespace, comments, errors). |
| **AST** | Abstract Syntax Tree from goja's parser (semantic nodes only). |
| **jsparse** | The static analysis package: parser + indexer + scope resolver. |
| **Bobatea** | The Bubble Tea TUI framework used for the interactive REPL. |
| **Glazed** | The CLI framework used for command definitions and flag parsing. |
| **docmgr** | The ticket documentation management tool used for this review. |
| **reMarkable** | An e-ink tablet used for reviewing documents. |
