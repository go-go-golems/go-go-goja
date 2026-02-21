---
Title: EngineFactory WithModules architecture deep dive
Ticket: GC-05-ENGINE-MODULE-COMPOSITION
Status: active
Topics:
    - go
    - architecture
    - refactor
    - tooling
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../engine/factory.go
      Note: Current Factory baseline and insertion point for WithModules-style composition
    - Path: ../../../../../../engine/options.go
      Note: Current Option model to integrate module-oriented options
    - Path: ../../../../../../engine/runtime.go
      Note: Compatibility path currently routing Open/New through Factory
    - Path: ../../../../../../modules/common.go
      Note: Global DefaultRegistry implementation and EnableAll behavior motivating explicit composition
    - Path: packages/engine/src/chat/runtime/registerChatModules.ts
      Note: One-time global bootstrap stopgap used before full factory-level modularization
    - Path: packages/engine/src/chat/sem/semRegistry.ts
      Note: External registry reset behavior that illustrated handler-loss failure mode
ExternalSources: []
Summary: Detailed investigation and design brainstorming on EngineFactory module composition, lifecycle, ordering/dependencies, typing strategy, and migration path from global registries.
LastUpdated: 2026-02-20T10:33:15.789070437-05:00
WhatFor: Capture detailed architecture analysis and design options for introducing module-aware EngineFactory composition without relying on global mutable registries.
WhenToUse: Use when planning a follow-up implementation ticket for EngineFactory module ordering, dependency validation, and extension lifecycle.
---


# EngineFactory WithModules Architecture Deep Dive

## Scope

This document captures a detailed investigation and design brainstorming session around evolving EngineFactory toward explicit module composition:

- `EngineFactory.WithModules(...)` style configuration
- optional later registration before runtime creation
- deterministic module ordering and dependency handling
- conflict detection and safety guarantees
- typing/ergonomics tradeoffs
- migration strategy from global registration patterns

The immediate trigger was a practical runtime issue in a separate codebase where global SEM registries could lose handlers if default registration cleared shared maps.

## Prompt Context

The design questions that drove this write-up were:

1. Should handler/module registration be global and one-time?
2. Would an `EngineFactory` with module options be a better long-term model?
3. What are the concrete pros/cons of `WithModules(...)`?
4. What exactly is the ordering/dependency downside?
5. Why does typing get harder in extensible module systems?

## Investigation Summary

### A) Current go-go-goja EngineFactory shape

Current implementation in `engine/factory.go`:

- `NewFactory(opts...)` computes settings and creates one `require.Registry`
- calls `modules.EnableAll(reg)` unconditionally
- `NewRuntime()` creates a VM, enables require module via the prepared registry

Current module mechanism in `modules/common.go`:

- modules self-register through package `init()` into `modules.DefaultRegistry`
- `EnableAll(reg)` iterates global registry and registers every module
- no duplicate module name detection
- no declared dependency graph
- no deterministic conflict policy beyond iteration order

### B) Observed failure mode in global registry systems

In the hypercard/webchat runtime work (external project context), a default registration path previously cleared the handler map and then re-registered base handlers. Extension handlers registered elsewhere were dropped when this happened later in lifecycle, causing event types to be silently ignored.

This is what "avoid losing handlers" means in concrete terms:

1. shared mutable global registry exists
2. one codepath performs reset/rebuild of defaults
3. extension modules assume prior registration still holds
4. extension handlers disappear after reset
5. runtime behavior degrades non-obviously (events no longer handled)

One-time idempotent bootstrap fixed the immediate issue there, but it is still global mutable state.

## Problem Statement

The current approach mixes these concerns:

1. bootstrap policy (what gets installed)
2. lifecycle timing (when install occurs)
3. conflict/dependency semantics
4. extensibility boundary (who can add modules)

Without explicit composition semantics, modules become order-sensitive and difficult to validate.

## Design Goal

Introduce an explicit composition model where:

1. module set is visible and reviewable
2. module install order is deterministic
3. dependencies and conflicts are validated before runtime use
4. built factory/runtime state is immutable
5. dynamic extension rules are explicit (allowed or forbidden)

## Candidate API Shapes

### Option 1: Constructor-only modules

```go
f, err := engine.NewFactory(
    engine.WithModules(base.Module(), hypercard.Module()),
)
```

Pros:

- simplest lifecycle
- easy validation path
- immutable by default

Cons:

- awkward when composition is built incrementally by different packages

### Option 2: Builder-style `WithModules(...)` then `Build()` (recommended)

```go
builder := engine.NewEngineFactory().
    WithOptions(engine.WithRequireOptions(...)).
    WithModules(base.Module(), hypercard.Module())

builder = builder.WithModules(myCustom.Module()) // still pre-build

factory, err := builder.Build()
rt, req := factory.NewRuntime()
```

Pros:

- ergonomic for staged assembly
- clear lifecycle boundary at `Build()`
- enables strong preflight validation

Cons:

- must prevent accidental builder reuse or mutation after `Build()`

### Option 3: Runtime hot-add (`factory.UseModule(...)`)

```go
factory.UseModule(livePatch.Module())
```

Pros:

- maximum flexibility

Cons:

- hardest to reason about correctness
- can cause behavior drift between already-created and future runtimes
- much higher testing and support burden

Recommendation: avoid this initially.

## Recommended Architecture

Adopt Option 2 with strict phase separation:

1. **Compose phase** (mutable): collect modules/options
2. **Build phase** (validation): resolve order, check conflicts, finalize plan
3. **Runtime phase** (immutable): create runtimes from frozen plan

### Core types (proposed sketch)

```go
type EngineModule interface {
    ID() string
    DependsOn() []string
    Register(*RegistrationContext) error
}

type RegistrationContext struct {
    RequireRegistry *require.Registry
    // future extension points:
    // EventRegistry, RendererRegistry, CodecRegistry, Diagnostics, etc.
}

type FactoryBuilder struct {
    opts    []Option
    modules []EngineModule
    built   bool
}

type Factory struct {
    settings      openSettings
    registry      *require.Registry
    installedPlan []string // ordered module IDs
}
```

## Module Ordering and Dependency: Detailed Risk Analysis

This was called out as a primary con because extension systems fail most often here.

### What can go wrong

1. **Override collisions**
- Two modules register the same symbol (module name, handler key, renderer kind).
- "Last write wins" silently changes behavior.

2. **Implicit prerequisites**
- Module B expects artifacts from A (normalizers, handlers, codecs).
- If B registers first, B may partially initialize or panic.

3. **Dependency cycles**
- A depends on B and B depends on A, directly or transitively.
- Installation order cannot be resolved.

4. **Non-deterministic order**
- If order depends on map iteration or unstable discovery, behavior differs across runs.

### Required safeguards

1. Each module must have stable `ID()`.
2. Dependencies must be declarative (`DependsOn()`).
3. Build must perform deterministic topological sort.
4. Cycles and missing dependencies must fail fast with explicit diagnostics.
5. Duplicate registration keys should fail by default (or require explicit override policy).

### Expected build-time validation output

- missing dependencies: `"module hypercard requires base-sem (not provided)"`
- cycle detection: `"dependency cycle: a -> b -> c -> a"`
- key conflict: `"renderer key tool_result registered by both base and overrideX"`

## Typing Complexity: What Gets Harder

Typing gets harder when modules can extend runtime capabilities.

### In Go (this repo)

Go lacks TypeScript-like inferred intersection types for arbitrary plugin composition. As modules add capabilities, we generally choose between:

1. compile-time explicit interfaces (safe but rigid)
2. capability maps / interface{} lookups (flexible but less safe)
3. generic wrappers (possible but can get verbose and brittle)

Main pressure points:

- expressing "module B requires capability provided by A" in type-safe API
- keeping `EngineModule` contract simple while allowing richer context
- avoiding a giant context type with too many optional fields

### In TypeScript (cross-project context)

Typing challenges appear as:

- accumulating context type through builder chain (`C -> C & Added`)
- variadic tuple inference for `WithModules(a,b,c)`
- loss of precision when modules are loaded dynamically at runtime

### Practical compromise

1. Keep module contract narrow (`Register(ctx) error`)
2. Enforce dependency contracts primarily at runtime validation
3. Use typed helper accessors for well-known registries
4. Avoid over-engineering a "fully inferred plugin type algebra"

## Pros/Cons of `EngineFactory.WithModules(...)`

### Pros

1. explicit composition at callsite
2. no hidden global side effects
3. deterministic reproducible startup behavior
4. easier test isolation (multiple independent factories)
5. better diagnostics for dependency/conflict issues
6. cleaner future extension surface

### Cons

1. more API surface and implementation complexity
2. stricter lifecycle means slightly more ceremony
3. requires module metadata discipline (`ID`, deps, conflict keys)
4. migration work from global `init()` registration model

## Migration Strategy from Current go-go-goja State

### Current baseline

- module discovery via global `modules.DefaultRegistry` and blank imports
- factory always calls `modules.EnableAll(reg)`

### Migration phases

1. **Phase M1: additive bridge**
- introduce `EngineModule` and builder API
- keep existing global path as default fallback

2. **Phase M2: first-class module set**
- add `WithModules(...)`
- implement ordering/validation engine
- generate explicit installed plan for observability

3. **Phase M3: deprecate global path**
- retain compatibility shim with warning
- shift docs/examples to explicit module composition

4. **Phase M4: hard cutover**
- remove hidden `EnableAll` default path (or gate behind explicit opt-in)

## Suggested Conflict Policy

Default policy: **fail on duplicate key registration**.

Allow explicit override only via API flag:

```go
engine.WithConflictPolicy(engine.ConflictAllowOverride)
```

Rationale:

- safer default in production systems
- makes overrides intentional and reviewable

## Testing Plan for Follow-up Ticket

### Unit tests

1. topological sort success/failure
2. cycle detection
3. missing dependency detection
4. duplicate key conflict detection
5. idempotent Build behavior and builder freeze semantics

### Integration tests

1. build factory with base + custom modules, create runtime, execute `require()`
2. verify runtime instances are isolated and deterministic
3. verify explicit module subset excludes unavailable modules

### Regression tests

1. "lost handler/module" class of bug (extension survives normal bootstrap)
2. deterministic install plan equality across repeated runs

## Implementation Sketch (Algorithm)

1. collect modules into `map[id]module`
2. validate unique IDs
3. validate all dependencies exist
4. run topological sort on dependency graph
5. instantiate `require.Registry`
6. register modules in sorted order
7. freeze `Factory` with immutable install plan

## Open Questions

1. Should existing blank-import module registration remain supported long-term?
2. Should module dependency be strict hard dependency only, or also support soft/optional dependencies?
3. Should module conflict keys be standardized (e.g., `require:<name>`, `handler:<event>`)?
4. Should build metadata be exposed for diagnostics (`Factory.Plan()`)?
5. Should runtime-specific modules ever be allowed post-build? (default recommendation: no)

## Recommendation

Proceed with a follow-up implementation ticket using:

1. `FactoryBuilder.WithModules(...)` pre-build API
2. deterministic dependency solver
3. fail-fast conflict/missing-dependency checks
4. immutable built factory
5. compatibility bridge for existing global module registration

This gives the extensibility benefits requested while avoiding the global-state fragility observed in handler registry systems.

## Appendix A: Why One-time Global Bootstrap Is Still Not the End State

One-time global bootstrap is a practical safety fix because it avoids reinitialization churn, but it still leaves:

1. global mutable composition state
2. hidden coupling between callsites/tests
3. poor support for multiple independent engine instances

Factory-level composition solves those structurally.

## Appendix B: Future Work Backlog Candidates

1. add module introspection endpoint for debugging (`factory.DescribeModules()`)
2. add tracing around module install duration
3. add `WithModuleSet("minimal"|"default"|"all")` presets
4. generate docs table from installed module metadata
