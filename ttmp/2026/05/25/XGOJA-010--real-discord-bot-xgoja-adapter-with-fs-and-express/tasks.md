# Tasks

## Documentation

- [x] Create XGOJA-010 ticket.
- [x] Create dedicated design document for real discord-bot xgoja adapter.
- [x] Create dedicated implementation diary.

## Provider/API implementation

- [x] Export discord runtime module loader from `internal/jsdiscord` for provider use.
- [x] Export UI module loader from `internal/jsdiscord` for provider use.
- [x] Add `discord-bot/pkg/xgoja/provider` with `Register(*providerapi.Registry)`.
- [x] Register `discord` and `ui` modules.
- [x] Register `bots` command set provider.
- [x] Add public botcli helper returning Glazed commands for provider-owned mounting.
- [x] Bridge xgoja runtime profiles into botcli ordinary jsverb runtime execution.
- [x] Bridge xgoja runtime profiles into host-managed bot runs.

## Example and smoke tests

- [x] Add generated xgoja example under `discord-bot/examples/xgoja/discord-bot-provider`.
- [x] Add sample bot with `ping` and `read-config` commands.
- [x] Include `fs` in the generated runtime profile.
- [x] Decide and document first-pass express behavior.
- [x] Smoke `xgoja doctor` and `xgoja build`.
- [x] Smoke generated `bots list`.
- [x] Smoke generated `bots help fs-express-smoke`.
- [x] If credentials are present, run generated bot in tmux and report session/test commands.

## Validation and delivery

- [x] Run focused `discord-bot` tests.
- [x] Run generated example smoke.
- [x] Update diary/changelog.
- [ ] Commit implementation.
