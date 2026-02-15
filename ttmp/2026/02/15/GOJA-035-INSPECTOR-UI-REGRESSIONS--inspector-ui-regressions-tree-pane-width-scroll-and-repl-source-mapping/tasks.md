# Tasks

## TODO (Follow-up Findings)

- [x] Fix drawer lexical binding resolution to avoid nondeterministic scope-map lookups (`cmd/inspector/app/model.go`) and add shadowing tests.
- [x] Guard globals half-page navigation when list is empty (`cmd/smalltalk-inspector/app/update.go`) and add regression test.
- [x] Disambiguate runtime method source mapping by candidate source matching (`pkg/inspector/runtime/function_map.go`) and add ambiguity test.
- [x] Run targeted + full test suite and record outcomes in ticket changelog/diary.

## Done

- [x] Reproduce both regressions in tmux and capture evidence logs in `scripts/`.
- [x] Write GOJA-035 analysis doc with root-cause breakdown and fix plan.
- [x] Implement REPL-symbol source fallback in `cmd/smalltalk-inspector` using declaration-aware REPL source log entries.
- [x] Add regression test for REPL symbol source fallback behavior.
- [x] Implement tree pane UX cleanup in `cmd/inspector` (narrower split, clamped tree rows, cleaner list delegate, reduced meta height).
- [x] Add tests for tree pane width policy and tree title clamping.
- [x] Validate fixes with `go test` and tmux re-runs.
- [x] Update ticket diary/changelog with commands, outcomes, and review notes.
