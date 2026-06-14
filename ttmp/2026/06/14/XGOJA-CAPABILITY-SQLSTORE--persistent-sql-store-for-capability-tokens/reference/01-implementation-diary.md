---
Title: Implementation diary
Ticket: XGOJA-CAPABILITY-SQLSTORE
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Capability SQL demo wiring and invite endpoints
    - Path: examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
      Note: Invite issue/accept smoke flow
    - Path: examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh
      Note: Persisted capability row assertion
    - Path: pkg/gojahttp/auth/capability/sqlstore/schema.go
      Note: SQLite/Postgres auth_capabilities schema
    - Path: pkg/gojahttp/auth/capability/sqlstore/sqlstore.go
      Note: SQL capability store implementation and atomic redemption
    - Path: pkg/gojahttp/auth/capability/sqlstore/sqlstore_test.go
      Note: SQLite contract tests for SQL store
ExternalSources: []
Summary: Chronological diary for the capability SQL store implementation.
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: Use this to resume or review the capability SQL store work.
WhenToUse: Read before continuing XGOJA-CAPABILITY-SQLSTORE.
---



# Diary

## Goal

This diary records the design and implementation of a persistent SQL store for capability tokens used by host-owned auth flows such as organization invites and narrow bearer delegations.

## Step 1: Create ticket and design the capability SQL store

I created a focused docmgr ticket for moving capability token storage from memory-only demos to a durable SQL adapter. The design keeps the existing `capability.Store` interface as the seam and treats raw-token handling as the main security invariant: raw tokens are returned once by `Service.Issue`, while the SQL store persists only `HashToken(token)`.

The design also calls out atomic single-use redemption as the correctness-critical behavior. A production SQL store must let exactly one concurrent redeemer mark `used_at`, while all other concurrent attempts receive `capability.ErrUsed`.

### Prompt Context

**User prompt (verbatim):** "ok, create two tickets for capability store and appauth store. Add design documents to each, add tasks, and then implement them task by task, committing at appropriate intervals, keeping a detailed diary. Update the docs and demo too."

**Assistant interpretation:** Create separate docmgr tickets for persistent capability and appauth stores, write design docs and task lists, then implement the stores incrementally with commits, diary entries, docs, and demo updates.

**Inferred user intent:** Finish the production-persistence follow-up for host-owned Express auth so the Keycloak demo can exercise durable sessions, audit, appauth, and capabilities.

**Commit (code):** N/A — ticket/design setup before implementation.

### What I did
- Created ticket `XGOJA-CAPABILITY-SQLSTORE`.
- Added `design/01-capability-sql-store-implementation-plan.md`.
- Added this diary document.
- Added phased tasks for ticket setup, SQL store implementation, and demo/docs integration.
- Inspected existing capability interfaces and contract tests.
- Used `sessionauth/sqlstore` and `audit/sqlstore` as implementation patterns.

### Why
- Capability tokens are security-sensitive and need durable storage before they can support production-shaped flows such as invite acceptance or scoped links.
- The existing memory store is useful for tests but cannot survive process restarts or run across multiple app instances.
- Keeping the adapter behind `capability.Store` preserves the host-owned security model and avoids exposing SQL details to JavaScript route plans.

### What worked
- The existing `capability.Store` interface is small enough for a direct SQL adapter.
- The reusable `capabilitytest.RunStoreContract` already captures the important semantics, including concurrent single-use redemption.
- Existing SQL store packages provide a clear style for dialects, schemas, and `ApplySchema`.

### What didn't work
- No implementation commands failed in this step.
- No tests were run yet because this was ticket/design setup.

### What I learned
- The contract test requires clone isolation for both input and returned values.
- The SQL adapter must treat purpose mismatches, expiry, revocation, and used single-use capabilities as domain errors from the `capability` package.
- Raw tokens must not appear in schema, audit, logs, or docs except as one-time caller-returned values.

### What was tricky to build
- The main design sharp edge is atomic redemption. A naive select-then-update can allow two concurrent redeemers to succeed. The planned approach is a transaction plus a conditional `UPDATE ... WHERE used_at IS NULL`, checking affected rows before returning success.
- Another subtlety is that `ByID` is intentionally for administrative inspection and should return the stored hash, while service-level `Issue` and `Redeem` redact token hashes before returning to callers.

### What warrants a second pair of eyes
- Review the final SQL redemption path for race safety under Postgres and SQLite.
- Review whether `token_hash` should be `BYTEA/BLOB` rather than hex text; the design currently chooses binary storage.
- Review indexes for operational cleanup queries around `expires_at` and `revoked_at`.

### What should be done in the future
- Implement `capability/sqlstore` and run the store contract.
- Add Keycloak demo invite issue/accept coverage using persisted capability rows.

### Code review instructions
- Start with `pkg/gojahttp/auth/capability/capability.go` for interface semantics.
- Then review `pkg/gojahttp/auth/internal/capabilitytest/store_contract.go` for required behavior.
- Validate implementation with `go test ./pkg/gojahttp/auth/capability/... -count=1`.

### Technical details
- Planned package: `pkg/gojahttp/auth/capability/sqlstore`.
- Planned table: `auth_capabilities`.
- Planned validation command:
  ```bash
  go test ./pkg/gojahttp/auth/capability/... -count=1
  ```

## Step 2: Implement capability/sqlstore

I implemented the durable `database/sql` adapter for capability tokens. The package follows the existing SQL store shape: explicit dialect selection, separate SQLite/Postgres schema constants, `ApplySchema(ctx)` for examples and tests, and SQLite contract coverage through the reusable capability store contract.

The most important behavior is single-use redemption. The store loads the capability by token hash inside a transaction, validates purpose/expiry/revocation/used state, and then marks `used_at` with a conditional update that only succeeds while `used_at IS NULL`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the first persistent auth store task-by-task and keep validation/diary state current.

**Inferred user intent:** Make capability tokens production-shaped by persisting token hashes and enforcing single-use redemption safely.

**Commit (code):** pending — capability SQL store implementation in progress.

### What I did
- Added `pkg/gojahttp/auth/capability/sqlstore/schema.go` with SQLite and Postgres DDL for `auth_capabilities`.
- Added `pkg/gojahttp/auth/capability/sqlstore/sqlstore.go` implementing `capability.Store`.
- Added `pkg/gojahttp/auth/capability/sqlstore/sqlstore_test.go` using `capabilitytest.RunStoreContract` against SQLite.
- Added generated-style `logcopter.go` stub for the new package.
- Ran:
  ```bash
  go test ./pkg/gojahttp/auth/capability/... -count=1
  ```

### Why
- Capability tokens need persistence for invite, reset, verification, and narrow delegation flows.
- The existing memory store semantics are correct but process-local.
- The contract test gives a stable definition of production behavior for any future store implementation.

### What worked
- The final targeted test run passed:
  ```text
  ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability	0.004s
  ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore	0.007s
  ```
- The implementation stores only `token_hash` and never stores raw tokens.
- The contract test's concurrent single-use case passed after fixing SQLite connection handling.

### What didn't work
- The first SQL store contract run failed during the concurrent redemption test:
  ```text
  --- FAIL: TestSQLiteStoreContract (0.00s)
      --- FAIL: TestSQLiteStoreContract/single_use_redemption_is_atomic (0.00s)
          store_contract.go:148: unexpected concurrent redeem err=scan capability: no such table: auth_capabilities
  FAIL
  FAIL	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore	0.010s
  ```
- Cause: SQLite `:memory:` databases are per connection, and concurrent test goroutines opened separate connections that did not share the schema.
- Fix: set `db.SetMaxOpenConns(1)` in the SQLite contract harness so all goroutines use the same in-memory database connection.

### What I learned
- The existing contract test is strong enough to catch both clone isolation and concurrency hazards.
- For in-memory SQLite tests in Go, connection pool behavior matters as much as SQL logic.
- The same conditional-update redemption design maps cleanly to both SQLite and Postgres using dialect-specific placeholder constants.

### What was tricky to build
- The race-safe redemption path needed to distinguish domain failures from SQL errors. Purpose mismatch, expiry, revocation, and used state return `capability` package errors; SQL scan/transaction failures are wrapped with operation context.
- The store's `ByID` intentionally returns the stored hash because it is an administrative store API, while the service layer redacts hashes before returning issued/redeemed capabilities to application callers.

### What warrants a second pair of eyes
- Review the transaction isolation assumptions for Postgres concurrent single-use redemption.
- Review whether `token_hash BYTEA/BLOB UNIQUE` is preferred over a hex string for operational debugging and indexing.
- Review whether revocation should be idempotent or continue returning `ErrNotFound` for missing IDs; the implementation follows the current memory-store contract.

### What should be done in the future
- Wire this store into the Keycloak demo and smoke so persisted capability rows are exercised outside unit tests.
- Add operational cleanup examples for expired/revoked capabilities if needed.

### Code review instructions
- Start with `pkg/gojahttp/auth/capability/sqlstore/schema.go` for table shape.
- Then review `pkg/gojahttp/auth/capability/sqlstore/sqlstore.go`, especially `Redeem`.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/capability/... -count=1
  ```

### Technical details
- Atomic single-use update query shape:
  ```sql
  UPDATE auth_capabilities SET used_at = ? WHERE id = ? AND used_at IS NULL
  ```
- Postgres uses the same logic with `$1` / `$2` placeholders.

## Step 3: Wire capability/sqlstore into the Keycloak demo

I updated the production-shaped Keycloak example so the demo host can persist capability tokens in Postgres alongside sessions, audit records, and appauth state. The demo now exposes a small org-invite flow: an authenticated admin issues a single-use invite capability and a public endpoint redeems it exactly once.

This makes capability persistence visible in the end-to-end smoke instead of only in unit tests. The smoke verifies successful invite issue, successful invite accept, failed token reuse, and a persisted `auth_capabilities` row with `used_at` set.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Update the full demo so capability SQL persistence is exercised end-to-end, then record validation and review notes.

**Inferred user intent:** Provide a runnable proof that the new persistent capability store works in the production-shaped Keycloak flow.

**Commit (code):** pending — demo/docs integration in progress.

### What I did
- Added `--capability-db-dsn` / `CAPABILITY_DB_DSN` support to the Keycloak host.
- Added `newCapabilityService` to use `capability/sqlstore` with Postgres when configured, falling back to `capability.NewMemoryStore()` otherwise.
- Added `POST /orgs/o1/invites` to issue a CSRF-protected, admin-authorized org invite capability.
- Added `POST /org-invites/accept` to redeem invite tokens and reject reuse.
- Updated `scripts/keycloak_smoke.py` to issue and accept an invite and verify reuse returns 409.
- Updated `scripts/smoke.sh` to pass the capability DSN and assert persisted used capability rows.
- Updated the example README with capability persistence notes.

### Why
- SQL store contract tests prove adapter correctness, but the user asked for docs and demo updates too.
- The invite flow is the smallest concrete capability-token use case already present in the package API.
- Persisting and checking `used_at` in Postgres demonstrates that raw token redemption changes durable state.

### What worked
- Targeted tests passed:
  ```bash
  go test ./examples/xgoja/19-express-keycloak-auth-host/cmd/host ./pkg/gojahttp/auth/appauth/... ./pkg/gojahttp/auth/capability/... -count=1
  ```
- The full Keycloak smoke passed:
  ```text
  ok invite issue                 200
  ok invite accept                200
  ok invite accept reused         409
  ok persisted capability records 1
  ```

### What didn't work
- No demo smoke failures occurred after the handlers and smoke assertions were added.

### What I learned
- Capability flows can stay entirely host-owned; JavaScript route plans do not need to know token storage details.
- The existing `IssueOrgInvite` / `AcceptOrgInvite` helpers are sufficient for a concise end-to-end demo.

### What was tricky to build
- The tricky part was keeping the invite flow small while still security-shaped. The issue endpoint verifies an app session, enforces CSRF, and checks `appauth.ActionOrgInvite` before issuing the token.
- The accept endpoint is intentionally public because invite links are bearer capabilities; its safety comes from token entropy, expiry, single-use state, and hash-only persistence.

### What warrants a second pair of eyes
- Review whether the demo should keep invite issue/accept as Go host endpoints or move route declarations into JavaScript once `.body(...)` exists.
- Review whether 409 is the right status for used/expired/revoked capability tokens.

### What should be done in the future
- Add production docs for operational cleanup of expired capability rows.

### Code review instructions
- Review `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go`, especially `newCapabilityService`, `issueInviteHandler`, and `acceptInviteHandler`.
- Review `examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py` for the invite flow.
- Validate with `make -C examples/xgoja/19-express-keycloak-auth-host smoke`.

### Technical details
- Smoke SQL assertion:
  ```sql
  SELECT count(*) FROM auth_capabilities
  WHERE purpose = 'org.invite.accept' AND used_at IS NOT NULL;
  ```
