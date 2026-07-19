---
Title: Security operations, retention, and redaction guide
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - security
    - audit
    - oidc
    - auth
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/audit/audit.go
      Note: |-
        Audit normalization and recursive secret-key redaction.
        Recursive protocol-secret redaction
    - Path: repo://pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: |-
        OIDC lifecycle audit and metric emission.
        OIDC lifecycle observations
    - Path: repo://pkg/gojahttp/auth/programauth/device_handlers.go
      Note: |-
        Device, refresh, and revocation lifecycle emission.
        Device lifecycle observations
    - Path: repo://pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go
      Note: |-
        Atomic OAuth token-pair persistence.
        Atomic pair persistence
    - Path: repo://pkg/gojahttp/ratelimit.go
      Note: |-
        Rate-limit decision emission.
        Rate-limit decision observations
ExternalSources: []
Summary: Operating contract for safe authentication telemetry, bounded audit retention, incident triage, and credential redaction in a generated xgoja host.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Security operations, retention, and redaction guide

## Scope

This guide describes the operational boundary added in Phase 4. It covers the
application-owned OIDC relying-party flow, application-owned device credentials,
and planned-route rate limiting. It does not claim that application credentials
are tiny-idp access tokens. That resource-server design remains Phase 5 work.

The core rule is simple: security telemetry must describe a decision without
recording the bearer material, one-time protocol value, or cookie that made the
decision possible. A useful audit record says *which class of decision occurred*
and *why it was accepted or rejected*. It never says `state=...`, `code=...`,
`Authorization: Bearer ...`, a PKCE verifier, an ID token, a refresh token, or
a session identifier.

## Event and metric contract

`gojahttp.SecurityEventObserver` is the intentionally small metrics boundary.
Each event has exactly `name`, `outcome`, and `reason`. Those fields must remain
bounded enumerations; do not put a user ID, route parameter, user code, token
prefix, exception string, callback URL, or remote address in them. The default
`MemorySecurityMetrics` is a testable counter. A production host can implement
the observer with its metrics backend without importing that backend into
`gojahttp`. Supply the observer through `hostauth.BuilderOptions.SecurityEvents`;
the built `hostauth.Services` retains the same observer for diagnostics.

Structured audit uses `gojahttp.AuditSink` and the `audit.Record` schema. The
same lifecycle observation reaches the audit sink, but request metadata is
normalized by `audit.Normalizer`: IP addresses are hashed, while ordinary route
audits retain only their existing safe actor/resource fields.

| Event | Outcomes and reasons | Emitted by |
| --- | --- | --- |
| `oidc.login` | `issued`; `failed` generation/store reasons; `rejected` method | `keycloakauth.Handlers` |
| `oidc.callback` | `accepted`; `rejected` provider error, transaction unavailable, nonce mismatch, token/claim verification reasons | `keycloakauth.Handlers` |
| `oidc.logout` | `accepted`, `rejected/method` | `keycloakauth.Handlers` |
| `programauth.device.start` | issued, rejected, failed | device handlers |
| `programauth.device.poll` | issued; rejected/pending, slow-down, expired, denied, consumed, invalid-code | device handlers |
| `programauth.device.approve` | accepted; rejected CSRF, scope, session, or approval failure | device handlers |
| `programauth.refresh` | rotated; rejected grant/request failures | device handlers |
| `programauth.refresh_revoke` | accepted; rejected/failed | device handlers |
| `auth.rate_limit` | allowed, denied, error; reason is the declared policy name | enforcer |

This produces a reviewable event sequence:

```text
browser -> oidc.login/issued -> IdP
IdP -> oidc.callback/accepted -> local session
CLI -> device.start/issued -> device.poll/pending
browser -> device.approve/accepted -> device.poll/issued
CLI -> refresh/rotated -> refresh reuse/rejected
request -> rate_limit/{allowed|denied}
browser -> oidc.logout/accepted
```

## Token persistence invariant

Access and refresh tokens are separate tables because their lifetimes and
authentication roles differ. A successful issuance or rotation nevertheless
must publish them as one logical pair. `OAuthTokenPairStore` is an optional
native capability: hostauth supplies it only when its access and refresh stores
are the same SQL `programauth/sqlstore.Store`.

```pseudocode
transaction:
    verify current refresh is neither used nor revoked
    insert next access token hash
    insert next refresh token hash
    mark current refresh used with replacement ID
commit
```

If any insert or conditional update fails, the transaction rolls back. The
legacy split-store fallback retains compensating cleanup because no common
transaction exists there. It is correct for tests and memory stores, but it is
not the production shared-SQL path. `TestSQLStoreOAuthTokenPairRollbackLeavesNoAccessToken`
proves that a duplicate refresh insert cannot leak an unreturned access token.

## Retention policy

Set the organization’s approved retention period before deploying. A practical
starting point is 90 days of searchable security audit records, a restricted
longer-term aggregate of security counters, and immediate preservation only for
an active incident or legal requirement. The exact interval is a governance
decision, not a framework default.

Run deletion as a scheduled, separately authorized database operation. Do not
delete rows in request handlers. The current SQL schema is
`auth_audit_records`, indexed by `created_at`, `event`, and `outcome`.

```sql
-- Preview the deletion boundary before executing it.
SELECT count(*)
FROM auth_audit_records
WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days';

-- PostgreSQL retention job after approved backup/incident checks.
DELETE FROM auth_audit_records
WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
```

For SQLite, calculate the UTC cutoff in the scheduler and bind it as a
parameter. Batch large deletes and checkpoint/vacuum according to the database
operator’s runbook. Retention does not apply to live sessions, OIDC transaction
rows, or credential rows; each has its own expiry/revocation lifecycle.

## Incident triage queries

Use aggregate, bounded queries first. Export individual audit records only to
the incident workspace with the same access controls as authentication logs.

```sql
-- Authentication failures by class in the last hour.
SELECT event, outcome, reason, count(*) AS n
FROM auth_audit_records
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '1 hour'
  AND event IN ('oidc.callback', 'programauth.device.poll', 'programauth.refresh')
GROUP BY event, outcome, reason
ORDER BY n DESC;

-- Rate-limit pressure by declared policy.
SELECT reason AS policy, outcome, count(*) AS n
FROM auth_audit_records
WHERE event = 'auth.rate_limit'
  AND created_at >= CURRENT_TIMESTAMP - INTERVAL '15 minutes'
GROUP BY reason, outcome
ORDER BY n DESC;

-- A bounded recent timeline for one known local actor.
SELECT event, outcome, reason, created_at
FROM auth_audit_records
WHERE actor_id = :actor_id
ORDER BY created_at DESC
LIMIT 100;
```

An increase in `oidc.callback/rejected/transaction_unavailable` can be callback
replay, expiry, a browser retry, lost storage, or an incorrect callback target.
It is not evidence to inspect or recover raw `state`. An increase in
`programauth.refresh/rejected/invalid_grant` should be correlated with the
family-level reuse behavior, deployment changes, and client version—not with
captured refresh tokens.

## Redaction verification

`audit.RedactMap` recursively replaces values whose keys contain protocol or
credential fragments: token, secret, password, cookie, session, authorization,
credential, code, state, nonce, verifier, or proof. This is deliberately
conservative. A value is retained only under a safe key.

Tests provide the enforcement evidence:

- `TestNormalizeRecordAndRedaction` serializes an audit record containing a
  session ID, bearer authorization, capability, state, nonce, and PKCE verifier
  and asserts none of the secret values survives.
- `TestDeviceTokenHandlerDoesNotRevealCodeEnumerationDetails` submits malformed
  and unknown device codes, expects `invalid_grant`, and asserts the supplied
  value is absent from the response.
- Token SQL tests use only hashes and prefixes in persistence. Raw values are
  returned once to the caller and are never put into audit attributes.

When adding a new handler, review every `AuditEvent` and `SecurityEvent` with
this checklist:

- Is every label an enum or a declared route policy?
- Does the event avoid raw request bodies, query strings, and headers?
- Does the error reason classify the condition rather than include `err.Error()`
  from an untrusted provider or storage driver?
- Are tests asserting both the public error shape and the absence of the input
  credential in responses/log records?

## Review and handoff

Start code review with `oauth_token.go` and the SQL pair-store methods, then
follow hostauth’s capability wiring. Review `keycloakauth.go`, device handlers,
and `ratelimit.go` as a single decision-observation surface. Finally run:

```sh
go test ./pkg/gojahttp/auth/programauth/... ./pkg/gojahttp/auth/audit/... ./pkg/gojahttp ./pkg/xgoja/hostauth/...
go test ./... -count=1
```

The expected result is atomic pair coverage, lifecycle telemetry coverage,
safe-redaction coverage, and the existing account-isolation/grant-narrowing,
callback-replay, and refresh-reuse tests remaining green.
