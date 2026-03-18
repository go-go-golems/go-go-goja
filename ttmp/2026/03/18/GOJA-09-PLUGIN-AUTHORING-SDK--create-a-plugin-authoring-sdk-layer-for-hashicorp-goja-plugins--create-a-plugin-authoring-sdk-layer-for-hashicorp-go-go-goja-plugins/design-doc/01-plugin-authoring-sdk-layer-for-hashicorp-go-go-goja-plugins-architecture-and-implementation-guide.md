---
Title: Plugin authoring SDK layer for HashiCorp go-go-goja plugins architecture and implementation guide
Ticket: GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - architecture
    - tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: |-
        Factory runtime creation order explains where plugin registration happens
        Factory runtime creation order that hosts plugins today
    - Path: engine/runtime.go
      Note: |-
        Runtime cleanup hooks explain plugin lifecycle ownership
        Runtime cleanup ownership that plugin processes depend on
    - Path: engine/runtime_modules.go
      Note: |-
        Runtime-scoped module seam that plugin loading already uses and the SDK must preserve
        Runtime-scoped module seam the SDK must preserve
    - Path: modules/common.go
      Note: |-
        Existing native-module authoring model provides the closest in-repo comparison point
        Built-in native-module authoring baseline for SDK ergonomics
    - Path: modules/exports.go
      Note: |-
        Existing export helper shows the style used by built-in native modules
        Existing export helper style used by built-in modules
    - Path: pkg/doc/02-creating-modules.md
      Note: |-
        Existing native-module tutorial is the baseline authoring experience the SDK should approach
        Native-module tutorial used as the usability comparison point
    - Path: pkg/doc/13-plugin-developer-guide.md
      Note: |-
        Current plugin implementation guide documents the runtime and transport layers the SDK must fit into
        Current plugin architecture guide that the SDK design builds on
    - Path: pkg/hashiplugin/contract/contract.go
      Note: |-
        Shared plugin interface that the SDK should implement rather than replace
        Low-level plugin interface the SDK should implement
    - Path: pkg/hashiplugin/contract/jsmodule.proto
      Note: |-
        RPC manifest and invoke schema that defines the author-facing capability boundary
        Manifest and invoke schema that defines the supported plugin surface
    - Path: pkg/hashiplugin/host/reify.go
      Note: |-
        Host reification shows which export shapes are actually supported today
        Host reification confirms only function and object-method exports exist today
    - Path: pkg/hashiplugin/host/validate.go
      Note: |-
        Manifest validation rules define the invariants the SDK should make hard to violate
        Host validation rules the SDK should align with
    - Path: pkg/hashiplugin/shared/plugin.go
      Note: |-
        Shared transport and serve wiring that current plugin authors must wire manually
        Shared serve and dispense wiring the SDK should hide
    - Path: plugins/examples/greeter/main.go
      Note: |-
        Current user-facing example shows the exact boilerplate the SDK should collapse
        Current example plugin shows author-facing duplication to remove
    - Path: plugins/testplugin/echo/main.go
      Note: |-
        Current test plugin confirms the same boilerplate is duplicated in fixtures
        Test plugin confirms the same duplication in fixtures
ExternalSources: []
Summary: Detailed repo-grounded design for adding an author-facing SDK layer that hides plugin manifest, dispatch, and serve boilerplate while preserving the current host-owned runtime and transport contracts.
LastUpdated: 2026-03-18T16:20:00-04:00
WhatFor: Give a new engineer enough context to design and implement a plugin authoring SDK without breaking the existing host/runtime plugin model.
WhenToUse: Use when implementing, reviewing, or extending the author-facing SDK for HashiCorp go-go-goja plugins.
---


# Plugin authoring SDK layer for HashiCorp go-go-goja plugins architecture and implementation guide

## Executive Summary

`go-go-goja` now has working HashiCorp plugin support and an initial author-facing SDK in `pkg/hashiplugin/sdk`, but the architectural reasons for that package are still worth documenting explicitly. Before the SDK existed, plugin authors wrote protobuf-shaped manifest data by hand, manually dispatched `InvokeRequest` values with stringly-typed switches, manually converted return values to `structpb.Value`, and manually wired `plugin.Serve(...)` with the shared handshake and versioned plugin sets. The current SDK-backed examples in `plugins/examples/greeter/main.go` and `plugins/testplugin/echo/main.go` show the intended end state; this document explains why that API shape is the right fit for the existing runtime.

The recommended next step is an author-facing SDK package that sits above `pkg/hashiplugin/contract` and `pkg/hashiplugin/shared`, but below `pkg/hashiplugin/host`. The host/runtime architecture should stay exactly as it is: the host owns `goja.Runtime`, runtime lifecycle, and CommonJS module registration; plugin subprocesses still expose `Manifest(...)` plus `Invoke(...)` over `go-plugin` gRPC. What changes is how plugin binaries are authored. Instead of hand-writing transport-oriented code, authors should declare exports through a small Go API and let the SDK build the manifest, dispatch calls, convert arguments/results, and serve the plugin.

The goal is not to invent a new plugin model. The goal is to make the existing one easy to author correctly.

## Problem Statement And Scope

### The problem in one sentence

The current plugin runtime integration is usable for hosts, but plugin authors still work at the transport-contract level instead of an authoring-contract level.

### Why this matters

An intern reading the repo today will find a mismatch:

- Native built-in modules have a reasonably direct authoring story through `modules.NativeModule` and `modules.SetExport(...)` in `modules/common.go:29-33` and `modules/exports.go:7-12`.
- Plugin-backed modules do not have an equivalent convenience layer. They implement `contract.JSModule` directly in `pkg/hashiplugin/contract/contract.go:5-10`, build `contract.ModuleManifest` structs by hand, and route every export through a single manual `Invoke(...)` switch.

That mismatch creates real costs:

- More boilerplate per plugin.
- More chances to drift between manifest and implementation.
- More chances to violate host validation rules in `pkg/hashiplugin/host/validate.go:10-76`.
- Harder tutorials, because authors learn transport details before they learn the conceptual model.

### In scope

This ticket covers the design of an author-facing SDK layer for plugins, including:

1. the package shape,
2. the public authoring API,
3. the internal dispatcher model,
4. migration of current examples and tests,
5. testing strategy,
6. documentation impacts.

### Out of scope

This ticket does not propose:

1. changing host-owned runtime composition,
2. changing the `go-plugin` transport choice,
3. adding streaming, callbacks, or async cross-process semantics,
4. replacing the manifest schema with code generation,
5. removing the existing low-level contract package.

## Read This First

If you are new to the codebase, read these files in this order before implementing the SDK:

1. `pkg/hashiplugin/contract/contract.go`
2. `pkg/hashiplugin/contract/jsmodule.proto`
3. `pkg/hashiplugin/shared/plugin.go`
4. `pkg/hashiplugin/host/validate.go`
5. `pkg/hashiplugin/host/reify.go`
6. `pkg/hashiplugin/host/client.go`
7. `pkg/hashiplugin/host/registrar.go`
8. `engine/runtime_modules.go`
9. `engine/factory.go`
10. `engine/runtime.go`
11. `plugins/examples/greeter/main.go`
12. `plugins/testplugin/echo/main.go`
13. `pkg/doc/02-creating-modules.md`
14. `pkg/doc/13-plugin-developer-guide.md`

Why this order matters:

- The contract explains what a plugin binary must implement.
- The host explains what shapes the runtime actually accepts.
- The engine explains where plugin modules are attached and cleaned up.
- The examples show the exact authoring friction the SDK should remove.
- The native-module docs show the usability bar plugin authoring should move toward.

## System Orientation

### The host still owns the JavaScript runtime

The most important architectural fact is unchanged from GOJA-08: the host process owns `goja.Runtime`, the runtime event loop, `require.Registry`, and runtime cleanup. That is explicit in:

- `engine/runtime_modules.go:12-25`
- `engine/factory.go:151-213`
- `engine/runtime.go:23-86`

Plugins do not own a VM. They do not receive `goja.Value` pointers. They only provide:

- a manifest describing one module,
- an invoke handler for export calls,
- plain protobuf-compatible data at the transport boundary.

### The runtime seam already exists

The engine side is already in good shape for host loading:

```text
entrypoint
  -> engine.NewBuilder()
  -> WithRuntimeModuleRegistrars(host.NewRegistrar(...))
  -> Factory.NewRuntime(ctx)
  -> registrar discovers and loads plugins
  -> host reifies each plugin as a CommonJS native module
  -> JS calls require("plugin:...")
```

Evidence:

- runtime registrar interface: `engine/runtime_modules.go:12-17`
- factory runtime order: `engine/factory.go:176-213`
- cleanup hook registration: `engine/runtime.go:36-54`

This means the SDK should not touch engine code. It should make it easier to produce plugin binaries that fit this runtime path.

### The transport contract is intentionally narrow

The shared contract in `pkg/hashiplugin/contract/jsmodule.proto:10-44` exposes exactly two RPCs:

1. `GetManifest(...)`
2. `Invoke(...)`

The handwritten Go interface mirrors that in `pkg/hashiplugin/contract/contract.go:5-10`.

The current manifest model supports only:

- module name,
- version,
- export list,
- capabilities,
- doc string.

The current invoke model supports only:

- one export name,
- one method name for object exports,
- a list of protobuf `Value` arguments,
- one protobuf `Value` result.

This is a narrow but healthy boundary. The SDK should embrace it, not hide it so thoroughly that authors stop understanding what the host can actually load.

## Current-State Analysis

## What plugin authoring looked like before the SDK

Before `pkg/hashiplugin/sdk` was added, both plugin binaries followed the same repetitive pattern.

### Example: `plugins/examples/greeter/main.go`

Observed structure in the pre-SDK form of `plugins/examples/greeter/main.go`:

1. Define a zero-value module type.
2. Implement `Manifest(context.Context)`.
3. Hand-construct `contract.ModuleManifest`.
4. Hand-construct `[]*contract.ExportSpec`.
5. Implement `Invoke(context.Context, *contract.InvokeRequest)`.
6. Switch on `req.GetExportName()`.
7. For object exports, switch again on `req.GetMethodName()`.
8. Manually convert results with `structpb.NewValue(...)`.
9. Manually wire `plugin.Serve(...)` using `shared.Handshake` and `shared.VersionedServerPluginSets(...)`.

### Example: `plugins/testplugin/echo/main.go`

Observed structure in the pre-SDK form of `plugins/testplugin/echo/main.go`:

1. Same imports.
2. Same module type shape.
3. Same manifest-building code pattern.
4. Same `Invoke(...)` switch pattern.
5. Same `plugin.Serve(...)` block.

That duplicated logic is what the SDK was designed to replace.

## What the host actually validates

`pkg/hashiplugin/host/validate.go:10-76` shows the invariants authors must satisfy:

- module name must be non-empty,
- module name must start with the configured namespace (`plugin:` by default),
- allowlist may reject the module,
- export names must be unique,
- function exports may not define methods,
- object exports must define at least one method,
- object methods must be unique and non-empty.

Important consequence:

The SDK should make these invalid shapes hard to express. If the SDK still lets authors casually build invalid manifests, it has not solved the real problem.

## What the host actually reifies

`pkg/hashiplugin/host/reify.go:14-86` shows the host-side runtime behavior:

- each manifest becomes one CommonJS native module,
- function exports become callable Go closures,
- object exports become Goja objects whose methods call back into `Invoke(...)`,
- JS arguments are converted through `arg.Export()` then `structpb.NewValue(...)`,
- RPC results come back as protobuf values and are turned into `vm.ToValue(...)`.

Important consequence:

The host currently supports only two author-visible export shapes:

1. top-level functions,
2. top-level objects with methods.

The SDK should model those shapes directly instead of pretending arbitrary nested structures are supported.

## Comparison with built-in native modules

The closest in-repo authoring comparison is the native module system:

- interface: `modules/common.go:29-33`
- export helper: `modules/exports.go:7-12`
- tutorial: `pkg/doc/02-creating-modules.md:18-56`

Built-in module authors write code like:

```go
type m struct{}

func (m) Name() string { return "example" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    modules.SetExport(exports, "example", "hello", func(name string) string {
        return "Hello, " + name
    })
}
```

That is not zero-boilerplate, but it is conceptually direct. Plugin authoring is currently farther from this baseline than it needs to be.

## Gap Analysis

The current plugin runtime subsystem is complete enough to load plugins, but incomplete as an authoring story.

### Gap 1: Transport details leak into every plugin binary

Evidence:

- `shared.Handshake` and `shared.VersionedServerPluginSets(...)` are referenced directly in `plugins/examples/greeter/main.go:95-99` and `plugins/testplugin/echo/main.go:69-74`.

Problem:

- every plugin binary has to know how to boot the transport layer.

Desired SDK effect:

- authors call one `sdk.Serve(...)` function instead.

### Gap 2: Manifest construction is manual and repetitive

Evidence:

- both example plugins hand-construct `contract.ModuleManifest` and `ExportSpec` trees.

Problem:

- authors can accidentally declare one shape and implement another.

Desired SDK effect:

- manifest generation should be derived from a higher-level export declaration.

### Gap 3: Dispatch is stringly typed

Evidence:

- `Invoke(...)` in `plugins/examples/greeter/main.go:40-60` and `plugins/testplugin/echo/main.go:38-67` uses string switches over export and method names.

Problem:

- authors repeat branch logic the SDK could centralize,
- missing branches are detected only at runtime.

Desired SDK effect:

- the SDK should route based on a registered handler table.

### Gap 4: Value conversion is spread across plugins

Evidence:

- `structpb.NewValue(...)` appears directly in example plugin code.

Problem:

- authors have to learn protobuf-value conversion immediately,
- result-shape errors become repetitive and inconsistent.

Desired SDK effect:

- handlers should return plain Go values and let the SDK convert them.

### Gap 5: Documentation teaches the low-level surface first

Evidence:

- the current tutorial includes the full low-level plugin implementation in `pkg/doc/14-plugin-tutorial-build-install.md:155-204`.

Problem:

- the tutorial is correct, but it teaches transport scaffolding before it teaches the conceptual model.

Desired SDK effect:

- tutorials should teach a compact, purpose-built plugin DSL first and explain the lower-level contract second.

## Design Goals

The SDK should be designed against explicit goals.

### Primary goals

1. Preserve the existing host/runtime architecture.
2. Remove boilerplate from plugin binaries.
3. Make valid manifest shapes the default.
4. Keep the supported export model explicit and small.
5. Be easy enough for an intern to read and extend.

### Non-goals

1. Reflection-heavy automatic binding in v1.
2. Hiding every transport detail from advanced users.
3. Supporting arbitrary nested JS object graphs.
4. Building a code generator before the manual API has stabilized.

## Proposed Architecture

## Package layout

The recommended package is:

```text
pkg/hashiplugin/sdk
```

Why `sdk`:

- short and discoverable,
- author-facing,
- clearly separate from `contract`, `shared`, and `host`,
- does not imply host-side runtime ownership.

Recommended internal files:

```text
pkg/hashiplugin/sdk/
  module.go        // Module builder and immutable manifest/export model
  export.go        // Function/object export declarations
  call.go          // Author-facing call context and argument helpers
  convert.go       // Go <-> structpb.Value conversion helpers
  dispatch.go      // Invoke routing implementation
  serve.go         // sdk.Serve(...) wrapper around shared/plugin.go
  errors.go        // validation and argument errors
  sdk_test.go      // unit tests for manifest/dispatch/conversion
```

The important layering should look like this:

```text
plugin author code
    |
    v
pkg/hashiplugin/sdk
    |
    v
pkg/hashiplugin/contract + pkg/hashiplugin/shared
    |
    v
go-plugin gRPC transport
    |
    v
pkg/hashiplugin/host
```

The SDK should depend on `contract` and `shared`. The host should not depend on the SDK.

## Public API shape

The author-facing API should be declarative first and low-level second.

### Recommended top-level API

```go
package sdk

type Handler func(context.Context, *Call) (any, error)

type Module struct { ... } // implements contract.JSModule

func NewModule(name string, opts ...ModuleOption) (*Module, error)
func MustModule(name string, opts ...ModuleOption) *Module

func Version(v string) ModuleOption
func Doc(doc string) ModuleOption
func Capabilities(values ...string) ModuleOption
func Function(name string, fn Handler, opts ...ExportOption) ModuleOption
func Object(name string, opts ...ObjectOption) ModuleOption

func ExportDoc(doc string) ExportOption
func Method(name string, fn Handler, opts ...ExportOption) ObjectOption

func Serve(mod contract.JSModule)
func ServeModule(name string, opts ...ModuleOption)
```

### Recommended authoring example

The current `greeter` example should become conceptually similar to:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
    sdk.ServeModule(
        "plugin:greeter",
        sdk.Version("v1"),
        sdk.Doc("Example plugin for greeter-style functions"),
        sdk.Function("greet", func(_ context.Context, call *sdk.Call) (any, error) {
            name := call.StringDefault(0, "world")
            return fmt.Sprintf("hello, %s", name), nil
        }),
        sdk.Object("strings",
            sdk.Method("upper", func(_ context.Context, call *sdk.Call) (any, error) {
                return strings.ToUpper(call.StringDefault(0, "")), nil
            }),
            sdk.Method("lower", func(_ context.Context, call *sdk.Call) (any, error) {
                return strings.ToLower(call.StringDefault(0, "")), nil
            }),
        ),
        sdk.Object("meta",
            sdk.Method("pid", func(_ context.Context, call *sdk.Call) (any, error) {
                return os.Getpid(), nil
            }),
        ),
    )
}
```

This keeps the important domain logic visible while collapsing transport boilerplate.

## Core SDK types

### `sdk.Module`

Responsibilities:

- hold immutable metadata,
- hold the author-registered exports,
- implement `contract.JSModule`,
- generate a valid manifest from the authoring declarations,
- route invoke requests to handlers.

Why this matters:

- the host should continue to see a normal `contract.JSModule`,
- the SDK should be swappable with manual implementations,
- advanced users should still be able to bypass the SDK if needed.

### `sdk.Call`

Recommended responsibilities:

- expose `ExportName` and `MethodName`,
- expose raw protobuf args for escape hatches,
- expose converted Go values for common cases,
- provide helper methods for typed reads with good errors.

Suggested API:

```go
type Call struct {
    ExportName string
    MethodName string
    Args       []any
    RawArgs    []*structpb.Value
}

func (c *Call) Len() int
func (c *Call) Value(i int) (any, error)
func (c *Call) String(i int) (string, error)
func (c *Call) StringDefault(i int, fallback string) string
func (c *Call) Float64(i int) (float64, error)
func (c *Call) Bool(i int) (bool, error)
func (c *Call) Map(i int) (map[string]any, error)
func (c *Call) Slice(i int) ([]any, error)
```

Why not only raw protobuf values:

- raw protobuf values are transport-oriented, not author-oriented,
- the SDK should reduce author exposure to `structpb` unless needed.

### `sdk.Handler`

Recommended signature:

```go
type Handler func(context.Context, *Call) (any, error)
```

Why this signature:

- keeps `context.Context` available for deadlines/cancellation,
- keeps one stable authoring shape for both functions and object methods,
- returns plain Go values that the SDK can convert centrally.

### Export descriptors

The SDK should model the only two currently supported export shapes directly:

```go
type FunctionExport struct { ... }
type ObjectExport struct { ... }
type MethodExport struct { ... }
```

Why explicit descriptor types instead of maps:

- easier internal validation,
- easier documentation,
- easier parity with the host's manifest validation rules.

## Internal flow

## Module construction flow

Recommended flow:

```text
author calls sdk.NewModule(...)
    -> normalize module metadata
    -> validate export uniqueness
    -> validate object method uniqueness
    -> build immutable in-memory export table
    -> build cached contract.ModuleManifest
    -> return *sdk.Module implementing contract.JSModule
```

Pseudocode:

```go
func NewModule(name string, opts ...ModuleOption) (*Module, error) {
    m := &Module{name: strings.TrimSpace(name)}
    for _, opt := range opts {
        if err := opt.apply(m); err != nil {
            return nil, err
        }
    }
    if err := validateModuleDefinition(m); err != nil {
        return nil, err
    }
    m.manifest = buildManifest(m)
    m.dispatch = buildDispatchTable(m)
    return m, nil
}
```

## Invoke flow

Recommended dispatch path:

```text
host calls Invoke(export_name, method_name, args)
    -> sdk.Module finds handler in dispatch table
    -> args converted from []*structpb.Value to []any
    -> sdk.Call constructed
    -> handler executes
    -> result converted to *structpb.Value
    -> response returned
```

Pseudocode:

```go
func (m *Module) Invoke(ctx context.Context, req *contract.InvokeRequest) (*contract.InvokeResponse, error) {
    handler, ok := m.lookup(req.GetExportName(), req.GetMethodName())
    if !ok {
        return nil, fmt.Errorf("unsupported export %q method %q", req.GetExportName(), req.GetMethodName())
    }

    args, err := decodeArgs(req.GetArgs())
    if err != nil {
        return nil, err
    }

    result, err := handler(ctx, &Call{
        ExportName: req.GetExportName(),
        MethodName: req.GetMethodName(),
        Args:       args,
        RawArgs:    req.GetArgs(),
    })
    if err != nil {
        return nil, err
    }

    value, err := encodeResult(result)
    if err != nil {
        return nil, err
    }

    return &contract.InvokeResponse{Result: value}, nil
}
```

## Serve flow

The current shared transport should stay intact. The SDK should only wrap it:

```go
func Serve(mod contract.JSModule) {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig:  shared.Handshake,
        VersionedPlugins: shared.VersionedServerPluginSets(mod),
        GRPCServer:       plugin.DefaultGRPCServer,
    })
}
```

And the convenience wrapper:

```go
func ServeModule(name string, opts ...ModuleOption) {
    mod, err := NewModule(name, opts...)
    if err != nil {
        panic(err)
    }
    Serve(mod)
}
```

## Validation strategy inside the SDK

The SDK should validate author declarations before runtime.

### SDK-side validation should catch

1. empty module name,
2. empty export names,
3. duplicate exports,
4. object exports with zero methods,
5. duplicate object methods,
6. nil handlers,
7. optional namespace mismatch when using the default namespace policy.

### Why duplicate some host validation

Because the SDK's job is to make invalid states hard to express. Host validation remains authoritative, but SDK validation should fail earlier and with author-focused errors.

### What should remain host-only

The following should stay host-side because they are deployment-policy concerns:

1. allowlist checks,
2. discovery-pattern policy,
3. startup timeout policy,
4. runtime-side namespace configuration overrides.

## Recommended Design Decisions

### Decision 1: Keep the low-level contract public

Do not remove or hide `contract.JSModule`.

Why:

- advanced users may still want full manual control,
- the host and transport layers already depend on it,
- the SDK should be an ergonomic layer, not a mandatory layer.

### Decision 2: Do not use reflection-heavy typed binding in v1

Avoid APIs like:

```go
sdk.Function("greet", func(name string) string { ... })
```

as the primary implementation strategy in v1.

Why:

- reflection adds edge cases around signatures and error messaging,
- explicit `Handler` functions are easier for interns to understand,
- the repo does not yet have a strong shared convention for reflection-based JS binding on the plugin side.

Typed sugar can be added later if the lower-level explicit API stabilizes.

### Decision 3: Model only the host-supported export shapes

Only expose SDK builders for:

1. functions,
2. objects with methods.

Do not add nested object builders, property exports, or async stream abstractions now.

Why:

- `host/reify.go:30-49` only supports these two shapes,
- the SDK should reflect the real runtime capability model.

### Decision 4: Make `ServeModule(...)` the happy path

Most plugin binaries should be one small `main.go`.

That means the common pattern should be:

```go
func main() {
    sdk.ServeModule("plugin:foo", ...)
}
```

Why:

- shortest copyable example,
- smallest tutorial surface,
- clearest migration from current examples.

### Decision 5: Centralize protobuf conversion in the SDK

Plugin authors should mostly return:

- strings,
- numbers,
- booleans,
- slices,
- maps,
- nil.

The SDK should call `structpb.NewValue(...)` centrally and standardize any conversion errors.

Why:

- fewer repeated `structpb` imports,
- fewer inconsistent error messages,
- better future leverage if conversion rules need to evolve.

## Alternatives Considered

## Alternative 1: Keep manual authoring only

Description:

- leave plugins exactly as they are today,
- improve docs only.

Why rejected:

- documentation cannot remove the manifest/dispatch/serve duplication,
- it does not prevent authoring mistakes,
- it leaves the plugin story significantly rougher than built-in native modules.

## Alternative 2: Generate plugin code from a schema

Description:

- ask authors to write YAML/JSON/proto metadata,
- generate Go plugin boilerplate.

Why rejected for now:

- code generation is a larger workflow commitment,
- the desired manual API is not stable enough yet,
- a hand-authored SDK is easier to iterate on and review first.

## Alternative 3: Reflection-first typed binding

Description:

- accept arbitrary Go functions and structs,
- inspect signatures automatically,
- derive manifests from reflection.

Why rejected for v1:

- more magical,
- more error-prone,
- harder to debug,
- harder to explain to an intern than explicit handler descriptors.

## Alternative 4: Move policy into the SDK

Description:

- let the SDK own discovery, validation policy, and host integration.

Why rejected:

- `pkg/hashiplugin/host` already owns runtime-side policy,
- authoring and hosting are different concerns,
- this would blur the clean layering introduced in GOJA-08.

## Detailed Implementation Plan

## Phase 1: Add the SDK package and manifest builder

Files to add:

- `pkg/hashiplugin/sdk/module.go`
- `pkg/hashiplugin/sdk/export.go`
- `pkg/hashiplugin/sdk/serve.go`

Goals:

1. author can declare module metadata,
2. author can declare function/object exports,
3. SDK can build a `contract.ModuleManifest`,
4. SDK can expose `Manifest(...)`,
5. SDK can expose `Serve(...)`.

Suggested first tests:

- module manifest contains correct function export,
- object export includes methods in manifest,
- duplicate export names fail,
- empty method names fail.

## Phase 2: Add dispatch and conversion helpers

Files to add:

- `pkg/hashiplugin/sdk/dispatch.go`
- `pkg/hashiplugin/sdk/call.go`
- `pkg/hashiplugin/sdk/convert.go`

Goals:

1. SDK can route `Invoke(...)` without author-written string switches,
2. SDK can decode request args into `[]any`,
3. SDK can encode plain Go results into `*structpb.Value`,
4. SDK can produce author-friendly errors.

Suggested tests:

- function handler receives args in order,
- object method handler receives method dispatch correctly,
- unsupported export returns a clear error,
- unsupported return type produces a stable conversion error.

## Phase 3: Migrate examples and fixtures

Files to update:

- `plugins/examples/greeter/main.go`
- `plugins/testplugin/echo/main.go`
- `plugins/testplugin/invalid/main.go`

Goals:

1. prove the SDK works for real plugin binaries,
2. shrink example plugin code materially,
3. keep the invalid fixture available for host validation tests.

Notes:

- the invalid fixture should remain low-level or use the SDK in a way that intentionally violates namespace policy, depending on which failure mode needs coverage.
- one useful pattern is to leave one fixture manual so the test suite still covers the raw contract path too.

## Phase 4: Update docs and tutorials

Files to update:

- `pkg/doc/12-plugin-user-guide.md`
- `pkg/doc/13-plugin-developer-guide.md`
- `pkg/doc/14-plugin-tutorial-build-install.md`

Goals:

1. teach `sdk.ServeModule(...)` as the default authoring path,
2. explain the low-level contract as an advanced topic,
3. show how the SDK relates to the host/runtime model.

## Phase 5: Add parity and end-to-end tests

Files to add/update:

- `pkg/hashiplugin/sdk/sdk_test.go`
- `pkg/hashiplugin/host/registrar_test.go`

Goals:

1. SDK-generated plugins load through the existing host path,
2. runtime behavior remains unchanged from JavaScript,
3. examples remain usable via `repl` and `js-repl`.

## Testing Strategy

The test strategy should prove both API ergonomics and runtime compatibility.

### Unit tests for the SDK package

Add table-driven unit tests for:

- manifest generation,
- duplicate detection,
- object method registration,
- argument decoding,
- result encoding,
- dispatch errors.

### Integration tests through the existing host package

Reuse the current pattern in `pkg/hashiplugin/host/registrar_test.go:17-162`:

1. build a plugin binary from source,
2. create an engine runtime,
3. require the plugin module,
4. call exported functions,
5. close the runtime,
6. verify process exit when relevant.

This is the most important compatibility proof because it exercises the real host/runtime path instead of a mock.

### Documentation verification

After migration, verify:

- `go run ./cmd/repl help goja-plugin-user-guide`
- `go run ./cmd/repl help goja-plugin-developer-guide`
- `go run ./cmd/repl help plugin-tutorial-build-install`

### Manual smoke path

The same smoke path should continue to work:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-greeter ./plugins/examples/greeter
go run ./cmd/repl
```

Then:

```javascript
const greeter = require("plugin:greeter")
greeter.greet("hello")
greeter.strings.upper("hello")
```

## Risks

## Risk 1: SDK and host validation drift

If SDK-side validation differs from `host.ValidateManifest(...)`, authors could see confusing mismatches.

Mitigation:

- keep SDK validation minimal and structurally aligned with host validation,
- add parity tests for manifest shapes.

## Risk 2: Over-designing the author-facing API

If v1 adds too much sugar, the API will be harder to stabilize.

Mitigation:

- start with explicit `Handler` functions,
- add only the convenience helpers clearly justified by current examples.

## Risk 3: Hiding transport details too completely

If the SDK makes the plugin boundary feel like an in-process native module, authors may design APIs that do not fit RPC boundaries well.

Mitigation:

- keep docs explicit that this is still a process boundary,
- keep `context.Context` and plain-data constraints visible.

## Risk 4: Reflection creep

There will be pressure to accept arbitrary Go signatures immediately.

Mitigation:

- defer reflection-first binding until the explicit API has proven itself in examples and real plugins.

## Open Questions

1. Should the package name be `pkg/hashiplugin/sdk` or `pkg/hashiplugin/authoring`? I recommend `sdk` for brevity, but `authoring` is more explicit.
2. Should `ServeModule(...)` panic on invalid definitions or return an error? I recommend `Serve(mod)` plus `MustModule(...)` so both styles exist.
3. Should `sdk.Call` expose typed getters only, or also retain raw `[]any` and `[]*structpb.Value`? I recommend both, with raw access as an escape hatch.
4. Should SDK-side namespace validation default to `plugin:` even though the host is configurable? I recommend yes, with an internal override only if the host policy ever genuinely changes.
5. Should the invalid test fixture continue to be handwritten to keep coverage for the raw contract path? I recommend yes.

## Intern-Oriented Build Order

If you are the intern implementing this, do the work in this order:

1. Read the files listed in the "Read This First" section.
2. Write down the two supported export shapes from `host/reify.go`.
3. Implement `sdk.Module` and manifest generation first.
4. Add unit tests before adding dispatch.
5. Implement dispatch and conversion helpers second.
6. Migrate `plugins/examples/greeter`.
7. Run the existing host integration tests.
8. Only after the example plugin works should you update docs.

This order matters because it keeps the work aligned with the real runtime path. If you start with tutorial rewrites or reflection ideas, you will likely lose track of the actual host invariants.

## Reference Map

Use this file map when you are implementing or reviewing the SDK:

- `engine/runtime_modules.go:12-25` - runtime-scoped registration seam
- `engine/factory.go:176-213` - runtime creation order and registrar execution
- `engine/runtime.go:36-86` - runtime cleanup hooks
- `modules/common.go:29-33` - built-in native module authoring interface
- `modules/exports.go:7-12` - export helper for built-in modules
- `pkg/hashiplugin/contract/contract.go:5-10` - low-level plugin interface
- `pkg/hashiplugin/contract/jsmodule.proto:10-44` - wire contract
- `pkg/hashiplugin/shared/plugin.go:20-75` - handshake and serve/dispense helpers
- `pkg/hashiplugin/host/validate.go:10-76` - authoritative manifest validation rules
- `pkg/hashiplugin/host/reify.go:14-86` - supported runtime export shapes
- `pkg/hashiplugin/host/client.go:69-121` - manifest fetch and validation timing
- `pkg/hashiplugin/host/registrar.go:25-73` - runtime-side plugin registration flow
- `plugins/examples/greeter/main.go:17-100` - current authoring pain in the user-facing example
- `plugins/testplugin/echo/main.go:16-75` - duplicated authoring pattern in tests
- `pkg/doc/02-creating-modules.md:18-56` - usability baseline from built-in native modules
- `pkg/doc/13-plugin-developer-guide.md:23-175` - current subsystem architecture

## Recommended End State

The end state should feel like this:

- Host engineers continue thinking in terms of `host.Config`, `host.NewRegistrar(...)`, runtime registrars, and CommonJS module registration.
- Plugin authors think in terms of `sdk.ServeModule(...)`, `sdk.Function(...)`, and `sdk.Object(...sdk.Method(...))`.
- Advanced users can still drop down to `contract.JSModule` directly when necessary.

That is the right separation of concerns for this codebase.
