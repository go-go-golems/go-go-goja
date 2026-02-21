---
Title: Pragmatic MVP API for module composition and runtime ownership
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-21T15:24:25.02584737-05:00
WhatFor: ""
WhenToUse: ""
---

# Pragmatic MVP API for module composition and runtime ownership

## Executive Summary

This document proposes a practical MVP for `go-go-goja` engine composition that intentionally **does not include dependency resolution yet**.

Revision: **v2** (2026-02-21) updates the proposal to a **clean API rewrite with no backward-compatibility requirement** and adds explicit analysis of runtime-scoped registration pain points.

Goals of this MVP:

1. Make module registration explicit (no hidden global side effects).
2. Make runtime ownership/concurrency safe by default.
3. Enable clean integration of runtime-scoped modules like `require("geppetto")`.
4. Remove legacy API ambiguity by cutting over to one composition model.

Out of scope for this MVP:

- topological dependency solver
- conflict graph/resolution policies beyond duplicate ID checks
- dynamic post-build module hot-add

## Problem Statement

Current `go-go-goja` behavior in `engine/factory.go`:

- creates one `require.Registry`
- calls `modules.EnableAll(reg)`
- returns VM+require module via `NewRuntime()`

This works for baseline modules, but it has two practical gaps:

1. Composition is mostly implicit/global.
2. Runtime-scoped module registration (for example Geppetto with a runtime owner runner) is awkward.

Additionally, host applications repeatedly reconstruct similar bootstrap logic:

- event loop creation
- runtime owner bridge
- module registration with per-runtime options
- require loader/global folder configuration

The MVP should centralize this without overengineering.

## Why runtime-scoped module registration is awkward today

Runtime-scoped registration means module setup needs values that exist only per runtime instance:

- `*goja.Runtime`
- event loop
- runtime owner runner
- runtime lifecycle (startup/shutdown boundaries)

Current awkwardness comes from a model mismatch:

1. **Timing mismatch**
   - Factory registers modules early (registry phase).
   - Runtime-scoped modules need to bind later (runtime phase), when VM/runner exist.

2. **Registry access mismatch**
   - Factory-owned registry is internal.
   - Callers cannot cleanly inject per-runtime registration logic into that path.

3. **One-registry vs many-runtimes mismatch**
   - One factory can create many runtimes.
   - Capturing one runtime's owner inside static registration is wrong for subsequent runtimes.

4. **Lifecycle mismatch**
   - Runtime-scoped modules need startup and teardown hooks.
   - Static registration path has no explicit runtime lifecycle callbacks.

5. **Concurrency safety mismatch**
   - goja requires single-owner access discipline.
   - Using stale/wrong owner bridge can post work to wrong or closed loop.

Conclusion: we need an API where runtime lifecycle is first-class, not bolted on.

## Proposed MVP Architecture

### Core idea

Split composition into two explicit layers:

1. **Static factory modules** (registered on factory-owned `require.Registry` before runtime creation).
2. **Runtime initializers** (run after VM creation, with access to VM/event loop/runner).

This supports both:

- plain native modules (`database`, `fs`, `exec`)
- runtime-aware modules (`geppetto` with `runtimeowner.Runner` in options)

## Clean rewrite (no backwards compatibility)

This v2 proposal intentionally removes legacy paths:

- remove `engine.New()`
- remove `engine.Open(...)`
- remove `engine.NewWithOptions(...)`
- remove implicit `modules.EnableAll(...)` bootstrapping

New required flow:

1. `engine.NewBuilder()`
2. explicit `WithRequireOptions(...)`
3. explicit `WithModules(...)`
4. `Build()`
5. `Factory.NewRuntime(...)`

## Proposed Public API (MVP)

### 1) Factory builder

```go
package engine

type FactoryBuilder struct {
    opts       []Option
    modules    []ModuleSpec
    runtimeInits []RuntimeInitializer
    built      bool
}

func NewBuilder(opts ...Option) *FactoryBuilder

func (b *FactoryBuilder) WithModules(mods ...ModuleSpec) *FactoryBuilder
func (b *FactoryBuilder) WithRuntimeInitializers(inits ...RuntimeInitializer) *FactoryBuilder
func (b *FactoryBuilder) Build() (*Factory, error)
```

### 2) Module specs (static registration)

```go
package engine

import "github.com/dop251/goja_nodejs/require"

type ModuleSpec interface {
    ID() string
    Register(reg *require.Registry) error
}

type NativeModuleSpec struct {
    ModuleName string
    Loader     require.ModuleLoader
}
```

Behavior:

- duplicate module IDs fail at `Build()`
- registration order is the user-provided order (deterministic, no sorting yet)

### 3) Runtime initializer hooks

```go
package engine

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
    "github.com/dop251/goja_nodejs/eventloop"
    "github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type RuntimeContext struct {
    VM       *goja.Runtime
    Require  *require.RequireModule
    Loop     *eventloop.EventLoop
    Owner    runtimeowner.Runner
}

type RuntimeInitializer interface {
    ID() string
    InitRuntime(ctx *RuntimeContext) error
}
```

Behavior:

- runtime initializers run in explicit order after VM + require are ready
- duplicate initializer IDs fail in `Build()`

### 4) Owned runtime object

```go
package engine

type OwnedRuntime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner
}

func (r *OwnedRuntime) Close(ctx context.Context) error
```

Factory API:

```go
func (f *Factory) NewOwnedRuntime() (*OwnedRuntime, error)
```

This gives host apps a clean, concurrency-safe runtime package.

## Lifecycle Semantics

```text
compose builder
  -> validate duplicate IDs
  -> build immutable Factory
      -> prepare require.Registry
      -> register static ModuleSpecs

Factory.NewOwnedRuntime()
  -> create VM
  -> create/start event loop
  -> create runtimeowner.Runner
  -> enable require on VM
  -> run RuntimeInitializers in order
  -> return OwnedRuntime
```

## Why this is enough for MVP

It solves immediate product needs:

- deterministic composition without globals
- reusable runtime owner concurrency contract
- clean Geppetto registration path
- no dependency solver complexity yet

It also leaves clear extension points for later:

- dependency metadata on `ModuleSpec`
- conflict key policies
- optional module sets/presets

## Concrete Code Example: Register Geppetto cleanly

### Runtime initializer for Geppetto

```go
package geppettoinit

import (
    "github.com/dop251/goja_nodejs/require"
    gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
    "github.com/go-go-golems/go-go-goja/engine"
    "github.com/rs/zerolog"
)

type Init struct {
    Logger zerolog.Logger
}

func (i Init) ID() string { return "geppetto-runtime-init" }

func (i Init) InitRuntime(ctx *engine.RuntimeContext) error {
    // Register native module against a runtime-scoped registry and enable it.
    reg := require.NewRegistry()
    gp.Register(reg, gp.Options{
        Runner: ctx.Owner,
        Logger: i.Logger,
    })
    reg.Enable(ctx.VM)
    return nil
}
```

Note:

- For MVP we can keep this simple by using an initializer-managed registry for Geppetto.
- In a refinement pass, we can expose the factory registry directly on `RuntimeContext` if needed.

### Host usage

```go
builder := engine.NewFactoryBuilder(
    engine.WithRequireOptions(
        require.WithLoader(require.DefaultSourceLoader),
        require.WithGlobalFolders("./scripts"),
    ),
)

builder = builder.WithRuntimeInitializers(
    geppettoinit.Init{Logger: logger},
)

factory, err := builder.Build()
if err != nil { return err }

rt, err := factory.NewOwnedRuntime()
if err != nil { return err }
defer rt.Close(context.Background())

_, err = rt.VM.RunScript("/abs/path/scripts/main.js", scriptSource)
if err != nil { return err }
```

## Concrete Code Example: Static module composition

```go
fsSpec := engine.NativeModuleSpec{
    ModuleName: "fs",
    Loader:     fsModule.Loader,
}
dbSpec := engine.NativeModuleSpec{
    ModuleName: "database",
    Loader:     dbModule.Loader,
}

factory, err := engine.NewFactoryBuilder().
    WithModules(fsSpec, dbSpec).
    Build()
```

Validation behavior:

- if both specs return same `ID()`, `Build()` returns explicit error.

## API sketch for backward compatibility

Not applicable in v2. This design assumes a hard cutover.

Legacy compatibility wrappers are intentionally omitted to keep one canonical runtime composition model.

## Error model (MVP)

`Build()` should fail fast on:

1. duplicate module IDs
2. duplicate runtime initializer IDs
3. module registration errors

`NewOwnedRuntime()` should fail fast on:

1. runtime initialization hook errors
2. owner/loop construction failures

No best-effort partial startup; explicit failures are easier to debug.

## Concurrency Model

goja runtime constraint:

- one runtime, one goroutine at a time.

MVP enforcement:

- `OwnedRuntime.Owner` is always created and provided.
- async/module callbacks should use owner runner (`Call`/`Post`) rather than touching VM directly from arbitrary goroutines.

This gives us a consistent safety baseline across modules.

## Migration Plan

### Phase 1: Implement new canonical API

1. Add `FactoryBuilder`, `ModuleSpec`, `RuntimeInitializer`, `OwnedRuntime`.
2. Delete old entrypoints and implicit module enable path.

### Phase 2: Adopt in first host

1. Update one app (for example extraction runner) to use `NewFactoryBuilder`.
2. Move manual runtime bootstrap code into initializers.

### Phase 3: Promote docs/examples

1. Make builder API the primary documented path.
2. Remove legacy references from docs and examples.

## Risks and Mitigations

1. Risk: Two registries accidentally diverge (factory registry vs initializer local registry).
   - Mitigation: document pattern; optionally expose shared registry on runtime context in follow-up.
2. Risk: Initializer order sensitivity.
   - Mitigation: explicit order in API docs; keep IDs for diagnostics.
3. Risk: Lifecycle leaks (event loop not stopped).
   - Mitigation: `OwnedRuntime.Close()` required; include in examples and tests.

## Implementation Checklist (no dependency solver)

- [ ] Add `FactoryBuilder` and `Build()`.
- [ ] Add `ModuleSpec` + helper `NativeModuleSpec`.
- [ ] Add `RuntimeInitializer`.
- [ ] Add `OwnedRuntime` and `Close()`.
- [ ] Add duplicate-ID validation.
- [ ] Route existing `Open/New/NewWithOptions` through new path.
- [ ] Add tests:
  - [ ] duplicate module IDs fail
  - [ ] duplicate initializer IDs fail
  - [ ] initializer order is deterministic
  - [ ] owned runtime close shuts down owner and loop
  - [ ] Geppetto initializer integration smoke test

## Open Questions (deferred to post-MVP)

1. Should `RuntimeContext` expose shared `*require.Registry` to initializers?
2. Do we want initializer conflict keys in addition to ID checks?
3. Should we add a minimal optional `Requires() []string` API before full dependency solver?

## Recommendation

Implement this v2 MVP rewrite now, explicitly skipping dependency resolution for this iteration.

This gives immediate value:

- better ergonomics
- safer runtime concurrency defaults
- cleaner Geppetto integration path

and a single, unambiguous API surface for future dependency/conflict resolution work.
