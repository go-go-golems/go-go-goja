---
Title: 'Fuzzing replapi: Comprehensive Analysis and Implementation Guide'
Ticket: GOJA-050
Status: active
Topics:
    - fuzzing
    - replapi
    - testing
    - security
    - goja
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime factory
    - Path: pkg/repl/adapters/bobatea/replapi.go
      Note: TUI adapter
    - Path: pkg/replapi/app.go
      Note: Main facade - fuzzing entry point
    - Path: pkg/replapi/config.go
      Note: Profile configuration
    - Path: pkg/repldb/store.go
      Note: SQLite store
    - Path: pkg/replsession/evaluate.go
      Note: Evaluation pipeline
    - Path: pkg/replsession/observe.go
      Note: Runtime observation layer
    - Path: pkg/replsession/persistence.go
      Note: SQLite write layer
    - Path: pkg/replsession/policy.go
      Note: Session policy types
    - Path: pkg/replsession/rewrite.go
      Note: IIFE wrapper - contains empty-string panic bug
    - Path: pkg/replsession/service.go
      Note: Session kernel
    - Path: pkg/runtimeowner/runner.go
      Note: Thread-safe VM access
ExternalSources: []
Summary: End-to-end design and intern-ready implementation guide for fuzz-testing the go-go-goja replapi layer and its underlying replsession kernel, covering architecture, attack surface, seed corpus design, harness patterns, and phased rollout.
LastUpdated: 2026-04-20T09:50:23.424841528-04:00
WhatFor: Guide a new intern through building, running, and extending a fuzzing infrastructure for go-go-goja's REPL evaluation pipeline.
WhenToUse: When adding fuzz tests, extending coverage, or debugging fuzz-found crashes in replapi/replsession.
---



# Fuzzing go-go-goja replapi: Comprehensive Analysis and Implementation Guide

## Executive Summary

The **go-go-goja** project is a Go-based JavaScript runtime built on top of [goja](https://github.com/dop251/goja) (a pure-Go ECMAScript 5.1+ interpreter). At its heart sits **replapi**, a layered API that manages interactive JavaScript REPL sessions — from raw code execution through an elaborate rewrite pipeline to persistent SQLite-backed session storage and restore.

This document is a **comprehensive, intern-ready guide** that:

- Explains every layer of the replapi system in detail with prose, diagrams, and code references.
- Identifies every fuzzable surface — parsing, rewriting, execution, persistence, and restore.
- Provides a phased implementation plan with concrete seed corpora, harness patterns, and integration strategy.
- Documents the results of four proof-of-concept fuzz experiments, including **one crash** already discovered by our fuzzer.

**Key finding from experiments**: Experiment 03 (rewrite-pipeline-fuzz) immediately discovered a panic in the instrumented evaluation path: passing an empty string `""` causes an `index out of range [0] with length 0` panic inside `buildRewrite`. This validates the approach and demonstrates the value of systematic fuzzing.

---

## Problem Statement and Scope

### Why fuzz go-go-goja?

go-go-goja accepts **arbitrary user-supplied JavaScript** and executes it through a multi-stage pipeline. Every stage is a potential crash, hang, memory leak, or correctness bug:

1. **Parsing**: The `jsparse` package parses JavaScript using both goja's built-in parser and tree-sitter. Malformed or edge-case inputs can trigger panics in parser code.
2. **Rewriting**: The `buildRewrite` function transforms user code into async IIFEs with binding capture. It performs string slicing, index arithmetic, and AST traversal — all brittle under unexpected inputs.
3. **Execution**: The goja VM executes the rewritten code. Goja is a complex interpreter with built-in modules, promises, proxies, and more. Unusual JS patterns can trigger Go-level panics.
4. **Observation**: The instrumented path snapshots global state, diffs before/after, inspects object prototypes, and extracts property descriptors — all of which touch goja reflection APIs that may not handle every edge case.
5. **Persistence**: Evaluations are serialized to JSON and written to SQLite. Unusual values (deeply nested objects, symbols, functions) can stress the serialization layer.
6. **Restore**: Replaying a sequence of cells reconstructs a live session. If any cell in the sequence fails, the restore pipeline must handle it gracefully.

### Scope

**In scope** (this document):

- Fuzzing `replapi.App` methods: `CreateSession`, `Evaluate`, `Snapshot`, `Restore`, `DeleteSession`, `ListSessions`, `History`, `Export`, `ReplaySource`, `Bindings`, `Docs`, `WithRuntime`.
- Fuzzing `replsession.Service` directly (for lower-level harnesses).
- Fuzzing the `buildRewrite` function in isolation.
- Fuzzing persistence round-trips (evaluate → persist → restore → verify).
- Seed corpus curation and coverage goals.
- CI integration strategy.

**Out of scope** (future work):

- Fuzzing native modules (`database`, `exec`, `fs`, `timer`) in isolation.
- Fuzzing the TUI layer (Bubble Tea / Bobatea adapter).
- Fuzzing the HTTP server (`replhttp`) or gRPC plugin host (`hashiplugin`).
- Fuzzing the tree-sitter parser integration in isolation (that belongs in `pkg/jsparse`).
- Network-level fuzzing or penetration testing.

---

## Current-State Architecture

### System Overview Diagram

The go-go-goja REPL system is organized in layers. The user's JavaScript source enters at the top and flows down through each layer:

```
┌──────────────────────────────────────────────────────────────────────┐
│                          User / TUI / HTTP                          │
│                    (cmd/goja-repl, cmd/inspector)                   │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         replapi.App (Facade)                        │
│  ┌──────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ ┌──────────┐ │
│  │ Create   │ │ Evaluate  │ │ Snapshot  │ │ Restore │ │ Delete   │ │
│  │ Session  │ │           │ │           │ │         │ │ Session  │ │
│  └──────────┘ └───────────┘ └───────────┘ └─────────┘ └──────────┘ │
│  Config: Profile, Store, AutoRestore, SessionOptions                │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                     replsession.Service (Kernel)                    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Evaluation Pipeline                       │    │
│  │                                                             │    │
│  │  Source ──► jsparse.Analyze ──► buildRewrite ──► execute   │    │
│  │               │                   │               │        │    │
│  │               ▼                   ▼               ▼        │    │
│  │          StaticReport     RewriteReport    ExecutionReport │    │
│  │                                               RuntimeReport│    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  sessionState: cells[], bindings{}, consoleSink[], ignored{}        │
│  Policy: EvalPolicy + ObservePolicy + PersistPolicy                 │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      engine.Runtime (VM Host)                       │
│  ┌────────────────┐  ┌──────────────┐  ┌─────────────────────────┐  │
│  │  goja.Runtime  │  │ eventloop    │  │ runtimeowner.Runner     │  │
│  │  (JS VM)       │  │ (EventLoop)  │  │ (Thread-safe Call/Post) │  │
│  └────────────────┘  └──────────────┘  └─────────────────────────┘  │
│  engine.Factory: builds Runtime with modules + initializers         │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        repldb.Store (SQLite)                        │
│  sessions, evaluations, binding_versions, binding_docs              │
│  WAL mode, foreign keys, busy_timeout=5000                         │
└──────────────────────────────────────────────────────────────────────┘
```

### Layer-by-Layer Explanation

#### 1. `replapi.App` — The Facade

`replapi.App` is the **public API surface** that TUI commands and HTTP handlers call. It is a thin facade that:

- Holds a `Config` (profile, store reference, auto-restore setting).
- Delegates to `replsession.Service` for all live session operations.
- Delegates to `repldb.Store` for all persistence queries (history, export, list sessions).
- Implements auto-restore: if a session ID is not found in memory, it checks the store and replays history.

**File**: `pkg/replapi/app.go` (~200 lines)

Key methods:

| Method | What it does | Fuzzing interest |
|--------|-------------|------------------|
| `CreateSession(ctx)` | Creates a new in-memory session | Low (deterministic) |
| `Evaluate(ctx, sessionID, source)` | Main entry point — evaluates JS source | **Critical** |
| `Snapshot(ctx, sessionID)` | Returns current session summary | Medium |
| `Restore(ctx, sessionID)` | Replays persisted cells into fresh runtime | **High** |
| `DeleteSession(ctx, sessionID)` | Deletes session from memory + store | Low |
| `ListSessions(ctx)` | Lists persisted sessions | Low |
| `History(ctx, sessionID)` | Returns evaluation history | Low |
| `Export(ctx, sessionID)` | Returns structured export of session | Medium |
| `ReplaySource(ctx, sessionID)` | Returns raw sources for replay | Low |
| `Bindings(ctx, sessionID)` | Returns current bindings | Medium |
| `Docs(ctx, sessionID)` | Returns extracted JSDoc records | Low |
| `WithRuntime(ctx, sessionID, fn)` | Runs fn against the live goja runtime | Medium |

#### 2. `replapi.Config` — Profiles and Options

`replapi.Config` controls behavior through three named profiles:

- **`raw`**: Direct goja execution, no binding capture, no static analysis, 5-second timeout. Closest to bare goja.
- **`interactive`**: Full instrumented pipeline — async IIFE wrapping, binding capture, static analysis, runtime snapshots, console capture, JSDoc extraction. In-memory only.
- **`persistent`**: Same as interactive, plus SQLite persistence (sessions, evaluations, binding versions, binding docs).

Each profile maps to a `replsession.SessionPolicy` with three sub-policies:

- **`EvalPolicy`**: Mode (raw/instrumented), capture last expression, top-level await, timeout.
- **`ObservePolicy`**: Static analysis, runtime snapshot, binding tracking, console capture, JSDoc extraction.
- **`PersistPolicy`**: Sessions, evaluations, binding versions, binding docs.

**Files**: `pkg/replapi/config.go`, `pkg/replsession/policy.go`

#### 3. `replsession.Service` — The Kernel

`replsession.Service` manages in-memory sessions. Each session is a `sessionState` struct containing:

- A goja `engine.Runtime` (VM + event loop + owner).
- A mutable `cells` slice (one `cellState` per evaluation).
- A mutable `bindings` map (name → `bindingState`).
- A `consoleSink` for captured console events.
- An `ignored` set for sentinel names (`__doc__`, `__package__`, etc.).

The service is **thread-safe**: it uses `sync.RWMutex` at the service level and `sync.Mutex` per session.

**File**: `pkg/replsession/service.go` (~400 lines)

#### 4. The Evaluation Pipeline

When `Evaluate(ctx, sessionID, source)` is called, the following pipeline executes under the session lock:

```
Source string
    │
    ├─── if policy.UsesInstrumentedExecution() ───► Instrumented Path
    │                                                     │
    │   ┌──────────────┐                           ┌──────┴──────┐
    │   │ jsparse.     │                           │ buildRewrite │
    │   │ Analyze()    │                           │ (async IIFE) │
    │   └──────┬───────┘                           └──────┬───────┘
    │          ▼                                          ▼
    │   StaticReport (AST rows,                      RewriteReport
    │     CST rows, diagnostics,                     (transformed source,
    │     bindings, scope, refs)                      helper names)
    │          │                                          │
    │          ▼                                          ▼
    │   ┌──────────────────────────────────────────────────────┐
    │   │  snapshotGlobals (before)                            │
    │   │  executeWrapped (run transformed source)             │
    │   │  snapshotGlobals (after)                             │
    │   │  diffGlobals → new/updated/removed bindings          │
    │   │  refreshBindingRuntimeDetails                        │
    │   │  persistCell (if persistent)                         │
    │   └──────────────────────────────────────────────────────┘
    │                         │
    │                         ▼
    │              EvaluateResponse (CellReport + SessionSummary)
    │
    └─── else ──────────────► Raw Path
                                │
                    ┌───────────┴───────────┐
                    │ executeRaw            │
                    │ (direct vm.RunString) │
                    └───────────┬───────────┘
                                │
                                ▼
                    EvaluateResponse (CellReport + SessionSummary)
```

**File**: `pkg/replsession/evaluate.go` (~350 lines)

#### 5. `buildRewrite` — The IIFE Wrapper

The `buildRewrite` function (in `pkg/replsession/rewrite.go`) transforms user source into an async IIFE that:

1. Wraps the entire cell in `(async function () { ... })()`.
2. Declares a `__ggg_repl_last_N__` variable to capture the last expression.
3. Declares a `__ggg_repl_bindings_N__` object to capture top-level declarations.
4. Returns `{ __ggg_repl_bindings_N__: {name1, name2, ...}, __ggg_repl_last_N__: value }`.

After execution, `persistWrappedReturn` reads this result object and:

- Sets each captured name onto the VM global object (so it's visible in subsequent cells).
- Extracts the last expression value for the cell report.

**Example transformation**:

```javascript
// Input:
const x = 1; x + 1

// Output:
(async function () {
  let __ggg_repl_last_1__;
  const x = 1; __ggg_repl_last_1__ = (x + 1);
  return {
    "__ggg_repl_bindings_1__": {
      "x": (typeof x === "undefined" ? undefined : x)
    },
    "__ggg_repl_last_1__": (typeof __ggg_repl_last_1__ === "undefined" ? undefined : __ggg_repl_last_1__)
  };
})()
```

**File**: `pkg/replsession/rewrite.go` (~200 lines)

#### 6. `engine.Runtime` — The VM Host

`engine.Runtime` bundles:

- `*goja.Runtime` — the JavaScript VM.
- `*eventloop.EventLoop` — the Node.js-compatible event loop (for `setTimeout`, `setInterval`, Promises).
- `runtimeowner.Runner` — a thread-safe request/response wrapper that schedules work onto the event loop goroutine.
- `Values` map — runtime-scoped key-value store for modules.
- `Closer` hooks — cleanup functions called on `Close()`.

`engine.Factory` creates `Runtime` instances from an immutable build plan (modules, registrars, initializers).

**Files**: `engine/factory.go`, `engine/runtime.go`

#### 7. `runtimeowner.Runner` — Thread Safety

`Runner` ensures all VM operations happen on the owner goroutine (the event loop goroutine). It provides:

- `Call(ctx, op, fn)` — schedules `fn(vm)` on the event loop and waits for the result.
- `Post(ctx, op, fn)` — schedules `fn(vm)` fire-and-forget.
- `Shutdown(ctx)` — marks the runner as closed.

It detects re-entrant calls (calling from the owner goroutine) and handles them directly without scheduling.

**File**: `pkg/runtimeowner/runner.go` (~200 lines)

#### 8. `repldb.Store` — SQLite Persistence

The store uses SQLite with WAL mode, foreign keys, and a 5-second busy timeout. It persists:

- `sessions` — session metadata (ID, timestamps, engine kind, policy JSON).
- `evaluations` — per-cell records (source, rewritten source, result JSON, error, analysis, globals before/after, console events).
- `binding_versions` — versioned snapshots of binding state (name, action, runtime type, export JSON).
- `binding_docs` — extracted JSDoc documentation per symbol per cell.

**Files**: `pkg/repldb/store.go`, `pkg/repldb/types.go`, `pkg/repldb/schema.go`

---

## Fuzzing Targets and Attack Surface Analysis

This section identifies every surface that can be fuzzed, ranked by risk and expected reward.

### Target 1: `replapi.Evaluate` — Raw Mode

**Entry point**: `replapi.App.Evaluate(ctx, sessionID, source)` with `ProfileRaw`

**What it exercises**:
- Direct `vm.RunString(source)` on the goja VM.
- Top-level await wrapping (`wrapTopLevelAwaitExpression`).
- Promise polling (`waitPromise`) for async results.
- Interrupt mechanism for timeouts.

**Why it's interesting**: The goja VM is a complex interpreter. Unusual JavaScript patterns can trigger:
- Go-level panics in the VM.
- Infinite loops that the interrupt mechanism might not catch.
- Memory exhaustion (e.g., `new Array(1e9)`).
- Stack overflows from deep recursion.

**Fuzz input**: Arbitrary string (the JavaScript source).

**Expected behavior**: The evaluation should either return a result or an error. It should **never panic**.

**API reference**:
```go
// pkg/replsession/evaluate.go
func (s *sessionState) executeRaw(ctx context.Context, source string, policy SessionPolicy) (executionOutcome, error)
```

### Target 2: `replapi.Evaluate` — Instrumented Mode

**Entry point**: `replapi.App.Evaluate(ctx, sessionID, source)` with `ProfileInteractive` or `ProfilePersistent`

**What it exercises** (in addition to raw mode):
- `jsparse.Analyze(filename, source, nil)` — parser and resolver.
- Tree-sitter CST parsing.
- `buildRewrite(source, analysis, cellID)` — the IIFE wrapper.
- `executeWrapped(ctx, rewrite)` — executing the transformed source.
- `persistWrappedReturn(ctx, value, bindingsKey, lastKey)` — extracting bindings from the result.
- `snapshotGlobals(ctx)` — global object snapshot before and after.
- `diffGlobals(before, after, bindings)` — computing added/updated/removed.
- `refreshBindingRuntimeDetails(ctx)` — introspecting each binding's runtime value.
- `ownPropertiesView(obj, vm)` — property descriptor inspection.
- `prototypeChainView(obj, vm)` — prototype chain traversal.

**Why it's critical**: This path has the most code and the most string/index manipulation. The `buildRewrite` function performs source slicing using AST indices, which is inherently fragile.

**Known bug found by fuzzer**: Empty string input causes `index out of range [0]` in `buildRewrite`'s `finalExpressionStatement` check.

**API references**:
```go
// pkg/replsession/rewrite.go
func buildRewrite(source string, result *jsparse.AnalysisResult, cellID int) RewriteReport

// pkg/replsession/evaluate.go
func (s *sessionState) executeWrapped(ctx context.Context, rewrite RewriteReport) (executionOutcome, error)
func (s *sessionState) persistWrappedReturn(ctx context.Context, value goja.Value, bindingsKey string, lastKey string) ([]string, string, bool, error)
```

### Target 3: Multi-Cell Session Sequences

**Entry point**: Multiple calls to `Evaluate` on the same session.

**What it exercises**:
- State accumulation across cells (bindings, console history).
- The `bindings` map being updated, merged, and cleaned across evaluations.
- Binding version persistence (`bindingPersistenceRecords`).
- Interaction between declaration capture and runtime-discovered globals.

**Why it's interesting**: State-dependent bugs only appear after several evaluations. For example:
- Declaring a variable in cell 1, then re-declaring it in cell 2.
- Deleting a binding in cell 3 and checking that it's removed from runtime.
- The binding's `DeclaredInCell` vs `LastUpdatedCell` tracking.

**Fuzz input**: A sequence of N strings (where N = 2-5 is sufficient for most bugs).

**API reference**:
```go
// pkg/replsession/service.go
func (s *Service) Evaluate(ctx context.Context, sessionID string, source string) (*EvaluateResponse, error)
```

### Target 4: Persistence and Restore

**Entry point**: `replapi.App.Restore(ctx, sessionID)`

**What it exercises**:
- `repldb.Store.LoadSession` and `LoadReplaySource` — SQLite reads.
- `replsession.Service.RestoreSession` — creates a temporary service, replays all cells, then adopts the session.
- `restoreOptionsForRecord` — session metadata deserialization.
- The full evaluate pipeline during replay.

**Why it's interesting**: Restore replays all cells from scratch. If any cell in the sequence fails, the restore aborts. The replay must be **deterministic** — the same source must produce the same bindings every time.

**Fuzz input**: A sequence of strings (seed, then restore and verify).

**API references**:
```go
// pkg/replapi/app.go
func (a *App) Restore(ctx context.Context, sessionID string) (*replsession.SessionSummary, error)

// pkg/replsession/service.go
func (s *Service) RestoreSession(ctx context.Context, opts SessionOptions, history []string) (*SessionSummary, error)
```

### Target 5: `buildRewrite` in Isolation

**Entry point**: `buildRewrite(source, analysis, cellID)` directly.

**What it exercises**:
- All string slicing and index arithmetic.
- The `finalExpressionStatement` detection.
- The `declaredNamesFromResult` extraction.
- The IIFE construction via `strings.Builder`.

**Why it's interesting**: This function is pure (no side effects), making it ideal for fast fuzzing. The goja VM is not needed.

**Fuzz input**: Arbitrary string. Dependency: `jsparse.Analyze` must run first to produce the `AnalysisResult`.

**API reference**:
```go
// pkg/replsession/rewrite.go
func buildRewrite(source string, result *jsparse.AnalysisResult, cellID int) RewriteReport
```

### Target 6: Global Snapshotting and Diffing

**Entry point**: `snapshotGlobals(ctx)` and `diffGlobals(before, after, bindings)`.

**What it exercises**:
- Iterating over all global object keys.
- Calling `inspectorruntime.ValuePreview` on every value.
- `runtimeValueKind` classification (undefined, null, function, string, boolean, number, object, unknown).
- `globalStateFromValue` property counting.
- The diff algorithm (added, updated, removed).

**Why it's interesting**: Unusual goja values (Proxy, WeakMap, Symbol, etc.) might not be handled correctly by `runtimeValueKind` or `gojaValuePreview`.

### Target 7: `replapi.Config` Resolution

**Entry point**: `normalizeConfig`, `validateConfig`, `resolveCreateSessionOptions`.

**What it exercises**: Edge cases in profile defaults, policy overrides, and session option resolution.

**Why it's interesting**: Low risk but quick to fuzz. Could find nil dereferences or empty-string comparison bugs.

---

## Fuzzing Design and Approach

### Tooling: Go Native Fuzzing

Go 1.18+ includes a built-in fuzzing framework (`go test -fuzz=...`). It provides:

- **Coverage-guided mutators**: Automatically mutates seed inputs based on code coverage.
- **Deterministic reproduction**: Crashes are saved to `testdata/fuzz/FuzzName/<hash>` and replayed with `go test -run=FuzzName/<hash>`.
- **Worker parallelism**: Runs 8+ fuzz workers in parallel by default.
- **Seed corpus**: Explicit `f.Add(value)` calls define the starting corpus.
- **Integration with `go test`**: Fuzz tests live alongside regular tests.

### Harness Architecture

Each fuzz target should follow this pattern:

```go
func FuzzTargetName(f *testing.F) {
    // 1. Add seed corpus
    for _, seed := range seeds {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, input string) {
        // 2. Create fresh App + Session per invocation
        //    (isolates state, avoids cross-contamination)
        ctx := context.Background()
        app := newTestApp(t)
        session, _ := app.CreateSession(ctx)

        // 3. Wrap in recover() to catch panics
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("panic on input %q: %v", input, r)
            }
        }()

        // 4. Call with timeout
        timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        app.Evaluate(timeoutCtx, session.ID, input)
    })
}
```

### Key Design Decisions

1. **Fresh App per invocation**: Creating a new `engine.Factory` + `replapi.App` per fuzz iteration is expensive (~1ms), but it provides perfect isolation. For faster targets (like `buildRewrite` in isolation), we can reuse the factory.

2. **Timeout on every call**: The default `EvalPolicy.TimeoutMS` is 5000ms. We set a context timeout slightly above this to catch hangs.

3. **Recover panics as failures**: Go's fuzzer treats `t.Fatalf` as a failure, not a crash. We use `recover()` + `t.Fatalf` so panics are recorded as reproducible failures.

4. **Three profiles, three harnesses**: Each profile exercises different code paths. We need separate harnesses for raw, interactive, and persistent modes.

5. **Multi-string inputs for stateful fuzzing**: Session lifecycle bugs require sequences. We use `f.Add(a, b)` for two-step and `f.Add(a, b, c)` for three-step sequences.

### Performance Considerations

- **Raw mode**: ~2,000-8,000 exec/sec per worker (fast, no rewriting).
- **Instrumented mode**: ~50-500 exec/sec per worker (rewrite + observation overhead).
- **Persistent mode**: ~50-200 exec/sec per worker (SQLite write overhead).
- **Isolated rewrite**: ~10,000+ exec/sec (pure function, no VM).

For CI, target 30-60 seconds of fuzzing per harness. For overnight runs, target 1-4 hours.

---

## Seed Corpus Design

The seed corpus is the starting point for the mutator. A well-curated seed corpus dramatically improves coverage by giving the mutator interesting byte patterns to work with.

### Category 1: Minimal Valid JavaScript

These are simple expressions that exercise basic VM paths:

```go
seeds := []string{
    "",            // empty
    " ",           // whitespace only
    ";",           // empty statement
    "1",           // literal
    "'hello'",     // string literal
    "true",        // boolean
    "null",        // null
    "undefined",   // undefined
}
```

### Category 2: Declarations (for rewrite capture)

These exercise the `declaredNamesFromResult` and binding capture:

```go
seeds := []string{
    "const x = 1",
    "let y = 2",
    "var z = 3",
    "const a = 1, b = 2",           // multiple bindings
    "function f() { return 1 }",    // function declaration
    "class A { m() { return 1 } }", // class declaration
    "const { a, b } = { a: 1, b: 2 }",  // destructuring
    "const [x, y] = [1, 2]",            // array destructuring
}
```

### Category 3: Expression-Statement Combinations

These exercise `finalExpressionStatement` detection and last-expression capture:

```go
seeds := []string{
    "const x = 1; x",               // declaration + expression
    "const x = 1; x + 1",           // declaration + expression
    "const x = 1; const y = 2; x + y", // multiple + expression
    "1 + 2",                         // expression only
    "'hello'.toUpperCase()",         // method call
    "[1,2,3].map(x => x * 2)",      // array method + arrow
}
```

### Category 4: Async and Promise Patterns

These exercise the promise polling and top-level await wrapping:

```go
seeds := []string{
    "Promise.resolve(1)",
    "new Promise(r => setTimeout(() => r(1), 10))",
    "async function f() { return 1 }; f()",
    "await Promise.resolve(1)",  // top-level await
}
```

### Category 5: Error-Producing Inputs

These should produce errors but **never panics**:

```go
seeds := []string{
    "throw new Error('test')",
    "undefined.property",
    "null.toString()",
    "JSON.parse('{invalid')",
    "new (-1)",
    "(function() { return arguments })(1,2)",
}
```

### Category 6: Unicode and Edge-Case Strings

```go
seeds := []string{
    "const 你好 = 'world'; 你好",
    "'\\x00\\x01\\x02'",
    "'\\uffff'",
    "`template ${1 + 2}`",
    "// comment\n1",
    "/* block */ 2",
    "String.fromCharCode(0, 0xFFFF)",
}
```

### Category 7: Deeply Nested / Recursive Structures

```go
seeds := []string{
    "((((1))))",
    "if(true){if(true){if(true){1}}}",
    "const a = {a:{a:{a:1}}}; a",
    "const f = () => () => () => 1; f()()()",
    "try { throw 1 } catch(e) { e }",
    "try { try { throw 1 } catch(e) { throw e } } catch(e2) { e2 }",
}
```

### Category 8: Object/Property Edge Cases

These exercise the runtime observation layer (property descriptors, prototype chains):

```go
seeds := []string{
    "Object.create(null)",
    "const obj = { get x() { return 1 } }; obj.x",
    "const key = 'a'; const obj = { [key]: 1 }; obj",
    "const s = Symbol('test'); const obj = { [s]: 1 }; obj",
    "Object.defineProperty({}, 'x', { value: 1, writable: false })",
    "new Proxy({}, { get: () => 42 })",
    "Object.freeze({ a: 1 })",
    "Object.seal({ a: 1 })",
}
```

### Category 9: JS Type Coercion Wat Moments

```go
seeds := []string{
    "[] + []",
    "[] + {}",
    "{} + []",
    "+true",
    "!!''",
    "'1' - 1",
    "'1' + 1",
    "null == undefined",
    "null === undefined",
}
```

### Category 10: Patterns That Stress Serialization

These produce values that go through JSON marshaling during persistence:

```go
seeds := []string{
    "new Map([[1, 'a'], [2, 'b']])",
    "new Set([1, 2, 3])",
    "new Date()",
    "new Error('test')",
    "const arr = new Array(1000).fill(0); arr",
    "JSON.parse('{\"a\":1}')",
    "const circular = {}; circular.self = circular; circular", // may hang or error
}
```

---

## Phased Implementation Plan

### Phase 1: Core Harnesses (Week 1)

**Goal**: Get three fuzz harnesses running in CI with good seed corpora.

#### Harness 1: `FuzzEvaluateRaw`

**File**: `fuzz/fuzz_evaluate_raw_test.go`

- Creates fresh `replapi.App` with `ProfileRaw` per invocation.
- Single string input.
- 5-second context timeout.
- Panic recovery → `t.Fatalf`.
- Seed corpus: Categories 1, 5, 6, 9.

**Pseudocode**:
```
FuzzEvaluateRaw(f):
  seeds = [Categories 1, 5, 6, 9]
  for seed in seeds: f.Add(seed)
  
  f.Fuzz(func(t, source):
    factory = engine.NewBuilder().WithModules(DefaultRegistryModules()).Build()
    app = replapi.New(factory, NopLogger, WithProfile(ProfileRaw))
    session = app.CreateSession(ctx)
    
    defer func():
      if r := recover(); r != nil:
        t.Fatalf("panic: %v", r)
    
    ctx, cancel = context.WithTimeout(ctx, 5s)
    defer cancel()
    app.Evaluate(ctx, session.ID, source)
  )
```

#### Harness 2: `FuzzEvaluateInstrumented`

**File**: `fuzz/fuzz_evaluate_instrumented_test.go`

- Creates fresh `replapi.App` with `ProfileInteractive` per invocation.
- Single string input.
- Exercises: parse → rewrite → execute → observe.
- Seed corpus: Categories 1-8.

#### Harness 3: `FuzzSessionSequence`

**File**: `fuzz/fuzz_session_sequence_test.go`

- Two string inputs: first evaluate, second evaluate.
- Creates one session, evaluates both in sequence.
- Both raw and interactive variants.
- Seed corpus: pairs from Categories 2, 3.

**Estimated effort**: 2-3 days.

**Files to create**:
- `fuzz/fuzz_evaluate_raw_test.go`
- `fuzz/fuzz_evaluate_instrumented_test.go`
- `fuzz/fuzz_session_sequence_test.go`
- `fuzz/testutil_test.go` (shared helpers)

---

### Phase 2: Advanced Harnesses (Week 2)

**Goal**: Cover persistence, restore, and isolated components.

#### Harness 4: `FuzzPersistenceRoundTrip`

**File**: `fuzz/fuzz_persistence_test.go`

- Creates persistent app, evaluates a sequence, closes store.
- Opens new persistent app with same store, restores session.
- Evaluates continuation input on restored session.
- Verifies state consistency (binding count, cell count).
- **Two or three string inputs**.

**Pseudocode**:
```
FuzzPersistenceRoundTrip(f):
  f.Fuzz(func(t, seed, restore, continuation):
    // Phase 1: Seed
    store1 = repldb.Open(tempfile)
    app1 = replapi.New(factory, NopLogger, WithProfile(Persistent), WithStore(store1))
    session = app1.CreateSession(ctx)
    app1.Evaluate(ctx, session.ID, seed)
    
    store1.Close()
    
    // Phase 2: Restore
    store2 = repldb.Open(same-file)
    app2 = replapi.New(factory, NopLogger, WithProfile(Persistent), WithStore(store2))
    snapshot = app2.Snapshot(ctx, session.ID) // triggers auto-restore
    
    // Phase 3: Continue
    app2.Evaluate(ctx, session.ID, restore)
    app2.Evaluate(ctx, session.ID, continuation)
    
    store2.Close()
  )
```

#### Harness 5: `FuzzRewriteIsolated`

**File**: `fuzz/fuzz_rewrite_test.go`

- Calls `jsparse.Analyze` + `buildRewrite` directly (no VM needed).
- Tests that `buildRewrite` never panics for any input.
- **Very fast**: 10,000+ exec/sec.

**Pseudocode**:
```
FuzzRewriteIsolated(f):
  f.Fuzz(func(t, source):
    analysis = jsparse.Analyze("<fuzz>", source, nil)
    report = buildRewrite(source, analysis, 1)
    
    // Invariant: TransformedSource should be valid JS
    if report.Mode == "async-iife-with-binding-capture":
      assert(report.TransformedSource != "")
      assert(strings.Contains(report.TransformedSource, "async function"))
  )
```

#### Harness 6: `FuzzGlobalSnapshot`

**File**: `fuzz/fuzz_globals_test.go`

- Creates interactive session.
- Evaluates input that produces unusual values.
- Calls `Snapshot` and checks that serialization doesn't panic.
- Focus on values that stress `runtimeValueKind`, `gojaValuePreview`, and `ownPropertiesView`.

**Estimated effort**: 3-4 days.

---

### Phase 3: CI Integration and Hardening (Week 3)

**Goal**: Run fuzz tests in CI with time-boxed execution.

#### CI Configuration

Add a GitHub Actions workflow that runs fuzz tests for 60 seconds each:

```yaml
# .github/workflows/fuzz.yml
name: Fuzz
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # nightly at 2 AM

jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - name: Fuzz evaluate-raw
        run: cd fuzz && go test -fuzz=FuzzEvaluateRaw -fuzztime=60s -v
      - name: Fuzz evaluate-instrumented
        run: cd fuzz && go test -fuzz=FuzzEvaluateInstrumented -fuzztime=60s -v
      - name: Fuzz session-sequence
        run: cd fuzz && go test -fuzz=FuzzSessionSequence -fuzztime=60s -v
      - name: Fuzz persistence
        run: cd fuzz && go test -fuzz=FuzzPersistenceRoundTrip -fuzztime=30s -v
      - name: Fuzz rewrite
        run: cd fuzz && go test -fuzz=FuzzRewriteIsolated -fuzztime=60s -v
```

#### Regression Corpus

When a crash is found:

1. The failing input is saved to `fuzz/testdata/fuzz/FuzzName/<hash>`.
2. Commit this file to the repository.
3. Future `go test -run=FuzzName/<hash>` will replay it.
4. Regular `go test ./fuzz/` runs all saved regression inputs.

#### Overnight Runs

For thorough testing, run overnight fuzzing locally:

```bash
# Run each harness for 1 hour
go test -fuzz=FuzzEvaluateRaw -fuzztime=1h -v
go test -fuzz=FuzzEvaluateInstrumented -fuzztime=1h -v
go test -fuzz=FuzzRewriteIsolated -fuzztime=1h -v
```

**Estimated effort**: 2-3 days.

---

### Phase 4: Advanced Coverage (Week 4+)

**Goal**: Extend coverage to modules, concurrency, and edge cases.

#### Harness 7: `FuzzWithModules`

- Enables native modules (`database`, `fs`, `exec`, `timer`) during fuzzing.
- Inputs that call `require('database')`, `require('fs')`, etc.
- **Risk**: Module side effects (file system writes, process execution). Must be sandboxed.

#### Harness 8: `FuzzConcurrency`

- Creates one session and calls `Evaluate` from multiple goroutines concurrently.
- Tests that the session mutex and service mutex prevent data races.
- Run with `go test -race`.

#### Harness 9: `FuzzConfigResolution`

- Fuzzes `normalizeConfig`, `validateConfig`, `resolveCreateSessionOptions`.
- Uses struct inputs (profile, policy flags).
- Quick to run, good for catching nil-dereference bugs.

#### Coverage Monitoring

Use Go's built-in coverage reporting to track which paths are exercised:

```bash
go test -fuzz=FuzzEvaluateRaw -fuzztime=5m -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Experiment Results

We built and ran four proof-of-concept fuzz experiments in the `scripts/` directory. Here are the results.

### Experiment 01: Basic REPL API Fuzz (`scripts/01-basic-replapi-fuzz/`)

**Type**: Standalone Go program (not using `go test -fuzz`).
**Profile**: Raw mode.
**Corpus**: 37 hand-picked JavaScript inputs.
**Result**: All 37 inputs processed without panics. The raw evaluation path handles edge cases well.

```
=== Results: passed=37 failed=0 panicked=0 total=37 ===
```

**Key observations**:
- Empty string, whitespace-only, and bare semicolons all produce `undefined` result without error.
- `()` produces a runtime error (expected — not a valid expression).
- JavaScript type coercion "wat" inputs (e.g., `[] + []`) all produce correct results.
- Unicode identifiers work correctly (`const 你好 = 'world'`).

### Experiment 02: Native Go Fuzz — Raw Mode (`scripts/02-native-go-fuzz/`)

**Type**: `go test -fuzz=FuzzEvaluateRaw`
**Profile**: Raw mode.
**Duration**: 12 seconds of coverage-guided fuzzing.
**Result**: 33,043 executions, 63 new interesting inputs found. No panics.

```
fuzz: elapsed: 12s, execs: 33043 (0/sec), new interesting: 63 (total: 227)
--- PASS: FuzzEvaluateRaw (12.01s)
```

**Performance**: Started at ~2,500 exec/sec (baseline coverage), ramped up to ~7,300 exec/sec (fuzzing phase), then settled.

### Experiment 03: Rewrite Pipeline Fuzz (`scripts/03-rewrite-pipeline-fuzz/`) 🐛

**Type**: `go test -fuzz=FuzzRewritePipeline`
**Profile**: Interactive mode (instrumented).
**Duration**: 0.23 seconds (crashed immediately).
**Result**: **PANIC FOUND** — empty string causes index out of range.

```
fuzz_test.go:96: PANIC in rewrite pipeline source="": runtime error: index out of range [0] with length 0

Failing input written to testdata/fuzz/FuzzRewritePipeline/5838cdfae7b16cde
```

**Root cause**: In `buildRewrite` (file `pkg/replsession/rewrite.go`), the function `finalExpressionStatement` accesses `result.Program.Body[len(result.Program.Body)-1]` without checking if the body is empty. When parsing an empty string, the program body is empty, causing the panic.

**Impact**: Any interactive or persistent session that receives an empty string evaluation will crash the server.

**Fix**: Add a length check in `finalExpressionStatement`:

```go
func finalExpressionStatement(result *jsparse.AnalysisResult, source string) (*ast.ExpressionStatement, string, bool) {
    if result == nil || result.Program == nil || len(result.Program.Body) == 0 {
        return nil, "", false
    }
    // ... rest of function
}
```

Note: This is also protected at the `Evaluate` level, which already checks `analysis == nil || analysis.ParseErr != nil` before calling the instrumented path. However, an empty string may produce a valid (but empty) program, bypassing the parse-error check.

### Experiment 04: Persistence Fuzz (`scripts/04-persistence-fuzz/`)

**Type**: `go test -fuzz=FuzzPersistenceRestore`
**Profile**: Persistent mode.
**Duration**: 10.5 seconds.
**Result**: 1,938 executions, no crashes. Persistence and restore work correctly for fuzz-derived inputs.

```
fuzz: elapsed: 11s, execs: 1938 (168/sec), new interesting: 0 (total: 4)
--- PASS: FuzzPersistenceRestore (10.57s)
```

---

## Detailed Code Walkthrough for Interns

This section provides a guided tour of every file you need to understand to build and maintain the fuzz infrastructure.

### How to Read This Section

For each file, we provide:
- **Location**: The file path relative to the repository root.
- **Purpose**: What the file does in plain English.
- **Key types/functions**: The important symbols you'll interact with.
- **Fuzzing relevance**: Why this file matters for fuzzing.

---

### File 1: `pkg/replapi/app.go` — The Facade

**Location**: `pkg/replapi/app.go` (~200 lines)
**Purpose**: `App` is the top-level API that users of the REPL system interact with. It combines a `replsession.Service` (live sessions) with an optional `repldb.Store` (persistence).

**Key types**:

```go
type App struct {
    config  Config
    service *replsession.Service
    store   *repldb.Store
}
```

**Key methods**:
- `New(factory, logger, opts...)` — creates an App. Returns error if config is invalid.
- `Evaluate(ctx, sessionID, source)` — **the main fuzzing target**.
- `CreateSession(ctx)` — creates a new in-memory session.
- `Restore(ctx, sessionID)` — replays persisted history into a fresh runtime.
- `ensureLiveSession(ctx, sessionID)` — auto-restore: if session not in memory, tries to restore from store.

**Fuzzing relevance**: This is the **entry point** for all fuzz harnesses.

---

### File 2: `pkg/replapi/config.go` — Profiles and Options

**Location**: `pkg/replapi/config.go` (~200 lines)
**Purpose**: Defines the three named profiles (raw, interactive, persistent) and the option functions used to configure the App.

**Key types**:

```go
type Profile string  // "raw", "interactive", "persistent"

type Config struct {
    Profile        Profile
    Store          *repldb.Store
    AutoRestore    bool
    SessionOptions replsession.SessionOptions
}
```

**Fuzzing relevance**: The profile determines which code path `Evaluate` takes.

---

### File 3: `pkg/replsession/service.go` — The Session Kernel

**Location**: `pkg/replsession/service.go` (~400 lines)
**Purpose**: Manages in-memory sessions. Each session is a `sessionState` with its own goja runtime.

**Key types**:

```go
type sessionState struct {
    id          string
    profile     string
    policy      SessionPolicy
    runtime     *engine.Runtime
    mu          sync.Mutex
    nextCellID  int
    cells       []*cellState
    bindings    map[string]*bindingState
    consoleSink []ConsoleEvent
    ignored     map[string]struct{}
}
```

**Fuzzing relevance**: The `sessionState` is where all mutable state lives. Understanding its fields is key to understanding what can go wrong across multi-cell evaluations.

---

### File 4: `pkg/replsession/evaluate.go` — The Pipeline

**Location**: `pkg/replsession/evaluate.go` (~350 lines)
**Purpose**: Implements the two evaluation paths: raw and instrumented.

**Instrumented path flow**:
1. `snapshotGlobals(before)`
2. `executeWrapped(rewrite)` — runs the transformed IIFE source on the VM
3. `snapshotGlobals(after)`
4. `diffGlobals(before, after, bindings)`
5. Classify diffs: new bindings vs leaked globals
6. `refreshBindingRuntimeDetails` — introspects each binding's value
7. `persistCell` — writes to SQLite if persistent

**Fuzzing relevance**: This is the **highest-value file** for fuzzing.

---

### File 5: `pkg/replsession/rewrite.go` — The IIFE Wrapper

**Location**: `pkg/replsession/rewrite.go` (~200 lines)
**Purpose**: Transforms user JavaScript source into an async IIFE that captures bindings.

**The transformation** (example):

```javascript
// INPUT:  const x = 1; x + 1
// OUTPUT:
(async function () {
  let __ggg_repl_last_1__;
  const x = 1; __ggg_repl_last_1__ = (x + 1);
  return {
    "__ggg_repl_bindings_1__": {
      "x": (typeof x === "undefined" ? undefined : x)
    },
    "__ggg_repl_last_1__": (typeof __ggg_repl_last_1__ === "undefined" ? undefined : __ggg_repl_last_1__)
  };
})()
```

**Key functions**:
- `buildRewrite(source, analysis, cellID)` — main transformation function.
- `declaredNamesFromResult(result)` — extracts names from root scope.
- `finalExpressionStatement(result, source)` — detects if the last statement is an expression.
- `replaceSourceRange(source, start, end, replacement)` — string slicing by AST indices.

**Known bug**: Empty string causes panic in `finalExpressionStatement` because `result.Program.Body` is empty.

**Fuzzing relevance**: **Critical**. The string slicing is fragile.

---

### File 6: `pkg/replsession/policy.go` — Session Policy

**Location**: `pkg/replsession/policy.go` (~150 lines)
**Purpose**: Defines the three-layer policy that controls session behavior.

**Profile defaults**:

| Feature | Raw | Interactive | Persistent |
|---------|-----|-------------|------------|
| Eval mode | raw | instrumented | instrumented |
| Capture last expr | no | yes | yes |
| Top-level await | no | yes | yes |
| Timeout (ms) | 5000 | 5000 | 5000 |
| Static analysis | no | yes | yes |
| Runtime snapshot | no | yes | yes |
| Binding tracking | no | yes | yes |
| Console capture | no | yes | yes |
| JSDoc extraction | no | yes | yes |
| Persistence | no | no | yes |

---

### File 7: `pkg/replsession/observe.go` — Runtime Observation

**Location**: `pkg/replsession/observe.go` (~300 lines)
**Purpose**: Implements global snapshotting, binding tracking, and runtime introspection.

**Key functions**:
- `snapshotGlobals(ctx)` — iterates over all non-builtin global keys.
- `diffGlobals(before, after, bindings)` — computes added/updated/removed diffs.
- `refreshBindingRuntimeDetails(ctx)` — introspects each binding's runtime value.
- `runtimeValueKind(value)` — classifies a goja value.

**Fuzzing relevance**: The `runtimeValueKind` switch doesn't handle all possible Go types that `value.Export()` can return.

---

## Risks, Alternatives, and Open Questions

### Risks

1. **Performance impact on CI**: Fuzz tests are CPU-intensive. Running them for 60 seconds per harness adds ~5 minutes to CI. Mitigation: only run on `main` branch and nightly, not every PR.

2. **Non-deterministic failures**: Go's coverage-guided fuzzer is deterministic within a single run, but different runs may find different bugs. Mitigation: commit all crash-reproducing inputs to the repository.

3. **Module side effects**: Enabling native modules (`database`, `fs`, `exec`) during fuzzing could cause real file system or process operations. Mitigation: Phase 1-3 fuzzing runs without modules; Phase 4 uses sandboxed temporary directories.

4. **Event loop goroutine leaks**: Each `engine.Factory.NewRuntime()` starts a new event loop goroutine. If the runtime isn't closed properly, goroutines leak. Mitigation: always call `app.DeleteSession()` or ensure the test process exits cleanly.

5. **SQLite file leaks**: Persistent mode tests create temp databases. Mitigation: use `t.TempDir()` which is cleaned up automatically.

### Alternatives Considered

1. **Using `go-fuzz` (third-party)**: Go's native fuzzing (1.18+) is sufficient and doesn't require external tools. Rejected.

2. **Using libFuzzer via CGo**: Could achieve higher throughput, but adds CGo complexity. Deferred to Phase 4 if needed.

3. **Property-based testing with `rapid`**: The `rapid` library provides a different API style but similar coverage-guided mutation. Go native fuzzing is sufficient. Rejected.

4. **Snapshot testing**: Compare full `EvaluateResponse` JSON against golden files. Useful for regression but not fuzzing. Can be added alongside fuzz tests.

### Open Questions

1. **Should we fix the empty-string panic before landing fuzz tests?** Recommendation: Yes, fix it first, then land the fuzz infrastructure. The fix is trivial (one line guard).

2. **Should fuzz tests live in `./fuzz/` or alongside the code?** Recommendation: `./fuzz/` for organization, with shared test helpers in `./fuzz/testutil_test.go`.

3. **How to handle the `eval()` builtin?** Goja supports `eval()`, which could execute arbitrary code generated by the fuzzer. This is actually desirable — it tests the VM's handling of dynamically generated code.

4. **What coverage percentage should we target?** Recommendation: 70%+ for `pkg/replsession/` and `pkg/replapi/`. Use `go test -coverprofile` to measure.

---

## Key File References

### Core Pipeline (highest priority for fuzzing)

| File | Lines | Role |
|------|-------|------|
| `pkg/replapi/app.go` | ~200 | Facade — entry point for all harnesses |
| `pkg/replapi/config.go` | ~200 | Profile/policy configuration |
| `pkg/replsession/service.go` | ~400 | Session kernel — creates/manages sessions |
| `pkg/replsession/evaluate.go` | ~350 | Evaluation pipeline (raw + instrumented) |
| `pkg/replsession/rewrite.go` | ~200 | IIFE wrapper — highest crash risk |
| `pkg/replsession/policy.go` | ~150 | Policy types and defaults |
| `pkg/replsession/observe.go` | ~300 | Runtime observation (globals, bindings) |
| `pkg/replsession/persistence.go` | ~200 | SQLite write layer |
| `pkg/replsession/types.go` | ~200 | All view/report/summary types |

### Infrastructure

| File | Lines | Role |
|------|-------|------|
| `engine/factory.go` | ~200 | Runtime factory |
| `engine/runtime.go` | ~100 | Runtime lifecycle |
| `pkg/runtimeowner/runner.go` | ~200 | Thread-safe VM access |
| `pkg/runtimeowner/types.go` | ~30 | Runner/CallFunc/PostFunc interfaces |
| `pkg/runtimebridge/runtimebridge.go` | ~50 | VM-to-bindings lookup |
| `pkg/repldb/store.go` | ~100 | SQLite store |
| `pkg/repldb/types.go` | ~80 | Persistence record types |

### Experiment Scripts

| Script | Location | Purpose |
|--------|----------|--------|
| 01 | `scripts/01-basic-replapi-fuzz/main.go` | Standalone corpus runner |
| 02 | `scripts/02-native-go-fuzz/fuzz_test.go` | Native Go fuzz: raw + instrumented + lifecycle |
| 03 | `scripts/03-rewrite-pipeline-fuzz/fuzz_test.go` | Native Go fuzz: instrumented (found crash!) |
| 04 | `scripts/04-persistence-fuzz/fuzz_test.go` | Native Go fuzz: persistence round-trip |

---

## Quick-Start Guide for the Intern

### Day 1: Reproduce the crash

```bash
cd ttmp/2026/04/20/GOJA-050--fuzzing-go-go-goja-replapi-analysis-design-and-implementation-guide/scripts/03-rewrite-pipeline-fuzz
go test -fuzz=FuzzRewritePipeline -v -fuzztime=5s
# You should see the panic on empty string
```

### Day 2: Run all experiments

```bash
# Experiment 01
cd scripts/01-basic-replapi-fuzz && go run main.go

# Experiment 02
cd scripts/02-native-go-fuzz && go test -fuzz=FuzzEvaluateRaw -v -fuzztime=10s

# Experiment 04
cd scripts/04-persistence-fuzz && go test -fuzz=FuzzPersistenceRestore -v -fuzztime=10s
```

### Day 3: Create the `./fuzz/` package

1. Create `fuzz/testutil_test.go` with shared helpers (`newTestFactory`, `newRawApp`, `newInteractiveApp`, `newPersistentApp`, `safeEvaluate`).
2. Create `fuzz/fuzz_evaluate_raw_test.go` — copy from experiment 02.
3. Create `fuzz/fuzz_evaluate_instrumented_test.go` — copy from experiment 03, fix the empty-string bug first.
4. Create `fuzz/fuzz_session_sequence_test.go` — copy from experiment 02's lifecycle test.
5. Run `go test ./fuzz/ -v` to verify all seed inputs pass.

### Week 2: Extend coverage

- Add the persistence round-trip harness.
- Add the isolated rewrite harness.
- Curate the full seed corpus from all 10 categories.
- Add the CI workflow.

---

## Glossary

| Term | Definition |
|------|------------|
| **goja** | A pure-Go ECMAScript 5.1+ interpreter (`github.com/dop251/goja`) |
| **goja_nodejs** | Node.js compatibility layer for goja (event loop, require, console) |
| **replapi** | The public API facade (`pkg/replapi/`) that manages REPL sessions |
| **replsession** | The session kernel (`pkg/replsession/`) that runs the evaluation pipeline |
| **repldb** | The SQLite persistence layer (`pkg/repldb/`) |
| **runtimeowner** | Thread-safe VM access wrapper (`pkg/runtimeowner/`) |
| **runtimebridge** | VM-to-bindings lookup (`pkg/runtimebridge/`) |
| **IIFE** | Immediately Invoked Function Expression — the pattern used to wrap cell source |
| **Profile** | A named preset (raw/interactive/persistent) that controls session behavior |
| **Policy** | A three-layer configuration (Eval + Observe + Persist) within a profile |
| **Instrumented** | The evaluation path that wraps source in an async IIFE and captures bindings |
| **Raw** | The evaluation path that runs source directly without transformation |
| **Seed corpus** | Initial set of inputs given to the fuzzer before mutation begins |
| **Coverage-guided** | The fuzzer mutates inputs to explore new code paths |

