---
Title: ProgramAuth SQL Production Migration Notes
Ticket: XGOJA-PROGRAMMATIC-AUTH-DESIGN
Status: active
Topics:
    - goja
    - xgoja
    - auth
    - security
    - rest-api
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/gojahttp/auth/programauth/sqlstore/schema.go
      Note: Canonical SQL programauth DDL
    - Path: pkg/gojahttp/auth/programauth/sqlstore/schema.go:Canonical SQLite/PostgreSQL DDL for programauth tables and indexes
    - Path: pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go
      Note: SQL programauth store transaction and transition behavior
    - Path: pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go:SQL store implementation and transaction/transition behavior
    - Path: pkg/xgoja/hostauth/stores.go
      Note: Generated host programauth store construction
    - Path: pkg/xgoja/hostauth/stores.go:Generated host store construction for programauth
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# ProgramAuth SQL Production Migration Notes

## Goal

Provide production operators with a concise reference for deploying SQL-backed programmatic-auth stores in generated xgoja hosts.

## Context

ProgramAuth stores back automation agents, API tokens, access tokens, rotating refresh-token families, and device authorization codes. For multi-process deployments, these records must be durable so revocation, refresh-token reuse detection, and device-code consumption are visible across pods.

The generated hostauth configuration exposes a separate `programauth` store family. It inherits from `auth.stores.default` unless overridden.

## Quick reference

### Configuration

Use the default store when all auth tables live in the same database:

```yaml
auth:
  mode: oidc
  stores:
    default:
      driver: postgres
      dsn: ${AUTH_DATABASE_URL}
      apply-schema: true
```

Use a dedicated programauth DSN when automation credential state is isolated:

```yaml
auth:
  mode: oidc
  stores:
    default:
      driver: postgres
      dsn: ${AUTH_DATABASE_URL}
      apply-schema: false
    programauth:
      driver: postgres
      dsn: ${PROGRAMAUTH_DATABASE_URL}
      apply-schema: false
```

Equivalent flat flags:

```bash
--auth-programauth-store-driver postgres \
--auth-programauth-store-dsn "$PROGRAMAUTH_DATABASE_URL" \
--auth-programauth-store-apply-schema=false
```

### Tables

| Table | Purpose |
| --- | --- |
| `auth_program_agents` | Durable automation principals and policy grants. |
| `auth_program_api_tokens` | Hashes and lifecycle metadata for direct API tokens. |
| `auth_program_access_tokens` | Hashes and expiry metadata for short-lived access tokens. |
| `auth_program_refresh_tokens` | Rotating refresh-token families, generation numbers, use/revocation markers. |
| `auth_program_device_authorizations` | Device/user-code lifecycle, poll metadata, approval/denial/consume state. |

### Required invariants

- Raw token values are only returned at issuance. SQL stores persist hashes plus lookup prefixes.
- `token_prefix` and `device_code_prefix` are lookup accelerators, not authenticators. The full raw secret must hash-match before use.
- Refresh-token rotation is transactional at the refresh-token store boundary: insert replacement and mark current token used in one transaction.
- Refresh-token reuse must revoke the whole family through `RevokeRefreshTokenFamily`.
- Device authorization consumption is a conditional update: only approved, not denied, and not already consumed rows can be consumed.
- Device approval must not broaden grants; service logic intersects requested and approved grant sets.

### Migration practice

For demos and single-app local databases, `apply-schema: true` is acceptable.

For production:

1. Extract the DDL from `pkg/gojahttp/auth/programauth/sqlstore/schema.go`.
2. Apply it through the normal migration tool under an owner role.
3. Run the application with `apply-schema: false`.
4. Grant the runtime role DML rights on the programauth tables and indexes, not broad DDL rights.
5. Verify all generated host instances use the same `programauth` DSN before enabling multiple replicas.

### Cleanup jobs

ProgramAuth stores intentionally keep lifecycle metadata for auditability. Add scheduled cleanup jobs only after deciding retention requirements.

Suggested candidates:

| Data | Conservative cleanup condition |
| --- | --- |
| Expired access tokens | `expires_at < now() - retention_window` |
| Expired refresh-token families | all family rows expired or revoked before retention window |
| Consumed device authorizations | `consumed_at < now() - retention_window` |
| Denied/expired device authorizations | `denied_at` or `expires_at` before retention window |
| Revoked API tokens | `revoked_at < now() - retention_window` |

Do not delete active refresh-family rows piecemeal; family-level reasoning is needed for incident review and reuse detection.

## Validation checklist

Before rollout:

```bash
go test ./pkg/gojahttp/auth/programauth ./pkg/gojahttp/auth/programauth/sqlstore ./pkg/xgoja/hostauth
go test ./...
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

Operational checks:

- Start one generated host with SQL `programauth` stores and `apply-schema: true` in a disposable database.
- Confirm the five `auth_program_*` tables exist.
- Issue an API token and verify only token hashes are stored.
- Start a device flow, approve it, poll once successfully, then confirm a second poll fails as consumed.
- Refresh a token pair and confirm the old refresh token has `used_at` and `replaced_by_id` set.
- Reuse the old refresh token and confirm family rows receive `revoked_at`.

## Usage examples

### Local SQLite smoke

```yaml
auth:
  mode: dev
  stores:
    default:
      driver: memory
    programauth:
      driver: sqlite
      dsn: file:programauth.sqlite?cache=shared
      apply-schema: true
```

### Shared Postgres production shape

```yaml
auth:
  mode: oidc
  stores:
    default:
      driver: postgres
      dsn: ${AUTH_DATABASE_URL}
      apply-schema: false
    programauth:
      driver: postgres
      dsn: ${PROGRAMAUTH_DATABASE_URL}
      apply-schema: false
```

## Related references

- `cmd/xgoja/doc/20-hostauth-config-reference.md`
- `cmd/xgoja/doc/21-auth-stores-reference.md`
- `cmd/xgoja/doc/25-programmatic-auth-javascript-apis.md`
- `cmd/xgoja/doc/28-device-authorization-programmatic-access.md`
- `pkg/gojahttp/auth/programauth/sqlstore/schema.go`
