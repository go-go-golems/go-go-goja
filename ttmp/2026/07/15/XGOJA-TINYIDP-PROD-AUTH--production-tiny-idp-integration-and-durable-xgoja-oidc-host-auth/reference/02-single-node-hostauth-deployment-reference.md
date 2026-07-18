---
Title: Single-node hostauth deployment reference
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - auth
    - oidc
    - security
    - xgoja
    - deployment
    - database
    - rate-limiter
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/xgoja/hostauth/glazed.go
      Note: Operator-facing configuration flags
    - Path: repo://pkg/xgoja/hostauth/preflight.go
      Note: Single-node production preflight contract
    - Path: repo://pkg/xgoja/hostauth/readiness.go
      Note: Safe machine-readable topology report
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-15T18:46:24.779417468-04:00
WhatFor: ""
WhenToUse: ""
---


# Single-node hostauth deployment reference

## Purpose

This is the operator contract for an xgoja generated application that uses
tiny-idp for browser OIDC login. It describes the only production profile that
`hostauth` currently accepts: one durable serving process. The profile is a
deliberate boundary, not a claim that SQLite or an in-memory rate limiter works
as a multi-replica service.

The contract is activated with `auth.deployment.profile: single-node`. During
configuration resolution the host rejects memory auth stores, runtime schema
application, insecure session cookies, and any non-OIDC mode. OIDC issuer and
browser callback URLs are also HTTPS-only because the insecure-localhost escape
hatch is disabled by the profile.

## Deployment topology

```text
browser -- HTTPS --> reverse proxy -- private HTTP --> one xgoja process
   |                                      |
   |                                      +--> durable SQL database
   |
   +-- HTTPS --> tiny-idp -- TLS --> tiny-idp database/key material
```

The reverse proxy is responsible for the browser-visible certificate and must
preserve the original Host and forwarding headers according to the application
server's proxy policy. `auth.oidc.public-base-url` is the public HTTPS origin,
not the loopback listener address. The registered tiny-idp client must contain
exactly `https://app.example.test/auth/callback` and the intended post-logout
landing URL.

## Required configuration

The following configuration is representative. It intentionally does not
contain a DSN with credentials; inject that through the normal deployment
configuration mechanism rather than committing it to a tutorial file.

```yaml
auth:
  mode: oidc
  deployment:
    profile: single-node
  session:
    cookie:
      allow-insecure-http: false
      same-site: lax
      path: /
  rate-limiter:
    driver: memory
  stores:
    default:
      driver: sqlite                 # or postgres for the same one-process contract
      dsn: /var/lib/example/auth.sqlite
      apply-schema: false
  oidc:
    issuer-url: https://idp.example.test
    client-id: example-app
    public-base-url: https://app.example.test
    after-login-url: /
    after-logout-url: /
```

`apply-schema: false` is mandatory. Apply and verify migrations as a separate
deployment job before starting the process. The application owns any domain
tables and migrations; hostauth owns the schemas for sessions, audit records,
application users, capability data, programmatic credentials, and short-lived
OIDC login transactions.

## Preflight and readiness

Configuration preflight is performed by `hostauth.ResolveConfig`, before
database handles or OIDC discovery are opened. The following error categories
are intentional release blockers:

- `auth.stores.<name>.driver`: a memory store was selected.
- `auth.stores.<name>.apply-schema`: a process attempted to migrate its own
  production database.
- `auth.session.cookie.allow-insecure-http`: a browser cookie could be sent on
  HTTP.
- `auth.oidc.*`: an issuer or callback is missing, malformed, or non-HTTPS.

After successful host construction, `GET /auth/readyz` returns a JSON topology
declaration such as:

```json
{
  "ready": true,
  "mode": "oidc",
  "profile": "single-node",
  "rateLimiter": "memory",
  "stores": [
    {"name":"session","driver":"sqlite"},
    {"name":"audit","driver":"sqlite"}
  ]
}
```

The endpoint deliberately omits DSNs, database credentials, cookies, client
secrets, authorization codes, state, nonce, PKCE verifier, and bearer tokens.
It asserts that configuration was accepted; it is not a substitute for an
application database ping or a reverse-proxy health check.

## Rate-limit limitation

The configured `memory` limiter stores counters inside the xgoja process. It
is correct only when one process serves all requests. Do not place two replicas
behind a load balancer, use rolling overlap, or autoscale this profile: each
process would make an independent allow/deny decision. A future distributed
driver must be implemented and selected explicitly before the host exposes a
multi-replica profile.

## Credentials, keys, audit, and recovery

- Keep tiny-idp signing keys and the generated application's database
  credentials in a deployment secret manager with least-privilege access.
- Register exact redirect and post-logout URLs in tiny-idp. Do not use broad
  wildcard origins.
- Back up the SQL database, test restoration, and include auth schema versions
  in the release runbook.
- Retain audit records according to the product's security and privacy policy.
  Before a retention job is enabled, review its query plan and prove that it
  cannot delete recent security evidence.
- Treat OIDC transaction material and bearer credentials as secrets. Do not
  add them to request logs, readiness output, error pages, or support bundles.

## Release checklist

1. Run migrations outside the serving process and record the applied version.
2. Verify tiny-idp discovery and JWKS over the production TLS path.
3. Verify the registered callback and post-logout redirect exactly match the
   public origin.
4. Start exactly one application process and check `/auth/readyz`.
5. Execute the browser login, logout, callback-replay, device approval, and
   token-revocation smoke suite.
6. Confirm audit retention and backup/restore ownership before declaring the
   deployment production-ready.

## Goal

<!-- What is the purpose of this reference document? -->

## Context

<!-- Provide background context needed to use this reference -->

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
