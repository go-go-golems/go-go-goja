# Tasks

## Completed

- [x] Create ticket `GOJA-042-REPL-CLEANUP`
- [x] Write the cleanup/refactor analysis/design/implementation guide
- [x] Record an investigation diary
- [x] Validate the ticket with `docmgr doctor`
- [x] Upload the ticket bundle to reMarkable

## Planned implementation work

- [x] Define a target module split for `pkg/replsession`
- [x] Extract evaluation orchestration from the main service file
- [x] Extract persistence shaping and snapshot helpers into focused files
- [x] Revisit `SessionOptions` naming at the app vs kernel boundary
- [x] Decide whether the older evaluator path should be consolidated, documented, or kept
- [x] Add lightweight tests to guard against refactor regressions

## Notes

- First implementation slice moved evaluation orchestration and execution helpers from `service.go` into `evaluate.go` without intentionally changing behavior.
- Second implementation slice moved persistence shaping and binding export/doc persistence helpers into `persistence.go`.
- Third implementation slice moved runtime observation, binding/runtime refresh, and summary shaping into `observe.go`.
- `service.go` is now down to session lifecycle, runtime/bootstrap setup, and a handful of shared utility helpers instead of being the single file for the whole subsystem.
- The `replapi` vs `replsession` session-option boundary is now clarified with comments, an internal helper rename, and focused tests instead of a public API rename.
- The Bobatea evaluator path is now explicitly treated as an active TUI integration surface, not deprecated code.
- During validation, `TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline` failed once in the full package run but passed immediately in isolation and on rerun, so the ticket currently treats that as an existing flaky timing edge rather than a cleanup regression.
