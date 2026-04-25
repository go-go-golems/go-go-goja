---
Title: Changelog
Ticket: GOJA-053
LastUpdated: 2026-04-25T08:00:00-04:00
---

# Changelog

## 2026-04-25: Initial analysis and design

- Created ticket GOJA-053 with comprehensive design doc
- Completed full investigation of goja_nodejs module inventory
- Confirmed 4 missing module wirings (buffer, process, url, util)
- Designed enhanced fs module API (8 new functions)
- Created phased implementation plan (3 phases, 23 tasks)
- Created investigation diary with evidence and findings

## 2026-04-25 (update): Pivoted to promise-based fs design

- Discovered the runtime already has full async/promise support via timer module pattern
- Confirmed REPL evaluator handles `await` and Promise polling
- Updated design doc: async-first fs with sync wrappers
- New file structure: `fs.go` + `fs_async.go` + `fs_sync.go`
- Updated tasks to 20 items across 3 phases

### Related files

- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/modules/timer/timer.go`: Proven promise pattern
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go`: await/Promise handling
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/pkg/runtimeowner/runner.go`: Post() for safe owner-thread scheduling

## 2026-04-25: Implemented Track A/B/C

- Track A implemented imported goja_nodejs primitives and configurable `process` global (commit eb9401a1989289ffb286b0aa3d4a1f6821cf4474).
- Track B implemented JavaScript timing primitives (`performance.now`, `console.time*`, `require("time")`) (commit a0e5628cbc7fb7d299349640b5de04ce40369f1b).
- Track C implemented promise-based fs primitives plus sync wrappers (commit 79a36627d0e456da1a487d51d06fbf41ecbefb0f).
- Validation passed: `go test ./engine ./modules/fs ./modules/time -count=1`.
- Broader pre-commit hook surfaced an unrelated existing failure in `pkg/replsession`: `TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline` expected timeout status but got `runtime-error`.

### Related files

- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/engine/factory.go`: Runtime global installation for Buffer, URL, performance, and console timers.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/engine/module_specs.go`: `ProcessEnv()` opt-in runtime initializer.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/engine/nodejs_init.go`: goja_nodejs core module registration imports.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/engine/performance.go`: Performance and console timing globals.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/modules/time/time.go`: Explicit timing module.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/modules/fs/fs.go`: fs module wiring and declarations.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/modules/fs/fs_async.go`: Promise-based fs implementations.
- `/home/manuel/workspaces/2026-04-25/add-primitive-modules/go-go-goja/modules/fs/fs_sync.go`: Synchronous fs implementations.

## 2026-04-25: Added Buffer support to fs primitives

- Updated fs read APIs to return Buffer by default and string when encoding is provided.
- Updated fs write/append APIs to accept string, Buffer, TypedArray, and DataView inputs.
- Added async and sync Buffer smoke tests (commit ab179dd715d41f0f81adb33f199b9414145a5266).
- Validation passed: `go test ./modules/fs -count=1` and `go test ./engine ./modules/fs ./modules/time ./pkg/replsession -count=1`.

## 2026-04-25: Added path, os, crypto, and fs compatibility follow-ups

- Added `path` module with host filepath helpers.
- Added `os` module with common host OS helpers.
- Added `crypto` module with `randomUUID`, `randomBytes`, and basic `createHash` support.
- Improved fs errors to throw/reject Error objects with `code`, `path`, and `syscall`.
- Added fs options support and `rm/rmSync`.
- Added runtime smoke tests for all new modules and fs compatibility behavior (commit 0a0c49c1fea5c8d4828ed5e57fd6fbcd544dccad).
- Validation passed: `go test ./modules/path ./modules/fs ./modules/os ./modules/crypto ./engine -count=1`.

## 2026-04-25: Added granular default module selection

- Data-only modules (`crypto`, `path`, `time`, `timer`) are now registered automatically for every engine runtime.
- Host-access modules (`fs`, `os`, `exec`, `database`) require explicit selection through `engine.DefaultRegistryModule(name)` or `engine.DefaultRegistryModulesNamed(...)`.
- `engine.DefaultRegistryModules()` remains available for trusted runtimes that want the whole registry.
- Updated help docs for third-party embedding and module selection (commits a7a6c9716d6bab3bcb9dfe943c6dbe4493aab4e1 and 0b01fc0b7ca6072040b5e83f903b30409a80f737).

## 2026-04-25: Added jsverbs objectFromFile support

- `pkg/jsverbs/command.go` now maps `objectFromFile` and related Glazed field types into Glazed field definitions.
- Added an end-to-end test proving JSON loaded through `objectFromFile` arrives in JavaScript as an object.
- Added `scripts/validate-jsverbs-objectfromfile.sh` as a repeatable CLI smoke test using `cmd/jsverbs-example`.
- Updated the jsverbs reference help page with the expanded field type mapping.
