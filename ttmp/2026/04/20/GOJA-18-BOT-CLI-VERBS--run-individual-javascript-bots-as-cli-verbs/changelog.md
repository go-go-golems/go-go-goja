# Changelog

## 2026-04-20

- Created ticket `GOJA-18-BOT-CLI-VERBS` for the requested `go-go-goja bots list|run|help` command surface.
- Analyzed the current `go-go-goja` `jsverbs` scan/describe/invoke pipeline and the `engine` runtime composition flow.
- Analyzed `loupedeck`'s `verbs` bootstrap and runtime command wrapper patterns as the main reusable reference.
- Documented the design decision to keep sandbox `defineBot(...)` out of the v1 CLI discovery path.
- Wrote the primary design / implementation guide, quick-reference API doc, and diary.
- Related the key code files to the ticket docs.
- Validated the ticket with `docmgr doctor`.
- Uploaded the ticket bundle to reMarkable at `/ai/2026/04/20/GOJA-18-BOT-CLI-VERBS`.
- Implemented the first working `go-go-goja bots` CLI under `cmd/go-go-goja` and `pkg/botcli`, including `list`, `run`, and `help`.
- Chose explicit `__verb__` discovery for v1 by scanning with `IncludePublicFunctions = false`.
- Added end-to-end bot CLI tests covering listing, structured output, text output, async Promise settlement, and relative `require()` support.
- Refreshed the reMarkable bundle after implementation so the uploaded PDF matches the current repo and diary state.
- Added dedicated `testdata/botcli` fixtures plus duplicate-repository fixtures to validate bot CLI behavior with bot-specific inputs.
- Added empty/multi-repository/duplicate repository tests and a help page for authoring bot scripts with explicit `__verb__` metadata.
- Refreshed the reMarkable bundle again so the uploaded PDF includes the fixture/docs follow-up work.
- Added a realistic example repository under `examples/bots` covering structured output, text output, async verbs, relative `require()`, bound sections/context, package metadata, and `bind: all`.
- Added a ticket playbook with exact smoke-test commands for the real example repository.
- Refreshed the reMarkable bundle again so the uploaded PDF includes the real example repository and smoke-test playbook.
