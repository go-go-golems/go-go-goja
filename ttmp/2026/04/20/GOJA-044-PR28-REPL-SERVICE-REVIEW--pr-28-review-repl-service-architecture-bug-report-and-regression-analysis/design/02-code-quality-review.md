---
title: PR #28 Code Quality Review
ticket: GOJA-044-PR28-REPL-SERVICE-REVIEW
doc-type: analysis
topics: goja, repl, code-quality, architecture, naming
createdAt: 2026-04-20
---

# PR #28 Code Quality Review

## Overview

This document covers structural quality issues in the new REPL service packages
introduced by PR #28: naming confusion, misplaced concerns, dead fields,
unnecessary complexity, legacy adapters, and files that should be split up.

The bug report (design/01-pr28-bug-report.md) covers correctness issues;
this document covers **readability, maintainability, and architectural hygiene**.

---

## 1. `replsession` Is a 3200-Line Monolith with Blurred Internal Boundaries

**Severity:** 🟡 Structural
**Files:** `pkg/replsession/*.go` (7 files, 3223 lines, 52 exported symbols)

### Problem

`replsession` is the only package in the new architecture that contains multiple
unrelated concerns in the same package:

| File | Lines | Actual Concern | Belongs In |
|------|-------|----------------|------------|
| `types.go` | 301 | 32 JSON view structs for the web UI | `replsession/types/` or `replsession/views/` |
| `rewrite.go` | 440 | IIFE rewrite (5 funcs) + static analysis report building (10 funcs) | Split `rewrite.go` and `static_report.go` |
| `observe.go` | 434 | Global snapshots, binding runtime inspection, diffing, summary building | `replsession/observe/` |
| `persistence.go` | 299 | SQLite persistence, binding export snapshots, JSDoc extraction | `replsession/persist/` |
| `evaluate.go` | 545 | Evaluation pipeline, raw + instrumented paths, promise polling, timeout | Core — stays |
| `service.go` | 493 | Session lifecycle, option resolution, console capture | Core — stays |
| `policy.go` | 187 | Policy types, normalization, profiles | `replsession/policy/` |

The package has **52 exported symbols** (types + functions). Of those, **32 are View structs**
in `types.go` that exist purely for JSON serialization to the web UI. They are not used
by any other package's Go code — only serialized to JSON. These inflate the API surface
and make godoc harder to navigate.

### Recommendation

- Extract the 32 `*View` types into a sub-package `replsession/views` or keep them in
  `types.go` but acknowledge it's a serialization layer.
- Split `rewrite.go` into `rewrite.go` (IIFE builder) and `static_report.go` (AST/CST/scope
  flattening). The current file has 10 static-analysis functions that have nothing to do
  with source rewriting.
- Consider `observe.go` as a candidate for its own package if it continues to grow —
  it already has 20 functions covering global snapshots, diffing, prototype chain walking,
  and binding detail refresh.

---

## 2. `rewrite.go` Has Two Unrelated Jobs

**Severity:** 🟡 Structural
**File:** `pkg/replsession/rewrite.go` (440 lines)

### Problem

`rewrite.go` contains exactly two groups of functions that never call each other:

**Group A — Source rewriting** (belongs here):
- `buildRewrite()` — constructs the async IIFE wrapper
- `declaredNamesFromResult()` — extracts root-scope binding names
- `finalExpressionStatement()` — finds the trailing expression for capture
- `sourceSlice()`, `replaceSourceRange()` — source text utilities

**Group B — Static analysis report building** (does not belong here):
- `buildStaticReport()` — the main entry point for `StaticReport` construction
- `buildScopeView()` — recursive scope tree → JSON
- `expandAllNodes()` — AST node expansion
- `inspectorRows()`, `buildRows()` — AST → flat row list
- `flattenCST()` — tree-sitter CST → flat row list
- `trimForDisplay()` — string truncation
- `declarationSnippet()`, `bindingReferences()` — analysis → view extraction
- `rangeFromNode()` — AST node → RangeView

Group B is called from `evaluate.go:67` and from `observe.go`. It has zero dependency
on the rewrite logic. The only reason it's in `rewrite.go` is that it was written at
the same time.

### Recommendation

Move Group B to a new file `pkg/replsession/static_report.go`. No logic changes needed.

---

## 3. Confusing Dual `SessionOptions` Types

**Severity:** 🟡 Naming/API Surface
**Files:** `pkg/replapi/config.go`, `pkg/replsession/policy.go`

### Problem

There are two types both named `SessionOptions` in two different packages:

```go
// pkg/replsession/policy.go
type SessionOptions struct {
    ID        string        `json:"id,omitempty"`
    CreatedAt time.Time     `json:"createdAt,omitempty"`
    Profile   string        `json:"profile,omitempty"`   // string
    Policy    SessionPolicy `json:"policy"`              // value type
}

// pkg/replapi/config.go
type SessionOptions struct {
    ID        string
    CreatedAt time.Time
    Profile   *Profile          // pointer to string-like enum
    Policy    *replsession.SessionPolicy  // pointer
}
```

Callers must write `replapi.SessionOptions` vs `replsession.SessionOptions` and
understand that they differ in:
- `Profile` is `string` vs `*replapi.Profile` (pointer to alias)
- `Policy` is value vs pointer
- `replapi.SessionOptions` uses pointer fields to mean "override this"
- `replsession.SessionOptions` uses concrete fields with JSON tags

The conversion happens in `replapi/config.go:resolveCreateSessionOptions()` which
has to carefully merge nil-pointer checks with Normalize calls.

### Recommendation

Rename `replapi.SessionOptions` to `CreateSessionOverrides` or `SessionOverrides`
to make the distinction clear. The current name suggests it's the same type as
`replsession.SessionOptions`, which it isn't.

---

## 4. Policy Normalized 4+ Times Per CreateSession Call

**Severity:** 🟡 Complexity / Performance
**Files:** `pkg/replapi/config.go`, `pkg/replsession/service.go`, `pkg/replsession/policy.go`

### Problem

`NormalizeSessionPolicy()` is called **at least 4 times** during a single `CreateSession`
call through `replapi`:

1. `normalizeConfig()` → `NormalizeSessionOptions()` → `NormalizeSessionPolicy()`
2. `WithDefaultSessionOptions()` → `NormalizeSessionOptions()` → `NormalizeSessionPolicy()`
3. `resolveCreateSessionOptions()` → `NormalizeSessionOptions()` → `NormalizeSessionPolicy()`
4. `service.resolveSessionOptions()` → `NormalizeSessionOptions()` → `NormalizeSessionPolicy()`

The function is idempotent (it only sets defaults for empty/missing fields), so the
repeated calls aren't a correctness bug. But they suggest the normalization boundary
is unclear: is the caller or the callee responsible for normalizing?

Additionally, `WithDefaultSessionOptions` is applied **twice** in `NewWithConfig`:

```go
serviceOpts := []replsession.Option{
    replsession.WithDefaultSessionOptions(config.SessionOptions),  // first
}
if config.Store != nil {
    serviceOpts = append(serviceOpts, replsession.WithPersistence(config.Store))
    serviceOpts = append(serviceOpts, replsession.WithDefaultSessionOptions(config.SessionOptions))  // second, identical
}
```

The second call overwrites the first one's effect (both set `service.defaultSessionOpts`),
so it's harmless but confusing.

### Recommendation

- Pick **one** normalization boundary: either normalize at construction (in `NewWithConfig`)
  and pass already-normalized options down, or normalize at the leaf (in `resolveSessionOptions`).
  Not both.
- Remove the duplicate `WithDefaultSessionOptions` call.
- Add a comment or assertion that `NormalizeSessionOptions` is idempotent.

---

## 5. Provenance Boilerplate Is Hardcoded String Repetition

**Severity:** 🟢 Noise / Maintainability
**Files:** `pkg/replsession/observe.go`, `pkg/replsession/evaluate.go`

### Problem

`ProvenanceRecord` arrays are hardcoded inline in 6 places across 2 files:

```go
Provenance: []ProvenanceRecord{
    {Section: "session.bindings", Source: "aggregated persistent bindings stored across cells"},
    {Section: "session.history", Source: "evaluation reports recorded after each submitted cell"},
    {Section: "session.globals", Source: "current non-builtin goja global object snapshot"},
},
```

This block appears identically in `buildSummaryLockedWithGlobals()` and would need
to be updated in multiple places if the descriptions change. The strings are not
validated, indexed, or used for anything except display.

### Recommendation

Extract provenance templates into package-level constants or a `provenanceForSummary()`
helper. Alternatively, if provenance is only for debugging/display, consider making it
optional behind an `includeProvenance` policy flag to reduce response sizes.

---

## 6. Dead `PersistPolicy` Fields

**Severity:** 🟡 Dead Code
**File:** `pkg/replsession/policy.go`

### Problem

`PersistPolicy` has 5 fields:

```go
type PersistPolicy struct {
    Enabled         bool `json:"enabled"`
    Sessions        bool `json:"sessions"`
    Evaluations     bool `json:"evaluations"`
    BindingVersions bool `json:"bindingVersions"`
    BindingDocs     bool `json:"bindingDocs"`
}
```

But only 3 are actually checked in non-test code:
- `Enabled` — checked in `service.go`, `app.go`
- `BindingVersions` — checked in `persistence.go:91`
- `BindingDocs` — checked in `persistence.go:84`

`Sessions` and `Evaluations` are **never read individually**. They're set to `true`
in `PersistentSessionOptions()` and persisted in JSON, but no code path checks them.
When `Persist.Enabled` is true, sessions and evaluations are *always* persisted
regardless of these fields.

### Recommendation

Either:
- Remove `Sessions` and `Evaluations` fields (they suggest granularity that doesn't exist)
- Or implement the granularity they promise (skip session/evaluation persistence when false)
- Or add a comment explaining they're reserved for future use

---

## 7. Legacy Evaluator Still Alive with Duplicated Logic

**Severity:** 🟡 Technical Debt
**Files:** `pkg/repl/evaluators/javascript/evaluator.go` (764 lines still remain)

### Problem

PR #28 introduces the new `replsession` evaluation pipeline but **does not remove** the
old `pkg/repl/evaluators/javascript/evaluator.go`. Instead, it:

1. Reduced the old evaluator from ~1380 to 764 lines (removed some code)
2. Extracted `Assistance` (676 lines) into a shared struct that both old and new use
3. Created `pkg/repl/adapters/bobatea/replapi.go` as a bridge: it uses the new `replapi.App`
   for evaluation but keeps the old `Assistance` for completion/help

Three functions are duplicated between old and new:
- `wrapTopLevelAwaitExpression()` — identical logic, different parameter names
- Promise polling loop — structurally identical (`waitForPromise` vs `waitPromise`)
- `promisePreview` / `promiseString` — same idea, slightly different output

The old evaluator is still imported by:
- `cmd/smalltalk-inspector/app/` (2 files)
- `pkg/repl/adapters/bobatea/` (3 files)
- `cmd/goja-repl/tui.go` (via bobatea adapter)

### Recommendation

This is a transitional state, not a bug. But it should be tracked:
- Add a `// DEPRECATED` comment on the old `Evaluator` type
- Create a follow-up ticket to migrate `smalltalk-inspector` and the TUI to use
  `replapi.App` directly, then remove the old evaluator
- Extract the shared promise-polling and await-wrapping logic into a common utility

---

## 8. `cmd/goja-repl/root.go` Is a 568-Line Command Dump

**Severity:** 🟢 File Organization
**File:** `cmd/goja-repl/root.go`

### Problem

`root.go` defines **10 command structs** and their `Run` methods in a single file:

`sessions`, `create`, `eval`, `snapshot`, `history`, `bindings`, `docs`, `export`, `restore`, `serve`

Every command follows the same pattern:
1. Call `c.newApp()` or `c.newAppWithOptions()` (boilerplate app setup)
2. `defer store.Close()`
3. Call one `app.*` method
4. `writeJSON(c.out, ...)`

The `commandSupport` / `appSupportOptions` / `newAppWithOptions` pattern is repeated
in every `Run` method. The only variation is which `app.*` method gets called.

### Recommendation

- Split into `cmd/goja-repl/cmd_sessions.go`, `cmd/goja-repl/cmd_eval.go`, etc.
  (one file per command, or group related commands)
- Extract the `newApp → defer Close → call method → writeJSON` pattern into a
  generic `runWithApp(fn)` helper to eliminate the 10× boilerplate

---

## 9. HTTP Routing via Manual String Splitting

**Severity:** 🟡 Fragility
**File:** `pkg/replhttp/handler.go`

### Problem

The HTTP handler does all routing via manual `strings.Split` and `strings.TrimPrefix`:

```go
mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
    trimmed := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
    parts := strings.Split(strings.Trim(trimmed, "/"), "/")
    if len(parts) == 0 || parts[0] == "" {
        // ...
    }
    sessionID := parts[0]
    if len(parts) == 1 {
        // CRUD on session
    }
    if len(parts) != 2 {
        // 404
    }
    switch parts[1] {
    case "evaluate": ...
    case "restore": ...
    case "history": ...
    case "bindings": ...
    case "docs": ...
    case "export": ...
    }
})
```

This is fragile:
- Trailing slashes produce different `parts` lengths
- No method+path validation until deep in the switch
- Adding a new sub-resource requires modifying the manual length checks
- No path parameter extraction (sessionID is just `parts[0]`)

### Recommendation

Use `http.ServeMux` with Go 1.22+ method-based routing patterns:

```go
mux.HandleFunc("GET /api/sessions/{id}", handleGetSession)
mux.HandleFunc("POST /api/sessions/{id}/evaluate", handleEvaluate)
```

Or use a lightweight router like `chi` (already in the go-go-golems ecosystem).

---

## 10. `types.go` Has 32 Structs with Only JSON Tags

**Severity:** 🟢 Documentation / Organization
**File:** `pkg/replsession/types.go` (301 lines)

### Problem

`types.go` contains **32 struct definitions** with zero logic — they are pure
serialization DTOs. Every field has a `json:"..."` tag. No methods. No validation.

The file has no package-level comment explaining its role as the serialization contract
with the web UI. A reader must scan all 32 structs to understand which are request types,
which are response types, and which are nested sub-structures.

### Recommendation

- Add a file-level doc comment explaining this is the JSON serialization layer
- Group the types with section comments:
  ```go
  // --- Request types ---
  type EvaluateRequest ...
  
  // --- Response types ---
  type EvaluateResponse ...
  type SessionSummary ...
  
  // --- Cell report sub-structures ---
  type CellReport ...
  type StaticReport ...
  type RewriteReport ...
  type RuntimeReport ...
  
  // --- Binding detail views ---
  type BindingView ...
  ```
- Consider generating these types from a JSON schema or OpenAPI spec if they
  need to stay in sync with the web UI TypeScript types

---

## 11. Inconsistent Error Value Naming

**Severity:** 🟢 Naming
**Files:** `pkg/replsession/service.go`, `pkg/repldb/read.go`

### Problem

Two packages define `ErrSessionNotFound`:

```go
// pkg/replsession/service.go
var ErrSessionNotFound = errors.New("replsession: session not found")

// pkg/repldb/read.go
var ErrSessionNotFound = errors.New("repldb: session not found")
```

The HTTP handler maps both to 404:
```go
case errors.Is(err, replsession.ErrSessionNotFound), errors.Is(err, repldb.ErrSessionNotFound):
    return http.StatusNotFound
```

These represent the same conceptual error (session doesn't exist) but with different
wrapping contexts. A caller who checks `errors.Is(err, replsession.ErrSessionNotFound)`
will miss the `repldb` variant and vice versa.

### Recommendation

Define `ErrSessionNotFound` once (in `replsession` or a shared `replerrors` package)
and wrap it in `repldb`:

```go
// repldb
var ErrSessionNotFound = replsession.ErrSessionNotFound
```

Or use `errors.Wrap(replsession.ErrSessionNotFound, "...")` so `errors.Is` works
for either.

---

## 12. `replessay` Generates HTML Inline Without Templates

**Severity:** 🟢 Maintainability
**File:** `pkg/replessay/handler.go` (491 lines)

### Problem

The essay handler builds HTML responses by concatenating strings and embedding
large JSON blobs directly in the response body. There are no Go templates, no
HTML files, and no asset pipeline. The handler:

1. Serves a React SPA from an embedded FS (fine)
2. Generates "section" responses that are structured JSON payloads consumed by
   the React app — but the handler does this with inline Go code building maps
   and slices by hand

### Recommendation

This is acceptable for a first implementation but will become hard to maintain
as sections grow. Consider:
- Using Go templates for any HTML generation
- Using struct-based JSON serialization instead of `map[string]any` for section responses
- Documenting the section API contract (what fields each section returns)

---

## Summary

| # | Severity | Issue | Location |
|---|----------|-------|----------|
| CQ-1 | 🟡 Structural | `replsession` monolith with 52 exported symbols | `pkg/replsession/` |
| CQ-2 | 🟡 Structural | `rewrite.go` has two unrelated jobs | `pkg/replsession/rewrite.go` |
| CQ-3 | 🟡 Naming | Dual `SessionOptions` types with different semantics | `replapi` vs `replsession` |
| CQ-4 | 🟡 Complexity | Policy normalized 4+ times per CreateSession | `replapi/config.go`, `replsession/service.go` |
| CQ-5 | 🟢 Noise | Provenance boilerplate repeated in 6 places | `observe.go`, `evaluate.go` |
| CQ-6 | 🟡 Dead code | `PersistPolicy.Sessions`/`.Evaluations` never read | `pkg/replsession/policy.go` |
| CQ-7 | 🟡 Debt | Old evaluator still alive, duplicated logic | `pkg/repl/evaluators/javascript/` |
| CQ-8 | 🟢 Organization | 10 commands in one 568-line file | `cmd/goja-repl/root.go` |
| CQ-9 | 🟡 Fragility | HTTP routing via manual string splitting | `pkg/replhttp/handler.go` |
| CQ-10 | 🟢 Organization | 32 DTO structs with no grouping comments | `pkg/replsession/types.go` |
| CQ-11 | 🟢 Naming | Duplicate `ErrSessionNotFound` in two packages | `replsession` + `repldb` |
| CQ-12 | 🟢 Maintainability | Essay handler builds responses inline | `pkg/replessay/handler.go` |

### Priority Recommendations

**Do before merge:**
- CQ-2: Split `rewrite.go` (zero-risk file move)
- CQ-6: Remove or document dead `PersistPolicy` fields

**Do in follow-up:**
- CQ-1: Extract view types and observe/persist into sub-packages
- CQ-3: Rename `replapi.SessionOptions` to `SessionOverrides`
- CQ-4: Pick a single normalization boundary
- CQ-7: Plan old evaluator removal roadmap
- CQ-9: Migrate to Go 1.22 routing patterns

**Nice to have:**
- CQ-5, CQ-8, CQ-10, CQ-11, CQ-12
