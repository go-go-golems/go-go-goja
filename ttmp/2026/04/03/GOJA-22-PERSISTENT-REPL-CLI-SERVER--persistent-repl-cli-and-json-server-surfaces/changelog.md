# Changelog

## 2026-04-03

- Initial workspace created
- Replaced the scaffold with a concrete implementation plan, ordered task list, and working diary for the CLI and JSON server phase
- Added restore-aware orchestration on top of replsession and repldb so persisted sessions can be replay-restored into fresh live runtimes (commit `8b35b5e`)
- Added the new `goja-repl` binary plus a JSON-only `replhttp` transport with session lifecycle, evaluation, history, bindings, docs, export, and restore surfaces (commit `1f9848d`)
