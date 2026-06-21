---
Title: SQL-backed programauth stores design
Ticket: XGOJA-PROGRAMMATIC-AUTH-DESIGN
Status: active
Topics:
    - goja
    - xgoja
    - auth
    - security
    - rest-api
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/gojahttp/auth/programauth/agent.go
      Note: |-
        AgentStore contract that SQL stores must implement.
        AgentStore contract for SQL store design
    - Path: pkg/gojahttp/auth/programauth/device.go
      Note: |-
        Device authorization store contract and approval/consume transitions.
        DeviceAuthorizationStore transition contract
    - Path: pkg/gojahttp/auth/programauth/oauth_token.go
      Note: |-
        Access/refresh token store contracts and refresh rotation requirements.
        Access/refresh token family contracts and rotation requirements
    - Path: pkg/gojahttp/auth/programauth/token.go
      Note: |-
        APITokenStore contract and hash/prefix lookup semantics.
        APITokenStore contract and prefix/hash auth design
    - Path: pkg/xgoja/hostauth/stores.go
      Note: |-
        Existing generated-host SQL store construction pattern for app/session/audit/capability stores.
        Existing generated hostauth SQL store construction pattern
ExternalSources: []
Summary: SQL schema and transaction plan for durable programauth stores.
LastUpdated: 2026-06-21T18:39:06.871727447-04:00
WhatFor: Guide implementation of production-ready programauth SQL stores.
WhenToUse: Use before implementing or reviewing SQL-backed agents, API tokens, access/refresh token families, or device authorizations.
---


# SQL-backed programauth stores design

## Executive Summary

Programauth currently has concurrency-safe in-memory stores for agents, API tokens, access tokens, refresh tokens, and device authorizations. Those stores are sufficient for tests and generated local demos, but they are not production stores. They do not survive process restarts, cannot coordinate multiple server processes, and cannot provide database-level atomicity for refresh-token rotation or device-code consumption.

This document defines the SQL-backed store plan. The implementation should add `pkg/gojahttp/auth/programauth/sqlstore` with SQLite and PostgreSQL dialect support, matching the existing `appauth/sqlstore`, `sessionauth/sqlstore`, `audit/sqlstore`, and `capability/sqlstore` package shape. The first version should implement the existing interfaces without changing service-level APIs:

- `programauth.AgentStore`
- `programauth.APITokenStore`
- `programauth.AccessTokenStore`
- `programauth.RefreshTokenStore`
- `programauth.DeviceAuthorizationStore`

The critical correctness requirements are transactional refresh-token rotation and atomic device-code transitions. Refresh rotation must mark the current refresh token used and insert the replacement refresh token while holding a database transaction. Device approval, denial, polling, and consumption must use conditional updates so a code cannot be approved twice, denied after approval, or consumed twice.

## Problem Statement

Programmatic auth now has durable security semantics but non-durable default storage. The memory stores clone returned records and protect local concurrency with mutexes, but they lose all data on restart and cannot coordinate multiple generated host instances. That is acceptable for demos. It is not acceptable for production API tokens, token families, or device-login state.

The highest-risk operation is refresh-token rotation. A refresh token is supposed to be single-use. If two callers present the same refresh token concurrently, exactly one caller may rotate it. The other caller must detect reuse and revoke the family. A SQL implementation that performs a loose `SELECT`, then a service-level check, then an `UPDATE`, then an `INSERT` can race unless the update is conditional or the row is locked in a transaction.

The second high-risk operation is device-code consumption. A device code may receive tokens only once after approval. The durable store must prevent double consumption even when two poll requests arrive at the same time.

## Proposed Solution

Add one SQL store package for all programauth store contracts:

```text
pkg/gojahttp/auth/programauth/sqlstore/schema.go
pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go
pkg/gojahttp/auth/programauth/sqlstore/sqlstore_test.go
```

The package should follow the existing auth SQL store pattern:

```go
type Dialect string

const (
    DialectSQLite   Dialect = "sqlite"
    DialectPostgres Dialect = "postgres"
)

type Config struct {
    DB      *sql.DB
    Dialect Dialect
}

type Store struct {
    db      *sql.DB
    dialect Dialect
}

func New(cfg Config) (*Store, error)
func (s *Store) Schema() string
func (s *Store) ApplySchema(ctx context.Context) error
```

`Store` should implement all five programauth store interfaces. A single package-level store keeps schema migrations, grant JSON encoding, token hash encoding, placeholder helpers, and transaction helpers in one place.

## Schema

Use `auth_program_*` table names to avoid colliding with existing `auth_app_*`, `auth_sessions`, `auth_audit_records`, and `auth_capabilities` tables.

### `auth_program_agents`

```sql
CREATE TABLE IF NOT EXISTS auth_program_agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    owner_user_id TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    disabled_at TIMESTAMP NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    policy_json TEXT NOT NULL DEFAULT '[]'
);
```

Indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_owner ON auth_program_agents(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_tenant ON auth_program_agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_disabled_at ON auth_program_agents(disabled_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_agents_created_at ON auth_program_agents(created_at, id);
```

`policy_json` stores the normalized `gojahttp.GrantSet.Grants` array.

### `auth_program_api_tokens`

```sql
CREATE TABLE IF NOT EXISTS auth_program_api_tokens (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    subject_user_id TEXT NOT NULL DEFAULT '',
    token_hash BLOB NOT NULL,
    token_prefix TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    grants_json TEXT NOT NULL DEFAULT '[]'
);
```

Indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_prefix ON auth_program_api_tokens(token_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_agent ON auth_program_api_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_subject ON auth_program_api_tokens(subject_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_revoked_at ON auth_program_api_tokens(revoked_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_api_tokens_created_at ON auth_program_api_tokens(created_at, id);
```

`token_hash` must store only the hash bytes. Raw token values are never stored.

### `auth_program_access_tokens`

```sql
CREATE TABLE IF NOT EXISTS auth_program_access_tokens (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    subject_user_id TEXT NOT NULL DEFAULT '',
    family_id TEXT NOT NULL,
    token_hash BLOB NOT NULL,
    token_prefix TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    grants_json TEXT NOT NULL DEFAULT '[]'
);
```

Indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_auth_program_access_tokens_prefix ON auth_program_access_tokens(token_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_program_access_tokens_family ON auth_program_access_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_access_tokens_agent ON auth_program_access_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_auth_program_access_tokens_expires_at ON auth_program_access_tokens(expires_at);
```

### `auth_program_refresh_tokens`

```sql
CREATE TABLE IF NOT EXISTS auth_program_refresh_tokens (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    subject_user_id TEXT NOT NULL DEFAULT '',
    family_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    token_hash BLOB NOT NULL,
    token_prefix TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    replaced_by_id TEXT NOT NULL DEFAULT '',
    grants_json TEXT NOT NULL DEFAULT '[]'
);
```

Indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_auth_program_refresh_tokens_prefix ON auth_program_refresh_tokens(token_prefix);
CREATE INDEX IF NOT EXISTS idx_auth_program_refresh_tokens_family ON auth_program_refresh_tokens(family_id, generation);
CREATE INDEX IF NOT EXISTS idx_auth_program_refresh_tokens_used_at ON auth_program_refresh_tokens(used_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_refresh_tokens_revoked_at ON auth_program_refresh_tokens(revoked_at);
```

### `auth_program_device_authorizations`

```sql
CREATE TABLE IF NOT EXISTS auth_program_device_authorizations (
    id TEXT PRIMARY KEY,
    client_name TEXT NOT NULL,
    device_code_hash BLOB NOT NULL,
    device_code_prefix TEXT NOT NULL,
    user_code_hash BLOB NOT NULL,
    user_code TEXT NOT NULL,
    verification_uri TEXT NOT NULL DEFAULT '',
    verification_uri_complete TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    poll_interval_seconds INTEGER NOT NULL,
    last_polled_at TIMESTAMP NULL,
    approved_at TIMESTAMP NULL,
    denied_at TIMESTAMP NULL,
    consumed_at TIMESTAMP NULL,
    agent_id TEXT NOT NULL DEFAULT '',
    subject_user_id TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    grants_json TEXT NOT NULL DEFAULT '[]'
);
```

Indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_auth_program_devices_device_prefix ON auth_program_device_authorizations(device_code_prefix);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_program_devices_user_code_hash ON auth_program_device_authorizations(user_code_hash);
CREATE INDEX IF NOT EXISTS idx_auth_program_devices_expires_at ON auth_program_device_authorizations(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_program_devices_status ON auth_program_device_authorizations(approved_at, denied_at, consumed_at);
```

## Transaction contracts

### Refresh rotation

`RefreshTokenStore.RotateRefreshToken` must be atomic. The store receives the current refresh token id, the fully prepared replacement refresh token, and the `usedAt` timestamp.

Required behavior:

```text
begin transaction
load current refresh token
if missing: return ErrRefreshTokenNotFound
if revoked: return ErrRefreshTokenRevoked
if used: return ErrRefreshTokenUsed
insert replacement refresh token with same family id
mark current used_at, replaced_by_id, updated_at
commit
return current and replacement clones
```

For PostgreSQL, prefer `SELECT ... FOR UPDATE` on the current row. For SQLite, `BEGIN IMMEDIATE` semantics may be needed if `database/sql` transaction behavior is not sufficient under concurrent writes. The first implementation can rely on SQLite's serialized writer behavior in tests, but the transaction boundary should still be explicit.

The conditional update should preserve the single-use invariant:

```sql
UPDATE auth_program_refresh_tokens
SET used_at = $1,
    replaced_by_id = $2,
    updated_at = $1
WHERE id = $3
  AND used_at IS NULL
  AND revoked_at IS NULL;
```

If zero rows are affected after loading an apparently usable token, return `ErrRefreshTokenUsed` or `ErrRefreshTokenRevoked` based on a second read. Service-level code already revokes the family when `ErrRefreshTokenUsed` is returned.

### Refresh family revocation

`RevokeRefreshTokenFamily` should update every non-revoked refresh token in the family:

```sql
UPDATE auth_program_refresh_tokens
SET revoked_at = $1,
    updated_at = $1
WHERE family_id = $2
  AND revoked_at IS NULL;
```

Access-token revocation for the same family is not currently part of the `RefreshTokenStore` interface. The current policy relies on short access-token TTLs. A future combined token-family store can add cross-table family revocation if the product requires immediate access-token invalidation.

### Device approval

`ApproveDeviceAuthorization` should be a conditional update. It should not approve expired, denied, consumed, or already-approved codes.

```sql
UPDATE auth_program_device_authorizations
SET approved_at = $1,
    updated_at = $1,
    agent_id = $2,
    subject_user_id = $3,
    tenant_id = $4,
    grants_json = $5
WHERE id = $6
  AND approved_at IS NULL
  AND denied_at IS NULL
  AND consumed_at IS NULL;
```

The service already checks expiry and terminal states before calling the store. The store should still protect state transitions in case of concurrent requests.

### Device denial

`DenyDeviceAuthorization` should only affect pending, unconsumed codes:

```sql
UPDATE auth_program_device_authorizations
SET denied_at = $1,
    updated_at = $1
WHERE id = $2
  AND approved_at IS NULL
  AND denied_at IS NULL
  AND consumed_at IS NULL;
```

### Device consumption

`ConsumeDeviceAuthorization` must be single-use:

```sql
UPDATE auth_program_device_authorizations
SET consumed_at = $1,
    updated_at = $1
WHERE id = $2
  AND approved_at IS NOT NULL
  AND denied_at IS NULL
  AND consumed_at IS NULL;
```

If zero rows are affected, reload the device and return the closest sentinel: `ErrDeviceNotFound`, `ErrDeviceDenied`, `ErrDeviceConsumed`, or a generic not-approved error. The service maps consumed, denied, and expired states to OAuth-style token endpoint errors.

## JSON encoding rules

Grant sets should be stored as JSON arrays of `gojahttp.Grant` values. Store code should normalize before writing and after reading.

```json
[
  {"Action":"report.read","TenantID":"o1","ResourceType":"","ResourceID":""}
]
```

The SQL store should provide helpers:

```go
func marshalGrantSet(grants gojahttp.GrantSet) (string, error)
func unmarshalGrantSet(raw string) (gojahttp.GrantSet, error)
```

`unmarshalGrantSet` should accept empty strings as an empty grant set only if older rows exist; newly written rows should use `[]`.

## Design Decisions

### Decision 1: One `programauth/sqlstore.Store` implements all store interfaces

Status: accepted.

A single store package avoids five separate SQL packages that would duplicate dialect helpers, grant JSON helpers, hash scanning helpers, and transaction utilities. The service layer already composes distinct interfaces, so one concrete type can satisfy them all while callers still depend on interface contracts.

### Decision 2: Keep service APIs unchanged

Status: accepted.

The first SQL implementation should not redesign `AgentService`, `APITokenService`, `OAuthTokenService`, or `DeviceService`. The immediate goal is durable storage parity with memory stores. Service API changes can happen later if token-family-wide access-token revocation or combined transactions become mandatory.

### Decision 3: Store raw hashes as binary columns

Status: accepted.

`TokenHash`, `DeviceCodeHash`, and `UserCodeHash` are byte slices in Go. SQLite can store them as `BLOB`; PostgreSQL should use `BYTEA`. This avoids unnecessary hex encoding and keeps constant-time comparisons on the Go side unchanged.

### Decision 4: Prefix lookup remains the first query filter

Status: accepted.

The stores should continue returning candidate rows by token prefix. Authentication still hashes the raw bearer token and compares candidate hashes in Go with `subtle.ConstantTimeCompare`. The prefix narrows the candidate set but is not the authority.

### Decision 5: Generated hostauth wiring comes after store parity tests

Status: accepted.

The SQL store package should first satisfy the programauth contracts and pass store-level tests. Generated-host config can then select memory or SQL programauth stores using the existing store configuration pattern.

## Alternatives Considered

### Alternative: Keep programauth memory-only and document external token services

Rejected. The current system already owns agent identity, route grants, bearer authentication, and device flow. Outsourcing persistence while keeping in-process enforcement would split critical state across systems and make local generated-host deployments harder to reason about.

### Alternative: Encode grants as space-separated scope strings only

Rejected. The internal model is `GrantSet`, not OAuth scope strings. Scope strings are useful for wire/debug views, but SQL should preserve typed dimensions so future policy migrations do not need to parse strings.

### Alternative: Add one SQL package per store type

Rejected for the first implementation. Separate packages may be useful later if dependencies diverge, but the initial store contracts share enough helpers and schema ownership that one package is simpler and easier to test.

## Implementation Plan

1. Add `programauth/sqlstore` with dialect, config, schema, `ApplySchema`, placeholders, scan helpers, grant JSON helpers, and time/null helpers.
2. Implement `AgentStore` and `APITokenStore` first because they are simpler and mirror existing memory-store behavior.
3. Implement `AccessTokenStore` and `RefreshTokenStore`, with transaction-backed `RotateRefreshToken` and family revocation tests.
4. Implement `DeviceAuthorizationStore`, including conditional update semantics for poll, approval, denial, and consumption.
5. Add SQLite tests for all contracts using in-memory SQLite.
6. Add PostgreSQL SQL generation tests if no test Postgres is available in CI.
7. Extend generated hostauth store construction so programauth stores can use the same default SQL DB as other auth stores.
8. Update production docs and help pages with migration notes and operational cleanup guidance.

## Open Questions

- Should refresh-token family revocation eventually revoke outstanding access tokens in the same family, or should short access-token TTLs remain the only immediate mitigation?
- Should `DeviceAuthorizationStore.ApproveDeviceAuthorization` return a sentinel for already-approved codes distinct from consumed or denied states?
- Should generated hostauth support separate DSNs for programauth stores, or should the first implementation reuse the top-level/default auth store DSN?
- Should token hash columns be indexed? Prefix is the intended lookup index; hash indexes are unnecessary for current code and could expose metadata value without improving normal auth paths.
- Should expired device-code cleanup be part of the store package or an explicit host maintenance job?

## References

- `pkg/gojahttp/auth/programauth/agent.go`
- `pkg/gojahttp/auth/programauth/token.go`
- `pkg/gojahttp/auth/programauth/oauth_token.go`
- `pkg/gojahttp/auth/programauth/device.go`
- `pkg/gojahttp/auth/programauth/memory_store.go`
- `pkg/gojahttp/auth/programauth/memory_token_store.go`
- `pkg/gojahttp/auth/programauth/memory_oauth_token_store.go`
- `pkg/gojahttp/auth/programauth/memory_device_store.go`
- `pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go`
- `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go`
