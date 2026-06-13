---
Title: Production auth deployment docs and policy adapter plan
Ticket: XGOJA-AUTH-PROD-DOCS
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
    - documentation
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/README.md
      Note: xgoja example index and numbering process target
    - Path: pkg/doc/29-express-auth-user-guide.md
      Note: Main planned Express auth guide that production docs should extend
    - Path: pkg/doc/31-express-auth-examples.md
      Note: Example and smoke documentation to link from deployment guide
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: Default app-owned authorizer model that policy adapters should not replace by default
ExternalSources: []
Summary: Plan production deployment documentation, optional policy adapter exploration, and process cleanup for gojahttp planned auth.
LastUpdated: 2026-06-12T20:52:00-04:00
WhatFor: Use when turning the planned Express auth implementation into deployable operator documentation and when evaluating future policy-engine adapters.
WhenToUse: After persistent stores and Keycloak/MFA hardening are underway or complete, especially before writing production deployment docs.
---


# Production auth deployment docs and policy adapter plan

## Executive summary

This ticket captures the remaining auth follow-up items after persistent stores and Keycloak/MFA hardening: production deployment documentation, optional policy-engine adapter planning, and docs/process cleanup around examples and generated TypeScript provider artifacts.

The intent is not to add a policy engine immediately. The current app-owned `appauth` model should remain the default. Policy adapters become useful only after the durable app auth data model, session persistence, audit persistence, and Keycloak production path are stable enough to provide meaningful inputs to Casbin, OpenFGA, OPA, or similar tools.

## Scope

1. Write a production deployment guide for planned Express auth hosts.
2. Document secure operational defaults for Keycloak, app sessions, CSRF, audit, persistent stores, TLS, reverse proxies, and cookies.
3. Capture an example-numbering and branch-merge checklist for `examples/xgoja`.
4. Document TypeScript provider API regeneration expectations.
5. Evaluate optional policy-engine adapters after the app-owned authorization model is proven with persistent stores.

## Non-goals

- Do not replace `appauth.Authorizer` with a policy engine in this ticket.
- Do not implement persistent stores here; that belongs to `XGOJA-AUTH-STORES`.
- Do not implement OIDC transaction or MFA session update primitives here; that belongs to `XGOJA-AUTH-KEYCLOAK-MFA`.
- Do not reintroduce Express-owned user/session storage.

## Proposed production guide outline

```text
1. Architecture overview
   - Express route plans declare intent
   - Go host owns identity, session, resources, authorization, CSRF, audit
   - Keycloak authenticates; app authorizes

2. Deployment topology
   - browser -> reverse proxy/TLS -> Go host -> Keycloak
   - Go host -> Postgres/session/audit/appauth/capability stores

3. Keycloak configuration
   - realm/client settings
   - redirect URI/web origin rules
   - Authorization Code + PKCE
   - confidential/public client guidance
   - MFA/required actions

4. App session settings
   - opaque cookie
   - Secure, HttpOnly, SameSite
   - idle and absolute expiry
   - revocation and rotation

5. CSRF
   - unsafe methods require `.csrf()`
   - header/token behavior
   - reverse-proxy and same-site caveats

6. Authorization model
   - app-owned users, tenants, memberships, resources
   - deny-by-default authorizer
   - route action strings
   - body/schema validation caveat

7. Audit
   - audit event lifecycle
   - redaction guarantees
   - operational queries

8. Persistent stores and migrations
   - sessions
   - audit
   - capabilities
   - appauth domain data

9. Production checklist
   - TLS
   - proxy headers
   - cookie security
   - Keycloak issuer consistency
   - backups
   - secret rotation
   - logs and audit retention

10. Troubleshooting
   - redirect mismatch
   - state/nonce failures
   - stale MFA
   - CSRF failures
   - raw route rejection
```

## Policy adapter evaluation plan

Evaluate adapters only after stores exist. The evaluation should answer:

| Adapter | Useful for | Questions |
| --- | --- | --- |
| Casbin | In-process RBAC/ABAC with Go integration | Can route actions + appauth resources map cleanly without duplicating DB state? |
| OpenFGA | Relationship-based authorization | Does tenant/resource membership data fit tuple modeling well enough to justify an external service? |
| OPA | Declarative policy over structured input | Is the operational complexity acceptable for planned route authz? |

Initial recommendation: keep `appauth.Authorizer` as the boring default and add adapters as optional `gojahttp.Authorizer` implementations, not as framework requirements.

## Example and docs process cleanup

The xgoja/v2 merge showed that example numbering can collide. Add a short process note covering:

- reserve example number ranges when a branch adds examples;
- check `examples/xgoja/README.md` before and after merge;
- regenerate TypeScript provider declarations when provider APIs change;
- re-run example smokes after renumbering;
- keep generated-binary examples distinct from host integration examples.

## Dependencies

- `XGOJA-AUTH-STORES` for durable sessions, audit, capabilities, and appauth stores.
- `XGOJA-AUTH-KEYCLOAK-MFA` for production Keycloak hardening, OIDC transaction storage, and MFA flows.
- PR #74 for planned Express auth baseline.

## Acceptance criteria

- A Glazed help page or docs page exists for production planned-auth deployment.
- The production guide includes checklist-style operator guidance.
- Example numbering and TypeScript regeneration process notes exist.
- Policy adapters are documented as future optional work with clear prerequisites.
- Docs link to the main Express auth guide and the Keycloak/auth host examples.
