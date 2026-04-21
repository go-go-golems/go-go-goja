# Tasks

## TODO

- [x] Re-read the current `replapi`, `replsession`, `goja-repl`, and `js-repl` boundaries and turn the ticket scaffold into a concrete implementation plan.
- [x] Phase 3.1: Introduce explicit `replapi` config, profiles, and session options that can express raw, interactive, and persistent policies without boolean soup.
- [x] Phase 3.2: Refactor `replsession.Service` so evaluation behavior is policy-driven, with a direct/raw execution path and the current instrumented persistent path selectable per session.
- [x] Phase 3.3: Update `replapi.App` construction and session lifecycle so persistence, auto-restore, and evaluation policy can be enabled or disabled coherently.
- [x] Phase 3.4: Update `cmd/goja-repl` to use the persistent profile explicitly and add targeted tests proving the configured behavior remains stable.
- [x] Phase 3.5: Adopt the new interactive profile in the traditional line REPL entry point (`cmd/repl`) to prove non-persistent consumption of `replapi`.
- [x] Phase 3.6: Add focused tests for raw mode, interactive mode, persistent mode, and override behavior.
- [x] Phase 3.7: Commit each code slice intentionally and record each slice in the diary and changelog.
