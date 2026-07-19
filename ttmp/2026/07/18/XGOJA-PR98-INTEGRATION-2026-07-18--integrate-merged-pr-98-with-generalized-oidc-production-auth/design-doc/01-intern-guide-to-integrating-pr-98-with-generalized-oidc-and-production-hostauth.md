---
Title: Intern guide to integrating PR 98 with generalized OIDC and production hostauth
Ticket: XGOJA-PR98-INTEGRATION-2026-07-18
Status: active
Topics:
    - auth
    - oidc
    - security
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://modules/express/auth_builders.go
      Note: Exposes fluent authentication declarations to JavaScript
    - Path: repo://pkg/gojahttp/auth/appauth/appauth.go
      Note: Defines local users external identities memberships and authorization
    - Path: repo://pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Defines generalized browser OIDC flow injected transport and logout
    - Path: repo://pkg/gojahttp/auth_plan.go
      Note: Defines typed route authentication and redacted auth results
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Composes the production authentication service graph
ExternalSources: []
Summary: Architecture, decisions, APIs, merge procedure, and validation plan for combining PR 98 production hostauth with the generalized OIDC branch.
LastUpdated: 2026-07-18T20:00:00-04:00
WhatFor: Teaching a new implementer how browser OIDC, application identity, device credentials, route enforcement, persistence, and production host wiring fit together.
WhenToUse: Before reviewing or changing the PR 98 integration, OIDC persistence, hostauth composition, logout, or external identity behavior.
---


# Intern guide to integrating PR 98 with generalized OIDC and production hostauth

## Executive summary

This integration combines two independently correct but differently shaped authentication efforts. The current feature branch generalized a Keycloak-specific browser-login package into `oidcauth`, made an OIDC identity issuer-scoped, supported an injected HTTP client for an identity provider running inside the same process, propagated the authenticated actor into trusted Go services, and made logout a CSRF-protected POST. The newly merged PR 98 adds the production mechanisms that surround that core: durable login transactions, device credential lifecycle operations, external OAuth bearer verification, trusted-proxy request identity, configurable rate limiting, audit and security events, readiness checks, maintenance, and operator-facing configuration.

The integration must not choose one parent wholesale. It must preserve the general OIDC security boundary while porting PR 98's production features out of the old `keycloakauth` namespace. The local application user remains distinct from an external identity. External identities are keyed by the complete `(issuer, subject)` pair. Browser logout remains `POST /auth/logout` with session CSRF. PR 98's health, device, OAuth, observability, and durable-state features are composed around those decisions.

The resulting system supports three request families:

- A browser signs in through OIDC and receives an application session cookie.
- A coding agent uses the device flow to obtain locally issued access and refresh tokens.
- A caller presents an externally issued Tiny-IDP OAuth bearer token that is verified against an exact issuer, resource, and scope contract.

JavaScript declares route requirements, but Go owns credential parsing, verification, authorization, CSRF decisions, rate limiting, audit, and the redacted authentication context.

## 1. The system being integrated

### 1.1 go-go-goja and xgoja

`go-go-goja` is a general-purpose Go runtime and module ecosystem around Goja. The `xgoja` command can assemble JavaScript verbs, native modules, generated runtime code, and an HTTP server. Authentication is not a JavaScript library that application code is trusted to implement. It is a host-owned service graph built in Go and presented to the Express-like JavaScript API through typed route builders.

The important boundary is:

```text
JavaScript application                     Go host
----------------------                     -------
declares route pattern       ---------->    validates route plan
declares auth requirements   ---------->    authenticates credential
implements business handler  <----------    supplies redacted actor/auth context
uses actor-bound services     ---------->    Go reads actor from owner context
```

The route plan types live in `pkg/gojahttp/auth_plan.go`. The host composition lives in `pkg/xgoja/hostauth/`. The JavaScript builder surface lives in `modules/express/auth_builders.go`.

### 1.2 Tiny-IDP and OIDC

Tiny-IDP is the external identity provider for browser users and, in the resource-server profile, an OAuth token issuer for API clients. OIDC supplies authenticated identity claims. OAuth supplies bearer-token authorization. They overlap in protocol machinery but answer different questions:

| Protocol use | Question answered | Result in this application |
|---|---|---|
| Browser OIDC authorization code flow | Which human signed in? | A local application session. |
| Local device authorization | Which coding agent did this user authorize? | A local agent plus access/refresh token family. |
| Tiny-IDP OAuth bearer introspection | Is this issuer token active and valid for this resource and scope? | A verified OAuth actor and redacted OAuth context. |

An intern should not treat the ID token, access token, application session, and local user record as interchangeable. Each has a separate lifetime and trust boundary.

### 1.3 Hostauth

`pkg/xgoja/hostauth/builder.go` constructs the concrete authentication system for one generated-host execution. On PR 98's `origin/main`, `BuildHostAuthServices` opens stores, creates a session manager, builds token and device services, creates a rate limiter, builds OAuth verifiers, mounts native handlers, and returns a `Services` bundle. This is dependency composition: every downstream component must receive the same stores, clock, rate limiter, audit sink, and security observer.

```text
Resolved configuration
        |
        v
   BuildStores -----------------------------+
        |                                    |
        +-> session store -> session manager |
        +-> appauth stores -> user resolver  |
        +-> programauth stores -> services   |
        +-> OIDC transaction store           |
        +-> audit store -> audit sink         |
                                             v
security observer + rate limiter + request identity
        |                                    |
        +----------------> AuthOptions <------+----> planned routes
        |
        +----------------> native handlers -------> /auth/*, /healthz
```

PR 98's builder is evidence for this graph at `origin/main:pkg/xgoja/hostauth/builder.go:59-151`. Its native endpoint inventory is at lines 156-217.

## 2. The authentication path

### 2.1 Browser login

Browser login begins at `GET /auth/login`. The OIDC handler creates three unpredictable values:

- `state` binds the callback to the login attempt.
- `nonce` binds the ID token to the authorization request.
- `PKCEVerifier` proves that the process redeeming the authorization code initiated the request.

PR 98 persists these values in a `TransactionStore` instead of holding them only in process memory. The interface appears at `origin/main:pkg/gojahttp/auth/keycloakauth/keycloakauth.go:73-99`. The generalized package must keep that interface under `oidcauth`.

Pseudocode:

```text
handleLogin(request):
    require request.method == GET
    state    = randomToken()
    nonce    = randomToken()
    verifier = randomToken()
    returnTo = validateLocalReturnPath(request.query.return_to)

    transactionStore.Put({state, nonce, verifier, now, returnTo})
    redirect to issuer authorization endpoint with:
        state
        nonce
        code_challenge = S256(verifier)
```

The callback atomically consumes the transaction, exchanges the code through the injected OIDC HTTP client, verifies the ID token and nonce, normalizes the identity into a local user, and creates a session.

```text
browser       xgoja host        Tiny-IDP       appauth/session stores
   | GET login    |                 |                    |
   |------------->| store tx        |                    |
   |<-------------| 302 authorize   |                    |
   |------------------------------->|                    |
   |<-------------------------------| callback code      |
   | callback     |                 |                    |
   |------------->| take tx         |                    |
   |              | exchange code ->|                    |
   |              |<- tokens -------|                    |
   |              | verify identity |                    |
   |              |----------------------> upsert/bind   |
   |              |----------------------> create session|
   |<-------------| session cookie + 302                 |
```

### 2.2 Application identity

OIDC defines `sub` as locally unique within an issuer. A bare subject is not globally unique. The durable lookup key is therefore:

```text
external identity key = canonical issuer URL + NUL + subject
```

The local user ID belongs to the application. It is the ID used in memberships, resource ownership, sessions, agents, and audit records. An external identity record binds `(issuer, subject)` to that user.

The integrated storage contract should be:

```go
type UserStore interface {
    ByID(ctx context.Context, id string) (*User, error)
    ByExternalIdentity(ctx context.Context, issuer, subject string) (*User, error)
    BindExternalIdentity(ctx context.Context, userID, issuer, subject string) error
    UpsertFromOIDC(ctx context.Context, issuer, subject, email string, verified bool) (*User, error)
    DisableUser(ctx context.Context, id string, disabledAt time.Time) error
}
```

There should be no `ByKeycloakSub` API. Keeping it would preserve an invalid assumption and create two identity paths. The user requested no backwards-compatibility adapters; callers and tests should be migrated directly.

The first-login operation must be atomic in SQL:

```text
upsertFromOIDC(issuer, subject, profile):
    begin transaction
    existing = select user joined through external_identity
    if existing is disabled: return not found
    if existing exists: update mutable profile fields; commit; return

    userID = opaqueStableOIDCUserID(issuer, subject)
    insert local user if absent
    insert external identity(issuer, subject, userID)
    commit
```

PR 98's separate identity table enables future account linking. The current branch's issuer-scoped user fields and deterministic ID make first-login creation safe. The implementation should combine the properties, not retain duplicate legacy columns as an alternate lookup path.

### 2.3 Sessions and logout

A browser session is application state, not an IDP token. The session contains a local user ID, projected tenant membership, selected non-secret claims, expiry metadata, and a CSRF token. A protected browser mutation uses the session cookie plus CSRF proof.

Logout mutates server-side state and must remain:

```http
POST /auth/logout
X-CSRF-Token: <session csrf token>
```

The operation is:

```text
verify CSRF
revoke request session
if revocation failed: return 500 and do not pretend success
clear cookie
record audit/security outcome
return 204
```

PR 98 registers `GET /auth/logout` at `origin/main:pkg/xgoja/hostauth/builder.go:210-215`. The integration removes it. Provider single sign-out is a separate future flow; it must not be smuggled into a state-changing GET handler.

### 2.4 Device authorization and local agent credentials

The device flow allows a CLI or coding agent without a browser session to ask a signed-in user for approval. PR 98 adds the complete lifecycle:

```text
agent                  host                    signed-in browser
  | POST device/start    |                           |
  |--------------------->| create device request     |
  |<---------------------| user_code + device_code   |
  |                      |<---- inspect request ------|
  |                      |<---- approve or deny ------|
  | POST device/token    |                           |
  |--------------------->| consume approved request   |
  |<---------------------| access + refresh token     |
  | POST refresh         |                           |
  |--------------------->| rotate token family        |
  |<---------------------| new access + refresh       |
```

The host owns allowed actions, maximum requested actions, verification URI, endpoint rate limits, enabled-user checks, refresh-family revocation, and agent disablement. JavaScript business code never handles raw device codes or refresh-token hashes.

### 2.5 Planned routes and Express syntax

Express builders produce a typed `RoutePlan`; they do not execute authentication. A route may require a session user, a local agent token, or an exact external OAuth contract.

Conceptual JavaScript:

```javascript
app.post("/api/messages",
  auth.oauth()
    .issuer("https://id.example.test")
    .resource("message-api")
    .scopes("messages.write"),
  auth.allow("bbs.post.create"),
  auth.audit("bbs.message.create"),
  handler)
```

The Go plan contains `OAuthRequirement{Issuer, Resource, Scopes}`. PR 98 defines this at `origin/main:pkg/gojahttp/auth_plan.go:81-92`. Enforcement must compare those exact values against verifier-confirmed data, not unverified JWT claims supplied by JavaScript.

After enforcement, the JavaScript handler receives a copied, redacted context:

```javascript
ctx.actor.id
ctx.auth.method
ctx.auth.principalKind
ctx.auth.oauth.issuer
ctx.auth.oauth.resources
ctx.auth.oauth.scopes
```

It never receives the bearer token, token hash, refresh token, device code, or introspection credentials. The actor is also installed in the Go owner context through `ContextWithActor`, allowing native application services to bind reads and writes to the authenticated principal without trusting a JavaScript-supplied ID.

## 3. Merge analysis

The common ancestor is `6a1a095`. The current branch adds six feature commits plus its previous merge diary. Updated `origin/main` ends at merge commit `b5f41a1`, which contains PR 98 and follow-up review remediation.

The pre-merge synthetic analysis identified five concentrated textual conflicts:

| File | Conflict meaning |
|---|---|
| `pkg/gojahttp/auth/appauth/appauth.go` | Issuer-scoped identity versus Keycloak-subject plus external bindings. |
| `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go` | Competing user schemas, queries, and upsert behavior. |
| `pkg/gojahttp/auth/oidcauth/oidcauth.go` | Rename conflict plus durable transactions/observability versus injected transport/strict logout. |
| `pkg/xgoja/hostauth/builder.go` | Production service graph versus injected OIDC transport and strict routes. |
| `pkg/xgoja/hostauth/builder_test.go` | Endpoint inventory and package/API expectations. |

Several files auto-merge but still require semantic review: `auth_plan.go`, `planned_dispatch.go`, appauth tests/schema, and hostauth store tests. An auto-merge means Git found non-overlapping lines; it does not prove the combined security invariant.

## 4. Design decisions

### Decision: Use provider-neutral OIDC naming

- **Context:** PR 98 developed durable OIDC mechanisms under `keycloakauth`; the feature branch already generalized the package.
- **Options considered:** Restore `keycloakauth`, maintain both packages, or port PR 98 into `oidcauth`.
- **Decision:** Keep one `oidcauth` package and move/port transaction persistence and observability into it.
- **Rationale:** The implementation uses standard OIDC discovery, code flow, nonce, and PKCE behavior rather than a Keycloak-only extension.
- **Consequences:** Imports, SQL-store package names, error prefixes, docs, and tests must change directly. There is no compatibility alias.
- **Status:** accepted

### Decision: Separate local users from issuer-scoped external identities

- **Context:** A subject is issuer-local, while PR 98 needs identity binding and user disablement.
- **Options considered:** Bare subject IDs, derived user ID only, or local users with an external identity relation.
- **Decision:** Store local users separately and bind `(issuer, subject)` identities; use a deterministic opaque local ID for automatic first login.
- **Rationale:** This prevents cross-issuer collision and permits later account linking without changing memberships or resource ownership.
- **Consequences:** SQL first-login must be transactional and uniqueness must be enforced on `(issuer, subject)`.
- **Status:** accepted

### Decision: Keep POST-only local logout

- **Context:** PR 98 supports GET provider logout; the feature branch hardened logout with CSRF and revocation error handling.
- **Options considered:** Restore GET, retain both methods, or keep POST only.
- **Decision:** Register and accept only POST with CSRF.
- **Rationale:** Logout changes application state and cookie/session authority. GET must remain safe and idempotent from the perspective of crawlers and cross-site navigation.
- **Consequences:** PR 98 audit/security observations are retained, but its GET redirect path and tests are removed or rewritten.
- **Status:** accepted

### Decision: Compose one service graph

- **Context:** Authentication behavior depends on shared state and shared policy instances.
- **Options considered:** Construct services separately for native handlers and route enforcement, or construct once and inject everywhere.
- **Decision:** Build stores, clocks, limiter, observer, token services, and identity resolver once per host execution.
- **Rationale:** Split instances create inconsistent rate limits, lifecycle state, and observability.
- **Consequences:** `Services` must expose the same concrete instances used by `AuthOptions` and native handlers.
- **Status:** accepted

## 5. Implementation plan

### Phase 1: Merge and establish the conflict baseline

1. Merge `origin/main` without selecting an entire side.
2. Record every unmerged path and all rename/delete conflicts.
3. Preserve untracked ticket files.
4. Use the merge stages to reconstruct parent intent.

### Phase 2: Resolve identity persistence

1. Define the final `User` and `UserStore` contracts.
2. Combine issuer-scoped lookup, external identity binding, and disablement.
3. Update memory store indexes.
4. Update SQLite and PostgreSQL schemas and queries.
5. Add tests for two issuers sharing a subject, disabled users, binding, and atomic first login.

### Phase 3: Port durable OIDC

1. Move PR 98 transaction types and SQL store to `oidcauth`.
2. Combine injected HTTP client, transaction store, audit, and security observer in `oidcauth.Config`.
3. Use the injected client for discovery, code exchange, and JWKS.
4. Preserve nonce, PKCE, one-time transaction consumption, and local return-path validation.
5. Keep POST-only CSRF logout.

### Phase 4: Compose hostauth

1. Combine builder options for the OIDC HTTP client, security observer, and external OAuth verifier.
2. Preserve PR 98 preflight, proxy, readiness, maintenance, device policy, and token-family services.
3. Pass the same rate limiter and observer to native handlers and planned routes.
4. Register health/readiness and full device lifecycle routes.
5. Register only POST logout.

### Phase 5: Review auto-merges and validate

1. Confirm actor context survives planned dispatch.
2. Confirm OAuth requirements and redacted contexts survive.
3. Confirm BBS action constants and resource rules survive.
4. Run focused package tests.
5. Run `go test ./...`, `go build ./...`, lint, and generation as appropriate.

## 6. Testing strategy

The focused sequence should isolate failures:

```bash
go test ./pkg/gojahttp/auth/appauth ./pkg/gojahttp/auth/appauth/sqlstore
go test ./pkg/gojahttp/auth/oidcauth ./pkg/gojahttp/auth/oidcauth/sqlstore
go test ./pkg/gojahttp/auth/programauth ./pkg/gojahttp/auth/programauth/sqlstore
go test ./pkg/gojahttp ./modules/express
go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/hostauth ./pkg/xgoja/providers/http
go test ./...
go build ./...
```

Security regressions require explicit tests:

- Two issuers using the same `sub` do not resolve to the same identity accidentally.
- A transaction is consumed once and an expired or replayed state has the same public failure.
- The injected client is used for discovery and token/JWKS operations.
- GET logout returns method-not-allowed or is not routed.
- POST logout without CSRF is forbidden.
- A disabled user cannot load a session, approve a device, or use a bound external OAuth identity.
- OAuth routes reject the wrong issuer, resource, or scope.
- Token-authenticated mutations do not run browser-session CSRF.
- JavaScript never sees raw credentials.
- Trusted proxy headers are ignored outside configured proxy prefixes.
- Readiness fails when required stores are unhealthy and recovers when dependencies recover.

## 7. Risks and review guidance

### Implemented result

The design was implemented in merge commit `fdfa96d`. The final code has these review anchors:

- `pkg/gojahttp/auth/appauth/appauth.go:70` defines the canonical external-identity user-store API.
- `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go:75` transactionally inserts fixture users and their issuer-scoped identity binding.
- `pkg/gojahttp/auth/oidcauth/oidcauth.go:23` combines durable transactions, audit/security observers, and the injectable OIDC client.
- `pkg/gojahttp/auth/oidcauth/oidcauth.go:294` implements POST-only CSRF-protected logout.
- `pkg/xgoja/hostauth/builder.go:160` composes health, readiness, device lifecycle, OAuth verification, observability, rate limiting, and the injected OIDC transport.
- `pkg/xgoja/hostauth/builder.go:273` normalizes the complete issuer and subject into the application user.

The old Go package/API names were removed rather than adapted. Repository-wide search finds no `keycloakauth`, `ByKeycloakSub`, or `ByOIDCIdentity` Go references.

The largest risk is not compilation. It is leaving two identity models active. Search for `KeycloakSub`, `ByKeycloakSub`, `keycloak_sub`, and `keycloakauth` after the port. Any remaining production use needs an explicit explanation.

The second risk is a silently split service graph. Review `BuildHostAuthServices` from store creation to the final `Services` literal and trace each concrete object into `AuthOptions`, native handlers, readiness, and maintenance.

The third risk is accepting auto-merged route code without testing the combined semantics. Read `ValidateRoutePlan`, the enforcer, planned dispatch, and the Express builder tests as one pipeline.

## 8. File reference map

- `pkg/gojahttp/auth_plan.go` — typed route security contract, actor, auth result, and OAuth context.
- `pkg/gojahttp/enforcer.go` — host-owned authentication, authorization, CSRF, rate-limit, audit, and OAuth requirement enforcement.
- `pkg/gojahttp/planned_dispatch.go` — installs the authenticated actor into the runtime owner context and invokes JavaScript.
- `modules/express/auth_builders.go` — JavaScript-facing fluent route requirement builders.
- `pkg/gojahttp/auth/appauth/appauth.go` — local user, membership, resource, and authorization contracts.
- `pkg/gojahttp/auth/appauth/sqlstore/` — durable local identity and authorization persistence.
- `pkg/gojahttp/auth/oidcauth/` — provider-neutral browser OIDC flow and login transactions.
- `pkg/gojahttp/auth/programauth/` — agents, device requests, access/refresh tokens, and lifecycle endpoints.
- `pkg/gojahttp/auth/tinyidpauth/` — external Tiny-IDP OAuth bearer introspection and verification.
- `pkg/xgoja/hostauth/config.go` — user/operator configuration model.
- `pkg/xgoja/hostauth/resolve.go` — resolved defaults and validation.
- `pkg/xgoja/hostauth/stores.go` — memory/SQLite/PostgreSQL store construction and health checks.
- `pkg/xgoja/hostauth/builder.go` — complete production dependency composition.
- `pkg/xgoja/hostauth/preflight.go` — fail-fast production deployment invariants.
- `pkg/xgoja/hostauth/readiness.go` — dependency-aware readiness report.
- `pkg/gojahttp/request_identity.go` — direct/trusted-proxy client identity.

## 9. Open questions

Provider single sign-out remains outside this integration. Local logout is fully specified; a future provider-logout endpoint needs a separately reviewed redirect and CSRF contract.

Account linking is enabled structurally by external identity bindings, but there is no user-facing linking workflow in scope. Automatic first login creates one local user per previously unseen issuer identity.

The initial production profile remains single-node where configured by PR 98. Multi-node transaction, rate-limit, and maintenance coordination require distributed implementations and are not implied by this merge.
