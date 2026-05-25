---
Title: xgoja HTTP Express Provider and Discord Outbound Runtime API
Ticket: XGOJA-011
Status: active
Topics:
  - xgoja
  - goja
  - providers
  - fs
  - architecture
  - command-registration
  - goja-nodejs
  - modules
  - runtime
  - web-ui
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles:
  - /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/modules/express/express.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/gojahttp/host.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/capabilities.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/commands.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/internal/jsdiscord/runtime.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider.go
ExternalSources: []
Summary: Intern-oriented design and implementation guide for mounting Express through xgoja and letting Discord bot JavaScript send messages from HTTP route handlers through a non-Express-specific outbound Discord API.
LastUpdated: 2026-05-25T12:30:00-04:00
---

# xgoja HTTP Express Provider and Discord Outbound Runtime API

## 1. Goal

We want a generated xgoja binary to run a real Discord JavaScript bot and also mount an HTTP/Express module into the same bot runtime. The JavaScript bot should be able to register an HTTP endpoint with a small UI, and that endpoint should make the Discord bot send a message.

The desired authoring experience is:

```js
const { defineBot } = require("discord")
const discord = require("discord")
const express = require("express")

const app = express.app()

app.get("/", (req, res) => {
  res.html(`
    <form method="POST" action="/say">
      <input name="channelId" placeholder="Discord channel ID" />
      <input name="message" placeholder="Message" />
      <button type="submit">Send</button>
    </form>
  `)
})

app.post("/say", async (req, res) => {
  await discord.channels.send(req.body.channelId, { content: req.body.message })
  res.json({ ok: true })
})

module.exports = defineBot(({ command, configure }) => {
  configure({ name: "http-say", description: "Discord bot with xgoja-mounted HTTP UI" })

  command("ping", async () => ({ content: "pong from xgoja" }))
})
```

The key architectural rule is:

> `discord-bot` must not know about Express. xgoja/go-go-goja owns the HTTP/Express module and server lifecycle. `discord-bot` only exposes a Discord outbound API that any JavaScript callback can use.

## 2. Existing system pieces

### 2.1 xgoja generated binaries

xgoja builds a Go binary from an `xgoja.yaml` file. The spec selects provider packages, runtime profiles, built-in commands, and provider-owned command sets.

Relevant files:

- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/pkg/xgoja/app/command_providers.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- `go-go-goja/pkg/xgoja/providerapi/commands.go`

A runtime profile selects modules:

```yaml
runtimes:
  bot:
    modules:
      - package: discord-bot
        name: discord
        as: discord
      - package: go-go-goja-host
        name: fs
        as: fs
        config:
          allow: true
```

Each module is a provider entry:

```go
type Module struct {
    Name        string
    DefaultAs   string
    Description string
    New         func(ModuleContext) (require.ModuleLoader, error)
}
```

That is enough for simple modules such as `fs`, because they are just `require()` loaders.

### 2.2 The current Express module

The current Express implementation is in:

- `go-go-goja/modules/express/express.go`
- `go-go-goja/pkg/gojahttp/host.go`

It is not just a plain loader. It is a runtime registrar around a `gojahttp.Host`:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{})
registrar := express.NewRegistrar(host)
```

The registrar does two important things:

1. Registers `require("express")`.
2. Connects the HTTP host to the current runtime owner so HTTP handlers can safely call JavaScript on the Goja owner thread.

The HTTP server itself is also not inside the JS module. Go must start it:

```go
http.ListenAndServe(":8787", host)
```

That is why Express is currently **not cleanly xgoja-mountable** as a normal provider module.

### 2.3 Discord bot JavaScript runtime

The Discord bot runtime lives in:

- `discord-bot/internal/jsdiscord/runtime.go`
- `discord-bot/internal/jsdiscord/host.go`
- `discord-bot/internal/jsdiscord/host_dispatch.go`
- `discord-bot/internal/jsdiscord/bot_ops.go`
- `discord-bot/pkg/xgoja/provider/provider.go`

The JS module `require("discord")` currently exposes bot definition helpers such as:

```js
const { defineBot } = require("discord")
```

Within a Discord dispatch callback, the JS receives a `ctx` object with Discord operations:

```js
command("announce", async (ctx) => {
  await ctx.discord.channels.send("channel-id", { content: "hello" })
})
```

Those operations are built from a `discordgo.Session` during actual Discord dispatch.

But an Express route handler is not a Discord dispatch callback. It does not receive `ctx.discord`. Therefore a route handler needs a top-level/session-bound Discord API:

```js
const discord = require("discord")
await discord.channels.send(channelId, { content: message })
```

That API should be independent of Express. It is just an outbound Discord client API for any JavaScript callback in the live bot runtime.

## 3. Target architecture

```text
+---------------------------+
| generated xgoja binary    |
|                           |
| runtime profile: bot      |
|  - discord-bot.discord    |
|  - discord-bot.ui         |
|  - go-go-goja-host.fs     |
|  - go-go-goja-http.express|
|                           |
| commandProviders:         |
|  - discord-bot.bots       |
+-------------+-------------+
              |
              | creates runtime using selected profile
              v
+---------------------------+
| engine.Runtime            |
|                           |
| require("discord")        |
| require("ui")             |
| require("fs")             |
| require("express")        |
+------+------+-------------+
       |      |
       |      +----------------------+
       |                             |
       v                             v
+--------------+             +----------------+
| Discord API  |             | gojahttp.Host  |
| session-bound|             | HTTP routes    |
+------+-------+             +-------+--------+
       |                             |
       | discordgo.Session           | net/http server
       v                             v
+--------------+             +----------------+
| Discord      |             | Browser / curl |
+--------------+             +----------------+
```

Responsibilities:

- xgoja/go-go-goja owns HTTP/Express module registration and HTTP server lifecycle.
- discord-bot owns Discord session binding and outbound Discord operations.
- The generated app composes both by selecting modules in one runtime profile.
- The JavaScript bot script composes both with `require("discord")` and `require("express")`.

## 4. Required changes

### 4.1 Add xgoja HTTP provider

Add package:

```text
go-go-goja/pkg/xgoja/providers/http
```

It should register package ID:

```go
const PackageID = "go-go-goja-http"
```

It should expose module:

```yaml
- package: go-go-goja-http
  name: express
  as: express
```

This provider must own:

- a `gojahttp.Host` per runtime;
- an Express loader/registrar for that host;
- a Glazed section for listen parameters;
- HTTP server startup and shutdown.

### 4.2 Export or adapt Express loader

The current `modules/express` package exposes `NewRegistrar(host)`, but the xgoja provider module API expects a loader.

Two reasonable options:

#### Option A: export `NewLoader`

Add to `modules/express`:

```go
func NewLoader(host *gojahttp.Host, opts ...Option) require.ModuleLoader {
    registrar := NewRegistrar(host, opts...)
    return registrar.Loader
}
```

But `loader` is currently unexported. We can either export `Loader` on the registrar or add `NewLoader` inside the package.

#### Option B: let the provider call `RegisterRuntimeModule`

The provider runtime initializer can call:

```go
express.NewRegistrar(host).RegisterRuntimeModule(ctx, reg)
```

This needs access to the runtime require registry and owner. xgoja's current `RuntimeHandle` only exposes `Runtime() *goja.Runtime` and `Close(...)`. Therefore option A is less invasive for the first pass.

Recommendation: **Option A**.

### 4.3 Add provider capability for listen parameters

The user requested a Glazed section for listen parameters.

The HTTP provider should expose a section, for example slug `http` and prefix `http-`:

```bash
--http-listen :8787
--http-enabled true
```

Pseudocode:

```go
type HTTPSettings struct {
    Enabled bool   `glazed:"enabled"`
    Listen  string `glazed:"listen"`
}

type Capability struct{}

func (Capability) CapabilityID() string { return "go-go-goja-http.config" }

func (Capability) ConfigSections(ctx providerapi.SectionContext) ([]schema.Section, error) {
    return []schema.Section{
        schema.NewSection("http",
            schema.WithPrefix("http-"),
            schema.WithBoolField("enabled", fields.WithDefault(true)),
            schema.WithStringField("listen", fields.WithDefault(":8787")),
        ),
    }, nil
}
```

### 4.4 Start and stop HTTP server

The HTTP provider needs runtime lifecycle integration.

Existing `engine.Runtime` has:

```go
AddCloser(func(context.Context) error) error
```

But `providerapi.RuntimeHandle` does not currently expose `AddCloser`. Do not break the interface. Add an optional interface:

```go
type RuntimeCloserRegistry interface {
    AddCloser(func(context.Context) error) error
}
```

Then the HTTP provider can do:

```go
if closer, ok := handle.(providerapi.RuntimeCloserRegistry); ok {
    closer.AddCloser(func(ctx context.Context) error {
        return server.Shutdown(ctx)
    })
}
```

The app-side runtime handle should implement this by delegating to `engine.Runtime.AddCloser`.

### 4.5 Share host between loader and initializer

The Express loader creates or uses a `gojahttp.Host`. The runtime initializer starts the HTTP server for that same host.

A practical first-pass provider can keep a runtime-keyed map:

```go
var hosts sync.Map // *goja.Runtime -> *gojahttp.Host

func expressLoader(vm *goja.Runtime, module *goja.Object) {
    host := gojahttp.NewHost(...)
    hosts.Store(vm, host)
    express.NewLoader(host)(vm, module)
}

func InitRuntimeFromSections(..., handle RuntimeHandle, vals *values.Values) error {
    vm := handle.Runtime()
    raw, ok := hosts.Load(vm)
    if !ok { return nil } // express not required yet
    host := raw.(*gojahttp.Host)
    server := &http.Server{Addr: settings.Listen, Handler: host}
    go server.ListenAndServe()
    closer.AddCloser(server.Shutdown)
}
```

Caveat: the loader only runs after JavaScript calls `require("express")`. If the initializer runs before the script has required express, no host exists yet.

Therefore the better first-pass design is:

- `Module.New(...)` creates a per-runtime-ish shared holder object captured by the loader closure;
- loader initializes the holder's host;
- runtime initializer can also initialize the holder's host before `require("express")` if necessary.

But because `Module.New(...)` is called during runtime module registration, not with the actual VM, a simple captured holder works:

```go
type runtimeHTTPModule struct {
    mu   sync.Mutex
    host *gojahttp.Host
}

func (m *runtimeHTTPModule) Host() *gojahttp.Host { ... }
func (m *runtimeHTTPModule) Loader(vm *goja.Runtime, module *goja.Object) { express.NewLoader(m.Host())(...) }
```

The capability still needs to find that holder. Since capabilities are package-level and not instance-level today, the provider can track holders by runtime once loader executes. If we need server startup before the first `require("express")`, extend providerapi later. For the bot script use case, script startup requires express immediately, so route registration happens before users hit the endpoint.

Acceptable first pass:

- server starts in the express loader when `require("express")` is called;
- listen settings are prepared by the runtime initializer and stored for the runtime;
- if settings are missing, use defaults/module config.

### 4.6 Apply module sections to command providers

XGOJA-008 built-ins aggregate module sections. Provider-owned commands currently must aggregate selected module sections themselves.

The `discord-bot` command provider should:

1. Inspect `CommandSetContext.SelectedModules`.
2. Ask `ConfigSectionCapability` modules for sections.
3. Add those sections to each bot run command.
4. Capture parsed values during `Run(...)`.
5. Use those values when creating xgoja runtimes for bot host execution.
6. Call `RuntimeInitializerCapability.InitRuntimeFromSections(...)` after `factory.NewRuntime(...)`.

Pseudocode:

```go
type initializingRuntimeFactory struct {
    xgojaFactory xgojaRuntimeFactory
    selected []providerapi.ModuleDescriptor
    currentValues atomic.Value // *values.Values
}

func (f *initializingRuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error) {
    rt, err := f.xgojaFactory.NewRuntime(ctx, profile, opts...)
    if err != nil { return nil, err }
    vals, _ := f.currentValues.Load().(*values.Values)
    initRuntimeFromSelectedModules(ctx, vals, rt, f.selected)
    return rt, nil
}
```

The wrapper command should set `currentValues` while running:

```go
func (c wrappedCommand) Run(ctx context.Context, vals *values.Values) error {
    c.factory.SetValues(vals)
    defer c.factory.ClearValues()
    return c.inner.Run(ctx, vals)
}
```

This keeps xgoja generic and lets package-owned command providers own package-specific composition.

### 4.7 Add Discord top-level outbound API

`discord-bot` should expose a non-Express-specific runtime API:

```js
const discord = require("discord")
await discord.channels.send(channelId, { content: "hello" })
```

Implementation shape:

- `RuntimeState` stores an optional outbound `*DiscordOps`.
- `RuntimeState.Loader` exports `channels`, `messages`, etc. from `discordOpsObject` in addition to `defineBot`.
- `Host.SetSession(session)` installs outbound ops after `discordgo.Session` exists.
- `bot.NewWithScript(...)` calls `jsHost.SetSession(session)` after loading JS host and before gateway open.

Pseudocode:

```go
type RuntimeState struct {
    moduleName string
    store *MemoryStore
    outbound atomic.Pointer[DiscordOps]
}

func (s *RuntimeState) SetOutboundOps(ops *DiscordOps) { ... }

func (s *RuntimeState) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    exports.Set("defineBot", ...)
    exports.Set("channels", discordOpsObject(vm, runtimebridge.CurrentContext(vm), s.outbound.Load()).Get("channels"))
}
```

This is not Express-specific. Any JavaScript callback in the runtime can call it.

## 5. Example xgoja.yaml

```yaml
name: xdiscord
target:
  kind: xgoja
  output: dist/xdiscord
packages:
  - id: discord-bot
    import: github.com/go-go-golems/discord-bot/pkg/xgoja/provider
    replace: ../../..
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtimes:
  bot:
    modules:
      - package: discord-bot
        name: discord
        as: discord
      - package: discord-bot
        name: ui
        as: ui
      - package: go-go-goja-host
        name: fs
        as: fs
        config:
          allow: true
      - package: go-go-goja-http
        name: express
        as: express
commands:
  eval:
    enabled: true
    runtime: bot
commandProviders:
  - id: discord-bots
    package: discord-bot
    name: bots
    mount: bots
    runtimeProfile: bot
    config:
      workingDirectory: "."
      repositories:
        - ./bots
```

Then run:

```bash
./dist/xdiscord bots http-say run --http-listen :8787 --sync-on-start
```

## 6. Manual test plan

1. Build generated binary.
2. Start bot in tmux:

   ```bash
   tmux new-session -d -s xgoja-discord-http -- \
     ./dist/xdiscord bots http-say run --http-listen :8787 --sync-on-start
   ```

3. Open the browser at:

   ```text
   http://localhost:8787/
   ```

4. Submit channel ID and message.
5. Verify Discord receives the message.
6. Stop:

   ```bash
   tmux kill-session -t xgoja-discord-http
   ```

## 7. Implementation order

1. Clean up any uncommitted hidden-global or discord-bot-owned express hack.
2. Add design and diary documents.
3. Add xgoja HTTP provider with config section and lifecycle helper.
4. Add/adjust Express loader export in `modules/express`.
5. Add command-provider module-section initialization path to `discord-bot/pkg/xgoja/provider`.
6. Add top-level Discord outbound API to `require("discord")`.
7. Update example bot to register an Express UI endpoint.
8. Smoke locally without Discord using `eval`/`bots help`.
9. Run in tmux with credentials and tell the user the attach URL/commands.

## 8. Risks and caveats

- The current provider capability API is package-level, so HTTP provider runtime state must be carefully scoped per runtime.
- Express route handlers run on the JS owner thread via `gojahttp.Host`; avoid long blocking Go calls from route handlers where possible.
- The Discord outbound API must fail clearly if called before a session is attached.
- HTTP listen defaults must avoid surprising public exposure. Default should be `127.0.0.1:8787`, not `:8787`.
- This depends on unreleased xgoja command-provider APIs until a new go-go-goja release is cut.
