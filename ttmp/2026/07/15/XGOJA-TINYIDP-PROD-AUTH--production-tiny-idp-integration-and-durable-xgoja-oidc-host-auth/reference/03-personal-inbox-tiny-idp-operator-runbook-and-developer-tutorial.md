---
Title: Personal inbox tiny-idp operator runbook and developer tutorial
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - auth
    - oidc
    - security
    - xgoja
    - examples
    - deployment
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/server.js
      Note: User-scoped JSON domain actions
    - Path: repo://examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/xgoja.yaml
      Note: Reference application OIDC and scope configuration
    - Path: repo://pkg/gojahttp/auth/programauth/device_handlers.go
      Note: Application-owned device refresh and revoke endpoints
    - Path: repo://ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/02-strict-personal-inbox-device-smoke.sh
      Note: Strict fixture orchestration
    - Path: repo://ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/03-personal-inbox-ui.spec.js
      Note: Playwright browser assertion
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-15T19:07:06.654741712-04:00
WhatFor: ""
WhenToUse: ""
---


# Personal inbox tiny-idp operator runbook and developer tutorial

## Purpose and scope

The Personal Knowledge Inbox Step 08 example is the smallest complete xgoja
application in this repository that demonstrates both human browser login and
application-owned programmatic credentials. tiny-idp is the OpenID Connect
issuer for people. xgoja `programauth` is the credential authority for a CLI
that acts on the inbox after a logged-in person approves a device code. These
are intentionally separate authorization layers.

The example has a small static HTML UI, a JSON inbox API, SQLite domain storage,
and explicit actions. It is not a resource server for tiny-idp access tokens.
It accepts `ggat_` access tokens issued by the application itself. Adding native
tiny-idp bearer-token acceptance remains Phase 5 work because that needs a
proper issuer/audience/scope/revocation resource-server contract.

## Architecture

```text
                         browser OIDC
  person  --------------------------------->  tiny-idp
    |                                             |
    |  app session cookie                          | ID token + verified claims
    v                                             v
Personal Inbox xgoja host  <--- callback ---  hostauth OIDC handler
    |
    | session-user actions
    +--> /api/inbox, /api/capture, /api/inbox/:id/archive
    |
    | device code / opaque application tokens
    +--> /auth/device/* ---> programauth ---> /api/programmatic/capture
    |
    +--> application SQLite + hostauth SQL stores + audit sink
```

The local app user is keyed by the stable issuer subject. Hostauth projects a
minimal identity into the session. It does not copy an OIDC access token into
the session and it does not automatically grant application memberships.

## Domain actions and HTTP surface

| Surface | Authority | Action | Notes |
| --- | --- | --- | --- |
| `GET /api/inbox` | browser session | `user.self.read` | Lists only the current app user’s rows. |
| `POST /api/capture` | browser session + CSRF | `user.self.read` | Creates a browser-submitted inbox item. |
| `POST /api/inbox/:id/archive` | browser session + CSRF | `user.self.read` | Archives only an item owned by the session user. |
| `POST /api/programmatic/capture` | `ggat_` agent token | `user.self.read` | Uses the approved device agent’s owner user ID. |
| `POST /auth/device/start` | public | requested action set | Produces device/user codes; it is not a login endpoint. |
| `POST /auth/device/approve` | browser session + CSRF | intersection of requested actions | Binds a pending device request to the session user. |
| `POST /auth/device/token` | device code | approved grants | Consumes an approved device code once and returns a token pair. |
| `POST /auth/device/refresh` | refresh token | inherited grants | Rotates the refresh credential and returns a replacement pair. |
| `POST /auth/device/revoke` | refresh token | N/A | Revokes the refresh-token family without disclosing token existence. |

The `express.agent()` programmatic route receives a local actor. Its
`ownerUserId` claim is created by `programauth` from the browser approver, which
is why Alice’s token cannot write into Bob’s inbox.

## Operator deployment contract

For a real deployment, use the hostauth `single-node` profile documented in
`reference/02-single-node-hostauth-deployment-reference.md`. The public browser
origin must be HTTPS, and tiny-idp must register exactly:

```text
redirect URI:            https://inbox.example.test/auth/callback
post-logout redirect URI: https://inbox.example.test/
```

The running process must use durable stores and externally-applied migrations:

```yaml
auth:
  mode: oidc
  deployment: { profile: single-node }
  session: { cookie: { allow-insecure-http: false } }
  rate-limiter: { driver: memory } # exactly one serving process
  stores:
    default:
      driver: postgres             # SQLite is also valid for one process
      dsn: <secret deployment DSN>
      apply-schema: false
  oidc:
    issuer-url: https://idp.example.test
    client-id: personal-inbox
    public-base-url: https://inbox.example.test
    scopes: [openid, profile, email]
```

Terminate TLS at the reverse proxy and keep the xgoja listener private. Apply
the hostauth and application SQL migrations before the process starts. Verify
`GET /auth/readyz` reports `profile: single-node` and non-memory store drivers.
The endpoint intentionally does not expose a DSN, secret, token, state, nonce,
or PKCE verifier.

## Run the strict integration fixture

The fixture is a disposable integration environment, not a production
deployment. tiny-idp is served with a generated self-signed certificate; the
app itself stays on loopback HTTP because this is a local test. The test does
not use the `single-node` profile because that profile correctly rejects local
HTTP. Production must put the application behind HTTPS and use the profile.

```bash
cd examples/xgoja/23-personal-knowledge-inbox/08-device-authorization
make strict-tinyidp-smoke TINYIDP_ROOT=/absolute/path/to/tiny-idp
```

The command:

1. builds the generated xgoja binary from `xgoja.yaml`;
2. provisions strict tiny-idp with TLS, signing key, audit file, Alice and Bob,
   exact callback and post-logout URIs, and a public PKCE client;
3. starts the app with a disposable domain database and hostauth database;
4. verifies an unauthenticated CLI device poll remains pending;
5. runs the Python protocol smoke for Alice and Bob, including token rotation,
   refresh-family revocation, and cross-user inbox isolation; and
6. runs Playwright against system Chromium to exercise the real login form,
   callback, session UI, and device-approval form.

The fixture scripts are ordered by purpose:

- `scripts/01-strict-tinyidp-fixture.sh` provisions and exports the issuer
  environment.
- `scripts/02-strict-personal-inbox-device-smoke.sh` starts the app and runs
  CLI/protocol/browser validation.
- `scripts/03-personal-inbox-ui.spec.js` is the Playwright UI assertion.
- `scripts/package.json` and `pnpm-lock.yaml` pin the Playwright runner; use
  system Chromium through `PLAYWRIGHT_CHROMIUM_EXECUTABLE` when needed.

## Developer workflow

Build and run the local tutorial with its ordinary Keycloak defaults:

```bash
make smoke
make keycloak-up
./dist/personal-knowledge-inbox-device-authorization serve inbox server \
  --db /tmp/personal-inbox.sqlite
```

For device credentials, use the generated CLI verbs. Treat the opaque token
values as secrets and do not paste them into a committed terminal transcript.

```bash
bin=./dist/personal-knowledge-inbox-device-authorization
$bin verbs inboxctl device-start --base-url http://127.0.0.1:18795
$bin verbs inboxctl device-token --base-url http://127.0.0.1:18795 --device-code 'ggdc_...'
$bin verbs inboxctl device-refresh --base-url http://127.0.0.1:18795 --refresh-token 'ggrt_...'
$bin verbs inboxctl device-revoke --base-url http://127.0.0.1:18795 --refresh-token 'ggrt_...'
$bin verbs inboxctl token-capture --base-url http://127.0.0.1:18795 --access-token 'ggat_...' --title 'CLI capture' --url https://example.test
```

Device start creates a pending request. A browser user logs in and submits the
shown user code to `/auth/device/approve`; only then can the CLI poll. Refresh
is single-use rotation. Reusing a rotated refresh token revokes the family.
Explicit revocation also revokes that refresh family. Existing access tokens
are not made magically invalid; their bounded lifetime is the current contract.

## Review checklist

- Confirm the tiny-idp client is public + PKCE and has no wildcard redirects.
- Confirm the app requests `openid profile email`; `openid` alone does not
  require an issuer to provide the UI’s profile/email claims.
- Confirm the domain rows are scoped by the local session/agent owner ID, never
  by a client-supplied user identifier.
- Confirm browser mutation and device approval routes enforce CSRF.
- Confirm `device-token` is pending before approval, single-use after approval,
  and refresh reuse produces `invalid_grant`.
- Confirm audit/log configuration redacts transaction state, nonce, verifier,
  authorization code, and opaque credentials.
- Confirm a deployment with two processes is rejected operationally until a
  distributed rate limiter has a specified and tested implementation.

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
