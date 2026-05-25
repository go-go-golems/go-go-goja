---
Title: Implementation Diary
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
DocType: reference
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Diary for implementing xgoja-owned HTTP/Express mounting and Discord runtime outbound APIs.
LastUpdated: 2026-05-25T12:30:00-04:00
---

# Implementation Diary

## Step 1: Cleaned up hidden-global hack and wrote the design

### Prompt context

The user clarified that `discord-bot` should not know anything about Express. Express/HTTP must be mounted by xgoja, and the Discord bot module should only expose whatever outbound Discord APIs are needed for route handlers to send messages.

### What I did

- Stopped the old tmux session `xgoja-discord-bot` so stale generated processes do not hide lifecycle bugs.
- Removed the uncommitted hidden-global experiment:
  - reverted `discord-bot/internal/jsdiscord/host.go`;
  - reverted `discord-bot/internal/jsdiscord/host_options.go`;
  - deleted uncommitted `discord-bot/internal/jsdiscord/host_outbound.go`.
- Created XGOJA-011.
- Wrote `design/01-http-express-provider-and-discord-outbound-api.md`.

### Design conclusion

- xgoja/go-go-goja owns a new HTTP/Express provider and the HTTP server lifecycle.
- `discord-bot` owns a top-level outbound Discord API such as `require("discord").channels.send(...)`.
- The `discord-bot` provider command must aggregate module-provided sections and initialize selected runtime-profile capabilities when it creates runtimes through xgoja.

### Current tmux status

No tmux session is running for this new ticket yet.

## Step 2: Uploaded design bundle to reMarkable

Uploaded the design guide, diary, and task list to reMarkable.

```text
OK: uploaded XGOJA-011 HTTP Express Discord outbound design.pdf -> /ai/2026/05/25/XGOJA-011
```

No tmux session is running for this new ticket yet.

## Step 3: Implemented xgoja HTTP provider slice

### What changed

- Added optional `providerapi.RuntimeCloserRegistry` so provider runtime initializers can register cleanup hooks without changing the core `RuntimeHandle` contract.
- Implemented that optional closer registry on the xgoja app runtime handle.
- Added `modules/express.NewLoader(host, ...)`, which adapts runtimebridge owner bindings into the runtimeowner interface required by `gojahttp.Host`.
- Added `pkg/xgoja/providers/http` with package ID `go-go-goja-http`.
- Registered an `express` module and an HTTP config capability.
- Added Glazed section `http` with prefixed flags:
  - `--http-enabled`
  - `--http-listen`
- The express loader starts an HTTP server for the runtime and the runtime closer shuts it down.

### Validation

```bash
go test ./pkg/xgoja/providers/http ./modules/express ./pkg/xgoja/app ./pkg/xgoja/providerapi -count=1
```

Result: passed.

### Notes

This slice only adds xgoja-owned HTTP/Express. The Discord bot command provider still needs to aggregate module sections for provider-owned commands and initialize selected module capabilities when it creates runtimes through xgoja.
