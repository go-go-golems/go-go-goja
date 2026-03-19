---
Title: Registrar state and plugin diagnostics hardening implementation plan
Ticket: GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW
Status: active
Topics:
    - goja
    - analysis
    - architecture
    - tooling
    - refactor
    - repl
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime construction needs to retain registrar-produced values and a runtime-owned context.
    - Path: engine/runtime.go
      Note: Owned runtime lifecycle needs durable values and cancellation accessors.
    - Path: engine/runtime_modules.go
      Note: Setup-time registrar value flow begins here and should continue into the owned runtime.
    - Path: pkg/hashiplugin/host/client.go
      Note: Plugin startup currently discards diagnostics and validates manifests during load.
    - Path: pkg/hashiplugin/host/reify.go
      Note: Plugin invocation currently routes through context.Background instead of runtime-owned cancellation.
    - Path: pkg/hashiplugin/host/report.go
      Note: Load report summaries/details need stronger diagnostics semantics.
    - Path: pkg/hashiplugin/host/registrar.go
      Note: Plugin registrar already computes runtime-scoped state that should become durable.
    - Path: pkg/docaccess/runtime/registrar.go
      Note: Docs registrar is another producer of runtime-scoped values that will benefit from the same seam.
ExternalSources: []
Summary: Implementation plan for making registrar-produced state first-class on owned runtimes and for hardening plugin diagnostics and cancellation behavior without overcomplicating the engine or plugin host layers.
LastUpdated: 2026-03-18T17:14:08.390493099-04:00
WhatFor: Turn the GOJA-13 review findings on runtime state ownership and plugin operational behavior into a concrete implementation strategy that can drive follow-up cleanup work.
WhenToUse: Use when implementing runtime value persistence, runtime cancellation propagation, or stronger plugin discovery/load diagnostics.
---

# Registrar state and plugin diagnostics hardening implementation plan

## Executive Summary

The GOJA-13 review found one core mismatch in the new architecture: registrars are runtime-scoped setup hooks, but the values they produce do not become runtime-owned state. At the same time, the plugin subsystem already behaves like a first-class runtime feature but still hides too much operational detail and bypasses runtime-owned cancellation during invocation.

This document proposes a focused correction rather than a broad redesign. The immediate goal is to make owned runtimes retain setup-time registrar values and expose a runtime-owned context. Once that exists, plugin loading, docs integration, and future evaluator-side help features can use a single runtime-scoped seam rather than rebuilding state externally. In parallel, the plugin host should preserve enough diagnostics to explain failures and should route invocations through the runtime context instead of `context.Background()`.

The result is a more coherent runtime model:

- registrars produce values during setup
- the returned runtime owns those values
- plugin calls derive from runtime-owned cancellation
- diagnostics are captured where failures actually occur

## Problem Statement

### Setup-only registrar state breaks runtime coherence

[`engine/runtime_modules.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime_modules.go) already provides `RuntimeModuleContext.Values`, and registrars use it meaningfully. For example:

- [`pkg/hashiplugin/host/registrar.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/registrar.go) stores the loaded plugin snapshot
- [`pkg/docaccess/runtime/registrar.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/docaccess/runtime/registrar.go) builds a docs hub that later consumers will want to inspect

But [`engine/factory.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go) discards that state once setup is complete because [`engine.Runtime`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go) has no place to retain it.

That forces later features into awkward patterns:

- REPLs have to keep parallel wiring outside the runtime
- evaluator help/doc work cannot naturally consult runtime-owned docs state
- diagnostics and runtime inspection have no durable access to registrar outputs

### Diagnostics are weaker than the plugin feature surface

The plugin layer in [`pkg/hashiplugin/host`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host) launches subprocesses, validates manifests, loads modules into runtimes, and reports discovery/load outcomes. That is real operational complexity. However, the current loader behavior still discards plugin stderr and compresses some failures into summaries that are less specific than they should be.

Current issues:

- [`client.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/client.go) sends stdout/stderr to `io.Discard`
- discovery failures are not always carried forward with enough context
- [`report.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/report.go) is useful but not strongly error-first

If this remains unchanged, plugin support will technically work while still feeling brittle and opaque during failure cases.

### Runtime cancellation does not reach plugin calls

The runtime now has explicit ownership and cleanup, but plugin invocation in [`reify.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/reify.go) still invokes loaded modules with `context.Background()`. That undermines the runtime lifecycle model:

- runtime shutdown does not naturally cancel in-flight plugin calls
- future evaluator/request cancellation has no clean root to derive from
- plugins behave like external helpers rather than owned runtime resources

## Proposed Solution

### Part 1: Make registrar values runtime-owned

Add a durable value store to [`engine.Runtime`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go) and copy registrar-produced values from `RuntimeModuleContext.Values` during runtime creation.

Suggested shape:

```go
type Runtime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner

    Values map[string]any

    runtimeCtx       context.Context
    runtimeCtxCancel context.CancelFunc

    closeOnce sync.Once
    closerMu  sync.Mutex
    closers   []func(context.Context) error
    closing   bool
}

func (r *Runtime) Value(key string) (any, bool) { ... }
func (r *Runtime) Context() context.Context { ... }
```

This is intentionally small and specific. It is not a generic service container. It is just enough runtime-owned state to carry forward what registrars already compute.

### Part 2: Install a runtime-owned cancellation context

Create a root context for each owned runtime in [`Factory.NewRuntime`](../../../../../../engine/factory.go):

```go
runtimeCtx, runtimeCtxCancel := context.WithCancel(context.Background())
```

Store the context and cancel function on the runtime, then invoke `runtimeCtxCancel()` during `Runtime.Close(...)` before closing owned resources. This provides one runtime-scoped cancellation root for plugin calls and future runtime-integrated features.

### Part 3: Route plugin invocation through the runtime context

Update plugin reification so exported JS wrappers invoke plugins with `runtime.Context()` instead of `context.Background()`. `LoadedModule.Invoke(...)` in [`client.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/client.go) should keep layering call timeouts on top of the provided context, so the behavior becomes:

- runtime context sets the lifetime root
- timeout adds an upper bound when the caller has not already set a deadline

### Part 4: Strengthen diagnostics at the loader/report layer

Preserve the current `LoadReport` concept, but improve the fidelity of the information it contains:

- capture bounded stderr for plugin startup/load failures
- keep directory-level and candidate-level failures distinct
- make summaries reflect errors before successes
- let entrypoints format the report instead of inventing extra heuristics

### Runtime data flow after the change

```text
Builder
  -> Factory.NewRuntime
       -> create RuntimeModuleContext
       -> run registrars
            -> plugin registrar stores loaded-module snapshot
            -> docs registrar stores doc hub
       -> create runtime-owned context
       -> build Runtime
            -> copy moduleCtx.Values into Runtime.Values
            -> attach runtime context
       -> return Runtime

Consumers
  -> runtime.Value("hashiplugin.loadedModules")
  -> runtime.Value("docaccess.hub")
  -> runtime.Context()
```

### Plugin invocation flow after the change

```text
require("plugin:examples:kv")
  -> reified JS wrapper
      -> use runtime.Context()
      -> apply timeout in LoadedModule.Invoke when needed
      -> gRPC call plugin subprocess
      -> runtime closes
           -> runtime context cancels
           -> in-flight plugin call observes cancellation
```

## Design Decisions

### Decision: add state directly to `engine.Runtime`

Rationale:

- the runtime already owns lifecycle and cleanup
- registrars already act as runtime-scoped contributors
- a simple value map plus accessor is sufficient for current needs

The project does not need a heavier dependency-injection or service-registry abstraction here.

### Decision: fix diagnostics in the host layer, not in CLI wrappers

Rationale:

- plugin loading should be diagnosable regardless of whether the caller is `repl`, `js-repl`, tests, or future embedding code
- the host layer is where process startup, manifest fetch, and validation failures actually happen

### Decision: use runtime-owned cancellation as the plugin invocation root

Rationale:

- runtime shutdown should mean something operationally
- the runtime already models ownership explicitly
- future evaluator-side cancellation becomes easier if the base seam already exists

## Alternatives Considered

### Alternative 1: keep setup-time values ephemeral

This avoids touching `engine.Runtime`, but it pushes the same problem into every consumer. REPLs, evaluators, and future docs features would all need their own copies of setup-time state or side channels to reach it.

### Alternative 2: create a general runtime service registry

This would be more structured than a simple value map, but it is premature. The runtime currently only needs a small number of durable values:

- loaded plugin snapshot
- docs hub
- future evaluator helpers

Adding a more abstract registry now would create ceremony without solving a more urgent problem.

### Alternative 3: improve only entrypoint output

Better CLI messaging would help a little, but it would still rely on weak underlying signals. Diagnostics should be captured at load time, not reconstructed later from partial state.

## Implementation Plan

### Phase 1: Runtime value persistence

1. Extend [`engine.Runtime`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go) with:
   - `Values map[string]any`
   - `runtimeCtx context.Context`
   - `runtimeCtxCancel context.CancelFunc`
2. Add `Value(key string) (any, bool)` and `Context() context.Context`.
3. Update [`engine/factory.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go) to:
   - allocate the runtime context
   - copy `moduleCtx.Values` into the returned runtime
   - attach the context to the runtime
4. Cancel the runtime context in `Runtime.Close(...)` before other cleanup.

### Phase 2: Plugin cancellation propagation

1. Update [`pkg/hashiplugin/host/reify.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/reify.go) to call `loaded.Invoke(rt.Context(), req)`.
2. Keep timeout wrapping in [`client.go`](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/client.go).
3. Add tests that show runtime close/cancellation reaches plugin calls.

### Phase 3: Diagnostics hardening

1. Capture bounded plugin stderr in the loader.
2. Preserve directory discovery failures explicitly in the report.
3. Make report summaries prioritize failures.
4. Add tests for invalid plugins, failed discovery, and diagnostic text retention.

### Phase 4: Follow-on consumers

1. Persist the docs hub into runtime values.
2. Use the runtime value seam in evaluator-side documentation help.
3. Consolidate duplicated entrypoint plugin bootstrap/report helpers after the report surface stabilizes.

## Testing Strategy

- add engine tests for value copying and runtime context access
- add host tests for plugin summary/detail rendering in failure-heavy cases
- add cancellation-focused host/runtime tests
- regression test docs registrar and plugin registrar once runtime values persist

## Open Questions

- Should `Runtime.Values` remain exported or become private after the accessor methods are in place?
- How much stderr should be retained per plugin before truncation?
- Should missing default plugin directories be silent, informational, or warning-level in user-facing summaries?

## References

- Review report: `design-doc/01-origin-main-review-report-for-plugin-and-documentation-architecture.md`
- Owned runtime creation: [factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go)
- Owned runtime lifecycle: [runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go)
- Setup-time registrar context: [runtime_modules.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime_modules.go)
- Plugin loader: [client.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/client.go)
- Plugin reification: [reify.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/reify.go)
- Plugin reporting: [report.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/hashiplugin/host/report.go)
- Docs registrar: [registrar.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/docaccess/runtime/registrar.go)
