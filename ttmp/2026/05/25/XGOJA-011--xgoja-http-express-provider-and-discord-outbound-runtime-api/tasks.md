# Tasks

## Documentation and setup

- [x] Create XGOJA-011 ticket.
- [x] Clean up uncommitted hidden-global / discord-owned express experiment.
- [x] Write detailed design and implementation guide.
- [x] Upload design bundle to reMarkable.
- [x] Commit initial ticket docs.

## go-go-goja HTTP provider

- [x] Add optional `RuntimeCloserRegistry` interface to providerapi.
- [x] Implement `RuntimeCloserRegistry` on app runtime handle.
- [x] Export or otherwise expose an Express loader from `modules/express`.
- [x] Add `pkg/xgoja/providers/http` package.
- [x] Register provider package ID `go-go-goja-http`.
- [x] Register `express` module.
- [x] Add HTTP Glazed config section with `--http-listen` and `--http-enabled`.
- [x] Start an HTTP server for the runtime and register runtime closer shutdown.
- [x] Add focused provider tests.

## discord-bot command provider/runtime bridge

- [x] Add helper to collect selected module config sections in `discord-bot/pkg/xgoja/provider`.
- [x] Wrap provider-owned bot commands to carry parsed values into runtime creation.
- [x] Run selected module runtime initializers after xgoja runtime creation.
- [x] Add focused provider tests for section exposure and initializer invocation.

## Discord outbound runtime API

- [x] Add session-bound outbound ops to `RuntimeState`.
- [x] Expose top-level `discord.channels.send(channelId, payload)` from `require("discord")`.
- [x] Attach outbound ops from live `discordgo.Session` in `bot.NewWithScript` / `jsdiscord.Host`.
- [x] Add tests for top-level outbound API using fake DiscordOps or test session hooks.

## Example and manual test

- [x] Update `examples/xgoja/discord-bot-provider/xgoja.yaml` to include `go-go-goja-http` express module.
- [x] Update sample bot JS to register `/` and `/say` Express endpoints.
- [x] Update Makefile with `--http-listen 127.0.0.1:8787`.
- [x] Smoke generated build/list/help/eval.
- [x] Run generated bot in tmux and tell the user session name and test URL.

## Final validation

- [x] Run focused go-go-goja tests.
- [x] Run focused discord-bot tests.
- [x] Run generated example smoke.
- [x] Update diary/changelog after each slice.
- [x] Commit at appropriate intervals.
- [x] Run `docmgr doctor --ticket XGOJA-011 --stale-after 30`.
