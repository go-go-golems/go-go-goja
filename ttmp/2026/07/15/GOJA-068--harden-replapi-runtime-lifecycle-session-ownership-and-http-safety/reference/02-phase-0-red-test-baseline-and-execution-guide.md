---
Title: Phase 0 red-test baseline and execution guide
Ticket: GOJA-068
Status: active
Topics:
    - goja
    - repl
    - replapi
    - lifecycle
    - persistent-repl
    - http
    - security
    - sqlite
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/goja-repl/tui.go
      Note: P1 shared ParseProfile use in TUI
    - Path: repo://pkg/replapi/config.go
      Note: P1 typed profile validation, complete preset normalization, and policy consistency
    - Path: repo://pkg/replapi/config_test.go
      Note: P1 promoted profile regressions and replacement/mismatch tests
    - Path: repo://pkg/replapi/hardening_regression_test.go
      Note: Remaining P0.5 persistent-owner red regression
    - Path: repo://pkg/replhttp/hardening_regression_test.go
      Note: P0.1 regression promoted into normal CI by P2
    - Path: repo://pkg/replsession/hardening_regression_test.go
      Note: P0.4 fail-closed persistence regression promoted by P3
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/06-verify-phase-0-red-tests.sh
      Note: P0.7 exact expected-failure verification harness
ExternalSources: []
Summary: Defines the opt-in GOJA-068 hardening test suite, the eight expected failures on the Phase 0 baseline, green CI commands, and the process for promoting tests as later phases fix them.
LastUpdated: 2026-07-15T14:25:00-04:00
WhatFor: Run and interpret the Phase 0 red regressions without making the repository's default test suite fail.
WhenToUse: Use before beginning P1, after changing replapi lifecycle/persistence behavior, and when promoting a fixed regression into the normal test suite.
---



# Phase 0 red-test baseline and execution guide

## Goal

Phase 0 captured desired behavior before implementing it. Eight tests asserted the future lifecycle, configuration, persistence, and ownership contracts on the audited baseline. Red tests use the `replapi_hardening` build tag instead of breaking normal CI.

Phase 1 promoted the three profile regressions, Phase 2 promoted HTTP lifetime/canceled waiter/delete serialization, Phase 3 promoted fail-closed persistence, and Phase 5 promoted cross-app durable ownership. All eight contracts now run in normal CI. No test remains behind `replapi_hardening`; the harness is retained only to assert that the expected-red set is empty.

## Test Inventory

| Task | Package | Test | Status / expected failure |
|---|---|---|---|
| P0.1 | `pkg/replhttp` | `TestHardeningHTTPSessionRuntimeOutlivesCreateRequest` | **Promoted by P2:** app-owned lifetime survives request completion |
| P0.2 | `pkg/replapi` | `TestHardeningCanceledWaiterDoesNotExecuteLate` | **Promoted by P2:** capacity-one operation gate honors caller cancellation |
| P0.3 | `pkg/replapi` | `TestHardeningPartialRawConfigUsesRawPreset` | **Promoted by P1:** complete raw preset now resolves |
| P0.3 | `pkg/replapi` | `TestHardeningUnknownAppProfileIsRejected` | **Promoted by P1:** typed app-profile error |
| P0.3 | `pkg/replapi` | `TestHardeningUnknownSessionProfileIsRejected` | **Promoted by P1:** typed per-session error |
| P0.5 | `pkg/replapi` | `TestHardeningPersistentSessionRejectsSecondLiveOwner` | **Promoted by P5:** unexpired lease rejects a second writable owner |
| P0.6 | `pkg/replapi` | `TestHardeningDeleteWaitsForActiveSessionOperation` | **Promoted by P2:** delete serializes behind the active operation gate |
| P0.4 | `pkg/replsession` | `TestHardeningPersistenceFailureBlocksLaterEvaluation` | **Promoted by P3:** failed commit degrades the session and blocks later JavaScript |

Source files:

```text
pkg/replhttp/hardening_regression_test.go          # promoted P2 HTTP test
pkg/replapi/hardening_regression_test.go            # promoted P5 ownership test
pkg/replapi/config_test.go                          # promoted P1 tests
pkg/replapi/lifecycle_test.go                       # promoted P2 app tests
pkg/replsession/hardening_regression_test.go         # promoted P3 test
pkg/replsession/lifecycle_test.go                    # P2 lifecycle coverage
```

## Quick Reference

### Confirm normal CI remains green

```bash
go test ./...

go test -race \
  ./pkg/replapi \
  ./pkg/replsession \
  ./pkg/repldb \
  ./pkg/replhttp \
  ./pkg/repl/adapters/bobatea
```

### Confirm the retired build tag hides nothing

```bash
go test -tags replapi_hardening \
  ./pkg/replapi ./pkg/replsession ./pkg/replhttp \
  -run '^TestHardening' -count=1
```

This now passes because all hardening tests are untagged.

### Verify the exact Phase 0 red baseline

Use the ticket harness. It now verifies that no expected-red case remains:

```bash
./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/06-verify-phase-0-red-tests.sh
```

Expected output:

```text
Phase 0 red baseline confirmed: 0 expected failures.
```

### Run all promoted hardening tests directly

This command now exits successfully; the build tag is inert because every test was promoted by P1–P5:

```bash
go test -tags replapi_hardening \
  ./pkg/replapi ./pkg/replsession ./pkg/replhttp \
  -run '^TestHardening' -count=1 -v
```

## Determinism Rules

The P0 tests use explicit coordination where possible:

- The canceled-waiter test holds the session operation through channels, not JavaScript timing.
- The delete test holds `WithRuntime` through channels and checks whether delete returns before release.
- The persistence test injects failure at an exact cell ID.
- The ownership test uses two explicit app instances over one temporary SQLite store.
- The HTTP context test uses a real `httptest.Server`, because direct `ServeHTTP` does not model server request cancellation reliably.

The HTTP observation window and queue/delete guard windows are bounded. If they prove flaky under CI load, replace timing with stronger server/lifecycle hooks when Phase 2 introduces them; do not merely increase sleeps indefinitely.

## Promotion Workflow

When a phase fixes one regression:

1. Run the individual tagged test and confirm it now passes.
2. Move the test into an untagged test file in the same package.
3. Remove that case from `06-verify-phase-0-red-tests.sh`.
4. Run the normal package tests and focused race suite.
5. Check the corresponding implementation task, not the P0 task again.
6. Record promotion in the GOJA-068 diary and changelog.

Example:

```bash
go test -tags replapi_hardening ./pkg/replapi \
  -run '^TestHardeningPartialRawConfigUsesRawPreset$' -count=1

# After moving it into the normal suite:
go test ./pkg/replapi -run '^TestHardeningPartialRawConfigUsesRawPreset$' -count=1
```

## Phase 0 Gate Evidence

At the time P0 was completed:

- `go test ./...` passed.
- The focused race suite passed.
- The tagged packages compiled with `-run '^$'`.
- The red-test harness confirmed eight expected assertion failures.
- `git diff --check` passed.

P1 promoted three profile tests, P2 promoted three lifecycle tests, P3 promoted fail-closed persistence, and P5 promoted durable ownership. The active red harness is empty. The original eight-test result remains the historical P0 gate evidence.

## Related

- [Primary design](../design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md)
- [Investigation diary](./01-investigation-diary.md)
- [Task tracker](../tasks.md)
- Ticket probe scripts under `../scripts/01-*` through `../scripts/05-*`
