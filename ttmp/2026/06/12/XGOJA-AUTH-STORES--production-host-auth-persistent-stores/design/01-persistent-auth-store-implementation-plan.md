---
Title: Persistent auth store implementation plan
Ticket: XGOJA-AUTH-STORES
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: Defines app-owned user
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Defines audit.Store and normalized redacted audit records
    - Path: pkg/gojahttp/auth/audit/sqlstore/schema.go
      Note: Audit SQL schema referenced by implementation plan
    - Path: pkg/gojahttp/auth/audit/sqlstore/sqlstore.go
      Note: Durable audit store adapter implementing the audit persistence phase
    - Path: pkg/gojahttp/auth/capability/capability.go
      Note: Defines capability.Store and atomic redemption requirements
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Defines sessionauth.Store and session security semantics for persistent store work
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/schema.go
      Note: Session SQL schema referenced by implementation plan
    - Path: pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go
      Note: First durable store adapter implementing the session persistence phase
ExternalSources: []
Summary: Plan persistent production stores for gojahttp sessions, audit records, capability tokens, and app-owned auth domain data.
LastUpdated: 2026-06-12T20:29:29.994874399-04:00
WhatFor: Use when implementing production-grade persistence for the host auth packages introduced by the Express planned-auth work.
WhenToUse: Before replacing in-memory session, audit, capability, user, membership, tenant, or resource stores with durable stores.
---




# Persistent auth store implementation plan

## Executive summary

This ticket tracks the production persistence layer for the host-side auth packages. The current implementation intentionally starts with in-memory stores so planned Express auth can be demonstrated and tested without external services. Production hosts need durable stores for server-side sessions, audit records, capability tokens, users, tenants, memberships, and resources.

The main design rule is to keep persistence behind existing Go interfaces. `modules/express` and JavaScript route plans should not learn about SQL schemas. The host chooses stores, wires them into `gojahttp.HostOptions.Auth`, and keeps application authorization app-owned.

## Scope

Implement durable store adapters and migrations for these areas:

1. `sessionauth.Store` for opaque server-side app sessions.
2. `audit.Store` for normalized, redacted audit records.
3. `capability.Store` for hashed, expiring, optionally single-use bearer capabilities.
4. App-owned user, tenant, membership, and resource storage sufficient to back `appauth.Resolver` and `appauth.Authorizer` in production examples.

Postgres should be the default target unless a project-level database decision says otherwise. SQLite can be useful for tests, but the production path should prove transactional behavior on Postgres.

## Current-state evidence

The current host auth packages already define the persistence seams:

| Package | Existing seam | Current implementation |
| --- | --- | --- |
| `sessionauth` | `Store` with `Create`, `Get`, `Touch`, `Rotate`, `Revoke` | `MemoryStore` |
| `audit` | `Store.InsertAuditRecord` | `MemorySink`, `LogSink`, `Sink{Store}` |
| `capability` | `Store` with `Create`, `Redeem`, `Revoke`, `ByID` | `MemoryStore` |
| `appauth` | `UserStore`, `MembershipStore`, `ResourceStore` | `MemoryStore` |

Relevant files:

```text
pkg/gojahttp/auth/sessionauth/sessionauth.go
pkg/gojahttp/auth/audit/audit.go
pkg/gojahttp/auth/capability/capability.go
pkg/gojahttp/auth/appauth/appauth.go
examples/xgoja/18-express-auth-host/cmd/host/main.go
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
```

## Proposed package layout

Prefer small storage subpackages over adding database dependencies to the in-memory packages directly:

```text
pkg/gojahttp/auth/sessionauth/sqlstore
pkg/gojahttp/auth/audit/sqlstore
pkg/gojahttp/auth/capability/sqlstore
pkg/gojahttp/auth/appauth/sqlstore
```

Each package should expose:

- migrations or schema DDL,
- constructor accepting `*sql.DB` or a minimal query interface,
- interface conformance tests,
- Postgres-focused integration tests where transactions matter,
- no Express or JavaScript dependencies.

## Store semantics

### Sessions

Session persistence must preserve security semantics:

- session IDs are opaque random tokens;
- expired and revoked sessions are rejected;
- `Touch` extends idle expiry without changing absolute expiry;
- `Rotate` replaces old ID with a new ID atomically;
- `MFAAt` is stored and loaded because `sessionauth.Manager.Authenticate` enforces `.mfaFresh(...)` against it;
- cookie settings remain `sessionauth.Manager` concerns, not store concerns.

### Audit

Audit storage must preserve incident-review usefulness without storing secrets:

- store normalized `audit.Record` fields;
- keep event, outcome, route, action, actor, tenant, resource, request ID, IP hash, user agent, status, reason, attributes, and timestamp;
- do not store raw cookies, session IDs, access tokens, capability tokens, authorization headers, passwords, or OAuth codes;
- tests should prove redaction before insert.

### Capabilities

Capability storage has the strongest transactional requirement:

- store token hashes, never raw tokens;
- `Redeem` must be atomic for single-use tokens;
- expired, revoked, wrong-purpose, and already-used capabilities must fail closed;
- revocation should be idempotent;
- org-invite helpers should work on top of the generic store.

A SQL implementation should use a transaction or conditional update such as:

```sql
UPDATE auth_capabilities
SET used_at = now()
WHERE token_hash = $1
  AND purpose = $2
  AND revoked_at IS NULL
  AND expires_at > now()
  AND (single_use = false OR used_at IS NULL)
RETURNING *;
```

### App auth domain data

The first production appauth store should stay intentionally small:

- users keyed by app user ID and Keycloak subject;
- tenants;
- memberships with roles and revocation;
- resources or a pluggable resource table strategy;
- `UpsertFromOIDC` for Keycloak callback normalization;
- deny-by-default authorization semantics preserved.

## Implementation phases

### Phase 1 — Store contract tests

Create reusable tests for each store interface so memory and SQL stores run through the same behavior. This prevents SQL adapters from drifting from the already-tested in-memory behavior.

### Phase 2 — Session SQL store

Implement session DDL, store adapter, expiry/revocation behavior, `Touch`, and `Rotate` atomicity. Include MFA timestamp persistence tests.

### Phase 3 — Audit SQL store

Implement audit DDL and insert adapter. Add tests proving redaction and queryable records for allowed, denied, completed, and failed route outcomes.

### Phase 4 — Capability SQL store

Implement capability DDL and atomic redemption. Add concurrency tests for single-use redemption.

### Phase 5 — App auth SQL starter store

Implement a starter appauth SQL store sufficient for the Keycloak example: users, tenants, memberships, and project-like resources.

### Phase 6 — Production example integration

Add an optional Docker Compose Postgres profile or a new production store example that wires Keycloak + Postgres + sessionauth + appauth + audit + capability.

## Testing strategy

Use three levels of validation:

1. Interface tests for memory and SQL stores.
2. Postgres integration tests for transactions and atomicity.
3. End-to-end example smoke with Keycloak and Postgres.

Required commands should include:

```bash
go test ./pkg/gojahttp/auth/sessionauth/... ./pkg/gojahttp/auth/audit/... ./pkg/gojahttp/auth/capability/... ./pkg/gojahttp/auth/appauth/... -count=1
make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

If a new Postgres-backed example is added, it should have its own `make smoke` target.

## Risks and open questions

| Question | Why it matters | Initial direction |
| --- | --- | --- |
| Should stores use `database/sql` directly or a project query abstraction? | Determines dependency surface and test ergonomics. | Start with `database/sql` unless another repo standard exists. |
| Should migrations be embedded SQL files or Go strings? | Affects deployment and versioning. | Prefer embedded SQL migrations with explicit version names. |
| Should appauth resources be generic rows or app-specific adapters? | Real apps have domain-specific resource tables. | Provide a starter store and keep resolver interfaces replaceable. |
| Should audit insert failures block requests? | Current audit is best-effort. | Keep route audit best-effort, but expose store errors for direct callers/tests. |

## References

- `pkg/gojahttp/auth/sessionauth/sessionauth.go`
- `pkg/gojahttp/auth/audit/audit.go`
- `pkg/gojahttp/auth/capability/capability.go`
- `pkg/gojahttp/auth/appauth/appauth.go`
- `examples/xgoja/19-express-keycloak-auth-host`
