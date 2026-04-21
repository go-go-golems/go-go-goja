---
title: PR #28 Review — REPL Service Architecture Bug Report
ticket: GOJA-044-PR28-REPL-SERVICE-REVIEW
doc-type: analysis
topics: goja, repl, code-review, architecture, testing
createdAt: 2026-04-20
---

# PR #28 Review — REPL Service Architecture Bug Report

**PR:** [go-go-golems/go-go-goja#28](https://github.com/go-go-golems/go-go-goja/pull/28)
**Branch:** `task/add-repl-service` → `main`
**Author:** wesen (Manuel Odendahl)
**Status:** OPEN
**Commits:** 49 | **Changed files:** 239 | **+35,464 / −1,169**

## Executive Summary

PR #28 introduces a comprehensive REPL service architecture with persistence, session management,
source rewriting, and a web-based session inspector ("Essay"). It replaces the old `cmd/js-repl`
and `cmd/repl` commands with a unified `goja-repl` CLI. The code is well-structured, thoroughly
tested at the unit level, and demonstrates strong architectural thinking.

**One critical crash bug was found** (empty/whitespace source causes server panic),
along with several medium-severity issues and a number of code quality observations.

---

## Bug Report

### BUG-1 (Critical): Empty/Whitespace Source Causes Server Panic

**Severity:** 🔴 Critical — Denial of Service, crashes the HTTP handler goroutine
**Location:** `pkg/replsession/evaluate.go:54` → `pkg/jsparse/resolve.go:123`
**Reproducible:** Always

**Description:**
Submitting an empty string `""` or whitespace-only string `"   "` as source to the evaluate
endpoint causes an index-out-of-range panic in `jsparse.Resolve`. The goja AST parser returns
a `Program` with an empty `Body` slice. When `Resolve()` calls `program.Idx0()`, goja's
implementation accesses `Body[0]`, panicking.

**Stack trace:**
```
runtime error: index out of range [0] with length 0
github.com/dop251/goja/ast.(*Program).Idx0(...)
    ast/node.go:696
github.com/go-go-golems/go-go-goja/pkg/jsparse.Resolve(...)
    pkg/jsparse/resolve.go:123
github.com/go-go-golems/go-go-goja/pkg/jsparse.Analyze(...)
    pkg/jsparse/analyze.go:45
github.com/go-go-golems/go-go-goja/pkg/replsession.(*Service).Evaluate(...)
    pkg/replsession/evaluate.go:54
```

**Root cause:**
In `evaluate.go:54`, `jsparse.Analyze()` is called unconditionally when `shouldAnalyze(policy)`
returns true (which is always true for instrumented sessions). The analysis calls `Resolve()`
which assumes `program.Body` is non-empty. There is no nil/empty guard.

**Fix:** Either:
1. Skip analysis and return early for empty/whitespace source in `Evaluate()` (preferred)
2. Add an empty-body guard in `jsparse.Resolve()` before accessing `program.Idx0()`

**Reproduction:**
```bash
SID=$(curl -sf http://localhost:3092/api/sessions -X POST | jq -r '.session.id')
curl http://localhost:3092/api/sessions/$SID/evaluate \
  -X POST -H 'Content-Type: application/json' -d '{"source":""}'
# Server panics, returns empty reply (curl error 52)
```

**Experiment:** `scripts/exp02-edge-case-evaluation.sh` (triggers crash at empty source test)

---

### BUG-2 (Medium): No Panic Recovery in HTTP Handler

**Severity:** 🟡 Medium — Any unhandled panic in the evaluation pipeline kills the connection
without a meaningful error response
**Location:** `pkg/replhttp/handler.go`

**Description:**
The HTTP handler does not wrap evaluation calls in a `defer` + `recover()`. The empty-source
panic (BUG-1) causes the goroutine to crash with Go's default `net/http` panic handler, which
logs the stack trace and closes the connection. The client receives an empty response with no
status code or error message.

**Fix:** Add a top-level `defer func() { if r := recover(); r != nil { ... } }()` in the
evaluate handler, or wrap all handler functions with a recovery middleware.

---

### BUG-3 (Medium): Export API Uses PascalCase While All Other Endpoints Use camelCase

**Severity:** 🟡 Medium — API inconsistency
**Location:** `pkg/replhttp/handler.go:116` / `pkg/repldb/types.go`

**Description:**
The `/api/sessions/{id}/export` endpoint returns JSON with PascalCase field names (`Session`,
`Evaluations`, `SessionID`, `CreatedAt`) because it serializes `repldb.SessionExport` directly.
All other endpoints (`/api/sessions`, `/evaluate`, `/snapshot`) use camelCase (`id`, `createdAt`,
`cellCount`, `bindingCount`) via the `replsession` types that have explicit `json:"..."` tags.

The `repldb.SessionRecord`, `EvaluationRecord`, and related types in `pkg/repldb/types.go`
do **not** have `json` struct tags at all, so they serialize using Go's default PascalCase.

**Impact:** Any JavaScript/TypeScript client consuming the API must handle inconsistent
casing, or the export endpoint must be treated specially.

**Fix:** Add `json:"..."` tags to all `repldb` types, matching the camelCase convention used
by `replsession` types.

---

### BUG-4 (Low): `console` Events Return `null` Instead of `[]` When Empty

**Severity:** 🟢 Low — JSON serialization inconsistency
**Location:** `pkg/replsession/evaluate.go` (ExecutionReport)

**Description:**
When no console events are captured, `cell.execution.console` is `null` in the JSON response
rather than `[]`. This is because `state.consoleSink` is set to `nil` before execution, and
when no console calls are made, `append([]ConsoleEvent(nil), state.consoleSink...)` returns `nil`.

Go's `json.Marshal` serializes nil slices as `null`, while empty slices serialize as `[]`.

**Impact:** Downstream JavaScript code that checks `cell.execution.console.length` will throw
`TypeError: Cannot read properties of null` instead of getting `0`.

**Fix:** Initialize to `[]ConsoleEvent{}` instead of `nil`.

---

### BUG-5 (Low): Binding `sessionBound` Is Always `false` for Newly Declared Bindings

**Severity:** 🟢 Low — Misleading metadata
**Location:** `pkg/replsession/observe.go` (diffGlobals)

**Description:**
In `evaluateInstrumented()`, when a new binding is declared via `const x = 1`, the global
diff correctly reports it as "added". However, the `sessionBound` field on the diff is `false`
because at the time `diffGlobals()` runs, the binding has not yet been added to `state.bindings`.
The binding is added later in the `upsertDeclaredBinding` call.

This means the first evaluation always shows `sessionBound: false` for declared bindings,
which is misleading — the binding *is* session-bound, it just hasn't been registered yet.

**Fix:** Move binding registration before diff computation, or post-process the diffs to
correct `sessionBound`.

---

## Architecture Review

### Strengths

1. **Clean layered architecture.** The separation into `replapi` (facade), `replsession` (kernel),
   `repldb` (persistence), and `replhttp` (transport) is well-executed. Each layer has clear
   responsibilities and the dependency graph is acyclic.

2. **Policy-driven behavior.** The `SessionPolicy` / `EvalPolicy` / `ObservePolicy` /
   `PersistPolicy` hierarchy provides fine-grained control over session behavior without
   combinatorial explosion. The three profiles (raw, interactive, persistent) give sensible
   defaults.

3. **Instrumented rewrite pipeline.** The async-IIFE wrapper with binding capture is a clever
   technique for making `const`/`let`/`class` declarations persistent across cells without
   polluting the global scope. The rewrite report gives full transparency about what happened.

4. **Comprehensive persistence.** The SQLite schema stores evaluations, console events, binding
   versions (with export snapshots and doc digests), and binding docs. The replay-via-source
   approach for restore is simple and correct.

5. **Good test coverage.** The unit tests cover persistence, policy behavior, timeout recovery,
   and the HTTP lifecycle. Edge cases like "session usable after timeout" are explicitly tested.

6. **Timeout and interrupt handling.** The `runString` method uses a goroutine to send
   `vm.Interrupt()` when context expires, and properly cleans up with `ClearInterrupt()`.
   The post-timeout usability test confirms this works correctly.

### Concerns

1. **No graceful handling of malformed JSON in HTTP handler.** The evaluate endpoint decodes
   JSON but only checks for generic decode errors. A request body like `{"source": 123}` would
   pass decoding but evaluate the string `"123"` — which is valid JS but may not be what was
   intended. There's no schema validation.

2. **Session map grows unboundedly in memory.** `Service.sessions` is only cleaned up on
   explicit `DeleteSession()`. A long-running server with many sessions will accumulate
   `sessionState` objects (each with their own goja runtime) in memory. There's no TTL,
   no eviction, no max-sessions limit.

3. **`RestoreSession` creates a temporary service per restore.** This allocates a fresh factory,
   runtime, and service for each replay, which is correct but expensive. If restore is called
   frequently, this could be a performance concern.

4. **`Owner.Call` pattern adds complexity.** The `state.runtime.Owner.Call(ctx, name, fn)` pattern
   is used extensively to execute Go callbacks inside the goja VM lock. This is necessary for
   correctness but makes the code harder to follow. The callback closures capture state that
   may be surprising (e.g., `state.consoleSink` mutation inside a VM callback).

5. **SQLite connection is shared across all goroutines.** The `repldb.Store` uses a single
   `*sql.DB` (which is safe via `database/sql` connection pool). However, WAL mode with
   `_busy_timeout=5000` means long-running transactions could still cause contention under
   heavy load.

6. **No authentication or authorization.** The HTTP server exposes a raw JavaScript execution
   endpoint without any auth. Anyone with network access can create sessions and execute
   arbitrary JavaScript. This should be documented as explicitly unsafe for public deployment.

7. **`WithPersistence` is applied twice in `NewWithConfig`.** In `replapi/config.go`, the
   `NewWithConfig` function appends `WithPersistence(store)` and then also appends
   `WithDefaultSessionOptions(config.SessionOptions)` again. The second `WithDefaultSessionOptions`
   overrides the first one set by `WithPersistence` (which internally sets
   `PersistentSessionOptions()`). This works because the config already has the persistent
   session options, but it's confusing.

### Missing Test Coverage

1. **Empty/whitespace source** — Not tested. BUG-1 proves this crashes.
2. **Restore from persisted data after server restart** — No integration test that writes to
   SQLite, closes everything, reopens, and verifies restore.
3. **Concurrent access to different sessions** — Tests only exercise single-session concurrency.
4. **Binding version deduplication** — The `dedupeSortedStrings` function is tested indirectly
   but not in isolation.
5. **Error responses for invalid routes** — No tests for 404, method-not-allowed, etc.
6. **Large source submission** — No test for source that exceeds reasonable size limits.
7. **`WithRuntime` callback** — Not tested in the HTTP or API layer.

---

## Experiment Results

### EXP-01: API Smoke Test ✅
- Create session, evaluate `const x = 1; x` → ok, result=1
- Evaluate `x + 1` → ok, result=2 (binding persists across cells)
- Snapshot, list, history, bindings, export all work

### EXP-02: Edge Case Evaluation
- **const re-declaration**: Works (cell-local scoping via IIFE) ✅
- **let re-assignment**: Works ✅
- **function declarations**: Persist across cells ✅
- **class declarations**: Persist across cells ✅
- **syntax error recovery**: Session remains usable after parse error ✅
- **empty/whitespace source**: **CRASH** (BUG-1) ❌

### EXP-03: Extended Edge Cases
- **Delete + restore**: Soft delete works, deleted sessions can't be restored or snapshotted ✅
- **Concurrent evaluation**: 5 concurrent requests all succeed, serialized by mutex ✅
- **Large object literal**: 200-property object works, binding properties truncated at 20 ✅
- **Destructuring**: Array and object destructuring work, bindings captured ✅
- **No-final-expression cells**: Return `undefined`, all bindings still captured ✅
- **Template literals**: Work correctly ✅
- **Arrow functions**: Persist across cells, function kind tracked ✅
- **Heavy concurrent (10 requests)**: All succeed ✅

---

## Summary of Findings

| ID | Severity | Title | Status |
|----|----------|-------|--------|
| BUG-1 | 🔴 Critical | Empty source causes server panic | Reproduced, root cause identified |
| BUG-2 | 🟡 Medium | No panic recovery in HTTP handler | Confirmed |
| BUG-3 | 🟡 Medium | Export API PascalCase inconsistency | Confirmed |
| BUG-4 | 🟢 Low | Console events null vs empty array | Confirmed |
| BUG-5 | 🟢 Low | sessionBound always false for new bindings | Confirmed |

---

### BUG-6 (Medium): Thrown Errors Lose Message in Instrumented Mode

**Severity:** 🟡 Medium — Incorrect error reporting
**Location:** `pkg/replsession/evaluate.go` (executeWrapped → waitPromise)
**Reproducible:** Always in instrumented mode

**Description:**
When `throw new Error('boom')` is evaluated in instrumented mode, the async IIFE wrapper
causes the throw to reject the returned promise. The `waitPromise()` method catches the
rejection and formats it as `"promise rejected: {}"` — the error message 'boom' is lost
because `gojaValuePreview()` on the Error object returns `{}`.

**Expected:** `Error: boom` or at least `boom` in the error text.
**Actual:** `promise rejected: {}`

**Root cause:** The rejection value is a goja Error object. `gojaValuePreview()` calls
`inspectorruntime.ValuePreview()` which likely uses `.toString()` or JSON serialization
on the Error object, neither of which includes the message by default.

**Fix:** In `waitPromise()`, when the promise is rejected, extract the `.message` property
from the rejection value if it's an Error object. Or use `goja.Exception` error handling
in the IIFE execution path instead of the promise polling loop.

**Experiment:** `scripts/exp04_lowlevel.go` → T12

---

### BUG-7 (Medium): Top-Level Await Not Supported in Instrumented Mode

**Severity:** 🟡 Medium — Feature parity gap
**Location:** `pkg/replsession/evaluate.go:54` / `pkg/jsparse/analyze.go:45`
**Reproducible:** Always in instrumented mode without event loop

**Description:**
When `supportTopLevelAwait: true` is set and the session uses instrumented mode (default
for interactive/persistent profiles), evaluating `await Promise.resolve(99)` returns
`parse-error` instead of executing the await. The goja parser rejects `await` as a
syntax error outside async functions, and the parse happens *before* the IIFE wrapper
is applied.

In raw mode, `wrapTopLevelAwaitExpression()` pre-wraps `await ...` expressions in an
async IIFE, which works. But in instrumented mode, `buildRewrite()` uses `jsparse.Analyze()`
first, which triggers goja's parser, which fails on `await`.

Note: This *works* when the engine factory includes a NodeJS-compatible event loop module
(which the HTTP server does), but fails with a bare `engine.NewBuilder().Build()` factory.
This means behavior depends on which modules are loaded, not just on the policy.

**Fix:** In instrumented mode, pre-wrap `await` expressions before parsing, similar to
`wrapTopLevelAwaitExpression()`. Or, use a parser that accepts top-level await.

**Experiment:** `scripts/exp04_lowlevel.go` → T18

---

### BUG-8 (Low): Function Source Mapping Is Always Nil

**Severity:** 🟢 Low — Feature not working
**Location:** `pkg/replsession/observe.go` (refreshBindingRuntimeDetails)
**Reproducible:** Always

**Description:**
The `FunctionMapping` field on binding runtime views is always `nil`, even for functions
declared in previous cells. `MapFunctionToSource()` returns `nil` because the function
was created inside the IIFE wrapper (transformed source), not the original cell source.
The analysis result (`cell.analysis`) contains the AST for the original source, but the
runtime function was compiled from the rewritten source, so the line/column mapping
doesn't match.

**Fix:** Either map from the rewritten source's AST, or adjust the mapping to account
for the IIFE wrapper offset. Alternatively, use the static analysis binding information
(which correctly identifies the function) as the canonical source mapping.

**Experiment:** `scripts/exp04_lowlevel.go` → T09

---

## Updated Summary

| ID | Severity | Title | Source |
|----|----------|-------|--------|
| BUG-1 | 🔴 Critical | Empty source causes server panic | HTTP + Go tests |
| BUG-2 | 🟡 Medium | No panic recovery in HTTP handler | Manual |
| BUG-3 | 🟡 Medium | Export API PascalCase inconsistency | HTTP tests |
| BUG-4 | 🟢 Low | Console events null vs empty array | HTTP tests |
| BUG-5 | 🟢 Low | sessionBound always false for new bindings | Code review |
| BUG-6 | 🟡 Medium | Thrown errors lose message in instrumented mode | Go low-level test |
| BUG-7 | 🟡 Medium | Top-level await not supported in instrumented mode | Go low-level test |
| BUG-8 | 🟢 Low | Function source mapping always nil | Go low-level test |

**Recommendation:** Fix BUG-1 and BUG-6 before merge. BUG-2, BUG-3, BUG-7, and BUG-8
can be addressed in follow-up PRs. BUG-4 and BUG-5 are cosmetic.

The architecture is sound and the unit test coverage is strong. The low-level Go test
suite (`scripts/exp04_lowlevel.go`) provides 22 test cases that exercise the `replsession`
service directly and can serve as a foundation for regression testing.
