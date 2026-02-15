# Changelog

## 2026-02-15

- Created GOJA-034 ticket workspace for user-facing API design over extracted inspector functionality.
- Added detailed design guide:
  - `design/01-user-facing-inspector-api-analysis-and-design-guide.md`
  - 4,460 words (8+ page equivalent) with architecture onboarding, capability inventory, API option analysis, decision matrix, pseudocode, and contract sketches.
- Updated ticket index and tasks to reflect completion of design/analysis objectives.
- Added detailed phase-A implementation plan for hybrid architecture with clean cutover assumptions:
  - `design/02-hybrid-implementation-plan-clean-cutover.md`
- Expanded tasks for executable implementation phases and added GOJA-034 diary scaffold:
  - `tasks.md`
  - `reference/01-diary.md`
- Committed planning artifacts:
  - `b31157b` — `docs(goja-034): add clean-cutover implementation plan and tasks`
- Implemented phase-A hybrid facade and clean cutover:
  - `cd823ad` — `inspectorapi: add hybrid service layer and cut over smalltalk static flows`
  - Added `pkg/inspectorapi` with:
    - document lifecycle (open/update/close),
    - globals/members/declaration methods,
    - runtime merge + REPL declaration extraction,
    - tree/sync wrappers.
  - Cut over `cmd/smalltalk-inspector/app` static orchestration from `inspectoranalysis.Session` to `inspectorapi.Service`.
  - Added service tests and updated smalltalk member tests.
  - Validation completed:
    - `go test ./pkg/inspectorapi/... -count=1`
    - `go test ./cmd/smalltalk-inspector/... -count=1`
    - `go test ./... -count=1`
