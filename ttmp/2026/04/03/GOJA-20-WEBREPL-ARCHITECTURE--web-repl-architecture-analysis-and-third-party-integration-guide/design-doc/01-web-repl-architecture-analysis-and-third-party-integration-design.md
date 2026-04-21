---
Title: Web REPL Prototype Architecture Analysis and Third-Party Integration Design
Ticket: GOJA-20-WEBREPL-ARCHITECTURE
Status: active
Topics:
    - webrepl
    - architecture
    - rest-api
    - llm-agent-integration
    - persistent-repl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/web-repl/main.go
      Note: CLI entry point
    - Path: engine/factory.go
      Note: Factory builder pattern for runtime creation
    - Path: engine/runtime.go
      Note: Runtime struct with lifecycle management
    - Path: pkg/inspector/runtime/introspect.go
      Note: Object inspection and prototype chain walking
    - Path: pkg/jsparse/analyze.go
      Note: Analyze() entry point for static analysis pipeline
    - Path: pkg/jsparse/resolve.go
      Note: Full lexical scope resolver
    - Path: pkg/webrepl/rewrite.go
      Note: Async IIFE source rewriting
    - Path: pkg/webrepl/server.go
      Note: HTTP routing and embedded static assets
    - Path: pkg/webrepl/service.go
      Note: Core evaluation pipeline
    - Path: pkg/webrepl/types.go
      Note: All JSON-serializable types for REST API
ExternalSources: []
Summary: Historical prototype-centered analysis of the staged webrepl package. Useful as implementation evidence, but superseded as the recommended architecture by design-doc/02.
LastUpdated: 2026-04-03T16:30:04.776552774-04:00
WhatFor: Use this document to understand how the original staged webrepl prototype was put together and what subsystems it exercised.
WhenToUse: Use when tracing the existing prototype or comparing the new CLI/server-first design against the earlier webrepl-centered plan.
---











# Web REPL Architecture Analysis and Third-Party Integration Design

## Status Note

This document remains useful as a detailed analysis of the staged `pkg/webrepl` prototype.
It is no longer the recommended target architecture for implementation ordering.
For the current recommended plan, read `design-doc/02-cli-and-server-first-persistent-repl-architecture-and-implementation-guide.md` first.

## 1. Executive Summary

The `pkg/webrepl` package is a prototype that exposes a **persistent, session-based JavaScript REPL** over HTTP. It was built as a first exploration by our PhD researcher to answer a single question: *can we give an LLM agent (or a human via a browser) the ability to evaluate JavaScript in a long-lived goja process, while surfacing rich static-analysis and runtime-introspection metadata alongside every evaluation result?*

The prototype answers "yes" convincingly. It composes four existing subsystems—the **engine** runtime factory, the **jsparse** static-analysis pipeline, the **inspector** runtime-introspection library, and the **hashiplugin** plugin host—into a thin HTTP service that manages named sessions, rewrites user code into a safe async-IIFE wrapper, evaluates it inside a goja VM, and returns a detailed JSON report covering parse diagnostics, scope resolution, AST/CST snapshots, the rewritten source, runtime diffs, binding provenance, and console output.

The long-term goal is to turn this into a **reusable building block** so that any third-party Go package that embeds `go-go-goja` can expose its own REST API and CLI verbs, letting LLM agents interact with a long-running JS process—querying globals, calling functions, inspecting state, and iterating on code—without writing their own session management, rewriting, or introspection logic.

This document is written for a new intern joining the project. It explains every moving part, how they connect, where the boundaries are, and how we plan to evolve the prototype into a production-quality, extensible library.

---

## 2. Problem Statement and Scope

### 2.1 The problem

LLM agents that generate JavaScript today operate in a fire-and-forget mode: they emit a snippet, a tool executes it, and the result comes back as a flat string. There is no persistent state between calls, no way to ask "what globals exist?", no way to see how a `class` was rewritten, and no way to distinguish a parse error from a runtime exception from a leaked global.

We need a **persistent JS runtime** with:

1. **Named sessions** — create, evaluate cells, snapshot state, destroy.
2. **Rich introspection** — after every cell, the caller gets static analysis (bindings, scope tree, AST, CST, diagnostics) and runtime analysis (global diffs, binding metadata, prototype chains, function-to-source mappings).
3. **Safe rewriting** — lexical declarations (`let`, `const`, `class`) must survive across cells even though goja evaluates each snippet in global scope.
4. **Extensibility** — third-party packages must be able to register native modules, add REST routes, or expose CLI verbs on top of this infrastructure.
5. **Console capture** — `console.log/warn/error/…` output must be captured per-cell and returned in the response, not lost to stderr.

### 2.2 What is in scope

- The `pkg/webrepl` package (4 Go files, ~1 734 lines) and its embedded static UI.
- The `cmd/web-repl` binary (118 lines).
- The supporting subsystems: `engine/`, `pkg/jsparse/`, `pkg/inspector/`, `pkg/hashiplugin/host/`, `pkg/runtimeowner/`.

### 2.3 What is out of scope

- The TUI inspector (Bubble Tea / bobatea integration).
- The goja VM internals themselves.
- The hashiplugin gRPC contract definition (only the host-side wiring matters here).

---

## 3. Repository Map and Key Packages

The following ASCII diagram shows how the webrepl package relates to the rest of the repository. Arrows mean "imports / depends on".

```
┌───────────────────────────────────────────────────────────────────────┐
│                        cmd/web-repl/main.go                          │
│  (Cobra CLI entry point — wires flags, builds factory, starts HTTP)  │
└───────────┬───────────────────────────┬───────────────────────────────┘
            │                           │
            ▼                           ▼
┌───────────────────────┐   ┌───────────────────────────────────────────┐
│  pkg/hashiplugin/host │   │              pkg/webrepl                  │
│  RuntimeSetup         │   │  ┌─────────┐ ┌──────────┐ ┌───────────┐  │
│  (plugin discovery +  │──▶│  │ server  │ │ service  │ │ rewrite   │  │
│   registrar wiring)   │   │  │ .go     │ │ .go      │ │ .go       │  │
└───────────────────────┘   │  └────┬────┘ └────┬─────┘ └─────┬─────┘  │
                            │       │           │             │         │
                            │       │     ┌─────▼─────┐      │         │
                            │       │     │  types.go  │      │         │
                            │       │     └───────────┘      │         │
                            │       │           │             │         │
                            │  ┌────▼───────────▼─────────────▼──────┐  │
                            │  │           static/                   │  │
                            │  │  index.html  app.js  app.css        │  │
                            │  └─────────────────────────────────────┘  │
                            └──────────┬────────────────────────────────┘
                                       │
          ┌────────────────────────────┼────────────────────────────┐
          │                            │                            │
          ▼                            ▼                            ▼
┌──────────────────┐     ┌──────────────────────┐     ┌──────────────────────┐
│    engine/       │     │    pkg/jsparse/       │     │   pkg/inspector/     │
│  Factory         │     │  Analyze()            │     │   analysis/          │
│  FactoryBuilder  │     │  AnalysisResult       │     │     Session          │
│  Runtime         │     │  Index, Resolution    │     │     CrossReferences  │
│  ModuleSpec      │     │  TSParser (tree-sit.) │     │   runtime/           │
│  RuntimeInit.    │     │  BindingKind          │     │     InspectObject    │
│  RuntimeModReg.  │     │  ScopeRecord          │     │     WalkPrototype    │
└──────────────────┘     └──────────────────────┘     │     ValuePreview     │
        │                                              │     MapFunctionToSrc │
        ▼                                              │   core/              │
┌──────────────────┐                                   │     members          │
│ pkg/runtimeowner │                                   └──────────────────────┘
│  Runner          │
│  Call(ctx,op,fn) │
│  (single-thread  │
│   goja access)   │
└──────────────────┘
```

### 3.1 File-by-file inventory of `pkg/webrepl`

| File | Lines | Responsibility |
|------|-------|----------------|
| `types.go` | 299 | All JSON-serializable request/response types. No logic. |
| `server.go` | 125 | HTTP mux, embedded static assets, REST endpoint wiring. |
| `service.go` | 880 | Session lifecycle, evaluation pipeline, runtime snapshots, binding bookkeeping. |
| `rewrite.go` | 430 | Source-to-source transformation (async IIFE wrapper), static report construction, CST flattening. |
| `static/index.html` | 124 | Bootstrap 5 single-page layout with editor, session panel, tabs. |
| `static/app.js` | 230 | Vanilla JS client: session create, evaluate, render JSON. |
| `static/app.css` | 48 | Dark-theme code panels, monospace fonts, console styling. |

---

## 4. Architectural Deep Dive

### 4.1 Lifecycle: from HTTP request to JSON response

When the user (or an LLM agent) sends `POST /api/sessions/{id}/evaluate` with `{"source": "const x = 42; x + 1"}`, the following pipeline executes:

```
  HTTP request
       │
       ▼
  ┌─ server.go ─────────────────────────────────────────────────┐
  │  1. Parse JSON body → EvaluateRequest{Source}               │
  │  2. Call service.Evaluate(ctx, sessionID, source)           │
  └──────────────────────────────────┬──────────────────────────┘
                                     │
       ┌─────────────────────────────▼──────────────────────────┐
       │  service.go: Evaluate()                                │
       │                                                        │
       │  A. Lock session mutex                                 │
       │  B. Assign cell ID (monotonic counter)                 │
       │  C. Parse source with tree-sitter → cstRoot            │
       │  D. Run jsparse.Analyze(filename, source) → analysis   │
       │  E. Build static report from analysis + CST            │
       │  F. Build rewrite from source + analysis + cellID      │
       │  G. Snapshot globals BEFORE evaluation                 │
       │  H. Execute rewritten source in goja VM                │
       │     - If result is a Promise → poll until resolved     │
       │     - Extract persisted bindings + last expression     │
       │  I. Snapshot globals AFTER evaluation                  │
       │  J. Diff before/after → added/updated/removed/leaked  │
       │  K. Upsert binding bookkeeping                         │
       │  L. Refresh runtime details for all tracked bindings   │
       │  M. Assemble CellReport + SessionSummary               │
       │  N. Return EvaluateResponse{Session, Cell}             │
       └────────────────────────────────────────────────────────┘
```

Every step in this pipeline is worth understanding in detail. Let us walk through each one.

### 4.2 Step C–E: Static Analysis Pipeline

**Entry point:** `jsparse.Analyze(filename, source, opts)` in `pkg/jsparse/analyze.go` (line 37).

This function does three things in sequence:

1. **Parsing** — calls `parser.ParseFile` from the `dop251/goja` parser package. This produces `*ast.Program`, the goja AST. Parse errors are recorded but do not abort; a partial AST may still be available.

2. **Indexing** — calls `BuildIndex(program, source)` in `pkg/jsparse/index.go`. This walks every AST node recursively and produces:
   - `Index.Nodes` — a flat `map[NodeID]*NodeRecord` with start/end offsets, line/col, depth, display labels, and source snippets.
   - `Index.RootID` — the root node ID.
   - `Index.OrderedByStart` — node IDs sorted by (start ASC, end DESC) for fast containment lookups.

3. **Resolution** — calls `Resolve(program, idx)` in `pkg/jsparse/resolve.go` (1001 lines). This is a full lexical-scope resolver that walks the AST and produces:
   - `Resolution.Scopes` — a map from `ScopeID` to `ScopeRecord`, each containing `Bindings` (name → `BindingRecord`).
   - `Resolution.NodeBinding` — for every `Identifier` AST node, which `BindingRecord` it resolves to.
   - `Resolution.Unresolved` — identifiers that could not be resolved (free variables, references to globals not declared in the snippet).

The resolver understands all JavaScript binding forms: `var`, `let`, `const`, `function`, `class`, `for`-loop initializers, `catch` parameters, and destructuring patterns.

**What the webrepl does with this:**

In `rewrite.go:buildStaticReport()` (line 182), the analysis result is converted into a `StaticReport` containing:

- `Diagnostics` — parse errors as severity+message pairs.
- `TopLevelBindings` — names, kinds, declaration lines, snippets, and reference counts for every root-scope binding.
- `Unresolved` — identifiers that reference globals or undeclared names.
- `References` — cross-reference groups (via `inspectoranalysis.CrossReferences`).
- `Scope` — a recursive `ScopeView` tree mirroring the resolver's scope hierarchy.
- `AST` — flattened AST rows (node ID, indented title, byte range + snippet).
- `CST` — flattened tree-sitter CST rows (depth, kind, text, row/col, error/missing flags).
- `FinalExpression` — the source range of the last expression statement, if any.
- `Summary` — compact label/value facts (diagnostic count, binding count, node counts).

**Why this matters for LLM agents:** An agent can inspect `topLevelBindings` to know what names a cell introduced, check `unresolved` to see what globals the code depends on, and read `diagnostics` to detect syntax errors before runtime—all without executing anything.

### 4.3 Step F: The Rewrite Pipeline

**Entry point:** `buildRewrite(source, analysis, cellID)` in `rewrite.go` (line 10).

**The core problem:** goja's `vm.RunString(source)` evaluates code in global scope. If the user writes `let x = 1` in cell 1 and `x + 1` in cell 2, cell 2 fails because `let` declarations do not persist across separate `RunString` calls—they are block-scoped to the implicit block of that evaluation.

**The solution:** Wrap each cell's source inside an **async IIFE** (Immediately Invoked Function Expression) that:

1. Captures all top-level declarations into a return object.
2. Captures the last expression's value into a helper variable.
3. After the IIFE resolves, the service reads the returned bindings and mirrors them back onto the goja global object via `vm.Set(name, value)`.

Here is what the rewriter produces for `const answer = 42; answer + 1`:

```javascript
(async function () {
  let __ggg_repl_last_1__;
  const answer = 42;
  __ggg_repl_last_1__ = (answer + 1);
  return {
    "__ggg_repl_bindings_1__": {
      "answer": (typeof answer === "undefined" ? undefined : answer)
    },
    "__ggg_repl_last_1__": (typeof __ggg_repl_last_1__ === "undefined"
      ? undefined : __ggg_repl_last_1__)
  };
})()
```

**Rewrite operations in detail:**

1. **`declaredNamesFromResult`** (line 81): Extracts all names from the root scope's `Bindings` map. These are the names that need to be captured.

2. **`finalExpressionStatement`** (line 97): Checks whether the last top-level statement is an `*ast.ExpressionStatement`. If so, it extracts its source text. The rewriter replaces that statement with `__ggg_repl_last_N__ = (<expr>);` so the REPL can report the value.

3. **`replaceSourceRange`** (line 121): A byte-offset–based string replacement. The goja AST uses 1-based byte offsets (`Idx0()`, `Idx1()`), and this function converts them to 0-based Go string indices.

4. **IIFE assembly** (line 42–76): A `strings.Builder` emits the `(async function () { ... })()` wrapper with the helper variable declaration, the (possibly rewritten) body, and the return statement that captures both the binding object and the last-expression helper.

**The `RewriteReport`** returned alongside the transformed source documents every operation:

```json
{
  "mode": "async-iife-with-binding-capture",
  "declaredNames": ["answer"],
  "helperNames": ["__ggg_repl_last_1__", "__ggg_repl_bindings_1__"],
  "operations": [
    {"kind": "wrap", "detail": "wrap cell source in an async IIFE..."},
    {"kind": "capture-bindings", "detail": "return top-level declarations..."},
    {"kind": "capture-last-expression", "detail": "replace the final top-level expression..."}
  ],
  "transformedSource": "(async function () { ... })()"
}
```

### 4.4 Step G–I: Runtime Execution and Snapshot Pipeline

**Thread safety:** The goja VM is single-threaded. All VM access goes through `runtimeowner.Runner.Call(ctx, opName, func)`, which serializes calls onto the owner goroutine. The `sessionState.mu` mutex protects the session's own bookkeeping but delegates VM access to the Runner.

**Global snapshots** (`snapshotGlobals`, service.go line 304):

Before and after evaluation, the service iterates `vm.GlobalObject().Keys()`, skips builtins (via `inspectorruntime.IsBuiltinGlobal`), and records for each non-builtin key:

```go
GlobalStateView{
    Name:          name,
    Kind:          runtimeValueKind(val),  // "function", "object", "string", ...
    Preview:       gojaValuePreview(val, vm, 120),
    Identity:      fmt.Sprintf("%p", obj), // pointer identity for objects
    PropertyCount: len(obj.Keys()),
}
```

**Diff algorithm** (`diffGlobals`, service.go line 327):

A straightforward set-difference comparison: names in `before` but not `after` are "removed"; names in `after` but not `before` are "added"; names in both but with changed `Preview`, `Kind`, `Identity`, or `PropertyCount` are "updated".

Each diff entry is tagged with `SessionBound: true/false` to indicate whether the name was explicitly tracked by the session's binding bookkeeping.

**Promise handling** (`waitPromise`, service.go line 271):

The async IIFE returns a `*goja.Promise`. The service polls it in a loop with 5ms sleep intervals, checking `promise.State()`:
- `Pending` → sleep and retry.
- `Fulfilled` → return `promise.Result()`.
- `Rejected` → return an error with the rejection reason.

This is a simple busy-wait. A production version could use goja's event loop integration or a channel-based notification.

### 4.5 Step J–L: Binding Bookkeeping

The session maintains `bindings map[string]*bindingState`, a persistent registry of every named value the user has introduced. Each `bindingState` tracks:

| Field | Source | Description |
|-------|--------|-------------|
| `Name` | parser or runtime | The JavaScript identifier name |
| `Kind` | `jsparse.BindingKind` | `var`, `let`, `const`, `function`, `class`, `param`, `catch` |
| `Origin` | service logic | `"declared-top-level"` or `"runtime-global-diff"` |
| `DeclaredInCell` | service logic | Which cell first introduced this name |
| `LastUpdatedCell` | service logic | Which cell last changed this name's value |
| `DeclaredLine` | parser | Source line of the declaration within its cell |
| `DeclaredSnippet` | parser | The AST node's source text |
| `Static` | parser | References, parameters, extends, class members |
| `Runtime` | runtime | Current value kind, preview, own properties, prototype chain, function→source mapping |

**Three binding discovery paths:**

1. **Persisted-by-wrap** — names returned by the IIFE's binding object. These are the "happy path": the user declared them with `let`/`const`/`function`/`class`, the rewriter captured them, and `persistWrappedReturn` mirrored them onto `vm.GlobalObject()`.

2. **Runtime-global-diff added** — names that appeared in the global diff but were NOT in the persisted set. These are "leaked globals": the user wrote `x = 42` (implicit global assignment) or mutated `globalThis.foo`. The service records them with `Origin: "runtime-global-diff"`.

3. **Runtime-global-diff updated** — names that already existed but whose value changed. If the name is already in the binding map, only `LastUpdatedCell` is bumped.

**`refreshBindingRuntimeDetails`** (service.go line 414): After every evaluation, the service iterates all tracked bindings and, for each one, reads the current runtime value from `vm.Get(name)`. It populates:

- `ValueKind` and `Preview` from `inspectorruntime.ValuePreview`.
- `OwnProperties` from `inspectorruntime.InspectObject` (capped at 20 properties).
- `PrototypeChain` from walking `obj.Prototype()` up to 4 levels, with 12 properties each.
- `FunctionMapping` — for function bindings, it calls `inspectorruntime.MapFunctionToSource` which tries to match the runtime function's name back to an AST `FunctionDeclaration` or class method in the declaring cell's analysis result.

### 4.6 Console Capture

**`installConsoleCapture`** (service.go line 241):

The service replaces the global `console` object with a custom goja object that has `log`, `info`, `debug`, `warn`, `error`, and `table` methods. Each method appends a `ConsoleEvent{Kind, Message}` to `sessionState.consoleSink`. The sink is drained and reset around each evaluation.

Messages are formatted by calling `inspectorruntime.ValuePreview` on each argument and joining them with spaces, mimicking browser console behavior.

### 4.7 The HTTP Layer

**`server.go`** defines five routes:

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/` | file server | Serves `static/index.html` |
| `GET` | `/static/*` | file server | Serves embedded CSS/JS |
| `POST` | `/api/sessions` | `CreateSession` | Creates a new session, returns `SessionSummary` |
| `GET` | `/api/sessions/{id}` | `Snapshot` | Returns current session state without evaluating |
| `DELETE` | `/api/sessions/{id}` | `DeleteSession` | Shuts down session runtime |
| `POST` | `/api/sessions/{id}/evaluate` | `Evaluate` | Evaluates one cell, returns `EvaluateResponse` |

Static assets are embedded via `//go:embed static/*` and served through `embed.FS`.

The routing uses `http.ServeMux` with manual path parsing (splitting on `/` and dispatching by segment count and method). There is no middleware, no authentication, no rate limiting, and no CORS headers. This is appropriate for a prototype but will need attention for production use.

### 4.8 The Web UI

The browser UI is a single-page application using Bootstrap 5 (CDN) and 230 lines of vanilla JavaScript. It maintains a global `state` object with `sessionId` and `lastResponse`.

**User flow:**

1. On page load, `createSession()` fires `POST /api/sessions`.
2. The user types code in a `<textarea>` and presses Ctrl+Enter or clicks "Run Cell".
3. `evaluateCell()` sends `POST /api/sessions/{id}/evaluate` with the source.
4. The response is rendered into four areas:
   - **Session panel** — ID, creation time, cell count, binding count, global count.
   - **Bindings table** — name, kind, preview, origin, declared cell, updated cell.
   - **History list** — reverse-chronological cell entries with source preview, result preview, status badge.
   - **Detail tabs** — Static (JSON), Rewrite (transformed source), Runtime (JSON), Raw JSON.

The UI is a developer tool, not a user-facing product. It exists to make the prototype's output visible and debuggable.

---

## 5. How the Webrepl Builds on Existing Subsystems

### 5.1 The `engine` package

The `engine` package provides the **Factory pattern** for creating configured goja runtimes:

```go
// Simplified flow in cmd/web-repl/main.go:
builder := engine.NewBuilder().WithModules(engine.DefaultRegistryModules())
// plugin host adds runtime module registrars:
builder = pluginSetup.WithBuilder(builder)
factory, _ := builder.Build()

// Later, per session:
rt, _ := factory.NewRuntime(ctx)
// rt.VM       = *goja.Runtime
// rt.Owner    = runtimeowner.Runner (serialized VM access)
// rt.Loop     = *eventloop.EventLoop
// rt.Require  = *require.RequireModule
```

**Key interfaces used by webrepl:**

- `engine.Factory` — immutable recipe for creating runtimes. The `Service` stores one and calls `NewRuntime()` per session.
- `engine.Runtime` — the per-session runtime bundle. The session stores it and uses `rt.Owner.Call()` for all VM operations.
- `engine.ModuleSpec` — static module registration (runs once per factory). Used by `DefaultRegistryModules()` to register `fs`, `exec`, `database` modules.
- `engine.RuntimeModuleRegistrar` — per-runtime module registration. Used by the hashiplugin `Registrar` to register discovered plugin modules.
- `engine.RuntimeInitializer` — per-runtime post-setup hooks.

### 5.2 The `pkg/jsparse` package

This is the **static analysis backbone**. The webrepl uses it for:

1. **`jsparse.Analyze()`** — the single entry point that returns `*AnalysisResult` containing the parsed AST, the node index, and the scope resolution.

2. **`jsparse.BindingKind`** — the enumeration (`var`, `let`, `const`, `function`, `class`, `param`, `catch`) used throughout the binding bookkeeping.

3. **`jsparse.Resolution`** — the scope tree and binding records. The rewriter reads `Resolution.Scopes[RootScopeID].Bindings` to discover which names need to be captured. The static report reads `Resolution.Unresolved` to report free variables.

4. **`jsparse.Index`** — the flat node map. Used for AST row generation, declaration snippet extraction, and cross-reference lookups.

5. **`jsparse.TSParser`** — the tree-sitter parser. Used independently of the goja parser to produce a CST (Concrete Syntax Tree) that preserves whitespace, comments, and error recovery nodes.

6. **`jsparse.NodeID`, `jsparse.ScopeID`** — type-safe IDs used in the index and resolution structures.

### 5.3 The `pkg/inspector` packages

Three sub-packages are used:

- **`inspector/analysis`** — `Session` wraps `*AnalysisResult` with convenience methods: `Globals()`, `BindingDeclLine()`, `FunctionMembers()`, `ClassMembers()`. Also provides `CrossReferences()` for finding all usages of a binding.

- **`inspector/runtime`** — `InspectObject()` returns own properties, `WalkPrototypeChain()` returns the prototype chain, `ValuePreview()` formats a goja value as a short string, `MapFunctionToSource()` maps runtime functions back to AST locations, `IsBuiltinGlobal()` filters out standard JS globals.

- **`inspector/core`** — `Member` struct used for class/function member introspection.

### 5.4 The `pkg/runtimeowner` package

The goja VM is not goroutine-safe. The `runtimeowner.Runner` interface provides a serialized `Call(ctx, opName, func)` method that ensures only one goroutine accesses the VM at a time. The webrepl service uses this for every VM operation: `runString`, `snapshotGlobals`, `persistWrappedReturn`, `refreshBindingRuntimeDetails`, `installConsoleCapture`.

### 5.5 The `pkg/hashiplugin/host` package

`RuntimeSetup` discovers HashiCorp go-plugin binaries on disk and produces a `RuntimeModuleRegistrar` that registers them as `require()`-able native modules in each new runtime. The `cmd/web-repl` binary accepts `--plugin-dir` and `--allow-plugin-module` flags to control this.

---

## 6. API Reference

### 6.1 REST API

#### Create Session

```
POST /api/sessions
Content-Type: application/json

Response 201:
{
  "session": {
    "id": "session-1",
    "createdAt": "2026-04-03T10:00:00Z",
    "cellCount": 0,
    "bindingCount": 0,
    "bindings": [],
    "history": [],
    "currentGlobals": [],
    "provenance": [...]
  }
}
```

#### Evaluate Cell

```
POST /api/sessions/{sessionId}/evaluate
Content-Type: application/json

Request:
{
  "source": "const x = 42;\nfunction greet(name) { return `hello ${name}`; }\ngreet('world')"
}

Response 200:
{
  "session": { ... },   // full SessionSummary after evaluation
  "cell": {
    "id": 1,
    "createdAt": "...",
    "source": "const x = 42; ...",
    "static": {
      "diagnostics": [],
      "topLevelBindings": [
        {"name": "greet", "kind": "function", "line": 2, "snippet": "function greet(name) { ... }"},
        {"name": "x", "kind": "const", "line": 1, "snippet": "const x = 42"}
      ],
      "unresolved": [],
      "references": [...],
      "scope": { "id": 0, "kind": "global", ... },
      "ast": [...],
      "cst": [...],
      "summary": [...]
    },
    "rewrite": {
      "mode": "async-iife-with-binding-capture",
      "declaredNames": ["greet", "x"],
      "transformedSource": "(async function () { ... })()",
      "operations": [...]
    },
    "execution": {
      "status": "ok",
      "result": "\"hello world\"",
      "durationMs": 2,
      "awaited": true,
      "console": [],
      "hadSideEffects": true
    },
    "runtime": {
      "beforeGlobals": [],
      "afterGlobals": [
        {"name": "greet", "kind": "function", "preview": "function greet(name) { ... }"},
        {"name": "x", "kind": "number", "preview": "42"}
      ],
      "diffs": [
        {"name": "greet", "change": "added", ...},
        {"name": "x", "change": "added", ...}
      ],
      "newBindings": ["greet", "x"],
      "persistedByWrap": ["greet", "x"],
      "currentCellValue": "\"hello world\""
    },
    "provenance": [...]
  }
}
```

#### Snapshot Session

```
GET /api/sessions/{sessionId}

Response 200:
{
  "session": { ... }  // SessionSummary
}
```

#### Delete Session

```
DELETE /api/sessions/{sessionId}

Response 200:
{
  "deleted": true
}
```

### 6.2 Key Go Types

```go
// The Service — manages sessions. One per process.
type Service struct {
    factory  *engine.Factory
    sessions map[string]*sessionState
}

// The session — one per client/agent conversation.
type sessionState struct {
    id        string
    runtime   *engine.Runtime      // owns the goja VM
    bindings  map[string]*bindingState  // persistent name registry
    cells     []*cellState              // evaluation history
}

// The cell — one per evaluation.
type CellReport struct {
    Static    StaticReport     // parser output
    Rewrite   RewriteReport    // transformation log
    Execution ExecutionReport  // runtime outcome
    Runtime   RuntimeReport    // global diffs + binding updates
}
```

---

## 7. Data Flow Diagram: Complete Cell Evaluation

```
User source string
        │
        ├──────────────────────────┐
        ▼                          ▼
   jsparse.Analyze()         TSParser.Parse()
        │                          │
        ▼                          ▼
  AnalysisResult              *TSNode (CST)
  ├─ Program (AST)                 │
  ├─ Index (nodes)                 │
  └─ Resolution (scopes)          │
        │                          │
        ├──────────┬───────────────┘
        ▼          ▼
  buildStaticReport(analysis, cstRoot)
        │
        ▼
  StaticReport { diagnostics, bindings, scope, AST rows, CST rows }
        │
        │
  buildRewrite(source, analysis, cellID)
        │
        ▼
  RewriteReport { transformedSource, declaredNames, operations }
        │
        ▼
  snapshotGlobals() → beforeGlobals
        │
        ▼
  executeWrapped(ctx, rewrite)
  ├─ runString(transformedSource) → goja.Value
  ├─ waitPromise(promise) → resolved value
  └─ persistWrappedReturn(value) → persisted names + last value
        │
        ▼
  snapshotGlobals() → afterGlobals
        │
        ▼
  diffGlobals(before, after) → diffs, added, updated, removed
        │
        ▼
  upsertDeclaredBinding() / upsertRuntimeDiscoveredBinding()
        │
        ▼
  refreshBindingRuntimeDetails()
  ├─ vm.Get(name) for each binding
  ├─ InspectObject() → own properties
  ├─ WalkPrototypeChain() → prototype levels
  └─ MapFunctionToSource() → source mapping
        │
        ▼
  Assemble CellReport + SessionSummary
        │
        ▼
  JSON response to caller
```

---

## 8. Gap Analysis: From Prototype to Production Library

### 8.1 What works well

1. **Clean separation of concerns.** Static analysis, rewriting, execution, and introspection are distinct phases with well-defined inputs and outputs.
2. **Rich introspection.** The binding metadata (static + runtime) is far richer than any existing JS REPL API.
3. **Provenance tracking.** Every section of the response documents how it was obtained.
4. **The rewrite strategy is sound.** The async-IIFE pattern correctly handles `let`/`const`/`class` persistence.

### 8.2 Gaps for third-party integration

| Gap | Description | Impact |
|-----|-------------|--------|
| **No `Service` interface** | `Service` is a concrete struct. Third parties cannot swap in custom session behavior. | Medium |
| **No hooks for custom routes** | The HTTP mux is built inside `NewHandler()` with no extension points. | High |
| **No CLI verb support** | There is no Cobra command tree that could be composed into another CLI. | High |
| **No middleware chain** | No auth, no rate limiting, no request logging middleware. | Medium |
| **No session timeout** | Sessions live forever until explicitly deleted. | Medium |
| **No streaming** | Console output and long evaluations block until complete. | Low for MVP |
| **No WebSocket** | The polling model works but is inefficient for interactive use. | Low for MVP |
| **No evaluation timeout** | A `while(true){}` cell hangs the session forever. | High |
| **Promise polling is busy-wait** | 5ms sleep loop wastes CPU. | Low |
| **No pagination** | Global snapshot and binding list can be arbitrarily large. | Medium |
| **Hardcoded limits** | AST/CST row limits, property limits are constants, not configurable. | Low |

### 8.3 Gaps for LLM agent ergonomics

| Gap | Description |
|-----|-------------|
| **No "list functions" endpoint** | Agents must parse the full session snapshot to find functions. |
| **No "describe binding" endpoint** | Getting detailed info about one binding requires the full snapshot. |
| **No "list completions" endpoint** | The `jsparse.ReplCompletion` infrastructure exists but is not exposed. |
| **No structured error codes** | Errors are free-form strings, not machine-parseable codes. |
| **No evaluation context/metadata** | No way for agents to tag cells with intent or metadata. |

---

## 9. Proposed Architecture for Third-Party Extensibility

### 9.1 Design goals

1. **A third-party Go package imports `pkg/webrepl` and gets a working REPL service with zero configuration.**
2. **Custom modules, routes, and CLI verbs can be composed on top.**
3. **LLM agents can interact through a clean REST API with targeted endpoints.**

### 9.2 Proposed layering

```
┌─────────────────────────────────────────────────────────────────┐
│  Third-party binary (e.g., myapp serve --repl)                  │
│  └── Composes: webrepl.Service + custom modules + custom routes │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│  pkg/webrepl (library layer)                                     │
│                                                                  │
│  ┌────────────┐  ┌──────────────┐  ┌───────────────────────┐    │
│  │  Service    │  │  Handler     │  │  CommandSet           │    │
│  │  (sessions, │  │  (REST API   │  │  (Cobra commands:     │    │
│  │   evaluate, │  │   + static   │  │   create, eval,       │    │
│  │   snapshot) │  │   assets)    │  │   snapshot, list,     │    │
│  └──────┬─────┘  └──────┬───────┘  │   describe, delete)   │    │
│         │               │          └───────────────────────┘    │
│         │               │                                        │
│  ┌──────▼───────────────▼──────────────────────────────────┐    │
│  │  ServiceOptions                                          │    │
│  │  - EvalTimeout, SessionTTL, MaxSessions                 │    │
│  │  - Middleware []func(http.Handler) http.Handler          │    │
│  │  - ExtraRoutes []Route                                   │    │
│  │  - OnSessionCreate / OnEvaluate hooks                    │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### 9.3 Proposed REST API extensions for LLM agents

```
GET  /api/sessions/{id}/bindings                → list all bindings (name, kind, preview)
GET  /api/sessions/{id}/bindings/{name}         → detailed binding info (static + runtime)
GET  /api/sessions/{id}/bindings/{name}/source  → declaration source snippet
GET  /api/sessions/{id}/globals                 → current non-builtin globals
GET  /api/sessions/{id}/completions?prefix=...  → REPL completion suggestions
POST /api/sessions/{id}/evaluate                → (existing) evaluate cell
GET  /api/sessions/{id}/history                 → cell history with pagination
GET  /api/sessions/{id}/history/{cellId}        → full CellReport for one cell
```

### 9.4 Proposed CLI verbs

```bash
# Start the server
webrepl serve --addr 127.0.0.1:3090 --plugin-dir ./plugins

# Scripted interaction (talks to a running server)
webrepl session create                          → prints session ID
webrepl eval --session <id> --source 'const x = 42'
webrepl eval --session <id> --file script.js
webrepl snapshot --session <id>                 → full JSON snapshot
webrepl bindings --session <id>                 → table of bindings
webrepl describe --session <id> --name greet    → detailed binding info
webrepl globals --session <id>                  → table of globals
webrepl history --session <id>                  → cell history
webrepl session delete --session <id>
```

### 9.5 Pseudocode: third-party integration

```go
package main

import (
    "github.com/go-go-golems/go-go-goja/engine"
    "github.com/go-go-golems/go-go-goja/pkg/webrepl"
    mymodules "mycompany/myapp/jsmodules"
)

func main() {
    // 1. Build a factory with custom modules
    factory, _ := engine.NewBuilder().
        WithModules(
            engine.DefaultRegistryModules(),
            mymodules.MyDatabaseModule(),   // custom native module
            mymodules.MyHTTPClientModule(), // another custom module
        ).
        Build()

    // 2. Create the REPL service with options
    svc := webrepl.NewService(factory, logger,
        webrepl.WithEvalTimeout(30 * time.Second),
        webrepl.WithSessionTTL(1 * time.Hour),
        webrepl.WithMaxSessions(100),
    )

    // 3. Build the HTTP handler with custom middleware and routes
    handler, _ := webrepl.NewHandler(svc,
        webrepl.WithMiddleware(authMiddleware, loggingMiddleware),
        webrepl.WithExtraRoutes(
            webrepl.Route{Method: "POST", Path: "/api/custom/deploy",
                Handler: myDeployHandler},
        ),
    )

    // 4. Serve
    http.ListenAndServe(":8080", handler)
}
```

---

## 10. Implementation Plan

### Phase 1: Hardening the prototype (1–2 weeks)

1. **Add evaluation timeout.** Wrap `runString` + `waitPromise` in a `context.WithTimeout`. Cancel the goja VM via `vm.Interrupt()` on timeout.
   - Files: `pkg/webrepl/service.go` (lines 261–300)

2. **Add session TTL and cleanup.** A background goroutine reaps sessions older than a configurable TTL.
   - Files: `pkg/webrepl/service.go` (new method `reapSessions`)

3. **Extract `ServiceOptions`.** Move hardcoded limits (`defaultASTRowLimit`, etc.) and new knobs (timeout, TTL, max sessions) into an options struct.
   - Files: `pkg/webrepl/service.go`

4. **Add structured error responses.** Define error codes (`PARSE_ERROR`, `RUNTIME_ERROR`, `SESSION_NOT_FOUND`, `EVAL_TIMEOUT`) and return them in a consistent `{"error": {"code": "...", "message": "..."}}` envelope.
   - Files: `pkg/webrepl/server.go`, `pkg/webrepl/types.go`

5. **Add request logging middleware.** Emit structured JSON logs for every request.
   - Files: `pkg/webrepl/server.go`

### Phase 2: LLM agent endpoints (1–2 weeks)

1. **Add `/api/sessions/{id}/bindings` endpoint.** List bindings with optional `?kind=function` filter.
2. **Add `/api/sessions/{id}/bindings/{name}` endpoint.** Return full `BindingView` for one name.
3. **Add `/api/sessions/{id}/completions?prefix=...` endpoint.** Wire to `jsparse.ReplCompletion`.
4. **Add `/api/sessions/{id}/history` with pagination.** `?offset=0&limit=20`.
5. **Add `/api/sessions/{id}/history/{cellId}` endpoint.** Return a single `CellReport`.

### Phase 3: Extensibility (1 week)

1. **Extract `Service` interface.** Define a `SessionManager` interface that `*Service` implements.
2. **Add `WithMiddleware` and `WithExtraRoutes` to `NewHandler`.** Use a variadic options pattern.
3. **Create `CommandSet` for CLI verbs.** A set of Cobra commands that talk to a `Service` (either in-process or via HTTP client).

### Phase 4: CLI binary (1 week)

1. **Add `serve` subcommand** (current `RunE` logic).
2. **Add `eval`, `snapshot`, `bindings`, `describe`, `globals`, `history` subcommands** using an HTTP client.
3. **Add `--format` flag** supporting `json`, `table`, `yaml`.

---

## 11. Testing Strategy

### 11.1 Unit tests

- **Rewrite tests:** Given source + fake analysis result → assert transformed source matches expected.
- **Diff tests:** Given before/after global maps → assert diffs, added, updated, removed.
- **Static report tests:** Given source → assert binding names, scope tree shape, unresolved identifiers.

### 11.2 Integration tests

- **Session lifecycle:** Create → evaluate 3 cells → snapshot → delete. Assert binding accumulation across cells.
- **Error handling:** Parse error cell → assert `status: "parse-error"`. Runtime error cell → assert `status: "runtime-error"`.
- **Console capture:** Evaluate `console.log("hello")` → assert console events in response.
- **Promise handling:** Evaluate `Promise.resolve(42)` → assert awaited and result.
- **Binding persistence:** Evaluate `let x = 1` in cell 1, then `x + 1` in cell 2 → assert result is `2`.

### 11.3 HTTP tests

- Use `httptest.NewServer` with the handler. Send real HTTP requests. Assert status codes, JSON structure, and content.

### 11.4 LLM agent simulation tests

- Script a sequence of create → eval → bindings → describe → eval → snapshot → delete using the HTTP client. Assert that the agent can discover all introduced names, get their types, and execute follow-up code that references them.

---

## 12. Risks, Alternatives, and Open Questions

### 12.1 Risks

| Risk | Mitigation |
|------|-----------|
| **goja VM hangs on infinite loops** | Phase 1 adds `vm.Interrupt()` with timeout |
| **Memory leak from abandoned sessions** | Phase 1 adds session TTL reaping |
| **Rewrite correctness edge cases** | Destructuring, `export`, generators, `with` — need thorough test coverage |
| **tree-sitter CGo dependency** | Optional; can degrade gracefully if tree-sitter is unavailable |
| **Single-process scalability** | One goja VM per session is lightweight (~2MB); 100 sessions = ~200MB |

### 12.2 Alternatives considered

| Alternative | Why rejected (for now) |
|---|---|
| **WebSocket-based streaming** | Adds complexity; HTTP request/response is simpler for agent integration |
| **Embedded V8 instead of goja** | goja is pure Go, no CGo (except optional tree-sitter), simpler deployment |
| **Separate process per session** | Higher overhead, harder state management, overkill for current scale |
| **gRPC instead of REST** | REST is more accessible to LLM agents and curl/scripts |

### 12.3 Open questions

1. **Should sessions persist across server restarts?** (Probably not for MVP; sessions are ephemeral.)
2. **Should the rewrite support `import`/`export` syntax?** (goja does not support ES modules natively; would need a bundler step.)
3. **Should we support multiple evaluation modes?** (e.g., "script mode" without rewriting, "module mode" with imports.)
4. **What authentication model for production?** (API keys? OAuth? Session tokens?)
5. **Should the completion endpoint use the goja parser or tree-sitter for error-tolerant completion?** (tree-sitter is better for incomplete input.)

---

## 13. Glossary

| Term | Definition |
|------|-----------|
| **Cell** | One unit of code submitted for evaluation, analogous to a Jupyter notebook cell. |
| **Session** | A named, long-lived goja runtime instance with persistent bindings and history. |
| **Binding** | A named JavaScript value (variable, function, class) tracked by the session. |
| **IIFE** | Immediately Invoked Function Expression — `(function() { ... })()`. |
| **Rewrite** | The source-to-source transformation that wraps user code in an async IIFE. |
| **Global diff** | The comparison of non-builtin global properties before and after evaluation. |
| **Provenance** | Metadata documenting how each section of the response was obtained. |
| **Factory** | The `engine.Factory` that creates pre-configured goja runtimes. |
| **Runner/Owner** | The `runtimeowner.Runner` that serializes VM access to a single goroutine. |
| **CST** | Concrete Syntax Tree from tree-sitter, preserving whitespace and comments. |
| **AST** | Abstract Syntax Tree from the goja parser, used for semantic analysis. |

---

## 14. Key File References

| File | Lines | Role |
|------|-------|------|
| `cmd/web-repl/main.go` | 118 | CLI entry point, flag parsing, factory + service + handler wiring |
| `pkg/webrepl/types.go` | 299 | All JSON-serializable types (request, response, views) |
| `pkg/webrepl/server.go` | 125 | HTTP routing, embedded static assets, JSON helpers |
| `pkg/webrepl/service.go` | 880 | Session lifecycle, evaluation pipeline, binding bookkeeping |
| `pkg/webrepl/rewrite.go` | 430 | Source rewriting (async IIFE), static report building, CST flattening |
| `pkg/jsparse/analyze.go` | 93 | `Analyze()` entry point → parse + index + resolve |
| `pkg/jsparse/resolve.go` | 1001 | Full lexical scope resolver |
| `pkg/jsparse/index.go` | 462 | AST node indexing and lookup |
| `pkg/jsparse/treesitter.go` | 185 | tree-sitter parser wrapper |
| `pkg/inspector/analysis/session.go` | 39 | `Session` wrapper for analysis results |
| `pkg/inspector/analysis/xref.go` | 85 | Cross-reference lookup |
| `pkg/inspector/runtime/introspect.go` | 188 | Object inspection and prototype walking |
| `pkg/inspector/runtime/globals.go` | 55 | Builtin-global filtering |
| `pkg/inspector/runtime/function_map.go` | 201 | Runtime function → source mapping |
| `engine/factory.go` | 224 | Factory builder pattern |
| `engine/runtime.go` | 110 | Runtime struct with lifecycle management |
| `engine/module_specs.go` | 103 | ModuleSpec, RuntimeInitializer interfaces |
| `engine/runtime_modules.go` | 45 | RuntimeModuleRegistrar interface |
| `pkg/runtimeowner/runner.go` | ~80 | Serialized VM access via `Call()` |
| `pkg/hashiplugin/host/runtime_setup.go` | ~60 | Plugin discovery → factory builder wiring |
