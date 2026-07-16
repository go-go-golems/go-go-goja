# Tasks

## Completed research and design

These tasks established the evidence and proposed architecture. Implementation has not started.

- [x] Map replapi, replsession, runtime, persistence, HTTP, CLI, and TUI ownership boundaries <!-- t:cjxy -->
- [x] Reproduce and characterize runtime context cancellation through HTTP session creation <!-- t:0stw -->
- [x] Design explicit app close and non-destructive session eviction APIs <!-- t:jha4 -->
- [x] Design single-owner coordination and concurrency contracts for persistent sessions <!-- t:rfhx -->
- [x] Audit HTTP security, limits, error contracts, and lifecycle integration <!-- t:6hkg -->
- [x] Define phased implementation and comprehensive test strategy <!-- t:jfpc -->
- [x] Validate ticket documentation and upload bundle to reMarkable <!-- t:452z -->

## Milestone A — Mandatory correctness core

Phases 0–3 are required even for a local, single-process REPL. Complete them in order. Do not begin a later phase until the preceding `GATE` task is checked.

### Phase 0 — Regression safety net

**Entry:** Current main reproduces the ticket probes. **Exit:** Every defect is represented by a deterministic package test that fails for the intended reason.

- [x] [P0.1] Convert the HTTP runtime-context probe into a real httptest.Server regression test <!-- t:pslx -->
- [x] [P0.2] Convert the canceled-waiter probe into a deterministic barrier-based session queue test <!-- t:k9kg -->
- [x] [P0.3] Add partial raw Config and unknown profile regression tests <!-- t:7zjh -->
- [x] [P0.4] Add injected persistence-failure regression proving that cell 3 currently follows failed cell 2 <!-- t:jomq -->
- [x] [P0.5] Add two-App persistent split-brain regression over one SQLite store <!-- t:tcqj -->
- [x] [P0.6] Add evaluate-versus-delete characterization test with deterministic synchronization <!-- t:v9vw -->
- [x] [P0.7] Record current expected failures and confirm unrelated REPL tests remain green <!-- t:5zkc -->
- [x] [P0.8] Run focused race baseline for replapi, replsession, repldb, replhttp, and Bobatea adapter <!-- t:n74g -->
- [x] [P0.GATE] Phase 0 complete: every reproduced defect has a deterministic failing regression test <!-- t:u873 -->

### Phase 1 — Configuration correctness

**Depends on:** Phase 0. **Exit:** Profile names, effective policy, timeout, and persistence requirements are consistent and validated.

- [x] [P1.1] Introduce typed unknown-profile validation for app and per-session profile inputs <!-- t:yk1h -->
- [x] [P1.2] Make explicit Config.Profile resolve the complete matching preset when SessionOptions are zero <!-- t:pu3r -->
- [x] [P1.3] Reject contradictory profile and SessionOptions profile/policy combinations <!-- t:dfp3 -->
- [x] [P1.4] Preserve and test full-policy replacement semantics for default and session overrides <!-- t:x2ej -->
- [x] [P1.5] Preserve persistent-profile store validation and auto-restore requirements <!-- t:q388 -->
- [x] [P1.6] Reuse shared profile validation in the TUI parser where practical <!-- t:8omh -->
- [x] [P1.7] Update configuration API comments and profile behavior tests <!-- t:5mzc -->
- [x] [P1.GATE] Phase 1 complete: profile labels, policy presets, timeouts, and store requirements cannot disagree <!-- t:m657 -->

### Phase 2 — Runtime lifecycle and cancellation

**Depends on:** Phase 1. **Exit:** Runtime lifetime belongs to the app/session, queue waits honor cancellation, and close/unload/delete have distinct tested semantics.

- [x] [P2.1] Define app and session lifecycle states plus typed closing/closed errors <!-- t:um0b -->
- [x] [P2.2] Change replapi constructors to accept an explicit app parent context and migrate direct callers <!-- t:x6ng -->
- [x] [P2.3] Add replsession service lifetime context and derive one child lifetime per session <!-- t:besf -->
- [x] [P2.4] Use operation context only for runtime startup and app/session context for runtime lifetime <!-- t:2b86 -->
- [x] [P2.5] Replace blocking session mutex acquisition with a context-aware capacity-one operation gate <!-- t:opvk -->
- [x] [P2.6] Merge caller, session-lifetime, and evaluation-timeout cancellation without goroutine leaks <!-- t:kd1k -->
- [x] [P2.7] Change WithRuntime callback to receive an operation context and document non-escape/non-reentrancy <!-- t:uxr2 -->
- [x] [P2.8] Implement non-destructive Service.UnloadSession and App.UnloadSession <!-- t:4bsw -->
- [x] [P2.9] Implement idempotent Service.Close and App.Close with aggregated errors <!-- t:jg2v -->
- [x] [P2.10] Serialize delete/unload/close against active evaluation and preserve retryable closing state on timeout <!-- t:fybn -->
- [x] [P2.11] Add close, unload, cancellation, active-evaluation, and close-hook exactly-once tests <!-- t:926t -->
- [x] [P2.GATE] Phase 2 complete: runtime lifetime is app-owned and every live runtime has deterministic non-destructive shutdown <!-- t:pibo -->

### Phase 3 — Fail-closed persistence and recovery

**Depends on:** Phase 2. **Exit:** An evaluation that executes but fails durable commit cannot be followed by another cell until exact retry or discard-and-restore recovery.

- [x] [P3.1] Define healthy, degraded, and fenced session health states and typed errors <!-- t:x6h2 -->
- [x] [P3.2] Build and retain the exact EvaluationRecord before attempting durable commit <!-- t:62z0 -->
- [x] [P3.3] Delay publishing committed cell ID/history until the durable append succeeds where possible <!-- t:r1uf -->
- [x] [P3.4] Mark persistent sessions degraded on any post-execution append failure <!-- t:4b1z -->
- [x] [P3.5] Reject all later JavaScript evaluation while a session is degraded or fenced <!-- t:ife2 -->
- [x] [P3.6] Implement exact pending-record retry without rerunning JavaScript <!-- t:mdzb -->
- [x] [P3.7] Implement RecoverSession as unload plus restore from the last durable head <!-- t:gaez -->
- [x] [P3.8] Define and test the public result/error shape for executed-but-uncommitted cells <!-- t:ftf8 -->
- [x] [P3.9] Add failure-injection tests for append, child rows, transaction commit, retry, and recovery <!-- t:5izy -->
- [x] [P3.GATE] Phase 3 complete: durable append failure cannot produce later cell gaps or silent VM/journal divergence <!-- t:3jv4 -->

## Milestone B — Conditional persistent multi-process ownership

Phases 4–5 are required if separate CLI processes, a CLI plus server, or multiple servers may access the same persistent database/session. They may be replaced by a deliberately narrower database-wide exclusive-owner design only after task P5.1 records that deployment contract. They are not needed for a strictly in-memory REPL.

### Phase 4 — Transactional SQLite migrations

**Depends on:** Phase 3 and a decision to evolve the durable schema. **Exit:** Existing v1 data upgrades safely and schema version metadata cannot lie.

- [x] [P4.1] Commit a real schema-v1 SQLite fixture with representative session/evaluation data <!-- t:h3c3 -->
- [x] [P4.2] Replace unconditional schema-version stamping with ordered migration descriptors <!-- t:ptf2 -->
- [x] [P4.3] Implement empty-database bootstrap through the same migration path <!-- t:vter -->
- [x] [P4.4] Apply each migration transactionally and update version only after successful statements <!-- t:f8td -->
- [x] [P4.5] Reject databases newer than the binary supports with a typed error <!-- t:gl4j -->
- [x] [P4.6] Test v1 upgrade data preservation, failed migration rollback, and second-open idempotence <!-- t:znrl -->
- [x] [P4.7] Test concurrent database opens during migration <!-- t:yad6 -->
- [x] [P4.8] Document operator backup and upgrade expectations <!-- t:sc2z -->
- [x] [P4.GATE] Phase 4 complete: schema v1 upgrades safely and future ownership schema can land without version lies <!-- t:y2jp -->

### Phase 5 — Persistent ownership and fencing

**Depends on:** Phase 4 for per-session leases. **Decision point:** P5.1 chooses per-session leases or a simpler explicitly exclusive database owner. **Exit:** The supported deployment model prevents two writable live owners for one durable session.

- [x] [P5.1] Decide and document supported deployment contract: per-session lease or simpler database-wide exclusive owner <!-- t:7uxb -->
- [x] [P5.2] If leases are selected, add schema-v2 session_leases migration and indexes <!-- t:bftj -->
- [x] [P5.3] Add process-unique App owner ID and injectable clock for deterministic tests <!-- t:wnfz -->
- [x] [P5.4] Implement atomic lease acquire for absent, same-owner, expired, and conflicting-owner cases <!-- t:u522 -->
- [x] [P5.5] Implement lease renewal, release, expiry, and monotonically increasing fencing epoch <!-- t:6lvw -->
- [x] [P5.6] Add fenced evaluation append verifying owner, epoch, expiry, and expected next cell ID in one transaction <!-- t:0z4l -->
- [x] [P5.7] Acquire ownership before persistent create/restore publication and renew during long replay/evaluation <!-- t:t47k -->
- [x] [P5.8] Mark stale owners fenced before later JavaScript and discard/recover their live runtimes <!-- t:tr2e -->
- [x] [P5.9] Release ownership during unload/App.Close before Store.Close and preserve soft-delete semantics <!-- t:3yxd -->
- [x] [P5.10] Add fake-clock takeover, stale-fence, simultaneous-owner, expected-cell, and sequential CLI tests <!-- t:kltu -->
- [x] [P5.GATE] Phase 5 complete: one durable session cannot have two writable live owners under the supported deployment contract <!-- t:g1f2 -->

## Milestone C — Transport hardening and release integration

Phase 6 has a mandatory baseline if `goja-repl serve` remains supported: bounded bodies/source, redacted errors, server timeouts, and safe remote-bind behavior. Protobuf `ErrorResponse` and generated-client refinements may ship in the same phase or a focused follow-up, but the phase gate requires the selected public contract to be complete. Phase 7 is required for any release of Phases 1–6.

### Phase 6 — HTTP and protobuf hardening

**Depends on:** Phase 2; typed degraded/ownership mappings additionally depend on Phases 3 and 5. **Exit:** HTTP parsing is bounded and versioned, errors are safe and stable, and server defaults match the threat model.

- [x] [P6.1] Add HandlerConfig defaults for maximum request-body and JavaScript-source sizes <!-- t:bwo0 -->
- [x] [P6.2] Apply http.MaxBytesReader before reading and return 413 without full allocation <!-- t:xz62 -->
- [x] [P6.3] Validate application/json content type, decoded source size, and exact supported schema version <!-- t:qjru -->
- [x] [P6.4] Add protobuf ErrorResponse with stable code, safe message, and request ID fields <!-- t:l1p9 -->
- [x] [P6.5] Map typed app/session/store errors to 400, 404, 409, 413, 500, and 503 responses <!-- t:m6a6 -->
- [x] [P6.6] Redact internal SQL, path, panic, runtime, and plugin details while logging server-side context <!-- t:xcxm -->
- [x] [P6.7] Regenerate Go and TypeScript protobuf bindings and update conversion fixtures <!-- t:t1wa -->
- [x] [P6.8] Add read, write, idle, header, and maximum-header server settings compatible with evaluation timeout <!-- t:t52w -->
- [x] [P6.9] Require explicit acknowledgement or strong warning for non-loopback serve addresses and document module sandbox flags <!-- t:e3fs -->
- [x] [P6.10] Add real HTTP tests for limits, cancellation, versioning, error redaction, headers, and status mapping <!-- t:ntfs -->
- [x] [P6.GATE] Phase 6 complete: HTTP parsing is bounded/versioned and transport errors are stable without claiming built-in authentication <!-- t:3l36 -->

### Phase 7 — Host integration, documentation, and release validation

**Depends on:** Every phase selected for the release. **Exit:** CLI, TUI, and server close in the correct order; all generated artifacts, help, migration notes, tests, ticket records, and delivery documents match the implementation.

- [x] [P7.1] Update every replapi constructor call site to supply an intentional parent context <!-- t:tix2 -->
- [x] [P7.2] Make CLI runWithApp close App before Store with a bounded shutdown context <!-- t:0mgp -->
- [x] [P7.3] Make TUI shutdown close the owning App while preserving the adapter non-ownership contract <!-- t:9gzn -->
- [x] [P7.4] Make serve shutdown stop HTTP, wait handlers, close App/release ownership, then close Store <!-- t:wjna -->
- [x] [P7.5] Update Bobatea assistance and other WithRuntime callbacks for the context-bearing signature <!-- t:l1te -->
- [x] [P7.6] Update replapi Glazed help, REPL usage help, examples, and API comments to match implemented behavior <!-- t:734b -->
- [x] [P7.7] Add constructor/lifecycle migration notes and remove obsolete unsafe examples without compatibility shims <!-- t:ufyd -->
- [x] [P7.8] Run gofmt, full Go tests, focused race tests, go vet, and glazed-lint <!-- t:zrxi -->
- [x] [P7.9] Run buf lint/generate and replapi-types typecheck/tests when protobuf changes land <!-- t:mhok -->
- [x] [P7.10] Update ticket diary/changelog/relations, run docmgr doctor, and refresh the reMarkable bundle <!-- t:0kua -->
- [x] [P7.GATE] Phase 7 complete: all hosts close in the correct order and release documentation/validation is complete <!-- t:ka9w -->
