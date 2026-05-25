---
Title: Discord Bot xgoja Adapter Design
Ticket: XGOJA-010
Status: active
Topics:
  - xgoja
  - goja
  - providers
  - discord
  - bot
  - fs
  - express
  - architecture
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles:
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/botcli/command_root.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/botcli/runtime_factory.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/internal/jsdiscord/host.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/internal/jsdiscord/runtime.go
  - /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/commands.go
ExternalSources: []
Summary: Design for a real discord-bot provider-shipped xgoja command adapter that can run Discord JavaScript bots with selected xgoja modules such as fs and eventually express.
LastUpdated: 2026-05-25T12:05:00-04:00
---

# Discord Bot xgoja Adapter Design

## Goal

Build a real `discord-bot` xgoja adapter so a generated xgoja binary can mount package-owned Discord bot commands and run JavaScript bot scripts with the same runtime-profile modules that the generated binary exposes elsewhere.

The near-term target is:

```bash
xdiscord bots list
xdiscord bots help fs-express-smoke
xdiscord bots fs-express-smoke run --sync-on-start
```

where `xdiscord` is a generated xgoja binary whose runtime profile includes:

- `discord` and `ui` from `discord-bot`;
- `fs` / `node:fs` from the go-go-goja host provider;
- an express-like module once the HTTP host boundary is designed.

## Current state

The `discord-bot` repository already has a mature package-owned command system in `pkg/botcli`:

- `NewBotsCommand(...)` builds a Cobra tree with `list`, `help`, and discovered bot commands.
- Discovered bot scripts are found from repositories using `BuildBootstrap(...)` and `DiscoverBots(...)`.
- Bot scripts use `require("discord")` and `defineBot(...)` from `internal/jsdiscord`.
- Runtime construction for bot CLI paths is centralized in:
  - `internal/jsdiscord.NewHost(...)` for host-managed bot lifecycle;
  - `pkg/botcli.defaultRuntimeFactory(...)` for ordinary jsverbs embedded in bot scripts.

XGOJA-008/009 added the missing generated-host extension points:

- provider-shipped command sets (`providerapi.CommandSetProvider`);
- module descriptors and runtime-profile selection;
- command-provider mounting that no longer mutates provider-owned command descriptions;
- Glazed `eval`, `run`, `repl`, and `jsverbs` module-section support.

## Proposed adapter shape

Add `discord-bot/pkg/xgoja/provider` with a public `Register(*providerapi.Registry) error` function.

The provider package should register package ID `discord-bot` and expose:

| Entry | Kind | Purpose |
| --- | --- | --- |
| `discord` | module | `require("discord")`, backed by `internal/jsdiscord.RuntimeState.Loader`. |
| `ui` | module | `require("ui")`, backed by the existing Discord UI helper loader. |
| `bots` | command set provider | Mount `botcli` commands into generated xgoja binaries. |

Example buildspec shape:

```yaml
packages:
  - id: discord-bot
    import: github.com/go-go-golems/discord-bot/pkg/xgoja/provider
    replace: ../../..
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host

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
      - package: go-go-goja-host
        name: node:fs
        as: node:fs
        config:
          allow: true

commandProviders:
  - id: discord-bots
    package: discord-bot
    name: bots
    mount: bots
    runtimeProfile: bot
    config:
      repositories:
        - ./bots
```

## Runtime bridge

The critical implementation point is that `discord-bot` bot execution must use the xgoja runtime profile selected by `commandProviders[].runtimeProfile`.

Today, `botcli` and `jsdiscord.NewHost` build their own engine runtimes. The adapter should add a small bridge:

```go
type xgojaRuntimeFactory interface {
    NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error)
}
```

The command set provider receives this as `CommandSetContext.RuntimeFactory`. It can wrap that object into the existing `botcli.RuntimeFactory` interface for jsverbs and a new `jsdiscord.HostOption` for host-managed bot runs.

Runtime creation should pass script-local require options:

- module roots derived from the bot script path;
- `require.WithGlobalFolders(scriptDir, scriptDir/node_modules)`;
- the bot script repository jsverbs loader when invoking ordinary jsverbs.

This lets one generated runtime profile own module selection, while `discord-bot` continues to own Discord-specific command and lifecycle behavior.

## Test functionality

Create an example bot repository under `discord-bot/examples/xgoja/discord-bot-provider/bots` with a bot script named `fs-express-smoke`.

Suggested bot behavior:

- Slash command `ping` returns `pong from xgoja`.
- Slash command `read-config` reads a local text file through `require("fs")` and returns a short value.
- Event `ready` logs that the xgoja-backed bot started.
- Express route registration can be included once express has a host boundary. For the first pass, document it as planned unless a real HTTP host is wired.

Example JavaScript:

```js
const { defineBot } = require("discord")
const fs = require("fs")

module.exports = defineBot(({ command, event, configure }) => {
  configure({ name: "fs-express-smoke", description: "xgoja Discord smoke bot" })

  event("ready", async (ctx) => {
    ctx.log.info("fs-express-smoke ready")
  })

  command("ping", { description: "Return a simple xgoja pong" }, async () => {
    return { content: "pong from xgoja" }
  })

  command("read-config", { description: "Read a local config file through fs" }, async () => {
    const text = fs.readFileSync("./bot-data/message.txt", "utf8").trim()
    return { content: `config says: ${text}` }
  })
})
```

## tmux run plan

Once the generated binary builds, start it in a tmux session only when credentials are present:

```bash
tmux new-session -d -s xgoja-discord-bot -- \
  ./examples/xgoja/discord-bot-provider/dist/xdiscord \
  bots fs-express-smoke run \
  --bot-token "$DISCORD_BOT_TOKEN" \
  --application-id "$DISCORD_APPLICATION_ID" \
  --guild-id "$DISCORD_GUILD_ID" \
  --sync-on-start
```

Tell the user the exact session name and attach command:

```bash
tmux attach -t xgoja-discord-bot
```

Also provide test steps:

1. Wait for the log line showing the bot connected.
2. In the configured Discord guild, run `/ping`.
3. Run `/read-config` and verify it returns the file contents.
4. Stop with `Ctrl-C` in tmux or `tmux kill-session -t xgoja-discord-bot`.

## Express caveat

`fs` can work immediately as a selected xgoja host module because it is a normal require module.

`express` needs one more host boundary decision. The existing go-go-goja express module is a runtime registrar around a `gojahttp.Host`; the generated xgoja provider-module API currently returns a require loader, not a runtime registrar plus HTTP server handle. For this ticket, express should be handled in one of two ways:

1. Add a small discord-bot-specific express host option for bot scripts, with an explicit listen flag and shutdown lifecycle; or
2. Extend xgoja provider capabilities later to support runtime registrars and host services.

Do not fake Discord production behavior. If express is only route-registration smoke in the first pass, label it clearly as local-only.

## Done criteria

- `discord-bot/pkg/xgoja/provider.Register` exists and registers modules plus command set provider.
- Generated example builds with local replaces.
- `bots list` and `bots help fs-express-smoke` work from generated binary.
- `eval` or `run` can require `discord` and `fs` through the selected profile.
- If credentials are present, a tmux session runs the generated bot and the user receives the session name and test commands.
