# Changelog

## 2026-04-07

- Created ticket `GOJA-042-REPL-CLEANUP`.
- Added a detailed design and implementation guide for the cleanup/refactor follow-up.
- Scoped this work as lower-priority architecture cleanup to be done after correctness work lands.
- Validated the ticket with `docmgr doctor` and uploaded the bundle to reMarkable.

## 2026-04-08

- Extracted the evaluation cluster from `pkg/replsession/service.go` into the new file `pkg/replsession/evaluate.go`.
- Kept function names and behavior stable on purpose; this slice is a structural refactor, not a semantics change.
- Validated the refactor with `go test ./pkg/replsession ./pkg/replapi`.
- Noted one existing timing-sensitive edge during validation: `TestServiceRawAwaitPromiseTimeoutUsesEvalDeadline` failed once in a full package run, then passed in isolation and on rerun.
- Extracted cell persistence, binding version/doc shaping, and binding export snapshot helpers into the new file `pkg/replsession/persistence.go`.
- Extracted runtime observation, binding runtime refresh, global snapshot/diff logic, and summary shaping into the new file `pkg/replsession/observe.go`.
- Reduced `pkg/replsession/service.go` from 1175 lines after the first split to 467 lines after the persistence and observation splits.
- Revalidated both cleanup slices with `go test ./pkg/replsession ./pkg/replapi`, and the pre-commit hook reran full lint and `go test ./...` successfully on both commits.
