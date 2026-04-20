---
Title: JS Sandbox API Reference and Example Bots
Ticket: GOJA-17-JS-SANDBOX-API
Status: active
Topics:
    - goja
    - js-bindings
    - architecture
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Native module and async runtime patterns that shape the sandbox host API
    - Path: cmd/jsverbs-example/main.go
      Note: Example host entrypoint for current JS command discovery and Glazed wiring
    - Path: pkg/doc/10-jsverbs-example-developer-guide.md
      Note: Detailed precedent for the current JS-to-Glazed pipeline
    - Path: pkg/doc/11-jsverbs-example-reference.md
      Note: Compact reference for the current JS command pipeline and runtime rules
    - Path: pkg/jsverbs/command.go
      Note: Command shape and result rendering rules relevant to script authors
    - Path: pkg/jsverbs/runtime.go
      Note: Runtime-owned execution and Promise-waiting precedent
ExternalSources: []
Summary: Copy/paste-friendly reference for the proposed sandbox module, runtime context, and example bot scripts.
LastUpdated: 2026-04-20T11:10:00-04:00
WhatFor: Give maintainers and future implementers a compact API cheat sheet for the scriptable JS sandbox.
WhenToUse: Use when sketching bot scripts or checking the intended JS-facing surface of the sandbox host API.
---


# JS Sandbox API Reference and Example Bots

## Goal

This reference is the short-form companion to the design guide. It collects the proposed JS-facing API in one place so an implementer can quickly see the intended names, expected runtime context, and a few representative bot scripts.

## Context

The sandbox API is meant for CommonJS scripts loaded through `go-go-goja`. It is intentionally small in v1:

- no permission system,
- in-memory storage only,
- runtime-scoped host state,
- `require("sandbox")` as the entrypoint.

The host decides which capabilities are registered. Scripts only see the helpers the host intentionally exposes.

## Quick Reference

### Entry point

```js
const { defineBot } = require("sandbox")
```

### Definition API

| API | Purpose | Notes |
| --- | --- | --- |
| `defineBot(builderFn)` | Declare the bot script | Returns a bot definition object the host can compile |
| `command(name, spec, handler)` | Register a command | `spec` is optional for simple commands |
| `event(name, handler)` | Register an event handler | Examples: `ready`, `guildMemberAdd`, `messageCreate` |
| `configure(options)` | Set host-independent metadata | Optional in v1 |

### Runtime context

| `ctx` member | Purpose |
| --- | --- |
| `ctx.args` | Parsed command arguments |
| `ctx.command` | Command metadata |
| `ctx.user` | Invoking user |
| `ctx.guild` | Current guild, if any |
| `ctx.channel` | Current channel, if any |
| `ctx.me` | Bot identity |
| `ctx.reply(...)` | Send a reply or response |
| `ctx.defer()` | Defer the current response |
| `ctx.log.info/debug/warn/error(...)` | Structured logging |
| `ctx.store.get(key, defaultValue)` | Read in-memory state |
| `ctx.store.set(key, value)` | Write in-memory state |
| `ctx.store.delete(key)` | Delete a value |
| `ctx.store.keys(prefix?)` | Enumerate stored keys |

### In-memory store rules

- Storage is process-local and runtime-local.
- Keys may be namespaced by guild, channel, or user.
- Methods are synchronous because the backing store is in memory.
- The API is still `await`-friendly if the host later makes some operations asynchronous.

### Host bootstrap sketch

```go
factory, err := engine.NewBuilder(
    engine.WithModules(engine.DefaultRegistryModules()),
    engine.WithRuntimeModuleRegistrars(sandbox.NewRegistrar(hostConfig)),
    engine.WithRequireOptions(
        engine.WithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions()),
    ),
).Build()
```

## Usage Examples

### 1) Ping bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command }) => {
  command("ping", async (ctx) => {
    await ctx.reply("pong")
  })
})
```

### 2) Welcome bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ event }) => {
  event("guildMemberAdd", async (ctx) => {
    await ctx.reply(`Welcome, ${ctx.user.mention}!`)
  })
})
```

### 3) Counter bot using in-memory storage

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ command }) => {
  command("count", async (ctx) => {
    const count = ctx.store.get("count", 0)
    ctx.store.set("count", count + 1)
    await ctx.reply(`Count is now ${count + 1}`)
  })
})
```

### 4) FAQ bot with a typed option

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
      install: "Use the install command from the repo.",
      help: "Ask in the support channel.",
    }

    await ctx.reply(faqs[ctx.args.topic])
  })
})
```

### 5) A tiny logging-only event bot

```js
const { defineBot } = require("sandbox")

module.exports = defineBot(({ event }) => {
  event("ready", async (ctx) => {
    ctx.log.info("bot ready", {
      user: ctx.me.username,
      guild: ctx.guild?.id,
    })
  })
})
```

## Bot patterns

### Hello bot

Use this when you want to prove the script runtime is alive.

```js
module.exports = defineBot(({ command }) => {
  command("hello", async (ctx) => {
    await ctx.reply("hello from JS")
  })
})
```

### State bot

Use this when you want to prove the in-memory store works.

```js
module.exports = defineBot(({ command }) => {
  command("increment", async (ctx) => {
    const current = ctx.store.get("hits", 0)
    ctx.store.set("hits", current + 1)
    await ctx.reply(`hits=${current + 1}`)
  })
})
```

### Community bot

Use this when the bot acts as a lightweight helper or FAQ surface.

```js
module.exports = defineBot(({ command, event }) => {
  command("faq", async (ctx) => {
    await ctx.reply("Use /help for the command list.")
  })

  event("guildMemberAdd", async (ctx) => {
    await ctx.reply(`Welcome to the server, ${ctx.user.name}!`)
  })
})
```

## Notes for implementers

- Keep the module name short and obvious. `sandbox` is the recommended v1 name.
- Keep commands and events explicit.
- Keep the store synchronous in v1.
- Do not add a permissions layer in v1.
- Use `runtimeowner.Runner` for any host-to-JS scheduling that must occur on the owner thread.
- Use `engine.WithModuleRootsFromScript(...)` when the bot script imports local helpers.

## Related

- `design-doc/01-js-sandbox-host-api-and-runtime-architecture.md`
- `engine/factory.go`
- `engine/runtime.go`
- `pkg/runtimeowner/runner.go`
- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`
- `cmd/jsverbs-example/main.go`
