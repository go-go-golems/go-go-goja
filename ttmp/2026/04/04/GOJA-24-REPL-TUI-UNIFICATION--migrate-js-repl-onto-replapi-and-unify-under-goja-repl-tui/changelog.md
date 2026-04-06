# Changelog

## 2026-04-04

- Initial workspace created
- Added the primary design guide for migrating `cmd/js-repl` onto `replapi` and folding the Bubble Tea frontend into `goja-repl tui`
- Recorded the initial investigation diary with line-anchored evidence and migration rationale
- Phase 1 landed in commit `62e774a` (`Add replapi-backed Bobatea runtime adapter`): added a `replapi` runtime access hook, a Bobatea adapter backed by `replapi.App`, and focused adapter/runtime tests
- Phase 2 landed in commit `6c4afd8` (`Share JS assistance between evaluator and replapi adapter`): extracted JavaScript completion/help into a shared assistance provider and wired both the old evaluator and new `replapi` adapter onto it
- Phase 3 and Phase 4 landed in commit `0412ae8` (`Add goja-repl tui and retire cmd/js-repl`): `cmd/goja-repl` now exposes `tui`, the Bubble Tea startup was moved under the unified binary, the TUI now runs through the `replapi` adapter, `cmd/js-repl` was removed, and user-facing docs/examples were updated to point at `goja-repl tui`
- Validation for the cutover passed with targeted adapter/CLI tests, a full `go test ./...`, and a `tmux` smoke run that exercised `goja-repl tui --alt-screen=false` interactively
- Ticket metadata was refreshed after the cutover, `docmgr doctor --ticket GOJA-24-REPL-TUI-UNIFICATION --stale-after 30` passed cleanly, and the updated ticket bundle was uploaded to reMarkable as `GOJA-24 REPL TUI Unification - updated` to avoid overwriting the earlier copy implicitly
- Phase 6 landed in commit `5856ade` (`Add runtime-only JS assistance adapter for inspector`): added a Bobatea assistance-only adapter for existing runtimes and moved `smalltalk-inspector` off the full JavaScript evaluator dependency while preserving completion/help/declaration tracking
- Validation for the inspector cleanup passed with `go test ./pkg/repl/adapters/bobatea ./cmd/smalltalk-inspector/app` and the repository pre-commit hook, which again ran `go generate ./...` and `go test ./...`

## 2026-04-06

- Phase 7 landed in commit `18d790f` (`Remove legacy web repl prototype`): removed `cmd/web-repl`, deleted `pkg/webrepl` and its embedded static browser UI, and completed the repository-side retirement of the old web prototype path
- Post-removal validation passed with a non-historical reference scan showing no remaining current references to `cmd/web-repl` / `pkg/webrepl` outside ticket history and with `go test ./...` passing cleanly
- Phase 8 landed in commit `5c0cec1` (`Remove standalone repl command`): removed `cmd/repl`, deleted the last standalone line-REPL entrypoint, and updated current onboarding/help/plugin docs so `goja-repl tui` is the canonical interactive surface
- Validation for the canonical-command cleanup passed with a non-historical reference scan showing no remaining current references to `cmd/repl` and with both `go test ./...` and the repository pre-commit hook succeeding
