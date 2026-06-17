---
Title: Appauth SQL Store Implementation Plan
Ticket: XGOJA-APPAUTH-SQLSTORE
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
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Demo host currently using appauth memory store
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: App auth interfaces and memory semantics
    - Path: pkg/gojahttp/auth/internal/appauthtest/store_contract.go
      Note: Reusable behavior contract for SQL store
ExternalSources: []
Summary: Plan for a database/sql-backed appauth store for users, Keycloak subjects, memberships, and resources.
LastUpdated: 2026-06-14T09:54:26.271101021-04:00
WhatFor: Use this to implement and review pkg/gojahttp/auth/appauth/sqlstore.
WhenToUse: When generic applications need persistent app-owned authorization state behind planned Express auth routes.
---


# Appauth SQL Store Implementation Plan

## Executive summary

`appauth` is the application authorization layer around external identity. Keycloak identifies a subject; `appauth` maps that subject to an app user, checks user status, models tenant memberships and roles, and resolves application resources for planned Express route authorization.

This ticket adds `pkg/gojahttp/auth/appauth/sqlstore`, a durable `database/sql` implementation of `appauth.UserStore`, `MembershipStore`, and `ResourceStore`. It keeps SQL details behind the existing Go interfaces so JavaScript route plans remain schema-agnostic.

## Current-state evidence

- `pkg/gojahttp/auth/appauth/appauth.go` defines `UserStore`, `MembershipStore`, `ResourceStore`, `Resolver`, `Authorizer`, and `MemoryStore`.
- `pkg/gojahttp/auth/internal/appauthtest/store_contract.go` defines reusable behavior: user lookup by ID/sub, OIDC upsert, disabled-user hiding, revoked membership filtering, role checks, resource lookup, and clone isolation.
- The Keycloak demo currently uses SQL-backed sessions/audit but still seeds users/resources/memberships in memory. This is the remaining gap for a fuller production-shaped demo.

## Proposed package API

```go
store, err := sqlstore.New(sqlstore.Config{
    DB:      db,
    Dialect: sqlstore.DialectPostgres,
})
err = store.ApplySchema(ctx)

user, err := store.UpsertFromOIDC(ctx, sub, email, verified)
resolver := appauth.Resolver{Store: store}
authorizer := appauth.Authorizer{Memberships: store}
```

For examples and tests, the SQL store should include seeding helpers equivalent to the memory store:

```go
err = store.AddUser(ctx, appauth.User{...})
err = store.AddMembership(ctx, appauth.Membership{...})
err = store.AddResource(ctx, appauth.Resource{...})
```

These helpers are not required by the runtime interfaces but make demos, tests, and migrations straightforward.

## Schema design

Tables:

- `auth_app_users`
  - `id TEXT PRIMARY KEY`
  - `keycloak_sub TEXT UNIQUE`
  - `email TEXT NOT NULL DEFAULT ''`
  - `display_name TEXT NOT NULL DEFAULT ''`
  - `email_verified BOOLEAN NOT NULL`
  - `disabled_at TIMESTAMP/TIMESTAMPTZ NULL`
- `auth_app_tenants`
  - `id TEXT PRIMARY KEY`
  - `slug TEXT UNIQUE`
  - `name TEXT NOT NULL DEFAULT ''`
  - `disabled_at TIMESTAMP/TIMESTAMPTZ NULL`
- `auth_app_memberships`
  - `user_id TEXT NOT NULL`
  - `tenant_id TEXT NOT NULL`
  - `role TEXT NOT NULL`
  - `revoked_at TIMESTAMP/TIMESTAMPTZ NULL`
  - primary key `(user_id, tenant_id, role)`
- `auth_app_resources`
  - `type TEXT NOT NULL`
  - `id TEXT NOT NULL`
  - `name TEXT NOT NULL DEFAULT ''`
  - `tenant_id TEXT NOT NULL DEFAULT ''`
  - `owner_id TEXT NOT NULL DEFAULT ''`
  - `claims_json` (`TEXT` in SQLite, `JSONB` in Postgres)
  - primary key `(type, id)`

The initial implementation does not enforce foreign keys by default because host apps may hydrate users/resources from external systems in different orders. Application migrations can add stricter constraints if desired.

## OIDC upsert semantics

`UpsertFromOIDC` must preserve the disabled-user rule:

1. If `keycloak_sub` exists and the user is disabled, return `gojahttp.ErrNotFound` and do not update the row.
2. If it exists and is enabled, update email and verification status and return the user.
3. If it does not exist, create `id = "user:" + sub` with the OIDC email fields.

## Testing strategy

- Add `pkg/gojahttp/auth/appauth/sqlstore/sqlstore_test.go` using SQLite in-memory databases.
- Run `appauthtest.RunStoreContract` against a harness backed by SQL seeding helpers.
- Add a focused test for disabled-user OIDC upsert if contract coverage is not enough.
- Validate:
  ```bash
  go test ./pkg/gojahttp/auth/appauth/... -count=1
  ```

## Demo/docs integration

Update the Keycloak host demo to replace in-memory `appauth.MemoryStore` with `appauth/sqlstore` when a SQL DSN is supplied. Seed the demo org, project, user membership, and user resource into SQL. The smoke should then exercise SQL-backed sessions, audit, appauth users/memberships/resources, and capability tokens once the capability SQL store is wired.

## Risks and review focus

- Disabled users must stay hidden and must not be re-enabled by OIDC callback upsert.
- Resource claims are JSON and should be cloned/decoded so callers cannot mutate internal state.
- Membership queries must filter revoked rows everywhere.
- The default `Authorizer` remains intentionally simple and deny-by-default; the SQL store is storage, not a policy engine.
