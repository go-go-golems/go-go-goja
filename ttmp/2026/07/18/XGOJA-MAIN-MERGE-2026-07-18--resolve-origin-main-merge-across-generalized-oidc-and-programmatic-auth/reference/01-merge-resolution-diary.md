---
Title: Merge resolution diary
Ticket: XGOJA-MAIN-MERGE-2026-07-18
Status: active
Topics:
    - auth
    - oidc
    - security
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/assets/public/app.js
      Note: Uses POST plus session CSRF for browser logout
    - Path: repo://pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Retained generalized OIDC and POST-only local logout semantics
    - Path: repo://pkg/gojahttp/planned_dispatch_test.go
      Note: Preserves actor-context tests and adds redacted token-auth behavior
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Merged host-owned OIDC and programmatic-auth service composition
    - Path: repo://pkg/xgoja/hostauth/builder_test.go
      Note: Validates native routes and memory/SQL program-auth service bundles
ExternalSources: []
Summary: Chronological record of resolving the origin/main merge across generalized OIDC, programmatic authentication, route dispatch, and logout clients.
LastUpdated: 2026-07-18T19:08:09.510138045-04:00
WhatFor: Reviewing the merge decisions, reproduced failures, validation evidence, and deferred security work.
WhenToUse: When reviewing commit 083618b or changing hostauth, OIDC logout, device authorization, or planned route authentication.
---


# Merge resolution diary

## Goal

Resolve the in-progress merge of `origin/main` into `task/prod-tiny-idp` without losing either branch's intended authentication work. The feature branch generalized the browser identity-provider integration from Keycloak-specific code to OIDC and hardened browser logout. `origin/main` added programmatic agent authentication, device authorization, bearer-token dispatch, rate limiting, and richer authentication results.

The user's request was: “Ok, go ahead and fix it. Commit at appropriate intervals, keep a detailed diary”. I interpreted “fix it” as resolving the existing merge conflicts, integrating compatible behavior from both parents, deliberately choosing between incompatible security behavior, running focused and repository-wide validation, and committing the merge plus this review record.

## Context

The merge began with five unmerged paths:

- `pkg/gojahttp/auth/oidcauth/oidcauth.go`
- `pkg/gojahttp/auth/oidcauth/oidcauth_test.go`
- `pkg/gojahttp/planned_dispatch_test.go`
- `pkg/xgoja/hostauth/builder.go`
- `pkg/xgoja/hostauth/builder_test.go`

The important conflict was semantic, not textual. The feature branch had replaced the older `keycloakauth` naming with a provider-neutral `oidcauth` package, added stable issuer identity to normalized claims, supported an injected HTTP client for same-process discovery, used public-client PKCE behavior, made logout a CSRF-protected POST, and treated revocation failures as failures. Main had meanwhile added program-auth stores and services, a composite session/bearer authenticator, device endpoints, a rate limiter, redacted authentication metadata for JavaScript, and a provider end-session redirect exposed as GET logout.

The resolution keeps generalized OIDC and the stricter local logout contract. It does not restore a state-changing GET `/auth/logout`. Provider-wide logout is therefore deferred until it can have an explicit, safe contract rather than being hidden inside a GET endpoint. At the same time, the resolution composes the programmatic-auth stores and services into the host-owned service graph and preserves main's richer planned-route authentication behavior.

## Quick Reference

### Chronological work log

#### 19:00–19:08 — inspect and classify

- Compared `HEAD` (`c6e464c`), `MERGE_HEAD` (`6a1a095`), and merge base (`fe6b1f6`).
- Read all three index stages for each conflict instead of selecting one parent wholesale.
- Classified the OIDC/provider-logout conflict as a security-policy choice.
- Classified hostauth builder and dispatch tests as additive conflicts that required a union of both parents.
- Created ticket `XGOJA-MAIN-MERGE-2026-07-18` and this diary before editing the merge.

#### 19:08 — restore a clean conflict-editing baseline

I selected the feature branch versions of the five conflicted files as an editing baseline, planning to add main's compatible behavior explicitly. The first command failed because this checkout is a linked worktree and its index lives outside the writable sandbox:

```text
fatal: Unable to create '/home/manuel/code/wesen/go-go-golems/go-go-goja/.git/worktrees/go-go-goja23/index.lock': Read-only file system
```

The same narrowly scoped `git checkout --ours` succeeded with permission to write the linked-worktree index. This did not complete the resolution; it only removed conflict markers so the combined implementation could be authored and reviewed normally.

#### 19:09–19:12 — compose host services

- Constructed one in-memory rate limiter and passed that same instance into `AuthOptions` and `Services`.
- Constructed agent, API-token, OAuth-token, and device services over `stores.ProgramAuth`.
- Used `programauth.CompositeAuthenticator` so a protected route accepts the browser session flow or supported bearer credentials.
- Added native `POST /auth/device/start`, `POST /auth/device/token`, and `POST /auth/device/approve` handlers when a device store exists.
- Kept OIDC's injected HTTP client so discovery, token exchange, and JWKS access work with the same-process issuer architecture.
- Exposed stores and constructed program-auth services through `Services`; this matters to generated/custom hosts that need the same concrete service graph used by enforcement.

The intended composition is:

```text
session manager ───────────────┐
API-token service ─────────────┼─> composite authenticator ─> planned route enforcement
OAuth access-token service ────┘

program-auth stores -> agent/token/device services -> native device handlers
memory rate limiter -------------------------------> route enforcement
OIDC HTTP client -> discovery/token/JWKS ----------> browser OIDC handlers
```

#### 19:12 — preserve both dispatch-test families

- Preserved the feature test proving the authenticated actor is installed in the runtime owner's context.
- Added main's test proving JavaScript receives a redacted, copied `ctx.auth` result rather than raw credentials or mutable host-owned scope slices.
- Added main's test proving bearer-token authentication can bypass session CSRF when `CSRFRequired` is false.
- Added a test authenticator adapter implementing both the legacy actor method and the richer `AuthenticateResult` method; this is test scaffolding, not a production compatibility layer.

#### 19:13 — align browser examples with the route contract

The three newly merged inbox UIs redirected a browser to `GET /auth/logout`, but the retained server route is `POST /auth/logout` with CSRF. Each UI now:

```javascript
await fetch("/auth/logout", {
  method: "POST",
  headers: csrfHeaders()
});
```

On success it clears local UI state and navigates to `/`; on failure it leaves the session state visible and reports the error. This avoids pretending logout succeeded when the protected POST failed.

#### 19:14 — formatting failure and correction

The first `gofmt` pass found an editing error in the appended handler slice:

```text
pkg/xgoja/hostauth/builder.go:157:3: expected operand, found '{'
```

The combined code appended elements to an existing `[]NativeHandler`, so each appended composite literal needed the explicit `NativeHandler{...}` type. I corrected the four entries and reran formatting successfully. Conflict-marker search and call-site search were then clean.

#### 19:15 — focused tests reveal an incomplete service payload

The first focused run passed `oidcauth` and `gojahttp` but failed two `hostauth` tests. The builder had constructed working program-auth services for `AuthOptions` and device handlers but had not copied their stores/services or the limiter into the returned `Services` payload. The diagnostic dump showed `AuthOptions.RateLimiter` populated while `Services.RateLimiter`, `AgentStore`, and related fields were nil.

I populated all corresponding `Services` fields from the same instances. This prevents a split service graph in which enforcement works but generated/custom host integrations see empty service handles.

#### 19:16 — sandbox-only test failure

After that correction, the focused tests hit an environment restriction rather than a code failure:

```text
panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted
```

The unchanged focused tests passed when rerun with loopback permission:

```text
ok github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/oidcauth
ok github.com/go-go-golems/go-go-goja/pkg/gojahttp
ok github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth
```

#### 19:16–19:18 — full validation

- `go test ./...` passed.
- The first `go build ./...` attempt could not read linked-worktree metadata for Go VCS stamping and printed repeated `error obtaining VCS status: exit status 128` messages.
- The unchanged `go build ./...` passed with permission to read that metadata.
- Conflict-resolution paths passed `git diff --cached --check`.
- A repository-wide staged whitespace check reported older whitespace in documentation arriving from main. Those unrelated upstream documents were preserved rather than mechanically rewritten during an auth merge.

#### 19:16–19:18 — checkpoint commit

Created merge commit `083618b` (`Merge origin/main into task/prod-tiny-idp`). The pre-commit hook independently ran:

- `golangci-lint` and glazed lint: zero issues.
- `go generate ./...`: successful, including the Dagger-backed bun demo generation.
- `go test ./...`: successful.

Git's recorded-resolution output confirmed all five conflicted paths were resolved in the commit.

### Resulting route contract

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/auth/login` | Begin browser OIDC authorization. |
| `GET` | `/auth/callback` | Complete browser OIDC authorization. |
| `POST` | `/auth/logout` | Revoke/clear the local application session with CSRF protection. |
| `GET` | `/auth/session` | Return the local session projection and CSRF token. |
| `POST` | `/auth/device/start` | Begin programmatic device authorization. |
| `POST` | `/auth/device/token` | Poll the device grant for issued tokens. |
| `POST` | `/auth/device/approve` | Approve a device code using the authenticated browser session. |

There is intentionally no `GET /auth/logout` route.

## Usage Examples

Review the merge with:

```bash
git show --cc 083618b
go test ./pkg/gojahttp/auth/oidcauth ./pkg/gojahttp ./pkg/xgoja/hostauth
go test ./...
go build ./...
```

When changing hostauth composition, verify identity, not merely non-nil fields: the limiter in `Services` should be the limiter passed into `AuthOptions`, and service methods should operate over the stores exposed in `Services`.

When changing logout, preserve the distinction between local application-session logout and provider single sign-out. Adding provider logout later requires an explicit design for redirect allow-listing, revocation failure semantics, CSRF, and whether the user requested local logout or identity-provider logout.

## Related

- Merge commit: `083618b`
- `pkg/xgoja/hostauth/builder.go`
- `pkg/xgoja/hostauth/builder_test.go`
- `pkg/gojahttp/planned_dispatch.go`
- `pkg/gojahttp/planned_dispatch_test.go`
- `pkg/gojahttp/auth/oidcauth/oidcauth.go`
- `pkg/gojahttp/auth/programauth/`
- `examples/xgoja/23-personal-knowledge-inbox/`

### Deferred follow-up

Provider end-session support remains intentionally deferred. It should be implemented as a separately named and reviewed flow, not by weakening the local logout route to a state-changing GET. No backwards-compatibility GET adapter was added.
