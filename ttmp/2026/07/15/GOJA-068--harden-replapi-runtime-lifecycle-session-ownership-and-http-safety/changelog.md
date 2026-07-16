# Changelog

## 2026-07-15

- Initial workspace created


## 2026-07-15

Completed evidence-backed replapi lifecycle and hardening analysis with five executable probes, twelve findings, proposed APIs/state machines, phased implementation, and test strategy.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md — Primary intern-oriented architecture and implementation guide
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Chronological investigation and reproduction record


## 2026-07-15

Validated GOJA-068 docs and probe packages, corrected standalone probe packaging, and uploaded the index/design/diary bundle to /ai/2026/07/15/GOJA-068 on reMarkable.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md — Validated implementation baseline included in delivered bundle
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Records validation failures, fixes, and successful upload evidence


## 2026-07-15

Refined implementation tracking into three milestones, eight dependency-gated phases, and 81 granular tasks with stable P#.N labels, validation commands, exit invariants, and PR boundaries.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md — Detailed phase dependencies, work packages, validation, and gates
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Records task refinement rationale and workflow
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/tasks.md — Authoritative phase-by-phase implementation tracker


## 2026-07-15

Published a separate GOJA-068 Phased Implementation Plan reMarkable bundle containing index, 81-task tracker, refined design guide, and diary; original analysis bundle was preserved.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md — Refined dependency-gated phase plan
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/tasks.md — Granular phase tracker included in the new bundle


## 2026-07-15

Completed Phase 0: added eight opt-in replapi_hardening red regressions, deterministic expected-failure harness, green full/race baselines, and test promotion guide.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/hardening_regression_test.go — Cancellation, profile, ownership, and delete serialization contracts
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replhttp/hardening_regression_test.go — HTTP request/runtime lifetime contract
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/hardening_regression_test.go — Persistence failure must block later cells
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/02-phase-0-red-test-baseline-and-execution-guide.md — Phase 0 commands, expected failures, and promotion workflow


## 2026-07-15

Completed Phase 1: added typed/canonical profile validation, complete bare-config preset resolution, mismatch/policy checks, shared TUI parsing, promoted three P0 regressions, and serialized root command tests after race detection.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/goja-repl/tui.go — Uses shared replapi profile parser
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/app.go — Propagates config and session override validation errors
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/config.go — Authoritative profile and policy configuration contract
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/config_test.go — Normal-CI coverage for all P1 contracts
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Records P1 implementation and race-test correction


## 2026-07-15

Completed Phase 2: explicit app-owned contexts, typed lifecycle phases, context-aware session gates, operation-context WithRuntime, non-destructive unload, retryable aggregated close, serialized delete, caller migration, lifecycle tests, and three promoted P0 regressions.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/engine/factory.go — Race-free event-loop startup before runtime publication
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/lifecycle.go — Public app unload/close and typed app errors
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/lifecycle.go — Core P2 lifecycle and shutdown state machine
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/service.go — Session-owned runtime context creation and restore publication
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Chronological P2 implementation, failures, validation, and review guide


## 2026-07-15

Completed Phase 3: added healthy/degraded/fenced states, response-plus-CommitError semantics, exact retained-record retry, delayed committed history publication, pre-execution health rejection, discard-and-restore recovery, failure injection, and promoted the persistence regression.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/recovery_test.go — P3 real SQLite-trigger recovery proof
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/health.go — P3 public health and recovery state machine
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/persistence.go — P3 exact record construction and fail-closed commit publication
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Step 9 P3 implementation and review record


## 2026-07-15

Completed Phase 4: replaced unconditional schema stamping with ordered transactional v1 migration, typed newer-version rejection, empty/concurrent bootstrap, rollback tests, immutable real v1 fixture, and operator backup/upgrade guidance.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repldb/migrations.go — P4 migration runner and schema version errors
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repldb/migrations_test.go — P4 data preservation, rollback, idempotence, newer-version, and concurrency tests
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repldb/testdata/repl-v1.sqlite — P4 representative historical database
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Step 10 P4 implementation and concurrency debugging record


## 2026-07-15

Completed Phase 5: selected per-session multi-process ownership, added schema-v2 leases, random app owner IDs, fake-clock acquisition/renewal/takeover, heartbeat during evaluation/replay, fenced append with expected-cell validation, stale-owner health fencing/recovery, lease-aware delete/close, and promoted the final red regression.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replapi/ownership_test.go — P5 end-to-end takeover, fencing, heartbeat, recovery, and release proofs
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repldb/lease.go — P5 atomic ownership and fenced journal writes
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repldb/migrations.go — P5 schema-v2 session_leases migration
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replsession/ownership.go — P5 runtime ownership guard and heartbeat
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Step 11 Phase 5 implementation and review record


## 2026-07-15

Completed Phase 6 HTTP/protobuf hardening: bounded exact-version evaluate decoding, generated stable ErrorResponse envelopes, typed status mapping, default redaction with server diagnostics, request/security headers, complete server limits, explicit non-loopback acknowledgement, generated Go/TypeScript fixtures, and real HTTP acceptance coverage; P6.GATE closed.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/goja-repl/cmd_serve.go — Server resource timeouts and remote-bind acknowledgement
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replhttp/handler.go — Bounded handler configuration, safe errors, status mapping, request IDs, headers, and diagnostics
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replhttp/proto_handler.go — Strict protobuf-JSON request parsing and generated response writing
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/replhttp/security_test.go — Real HTTP regression and security coverage
- /home/manuel/code/wesen/go-go-golems/go-go-goja/proto/goja/replapi/v1/replapi.proto — Public ErrorResponse schema


## 2026-07-15

Completed Phase 7 and GOJA-068: all constructor/callback callers migrated, CLI/TUI/server shutdown returns aggregated errors while closing app before store, Bobatea observes operation cancellation, public migration/API docs are current, and full Go/race/vet/Glazed/Buf/TypeScript/doc validation passes.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/goja-repl/root.go — Shared bounded app-before-store shutdown
- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/goja-repl/shutdown_test.go — Lease release and shutdown error tests
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/doc/04-repl-usage.md — Final protobuf HTTP client guidance
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/doc/34-replapi-guide.md — Breaking API and host migration guide
- /home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea/replapi.go — Context-aware assistance bridge


## 2026-07-15

Published the final GOJA-068 index, implementation guide, investigation diary, Phase 0 guide, tasks, and changelog bundle to reMarkable as /ai/2026/07/15/GOJA-068/GOJA-068 Final Replapi Hardening.pdf after a successful dry-run.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md — Final reviewed design bundle source
- /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/01-investigation-diary.md — Final implementation and validation narrative


## 2026-07-15

Ticket closed
