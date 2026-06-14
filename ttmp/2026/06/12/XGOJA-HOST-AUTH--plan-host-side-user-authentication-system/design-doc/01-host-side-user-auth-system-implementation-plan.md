---
Title: Host-side user auth system implementation plan
Ticket: XGOJA-HOST-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
    - keycloak
    - oidc
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/16-express-auth-host/cmd/host/main.go
      Note: |-
        Current runnable fake host example to evolve into a dev/demo auth package and smoke scenario.
        Current inline fake services to extract into proposed devauth package
    - Path: pkg/gojahttp/auth_plan.go
      Note: |-
        Existing host auth/resource/authz/CSRF/audit interfaces that the new host auth system should implement.
        Existing host auth/resource/authz/CSRF/audit interface boundary for planned routes
    - Path: pkg/gojahttp/planned_dispatch.go
      Note: |-
        Existing planned-route enforcement pipeline that production and demo auth systems plug into.
        Existing enforcement pipeline that host auth implementations plug into
    - Path: ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/sources/01-keycloak-oidc-session-authz-host-notes.md
      Note: Imported auth2 Keycloak/OIDC/session/authz source material
ExternalSources:
    - ../sources/01-keycloak-oidc-session-authz-host-notes.md
Summary: Opinionated implementation plan for production and dev/demo host-side authentication behind gojahttp planned routes.
LastUpdated: 2026-06-12T17:00:00-04:00
WhatFor: Use this as the roadmap for implementing login/logout, user/session storage, Keycloak/OIDC integration, authorization, capabilities, audit, and demo auth hosts for go-go-goja planned routes.
WhenToUse: Read before adding host-side auth packages or examples beyond the Express planned-route core.
---


# Host-side user auth system implementation plan

## Executive Summary

The Express planned-route work created the framework boundary: JavaScript declares route intent, `pkg/gojahttp` enforces a `RoutePlan`, and the embedding Go host supplies authentication, resource resolution, authorization, CSRF, and audit services. This ticket plans the next layer: an actual host-side user/auth system that implements those services for both production and dev/demo usage.

The production path should be opinionated. It should use Keycloak as the identity provider, OpenID Connect Authorization Code Flow with PKCE, server-side application sessions, normalized application users, app-owned tenant/membership/resource authorization, session-bound CSRF, persistent audit logs, and capability tokens for narrow delegation flows such as invites and email verification.

The dev/demo path should be intentionally simpler. It should provide in-memory users, sessions, resources, authorization policy, CSRF tokens, and audit capture so examples and tests can run without Keycloak, Redis, Postgres, or browser redirects. The dev/demo implementation should model the same interfaces and request flow as production so it teaches the right architecture instead of becoming a bypass path.

The key design rule remains:

> Keycloak authenticates identity. The Go application authorizes actions on specific resources. JavaScript route handlers declare intent and run only after Go enforcement succeeds.

## Problem Statement

The current Express auth implementation is a route-planning and enforcement layer, not a complete application auth system. It provides these interfaces:

```go
type Authenticator interface {
    Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (*Actor, error)
}

type ResourceResolver interface {
    ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
}

type Authorizer interface {
    Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
}

type CSRFProtector interface {
    VerifyCSRF(ctx context.Context, req CSRFRequest) error
}

type AuditSink interface {
    RecordAudit(ctx context.Context, event AuditEvent) error
}
```

That is the right boundary, but applications still need concrete implementations for:

- login and logout,
- app session creation and invalidation,
- Keycloak callback handling,
- user normalization,
- tenant membership,
- resource lookup,
- authorization decisions,
- CSRF tokens,
- audit persistence,
- capability tokens,
- dev/demo bootstrapping.

The implementation must avoid two mistakes:

1. It must not move user storage and identity semantics into `modules/express`.
2. It must not leave each host application to reinvent OIDC, sessions, CSRF, audit, and negative authz testing from scratch.

The answer is an optional host auth kit with clear package boundaries, plus a simpler in-memory kit for demos and tests.

## Proposed Solution

### Package layout

Keep the Express core in place:

```text
modules/express        JavaScript route declaration API
pkg/gojahttp           RoutePlan, Host, dispatch, host auth interfaces
```

Add optional host-side packages under `pkg/gojahttp/auth/`:

```text
pkg/gojahttp/auth/sessionauth
  Session-backed Authenticator and CSRFProtector adapters.

pkg/gojahttp/auth/keycloakauth
  Keycloak/OIDC login, callback, logout, token verification, user normalization hooks.

pkg/gojahttp/auth/appauth
  Opinionated app-owned user, tenant, membership, resource, and authorization contracts.

pkg/gojahttp/auth/capability
  Capability token issuing, redemption, hashing, expiry, revocation, and audit helpers.

pkg/gojahttp/auth/audit
  Audit event normalization and sinks for stdout/dev and SQL-backed production storage.

pkg/gojahttp/auth/devauth
  In-memory demo implementation of users, sessions, resources, authorizer, CSRF, and audit.
```

These packages should implement `gojahttp` interfaces, but `gojahttp` should not import them. The dependency direction should remain:

```text
auth packages -> pkg/gojahttp
pkg/gojahttp  -> no auth package imports
modules/express -> pkg/gojahttp only
```

This keeps the core small and lets production hosts choose the opinionated stack without forcing every xgoja binary to import Keycloak/OIDC/session dependencies.

### Production architecture

```text
Browser
  -> Go host /auth/login
  -> Keycloak authorization endpoint
  <- Go host /auth/callback with authorization code
  -> Go host exchanges code using PKCE
  -> Go host verifies ID token
  -> Go host normalizes Keycloak sub into app_user
  -> Go host creates server-side app session
  <- Browser receives __Host-app session cookie only

Browser request to planned route
  -> gojahttp Host
  -> sessionauth.Authenticator loads app session and user
  -> sessionauth.CSRFProtector verifies unsafe route token
  -> appauth.ResourceResolver loads app resource
  -> appauth.Authorizer checks app policy
  -> audit sink records decision/outcome
  -> JavaScript handler runs
```

The browser should not receive Keycloak access tokens or refresh tokens. If the application needs those tokens for upstream calls, store them server-side, encrypted or otherwise protected according to the deployment's secret-management model.

### Dev/demo architecture

The dev/demo stack should run with no external services:

```text
pkg/gojahttp/auth/devauth
  users:      in-memory map
  sessions:   in-memory map
  resources:  in-memory projects/orgs/users
  authz:      explicit in-memory policy
  csrf:       session-bound token
  audit:      in-memory slice + logger
```

It should provide helper endpoints for examples:

```text
POST /auth/dev/login
POST /auth/dev/logout
GET  /auth/dev/session
```

Seed data should be obvious:

```text
username: demo@example.test
password: demo-password
user id:  u1
tenant:   o1
project:  p1
role:     editor/admin enough for project.update
```

This is not production security. It is a runnable teaching and testing harness that implements the same `gojahttp.AuthOptions` surface.

## Component Design

### 1. Session core (`sessionauth`)

`sessionauth` should be the shared base for production and dev/demo. It should define a small store interface and adapters to `gojahttp.Authenticator` and `gojahttp.CSRFProtector`.

```go
type Session struct {
    ID            string
    UserID        string
    KeycloakSub   string
    Email         string
    EmailVerified bool
    TenantIDs     []string
    CSRFToken     string
    MFAAt         *time.Time
    CreatedAt     time.Time
    LastSeenAt    time.Time
    ExpiresAt     time.Time
    AbsoluteUntil time.Time
    RevokedAt     *time.Time
}

type Store interface {
    Create(ctx context.Context, session Session) error
    Get(ctx context.Context, id string) (*Session, error)
    Touch(ctx context.Context, id string, now time.Time) error
    Rotate(ctx context.Context, oldID string, next Session) error
    Revoke(ctx context.Context, id string) error
}
```

The package should provide:

- secure cookie parsing/setting/clearing helpers,
- session ID generation using CSPRNG,
- session rotation after login,
- idle and absolute timeout checks,
- `Authenticate` implementation reading `__Host-app`,
- `VerifyCSRF` implementation comparing `X-CSRF-Token` to the session token,
- optional `scs` integration or a narrow adapter around `scs`.

Cookie defaults:

```text
name: __Host-app
Secure: true
HttpOnly: true
SameSite: Lax or Strict
Path: /
Domain: unset
```

### 2. Keycloak/OIDC (`keycloakauth`)

`keycloakauth` should own login/callback/logout HTTP handlers and OIDC verification. It should use:

- `github.com/coreos/go-oidc/v3/oidc`,
- `golang.org/x/oauth2`,
- Authorization Code Flow with PKCE,
- state and nonce checks.

Core config:

```go
type Config struct {
    IssuerURL    string
    ClientID     string
    ClientSecret string // optional, depending on client type
    RedirectURL  string
    Scopes       []string
    AfterLoginURL string
    AfterLogoutURL string
}
```

Host integration:

```go
type UserNormalizer interface {
    NormalizeOIDCUser(ctx context.Context, claims OIDCClaims) (*AppUser, error)
}

type Handlers struct {
    Login    http.Handler
    Callback http.Handler
    Logout   http.Handler
}
```

Callback sequence:

```text
verify state
exchange code for tokens
verify ID token issuer/audience/nonce/expiry
extract sub/email/email_verified/name/groups/roles/MFA data if available
normalize app user by keycloak_sub
rotate/create app session
set __Host-app cookie
redirect to application
```

Logout should revoke the app session and clear the app cookie. Keycloak front-channel or end-session redirect can be added, but app session invalidation must not depend on Keycloak logout succeeding.

### 3. Application auth domain (`appauth`)

`appauth` should define opinionated but database-agnostic contracts for users, tenants, memberships, and resources. It should not require a particular ORM.

Minimal model:

```text
app_user:
  id
  keycloak_sub
  email
  display_name
  disabled_at
  created_at
  updated_at

tenant:
  id
  slug
  name
  disabled_at

membership:
  user_id
  tenant_id
  role
  created_at
  revoked_at
```

The package can provide interfaces and simple policy helpers:

```go
type UserStore interface {
    ByID(ctx context.Context, id string) (*User, error)
    ByKeycloakSub(ctx context.Context, sub string) (*User, error)
    UpsertFromOIDC(ctx context.Context, claims OIDCClaims) (*User, error)
}

type MembershipStore interface {
    MembershipsForUser(ctx context.Context, userID string) ([]Membership, error)
    HasRole(ctx context.Context, userID, tenantID string, roles ...string) (bool, error)
}

type ResourceStore interface {
    ResolveResource(ctx context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error)
}
```

Initial authorizer should be explicit Go code:

```go
func (a *Authorizer) Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
    switch req.Action {
    case "user.self.read":
        return allow(req.Actor != nil), nil
    case "project.update":
        return a.canProjectUpdate(ctx, req.Actor, req.Resource)
    case "org.member.invite":
        return a.hasTenantRole(ctx, req.Actor.ID, req.Resource.TenantID, "admin")
    default:
        return deny("unknown action"), nil
    }
}
```

Deny by default. Unknown actions deny. Backend errors should normally deny.

### 4. Audit (`audit`)

The audit package should normalize `gojahttp.AuditEvent` into storage records and provide two sinks:

- `audit.MemorySink` or `audit.LogSink` for dev/demo,
- `audit.SQLSink` for production.

Storage shape:

```text
audit_event:
  id
  event_name
  outcome
  reason
  status_code
  route_name
  method
  pattern
  action
  actor_id
  actor_kind
  tenant_id
  resource_type
  resource_id
  request_id
  ip_hash
  user_agent
  attributes_json
  created_at
```

Never log raw secrets:

- no session IDs,
- no access tokens,
- no refresh tokens,
- no authorization codes,
- no raw capability tokens.

### 5. Capabilities (`capability`)

Capabilities are narrow delegated authorities: invite links, email verification links, password reset links, temporary file links, and scoped API tokens. They should not replace normal per-resource authorization.

Store only hashed tokens:

```go
type Capability struct {
    ID           string
    Purpose      string
    SubjectID    string
    ResourceType string
    ResourceID   string
    Claims       map[string]string
    TokenHash    []byte
    ExpiresAt    time.Time
    SingleUse    bool
    UsedAt       *time.Time
    RevokedAt    *time.Time
    CreatedBy    string
    CreatedAt    time.Time
}
```

Rules:

```text
no capability without purpose
no capability without expiry
subject or resource required
single-use by default for links
revocable by default
token returned once
raw token never logged
redeem is atomic
```

Generic JavaScript capability builders can come later. First implement Go service APIs and one concrete use case, such as organization invites.

### 6. Dev/demo (`devauth`)

`devauth` should package what the current example does manually.

```go
type Kit struct {
    HostOptions gojahttp.AuthOptions
    Login       http.Handler
    Logout      http.Handler
    Session     http.Handler
    Audit       *MemoryAuditSink
}
```

A demo host should be able to write:

```go
kit := devauth.New(devauth.Config{Seed: devauth.DefaultSeed()})
host := gojahttp.NewHost(gojahttp.HostOptions{
    RejectRawRoutes: true,
    Auth: kit.AuthOptions(),
})

mux := http.NewServeMux()
mux.Handle("POST /auth/dev/login", kit.LoginHandler())
mux.Handle("POST /auth/dev/logout", kit.LogoutHandler())
mux.Handle("/", host)
```

The current `examples/xgoja/16-express-auth-host` should then be updated to use `devauth` instead of carrying all fake services inline.

## Design Decisions

### Decision 1: Keep production auth outside `modules/express`

Status: proposed

`modules/express` should not import Keycloak, session, SQL, or user-store code. Its job is to expose a JavaScript route API. Production auth belongs to optional host-side packages that implement `gojahttp` interfaces.

### Decision 2: Keycloak authenticates, app authorizes

Status: proposed

Use Keycloak for login, MFA, federation, account lifecycle, groups, and coarse roles. Use the app database and Go policy code for object ownership, tenant membership, workflow states, billing limits, and per-resource permissions.

### Decision 3: Browser receives an app session cookie, not IdP tokens

Status: proposed

OIDC tokens stay server-side. The browser receives an opaque `__Host-app` cookie. This reduces token leakage and keeps planned-route authentication a simple session lookup.

### Decision 4: Start authz with explicit Go functions

Status: proposed

Do not start with Casbin, OpenFGA, OPA, or Keycloak Authorization Services. Use explicit Go policy functions plus negative tests first. Add a policy engine only when the model becomes too complex for readable Go.

### Decision 5: Provide dev/demo auth as a first-class package

Status: proposed

Examples should not each reimplement fake auth. A dev/demo package makes smoke tests easier and gives users a working mental model before they configure Keycloak.

## Alternatives Considered

### Build a user store into Express

Rejected. It would couple a JavaScript routing module to application identity, DB schema, password/OIDC choices, and product policy.

### Use Keycloak Authorization Services from day one

Deferred. It can model fine-grained authorization, but it adds operational and conceptual weight. The first production host should prove app-owned authorization through explicit Go checks.

### Use OPA/OpenFGA/Casbin immediately

Deferred. These are useful when rules become complex, relationship-heavy, or centrally managed. Starting with explicit Go code makes tests and onboarding simpler.

### Make dev/demo auth use the same Keycloak path

Rejected for demos. A no-external-service demo is important for examples, CI smoke tests, and onboarding. It should model the same interfaces but not require Keycloak.

## Implementation Plan

### Phase 1: Dev/demo auth kit

1. Create `pkg/gojahttp/auth/devauth`.
2. Move the fake services from `examples/xgoja/16-express-auth-host/cmd/host/main.go` into reusable package types.
3. Add in-memory login/logout/session handlers.
4. Add session-cookie authentication and CSRF instead of bearer-only auth.
5. Update example 16 to use `devauth`.
6. Extend smoke tests:
   - `GET /me` before login -> 401,
   - bad login -> 401,
   - good login -> 200 + cookie + CSRF token,
   - `GET /me` with cookie -> 200,
   - unsafe mutation without CSRF -> 403,
   - unsafe mutation with CSRF -> 200,
   - logout -> 204,
   - `GET /me` after logout -> 401.

### Phase 2: Session auth package

1. Create `pkg/gojahttp/auth/sessionauth`.
2. Define session store interface.
3. Implement cookie helpers and secure defaults.
4. Implement `gojahttp.Authenticator` from session store.
5. Implement `gojahttp.CSRFProtector` from session store.
6. Add memory store for tests.
7. Add tests for expiry, revocation, missing cookie, session rotation, and CSRF mismatch.

### Phase 3: Production Keycloak/OIDC handlers

1. Create `pkg/gojahttp/auth/keycloakauth`.
2. Add config validation for issuer/client/redirect.
3. Add login handler with state, nonce, PKCE.
4. Add callback handler with token exchange and ID-token verification.
5. Add user normalization interface.
6. Create app session through `sessionauth.Store`.
7. Add logout handler that revokes app session and clears cookie.
8. Add integration tests with a fake OIDC provider or httptest JWKS/issuer.

### Phase 4: App auth contracts and explicit policy

1. Create `pkg/gojahttp/auth/appauth`.
2. Define `UserStore`, `MembershipStore`, and resource-store interfaces.
3. Define action constants for example/common actions.
4. Implement a simple explicit authorizer helper.
5. Add negative authorization tests for cross-user and cross-tenant access.

### Phase 5: Audit and capabilities

1. Create `pkg/gojahttp/auth/audit` with log/memory sink.
2. Define SQL record mapper without forcing a specific DB library.
3. Create `pkg/gojahttp/auth/capability`.
4. Implement token issue/redeem/revoke with hashed token storage interface.
5. Add one example flow: organization invite.

### Phase 6: Production example and docs

1. Add an example production host skeleton, likely `examples/xgoja/17-express-keycloak-auth-host`.
2. Provide docker-compose or documented external Keycloak setup only if needed.
3. Add help docs explaining dev/demo versus production auth.
4. Update migration docs to show how route plans plug into sessionauth/keycloakauth.

## Testing and Validation Strategy

Required test categories:

- session cookie parsing and security defaults,
- session expiry and revocation,
- CSRF valid/missing/mismatch,
- OIDC state/nonce/PKCE validation,
- ID token issuer/audience/expiry validation,
- user normalization by stable `sub`, not email,
- authorization negative tests,
- cross-tenant resource denial,
- audit redaction tests,
- capability token hashing and single-use redemption,
- example smoke tests.

Validation commands should include:

```bash
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
go test ./pkg/gojahttp/auth/... -count=1
make -C examples/xgoja/16-express-auth-host smoke
```

After the generated-build VCS stamping issue is fixed, full validation should be plain:

```bash
go test ./... -count=1
```

## Risks

- OIDC correctness is security-sensitive. Keep scope narrow and rely on `go-oidc`/`oauth2` rather than hand-rolled token validation.
- Session defaults must be safe. Insecure cookie settings should require explicit development-mode opt-in.
- Authorization bugs are often negative-case bugs. Tests must prove that user A cannot access user B's or tenant B's resources by guessing IDs.
- Audit can leak secrets if event attributes are not redacted. Add redaction tests early.
- Package boundaries can blur if examples reach into internals. Keep public constructors small and explicit.

## Open Questions

1. Which persistent session store should be first-class: Postgres, Redis, or adapter-only?
2. Should `sessionauth` wrap `alexedwards/scs` directly, or define a smaller store interface and provide an optional scs adapter?
3. Should production examples require a local Keycloak container, or should Keycloak remain documented while tests use a fake OIDC issuer?
4. Should strict audit mode be added now or later?
5. How much schema/body validation should be implemented before production host auth is declared shippable?

## References

- Source notes: `../sources/01-keycloak-oidc-session-authz-host-notes.md`
- Existing route-plan interfaces: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go`
- Existing planned dispatch: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/planned_dispatch.go`
- Existing runnable fake host: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/16-express-auth-host/cmd/host/main.go`
- Express auth wrap-up article: `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md`
