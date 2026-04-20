---
title: Investigation Diary
ticket: GOJA-044-PR28-REPL-SERVICE-REVIEW
doc-type: reference
topics: goja, repl, code-review, diary
createdAt: 2026-04-20
---

# Investigation Diary — GOJA-044

## 2026-04-20 09:30 — Setup and Orientation

- Cloned PR #28 branch (`task/add-repl-service`) in go-go-goja
- PR is 49 commits, 239 changed files, +35,464/−1,169 lines
- Core new packages: `replapi`, `replsession`, `repldb`, `replhttp`, `replessay`
- Created docmgr ticket GOJA-044-PR28-REPL-SERVICE-REVIEW

## 2026-04-20 09:35 — Source Code Review

Read all core packages:
- `pkg/replsession/` — service.go, types.go, evaluate.go, rewrite.go, observe.go, persistence.go, policy.go
- `pkg/replapi/` — app.go, config.go
- `pkg/repldb/` — store.go, schema.go, read.go, write.go, types.go
- `pkg/replhttp/` — handler.go
- `cmd/goja-repl/` — root.go

All unit tests pass: `replsession`, `replapi`, `repldb`, `replhttp` all green.

## 2026-04-20 09:39 — HTTP API Experiments (exp01, exp02)

- Started server at localhost:3092
- exp01: Full lifecycle smoke test — all PASS (create, eval, snapshot, list, history, bindings)
- exp02: Edge case evaluation — discovered **BUG-1** (empty source crash)
  - `{"source":""}` and `{"source":"   "}` both crash the server with `index out of range [0]`
  - Stack trace: `jsparse.Resolve` → `program.Idx0()` → `Body[0]` on empty program

## 2026-04-20 09:40 — Server Restart in tmux

- Killed old server, restarted in tmux session `goja-repl` with debug logging
- Confirmed server crash logs visible in tmux

## 2026-04-20 09:41 — Extended HTTP Edge Cases (exp03)

Results:
- Delete + restore: Soft delete works, deleted sessions properly rejected ✅
- Concurrent evaluation (5 concurrent): All succeed, mutex serializes ✅
- Concurrent heavy (10 concurrent): All succeed ✅
- Large object (200 properties): Works, truncated at 20 ownProperties ✅
- Export API returns PascalCase keys (`Session`, `Evaluations`) → **BUG-3**
- Destructuring: Array and object both work ✅
- Template literals: Work ✅
- Arrow functions: Persist across cells ✅
- For loops (no final expression): Return `undefined` ✅
- Multiple const declarations: All captured ✅

## 2026-04-20 09:50 — Wrote Bug Report v1

- Documented BUG-1 through BUG-5 in `design/01-pr28-bug-report.md`
- Created experiment scripts in `scripts/`

## 2026-04-20 09:53 — Go Low-Level Test Suite (exp04)

Wrote `exp04_lowlevel.go` — 22 test cases exercising `replsession.Service` directly.

Results: **19 passed, 3 failed**

### T09 FAIL: Function binding source mapping (BUG-8)
- `FunctionMapping` is always `nil` for declared functions
- `MapFunctionToSource()` returns nil because function was compiled from IIFE wrapper, not original source

### T12 FAIL: Thrown errors lose message (BUG-6)
- `throw new Error('boom')` → error text is `"promise rejected: {}"` instead of containing `'boom'`
- Root cause: IIFE wrapper makes `throw` reject the promise, `gojaValuePreview()` on Error object returns `{}`

### T18 FAIL: Top-level await parse error (BUG-7)
- `await Promise.resolve(99)` → parse-error in instrumented mode
- goja parser rejects `await` before IIFE wrapper is applied
- Works in HTTP server because event loop module accepts top-level await

## 2026-04-20 09:55 — Investigated Root Causes

### BUG-6 Detail
The rewrite transforms `throw new Error('boom')` into:
```javascript
(async function () {
  let __ggg_repl_last_1__;
  throw new Error('boom')
  return { ... };
})()
```
The throw rejects the promise. `waitPromise()` catches `PromiseStateRejected` and formats
the rejection value with `gojaValuePreview()`, which serializes Error objects as `{}`.

### BUG-7 Detail
In instrumented mode, `jsparse.Analyze()` is called on the raw source. goja's parser
treats `await` at top level as a syntax error. The IIFE wrapper (which would make it valid)
hasn't been applied yet. Only raw mode's `wrapTopLevelAwaitExpression()` handles this.

### BUG-8 Detail
`MapFunctionToSource()` tries to match a runtime function's source position against the
analysis AST. But the function was compiled from the IIFE-wrapped source, not the original.
The line/column offsets don't match, so the mapping returns nil.

## 2026-04-20 09:58 — Updated Bug Report

Added BUG-6, BUG-7, BUG-8 to the bug report with full descriptions, root causes, and fixes.
Updated the summary table and recommendations.

### Final Tally
- 🔴 Critical: 1 (BUG-1: empty source crash)
- 🟡 Medium: 4 (BUG-2, BUG-3, BUG-6, BUG-7)
- 🟢 Low: 3 (BUG-4, BUG-5, BUG-8)

### Key Insight
The low-level Go test (`exp04_lowlevel.go`) caught 3 bugs that the HTTP tests missed:
- Error message loss (BUG-6) — HTTP test showed "runtime-error" but didn't check the error text
- Top-level await (BUG-7) — HTTP server has event loop that masks the bug
- Function mapping (BUG-8) — Only visible via the binding runtime inspection
