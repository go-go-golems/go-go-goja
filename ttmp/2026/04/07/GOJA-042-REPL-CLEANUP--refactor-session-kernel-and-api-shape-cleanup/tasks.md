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
- [ ] Extract persistence shaping and snapshot helpers into focused files
- [ ] Revisit `SessionOptions` naming at the app vs kernel boundary
- [ ] Decide whether the older evaluator path should be consolidated, documented, or kept
- [ ] Add lightweight tests to guard against refactor regressions

## Notes

- First implementation slice moved evaluation orchestration and execution helpers from `service.go` into `evaluate.go` without intentionally changing behavior.
- During validation, `TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline` failed once in the full package run but passed immediately in isolation and on rerun, so the ticket currently treats that as an existing flaky timing edge rather than a cleanup regression.
