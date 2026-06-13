---
Title: Keycloak production hardening and MFA implementation plan
Ticket: XGOJA-AUTH-KEYCLOAK-MFA
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
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Current Keycloak host example to harden
    - Path: pkg/doc/29-express-auth-user-guide.md
      Note: Documents planned auth and .mfaFresh route declarations
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: OIDC login/callback/logout and transaction store seam
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: App session authentication and MFA freshness enforcement
ExternalSources: []
Summary: Plan production hardening for Keycloak/OIDC host auth and concrete MFA freshness flows.
LastUpdated: 2026-06-12T20:29:30.717451977-04:00
WhatFor: Use when hardening keycloakauth/sessionauth for production browser login and implementing host-owned MFA freshness updates.
WhenToUse: Before changing Keycloak handler behavior, OIDC transaction handling, logout, secure cookie deployment, or MFA challenge/session update flows.
---


# Keycloak production hardening and MFA implementation plan

## Executive summary

This ticket tracks the next layer after persistent auth stores: production hardening of the Keycloak/OIDC host path and a concrete MFA flow that updates `sessionauth.Session.MFAAt` so planned routes using `express.user().mfaFresh(...)` can be enforced end-to-end.

The existing code already has the correct architectural boundary. `keycloakauth` verifies Keycloak/OIDC and creates an app session. `sessionauth` authenticates planned routes with an opaque app cookie and now enforces MFA freshness against `Session.MFAAt`. What is still missing is production-grade OIDC transaction storage, deployment hardening, logout behavior, and host-owned MFA challenge/verification endpoints that can refresh `MFAAt`.

## Scope

This ticket covers roadmap items 2 and 3 from the auth follow-up list:

1. Production Keycloak host hardening.
2. MFA story beyond enforcement: challenge, verification, session update, and docs.

It intentionally depends on, but does not replace, the persistent-store ticket. Production Keycloak hardening needs durable session and transaction storage. MFA flows need durable session updates and audit records.

## Current-state evidence

Relevant files:

```text
pkg/gojahttp/auth/keycloakauth/keycloakauth.go
pkg/gojahttp/auth/keycloakauth/README.md
pkg/gojahttp/auth/sessionauth/sessionauth.go
pkg/gojahttp/auth/sessionauth/sessionauth_test.go
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml
examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py
pkg/doc/29-express-auth-user-guide.md
```

Current behavior:

- Keycloak login uses OIDC Authorization Code + PKCE.
- Callback verifies state, code, ID token, and nonce.
- `UserNormalizer` maps Keycloak claims into an app session projection.
- Browser receives an app session cookie, not IdP tokens.
- `sessionauth.Manager.Authenticate` enforces `SecuritySpec.MFAFreshWithin` against `Session.MFAAt`.
- The Keycloak example still uses in-memory sessions and transaction storage.

## Production Keycloak hardening

### OIDC transaction store

`keycloakauth.TransactionStore` currently has an in-memory default. Production deployments need a durable or shared transaction store so multi-instance callbacks work and login state survives process restarts within the short transaction TTL.

Implementation requirements:

- store state, nonce, PKCE verifier, redirect URL, and created timestamp;
- expire transactions after a short TTL;
- `Take` must be one-time;
- state/nonce/verifier must never be logged;
- integration tests should prove replay protection.

### Secure deployment settings

The example uses local HTTP settings. Production docs and code should make the secure path obvious:

- HTTPS-only redirect URLs;
- secure cookies;
- appropriate `SameSite` mode;
- reverse proxy headers and trusted proxy guidance;
- Keycloak issuer URL consistency;
- redirect URI and web origin validation;
- secret handling for confidential clients when used.

### Logout and session lifecycle

The current logout clears/revokes the app session. Production hardening should decide whether and how to support Keycloak end-session behavior.

Open design points:

- local app logout only vs app logout plus Keycloak end-session redirect;
- back-channel logout or front-channel logout support;
- whether refresh tokens are ever retained server-side;
- how logout audit events are recorded.

Initial direction: keep app-session logout mandatory and make Keycloak end-session optional/configured. Do not expose IdP tokens to browser JavaScript.

### Example hardening

The current Docker Compose Keycloak smoke is valuable and should remain fast. A production hardening phase should either extend it with optional Postgres-backed stores or add a second production-profile example.

The smoke should continue to verify:

- anonymous denial;
- Keycloak login;
- app session creation;
- CSRF denial and success;
- resource 404;
- logout;
- post-logout denial.

## MFA flow design

### What exists today

The route plan can declare MFA freshness:

```javascript
app.post("/billing/payment-methods")
  .auth(express.user().required().mfaFresh("10m"))
  .csrf()
  .allow("billing.payment_method.update")
  .handle(handler)
```

The Go route plan carries that as `SecuritySpec.MFAFreshWithin`, and `sessionauth.Manager.Authenticate` rejects sessions whose `MFAAt` is nil or stale.

### What is missing

A host needs a way to set `Session.MFAAt` after the user completes a second factor. That should be host-owned, not Express-owned, because MFA method choice belongs to the application/IdP/deployment.

Possible MFA paths:

1. **Keycloak-required MFA during login.** Keycloak enforces MFA before the initial app session is created. The normalizer or handler marks `MFAAt` when verified claims indicate MFA was completed.
2. **App step-up MFA.** A planned route returns 401 for stale MFA. The browser visits an app-owned MFA challenge endpoint. Successful verification updates `Session.MFAAt`.
3. **Keycloak step-up prompt.** The app redirects to Keycloak with prompt/acr/max_age parameters, then updates `MFAAt` on return if the ID token satisfies the requested assurance.

Initial recommendation: support explicit app-owned MFA update hooks first, then document how to integrate Keycloak-required MFA or step-up flows.

### Proposed sessionauth additions

Add a narrow session update API rather than exposing store internals:

```go
func (m *Manager) MarkMFAComplete(ctx context.Context, r *http.Request, at time.Time) error
```

This should:

- load and validate the current session;
- update `MFAAt` atomically;
- preserve revocation and expiry checks;
- optionally rotate the session ID after MFA completion if desired;
- emit audit events through a host-level caller, not directly from `sessionauth` unless an audit dependency is explicitly added.

This likely requires extending `sessionauth.Store` with an MFA update method or adding a narrower optional interface.

### Error behavior

MFA freshness denial currently maps to `gojahttp.ErrUnauthenticated`, which means planned routes return 401. That is correct if the client must complete additional authentication. A future response body/header may distinguish `mfa_required` from ordinary unauthenticated requests, but it should not leak sensitive details by default.

## Implementation phases

### Phase 1 — Production Keycloak settings and docs

Document secure Keycloak client settings, redirect URI rules, HTTPS/cookie requirements, and local-vs-production differences.

### Phase 2 — Durable OIDC transaction store

Add a SQL-backed `keycloakauth.TransactionStore` with one-time `Take`, TTL cleanup, and replay tests.

### Phase 3 — Logout hardening

Add optional Keycloak end-session support if the design confirms it is needed. Preserve app-session revocation as the mandatory logout behavior.

### Phase 4 — MFA session update primitive

Add a `sessionauth` API and store support for updating `Session.MFAAt`. Include tests for updating, stale route rejection before update, and successful route authentication after update.

### Phase 5 — MFA example flow

Add a small app-owned MFA example endpoint to the dev-auth or Keycloak example. It can use a deliberately simple demo factor, but it must demonstrate the important runtime behavior: route denied before step-up, MFA completion updates the session, route allowed after step-up.

### Phase 6 — Production smoke and docs

Extend the Keycloak smoke or add a dedicated smoke for MFA freshness and production settings.

## Testing strategy

Required tests should include:

```bash
go test ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth -count=1
make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

Additional tests should cover:

- OIDC transaction replay rejection;
- expired OIDC transaction rejection;
- production cookie settings;
- `MFAAt` update persistence;
- planned route denial with stale MFA;
- planned route success after MFA update.

## Body/schema validation and auth boundary

Body/schema validation is security-relevant, but it is not authentication. It belongs to request validation and authorization safety.

It matters for auth when authorization depends on request body fields. Examples:

- a body contains `tenantId`, `role`, `ownerId`, `resourceId`, or `permissions`;
- a route updates membership roles;
- a route creates an invite capability;
- a route performs partial updates where omitted vs null fields have different meaning.

Without schema validation, the authorizer may make decisions using untrusted or ambiguous data, or the handler may mutate fields that were not intended to be client-controlled. The safest long-term route-plan order is:

```text
authenticate actor
resolve path/query-bound resources
validate body schema and normalize typed input
authorize action against actor + resource + normalized input
verify CSRF for unsafe browser/session requests
run handler
record audit outcomes
```

The current auth work can proceed without body schemas because route identity/resource/action enforcement already uses path/query bindings and host-owned resource resolution. Body schemas should be a separate follow-up ticket after persistent stores and Keycloak/MFA hardening, because it is a broader request-validation layer, not the next blocker for authentication.

## References

- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`
- `pkg/gojahttp/auth/sessionauth/sessionauth.go`
- `examples/xgoja/19-express-keycloak-auth-host`
- `pkg/doc/29-express-auth-user-guide.md`
