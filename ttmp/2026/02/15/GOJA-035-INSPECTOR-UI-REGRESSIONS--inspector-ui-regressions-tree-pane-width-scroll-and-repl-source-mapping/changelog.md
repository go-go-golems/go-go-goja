# Changelog

## 2026-02-15

- Initial workspace created.
- Added tmux-based analysis document with verbatim capture excerpts and root-cause analysis for both regressions.
- Added reproducible tmux automation script and stored pre-fix captures under `scripts/`.
- Fixed smalltalk-inspector REPL symbol source fallback by storing declared names in REPL source entries and falling back when file declaration lookup misses.
- Added regression test: `TestJumpToBindingFallsBackToReplSource`.
- Improved inspector tree pane ergonomics:
  - source/tree split moved to ~60/40
  - tree row description rendering disabled for compact mode
  - width-aware tree title clamping with ellipsis
  - reduced tree metadata height to improve visible row count
- Added tree UX tests: `TestTreePaneWidthKeepsTreeCompact` and `TestBuildTreeListItemClampsTitle`.
- Validated with:
  - `go test ./cmd/smalltalk-inspector/app ./cmd/inspector/app -count=1`
