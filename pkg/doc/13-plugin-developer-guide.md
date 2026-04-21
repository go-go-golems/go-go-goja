---
Title: "HashiCorp plugin developer guide for go-go-goja"
Slug: goja-plugin-developer-guide
Short: "Detailed architecture guide for how runtime-scoped HashiCorp plugin support is implemented inside go-go-goja."
Topics:
- goja
- plugins
- hashicorp
- developer-guide
- architecture
- engine
Commands:
- goja-repl
Flags:
- --plugin-dir
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page is for developers working on the plugin subsystem itself rather than just using it.

The most important architectural fact is this: plugin support in `go-go-goja` is host-owned runtime composition, not remote JavaScript execution. The host process owns `goja.Runtime`, `require.Registry`, runtime lifecycle, and module registration. Plugin subprocesses provide discovery metadata and per-call execution over RPC, but they do not own the JavaScript VM.

That design choice is what keeps the rest of the system coherent. It lets existing `require()` semantics stay in the host, lets plugin modules look like ordinary native modules to JavaScript callers, and ensures cleanup happens through the same runtime lifecycle as any other runtime-owned resource.

## Read this first

If you are new to the subsystem, read these files in this order:

- `engine/runtime_modules.go`
- `engine/factory.go`
- `engine/runtime.go`
- `pkg/hashiplugin/sdk/module.go`
- `pkg/hashiplugin/sdk/export.go`
- `pkg/hashiplugin/sdk/call.go`
- `pkg/hashiplugin/sdk/dispatch.go`
- `pkg/hashiplugin/sdk/serve.go`
- `pkg/hashiplugin/contract/jsmodule.proto`
- `pkg/hashiplugin/shared/plugin.go`
- `pkg/hashiplugin/host/config.go`
- `pkg/hashiplugin/host/discover.go`
- `pkg/hashiplugin/host/client.go`
- `pkg/hashiplugin/host/validate.go`
- `pkg/hashiplugin/host/reify.go`
- `pkg/hashiplugin/host/registrar.go`
- `pkg/hashiplugin/host/registrar_test.go`
- `pkg/repl/evaluators/javascript/evaluator.go`
- `cmd/goja-repl/root.go`
- `cmd/goja-repl/tui.go`

That order mirrors the real layering:

- engine seam,
- authoring SDK,
- transport contract,
- host policy,
- runtime integration,
- entrypoint wiring.

## Problem statement

Before this work, the engine built one `require.Registry` up front and had no general runtime-scoped module-registration seam plus no general runtime cleanup hook for external resources. That made plugin support awkward because plugins want both:

- per-runtime registration, and
- per-runtime process cleanup.

If plugin clients had been attached to the factory instead, subprocess lifetime would have drifted away from runtime lifetime. That would have made runtime reuse and shutdown behavior confusing and fragile.

## Core architecture

The current plugin architecture has five layers:

```text
plugin authoring code
    |
    v
author-facing sdk
    |
    v
CLI / evaluator config
    |
    v
engine runtime builder
    |
    v
runtime module registrar
    |
    v
plugin host package
    |
    v
go-plugin gRPC transport
    |
    v
plugin subprocess
```

Each layer has one responsibility.

### 1. Plugin authors can use the SDK instead of the raw contract

`pkg/hashiplugin/sdk` is the new author-facing layer.

It provides:

- `sdk.MustModule(...)`
- `sdk.Function(...)`
- `sdk.Object(...)`
- `sdk.Method(...)`
- `sdk.MethodSummary(...)`
- `sdk.MethodDoc(...)`
- `sdk.MethodTags(...)`
- `sdk.Call`
- `sdk.Serve(...)`

This package does not replace `contract` or `shared`. It implements `contract.JSModule` on behalf of plugin authors and centralizes manifest building, invoke dispatch, and value conversion.

Method metadata is now intentionally split by role:

- `sdk.ExportDoc(...)` documents top-level function exports,
- `sdk.ObjectDoc(...)` documents object exports,
- `sdk.MethodSummary(...)` gives `goja-repl tui` and other compact UIs a one-line description,
- `sdk.MethodDoc(...)` provides the fuller method body,
- `sdk.MethodTags(...)` attaches lightweight classification labels for search and display.

### 2. Entry points choose whether plugins are enabled

The runtime is not globally plugin-aware by default. An entrypoint opts in by constructing a registrar and attaching it to the engine builder.

Current wired entrypoints:

- `cmd/goja-repl`
- `cmd/bun-demo`
- `pkg/repl/evaluators/javascript`

That means plugin support is explicit at composition time.

### 3. The engine provides the runtime-scoped seam

`engine.RuntimeModuleRegistrar` is the central extension seam. A registrar receives:

- the runtime context,
- the runtime-owned `require.Registry`,
- access to runtime cleanup registration through the runtime object.

This is the design point that makes plugins feasible without adding plugin-specific lifecycle hacks to the engine.

### 4. The host package owns policy

`pkg/hashiplugin/host` is the policy layer. It decides:

- how plugin binaries are discovered,
- which manifests are valid,
- how clients are launched,
- how modules are registered into the runtime,
- how plugin subprocesses are torn down.

This is intentionally separate from the transport contract so that future policy changes do not require redesigning the protobuf schema.

### 5. The transport package owns the shared wire contract

The shared `contract` and `shared` packages define:

- protobuf messages,
- gRPC service,
- `go-plugin` handshake constants,
- `GRPCPlugin` adapter helpers.

They are narrow on purpose. The contract should not accumulate host-only policy or engine-specific concepts.

That is also why the SDK belongs beside them rather than inside `host`: authoring ergonomics and host policy are different concerns.

## Runtime creation flow

This is the end-to-end flow from CLI flag to `require("plugin:echo")`.

```text
cmd/goja-repl tui
    |
    v
engine.NewBuilder()
    .WithModules(...)
    .WithRuntimeModuleRegistrars(host.NewRegistrar(...))
    |
    v
Factory.Build()
    |
    v
Factory.NewRuntime(ctx)
    |
    v
fresh require.Registry is created
    |
    v
static modules are registered
    |
    v
runtime registrars run
    |
    v
host registrar discovers and starts plugin clients
    |
    v
validated plugin manifests are reified as native modules
    |
    v
registry is enabled on the runtime
    |
    v
JavaScript code can call require("plugin:echo")
```

The order matters. The registrar phase must happen before `reg.Enable(vm)` because that is when native modules are registered into the runtime's `require` system.

## Key types and responsibilities

This section maps the main code objects to their jobs.

### `engine.RuntimeModuleRegistrar`

File: `engine/runtime_modules.go`

Purpose:

- lets code add runtime-scoped module registrations,
- runs after the fresh registry exists,
- runs before the registry is enabled on the VM.

Why it matters:

- plugin modules are not static global modules,
- a registrar can create per-runtime state safely,
- the seam is generic enough to reuse for other dynamic module sources later.

### `engine.Runtime.AddCloser`

File: `engine/runtime.go`

Purpose:

- lets runtime-integrated systems register cleanup callbacks,
- ensures cleanup is tied to runtime shutdown,
- runs close hooks in reverse registration order.

Why it matters:

- plugin subprocesses are external runtime resources,
- cleanup should not be hand-managed by CLI code,
- the same lifecycle pattern can support other runtime-owned resources later.

### `host.Config`

File: `pkg/hashiplugin/host/config.go`

Purpose:

- carries plugin discovery and client options.

Current defaults:

- pattern: `goja-plugin-*`
- namespace: `plugin:`
- startup timeout: `10s`
- call timeout: `5s`

Relevant policy fields:

- `Directories`
- `AllowModules`
- `Pattern`
- `Namespace`

This keeps user-facing configuration small while still centralizing policy.

### `host.Discover`

File: `pkg/hashiplugin/host/discover.go`

Purpose:

- find matching plugin binaries in configured directories,
- filter non-executable or invalid file candidates,
- deduplicate and sort results.

Why it matters:

- discovery should be deterministic,
- runtime creation should not depend on ambient PATH lookup,
- the trust boundary stays explicit at the directory level.

### `host.ValidateManifest`

File: `pkg/hashiplugin/host/validate.go`

Purpose:

- reject invalid module names,
- enforce namespace rules,
- validate export shape and uniqueness.

Why it matters:

- the manifest is the contract between a foreign process and the host runtime,
- validation must happen before registration,
- invalid plugins should fail fast rather than half-register.

### `host.LoadModule`

File: `pkg/hashiplugin/host/client.go`

Purpose:

- launch the plugin process through `go-plugin`,
- dispense the shared service,
- fetch and validate the manifest,
- keep the client handle for later invocation and cleanup.

Implementation notes:

- gRPC-only transport is used,
- stdout and stderr are suppressed by default,
- timeouts are applied to startup and calls,
- the loaded module retains both manifest and client.

### `host.RegisterModule`

File: `pkg/hashiplugin/host/reify.go`

Purpose:

- translate one validated manifest into one native module registration.

This is where the remote contract becomes an in-process `require()` module. For each manifest export, the host registers either:

- a Go function that forwards to `Invoke(...)`, or
- an object whose methods forward to `Invoke(...)`.

This is the most important conceptual bridge in the system: remote plugin exports are reified as local CommonJS exports.

### `host.NewRegistrar`

File: `pkg/hashiplugin/host/registrar.go`

Purpose:

- glue discovery, loading, validation, module reification, and cleanup registration into the engine seam.

In practice, this is the one object entrypoints need.

## Transport contract

The protobuf contract lives in `pkg/hashiplugin/contract/jsmodule.proto`.

At a high level, the service exposes:

- `GetManifest(...)`
- `Invoke(...)`

The manifest describes:

- module name,
- version,
- export list.

An export spec describes:

- export name,
- export kind,
- optional method list for object exports.

Invocation carries:

- export name,
- method name,
- arguments as `structpb.Value` entries.

Response carries:

- one `structpb.Value` result.

This contract is intentionally JSON-shaped. That keeps cross-process data handling simple and predictable at the cost of not trying to expose richer host-specific value types in v1.

## Value conversion path

When JavaScript calls a plugin-backed export, the conversion flow is:

```text
goja.Value arguments
    |
    v
arg.Export()
    |
    v
structpb.NewValue(...)
    |
    v
gRPC Invoke call
    |
    v
structpb.Value response
    |
    v
AsInterface()
    |
    v
vm.ToValue(...)
```

This path is simple, but it sets the practical constraints for plugin authors:

- keep values JSON-like,
- avoid expecting host object identity,
- avoid returning Goja-specific objects from plugin code.

## Integration in `goja-repl tui`

`cmd/goja-repl` now exposes the TUI through the `tui` subcommand. It resolves plugin directories directly from the shared root flags: explicit `--plugin-dir` flags win, otherwise the command scans `~/.go-go-goja/plugins/...`.

That means both the lower-level evaluator integration and the top-level TUI flag wiring are now present in:

- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/repl/adapters/bobatea/javascript.go`

The TUI entrypoint also uses the shared `replapi` runtime/session stack while keeping the Bobatea completion/help widgets.

It also exposes `--allow-plugin-module`, which is forwarded through the evaluator config into the host registrar.

## Tests and examples

The main end-to-end test is `pkg/hashiplugin/host/registrar_test.go`.

It covers:

- building a real plugin binary,
- loading `plugin:echo`,
- calling a function export,
- calling an object-method export,
- loading the SDK-authored `plugin:examples:greeter` example,
- loading the SDK-authored `plugin:examples:kv` example and verifying state survives across calls,
- loading the SDK-authored `plugin:examples:failing` example and verifying handler errors surface back to the caller,
- rejecting an invalid manifest,
- verifying subprocess shutdown on runtime close.

The user-facing example plugin sources currently live under:

- `plugins/examples/greeter`
- `plugins/examples/clock`
- `plugins/examples/validator`
- `plugins/examples/kv`
- `plugins/examples/system-info`
- `plugins/examples/failing`

The integration-test fixture plugins live under:

- `plugins/testplugin/echo`
- `plugins/testplugin/invalid`

This split is intentional. `plugins/examples/...` is for copyable authoring examples and documentation, while `plugins/testplugin/...` stays small and deterministic for integration tests.

The current state is intentionally mixed:

- `plugins/examples/greeter` now uses the richer SDK surface and is the primary authoring example,
- `plugins/examples/clock`, `validator`, `kv`, `system-info`, and `failing` expand the catalog so different SDK features are demonstrated in isolation,
- `plugins/testplugin/echo` also uses the SDK so integration tests exercise the real authoring path,
- `plugins/testplugin/invalid` remains handwritten so the suite still covers the raw contract path.

## Common extension points

This section explains where to make changes if the subsystem grows.

### Add more entrypoints

If you want plugin support in another runtime consumer:

1. find where that entrypoint builds an engine runtime or evaluator,
2. add plugin-directory configuration,
3. attach `host.NewRegistrar(...)`.

Do not reimplement discovery logic in the entrypoint.

### Strengthen trust policy

If you want checksums, allowlists, or richer trust policy:

- start in `host.Config`,
- enforce in discovery and validation before registration,
- keep the transport contract unchanged unless the plugin must publish new metadata.

### Extend the export model

If you want richer exported shapes:

1. extend the protobuf manifest schema,
2. update validation rules,
3. update reification logic,
4. add integration tests that require the new shape from JavaScript.

Do not skip the validation layer. Reification should assume the manifest is already trusted structurally.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Plugin support feels hard to trace across files | The feature spans engine, transport, host policy, and CLI wiring | Read the files in the order listed in the `Read this first` section |
| A new entrypoint cannot see plugins | The entrypoint builds a runtime but never attaches a registrar | Add `WithRuntimeModuleRegistrars(host.NewRegistrar(...))` or set `PluginDirectories` on the evaluator config |
| A plugin starts but registration fails | Manifest validation rejected the module shape | Start in `pkg/hashiplugin/host/validate.go` and compare the plugin manifest to the current rules |
| Runtime closes but a plugin process remains alive | Cleanup registration was skipped or a new integration path bypassed owned runtime shutdown | Confirm the runtime path uses `engine.Runtime` and registers closers through `AddCloser(...)` |
| A transport change breaks host code | Transport and host policy concerns got mixed together | Keep `contract` and `shared` narrow, and push policy back into `pkg/hashiplugin/host` |

## See Also

- `goja-repl help goja-plugin-user-guide` — User-facing reference for loading and calling plugins
- `goja-repl help plugin-tutorial-build-install` — Step-by-step plugin build and install walkthrough
- `goja-repl help repl-usage` — General REPL usage and command entrypoints
- `goja-repl help creating-modules` — In-process native module authoring, which is the closest existing parallel concept
