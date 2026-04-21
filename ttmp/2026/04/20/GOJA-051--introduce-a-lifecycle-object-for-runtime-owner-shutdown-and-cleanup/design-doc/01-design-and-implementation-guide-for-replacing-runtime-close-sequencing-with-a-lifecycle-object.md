---
Title: Design and implementation guide for replacing runtime close sequencing with a lifecycle object
Ticket: GOJA-051
Status: active
Topics:
    - goja
    - engine
    - lifecycle
    - repl
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: |-
        Runtime construction path and current injection of AddCloser into module registrars.
        Runtime construction path and current lifecycle participant wiring
    - Path: engine/runtime.go
      Note: |-
        Current runtime close orchestration and generic closer registration surface.
        Current runtime close sequencing and generic closer registration surface
    - Path: engine/runtime_modules.go
      Note: |-
        RuntimeModuleContext currently exposes only AddCloser instead of a richer lifecycle API.
        RuntimeModuleContext currently exposes only AddCloser
    - Path: engine/runtime_modules_test.go
      Note: |-
        Existing tests encode the current reverse-order closer contract.
        Existing tests encoding the reverse-order closer contract
    - Path: pkg/hashiplugin/host/registrar.go
      Note: |-
        Real runtime-scoped cleanup consumer that currently depends on AddCloser.
        Existing runtime-scoped cleanup consumer that depends on AddCloser
    - Path: pkg/runtimeowner/runner.go
      Note: |-
        Concrete shutdown semantics for the current owner runner implementation.
        Concrete owner shutdown behavior
    - Path: pkg/runtimeowner/types.go
      Note: |-
        Owner abstraction that is currently embedded directly in Runtime.Close sequencing.
        Owner interface contract used by Runtime.Close
ExternalSources: []
Summary: Evidence-backed design for replacing ad hoc runtime shutdown sequencing with an explicit lifecycle object and phase-aware cleanup registration API.
LastUpdated: 2026-04-20T11:30:00-04:00
WhatFor: Guide future cleanup of engine.Runtime close semantics so shutdown order, cleanup registration, and owner/event-loop teardown become explicit and testable.
WhenToUse: Use when refactoring runtime ownership, module cleanup, plugin lifecycle management, or close-time ordering in go-go-goja.
---


# Design and implementation guide for replacing runtime close sequencing with a lifecycle object

## Executive Summary

`engine.Runtime.Close()` currently acts as the entire shutdown protocol for a runtime. It cancels the runtime context, detaches the runtime bridge, runs a generic stack of cleanup hooks, shuts down the runtime owner, and stops the event loop in one method body. That implementation works, but it compresses too many lifecycle decisions into one opaque sequence and exposes only one generic extension point: `AddCloser(func(context.Context) error)`.

That shape is now too weak for the responsibilities the runtime carries. The engine already has multiple lifecycle participants with different shutdown needs: the owner runner, the event loop, runtime-scoped module registrars, bridge bindings, plugin-backed modules, and runtime initializers. Some resources must be cleaned up before owner shutdown. Others must not touch the VM anymore. Some want a cancellation signal before their cleanup runs. A single reverse-order closer stack does not make those distinctions explicit.

This ticket proposes introducing a dedicated lifecycle object for `engine.Runtime` with named shutdown phases and explicit hook registration. The key goal is to make runtime shutdown composable and understandable without having to reverse-engineer one method body. The guide below explains the current behavior, identifies the API gaps, proposes a lifecycle model, and lays out an implementation plan that preserves migration safety while moving the engine toward a cleaner ownership model.

## Problem Statement and Scope

The immediate problem is not that `Runtime.Close()` is broken. The problem is that the shutdown protocol is encoded as implicit sequencing inside one method, and other runtime participants can only register generic closers without expressing phase intent.

Observed current behavior:

1. `engine.Runtime` stores a single list of close hooks in `closers []func(context.Context) error` guarded by `closerMu` and `closeOnce` (`engine/runtime.go:25-38`).
2. `AddCloser` accepts only one generic hook type and only documents that hooks run before the runtime owner and event loop are shut down (`engine/runtime.go:58-75`).
3. `Close` performs five different actions in one block: cancel context, delete runtime bridge, run generic closers in reverse order, call `Owner.Shutdown(ctx)`, then call `Loop.Stop()` (`engine/runtime.go:79-114`).
4. `RuntimeModuleContext` exposes `AddCloser` to runtime-scoped registrars, so those registrars can attach cleanup only to the one generic close bucket (`engine/runtime_modules.go:19-27`).
5. Real runtime-scoped cleanup consumers, such as the HashiCorp plugin registrar, already use that generic hook to tear down loaded plugin modules (`pkg/hashiplugin/host/registrar.go:71-82`).

This ticket is specifically about the engine/runtime ownership boundary and cleanup API. It is not a general redesign of `pkg/runtimeowner.Runner` semantics, and it is not a request to rework all module APIs at once. The scope is:

- the shutdown model of `engine.Runtime`,
- the registration API exposed to runtime-scoped participants,
- the migration path for existing `AddCloser` consumers,
- and the tests and documentation needed to make the new lifecycle contract durable.

## Current-State Analysis

### Runtime shutdown is implemented as one implicit sequence

The current `Runtime` struct keeps lifecycle state directly on the runtime object:

- `closeOnce sync.Once`
- `closerMu sync.Mutex`
- `closers []func(context.Context) error`
- `closing bool`

This is visible in `engine/runtime.go:25-38`.

`Runtime.Close` then executes the full shutdown protocol inline:

1. mark the runtime as closing and snapshot the current closer list,
2. cancel the runtime-owned context,
3. delete the VM from `runtimebridge`,
4. run registered closers in reverse order,
5. call `Owner.Shutdown(ctx)`,
6. stop the event loop.

That ordering is implemented at `engine/runtime.go:85-111`.

The method is compact, but the compactness hides an important architectural fact: different shutdown actions have different invariants. Context cancellation, VM bridge detachment, third-party cleanup hooks, owner shutdown, and loop stop are not the same kind of operation.

### Runtime construction already has multiple lifecycle participants

`Factory.NewRuntime` assembles a runtime from several pieces:

- a fresh `goja.Runtime`,
- a Node-style event loop,
- a `runtimeowner.Runner`,
- runtime bridge bindings,
- runtime module registrars,
- and runtime initializers.

That construction is visible in `engine/factory.go:154-230`.

The important detail is that `RuntimeModuleContext` gives registrars access to:

- `Context`,
- `VM`,
- `Loop`,
- `Owner`,
- `AddCloser`,
- `Values`.

See `engine/runtime_modules.go:19-27`.

So the engine already acknowledges runtime-scoped lifecycle work. The API problem is that it models all custom cleanup as one undifferentiated `AddCloser` stack.

### The owner abstraction is intentionally minimal

`pkg/runtimeowner.Runner` exposes:

- `Call(...)`
- `Post(...)`
- `Shutdown(ctx)`
- `IsClosed()`

See `pkg/runtimeowner/types.go:20-25`.

The concrete runner implementation currently treats `Shutdown` as a simple closed-state transition:

```go
func (r *runner) Shutdown(context.Context) error {
    if r == nil {
        return nil
    }
    r.closed.Store(true)
    return nil
}
```

That behavior is in `pkg/runtimeowner/runner.go:54-59`.

This is not itself wrong, but it means the engine currently assumes `Runtime.Close` is the place where owner shutdown, scheduler rejection, and event loop stop are braided together.

### Tests encode the current generic reverse-order closer model

The existing engine tests validate two important pieces of behavior:

1. generic closers run in reverse registration order (`engine/runtime_modules_test.go:121-154`),
2. runtime module registrars can register cleanup hooks through `ctx.AddCloser(...)` and those hooks run on close (`engine/runtime_modules_test.go:49-65`, `156-170`).

Those tests are useful, but they also show the current limitation: the testable contract is a single LIFO closer stack, not a richer lifecycle model.

### Real consumers already want runtime-scoped cleanup

The HashiCorp plugin registrar loads runtime-scoped plugin modules and then registers a cleanup hook through `ctx.AddCloser(...)` that closes the loaded module set (`pkg/hashiplugin/host/registrar.go:71-82`).

This is exactly the type of consumer that benefits from explicit lifecycle phases. Plugin-backed resources are not “just another closer”; they are runtime-scoped infrastructure with ordering constraints.

## Gap Analysis

The current model has four important gaps.

### 1. Cleanup intent is not representable

`AddCloser` tells the engine only that “this should happen during close.” It does not say:

- must run before owner shutdown,
- must run after cancellation,
- must not touch the VM,
- must run before event loop stop,
- or must run after bridge detachment.

That means ordering intent is forced into registration order rather than declared explicitly.

### 2. `Runtime.Close` is harder to reason about than it needs to be

A reader has to learn the shutdown protocol by reading imperative code. There is no explicit lifecycle object, no phase model, and no domain name for the shutdown plan.

### 3. The extension point is too narrow for future engine work

If future runtime participants need different hooks for “before owner shutdown” versus “after owner shutdown,” the current API either grows ad hoc methods (`AddBeforeOwnerCloser`, `AddAfterOwnerCloser`, etc.) or continues overloading one generic closer stack.

### 4. Tests can only verify the accidental current sequencing model

Today the tests can assert reverse-order close hooks. They cannot assert higher-level lifecycle claims such as:

- owner shutdown always happens after runtime cleanup hooks,
- loop stop always happens even if a cleanup hook fails,
- hooks registered for one phase never interleave with another phase,
- or bridge deletion happens before any post-bridge phase starts.

## Proposed Solution

Introduce an explicit lifecycle object owned by `engine.Runtime`.

### Core design

Add a new engine-level type, for example in `engine/lifecycle.go`:

```go
type LifecyclePhase string

const (
    PhaseCancelContext   LifecyclePhase = "cancel-context"
    PhaseDetachBridge    LifecyclePhase = "detach-bridge"
    PhaseRuntimeCleanup  LifecyclePhase = "runtime-cleanup"
    PhaseOwnerShutdown   LifecyclePhase = "owner-shutdown"
    PhaseLoopStop        LifecyclePhase = "loop-stop"
)

type LifecycleHook func(context.Context) error

type Lifecycle struct {
    once    sync.Once
    mu      sync.Mutex
    closing bool
    hooks   map[LifecyclePhase][]LifecycleHook
    order   []LifecyclePhase
}
```

The runtime should own one `Lifecycle` instance and delegate shutdown orchestration to it.

### Runtime shape after refactor

```go
type Runtime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner
    Values  map[string]any

    runtimeCtx       context.Context
    runtimeCtxCancel context.CancelFunc

    lifecycle *Lifecycle
}
```

The main conceptual shift is this:

- `Runtime` still owns the runtime pieces,
- but `Lifecycle` owns shutdown registration and shutdown execution.

### New lifecycle API

The lifecycle object should expose explicit hook registration by phase.

```go
func NewLifecycle(order ...LifecyclePhase) *Lifecycle
func (l *Lifecycle) Register(phase LifecyclePhase, hook LifecycleHook) error
func (l *Lifecycle) Close(ctx context.Context) error
func (l *Lifecycle) IsClosing() bool
```

Then `Runtime` can expose a thin convenience API:

```go
func (r *Runtime) RegisterLifecycleHook(phase LifecyclePhase, hook LifecycleHook) error
func (r *Runtime) Close(ctx context.Context) error
```

### Recommended phase model

For the current engine shape, the following default phase order is enough:

1. `PhaseCancelContext`
2. `PhaseDetachBridge`
3. `PhaseRuntimeCleanup`
4. `PhaseOwnerShutdown`
5. `PhaseLoopStop`

Why this order:

- Context cancellation should happen first so runtime-scoped participants see closure early.
- Runtime bridge detachment should happen before generic cleanup hooks that must no longer rely on bridge lookups.
- Runtime cleanup hooks should finish before owner shutdown if they still need orderly resource cleanup around owner-owned structures.
- Owner shutdown should happen before loop stop so future owner implementations have a stable scheduler boundary.
- Loop stop should be last and unconditional.

### Registration surfaces

`RuntimeModuleContext` should stop exposing only `AddCloser` and instead expose the lifecycle registration surface directly:

```go
type RuntimeModuleContext struct {
    Context            context.Context
    VM                 *goja.Runtime
    Loop               *eventloop.EventLoop
    Owner              runtimeowner.Runner
    RegisterLifecycle  func(LifecyclePhase, LifecycleHook) error
    Values             map[string]any
}
```

The plugin registrar would then become explicit about intent:

```go
if ctx != nil && ctx.RegisterLifecycle != nil {
    err := ctx.RegisterLifecycle(PhaseRuntimeCleanup, func(context.Context) error {
        closeLoaded(loaded)
        return nil
    })
}
```

That change is not just cosmetic. It documents that plugin module teardown belongs to runtime cleanup, not to owner shutdown or loop stop.

### Compatibility and migration guidance

A zero-flag-day migration is reasonable here because there are already `AddCloser` consumers.

Recommended migration rule:

- keep `Runtime.AddCloser(...)` temporarily,
- implement it as a thin wrapper over `RegisterLifecycleHook(PhaseRuntimeCleanup, hook)`,
- update all internal call sites to use explicit phase registration,
- then decide whether to keep `AddCloser` as a convenience alias or remove it.

That lets the engine evolve without forcing unrelated call sites to change in one patch.

If the project prefers stricter cleanup over compatibility, it can remove `AddCloser` immediately and migrate all internal call sites in one change. The more conservative path is probably better for this repository because `RuntimeModuleContext` and plugin integration already depend on the old surface.

## Design Decisions

### Decision 1: use a lifecycle object instead of more helper methods on `Runtime`

Rationale:

- The problem is structural, not just cosmetic.
- Splitting `Close` into `beginShutdown`, `runClosers`, `shutdownOwner`, and `stopLoop` helpers would improve readability, but it would not improve the registration model.
- A lifecycle object gives the shutdown protocol a name and an API.

### Decision 2: use named phases instead of a single LIFO hook stack

Rationale:

- Ordering intent becomes explicit.
- New runtime participants can declare where they belong without relying on registration order accidents.
- Tests can validate phase boundaries directly.

### Decision 3: keep phase count small

Rationale:

- The current engine does not need a large state machine.
- Too many phases would make the API feel abstract before it proves its value.
- Five phases are enough to model the current close path cleanly.

### Decision 4: keep owner shutdown and loop stop as lifecycle-managed steps

Rationale:

- They are part of the runtime shutdown protocol, even if they are not externally registered hooks.
- Treating them as lifecycle phases keeps the whole shutdown sequence under one orchestration model.

### Decision 5: preserve error aggregation semantics

Rationale:

- `Runtime.Close` currently uses `errors.Join(...)` to retain multiple cleanup failures (`engine/runtime.go:99-107`).
- The lifecycle object should keep that behavior so shutdown remains failure-reporting rather than failure-short-circuiting.

## Pseudocode and Key Flows

### Lifecycle close orchestration

```go
func (l *Lifecycle) Close(ctx context.Context) error {
    if l == nil {
        return nil
    }

    var retErr error
    l.once.Do(func() {
        l.mu.Lock()
        l.closing = true
        hooks := cloneHooksByPhase(l.hooks)
        order := append([]LifecyclePhase(nil), l.order...)
        l.hooks = nil
        l.mu.Unlock()

        for _, phase := range order {
            phaseHooks := hooks[phase]
            for i := len(phaseHooks) - 1; i >= 0; i-- {
                if err := phaseHooks[i](ctx); err != nil {
                    retErr = errors.Join(retErr, fmt.Errorf("phase %s: %w", phase, err))
                }
            }
        }
    })
    return retErr
}
```

### Runtime construction wiring

```go
func newRuntimeLifecycle(rt *Runtime) *Lifecycle {
    lc := NewLifecycle(
        PhaseCancelContext,
        PhaseDetachBridge,
        PhaseRuntimeCleanup,
        PhaseOwnerShutdown,
        PhaseLoopStop,
    )

    _ = lc.Register(PhaseCancelContext, func(context.Context) error {
        if rt.runtimeCtxCancel != nil {
            rt.runtimeCtxCancel()
        }
        return nil
    })

    _ = lc.Register(PhaseDetachBridge, func(context.Context) error {
        if rt.VM != nil {
            runtimebridge.Delete(rt.VM)
        }
        return nil
    })

    _ = lc.Register(PhaseOwnerShutdown, func(ctx context.Context) error {
        if rt.Owner != nil {
            return rt.Owner.Shutdown(ctx)
        }
        return nil
    })

    _ = lc.Register(PhaseLoopStop, func(context.Context) error {
        if rt.Loop != nil {
            rt.Loop.Stop()
        }
        return nil
    })

    return lc
}
```

### Compatibility wrapper

```go
func (r *Runtime) AddCloser(fn func(context.Context) error) error {
    return r.RegisterLifecycleHook(PhaseRuntimeCleanup, fn)
}
```

## Alternatives Considered

### Alternative A: only split `Runtime.Close()` into helper methods

Example direction:

- `cancelContext()`
- `detachBridge()`
- `runClosers()`
- `shutdownOwner()`
- `stopLoop()`

Why rejected as the primary solution:

- It improves readability but leaves the registration model unchanged.
- Runtime participants would still have only one generic closer bucket.
- The main architectural problem would remain.

### Alternative B: add multiple ad hoc registration methods on `Runtime`

Example direction:

- `AddBeforeOwnerShutdownHook(...)`
- `AddAfterOwnerShutdownHook(...)`
- `AddLoopStopHook(...)`

Why rejected:

- It scales poorly as the lifecycle evolves.
- The runtime type becomes a bag of procedural shutdown APIs.
- The shutdown model is still not first-class.

### Alternative C: move all shutdown intelligence into `runtimeowner.Runner`

Why rejected:

- The owner runner is only one participant in shutdown.
- `Runtime.Close` also manages bridge deletion, runtime-scoped hooks, context cancellation, and event loop stop.
- Pushing all of that into `Runner` would incorrectly couple engine lifecycle to owner implementation details.

### Alternative D: keep the current closer stack and rely on documentation only

Why rejected:

- The problem is not lack of prose. The current API cannot represent phase intent.
- Documentation alone would not make tests stronger or future changes safer.

## Implementation Plan

### Phase 1: add lifecycle primitives without changing behavior

Files:

- `engine/lifecycle.go` (new)
- `engine/runtime.go`

Work:

1. Add `Lifecycle`, `LifecyclePhase`, and hook registration APIs.
2. Move `closeOnce`, `closerMu`, `closers`, and `closing` out of `Runtime` into `Lifecycle`.
3. Build the current close sequence as a lifecycle with phases matching current behavior.
4. Keep `Runtime.AddCloser(...)` as a compatibility wrapper to `PhaseRuntimeCleanup`.

Validation:

- existing runtime close tests still pass,
- no call site changes required yet.

### Phase 2: expose lifecycle registration in runtime-scoped contexts

Files:

- `engine/runtime_modules.go`
- `engine/factory.go`
- `pkg/hashiplugin/host/registrar.go`

Work:

1. Add `RegisterLifecycle` to `RuntimeModuleContext`.
2. Populate it from `Factory.NewRuntime`.
3. Migrate runtime module registrars that currently call `ctx.AddCloser(...)`.
4. Decide whether to keep `AddCloser` on `RuntimeModuleContext` temporarily as a wrapper for migration ease.

Validation:

- plugin registrar cleanup still runs,
- runtime module tests still verify cleanup behavior,
- no regression in runtime creation and teardown.

### Phase 3: strengthen tests around lifecycle phases

Files:

- `engine/runtime_modules_test.go`
- `engine/lifecycle_test.go` (new)

Work:

Add tests for:

1. phase ordering,
2. reverse ordering within a phase,
3. continued execution of later phases after earlier hook errors,
4. owner shutdown and loop stop still happen when runtime cleanup hooks fail,
5. late registration rejection after closing begins.

Validation commands:

```bash
go test ./engine/... -count=1
```

### Phase 4: decide final public surface

Options:

1. Keep `AddCloser` permanently as a convenience alias for `PhaseRuntimeCleanup`.
2. Deprecate and then remove it once all internal callers migrate to phase-aware registration.

Recommended direction:

- keep it for one cleanup cycle as an adapter,
- migrate all internal code to explicit lifecycle registration,
- then decide based on whether external consumers exist.

## Testing and Validation Strategy

The implementation should be considered complete only if it adds tests for lifecycle semantics, not just superficial API coverage.

### Required tests

1. `TestLifecycleRunsPhasesInDeclaredOrder`
2. `TestLifecycleRunsHooksInReverseOrderWithinPhase`
3. `TestLifecycleAggregatesErrorsAcrossPhases`
4. `TestRuntimeCloseStillRunsOwnerShutdownAndLoopStopAfterCleanupFailure`
5. `TestRuntimeModuleRegistrarCanRegisterPhaseAwareCleanup`
6. `TestLifecycleRejectsLateRegistration`

### Regression protection

Existing tests that currently anchor the generic closer behavior should either be preserved or rewritten to validate the equivalent lifecycle guarantees:

- `TestRuntimeCloseRunsClosersInReverseOrder`
- `TestRuntimeCloseRunsRegistrarClosers`

### Manual review focus

Reviewers should inspect:

- whether any hook is assigned to the wrong phase,
- whether bridge deletion timing matches module expectations,
- whether owner shutdown remains a no-op-safe operation,
- whether loop stop remains unconditional.

## Risks, Tradeoffs, and Open Questions

### Risks

1. A migration wrapper may let phase ambiguity linger longer than desired.
2. Incorrect phase assignment could create subtle shutdown regressions even if the API shape improves.
3. Some cleanup consumers may actually require owner-thread execution, which should be documented explicitly if discovered.

### Tradeoffs

- The lifecycle object adds one new abstraction, but it removes hidden policy from `Runtime.Close`.
- A phase model is slightly more ceremony than a single closer list, but it is a better fit for the engine’s actual responsibilities.
- A compatibility wrapper slows full cleanup, but it makes rollout safer.

### Open questions

1. Should the lifecycle object remain internal to `engine`, or should callers be allowed to inspect phases directly?
2. Should phase-specific hooks support metadata such as names for tracing/logging?
3. Should owner shutdown eventually become stronger than a closed-flag transition in `runtimeowner.Runner`, or is that intentionally outside scope?

## References

Evidence-backed references used for this guide:

- `engine/runtime.go:25-38` — runtime lifecycle state currently stored directly on `Runtime`
- `engine/runtime.go:58-75` — generic `AddCloser` registration surface
- `engine/runtime.go:79-114` — current `Close` sequencing
- `engine/factory.go:154-230` — runtime construction and lifecycle participant wiring
- `engine/runtime_modules.go:19-27` — `RuntimeModuleContext` currently exposing only `AddCloser`
- `pkg/runtimeowner/types.go:20-25` — owner interface
- `pkg/runtimeowner/runner.go:54-59` — current owner shutdown behavior
- `pkg/hashiplugin/host/registrar.go:71-82` — plugin cleanup using generic closer registration
- `engine/runtime_modules_test.go:49-65` — registrar cleanup test setup
- `engine/runtime_modules_test.go:121-170` — tests for reverse-order closers and registrar close behavior
