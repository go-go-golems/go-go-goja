---
Title: Production tiny-idp integration and durable xgoja OIDC host auth design and implementation guide
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - xgoja
    - auth
    - oidc
    - security
    - http
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: Defines OIDC transaction state, browser handlers, and the current memory-only default.
    - Path: repo://pkg/gojahttp/auth/keycloakauth/sqlstore/schema.go
      Note: Defines the transaction table schema and expiry index for external migrations.
    - Path: repo://pkg/gojahttp/auth/keycloakauth/sqlstore/sqlstore.go
      Note: Implements durable atomic OIDC login transaction persistence for SQLite and PostgreSQL.
    - Path: repo://pkg/gojahttp/auth/programauth
      Note: Defines application-owned device and programmatic API credential primitives.
    - Path: repo://pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Defines opaque local session and CSRF behavior after successful OIDC verification.
    - Path: repo://pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go
      Note: Provides the durable store implementation pattern used to design transaction storage.
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Composes host-owned sessions, native OIDC handlers, programmatic auth, and rate limiting.
    - Path: repo://pkg/xgoja/hostauth/config.go
      Note: Defines generated-host auth and persistent-store configuration seams.
    - Path: repo://ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/01-strict-tinyidp-fixture.sh
      Note: Provisions the strict TLS tiny-idp contract fixture used by later browser/CLI smokes.
ExternalSources: []
Summary: Intern-facing design and implementation guide for durable OIDC login transactions, production hostauth configuration, tiny-idp browser login, application-owned device access, and a future native-token resource-server boundary.
LastUpdated: 2026-07-15T15:10:11.112446721-04:00
WhatFor: ""
WhenToUse: ""
---



# Production tiny-idp integration and durable xgoja OIDC host auth design and implementation guide

## Executive Summary

The personal-inbox tutorial proves that a generated xgoja host can use an external OpenID Provider, establish an application session, protect planned Express routes, and issue application credentials through device login. That is a strong foundation for a small application. It is not yet sufficient to call the generated host production-grade because the security transaction linking an OIDC login initiation to its callback defaults to process memory.

This document defines the path from tutorial architecture to a production-ready integration with tiny-idp. The first implementation change is intentionally small: make OIDC authorization transactions durable, expiring, and atomically single-use, then wire them through `hostauth`. The surrounding work makes the operational contract explicit: safe configuration validation, durable rate-limit choices, monitoring, migrations, and an integration suite using strict tiny-idp. A separate later phase defines how an xgoja API could accept tiny-idp-issued device tokens. It is not part of the first example.

An intern should leave this guide understanding four boundaries:

- tiny-idp authenticates the human and provides standard OIDC browser SSO.
- The generated xgoja host is an OIDC relying party and creates its own application session.
- The application authorizes domain actions and resources through planned Express routes and local grants.
- `programauth` issues credentials for the application API; it does not make the application a generic OAuth resource server.

## 1. The system being designed

The target is a generated xgoja application that can be deployed alongside a separately operated production tiny-idp. The application offers a browser UI, a JSON API, and a CLI. A user signs in through tiny-idp. The application then retains an opaque, server-side session that represents its local user. The CLI can obtain a short-lived app API token after the user approves a device flow in the authenticated browser.

The separation has an important consequence. Tiny-idp knows identity, registered clients, OIDC scopes, browser authentication, and its own sessions. The application knows its messages, ownership, domain permissions, device grants for its API, and the local session cookie. Neither system needs to persist the other system’s browser token material in the first release.

```text
                         identity boundary

 Browser ── /auth/login ──> xgoja generated host
    │                              │ creates state, nonce, PKCE verifier
    │                              ▼
    └──────── authorization request ───────> tiny-idp
                                             │ authenticates human
 Browser <──── callback(code, state) ───────┘
    │                              │ consumes one OIDC transaction
    │                              │ verifies ID token
    ▼                              ▼
 application session cookie ──> planned Express routes ──> application data

 CLI ── device start/poll ───────> xgoja programauth ──> application API
 Browser ── device approve ──────> xgoja programauth
```

### 1.1 What this design is not

This is not a request to embed tiny-idp into every generated application. Running tiny-idp separately gives it an independent issuer URL, client-registration lifecycle, signing-key lifecycle, audit trail, and deployment boundary. The application is a relying party, not an IdP implementation.

This is also not a request to treat an OIDC ID token as a bearer token for the application API. An ID token proves facts to its client at a particular authentication event; it is not a general API authorization credential. The first example uses application-issued programmatic tokens for application API access.

## 2. Current implementation map

The current code provides the right primitives but leaves one critical dependency implicit.

| Responsibility | Current location | Current behavior | Production assessment |
|---|---|---|---|
| OIDC login/callback/logout | `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` | Discovery, authorization code, PKCE S256, nonce, ID-token verification, end-session redirect | Protocol mechanics are sound; transaction storage defaults to memory. |
| OIDC configuration | `pkg/xgoja/hostauth/config.go` | Issuer URL, client, public/callback URL, scopes, local login/logout paths | Generic in behavior; needs transaction-store and explicit production policy settings. |
| Host composition | `pkg/xgoja/hostauth/builder.go` | Builds session, audit, app auth, capability, programauth services and mounts native handlers | Does not supply a durable OIDC transaction store; uses memory rate limits. |
| Application sessions | `pkg/gojahttp/auth/sessionauth` and `sqlstore` | Opaque cookie points to server-side session containing local user/CSRF/expiry/claims | Suitable when a durable store is configured. |
| Planned route authorization | `pkg/gojahttp` and Express integration | Authentication, principal/CSRF, resource resolution, grants, authorizer, audit before JS handler | Good place for application policy. |
| App device credentials | `pkg/gojahttp/auth/programauth` | Device code, user approval, API and refresh tokens, grants, SQL stores | Good first device-auth path; needs lifecycle observability and atomic-family review. |

### 2.1 The current OIDC transaction

The transaction is explicitly small:

```go
type Transaction struct {
    State        string
    Nonce        string
    PKCEVerifier string
    CreatedAt    time.Time
    RedirectURL  string
}

type TransactionStore interface {
    Put(ctx context.Context, tx Transaction) error
    Take(ctx context.Context, state string) (Transaction, error)
}
```

`handleLogin` creates random `state`, `nonce`, and PKCE verifier values. It validates `return_to` as a local path, stores the transaction, and redirects to the issuer using the state, nonce, and S256 challenge. `handleCallback` requires state and code, calls `Take`, exchanges the code using the verifier, verifies the ID token, checks the nonce, normalizes the user, and only then creates a local application session.

This answers a common misunderstanding: the transaction store is not the session store. It exists before the user has a local session and is destroyed before session creation. Persisting sessions in SQL does not preserve a login that was initiated before a process restart.

## 3. Threat model and required invariants

The design does not claim to prevent every OAuth threat. It defines the properties that this host must preserve and the tests that establish them.

| Invariant | Failure prevented | Enforcement point |
|---|---|---|
| Callback state is unguessable and bound to a server-side transaction. | Login CSRF and callback substitution. | Generate cryptographically random state; `Take(state)`. |
| A callback transaction is consumed exactly once. | Callback and authorization-code replay through the host. | Atomic `TakeAndDeleteUnexpired`. |
| The code exchange uses the original PKCE verifier. | Intercepted authorization-code redemption. | Store verifier; provide it only to token exchange. |
| Returned ID token nonce equals stored nonce. | ID-token replay/substitution. | Verify after signature/audience verification. |
| Post-login redirect is local and validated before issuer redirect. | Open redirect after sign-in. | Reject absolute and `//` paths. |
| Local browser session is opaque and server-side. | Token exposure and client-side session tampering. | Session manager/store and secure cookie. |
| API credential permissions are application actions plus resource policy. | Scope-only overauthorization and cross-user access. | Planned route grant and resolver stages. |
| Secret protocol values never enter logs or audit payloads. | Credential leakage during debugging. | Redaction rules and test assertions. |

The transaction must survive exactly the failure modes that occur between initiating login and receiving the callback: a host restart, a rolling deployment, a replica change, and an operator-driven retry. It must not survive indefinitely. Its expiry is part of its security contract.

## 4. Proposed architecture

### 4.1 Layered authorization model

The system has two authorization layers with different semantics.

| Layer | Authority | Example question | Artifact |
|---|---|---|---|
| Identity and browser SSO | tiny-idp | “Has this person authenticated for this OIDC client?” | OIDC authorization response and verified ID token. |
| Application access | xgoja host and application | “May this user or CLI token create a message in this inbox?” | Local session or app credential plus domain grant/resource check. |

This is not duplicated authorization. The first decision establishes an authenticated human identity for the application. The second controls actions against application-owned resources. The application does not need tiny-idp to model every message, board, durable object, or ownership relation.

### 4.2 Data owned by each store

| Store | Required contents | Lifetime | Never store here |
|---|---|---|---|
| OIDC transaction store | state, nonce, PKCE verifier, safe local redirect, created/expiry times | About 10 minutes; one-use | Browser cookie, authorization code, raw token response, app data. |
| App session store | random session ID, local user ID, CSRF secret, expiry, selected claims | Idle/absolute session lifetime | tiny-idp refresh token unless a deliberate token-management feature is designed. |
| App-auth/programauth store | local users, actors, grants, device codes, hashed API/refresh credentials, revocation/lifecycle fields | According to local credential policy | OIDC transaction secrets. |
| tiny-idp data store | users, clients, IdP browser sessions, OAuth grants/tokens, issuer keys/audit | IdP policy | Application messages or app authorization policy. |

### 4.3 Supported deployment profiles

The configuration must state which topology it supports rather than allowing a deployment to infer safety from defaults.

| Profile | Transaction/session stores | Rate limiter | Supported use |
|---|---|---|---|
| `development` | Memory allowed | Memory allowed | Local tutorial and tests. |
| `single-node` | Durable SQLite or PostgreSQL | Memory allowed only when documented as one replica | One durable app process with backup and controlled restart behavior. |
| `production-ha` | Shared PostgreSQL or equivalent | Shared/distributed limiter | Multiple replicas and rolling deployments. |

Production configuration should reject a memory transaction store. HA configuration must also reject a local-only store and a process-local rate limiter. A preflight report should make the selected profile and detected limitations visible to operators without printing DSNs or secrets.

## 5. Durable transaction-store design

### 5.1 Interface and behavior

The existing interface is sufficient in shape, but its contract must be strengthened. `Take` means atomically claim and delete one transaction only when it exists and has not expired. It must not return the transaction twice when two callbacks race. It must remove expired rows as part of retrieval or make their cleanup behavior equivalent.

```go
// Put rejects incomplete transactions and records an explicit expiration.
Put(ctx, tx) error

// Take consumes exactly one transaction. It never returns a transaction
// that was expired at the database's comparison time.
Take(ctx, state) (Transaction, error)
```

The public result should distinguish only operationally useful categories. The browser receives a generic invalid-state failure. Logs and metrics may record `not_found`, `expired`, `already_consumed`, or `storage_error` as redacted categories. They must not record the state value, nonce, verifier, authorization code, or token.

### 5.2 Schema

One table is enough. Encryption at rest is provided by the database/storage layer; the verifier should still be treated as a credential secret in application access controls and diagnostics.

```sql
CREATE TABLE oidc_login_transactions (
    state_hash        TEXT PRIMARY KEY,
    nonce             TEXT NOT NULL,
    pkce_verifier     TEXT NOT NULL,
    redirect_path     TEXT NOT NULL,
    created_at        TIMESTAMP NOT NULL,
    expires_at        TIMESTAMP NOT NULL
);

CREATE INDEX oidc_login_transactions_expires_at_idx
    ON oidc_login_transactions (expires_at);
```

Storing a keyed hash of state rather than raw state is preferred. The browser still carries the raw state in the OAuth redirect. The host hashes it before query. This limits the value of a database read compromise and means application diagnostics cannot accidentally display a live callback key. The HMAC key must be stable across replicas and rotations need an explicit overlap policy. If that key-management work is not ready, raw state with strict redaction is an acceptable first implementation; document the tradeoff rather than implying that a hash is a complete encryption mechanism.

### 5.3 Atomic consumption algorithms

PostgreSQL can use one `DELETE ... WHERE ... RETURNING` statement:

```sql
DELETE FROM oidc_login_transactions
WHERE state_hash = $1
  AND expires_at > CURRENT_TIMESTAMP
RETURNING nonce, pkce_verifier, redirect_path, created_at, expires_at;
```

The delete is the claim. Exactly one concurrent callback can receive a row. A second callback receives no row. Expired rows do not return.

SQLite versions that support `RETURNING` can use the same construction. If a supported SQLite version cannot, the implementation must use `BEGIN IMMEDIATE`, select only an unexpired record, delete by primary key in that transaction, and commit before returning. Do not implement `SELECT` followed by an unrelated `DELETE` outside a transaction.

```text
Take(state):
  key = HashOrNormalize(state)
  begin transaction when required by driver
  row = delete unexpired transaction where state_hash = key returning fields
  if no row:
      return ErrTransactionUnavailable
  commit
  return Transaction(row)
```

### 5.4 Expiry and cleanup

Request-path correctness cannot depend on a background cleanup job. Retrieval always checks `expires_at`. A periodic cleanup deletes old rows to control table growth:

```sql
DELETE FROM oidc_login_transactions
WHERE expires_at <= CURRENT_TIMESTAMP;
```

The job must be safe for every replica to run, or a single maintenance owner must be explicit. Metrics should record cleanup count and current approximate pending-transaction count, never their values.

## 6. Hostauth integration design

`hostauth.Config` already resolves store configuration for session, audit, app auth, capability, and programauth services. Add a distinct `OIDCTransaction` store configuration rather than silently reusing a vaguely named store. It makes data ownership and production validation visible.

```yaml
auth:
  mode: oidc
  profile: production-ha
  oidc:
    issuer-url: https://id.example.net
    client-id: personal-inbox
    public-base-url: https://inbox.example.net
    scopes: [openid, profile, email]
    after-login-url: /
    after-logout-url: /
  stores:
    default:
      driver: postgres
      dsn: ${INJECTED_BY_DEPLOYMENT}
      apply-schema: false
    oidc-transaction:
      driver: postgres
      dsn: ${INJECTED_BY_DEPLOYMENT}
      apply-schema: false
```

The shown secret placeholder is illustrative. Generated application configuration must not invent an environment-variable convention without the host’s configuration system explicitly supporting it. The operator-facing guide should describe the selected secret-injection mechanism for the resulting binary.

The builder should construct a durable transaction store and pass it into OIDC handler configuration. Development mode may deliberately select `NewMemoryTransactionStore` when the profile allows it. OIDC mode must never silently select memory in a declared production profile.

```text
BuildHostAuthServices(config):
  stores = BuildStores(config.stores)
  ValidateProfile(config.profile, stores, cookie, publicURL, limiter)
  sessions = BuildSessionManager(stores.session)
  transactions = BuildOIDCTransactionStore(stores.oidcTransaction)
  oidc = NewOIDCHandlers(
      issuer=config.oidc.issuerURL,
      sessionManager=sessions,
      transactionStore=transactions,
  )
  return nativeHandlers, authenticators, auditSink, closers
```

### 6.1 Naming cleanup

The current package is named `keycloakauth`, while its mechanics use standard discovery and OpenID Connect interfaces. Its default user normalizer also writes a `keycloakSub` claim. This is misleading for tiny-idp users.

The desired end state is `pkg/gojahttp/auth/oidcauth` and an `oidcSub` claim. This change should be made as a clear, intentional API migration. Do not add a permanent duplicate package or configuration alias merely to hide the rename. Before changing it, determine whether the package is part of the public library contract and announce the migration in release notes.

## 7. Tiny-idp integration contract

The application is registered in tiny-idp as an OIDC client. The required callback and logout destinations are application URLs, not tiny-idp URLs.

| Item | Example | Constraint |
|---|---|---|
| Issuer | `https://id.example.net` | Must match discovery and ID-token issuer. |
| Callback URI | `https://inbox.example.net/auth/callback` | Exact registered redirect URI. |
| Post-logout URI | `https://inbox.example.net/` | Exact registered post-logout redirect URI. |
| Browser scopes | `openid profile email` | Start minimally; app permissions remain local. |
| Client type | Public with PKCE, or confidential where server secret handling is deliberate | The host always uses PKCE. |

The callback code is redeemed by the host. It verifies the returned ID token against tiny-idp discovery/JWKS and client audience. It normalizes the verified subject to a local user. The local session does not retain the OAuth token response because the browser app does not need it for its own API authorization.

Logout is two operations. The host revokes its local session and clears its cookie. It can then redirect to tiny-idp’s discovered end-session endpoint with its client identifier and registered post-logout URL. This order ensures the application remains logged out even when the IdP logout redirect fails or is unavailable.

## 8. Device login and application API design

The first example should use application-owned device authorization because its credentials represent access to application resources. The CLI begins the flow at the xgoja host, receives a verification URI and user code, and polls the host. The browser user signs in via tiny-idp if necessary, opens the approval UI, sees the requested app actions, and approves or denies. The host then issues an app token according to local `programauth` policy.

```text
CLI                         xgoja host                    Browser / tiny-idp
 | POST /auth/device/start       |                                  |
 |<-- device_code, user_code ----|                                  |
 | poll /auth/device/token ----->|                                  |
 |                                |<---- user opens verification ----|
 |                                |      browser login through IdP   |
 |                                |<---- approve with session + CSRF-|
 |<-- short-lived API token ------|                                  |
 | Authorization: Bearer ggat_... |                                  |
 |-------------------------------> planned route → domain action     |
```

The approval UI should be application-owned but use host-provided APIs and a consistent component contract. It needs to show user code, requesting client/agent label, requested local actions, expiry, approve, deny, success, and error states. Native Go handlers should own state transitions and token issuance; JavaScript renders the experience and calls guarded endpoints.

### 8.1 Why not use tiny-idp device authorization for the first app API?

Tiny-idp’s device grant produces an IdP-issued OAuth credential. Accepting it at an app API requires a resource-server contract. The application needs to validate issuer, target audience, authorized scopes, expiry, revocation behavior, client binding, and DPoP proof when used. xgoja’s current bearer authenticator recognizes its own API-token families, not generic issuer tokens.

This is a worthwhile future feature, but silently treating tiny-idp tokens as equivalent to `programauth` tokens would weaken the model and make revocation semantics unclear. Keep the flows separate until the resource-server adapter is specified and tested.

## 9. Production operations

### 9.1 Configuration preflight

At startup, the host should produce a validation result and fail closed for declared production profiles.

```text
if profile is production or production-ha:
  require HTTPS public base URL
  require Secure, HttpOnly session cookie
  require durable OIDC transaction store
  require durable session and programauth stores
  require externally managed schema migrations
  reject allow-insecure-http

if profile is production-ha:
  require shared transaction/session/programauth stores
  require distributed limiter or explicit compatible limiter
```

The exact `SameSite` policy depends on application topology. Cross-site OIDC navigation is a top-level browser redirect, so `Lax` is commonly workable. A deployment that needs `None` must enforce `Secure` and test its browser behavior. The preflight should validate configuration consistency; it should not claim to discover the correctness of every reverse-proxy rule.

### 9.2 Auditing and metrics

Record event names and safe metadata, not protocol secrets:

| Event | Safe fields | Never include |
|---|---|---|
| `oidc.login.started` | issuer ID, client ID, profile | state, nonce, verifier, return URL query data. |
| `oidc.callback.accepted` | issuer ID, local user ID, latency | code, ID token, access token, raw claims. |
| `oidc.callback.rejected` | redacted reason category | state, nonce, code, token. |
| `programauth.device.approved` | local user, actor, requested actions | device code, user code, bearer tokens. |
| `programauth.refresh.rotated` | token-family ID/opaque audit ID | old/new refresh tokens. |

Metrics should expose counts and latency by result category: started, accepted, expired, invalid state, nonce mismatch, exchange failure, and storage failure. Alert on sharp changes in rejected callbacks or device polling errors. Do not attach raw URL query strings as metric labels.

### 9.3 Migrations, backup, and recovery

Database migrations should run as an explicit deploy step, not on every production process startup. The OIDC transaction table is ephemeral, so losing it during a restore causes users with in-progress login to retry, which is acceptable. Losing session, grant, and token-revocation data has different semantics and must be addressed in the application’s backup and incident policy.

## 10. Test and verification plan

The test plan is part of the design because protocol code can look correct while preserving the wrong failure behavior.

### 10.1 Unit and storage tests

- Verify `Put` rejects missing state, nonce, verifier, and invalid redirect path.
- Verify `Take` returns a transaction exactly once.
- Run two concurrent `Take` calls for the same state; exactly one succeeds.
- Verify expired entries cannot be consumed even when cleanup has not run.
- Verify SQLite and PostgreSQL implementations have the same observable behavior.
- Verify no logs or error strings contain state, nonce, verifier, code, ID token, or bearer token.

### 10.2 OIDC integration tests

```text
1. Start strict tiny-idp with a registered application client.
2. Start generated xgoja host with durable transaction/session stores.
3. Request /auth/login; record callback state only in test-internal capture.
4. Complete tiny-idp sign-in and assert a local session is created.
5. Replay the identical callback and expect rejection.
6. Begin login, restart or route callback to a second host replica, and assert success.
7. Begin login, wait beyond TTL, and assert safe rejection.
8. Attempt an external return_to URL and assert the local fallback is used.
9. Log out and assert local session removal plus valid IdP end-session redirect.
```

### 10.3 Browser and CLI tests

Use Playwright for the UI and command-line smokes for the CLI:

- Browser sign-in follows the tiny-idp UI and returns to the application.
- Browser access to an API mutation requires the host CSRF token.
- A user can only see and modify resources they own.
- Device approval shows the requested actions and a denial is terminal.
- Polling respects pending, slow-down, approved, expired, and denied states.
- A CLI token from user A cannot create content as user B.
- Refresh rotation rejects old refresh credentials after successful rotation.

### 10.4 Test harness layout

Keep all ticket-specific experiments and launch scripts in this ticket’s `scripts/` directory. The production implementation should then promote reusable harnesses to the repository’s normal test packages.

```text
ttmp/.../XGOJA-TINYIDP-PROD-AUTH/
  scripts/
    01-run-strict-tinyidp-fixture.sh
    02-oidc-restart-smoke.sh
    03-device-flow-smoke.sh
  playbooks/
    deployment-and-release-verification.md
```

No scripts are required merely to author this design. Add them when Phase 0 begins so they reflect the real fixture and do not become stale pseudo-automation.

## 11. Implementation phases and acceptance criteria

### Phase 0: Contracts and test boundary

This phase establishes what the implementation is promising. It ends when deployment profiles, migration ownership, the nomenclature decision, and the strict tiny-idp fixture are documented and the test matrix exists.

**Acceptance criteria:** reviewers can name the expected storage/limiter topology for each profile and run a failing test that demonstrates a restart during login is currently unsupported.

### Phase 1: Durable transaction store

This phase implements storage and wires it into the host. It ends only after concurrency, expiry, and multi-process/restart semantics are tested against both supported SQL engines.

**Acceptance criteria:** a callback begun before host A stops can complete on host B using the same shared store, while a duplicate callback is rejected.

### Phase 2: Production profile and operations

This phase makes invalid production configuration impossible to ignore. It includes profile validation, safe startup diagnostics, rate-limit topology, migration documentation, audit/metric contracts, and readiness behavior.

**Acceptance criteria:** a declared production HA configuration using in-memory transaction state or a local-only database fails at startup with an actionable non-secret diagnostic.

### Phase 3: Reference application

This phase builds the polished little application: browser UI, API, local domain policy, CLI, application-owned device approval, and strict tiny-idp deployment configuration.

**Acceptance criteria:** Playwright proves browser login/logout and user isolation; CLI smoke proves device approval and a second user’s independent post; no app route needs raw IdP tokens.

### Phase 4: Credential lifecycle and observability

This phase hardens token-family persistence and operational evidence. It converts lifecycle rules into metrics, audits, and negative tests that an operator can use during an incident.

**Acceptance criteria:** refresh/revocation outcomes are atomic at their declared storage boundary, observable without secrets, and covered by failure injection.

### Phase 5: Native tiny-idp device-token resource server

This phase is optional and separate. It starts only after tiny-idp and xgoja agree on a resource-server validation contract.

**Acceptance criteria:** a tiny-idp-issued device token is validated with issuer/audience/scope/revocation semantics and optional DPoP proof requirements before it becomes an application actor. No ID token is accepted as an API bearer credential.

## 12. Alternatives considered

| Alternative | Decision | Reason |
|---|---|---|
| Keep the in-memory transaction store and rely on sticky sessions. | Reject for production. | Restart resilience still fails; routing configuration is not a substitute for durable callback state. |
| Place state/verifier in an encrypted browser cookie. | Do not choose as the default. | It changes key rotation, cookie-size, replay, and logout semantics; server-side single-use storage remains simpler to audit. |
| Reuse the app session store without a dedicated transaction concept. | Reject. | Transaction and session lifetime, contents, and security invariants differ. |
| Issue tiny-idp access tokens directly for the first app API. | Defer. | A resource-server validation and audience contract is not yet implemented in xgoja. |
| Make every application permission an OIDC scope. | Reject. | Domain resource ownership and local action policy belong to the application. |
| Continue Keycloak-specific names because the mechanics work. | Replace deliberately. | Generic OIDC vocabulary makes tiny-idp and other issuer integrations understandable and reduces false coupling. |

## 13. API and file reference

| File or API | Read this first | Why it matters |
|---|---|---|
| `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` | `Transaction`, `TransactionStore`, `New`, `handleLogin`, `handleCallback` | Defines the callback-security protocol and current memory default. |
| `pkg/gojahttp/auth/sessionauth/sessionauth.go` | `Session`, `NewSession`, CSRF validation | Defines the post-verification local browser session. |
| `pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go` | schema and CRUD methods | Template for durable host-owned store patterns. |
| `pkg/xgoja/hostauth/config.go` | `OIDCConfig`, `StoresConfig`, config resolution | Where production profile and transaction-store configuration belong. |
| `pkg/xgoja/hostauth/builder.go` | `BuildHostAuthServices`, native handler construction | Where stores and authenticators are assembled. |
| `pkg/gojahttp/auth/programauth` | device/token services and SQL stores | Defines application-owned programmatic credentials. |
| `examples/xgoja/personal-inbox` | Step 06–08 code and smoke tests | Concrete tutorial proof of browser OIDC and app device login. |
| tiny-idp `internal/fositeadapter/provider.go` | registered endpoints | Identifies OIDC authorization, token, UserInfo, device, end-session, health, and readiness surfaces. |

## 14. Intern walkthrough

Start by tracing one browser login in code. Read `handleLogin` until you understand why state, nonce, and PKCE verifier must exist before redirect. Then read `handleCallback` and identify each check that happens before `NewSession`. Next read the session SQL store and compare its fields and expiry rules with the transaction fields. They must remain separate.

After that, read the host builder. Ask one question: “where does the OIDC handler get its transaction store?” The current answer is that it does not; `New` selects memory. That is the exact implementation seam for Phase 1.

Finally, follow the personal-inbox device flow. Notice that its CLI credential is issued by application `programauth`, after a browser user is authenticated through OIDC. This is a designed two-layer system. Do not replace it with raw tiny-idp token acceptance until the resource-server work in Phase 5 exists.

## Key points

- OIDC transaction storage contains short-lived state, nonce, PKCE verifier, time, and safe local redirect data; it is neither an app session nor an OAuth token cache.
- The transaction must be durable across the supported deployment topology and atomically consumed exactly once.
- Tiny-idp is the identity authority; xgoja is the application session and authorization authority.
- Application-owned device credentials are the correct first API path because they authorize application resources directly.
- Native tiny-idp device tokens require a real resource-server adapter and must remain a separately tested feature.
- Production readiness is a configuration and operational contract, not merely a change from memory to SQL.

## References

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html), especially authentication request state and nonce requirements.
- [RFC 6749: OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749).
- [RFC 7636: Proof Key for Code Exchange by OAuth Public Clients](https://www.rfc-editor.org/rfc/rfc7636).
- [RFC 8252: OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252).
- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/rfc/rfc8628).
- [go-go-goja PR #95](https://github.com/go-go-golems/go-go-goja/pull/95), the personal-inbox auth and device-login implementation under review.
- [Personal Inbox project report](../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/07/13/PROJECT%20REPORT%20-%20go-go-goja%20-%20Personal%20Inbox%20Auth,%20Programmatic%20Access,%20and%20Device%20Login.md), local technical narrative and evidence trail.

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
