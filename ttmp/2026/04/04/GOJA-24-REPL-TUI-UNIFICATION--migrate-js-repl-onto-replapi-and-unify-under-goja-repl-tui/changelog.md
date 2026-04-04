# Changelog

## 2026-04-04

- Initial workspace created
- Added the primary design guide for migrating `cmd/js-repl` onto `replapi` and folding the Bubble Tea frontend into `goja-repl tui`
- Recorded the initial investigation diary with line-anchored evidence and migration rationale
- Phase 1 landed in commit `62e774a` (`Add replapi-backed Bobatea runtime adapter`): added a `replapi` runtime access hook, a Bobatea adapter backed by `replapi.App`, and focused adapter/runtime tests
