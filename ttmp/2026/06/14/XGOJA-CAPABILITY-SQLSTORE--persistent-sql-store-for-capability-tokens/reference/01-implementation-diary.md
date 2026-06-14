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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary for the capability SQL store implementation."
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: "Use this to resume or review the capability SQL store work."
WhenToUse: "Read before continuing XGOJA-CAPABILITY-SQLSTORE."
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
