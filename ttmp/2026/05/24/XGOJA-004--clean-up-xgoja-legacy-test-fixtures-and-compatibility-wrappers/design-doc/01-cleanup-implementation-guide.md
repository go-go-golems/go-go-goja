---
Title: Cleanup implementation guide
Ticket: XGOJA-004
Status: active
Topics:
    - xgoja
    - cleanup
    - goja
    - refactor
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/testprovider/provider.go
      Note: |-
        Legacy xgoja fixture provider to remove
        Legacy xgoja fixture provider targeted for deletion
    - Path: engine/module_specs.go
      Note: |-
        Contains deprecated DefaultRegistry* exported wrappers to hard-cut over
        Contains deprecated DefaultRegistry wrappers targeted for hard cutover
    - Path: modules/common.go
      Note: |-
        Contains EnableAll compatibility helper to remove after engine wrappers are gone
        Contains EnableAll compatibility helper
    - Path: pkg/jsverbs/runtime.go
      Note: |-
        Contains obsolete lightweight InvokeInGojaRuntime path after xgoja moved to engine.Runtime
        Contains obsolete lightweight InvokeInGojaRuntime path
    - Path: pkg/xgoja/testprovider/provider.go
      Note: |-
        Current public fixture provider used by generated binaries and examples
        Current generated-binary fixture provider that replaces the internal fixture
ExternalSources: []
Summary: Hard-cutover cleanup plan for xgoja legacy fixtures, tracked .orig files, obsolete jsverbs lightweight invocation, and deprecated engine default-registry wrappers.
LastUpdated: 2026-05-24T12:26:46.978242753-04:00
WhatFor: Guide the XGOJA-004 cleanup work in focused, reviewable commits.
WhenToUse: Use when removing legacy compatibility code after the xgoja engine-runtime cutover.
---


# Cleanup implementation guide

## Executive Summary

This ticket is a hard-cutover cleanup after the xgoja runtime and jsverbs work. The repository now has a single engine-backed xgoja runtime path, a public generated-binary fixture provider, and module-selection middleware that supersedes older DefaultRegistry helper functions. Several transitional files and APIs remain only because they were useful while the implementation was moving.

The cleanup removes those transitional pieces instead of preserving backwards-compatibility wrappers:

1. delete the old internal xgoja fixture provider and tracked `.orig` files,
2. remove the obsolete `jsverbs.Registry.InvokeInGojaRuntime` path and its direct test,
3. remove exported deprecated `engine.DefaultRegistry*` helpers,
4. remove `modules.EnableAll` if no active code still needs it,
5. update docs/tests to use `UseModuleMiddleware(...)` and `InvokeInRuntime(...)`.

## Problem Statement

The codebase currently contains several redundant or compatibility-shaped artifacts:

- `cmd/xgoja/internal/testprovider` duplicates the current fixture provider in `pkg/xgoja/testprovider`, but generated programs cannot import `cmd/xgoja/internal/...` reliably.
- Tracked `.orig` files (`engine/config.go.orig`, `engine/runtime.go.orig`, `modules/exports.go.orig`) are stale editor/merge artifacts; one contains conflict markers.
- `pkg/jsverbs.InvokeInGojaRuntime` exists for the pre-XGOJA-003 lightweight xgoja runtime. xgoja now creates `engine.Runtime` and calls `InvokeInRuntime`.
- `engine.DefaultRegistryModules`, `engine.DefaultRegistryModule`, `engine.DefaultRegistryModulesNamed`, and `engine.DataOnlyDefaultRegistryModules` are explicitly deprecated wrappers around default-registry module specs. Current code should use `UseModuleMiddleware`.
- `modules.EnableAll` only supports the deprecated all-default wrapper and can disappear once that wrapper is removed.

Keeping these around makes future readers ask which path is canonical and increases the chance that new code copies old examples.

## Proposed Solution

Perform a hard cutover. Do not add compatibility aliases. If a package has no active caller after the current engine-runtime migration, remove it and update docs/tests.

### Step 1: Remove dead fixture and artifact files

Delete:

```text
cmd/xgoja/internal/testprovider/
engine/config.go.orig
engine/runtime.go.orig
modules/exports.go.orig
```

Validation:

```bash
GOWORK=off go test ./cmd/xgoja ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./pkg/xgoja/testprovider -count=1
```

### Step 2: Remove obsolete jsverbs lightweight invocation

Delete from `pkg/jsverbs/runtime.go`:

- `InvokeInGojaRuntime`,
- private `invokeInGojaRuntime`,
- `waitForPromiseWithOwner`,
- `waitForPromiseDirect`, if unused after removing the public function.

Delete:

```text
pkg/jsverbs/runtime_direct_test.go
```

Update docs that still mention the old path so callers use:

```go
factory, err := engine.NewBuilder().
    WithRequireOptions(require.WithLoader(registry.RequireLoader())).
    Build()
rt, err := factory.NewRuntime(ctx)
result, err := registry.InvokeInRuntime(ctx, rt, verb, values)
```

Validation:

```bash
GOWORK=off go test ./pkg/jsverbs ./pkg/jsverbscli ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
```

### Step 3: Remove deprecated engine DefaultRegistry wrappers

Remove exported functions:

```go
DefaultRegistryModules()
DefaultRegistryModule(name string)
DefaultRegistryModulesNamed(names ...string)
DataOnlyDefaultRegistryModules()
```

Keep the internal machinery needed by middleware and runtime construction by replacing public helper usage with private helpers, for example:

```go
func defaultRegistryModule(name string) RuntimeModuleSpec
func defaultRegistryModulesNamed(names ...string) RuntimeModuleSpec
func dataOnlyDefaultRegistryModules() RuntimeModuleSpec
```

Update `engine/factory.go`:

```go
modules_ = append(modules_, defaultRegistryModule(name))
...
if err := dataOnlyDefaultRegistryModules().RegisterRuntimeModule(moduleCtx, reg); err != nil { ... }
```

Update tests that directly exercised deprecated helpers to instead verify middleware usage:

```go
engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("fs")).Build()
engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("fs", "os")).Build()
```

### Step 4: Remove `modules.EnableAll`

After `DefaultRegistryModules()` is gone, remove:

```go
func EnableAll(reg *require.Registry)
```

from `modules/common.go`, unless an active caller remains.

### Step 5: Update docs and run full validation

Update public docs to remove recommendations for deprecated wrappers. Prefer these patterns:

```go
// All default modules: plain builder.
factory, err := engine.NewBuilder().Build()

// Safe data-only modules.
factory, err := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()

// Specific default modules.
factory, err := engine.NewBuilder().UseModuleMiddleware(engine.MiddlewareOnly("fs", "os")).Build()

// Custom/native modules.
factory, err := engine.NewBuilder().WithModules(engine.NativeModuleSpec{...}).Build()
```

Full focused validation:

```bash
GOWORK=off go test \
  ./engine \
  ./pkg/runtimebridge \
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
  ./pkg/jsverbscli \
  ./pkg/gojahttp \
  ./pkg/doc \
  -count=1
```

Examples:

```bash
for dir in runtime-filesystem embedded-jsverbs provider-shipped-jsverbs; do
  make -C examples/xgoja/$dir smoke
done
```

## Design Decisions

### Hard cutover, no shims

The user explicitly requested hard cutover behavior. This ticket should remove compatibility wrappers rather than re-exporting aliases or adding transitional methods.

### Keep middleware as the module-selection API

`UseModuleMiddleware` is the canonical way to select default registry modules. It is expressive enough for all current uses:

- all modules: no middleware on a plain builder,
- safe modules: `MiddlewareSafe`,
- only named modules: `MiddlewareOnly`,
- additive/exclusion transforms: middleware composition.

### Keep `engine.NativeModuleSpec`

`NativeModuleSpec` is not deprecated. It is the explicit adapter for one native `require()` module loader and remains useful for tests, configured database modules, and host-provided modules.

### Keep `jsverbs.InvokeInRuntime`

`InvokeInRuntime` is the canonical caller-owned runtime path because it accepts `*engine.Runtime` and therefore uses owner scheduling, runtimebridge, lifecycle context, and close semantics.

## Alternatives Considered

### Leave deprecated wrappers but update docs

Rejected. The ticket is explicitly cleanup-focused and the user requested hard cutover.

### Move `cmd/xgoja/internal/testprovider` to another fixture package

Rejected. `pkg/xgoja/testprovider` already exists and has the richer fixture behavior needed by tests and examples.

### Keep `InvokeInGojaRuntime` for bare Goja callers

Rejected. The current runtime model intentionally discourages managed hosts from constructing bare Goja runtimes when owner scheduling and runtimebridge bindings matter. Bare callers can still write their own tiny invocation loop if they truly need one, but the exported jsverbs host API should point at `engine.Runtime`.

## Implementation Plan

1. Commit low-risk deletion of dead files and legacy fixture.
2. Commit jsverbs lightweight invocation removal and docs/test updates.
3. Commit engine default-registry wrapper removal and docs/test updates.
4. Commit any final docs/changelog/task cleanup.
5. Run focused validation and example smokes after each code-changing step.

## Open Questions

None. The requested policy is hard cutover.

## References

- XGOJA-002 diary: jsverbs mounting introduced `InvokeInGojaRuntime` as a transitional path.
- XGOJA-003 diary: xgoja moved to `engine.Runtime` and `InvokeInRuntime`.
- `pkg/xgoja/testprovider`: current generated-binary fixture provider.
