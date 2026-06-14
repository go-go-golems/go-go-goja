---
Title: Implementation diary
Ticket: XGOJA-APPAUTH-SQLSTORE
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
    - Path: pkg/gojahttp/auth/appauth/sqlstore/schema.go
      Note: SQLite/Postgres appauth schema
    - Path: pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go
      Note: SQL appauth store implementation
    - Path: pkg/gojahttp/auth/appauth/sqlstore/sqlstore_test.go
      Note: SQLite contract tests for SQL store
ExternalSources: []
Summary: Chronological diary for the appauth SQL store implementation.
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: Use this to resume or review the appauth SQL store work.
WhenToUse: Read before continuing XGOJA-APPAUTH-SQLSTORE.
---


# Diary

## Goal

This diary records the design and implementation of a persistent SQL store for app-owned auth data: users, Keycloak subject mappings, tenant memberships, roles, and route resources.

## Step 1: Create ticket and design the appauth SQL store

I created a focused docmgr ticket for turning `appauth` from an in-memory/custom-store abstraction into a reusable SQL-backed foundation for generic applications. The important boundary is that Keycloak remains the identity provider, while `appauth` owns application authorization state: users, tenant membership, roles, resources, and disabled-user checks.

The design keeps SQL behind the existing `appauth.UserStore`, `MembershipStore`, and `ResourceStore` interfaces. JavaScript planned routes continue to declare intent with `.resource(...)` and `.allow(...)`; the Go host decides how users and resources are resolved and authorized.

### Prompt Context

**User prompt (verbatim):** "ok, create two tickets for capability store and appauth store. Add design documents to each, add tasks, and then implement them task by task, committing at appropriate intervals, keeping a detailed diary. Update the docs and demo too."

**Assistant interpretation:** Create separate docmgr tickets for persistent capability and appauth stores, write design docs and task lists, then implement the stores incrementally with commits, diary entries, docs, and demo updates.

**Inferred user intent:** Finish the production-persistence follow-up for host-owned Express auth so the Keycloak demo can exercise durable sessions, audit, appauth, and capabilities.

**Commit (code):** N/A — ticket/design setup before implementation.

### What I did
- Created ticket `XGOJA-APPAUTH-SQLSTORE`.
- Added `design/01-appauth-sql-store-implementation-plan.md`.
- Added this diary document.
- Added phased tasks for ticket setup, SQL store implementation, and demo/docs integration.
- Inspected existing `appauth` interfaces and reusable store contract tests.
- Used `sessionauth/sqlstore` and `audit/sqlstore` as package/style references.

### Why
- The current Keycloak demo persists app sessions and audit rows, but users, memberships, and resources are still seeded in memory.
- Generic applications need a durable default for app-specific authorization without putting SQL or policy details into JavaScript route declarations.
- `appauth/sqlstore` lets the host keep ownership of authorization state while making the demo production-shaped across process restarts.

### What worked
- `appauth` already separates user, membership, and resource interfaces from the authorizer/resolver helpers.
- The contract test already covers the key behavior: disabled users hidden, OIDC upsert, revoked membership filtering, role checks, resource lookup, and claim clone isolation.
- The existing memory store provides straightforward seed helper semantics to mirror in SQL.

### What didn't work
- No implementation commands failed in this step.
- No tests were run yet because this was ticket/design setup.

### What I learned
- `UpsertFromOIDC` must not re-enable or return disabled users when Keycloak still authenticates the same subject.
- `Authorizer` is intentionally not a policy engine; it is a small deny-by-default starting point over app-owned memberships.
- Resources need JSON claims, but route plans should only see normalized `gojahttp.ResourceRef` values.

### What was tricky to build
- The main design tension is deciding how much schema to standardize. The plan chooses a minimal reusable schema for users, tenants, memberships, and generic resources, while leaving advanced policy-engine integration and app-specific constraints to future migrations.
- Another subtlety is OIDC upsert semantics. Creating a new `user:<sub>` row is useful for demos and simple apps, but existing disabled users must fail closed and remain unchanged.

### What warrants a second pair of eyes
- Review the final schema names and whether generic resource storage is sufficient for examples without encouraging apps to avoid domain-specific tables.
- Review whether foreign keys should remain optional in the default schema.
- Review disabled-user behavior in SQL upsert carefully.

### What should be done in the future
- Implement `appauth/sqlstore` and run the store contract.
- Wire the Keycloak demo to seed users, memberships, and resources into SQL.

### Code review instructions
- Start with `pkg/gojahttp/auth/appauth/appauth.go` for interface and memory-store semantics.
- Then review `pkg/gojahttp/auth/internal/appauthtest/store_contract.go` for required behavior.
- Validate implementation with `go test ./pkg/gojahttp/auth/appauth/... -count=1`.

### Technical details
- Planned package: `pkg/gojahttp/auth/appauth/sqlstore`.
- Planned tables: `auth_app_users`, `auth_app_tenants`, `auth_app_memberships`, `auth_app_resources`.
- Planned validation command:
  ```bash
  go test ./pkg/gojahttp/auth/appauth/... -count=1
  ```

## Step 2: Implement appauth/sqlstore

I implemented the durable `database/sql` adapter for app-owned authorization state. The new package persists users, Keycloak subject mappings, tenants, memberships, and generic resources while still satisfying the existing `appauth.UserStore`, `MembershipStore`, and `ResourceStore` interfaces.

The implementation mirrors the memory-store behavior that the planned route stack already depends on: disabled users are hidden, OIDC upsert fails closed for disabled users, revoked memberships are filtered out, role checks are explicit, and resource claims are decoded into caller-owned maps.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the appauth persistent store after designing the ticket, validating behavior through the reusable contract.

**Inferred user intent:** Give generic applications a reusable SQL-backed foundation for application authorization around Keycloak/OIDC identity.

**Commit (code):** pending — appauth SQL store implementation in progress.

### What I did
- Added `pkg/gojahttp/auth/appauth/sqlstore/schema.go` with SQLite and Postgres DDL for:
  - `auth_app_users`,
  - `auth_app_tenants`,
  - `auth_app_memberships`,
  - `auth_app_resources`.
- Added `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go` implementing:
  - `appauth.UserStore`,
  - `appauth.MembershipStore`,
  - `appauth.ResourceStore`,
  - `AddUser`, `AddTenant`, `AddMembership`, and `AddResource` bootstrap helpers.
- Added SQLite contract coverage in `pkg/gojahttp/auth/appauth/sqlstore/sqlstore_test.go` using `appauthtest.RunStoreContract`.
- Added generated-style `logcopter.go` stub for the new package.
- Ran:
  ```bash
  go test ./pkg/gojahttp/auth/appauth/... -count=1
  ```

### Why
- The Keycloak demo already persists sessions and audit, but app-owned authorization data was still memory-only.
- Generic applications need a default durable store for user/tenant/resource authorization data without putting SQL into JavaScript route declarations.
- Keeping the adapter behind the existing interfaces lets hosts replace it with custom domain stores or policy engines later.

### What worked
- The final targeted test run passed:
  ```text
  ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth	0.005s
  ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore	0.006s
  ```
- The reusable appauth contract covered OIDC upsert, disabled-user hiding, membership filtering, role checks, resource lookup, and resource claim clone isolation.
- The seeding helpers make the SQL store easy to use in examples and smoke tests.

### What didn't work
- No test failures occurred during appauth/sqlstore implementation.
- I reused the SQLite `db.SetMaxOpenConns(1)` harness pattern from the capability SQL store to avoid per-connection in-memory database surprises.

### What I learned
- The appauth interfaces were already small enough for one SQL store to implement all three runtime roles.
- OIDC upsert is the most important security-sensitive user-store method because it can otherwise accidentally re-enable disabled Keycloak subjects.
- Generic resource storage is enough for route authorization demos, but real applications may still choose domain-specific resource tables.

### What was tricky to build
- The main tricky part was avoiding dynamic query construction while still supporting SQLite and Postgres placeholders. The implementation uses dialect-specific query constants for each operation.
- Another subtlety is that SQL seeding helpers are not part of the runtime interfaces. They are intentionally concrete `*Store` methods for tests, examples, and simple bootstrap flows.

### What warrants a second pair of eyes
- Review whether the default schema should include foreign keys or remain loose for host bootstrap flexibility.
- Review `UpsertFromOIDC` to ensure disabled users cannot be updated or returned.
- Review whether `auth_app_resources` should remain generic or whether docs should strongly encourage domain-specific resource tables for production apps.

### What should be done in the future
- Wire this store into the Keycloak demo so appauth users/memberships/resources persist in Postgres.
- Add smoke assertions for persisted appauth rows.

### Code review instructions
- Start with `pkg/gojahttp/auth/appauth/sqlstore/schema.go` for schema shape.
- Then review `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go`, especially `UpsertFromOIDC`, membership queries, and resource claims JSON handling.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/appauth/... -count=1
  ```

### Technical details
- OIDC disabled-user rule:
  ```text
  existing subject + disabled_at != NULL -> gojahttp.ErrNotFound, no update
  ```
- Membership queries always include:
  ```sql
  revoked_at IS NULL
  ```
