---
Title: Runtime-Aware Module Registration Design and Implementation Guide
Ticket: XGOJA-003
Status: active
Topics:
    - xgoja
    - goja
    - engine
    - runtime
    - modules
    - refactor
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Factory builder/runtime registration flow to refactor
    - Path: engine/module_specs.go
      Note: Current split static module API and target runtime-aware API
    - Path: engine/runtime_modules.go
      Note: Current runtime registrar API to be removed
    - Path: pkg/xgoja/app/factory.go
      Note: xgoja runtime factory target adaptation
ExternalSources: []
Summary: Design and migration guide for hard-cutting the engine module API to one runtime-aware RegisterRuntimeModule contract and adapting xgoja to create engine runtimes from xgoja specs safely.
LastUpdated: 2026-05-23T22:05:00-04:00
WhatFor: Use this guide to implement and review XGOJA-003 without preserving legacy ModuleSpec or RuntimeModuleRegistrar wrappers.
WhenToUse: Read before changing engine module registration, engine.Factory, runtime-scoped modules, jsverbs runtime construction, or xgoja app runtime creation.
---


# Runtime-Aware Module Registration Design and Implementation Guide

## Executive Summary

This ticket replaces the current split between static engine modules and runtime-scoped module registrars with one runtime-aware module registration contract. The new contract is intentionally a hard cutover: no compatibility wrappers, no deprecated aliases, and no parallel legacy API.

The target API is:

```go
type RuntimeModuleSpec interface {
    ID() string
    RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

Every module registration now receives runtime context. That context includes the VM, event loop, runtime owner, lifecycle context, closer registration, and runtime value map. This makes runtime context first-class instead of splitting modules into two categories: modules that only receive a `require.Registry`, and registrars that receive full runtime context.

The important downstream goal is xgoja. xgoja currently has a separate lightweight runtime factory because provider modules need runtime context and the existing `engine.ModuleSpec` interface cannot pass it. After this refactor, xgoja can translate spec-selected provider modules into engine `RuntimeModuleSpec` values and create a real `engine.Runtime` without accidentally enabling default engine modules.

## Problem Statement

The engine currently has two registration concepts:

```go
type ModuleSpec interface {
    ID() string
    Register(reg *require.Registry) error
}

type RuntimeModuleRegistrar interface {
    ID() string
    RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

This split creates several problems.

First, static modules cannot receive runtime context. That is acceptable for trivial loaders, but it is not enough for modules that need lifecycle context, owner scheduling, closer registration, or runtime values. The result is pressure to invent a second path whenever a module needs runtime-aware behavior.

Second, xgoja provider modules already have a runtime-aware factory shape:

```go
type ModuleFactory func(providerapi.ModuleContext) (require.ModuleLoader, error)
```

The provider factory receives config and context before returning a loader. That shape maps naturally to runtime-aware engine module registration, but it does not map cleanly to the old static `ModuleSpec.Register(reg)` contract.

Third, the split makes the builder API harder to reason about. `WithModules` and `WithRuntimeModuleRegistrars` both install modules into the same `require.Registry`, but they do so through different extension points. The difference is not visible from the final JavaScript API; both produce `require()` modules.

## Proposed Solution

Use one module registration interface:

```go
type RuntimeModuleSpec interface {
    ID() string
    RegisterRuntimeModule(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

Then update `FactoryBuilder` to store only runtime-aware module specs:

```go
type FactoryBuilder struct {
    modules []RuntimeModuleSpec
    moduleMiddlewares []ModuleMiddleware
    runtimeInitializers []RuntimeInitializer
}
```

`WithModules` remains the builder method for adding modules, but its parameter type changes:

```go
func (b *FactoryBuilder) WithModules(mods ...RuntimeModuleSpec) *FactoryBuilder
```

`WithRuntimeModuleRegistrars` is removed. Former registrars become normal runtime module specs and are passed to `WithModules`.

At runtime creation time, the engine creates the `RuntimeModuleContext` before registering modules, then calls `RegisterRuntimeModule` for every selected module:

```go
moduleCtx := &RuntimeModuleContext{
    Context: runtimeCtx,
    VM: vm,
    Loop: loop,
    Owner: owner,
    AddCloser: rt.AddCloser,
    Values: runtimeValues,
}

for _, mod := range f.modules {
    if err := mod.RegisterRuntimeModule(moduleCtx, reg); err != nil {
        _ = rt.Close(ctx)
        return nil, fmt.Errorf("register module %q: %w", mod.ID(), err)
    }
}
```

## Why This Is a Hard Cutover

The user explicitly requested no backwards compatibility or legacy wrappers. That means this ticket should not keep `ModuleSpec`, `RuntimeModuleRegistrar`, `WithRuntimeModuleRegistrars`, or adapter wrappers around old `Register` methods.

The migration should update all in-repo call sites directly:

```go
// Before
engine.NewBuilder().WithRuntimeModuleRegistrars(express.NewRegistrar(host))

// After
engine.NewBuilder().WithModules(express.NewRegistrar(host))
```

```go
// Before
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error

// After
func (r *Registrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error
```

The API becomes simpler because there is one mental model: if something registers modules into a runtime, it is a `RuntimeModuleSpec`.

## Design Decisions

### Decision 1: Keep `WithModules` as the public builder method

The builder method name remains `WithModules` because that is the user-facing concept: add modules to the runtime. The type changes to runtime-aware modules.

This keeps builder code readable:

```go
factory, err := engine.NewBuilder().
    WithModules(express.NewRegistrar(host), uidsl.NewRegistrar()).
    Build()
```

### Decision 2: Remove `WithRuntimeModuleRegistrars`

Keeping both methods would preserve the old split. This ticket removes it so new code cannot choose the wrong extension point.

### Decision 3: Keep `RuntimeInitializer` separate

A runtime initializer is not necessarily a module registration. It runs after `require` is enabled and can set globals or inspect installed modules. Keeping it separate preserves the existing lifecycle ordering:

1. create runtime substrate;
2. register modules into `require.Registry`;
3. enable `require`;
4. install global services;
5. run runtime initializers.

### Decision 4: xgoja should use `engine.Runtime` safely

After the engine module contract is runtime-aware, xgoja can create engine runtimes without losing its explicit module-selection semantics. The xgoja runtime factory should build an engine factory with explicit modules only. Because `engine.FactoryBuilder.Build` preserves the historical default-registry fallback for normal engine callers, xgoja disables both implicit default-registry selection and automatic data-only default modules when constructing its internal engine builder.

The xgoja adapter module should look like this:

```go
type providerRuntimeModuleSpec struct {
    instance app.ModuleInstance
    module providerapi.Module
}

func (s providerRuntimeModuleSpec) ID() string {
    return s.instance.Package + "." + s.instance.Name + " as " + s.instance.Alias()
}

func (s providerRuntimeModuleSpec) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
    config, err := json.Marshal(s.instance.Config)
    if err != nil {
        return err
    }
    loader, err := s.module.New(providerapi.ModuleContext{
        Context: ctx.Context,
        Name: s.instance.Name,
        As: s.instance.Alias(),
        Config: config,
    })
    if err != nil {
        return err
    }
    reg.RegisterNativeModule(s.instance.Alias(), loader)
    return nil
}
```

## Migration Guide

### Static native modules

Before:

```go
type NativeModuleSpec struct { ... }
func (s NativeModuleSpec) Register(reg *require.Registry) error { ... }
```

After:

```go
type NativeModuleSpec struct { ... }
func (s NativeModuleSpec) RegisterRuntimeModule(_ *RuntimeModuleContext, reg *require.Registry) error { ... }
```

### Default registry modules

Before:

```go
engine.DefaultRegistryModule("fs")
```

After: same constructor name can remain, but it returns `RuntimeModuleSpec` and implements `RegisterRuntimeModule`.

### Runtime module registrars

Before:

```go
func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error
factory, err := engine.NewBuilder().WithRuntimeModuleRegistrars(r).Build()
```

After:

```go
func (r *Registrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error
factory, err := engine.NewBuilder().WithModules(r).Build()
```

### Runtime evaluator configuration

Before:

```go
RuntimeRegistrars []engine.RuntimeModuleRegistrar
builder = builder.WithRuntimeModuleRegistrars(config.RuntimeRegistrars...)
```

After:

```go
RuntimeModules []engine.RuntimeModuleSpec
builder = builder.WithModules(config.RuntimeModules...)
```

### xgoja runtime factory

Before: xgoja created its own runtime with `goja.New`, event loop, owner, runtimebridge, and a `require.Registry`.

After: xgoja translates selected provider modules into engine runtime modules, constructs the engine builder with `WithImplicitDefaultRegistryModules(false)` and `WithDataOnlyDefaultRegistryModules(false)`, and calls `engine.Factory.NewRuntime`.

## Implementation Plan

1. Replace `engine.ModuleSpec` and `engine.RuntimeModuleRegistrar` with one `RuntimeModuleSpec` interface.
2. Update engine built-in module specs to implement `RegisterRuntimeModule`.
3. Update `FactoryBuilder`, `Factory`, and `NewRuntime` to store and register `RuntimeModuleSpec` values.
4. Remove `WithRuntimeModuleRegistrars` and update all call sites to use `WithModules`.
5. Rename all `RegisterRuntimeModules` methods to `RegisterRuntimeModule`.
6. Update xgoja `RuntimeFactory` to use `engine.Runtime` with spec-selected provider module adapters.
7. Update jsverbs app code to use `engine.Runtime` through `InvokeInRuntime` where possible.
8. Run focused validation with `GOWORK=off`.
9. Update docs/help examples if public API names changed.

## Validation Plan

Focused validation command:

```bash
GOWORK=off go test \
  ./engine \
  ./pkg/jsverbs \
  ./pkg/xgoja/app \
  ./cmd/xgoja/internal/generate \
  ./cmd/xgoja \
  ./cmd/xgoja/internal/buildspec \
  ./pkg/xgoja/providerapi \
  ./pkg/xgoja/testprovider \
  ./pkg/xgoja/testcobra \
  ./pkg/xgoja/testadapter \
  ./modules/express \
  ./modules/uidsl \
  ./pkg/hashiplugin/host \
  ./pkg/repl/evaluators/javascript \
  ./pkg/docaccess/runtime \
  -count=1
```

Example smoke command:

```bash
for dir in runtime-filesystem embedded-jsverbs provider-shipped-jsverbs; do
  make -C examples/xgoja/$dir smoke
 done
```

## Risks

The main risk is large API churn. This is expected because the ticket is a hard cutover. The mitigation is to keep the new API smaller than the old API and update all in-repo call sites in one focused pass.

The second risk is accidentally changing default module exposure. The engine's default behavior should remain the same for a plain `engine.NewBuilder().Build()`: it should still expose default registry modules. xgoja must avoid that behavior by passing explicit modules generated from the xgoja runtime profile.

The third risk is lifecycle ordering. Runtime-aware modules now run at the same point where both old static modules and old runtime registrars used to run. The engine must construct `RuntimeModuleContext` before module registration and enable `require` after module registration.

## Review Instructions

Review in this order:

1. `engine/module_specs.go` for the new interface and built-in module conversions.
2. `engine/factory.go` for builder storage, validation, and runtime registration ordering.
3. Former registrar packages such as `modules/express`, `modules/uidsl`, `pkg/hashiplugin/host`, and `pkg/docaccess/runtime`.
4. `pkg/xgoja/app/factory.go` to verify xgoja now uses engine runtime safely and only with spec-selected modules.
5. `pkg/xgoja/app/root.go` and `pkg/jsverbs/runtime.go` to verify jsverb invocation uses the engine runtime path where possible.
6. Tests and examples.
