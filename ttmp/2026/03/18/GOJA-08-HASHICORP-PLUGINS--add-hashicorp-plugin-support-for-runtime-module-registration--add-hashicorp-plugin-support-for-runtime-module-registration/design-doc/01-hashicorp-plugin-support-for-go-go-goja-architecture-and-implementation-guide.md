---
Title: HashiCorp plugin support for go-go-goja architecture and implementation guide
Ticket: GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - architecture
    - security
    - tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/repl/main.go
      Note: Simple runtime entrypoint likely to expose plugin-directory configuration
    - Path: engine/factory.go
      Note: |-
        Current immutable factory and runtime creation flow that plugin support must extend
        Current immutable factory and runtime creation lifecycle
    - Path: engine/module_specs.go
      Note: |-
        Existing static module registration contract and builder composition seam
        Current module registration seam that plugin support must extend
    - Path: engine/runtime.go
      Note: |-
        Current runtime close behavior and lack of generic cleanup hooks
        Current runtime close behavior and missing cleanup hook support
    - Path: modules/common.go
      Note: |-
        Native module interface and default global registry behavior
        Global native module registry behavior and singleton receiver semantics
    - Path: modules/database/database.go
      Note: |-
        Stateful native module example that exposes singleton-lifecycle implications
        Stateful module example that shapes runtime-scoped plugin design
    - Path: modules/exports.go
      Note: Existing export helper that plugin reification can mirror
    - Path: modules/fs/fs.go
      Note: Simple stateless native module example for comparison
    - Path: pkg/jsverbs/runtime.go
      Note: Example of owner-serialized runtime execution used by another dynamic subsystem
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Existing owned-runtime consumer that should later accept plugin configuration
    - Path: pkg/runtimeowner/runner.go
      Note: |-
        Runtime owner contract that plugin-backed closures must respect
        Owner-goroutine contract for all goja-facing plugin closures
    - Path: ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/sources/local/Imported goja plugins note.md
      Note: |-
        Imported source memo that this document interprets and refines
        Imported source memo interpreted and refined by this design doc
ExternalSources:
    - local:Imported goja plugins note.md
Summary: Repo-grounded intern-facing design for adding HashiCorp go-plugin module support to go-go-goja without violating runtime ownership, module lifecycle, or trust-boundary constraints.
LastUpdated: 2026-03-18T09:14:54.589318508-04:00
WhatFor: Give a new engineer enough context to implement runtime-scoped plugin-backed native modules in go-go-goja with clear APIs, file targets, and test strategy.
WhenToUse: Use when implementing or reviewing HashiCorp plugin support in go-go-goja.
---


# HashiCorp plugin support for go-go-goja architecture and implementation guide

## Executive Summary

The imported note in [Imported goja plugins note.md](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/sources/local/Imported%20goja%20plugins%20note.md) gets the most important architectural point right: `goja.Runtime` must stay fully owned by the host process, and HashiCorp `go-plugin` should only handle plugin process discovery, version negotiation, transport, and RPC. Plugins should return manifests and plain data, not `goja.Value` or `goja.Object`.

My repo-specific conclusion is slightly different from the imported sketch in three places. First, plugin exports should be reified as CommonJS native modules, because `go-go-goja` already centers module composition around `require.Registry`, `modules.NativeModule`, and `engine.FactoryBuilder`. Second, plugin loading should be runtime-scoped rather than factory-scoped, because the current codebase uses a shared module registry and has no generic cleanup mechanism for long-lived external resources. Third, v1 should support only fixed functions and fixed object methods over a JSON-like data boundary; `DynamicObject`, callbacks, and streams should wait until the base lifecycle is correct.

The recommended implementation therefore has two layers:

1. Add a small engine refactor so a runtime can build its own `require.Registry`, register runtime-scoped synthetic modules, and register cleanup hooks.
2. Add a HashiCorp plugin host subsystem that discovers plugin binaries, validates manifests, creates plugin clients, and reifies each approved plugin as `require("plugin:<name>")`.

That is the smallest coherent design that fits the existing code.

## Problem Statement And Scope

### Problem

`go-go-goja` already supports native Go modules, but only through static registration paths:

1. Built-in modules satisfy `modules.NativeModule` and self-register into a package-global registry in [modules/common.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/common.go#L29).
2. `engine.FactoryBuilder` currently collects `ModuleSpec` values, builds one `require.Registry`, and stores that frozen registry inside `Factory` in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L17).
3. Runtime creation later enables that already-built registry inside a new VM in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L136).

That model is good for built-in modules compiled into the host binary, but it does not solve the user request: discover external plugin binaries at runtime and register them into a runtime safely.

The hard constraints are:

1. `goja` runtime access must stay on the owner goroutine. The repo already encodes this through `runtimeowner.Runner` in [pkg/runtimeowner/runner.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/runtimeowner/runner.go#L62).
2. Plugin subprocesses have independent lifecycles, so the host needs a real cleanup path. Current `Runtime.Close()` only shuts down the owner and event loop in [engine/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go#L23).
3. `go-go-goja` consumers already expect native functionality through `require(...)`, not through arbitrary globals. Examples include [cmd/repl/main.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go#L34) and [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go#L95).
4. Plugin trust boundaries need more validation than “binary exists and speaks the cookie.”

### In Scope

This ticket covers:

1. runtime-scoped plugin discovery and validation,
2. a manifest-driven RPC contract,
3. reifying plugin exports into CommonJS modules inside `goja`,
4. lifecycle management and cleanup,
5. test strategy, rollout phases, and file-level implementation guidance.

### Out Of Scope For V1

This ticket intentionally does not include:

1. passing JS callbacks into plugins,
2. streaming/event subscriptions,
3. function-valued return values,
4. plugin-driven mutation of arbitrary global runtime state,
5. hot-reload or hot-swap of already-required modules.

## System Orientation

If you are new to this repository, understand these parts first.

### 1. `engine.FactoryBuilder` is the runtime composition entrypoint

The builder currently has two extension seams:

1. `WithModules(...)` for static module registration in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L60),
2. `WithRuntimeInitializers(...)` for post-runtime hooks in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L67).

The factory created by `Build()` is immutable and currently holds a single `*require.Registry` in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L25).

### 2. `modules.DefaultRegistry` is global and stores module receivers

The module registry in [modules/common.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/common.go#L35) stores `NativeModule` instances. That means a stateful receiver like `DBModule` in [modules/database/database.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/database/database.go#L14) is not a per-runtime value by default.

That matters because plugin clients are also stateful resources. If you accidentally model them as global receiver singletons, you couple subprocess lifecycle and connection state to the process rather than to a runtime instance.

### 3. `runtimeowner.Runner` is the concurrency guard

`runtimeowner.Runner.Call(...)` and `Post(...)` schedule work on the runtime owner context in [pkg/runtimeowner/runner.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/runtimeowner/runner.go#L62). Any JS-facing closure that touches `goja.Value`, `vm.ToValue(...)`, `vm.NewGoError(...)`, or `module.exports` must assume it is executing within that owner-controlled context.

### 4. Current consumers build owned runtimes through the engine

- The simple REPL builds a runtime with `DefaultRegistryModules()` in [cmd/repl/main.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go#L34).
- The evaluator does the same when module support is enabled in [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go#L95).
- `pkg/jsverbs` already shows the pattern of running runtime-sensitive logic through `Owner.Call(...)` in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/jsverbs/runtime.go#L18).

These are the first places that will later want plugin-directory configuration, but not the first places you should edit.

## Current-State Analysis

### Static module registration only

`ModuleSpec` is a build-time abstraction that receives only a `*require.Registry` in [engine/module_specs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/module_specs.go#L14). It has no access to a runtime owner, no runtime cleanup hook, and no runtime-scoped configuration.

Observed behavior:

1. `Build()` validates specs and registers them immediately into one registry in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L90).
2. `NewRuntime()` later enables that same registry in a new VM in [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L156).

Inference: any module design that needs runtime-scoped data or runtime-scoped cleanup is awkward in the current engine shape.

### Runtime shutdown is too small for plugin subprocesses

`Runtime.Close()` only does two things:

1. `Owner.Shutdown(ctx)`
2. `Loop.Stop()`

Evidence: [engine/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go#L29).

Inference: plugin clients and plugin subprocesses need an additional cleanup path.

### The default module registry is process-global

`modules.Register(...)` appends module receivers into `DefaultRegistry` in [modules/common.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/common.go#L84). That is fine for built-ins like [modules/fs/fs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/fs/fs.go#L14), but `DBModule` demonstrates that modules can also hold mutable Go state in [modules/database/database.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/database/database.go#L15).

Inference: if plugin support simply adds plugin clients as more global modules, it will inherit the same cross-runtime coupling risk.

### JS-facing execution already assumes host-side value conversion

The evaluator and jsverbs runtime both convert JS values in the host process:

- `vm.RunString(...)` and result handling in [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go#L202),
- argument conversion with `vm.ToValue(...)` in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/jsverbs/runtime.go#L63),
- result export with `result.Export()` in [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/jsverbs/runtime.go#L71).

That aligns well with the imported note’s requirement to send only boring data over RPC.

## Gap Analysis

There are five repo-specific gaps between the current code and the requested plugin feature.

1. There is no runtime-scoped module registration seam before `require.Registry.Enable(vm)`.
2. There is no runtime cleanup hook for plugin clients/processes.
3. There is no manifest or policy layer for validating plugin exports before registration.
4. There is no namespace policy preventing a plugin from colliding with `fs`, `database`, or future built-ins.
5. There is no test fixture or example pattern for subprocess-backed modules.

## Imported Note Assessment

The imported memo is useful, but it should be treated as a starting hypothesis rather than the final design.

### What it gets right

1. Host-owned `goja.Runtime` is mandatory.
2. RPC payloads must be plain data, not JS runtime objects.
3. `go-plugin` should be used for process management, versioning, and transport, not as a way to share Goja internals.
4. gRPC is the right long-term transport choice.

### Where this design deliberately differs

1. The memo reifies plugin modules as globals with `vm.Set(...)`. This repo should instead expose them as native CommonJS modules because that matches the existing module system.
2. The memo does not address the current `Factory` lifecycle. This design does, because plugin support will otherwise leak subprocesses or share them globally by accident.
3. The memo mentions `DynamicObject` as an early option. This design defers it to phase 2, because fixed exports are enough for v1 and far easier to test.

## Proposed Solution

### High-level architecture

Treat each plugin as a remote module provider with a manifest and an invoke API. The host runtime discovers plugins, validates them, turns each manifest into a synthetic native module loader, and registers that loader under a reserved require namespace such as `plugin:<name>`.

Recommended runtime shape:

```text
+---------------------+        gRPC         +----------------------+
| go-go-goja host     | <----------------> | plugin subprocess    |
|                     |                    |                      |
| goja.Runtime        |                    | Manifest()           |
| require.Registry    |                    | Invoke(...)          |
| runtimeowner.Runner |                    | versioned protocol   |
+----------+----------+                    +----------------------+
           |
           | require("plugin:math")
           v
+------------------------------+
| synthetic native module      |
| module.exports.add = closure |
| closure Export() -> RPC      |
| RPC result -> vm.ToValue()   |
+------------------------------+
```

### Key design decisions

#### Decision 1: Use a reserved module namespace

Register plugin modules as `plugin:<name>` rather than bare names.

Why:

1. avoids collisions with built-ins such as `fs` and `database`,
2. makes provenance obvious in JS,
3. lets policy reject manifests that try to escape the reserved namespace.

Example JS:

```javascript
const math = require("plugin:math")
math.add(2, 3)
```

#### Decision 2: Add runtime-scoped module registration before `Enable(vm)`

The biggest engine change should be a new pre-enable seam. Conceptually:

```go
type RuntimeModuleRegistrar interface {
    ID() string
    RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error
}
```

Where `RuntimeModuleContext` includes at least:

```go
type RuntimeModuleContext struct {
    VM        *goja.Runtime
    Loop      *eventloop.EventLoop
    Owner     runtimeowner.Runner
    AddCloser func(func(context.Context) error)
}
```

Why this is the right seam:

1. plugin discovery and validation may depend on runtime-scoped policy,
2. synthetic native modules must be in the registry before `Enable(vm)`,
3. plugin clients need runtime cleanup.

#### Decision 3: Refactor `Factory` to rebuild a registry per runtime

Today `Factory` stores a single built registry. For plugin support, prefer storing the immutable composition plan and creating a fresh `require.Registry` inside `NewRuntime()`.

Pseudocode:

```go
type Factory struct {
    settings                builderSettings
    modules                 []ModuleSpec
    runtimeModuleRegistrars []RuntimeModuleRegistrar
    runtimeInitializers     []RuntimeInitializer
}

func (f *Factory) NewRuntime(ctx context.Context) (*Runtime, error) {
    vm := goja.New()
    loop := eventloop.NewEventLoop()
    go loop.Start()

    owner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
        Name: "go-go-goja-runtime",
        RecoverPanics: true,
    })

    rt := &Runtime{VM: vm, Loop: loop, Owner: owner}
    reg := require.NewRegistry(f.settings.requireOptions...)

    for _, mod := range f.modules {
        if err := mod.Register(reg); err != nil {
            _ = rt.Close(ctx)
            return nil, err
        }
    }

    preCtx := &RuntimeModuleContext{
        VM: vm,
        Loop: loop,
        Owner: owner,
        AddCloser: rt.AddCloser,
    }
    for _, registrar := range f.runtimeModuleRegistrars {
        if err := registrar.RegisterRuntimeModules(preCtx, reg); err != nil {
            _ = rt.Close(ctx)
            return nil, err
        }
    }

    req := reg.Enable(vm)
    console.Enable(vm)
    rt.Require = req

    // existing runtime initializer phase continues here
    return rt, nil
}
```

#### Decision 4: Add generic runtime cleanup hooks

Extend `engine.Runtime` with ordered cleanup functions:

```go
type Runtime struct {
    VM      *goja.Runtime
    Require *require.RequireModule
    Loop    *eventloop.EventLoop
    Owner   runtimeowner.Runner

    closers []func(context.Context) error
}
```

Shutdown order should be:

1. custom closers, in reverse registration order,
2. owner shutdown,
3. event loop stop.

Why:

1. plugin clients may depend on the owner still being alive while cleanup runs,
2. reverse order matches normal resource stack discipline,
3. the mechanism is reusable for future external resources.

#### Decision 5: Keep the RPC contract boring

V1 contract:

```go
type ModuleManifest struct {
    ModuleName string
    Version    string
    Exports    []ExportSpec
    Capabilities []string
}

type ExportSpec struct {
    Name    string
    Kind    string   // "function" | "object"
    Methods []string
    Doc     string
}

type InvokeRequest struct {
    ExportName string
    MethodName string
    Args       []*structpb.Value
}

type InvokeResponse struct {
    Result *structpb.Value
}
```

Supported value kinds in v1:

1. null
2. bool
3. string
4. finite number
5. arrays
6. objects/maps with string keys

Optional later:

1. bytes
2. richer typed wrappers
3. structured errors

#### Decision 6: Use gRPC and versioned plugin sets

The imported note is correct here. Use `go-plugin` gRPC mode, `AllowedProtocols` restricted to gRPC, `VersionedPlugins` for schema negotiation, and checksum/allowlist policy around discovery.

#### Decision 7: No `DynamicObject` in v1

Dynamic proxies are attractive, but they complicate caching, property enumeration, and test expectations. The current repo has no existing need for plugin-defined property mutation. Fixed functions and fixed object methods are enough for the first shipped feature.

## Proposed Package And File Layout

Recommended file targets:

```text
go-go-goja/
├── engine/
│   ├── factory.go
│   ├── module_specs.go
│   ├── runtime.go
│   └── plugin_registrars.go            # new
├── pkg/hashiplugin/
│   ├── contract/
│   │   ├── manifest.go                 # Go-side manifest types
│   │   ├── jsmodule.proto              # gRPC contract
│   │   └── generated/...               # generated protobuf/grpc code
│   ├── shared/
│   │   └── plugin.go                   # Handshake + PluginMap + GRPC adapter
│   ├── host/
│   │   ├── config.go
│   │   ├── discover.go
│   │   ├── validate.go
│   │   ├── client.go
│   │   ├── registrar.go
│   │   └── reify.go
│   └── testplugin/
│       └── echo/main.go                # example/integration fixture
├── testdata/plugins/
│   └── echo/                           # optional built binary fixtures or sources
└── cmd/repl/main.go                    # future flag wiring
```

This layout keeps the plugin transport and lifecycle code out of `modules/`, because plugin-backed modules are not ordinary compile-time `NativeModule` implementations.

## Detailed Control Flow

### Runtime creation flow

```text
Factory.NewRuntime
  -> create goja runtime
  -> create event loop
  -> create runtimeowner.Runner
  -> create fresh require.Registry
  -> register static ModuleSpecs
  -> register runtime-scoped plugin modules
       -> discover binaries
       -> validate path/policy
       -> start plugin clients
       -> fetch manifests
       -> validate manifests
       -> register synthetic native loaders
       -> register runtime closers
  -> Enable(vm)
  -> console.Enable(vm)
  -> run RuntimeInitializers
```

### Module require flow

```text
JS require("plugin:math")
  -> goja_nodejs invokes synthetic module loader
  -> loader creates module.exports object
  -> loader installs closures for each export from manifest
  -> closure converts call.Arguments via Export()
  -> closure invokes plugin RPC
  -> closure converts result back via vm.ToValue(...)
  -> JS receives ordinary values
```

### Shutdown flow

```text
Runtime.Close
  -> kill/close plugin clients
  -> owner.Shutdown(ctx)
  -> loop.Stop()
```

## API Sketches

### Engine API

Builder surface:

```go
builder := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    WithRuntimeModuleRegistrars(
        hashiplugin.NewRegistrar(hashiplugin.Config{
            Directories: []string{"./plugins"},
            Pattern:     "goja-plugin-*",
            Namespace:   "plugin:",
        }),
    )
```

New engine types:

```go
type RuntimeModuleRegistrar interface {
    ID() string
    RegisterRuntimeModules(ctx *RuntimeModuleContext, reg *require.Registry) error
}

type RuntimeModuleContext struct {
    VM        *goja.Runtime
    Loop      *eventloop.EventLoop
    Owner     runtimeowner.Runner
    AddCloser func(func(context.Context) error)
}
```

### Host plugin config

```go
type Config struct {
    Directories       []string
    Pattern           string
    Namespace         string
    AllowModules      []string
    RequireChecksum   bool
    SecureConfig      *plugin.SecureConfig
    Handshake         plugin.HandshakeConfig
    VersionedPlugins  map[int]plugin.PluginSet
    CallTimeout       time.Duration
}
```

### Manifest validation rules

Reject the plugin if any of these are true:

1. module name is empty,
2. module name does not live under the reserved namespace after normalization,
3. duplicate export names exist,
4. unsupported export kind is present,
5. object export has duplicate or empty method names,
6. manifest version/capability is incompatible with the host,
7. checksum or allowlist policy fails.

## Reification Pseudocode

This is the core host-side logic.

```go
func registerManifestModule(
    reg *require.Registry,
    requireName string,
    client JSModuleClient,
    manifest *contract.ModuleManifest,
) {
    reg.RegisterNativeModule(requireName, func(vm *goja.Runtime, moduleObj *goja.Object) {
        exports := moduleObj.Get("exports").(*goja.Object)

        for _, exp := range manifest.Exports {
            switch exp.Kind {
            case "function":
                name := exp.Name
                _ = exports.Set(name, func(call goja.FunctionCall) goja.Value {
                    args, err := exportArgs(call.Arguments)
                    if err != nil {
                        panic(vm.NewGoError(err))
                    }

                    resp, err := client.Invoke(context.Background(), &contract.InvokeRequest{
                        ExportName: name,
                        Args:       args,
                    })
                    if err != nil {
                        panic(vm.NewGoError(err))
                    }

                    return toGojaValue(vm, resp.Result)
                })

            case "object":
                obj := vm.NewObject()
                for _, method := range exp.Methods {
                    method := method
                    _ = obj.Set(method, func(call goja.FunctionCall) goja.Value {
                        args, err := exportArgs(call.Arguments)
                        if err != nil {
                            panic(vm.NewGoError(err))
                        }

                        resp, err := client.Invoke(context.Background(), &contract.InvokeRequest{
                            ExportName: exp.Name,
                            MethodName: method,
                            Args:       args,
                        })
                        if err != nil {
                            panic(vm.NewGoError(err))
                        }

                        return toGojaValue(vm, resp.Result)
                    })
                }
                _ = exports.Set(exp.Name, obj)
            }
        }
    })
}
```

Important note for the intern: the closure runs on the VM goroutine when JS calls it, so it is allowed to touch `goja.FunctionCall`, `vm.NewObject()`, and `vm.ToValue(...)`. The plugin RPC itself can block that goroutine in v1. That is acceptable for a synchronous API, but it means plugin latency is directly visible to JS execution.

## File-by-File Implementation Plan

### Phase 1: prepare engine lifecycle seams

Target files:

1. [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go)
2. [engine/module_specs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/module_specs.go)
3. [engine/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go)
4. `engine/plugin_registrars.go` (new)

Tasks:

1. Change `Factory` to store composition inputs rather than a single built `require.Registry`.
2. Add `runtimeModuleRegistrars []RuntimeModuleRegistrar` to builder and factory.
3. Add `WithRuntimeModuleRegistrars(...)`.
4. Add `Runtime.AddCloser(...)` and close-time execution.
5. Update `Build()` validation to include runtime registrar IDs.

Acceptance criteria:

1. Existing tests still pass.
2. A no-op runtime registrar can register a synthetic module into one runtime without mutating another runtime.

### Phase 2: add the plugin contract and shared adapter

Target files:

1. `pkg/hashiplugin/contract/*`
2. `pkg/hashiplugin/shared/plugin.go`
3. `go.mod`

Tasks:

1. Add `github.com/hashicorp/go-plugin` dependency.
2. Define protobuf contract for `Manifest` and `Invoke`.
3. Generate gRPC bindings.
4. Implement the `plugin.Plugin` adapter with `GRPCServer(...)` and `GRPCClient(...)`.
5. Define handshake config and versioned plugin map.

Acceptance criteria:

1. The host and an example plugin binary can handshake and dispense the service.

### Phase 3: add host discovery, validation, and reification

Target files:

1. `pkg/hashiplugin/host/config.go`
2. `pkg/hashiplugin/host/discover.go`
3. `pkg/hashiplugin/host/validate.go`
4. `pkg/hashiplugin/host/client.go`
5. `pkg/hashiplugin/host/reify.go`
6. `pkg/hashiplugin/host/registrar.go`

Tasks:

1. Discover candidate binaries from configured directories.
2. Filter by executable bit, filename pattern, allowlist, and optional checksum.
3. Start plugin clients with gRPC-only configuration.
4. Fetch manifests and validate them.
5. Register each validated manifest as a synthetic native module.
6. Register closers that shut down each plugin client.

Acceptance criteria:

1. `require("plugin:echo")` works in a runtime built with the registrar.
2. Invalid manifests are rejected before JS can `require(...)` them.
3. `Runtime.Close()` tears down plugin subprocesses.

### Phase 4: integrate with entrypoints and examples

Target files:

1. [cmd/repl/main.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go)
2. [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go)
3. optional example command or plugin fixture under `cmd/` or `testdata/`

Tasks:

1. Add optional plugin-directory configuration.
2. Wire the registrar into runtime creation when configured.
3. Provide a tiny example plugin for manual testing and documentation.

Acceptance criteria:

1. A developer can point the REPL at a plugin directory and `require(...)` the plugin.

### Phase 5: document and harden

Target files:

1. docs/help pages if plugin authoring is exposed publicly,
2. GOJA-08 ticket docs,
3. test fixtures and playbooks.

Tasks:

1. Document module namespace policy and manifest rules.
2. Document security caveats of checksums versus cookies.
3. Add failure-mode tests and review notes.

## Testing And Validation Strategy

Write tests in layers.

### Unit tests

1. discovery filters reject non-executables and unexpected names,
2. manifest validation rejects duplicates and invalid names,
3. reification produces the expected export shapes from a manifest.

### Engine tests

1. plugin registrar registers modules into one runtime without polluting another runtime,
2. cleanup hooks run on `Runtime.Close()`,
3. runtime creation fails cleanly if plugin validation fails.

### Integration tests

Use a tiny test plugin binary that exposes:

1. one function export,
2. one object export with one method,
3. one manifest validation edge case fixture.

Suggested end-to-end assertions:

1. `require("plugin:echo").ping("hi")` returns `"hi"`,
2. object method calls round-trip arrays and maps,
3. plugin subprocess exits after runtime close,
4. duplicate plugin names fail before runtime is returned.

### Manual verification

Suggested commands for the intern after implementation:

```bash
go test ./... -count=1
go test ./pkg/hashiplugin/... -count=1
go test ./engine/... -count=1
go run ./cmd/repl --plugin-dir ./testdata/plugins/bin
```

Inside the REPL:

```javascript
const echo = require("plugin:echo")
echo.ping("hello")
```

## Risks And Open Questions

### Risks

1. Long-running plugin RPC calls will block the VM goroutine in v1.
2. Proto/codegen workflow may introduce maintenance overhead if not standardized.
3. Security expectations can drift if developers mistake the handshake cookie for real validation.
4. The current built-in module singleton model may tempt implementers to make plugin clients global too.

### Open questions

1. Should plugin discovery happen once per runtime or once per process with cached manifest metadata and runtime-scoped client connections?
2. Should the namespace be `plugin:<name>` or `plugin/<name>`?
3. Should the host expose plugin docs/help text through the existing help system?
4. Do we want a dedicated CLI for inspecting manifests without launching a full runtime?

## Alternatives Considered

### Alternative 1: use the Go standard library `plugin` package

Rejected because it gives weaker portability, weaker process isolation, and does not match the explicit subprocess/RPC model requested by the user.

### Alternative 2: let plugins mutate the runtime directly

Rejected because it violates `goja` ownership and makes auditing and lifecycle management much harder.

### Alternative 3: register plugin exports as globals

Rejected because current repo conventions center around `require(...)`, and globals create more collision risk and weaker provenance.

### Alternative 4: discover and launch plugins during `Build()`

Rejected for v1 because the current factory lifecycle would make subprocess cleanup and per-runtime isolation too awkward.

## Intern Checklist

If you are the person implementing this:

1. Read [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go#L90), [engine/module_specs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/module_specs.go#L14), and [engine/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go#L23) first.
2. Read [modules/common.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/common.go#L35) and [modules/database/database.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/database/database.go#L14) next so you understand the current singleton module behavior.
3. Read [pkg/runtimeowner/runner.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/runtimeowner/runner.go#L62) before writing any JS-facing closure code.
4. Implement the engine seam changes before writing the plugin host.
5. Get one tiny plugin working end-to-end before adding policy and hardening.

## References

### Repository files

- [engine/factory.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go)
- [engine/module_specs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/module_specs.go)
- [engine/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/runtime.go)
- [modules/common.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/common.go)
- [modules/exports.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/exports.go)
- [modules/fs/fs.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/fs/fs.go)
- [modules/database/database.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/modules/database/database.go)
- [pkg/runtimeowner/runner.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/runtimeowner/runner.go)
- [pkg/repl/evaluators/javascript/evaluator.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go)
- [pkg/jsverbs/runtime.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/pkg/jsverbs/runtime.go)
- [cmd/repl/main.go](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/cmd/repl/main.go)

### Imported source

- [Imported goja plugins note.md](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/sources/local/Imported%20goja%20plugins%20note.md)

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
