---
Title: JS Sandbox Host API and Runtime Architecture
Ticket: GOJA-17-JS-SANDBOX-API
Status: active
Topics:
    - goja
    - js-bindings
    - architecture
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Runtime factory lifecycle and runtime-scoped owner setup
    - Path: engine/runtime.go
      Note: Runtime bootstrap and blank-import module registration pattern
    - Path: engine/runtime_modules.go
      Note: Runtime-scoped module registrar seam for host-owned sandbox state
    - Path: pkg/jsverbs/command.go
      Note: Current Glazed compilation path that the sandbox API should stay distinct from
    - Path: pkg/jsverbs/runtime.go
      Note: Current JS invocation and module-loader precedent for runtime-owned execution
    - Path: pkg/jsverbs/scan.go
      Note: Current JS discovery and sentinel-parsing precedent
    - Path: pkg/runtimebridge/runtimebridge.go
      Note: VM-local bindings used to keep host state attached to a runtime
    - Path: pkg/runtimeowner/runner.go
      Note: Owner-thread scheduler pattern for safe JS<->Go coordination
ExternalSources: []
Summary: Architecture and implementation guide for a CommonJS JS sandbox module that makes go-go-goja hosts scriptable.
LastUpdated: 2026-04-20T11:10:00-04:00
WhatFor: Explain the runtime model, JS API shape, memory-store strategy, and file-level implementation plan for a scriptable bot sandbox.
WhenToUse: Use when implementing or reviewing the sandbox host API or any JS-scriptable bot surface built on go-go-goja.
---



# JS Sandbox Host API and Runtime Architecture

## Executive Summary

The goal of this ticket is to make a `go-go-goja`-based host scriptable from JavaScript through a small, elegant sandbox API. In concrete terms, a host application should be able to load a JS file, give it a capability-oriented API, and let that script register commands and event handlers without the host needing to hard-code every behavior in Go.

The recommended design is to add a runtime-scoped sandbox module that exposes a single JS entrypoint, `require("sandbox")`, plus a small set of definition helpers and runtime context objects. Scripts should define behavior in CommonJS, for example:

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command, event }) => {
  command("ping", async (ctx) => {
    await ctx.reply("pong")
  })

  event("ready", async (ctx) => {
    ctx.log.info("bot ready", { user: ctx.me.username })
  })
})
```

This document is intentionally intern-friendly. It explains what already exists in the repo, why the new API belongs in a runtime-scoped host layer instead of a static command scanner, how an in-memory store should work, how the host and script should talk to each other, and how the API can grow later without forcing a rewrite.

## Problem Statement

`go-go-goja` already has strong building blocks for JavaScript execution:

- a runtime factory and lifecycle owner in `engine/factory.go` and `engine/runtime.go`
- a runtime-owned scheduling bridge in `pkg/runtimebridge`
- an owner-thread runner in `pkg/runtimeowner`
- native module registration through `modules.NativeModule`
- runtime-scoped module registration hooks through `engine.RuntimeModuleRegistrar`

It also already has a sophisticated JS-to-Glazed command pipeline in `pkg/jsverbs`, where scanned JavaScript functions are compiled into normal Glazed commands and then executed through Goja.

The missing piece is a host-facing sandbox API that feels natural to script authors. The current `pkg/jsverbs` subsystem is excellent for turning JS files into CLI commands, but it is not the same thing as a bot scripting surface. A bot scripting API needs to be:

- capability-oriented rather than command-scanner-oriented,
- runtime-scoped rather than global,
- small enough for a new contributor to understand quickly,
- friendly to in-memory state initially,
- explicit about what is available from JS and when.

In short, we need a script-author API that lets someone think in terms of `command`, `event`, `store`, `log`, and `reply`, while the host keeps the actual Goja/runtime/dispatcher machinery hidden.

## Current-State Architecture

The important thing to understand is that the repository already solves most of the hard runtime problems.

### Engine lifecycle

`engine.NewBuilder(...)` composes a runtime plan, validates it, and freezes it into an immutable factory. `Factory.NewRuntime(...)` then creates a VM, an event loop, a `runtimeowner.Runner`, a `require.Registry`, and runtime-scoped values (`engine/factory.go:154-230`). The factory stores runtime bindings in `pkg/runtimebridge` so modules can find the current VM, owner, and lifecycle context (`engine/factory.go:183-187`, `pkg/runtimebridge/runtimebridge.go:12-52`).

That matters because a sandbox API should not create its own hidden VM. It should plug into the same lifecycle that the rest of the repo already uses.

### Native modules and runtime registrars

There are two important extension seams:

- `modules.NativeModule` for reusable native module definitions (`modules/common.go:9-32`, `modules/common.go:86-117`)
- `engine.RuntimeModuleRegistrar` for runtime-scoped registration work (`engine/runtime_modules.go:12-25`)

That is the right place for a sandbox host API. A sandbox is not just a static native module with no state. It needs host-owned runtime state such as in-memory storage and an event dispatcher. A runtime registrar can build that state per VM and then register the JS module against the current runtime.

### Existing scriptable-command precedent

The `pkg/jsverbs` package already shows a full JS-to-command pipeline:

- static discovery in `pkg/jsverbs/scan.go`
- parameter/field binding in `pkg/jsverbs/binding.go`
- Glazed command construction in `pkg/jsverbs/command.go`
- JS invocation in `pkg/jsverbs/runtime.go`
- a runnable example host in `cmd/jsverbs-example/main.go`
- intern-friendly docs in `pkg/doc/10-jsverbs-example-developer-guide.md` and `pkg/doc/11-jsverbs-example-reference.md`

This is valuable precedent, but it solves a different problem. `jsverbs` is about scanning JavaScript into CLI verbs. The sandbox API in this ticket is about exposing host capabilities to JavaScript scripts that want to behave like bots.

### Async and owner-thread safety

When JS needs to touch Go-owned state after an async delay, the repo already has the right pattern. `pkg/runtimeowner/runner.go` provides `Call` and `Post` methods that marshal work back to the owner thread, and `README.md` explains Promise settlement from the owner thread with a timer example. That pattern is the correct mental model for any future async sandbox helper such as HTTP fetch, delayed jobs, or database calls.

For the first version of the sandbox API, we do **not** need to solve permissions or persistent storage. We only need a stable in-memory host surface. That keeps the design focused.

## Proposed Solution

### Recommendation in one sentence

Add a runtime-scoped `sandbox` module that exposes a small CommonJS bot-definition DSL plus an in-memory store and runtime context helpers.

### Design goals

The sandbox API should:

- feel small and obvious in JavaScript,
- separate definition time from runtime time,
- expose only the capabilities the host intentionally registered,
- keep all state in memory for v1,
- avoid a permission system in v1,
- allow future async helpers without changing the core API shape.

### Proposed JS entrypoint

Use CommonJS and a single sandbox module:

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command, event }) => {
  command("ping", async (ctx) => {
    await ctx.reply("pong")
  })

  event("ready", async (ctx) => {
    ctx.log.info("ready", { user: ctx.me.username })
  })
})
```

Why CommonJS?

- `go-go-goja` already uses `require()` as the primary module mechanism.
- `require()` keeps relative imports predictable with the existing module-root helpers.
- CommonJS keeps the runtime contract narrow and familiar for current code.

### Recommended API shape

The API should be capability-based and builder-oriented.

#### Module exports

`require("sandbox")` should export:

- `defineBot(builderFn)`
- optional helper constructors such as `command(...)` and `event(...)` if the project wants a more declarative style later

#### Builder helpers

The builder callback should receive a small helper object:

- `command(name, spec, handler)`
- `event(name, handler)`
- `configure(options)`
- `shared(name, value)` or similar only if needed later

#### Runtime context

Every handler should receive a `ctx` object with stable methods and data:

- `ctx.args` — parsed arguments for the command
- `ctx.command` — command metadata
- `ctx.user` — the invoking user
- `ctx.guild` — guild context when present
- `ctx.channel` — channel context when present
- `ctx.me` — the bot identity
- `ctx.reply(textOrPayload)`
- `ctx.defer()`
- `ctx.log.info/debug/warn/error(...)`
- `ctx.store.get(key, defaultValue)`
- `ctx.store.set(key, value)`
- `ctx.store.delete(key)`
- `ctx.store.keys(prefix?)`

The important rule is that `ctx` should feel like a focused capability object, not a raw escape hatch into the entire Go runtime.

### Why this API shape works

It separates three concerns that often get mixed together:

1. **Definition** — the bot author declares commands and event handlers.
2. **Invocation** — the host calls a specific handler with a runtime context.
3. **State** — the bot reads and writes host-owned memory through a small store API.

That means a script author can write small, testable pieces of logic, while the host retains control over lifecycle, dispatch, and actual Discord/network operations.

## Runtime Architecture

### Architecture diagram

```mermaid
flowchart TD
  A[JS script file] --> B[require("sandbox")]
  B --> C[defineBot / command / event helpers]
  C --> D[BotSpec in memory]
  D --> E[Host dispatcher]
  E --> F[engine.Factory.NewRuntime]
  F --> G[goja runtime + require registry]
  F --> H[runtimeowner.Runner]
  F --> I[runtimebridge bindings]
  I --> J[in-memory store]
  E --> K[interaction / event handlers]
  K --> L[ctx.reply / ctx.defer / ctx.log]
```

### Runtime flow

1. The host creates an `engine.Factory`.
2. The factory installs the modules it wants and creates a runtime.
3. A runtime registrar builds a sandbox host state object for that VM.
4. The registrar registers the `sandbox` module against the runtime’s `require.Registry`.
5. The host loads the JS file with `require(...)`.
6. The JS file returns a bot definition.
7. The host compiles that definition into a dispatch table.
8. When Discord or another event source fires, the host calls the matching JS handler through `runtimeowner.Runner`.

### Why runtime-scoped registration is the right seam

A sandbox host has VM-local state. The in-memory store, dispatcher, and runtime lifecycle belong to one runtime, not to the process globally. That makes `engine.RuntimeModuleRegistrar` a better fit than a global `modules.NativeModule` by itself.

In practical terms:

- `modules.NativeModule` is the shape of the JS-facing export.
- `engine.RuntimeModuleRegistrar` is the place where a particular runtime gets its own store and dispatcher.
- `pkg/runtimebridge` and `pkg/runtimeowner` ensure the JS code and the Go host can safely coordinate across goroutines.

## In-Memory Store Model

### What the store is

For v1, the store is just an in-memory key/value service with optional namespacing. There is no persistence layer, no encryption layer, and no permissions layer.

That is a deliberate simplification. The host should be able to prove the scripting model first, and only later decide whether state should survive restarts.

### Suggested Go shape

```go
type MemoryStore struct {
    mu   sync.RWMutex
    data map[string]any
}

func (s *MemoryStore) Get(key string, defaultValue any) any
func (s *MemoryStore) Set(key string, value any)
func (s *MemoryStore) Delete(key string)
func (s *MemoryStore) Keys(prefix string) []string
func (s *MemoryStore) Namespace(parts ...string) *MemoryStore
```

### Suggested JS shape

```js
const count = ctx.store.get("count", 0)
ctx.store.set("count", count + 1)
```

This API can be synchronous because the backing store is in memory. If the project later needs persistence, the same JS API can still be `await`-friendly if the host decides to return promises later.

### Namespacing recommendation

The most useful namespacing scheme is probably:

- global script state
- guild-scoped state
- channel-scoped state
- user-scoped state

That lets a bot author solve common cases without inventing their own key format every time.

Example:

```js
const hits = ctx.store.namespace("guild", ctx.guild.id)
const current = hits.get("commandCount", 0)
hits.set("commandCount", current + 1)
```

## Host-Side Bot Dispatch Model

The host should treat the script as a registration object, not as a free-for-all source of arbitrary runtime mutation.

### Pseudocode for host bootstrap

```go
factory, _ := engine.NewBuilder(
    engine.WithModules(engine.DefaultRegistryModules()),
    engine.WithRuntimeModuleRegistrars(
        sandbox.NewRegistrar(hostConfig),
    ),
    engine.WithRequireOptions(
        require.WithLoader(scriptLoader),
        engine.WithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions()),
    ),
).Build()

runtime, _ := factory.NewRuntime(ctx)
defer runtime.Close(ctx)

spec, _ := runtime.Require.Require("./bot.js")
compiled, _ := sandbox.Compile(spec)

host := sandbox.NewHost(compiled, runtime)
```

### Event dispatch flow

```text
Discord event
  -> host finds matching JS handler
  -> host creates ctx
  -> runtime.Owner.Call(...) or Post(...)
  -> handler may call ctx.reply / ctx.store / ctx.log
  -> host converts result into reply/side effects
```

### Why not expose the raw goja runtime

A raw runtime is too much power for script authors and too little structure for host maintainers. The sandbox API should define the supported affordances explicitly:

- what can be registered,
- what the handler context contains,
- what storage and logging look like,
- what the host guarantees about calling convention.

That keeps the surface area understandable.

## Design Decisions

### 1) Use a dedicated sandbox module

The sandbox should be a distinct module name, not a miscellaneous bucket of globals.

Reason:

- `require("sandbox")` is discoverable.
- the module boundary makes the API easy to document and test.
- the host can keep the module set intentionally small.

### 2) Keep v1 permission-free

Do not add a permissions policy in the first version.

Reason:

- the user explicitly asked to avoid that complexity,
- the host can already control capabilities by choosing which modules to register,
- the minimal model is easier for a new contributor to understand.

### 3) Keep storage in memory

Start with in-memory storage only.

Reason:

- it makes the API easy to reason about,
- it avoids persistence design questions too early,
- it lets scripts prove behavior without file/database dependencies.

### 4) Prefer runtime registrars over global state

The sandbox host state should be created per runtime.

Reason:

- runtimes are already first-class objects in `engine.Factory`,
- per-runtime state avoids cross-bot bleed and test interference,
- this is a better fit for the existing `runtimebridge` and `runtimeowner` model.

### 5) Keep the API capability-based

Expose only the helpers the host wants scripts to use.

Reason:

- a capability-based API is easier to audit,
- it avoids forcing scripts to know about Go internals,
- it keeps future changes localized.

### 6) Treat `pkg/jsverbs` as precedent, not as the sandbox itself

`pkg/jsverbs` already demonstrates JS discovery, binding, and runtime execution. It is useful precedent, but the sandbox host API should not be welded onto the command-scanning pipeline.

Reason:

- `jsverbs` answers “what commands exist in source?”
- the sandbox answers “what host abilities can a script use?”
- those are related, but different, problems.

## Examples of Bots

### 1) Ping / health bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command }) => {
  command("ping", async (ctx) => {
    await ctx.reply("pong")
  })
})
```

Use case:

- quick health check
- confirms command registration and reply flow

### 2) Welcome bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ event }) => {
  event("guildMemberAdd", async (ctx) => {
    await ctx.reply(`Welcome, ${ctx.user.mention}!`)
  })
})
```

Use case:

- greeting new members
- lightweight onboarding for community servers

### 3) Counter bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command }) => {
  command("count", async (ctx) => {
    const n = ctx.store.get("count", 0)
    ctx.store.set("count", n + 1)
    await ctx.reply(`Count is now ${n + 1}`)
  })
})
```

Use case:

- prove in-memory storage works
- show that script state survives multiple invocations in one runtime

### 4) FAQ bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command }) => {
  command("faq", {
    options: {
      topic: { type: "choice", values: ["rules", "install", "help"] },
    },
  }, async (ctx) => {
    const faqs = {
      rules: "Be respectful.",
      install: "Run the install command from the repo.",
      help: "Open the help channel or use /help.",
    }

    await ctx.reply(faqs[ctx.args.topic])
  })
})
```

Use case:

- static knowledge bot
- good for support or onboarding content

## Alternatives Considered

### 1) Reuse `pkg/jsverbs` directly as the host API

Rejected for v1 because `jsverbs` is built around source discovery and Glazed command compilation, not a bot-host capability model.

### 2) Expose the full Goja runtime to scripts

Rejected because that would make the sandbox harder to reason about and harder to document.

### 3) Add a permissions engine now

Rejected because the user explicitly asked to skip that and because the host can control capabilities through module registration already.

### 4) Make the store persistent immediately

Rejected because it adds a large design surface without improving the first proof of the sandbox API.

## Implementation Plan

### Phase 1: Define the contract

- Decide the module name (`sandbox`).
- Define the JS builder API and `ctx` shape.
- Write tests for the pure data model.

### Phase 2: Add the runtime-scoped host

- Add a runtime registrar that creates one sandbox host per VM.
- Store a host state object in `engine.RuntimeModuleContext.Values` or an equivalent runtime-owned structure.
- Register the `sandbox` native module against the runtime’s `require.Registry`.

### Phase 3: Implement the in-memory store and dispatch table

- Add the memory store.
- Add a command/event registry.
- Add a dispatcher that uses `runtimeowner.Runner` for execution.

### Phase 4: Wire the host into a demo application

- Add a thin command or host harness that loads a bot script.
- Use `engine.WithModuleRootsFromScript(...)` so local script imports keep working.
- Verify a minimal bot script can register a command and reply.

### Phase 5: Document and validate

- Add runtime tests for `defineBot`, `command`, `event`, `ctx.reply`, and `ctx.store`.
- Add doc examples to the help system.
- Run `go test ./...` and a manual smoke test script.

## Open Questions

- Should the first API expose `command` and `event` as builder methods, helper constructors, or both?
- Should `ctx.reply` be strictly Discord-centric, or should the host also expose a more generic `emit()` helper for future transports?
- Should the store namespace model be explicit (`ctx.store.namespace(...)`) or implicit (`ctx.guildStore`, `ctx.userStore`)?
- When the project eventually adds persistence, should the API keep sync-looking methods or move to promise-returning store calls?

## References

- `engine/factory.go`
- `engine/runtime.go`
- `engine/module_specs.go`
- `engine/runtime_modules.go`
- `engine/module_roots.go`
- `modules/common.go`
- `pkg/runtimebridge/runtimebridge.go`
- `pkg/runtimeowner/runner.go`
- `pkg/jsverbs/scan.go`
- `pkg/jsverbs/binding.go`
- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`
- `cmd/jsverbs-example/main.go`
- `pkg/doc/10-jsverbs-example-developer-guide.md`
- `pkg/doc/11-jsverbs-example-reference.md`
- `README.md`
