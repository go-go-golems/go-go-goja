# Tasks

## Phase 1: Ticket Setup And Evidence Collection

- [x] Create GOJA-13 for the origin-main review.
- [x] Inventory the committed branch surface with `git log`, `git diff --stat`, `git diff --name-only`, and `git diff --dirstat`.
- [x] Add a small reproducibility script under `scripts/` for the git review surface.

## Phase 2: Runtime And Plugin Architecture Review

- [x] Review the `engine` runtime composition changes.
- [x] Review the `pkg/hashiplugin` host, contract, shared transport, and SDK layers.
- [x] Identify duplication, runtime-lifecycle debt, validation drift, and error-handling gaps.

## Phase 3: Docs And REPL Architecture Review

- [x] Review `pkg/docaccess` and the runtime docs registrar.
- [x] Review `repl`, `js-repl`, and `bun-demo` bootstrap code for duplication and drift.
- [x] Review the coexistence story between the legacy `glazehelp` path and the new docs hub.
- [x] Review evaluator help/completion behavior against the new docs architecture.

## Phase 4: Reporting

- [x] Write a detailed review report with evidence-backed findings and cleanup sketches.
- [x] Write a chronological investigation diary.
- [x] Update changelog with the review deliverable.

## Phase 5: Closeout

- [x] Relate the key runtime/plugin/docs files to the review ticket docs.
- [x] Run `docmgr doctor --ticket GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW --stale-after 30`.
- [x] Upload the bundle to reMarkable and verify the remote listing.

## Phase 6: Implementation Planning From Review Findings

- [x] Add a focused implementation/design document for registrar state ownership and plugin diagnostics/cancellation hardening.
- [x] Turn the review findings into an actionable cleanup backlog with execution order.

## Phase 7: Runtime And Entry Point Cleanup

- [ ] Persist registrar-produced runtime values on `engine.Runtime`.
- [ ] Add a runtime-owned cancellation context and route plugin invocation through it.
- [ ] Strengthen plugin diagnostics and report summaries in `pkg/hashiplugin/host`.
- [x] Remove the legacy `modules/glazehelp` module and its registration path.
- [ ] Consolidate duplicated plugin bootstrap helpers across `repl`, `js-repl`, and `bun-demo`.

## Phase 8: HashiPlugin Contract And Validation Cleanup

- [x] Centralize shared manifest validation rules so SDK and host do not duplicate namespace/export/method shape checks.
- [x] Keep SDK-only validation focused on authoring-time concerns such as nil handlers and nil definitions.
- [ ] Decide whether method `summary` and `tags` should be exposed in the SDK now or explicitly documented as deferred.

## Phase 9: Docs Integration Follow-Up

- [ ] Persist the docs hub into runtime-owned state for evaluator/help consumers.
- [ ] Wire evaluator help and autocomplete to the unified `docaccess` hub instead of static/parallel surfaces.
