# Tasks

## TODO

- [x] Re-read the current CLI/server entry points and the persisted REPL core from GOJA-21.
- [x] Replace the GOJA-22 scaffold with a detailed CLI/server implementation plan and diary.
- [x] Phase 2.1: Add restore-aware orchestration that combines replsession live state with repldb persisted history.
- [x] Phase 2.2: Extend replsession as needed so a persisted session can be replay-restored into a fresh live runtime.
- [x] Phase 2.3: Add a JSON-only HTTP transport package with session lifecycle, evaluate, history, bindings, docs, export, and restore endpoints.
- [x] Phase 2.4: Add a new `cmd/goja-repl` binary with persistent `create`, `eval`, `snapshot`, `history`, `bindings`, `docs`, `export`, `restore`, and `serve` commands.
- [x] Phase 2.5: Add targeted tests for restore-aware orchestration, HTTP handlers, and CLI command behavior.
- [x] Phase 2.6: Commit the implementation in focused slices and record each slice in the diary and changelog.
