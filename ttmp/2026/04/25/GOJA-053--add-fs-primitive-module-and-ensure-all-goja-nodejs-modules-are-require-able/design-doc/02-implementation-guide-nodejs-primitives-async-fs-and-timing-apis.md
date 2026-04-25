---
Title: 'Implementation Guide: NodeJS Primitives, Async fs, and Timing APIs'
Ticket: GOJA-053
Status: active
Topics:
    - goja
    - modules
    - fs
    - nodejs-compat
    - goja-nodejs
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Task-by-task implementation guide for imported goja_nodejs primitives, promise-based fs APIs, and JavaScript timing primitives."
LastUpdated: 2026-04-25T09:05:00-04:00
WhatFor: "Use as the coding checklist and review guide while implementing GOJA-053."
WhenToUse: "When adding Node.js primitive imports/globals, rebuilding fs, or exposing timing/performance APIs."
---

# Implementation Guide: NodeJS Primitives, Async fs, and Timing APIs

## Executive Summary

GOJA-053 now has three implementation tracks:

1. **Imported goja_nodejs primitives**: make `buffer`, `url`, `util`, and `process` require-able; enable `Buffer`, `URL`, and `URLSearchParams` globals by default; keep the `process` global opt-in because `process.env` exposes host environment data.
2. **Promise-based fs module**: replace the legacy two-function `fs` module with an async-first module that exposes Promise-returning functions and synchronous convenience variants.
3. **Timing APIs**: expose timing primitives for JavaScript-side performance measurements, starting with `performance.now()` and `console.time()` / `console.timeEnd()` / `console.timeLog()`.

The implementation should proceed in small commits. Each commit must include focused tests. Smoke tests should execute real JavaScript through a real go-go-goja runtime, not just unit-test Go helper functions.

## Track A: Imported goja_nodejs Primitives

### Target behavior

```javascript
// Always available, no caller opt-in required:
Buffer.from("hello").toString();
new URL("https://example.com").hostname;
new URLSearchParams("a=1").get("a");
require("buffer");
require("url");
require("util");
require("process");

// Not available unless the caller opts in:
typeof process; // "undefined" by default
```

A caller that wants the global `process` object should opt in explicitly:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    WithRuntimeInitializers(engine.ProcessEnv()).
    Build()
```

### Files and symbols

- `engine/nodejs_init.go` — new blank-import file for goja_nodejs core module registration.
- `engine/factory.go` — import `buffer` and `url`, and call `buffer.Enable(vm)` and `url.Enable(vm)` after `console.Enable(vm)`.
- `engine/module_specs.go` — add `ProcessEnv()` runtime initializer.
- `engine/nodejs_primitives_test.go` — new smoke tests.

### Test plan

- Smoke test default runtime:
  - `Buffer.from("abc").toString()` returns `abc`.
  - `new URL("https://example.com/path").hostname` returns `example.com`.
  - `require("util").format("%s:%d", "x", 3)` returns `x:3`.
  - `require("process").env` is an object.
  - `typeof process` is `undefined` by default.
- Smoke test opt-in runtime:
  - With `engine.ProcessEnv()`, `typeof process` is `object`.
  - `process.env` is an object.

## Track B: Promise-Based fs Module

### Target behavior

```javascript
const fs = require("fs");

await fs.mkdir(tmp + "/dir", { recursive: true });
await fs.writeFile(tmp + "/dir/a.txt", "hello");
const text = await fs.readFile(tmp + "/dir/a.txt");
const entries = await fs.readdir(tmp + "/dir");
const stat = await fs.stat(tmp + "/dir/a.txt");
await fs.appendFile(tmp + "/dir/a.txt", " world");
await fs.copyFile(tmp + "/dir/a.txt", tmp + "/dir/b.txt");
await fs.rename(tmp + "/dir/b.txt", tmp + "/dir/c.txt");
await fs.unlink(tmp + "/dir/c.txt");

fs.writeFileSync(tmp + "/sync.txt", "sync");
fs.readFileSync(tmp + "/sync.txt");
```

### Files and symbols

- `modules/fs/fs.go` — module identity, docs, TypeScript declarations, `Loader()` wiring.
- `modules/fs/fs_async.go` — Promise-returning async helpers.
- `modules/fs/fs_sync.go` — synchronous helper functions.
- `modules/fs/fs_test.go` — smoke tests for async and sync functions.

### Concurrency invariants

- Background goroutines may perform blocking `os.*` calls.
- Background goroutines must not call `vm.ToValue`, `resolve`, `reject`, or any other goja VM operation directly.
- Promise settlement must happen through `bindings.Owner.Post(...)` on the owner thread.
- Async tests should use the `timer_test.go` pattern: set global state in JS, then poll via `require.Eventually` from Go.

## Track C: Timing APIs

### Target behavior

```javascript
const t0 = performance.now();
for (let i = 0; i < 1000; i++) {}
const dt = performance.now() - t0;

time.now();        // same shape as performance.now(), useful through require("time")
time.since(t0);    // milliseconds since timestamp

console.time("work");
for (let i = 0; i < 1000; i++) {}
console.timeLog("work");
console.timeEnd("work");
```

### Files and symbols

- `modules/time/time.go` — new go-go-goja module exposing `now()` and `since(startMs)`.
- `engine/factory.go` or a runtime initializer — install global `performance.now()` by default.
- `engine/performance.go` — recommended helper for installing `performance` without depending on module opt-in.
- `engine/console_time.go` or console wrapper code — add `console.time`, `console.timeLog`, `console.timeEnd` to the runtime console object after `console.Enable(vm)`.
- `modules/time/time_test.go` and/or `engine/performance_test.go` — smoke tests.

### Design decisions

- Use monotonic elapsed time from `time.Now()` and `time.Since(start)`; Go carries monotonic clock readings in `time.Time` values.
- `performance.now()` returns milliseconds as `float64`, matching Node/browser style.
- `time.now()` returns the same millisecond timestamp shape so scripts can `require("time")` when they prefer explicit imports.
- `console.time*` should write through the existing console object where possible and should not panic on unknown labels.

## Commit Plan

1. Commit A: imported goja_nodejs primitives + tests.
2. Commit B: timing primitives + tests.
3. Commit C: fs async/sync implementation + tests.
4. Commit D: docs/diary/changelog updates, if not included in each focused commit.

Each commit should run at least the relevant package tests before committing. Before final handoff, run `go test ./engine ./modules/fs ./modules/time -count=1` and a broader smoke test if time permits.
