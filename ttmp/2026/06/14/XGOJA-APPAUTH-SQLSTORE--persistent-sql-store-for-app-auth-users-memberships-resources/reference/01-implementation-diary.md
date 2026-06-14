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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary for the appauth SQL store implementation."
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: "Use this to resume or review the appauth SQL store work."
WhenToUse: "Read before continuing XGOJA-APPAUTH-SQLSTORE."
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
