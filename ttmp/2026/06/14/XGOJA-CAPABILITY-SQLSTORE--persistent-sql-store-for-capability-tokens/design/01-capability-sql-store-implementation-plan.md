---
Title: Capability SQL Store Implementation Plan
Ticket: XGOJA-CAPABILITY-SQLSTORE
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/gojahttp/auth/capability/capability.go
      Note: Capability Store interface and memory semantics
    - Path: pkg/gojahttp/auth/internal/capabilitytest/store_contract.go
      Note: Reusable behavior contract for SQL store
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go
      Note: Reference database/sql store pattern
ExternalSources: []
Summary: Plan for a database/sql-backed capability token store with SQLite/Postgres schemas, atomic single-use redemption, and demo integration.
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: Use this to implement and review pkg/gojahttp/auth/capability/sqlstore.
WhenToUse: When productionizing capability tokens such as organization invites or narrow delegation links.
---


# Capability SQL Store Implementation Plan

## Executive summary

The existing `pkg/gojahttp/auth/capability` package defines the right security boundary: raw tokens are returned once, only a hash is stored, and the `Store` interface owns create, redeem, revoke, and lookup behavior. The missing production piece is a durable SQL adapter that keeps those semantics under concurrency, especially single-use redemption.

This ticket adds `pkg/gojahttp/auth/capability/sqlstore`, mirroring the existing `sessionauth/sqlstore` and `audit/sqlstore` style: `database/sql`, explicit SQLite/Postgres dialects, `ApplySchema(ctx)` for examples/tests, contract tests with SQLite, and no leakage of raw capability tokens.

## Current-state evidence

- `pkg/gojahttp/auth/capability/capability.go` defines `Store`, `Capability`, `Service`, `HashToken`, and memory semantics.
- `pkg/gojahttp/auth/internal/capabilitytest/store_contract.go` already specifies clone isolation, redemption validation, expiry, revocation, not-found behavior, and concurrent single-use atomicity.
- `pkg/gojahttp/auth/sessionauth/sqlstore` and `pkg/gojahttp/auth/audit/sqlstore` establish the project pattern for dialects, schemas, `ApplySchema`, SQLite tests, placeholder constants, and generated logcopter stubs.

## Proposed package API

```go
store, err := sqlstore.New(sqlstore.Config{
    DB:      db,
    Dialect: sqlstore.DialectPostgres,
})
err = store.ApplySchema(ctx)

service := capability.Service{Store: store, Audit: auditSink}
issued, err := service.IssueOrgInvite(ctx, capability.OrgInviteSpec{...})
accepted, err := service.AcceptOrgInvite(ctx, issued.Token)
```

## Schema design

Table: `auth_capabilities`

- `id TEXT PRIMARY KEY`
- `purpose TEXT NOT NULL`
- `subject_id TEXT`
- `resource_type TEXT`
- `resource_id TEXT`
- `claims_json` (`TEXT` in SQLite, `JSONB` in Postgres)
- `token_hash BLOB/BYTEA NOT NULL UNIQUE`
- `expires_at TIMESTAMP/TIMESTAMPTZ NOT NULL`
- `single_use BOOLEAN NOT NULL`
- `used_at TIMESTAMP/TIMESTAMPTZ NULL`
- `revoked_at TIMESTAMP/TIMESTAMPTZ NULL`
- `created_by TEXT`
- `created_at TIMESTAMP/TIMESTAMPTZ NOT NULL`

Indexes should support lookup by token hash, purpose, resource, subject, expiry, and revocation.

## Atomic redemption algorithm

Redemption must be one transaction:

1. Select by `token_hash`.
2. Validate purpose, revoked, expiry, and single-use `used_at` state.
3. If `single_use`, update `used_at` only when it is still `NULL`:
   - SQLite/Postgres use a dialect-specific `UPDATE ... WHERE id = ? AND used_at IS NULL` / `$1` query.
   - If zero rows are affected, return `capability.ErrUsed`.
4. Return a cloned/redacted `Capability` with `UsedAt` set for successful single-use redemption.

This is sufficient for SQLite contract tests and safe for Postgres row-level atomicity without exposing raw tokens.

## Testing strategy

- Add `pkg/gojahttp/auth/capability/sqlstore/sqlstore_test.go` using SQLite in-memory databases.
- Run `capabilitytest.RunStoreContract` against the SQL store.
- Add focused tests for persisted token hash shape if needed.
- Validate:
  ```bash
  go test ./pkg/gojahttp/auth/capability/... -count=1
  ```

## Demo/docs integration

The Keycloak demo should expose capability persistence as optional host-auth infrastructure. A practical smoke extension can issue and accept an organization invite through host endpoints backed by `capability/sqlstore`; at minimum the example docs should show how to wire the SQL store and explain that raw tokens are never persisted.

## Risks and review focus

- Atomic single-use redemption is the correctness-critical path.
- Token hashes must remain binary/hash-only; raw tokens must never be logged or stored.
- Dialect query constants should avoid dynamic SQL placeholder concatenation that scanners flag.
- Claims JSON decoding must preserve empty maps and return useful errors for malformed rows.
