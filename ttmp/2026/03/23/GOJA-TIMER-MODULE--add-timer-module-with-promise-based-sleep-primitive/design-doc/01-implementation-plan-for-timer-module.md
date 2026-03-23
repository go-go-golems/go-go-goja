---
Title: Implementation plan for timer module
Ticket: GOJA-TIMER-MODULE
Status: active
Topics:
    - goja
    - javascript
    - modules
    - async
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime creation establishes the event loop used by timer promises
    - Path: engine/module_specs.go
      Note: DefaultRegistryModules wiring controls built-in module availability
    - Path: engine/runtime.go
      Note: Default blank imports determine whether timer ships in fresh runtimes
    - Path: pkg/doc/03-async-patterns.md
      Note: Async documentation already describes the timer pattern this ticket implements
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-23T14:56:29.634149479-04:00
WhatFor: Explain how to add and validate a shipped timer module for go-go-goja.
WhenToUse: Use when implementing or reviewing Promise-based async module patterns in go-go-goja.
---


# Implementation plan for timer module

## Executive Summary

`go-go-goja` already constructs a `goja_nodejs/eventloop.EventLoop` for every runtime, but a fresh runtime does not expose `setTimeout`, `setInterval`, or a built-in timer module. The repository documentation already teaches a timer pattern, so the code and docs are currently out of sync.

This ticket closes that gap by adding a real built-in `timer` module that exposes `sleep(ms) -> Promise<void>`. The implementation should follow the repository's preferred async pattern: create the Promise on the VM owner thread, do the waiting off-thread, and settle the Promise back onto the owner thread via the runtime event loop or `runtimeowner.Runner`.

## Problem Statement

Today, callers who want to pause in JavaScript have no shipped timing primitive available through `require(...)`. A newly created runtime only enables `require` and `console`, plus whichever native modules are explicitly registered. The default blank imports currently cover `database`, `exec`, and `fs`, but not `timer`.

This creates three problems:

- The documentation and README imply a built-in `timer` module exists, but the module package is absent.
- Downstream projects such as `scraper` cannot rely on a stable timing helper without building their own module.
- The absence of a shipped timer primitive tempts consumers to ask for `setTimeout` globals first, which is a broader surface area than is needed for the initial async use case.

## Proposed Solution

Add a new native module package at `modules/timer` that registers itself during `init()` like the existing built-in modules. The module will export a single function in milestone one:

- `sleep(ms: number): Promise<void>`

Behavioral contract:

- `sleep(ms)` creates a JavaScript Promise on the owner thread.
- The wait itself happens in a background goroutine using Go timers.
- Promise resolution or rejection is scheduled back onto the VM owner thread.
- Negative durations reject with a clear error instead of panicking.
- Zero duration is allowed and resolves asynchronously.

The runtime and docs should then be updated so `DefaultRegistryModules()` includes this module automatically via the normal blank-import path.

High-level flow:

```text
JS caller
  |
  | require("timer").sleep(25)
  v
timer module Loader
  |
  | create Promise on owner thread
  v
background goroutine
  |
  | wait 25ms using Go timer
  v
event loop / owner runner
  |
  | resolve Promise back on VM goroutine
  v
JS await continues
```

Pseudocode sketch:

```go
exports.Set("sleep", func(ms int64) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    runner := runtimeowner.NewRunner(vm, loop, ...)

    go func() {
        if ms < 0 {
            _ = runner.Post(ctx, "timer.sleep.reject", func(context.Context, *goja.Runtime) {
                _ = reject(vm.ToValue("timer.sleep: duration must be >= 0"))
            })
            return
        }

        time.Sleep(time.Duration(ms) * time.Millisecond)
        _ = runner.Post(ctx, "timer.sleep.resolve", func(context.Context, *goja.Runtime) {
            _ = resolve(goja.Undefined())
        })
    }()

    return vm.ToValue(promise)
})
```

## Design Decisions

### Module first, globals later

The first shipped timing primitive should be `require("timer").sleep(ms)`, not `setTimeout(...)`.

Rationale:

- It fits the existing native module registration model.
- It keeps the API intentionally small.
- It avoids introducing timer handles, callback semantics, and global namespace questions before they are needed.

### Promise-based API

The module should expose Promise-based behavior rather than callback-only helpers.

Rationale:

- It matches the existing async documentation.
- It is the most ergonomic surface for `await`.
- It keeps the API narrow and easy to test.

### Use the owner-thread settlement pattern already documented in the repo

The Promise must be settled on the VM owner thread via the runtime event loop or runner abstraction.

Rationale:

- Goja values and Promise resolution are not thread-safe across goroutines.
- The repository already documents this as the required pattern.
- Reusing the same pattern reduces the chance of subtle VM misuse.

### Keep the milestone-one API minimal

Milestone one only adds `sleep(ms)`. No `setTimeout`, `setInterval`, cancellation handles, or global mounting.

Rationale:

- It solves the concrete missing primitive with low risk.
- It avoids inventing a larger JavaScript timing compatibility surface.
- It gives downstream consumers a reliable building block immediately.

## Alternatives Considered

### Mount `setTimeout` and friends as globals immediately

Rejected for this ticket because it expands the API and semantic burden too early. A global timer API would need timer handles, callback scheduling rules, and clearer lifecycle semantics during runtime shutdown.

### Add no built-in timer support and let each downstream project provide its own module

Rejected because the repository documentation already teaches a generic async timer pattern, and downstreams should not need to duplicate such a basic primitive.

### Expose only synchronous sleep

Rejected because blocking the owner thread would be incorrect for a Goja runtime and would undermine the async execution model described in the docs.

## Implementation Plan

### Phase 1: Ticket setup and architecture capture

- Create the ticket workspace under `go-go-goja/ttmp`.
- Record the current runtime facts:
  - `Factory.NewRuntime` creates an event loop.
  - default shipped modules come from explicit blank imports plus `DefaultRegistryModules()`.
  - a fresh runtime does not currently expose timer globals.

### Phase 2: Timer module implementation

- Add `modules/timer/timer.go`.
- Implement `Name`, `Doc`, `Loader`, and `init()` registration.
- Expose `sleep(ms)` using Promise creation plus owner-thread settlement.

### Phase 3: Registration and coverage

- Blank-import the timer module from [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/runtime.go).
- Add tests under `modules/timer` and/or runtime integration tests proving `require("timer")` works.

### Phase 4: Documentation sync and validation

- Update the README and async docs so examples reflect a real shipped module.
- Run `gofmt` and `go test ./...`.
- Record commits and implementation notes in the diary.
- Run `docmgr doctor` for the ticket.

## Open Questions

The only likely open question is whether `sleep(ms)` should reject only negative inputs or also reject non-integer values explicitly. The initial implementation can accept numeric coercion through the Go binding and reject negative values. If future consumers need stricter JS-side validation, that can be added in a follow-up.

## References

- [factory.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/factory.go)
- [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/runtime.go)
- [module_specs.go](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/engine/module_specs.go)
- [03-async-patterns.md](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/pkg/doc/03-async-patterns.md)
- [README.md](/home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/README.md)
