---
Title: Investigation diary
Ticket: XGOJA-HOST-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
    - keycloak
    - oidc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md
      Note: Wrapped-up Obsidian project report before opening this follow-up ticket.
    - Path: examples/xgoja/16-express-auth-host/README.md
      Note: Updated dev auth cookie/CSRF/login/logout usage docs (commit 38871dc)
    - Path: examples/xgoja/16-express-auth-host/cmd/host/main.go
      Note: Runnable host example refactored to use devauth and login/logout smoke flow (commit 38871dc)
    - Path: pkg/gojahttp/auth/devauth/devauth.go
      Note: Dev/demo auth kit implementation for planned routes (commit 38871dc)
    - Path: pkg/gojahttp/auth/devauth/devauth_test.go
      Note: Unit coverage for dev auth login/session/CSRF/resource/authz/audit behavior (commit 38871dc)
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Reusable session-cookie Authenticator and CSRFProtector package (commit d939b95)
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth_test.go
      Note: Sessionauth behavior tests for cookies
    - Path: ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/design-doc/01-host-side-user-auth-system-implementation-plan.md
      Note: Primary design plan for production and dev/demo host auth packages.
    - Path: ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/sources/01-keycloak-oidc-session-authz-host-notes.md
      Note: Imported /tmp/auth2.md source material.
ExternalSources:
    - ../sources/01-keycloak-oidc-session-authz-host-notes.md
Summary: Chronological diary for planning the host-side user/auth system after Express planned auth cutover.
LastUpdated: 2026-06-12T17:00:00-04:00
WhatFor: Use this to resume or review the host-side auth system planning ticket.
WhenToUse: Read before implementing dev/demo auth, session auth, Keycloak/OIDC, app authorization, audit, or capabilities.
---



# Diary

## Goal

This diary records the creation of `XGOJA-HOST-AUTH`, a follow-up ticket for the actual host-side user/auth system behind the Express planned-route framework.

## Step 1: Wrap up Express auth article and create host-auth planning ticket

I first updated the Obsidian vault article for the completed Express planned-auth cutover, because the previous report still described the earlier MVP state. The wrapped-up article now records the hard verb-helper cutover, strict raw-route rejection, CSRF/audit support, runnable host example, remaining generated-build caveat, and the boundary between Express route planning and future host-side auth packages.

After that, I created a new docmgr ticket for the host-side user/auth system. The new plan separates an opinionated production stack from a simple dev/demo stack while keeping both behind the existing `gojahttp.AuthOptions` interfaces.

### Prompt Context

**User prompt (verbatim):** "ok, let's make a new ticket and plan out the implementation of the actual host side user/auth system. While I do want a production ready yet opinionated (on tech choices) setup, we should also have a simple version for dev/demo purposes. After creating the new ticket, give me an update on what that would lok like.

But before you do any of that, let's update and wrap up the project report we wrote in the obsidian vault, as we have finished implementing the express cutover. /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md

Only then continue with the new ticket."

**Assistant interpretation:** First update the existing Obsidian article to reflect the completed Express cutover, then create a new docmgr ticket for planning the concrete host-side auth/user system with both production and dev/demo implementations.

**Inferred user intent:** Close out the Express route-planning phase cleanly and start a separate implementation track for real authentication, sessions, users, authorization, and demo tooling.

**Commit (code):** N/A — documentation/ticket planning only in this step.

### What I did
- Rewrote the Obsidian article at `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md` to reflect the final Express planned-auth implementation state.
- Committed the Obsidian article update in the vault as `7defced` — `Wrap up go-go-goja Express auth article`.
- Created docmgr ticket `XGOJA-HOST-AUTH` with title `Plan host-side user authentication system` and topics `goja,http,security,xgoja,keycloak,oidc`.
- Imported `/tmp/auth2.md` into the new ticket as `sources/01-keycloak-oidc-session-authz-host-notes.md`.
- Created the primary design document `design-doc/01-host-side-user-auth-system-implementation-plan.md`.
- Wrote the design around two tracks:
  - production: Keycloak/OIDC, server-side sessions, app user normalization, app-owned authorization, audit, capabilities;
  - dev/demo: in-memory users/sessions/resources/authz/CSRF/audit with login/logout smoke support.
- Added and checked ticket task 1 for importing source material.

### Why
- The Express planned-auth work is now a framework boundary, not the entire auth product. A separate ticket keeps production identity/session/user-store work from being conflated with the Express route API.
- `/tmp/auth2.md` specifically argues for Keycloak/OIDC, app sessions, app-owned authorization, and capabilities. Those concerns need their own plan because they carry a much larger security and maintenance burden than route builders.
- A dev/demo implementation is needed so examples and tests remain runnable without Keycloak, Redis, Postgres, or browser redirects.

### What worked
- `docmgr ticket create-ticket` created the new workspace successfully.
- The imported source validated under `docmgr doctor` after frontmatter was added.
- The new design document was able to reuse the existing `gojahttp` interfaces instead of proposing a new enforcement boundary.
- The Obsidian report now matches the current state of the branch instead of the earlier MVP state.

### What didn't work
- No command failures occurred in this planning step.
- No Go tests were run because no production Go code was changed.

### What I learned
- The host-side work should be split into optional packages rather than moved into `modules/express`.
- The likely package boundary is `pkg/gojahttp/auth/...`, with packages such as `sessionauth`, `keycloakauth`, `appauth`, `capability`, `audit`, and `devauth`.
- The dev/demo stack should model the same interfaces as production, not bypass them, so it remains useful as onboarding material and smoke coverage.

### What was tricky to build
- The main tricky part was deciding how much opinion to put into the host layer without turning `go-go-goja` into a full application framework. The design resolves this by making Keycloak/session/authz packages optional and host-side, while keeping `modules/express` focused on route declarations.
- Another subtlety was the production/dev split. If the dev system is too different from production, examples teach the wrong model. If it is too production-like, it becomes hard to run. The proposed `devauth` package uses in-memory stores but implements the same `gojahttp` interfaces.

### What warrants a second pair of eyes
- Review whether the proposed package names and boundaries are right before implementation begins.
- Review whether `sessionauth` should wrap `alexedwards/scs` directly or expose a smaller store interface with optional adapters.
- Review whether `keycloakauth` belongs in this repo long-term or should eventually become a separate module once stabilized.

### What should be done in the future
- Implement Phase 1 first: extract a reusable `devauth` package from the current runnable example and add login/logout/session-cookie smoke coverage.
- Then implement `sessionauth` as the reusable base for production and demo session-backed authentication.
- Keep Keycloak/OIDC production work behind optional package imports.

### Code review instructions
- Start with `ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/design-doc/01-host-side-user-auth-system-implementation-plan.md`.
- Compare the proposed package boundaries against:
  - `pkg/gojahttp/auth_plan.go`,
  - `pkg/gojahttp/planned_dispatch.go`,
  - `examples/xgoja/16-express-auth-host/cmd/host/main.go`.
- Validate ticket hygiene with:
  - `docmgr doctor --ticket XGOJA-HOST-AUTH --stale-after 30`.

### Technical details
- Proposed optional host-auth package layout:
  ```text
  pkg/gojahttp/auth/sessionauth
  pkg/gojahttp/auth/keycloakauth
  pkg/gojahttp/auth/appauth
  pkg/gojahttp/auth/capability
  pkg/gojahttp/auth/audit
  pkg/gojahttp/auth/devauth
  ```
- Proposed first implementation target:
  ```text
  devauth: in-memory users, sessions, resources, authorization, CSRF, audit, login/logout handlers
  ```
- Proposed production foundation:
  ```text
  Keycloak OIDC Authorization Code + PKCE -> app user normalization -> server-side session -> gojahttp Authenticator/CSRFProtector -> ResourceResolver -> Authorizer -> AuditSink
  ```


## Step 2: Implement dev/demo auth kit and refactor runnable example

I implemented the first concrete host-auth slice: a reusable `devauth` package that provides in-memory users, sessions, CSRF, resources, authorization, audit capture, and login/logout/session handlers. The existing runnable Express auth host example now uses this package instead of carrying one-off fake services inline.

This keeps the example runnable with no external dependencies while moving it closer to the production architecture. Planned routes are still enforced through the same `gojahttp.AuthOptions` interfaces that a Keycloak/session-backed host will implement later.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** After planning the phases, begin implementing them sequentially, committing at sensible boundaries and recording detailed diary notes.

**Inferred user intent:** Turn the host-auth plan into working code while preserving an implementation trail that makes review and continuation straightforward.

**Commit (code):** 38871dc906b8e151b7b986a35ed728ae2ef50c7c — "Add dev auth kit for planned routes"

### What I did
- Rewrote `tasks.md` into detailed phases and task lists covering:
  - planning,
  - dev/demo auth,
  - shared session auth,
  - Keycloak/OIDC,
  - app auth domain,
  - audit/capabilities,
  - examples/docs/handoff.
- Added `pkg/gojahttp/auth/devauth` with:
  - `DefaultSeed()` for `demo@example.test` / `demo-password`, user `u1`, tenant `o1`, project `p1`, and an editor role,
  - in-memory sessions with CSPRNG session IDs and CSRF tokens,
  - `AuthOptions()` returning implementations for `Authenticator`, `ResourceResolver`, `Authorizer`, `CSRFProtector`, and `AuditSink`,
  - `POST /auth/dev/login`, `POST /auth/dev/logout`, and `GET /auth/dev/session` handlers,
  - in-memory audit capture plus optional logging.
- Added `pkg/gojahttp/auth/devauth/devauth_test.go` covering:
  - login success/failure,
  - session-cookie authentication,
  - CSRF success/failure,
  - resource resolution,
  - authorization allow/deny,
  - logout invalidation,
  - session handler and audit capture.
- Refactored `examples/xgoja/16-express-auth-host/cmd/host/main.go` to use `devauth.New(...).AuthOptions()`.
- Extended the example smoke flow to cover:
  - public health route,
  - `/me` before login -> 401,
  - bad login -> 401,
  - good login -> cookie + CSRF token,
  - `/auth/dev/session` -> 200,
  - `/me` after login -> 200,
  - unsafe project update without CSRF -> 403,
  - unsafe project update with CSRF -> 200,
  - missing project -> 404,
  - logout -> 204,
  - `/me` after logout -> 401.
- Updated `examples/xgoja/16-express-auth-host/README.md` with login/logout/cookie/CSRF curl usage.
- Added generated `pkg/gojahttp/auth/devauth/logcopter.go` after `go generate ./...` created it.

### Why
- The inline fake services in the example were useful proof, but they were not reusable. The new `devauth` package gives tests and examples a shared no-external-service host implementation.
- The dev/demo package lets future docs teach the same layering as production: Go host owns auth services; JavaScript declares planned route policy.
- Moving from bearer-token demo auth to session-cookie auth makes the example closer to the intended browser/BFF model from `/tmp/auth2.md`.

### What worked
- Package and example validation passed:
  ```bash
  go test ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host -count=1
  ```
- Targeted validation passed:
  ```bash
  go test ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
  ```
- Example smoke validation passed:
  ```bash
  make -C examples/xgoja/16-express-auth-host smoke
  ```
- The final commit pre-hook passed lint, `go generate ./...`, and `go test ./...`.

### What didn't work
- The first commit attempt failed because `go generate ./...` generated `pkg/gojahttp/auth/devauth/logcopter.go`, which defines a package-level variable named `log`. The new `devauth.go` file imported the standard library logger as `log`, causing a package-level name collision:
  ```text
  pkg/gojahttp/auth/devauth/logcopter.go:7:5: log already declared through import of package log ("log")
      pkg/gojahttp/auth/devauth/devauth.go:15:2: other declaration of log
  ```
- I fixed this by aliasing the standard library import to `stdlog` and updating the logger types/calls.
- I then added the generated `logcopter.go` file to the commit.

### What I learned
- New packages in this repository can get generated `logcopter.go` files during the pre-commit `go generate ./...` step. Avoid importing the standard `log` package under the default name in those packages if a generated `log` symbol will exist.
- The current `gojahttp` interfaces were sufficient for a practical session-cookie demo. No changes to the planned-route enforcement boundary were needed.
- A `devauth` package is a good proving ground for the later `sessionauth` API because it immediately exposes which session-cookie and CSRF operations should become reusable.

### What was tricky to build
- The main tricky part was preserving the example's simplicity while making it realistic enough to teach session-cookie auth. The smoke test now needs cookie persistence and a CSRF token extracted from login, so it uses a `cookiejar` and explicit login response decoding.
- Another subtlety was logout: because logout is a Go-native endpoint rather than a planned JavaScript route, it has to perform its own CSRF check. That mirrors production host endpoints, where login/callback/logout may live outside Express planned routes but still need security checks.
- The generated logcopter naming collision was also non-obvious because targeted `go test` passed before `go generate ./...` created the generated file during the pre-commit hook.

### What warrants a second pair of eyes
- Review whether `devauth.Config` has the right public surface before more examples depend on it.
- Review whether `devauth` should eventually be rewritten on top of `sessionauth` after Phase 2, or whether it should remain fully self-contained.
- Review whether logout should return 204 for missing sessions, as implemented, or 401 for stricter demos.
- Review whether dev passwords should remain plaintext in memory or be represented as explicitly named demo-only secrets to avoid accidental production copy/paste.

### What should be done in the future
- Start Phase 2 by extracting reusable session-cookie and CSRF mechanics into `pkg/gojahttp/auth/sessionauth`.
- Consider reusing `sessionauth` inside `devauth` after the shared package exists.
- Add docs/help pages once the session and Keycloak package boundaries are clearer.

### Code review instructions
- Start with `pkg/gojahttp/auth/devauth/devauth.go` for the new package API and interface implementations.
- Review `pkg/gojahttp/auth/devauth/devauth_test.go` for behavior coverage.
- Review `examples/xgoja/16-express-auth-host/cmd/host/main.go` to see how a host wires `devauth` into `gojahttp.NewHost` and serves login/logout endpoints beside planned routes.
- Review `examples/xgoja/16-express-auth-host/README.md` for user-facing instructions.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
  make -C examples/xgoja/16-express-auth-host smoke
  ```

### Technical details
- Basic host wiring:
  ```go
  kit := devauth.New(devauth.Config{})
  host := gojahttp.NewHost(gojahttp.HostOptions{
      Dev:             true,
      RejectRawRoutes: true,
      Auth:            kit.AuthOptions(),
  })
  ```
- Login/logout mux wiring:
  ```go
  mux := http.NewServeMux()
  mux.Handle("POST /auth/dev/login", kit.LoginHandler())
  mux.Handle("POST /auth/dev/logout", kit.LogoutHandler())
  mux.Handle("GET /auth/dev/session", kit.SessionHandler())
  mux.Handle("/", host)
  ```
- Smoke credential:
  ```text
  username: demo@example.test
  password: demo-password
  ```


## Step 3: Add reusable session-cookie auth package

I implemented the second host-auth phase by adding `sessionauth`, a reusable package for server-side session-cookie authentication and CSRF verification. This package is lower-level than `devauth`: it does not know about demo users, demo projects, or demo authorization, and instead focuses on the common session mechanics needed by both development and production hosts.

The package gives later Keycloak/OIDC work a stable target: after OIDC callback verification and app-user normalization, production code can create an application session, set the session cookie, and let planned routes authenticate through the same `gojahttp.Authenticator` and `gojahttp.CSRFProtector` interfaces.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing the planned phases sequentially after the dev/demo package, with commits and diary updates.

**Inferred user intent:** Build the reusable session foundation before starting the heavier Keycloak/OIDC production package.

**Commit (code):** d939b95e8b3127cc7b5049d343bedbe413411231 — "Add reusable session auth package"

### What I did
- Added `pkg/gojahttp/auth/sessionauth/sessionauth.go`.
- Defined reusable session types:
  - `Session`,
  - `Store`,
  - `ActorLoader`,
  - `ActorLoaderFunc`,
  - `Manager`.
- Added secure-by-default cookie behavior:
  - default secure cookie name `__Host-app`,
  - default secure cookie flag enabled,
  - explicit `AllowInsecureHTTP` escape hatch for localhost/dev tests,
  - insecure default cookie name `goja_app_session` when insecure mode is enabled.
- Implemented CSPRNG session IDs and CSRF tokens through `RandomToken()`.
- Implemented session creation with idle and absolute expiry timestamps.
- Implemented `Manager.Authenticate` as a `gojahttp.Authenticator`.
- Implemented `Manager.VerifyCSRF` as a `gojahttp.CSRFProtector` using `X-CSRF-Token`.
- Implemented cookie set/clear helpers.
- Implemented `MemoryStore` for tests and local development with create/get/touch/rotate/revoke operations.
- Added `sessionauth_test.go` covering:
  - authenticate success,
  - CSRF success,
  - missing and invalid cookies,
  - expired sessions,
  - revoked sessions,
  - rotated sessions,
  - CSRF mismatch,
  - cookie clearing,
  - custom actor loader projection.
- Marked Phase 2 tasks complete.

### Why
- `devauth` proved the example-level flow, but production Keycloak/OIDC needs a reusable session package that is not tied to demo users or resources.
- Browser-facing production auth should use an opaque app session cookie rather than exposing Keycloak tokens to JavaScript or local storage.
- CSRF verification should be session-bound and available as a host-side `gojahttp.CSRFProtector` so planned routes can continue declaring `.csrf()` without knowing session internals.

### What worked
- Session package validation passed:
  ```bash
  go test ./pkg/gojahttp/auth/sessionauth -count=1
  ```
- Combined host-auth validation passed:
  ```bash
  go test ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
  ```
- The runnable auth example smoke test still passed:
  ```bash
  make -C examples/xgoja/16-express-auth-host smoke
  ```
- The commit pre-hook passed lint, `go generate ./...`, and `go test ./...`.

### What didn't work
- No new command failures occurred during this phase.
- I did not refactor `devauth` to use `sessionauth` yet. That is intentionally left as a follow-up because Phase 2 first needed a standalone reusable package with tests.

### What I learned
- A small `Store` interface is enough for the first session abstraction: `Create`, `Get`, `Touch`, `Rotate`, and `Revoke`.
- `Touch` needs to update both `LastSeenAt` and `IdleExpiresAt`; otherwise an idle timeout would never renew on activity.
- Keeping actor projection behind `ActorLoader` avoids baking app-user lookup into session storage. Production code can load richer users from a DB; tests can use the default projection from session fields.

### What was tricky to build
- The main tricky part was designing secure defaults without making tests and localhost examples painful. The compromise is explicit: secure cookies and `__Host-app` are the default, while `AllowInsecureHTTP` opts into an HTTP-friendly cookie name and non-secure cookie flag.
- Another subtle point was error mapping. `Manager.Authenticate` converts missing/invalid/expired/revoked session conditions into `gojahttp.ErrUnauthenticated` so planned-route dispatch returns 401 rather than leaking session-store details.
- Session rotation is represented at the store level but not yet wired into a login handler. That gives Keycloak callback code the primitive it needs for session fixation defense.

### What warrants a second pair of eyes
- Review whether the `Store.Touch(ctx, id, now, idleExpiresAt)` signature is the right abstraction or whether it should accept a richer update struct.
- Review whether the default absolute timeout of 12 hours and idle timeout of 30 minutes are acceptable defaults for this package.
- Review whether `DefaultActorForSession` should include `KeycloakSub` in actor claims or keep it out by default.
- Review whether `MemoryStore.Revoke` should use an injected clock rather than `time.Now()`; current use is acceptable for tests but less deterministic than manager-owned clocks.

### What should be done in the future
- Consider refactoring `devauth` to use `sessionauth.Manager` internally.
- Start Phase 3 by implementing `keycloakauth` on top of `sessionauth`.
- Add persistent store adapters after the package API gets one more review.

### Code review instructions
- Start with `pkg/gojahttp/auth/sessionauth/sessionauth.go`, especially `Config`, `Manager`, `SessionFromRequest`, `Authenticate`, and `VerifyCSRF`.
- Review `MemoryStore` for store semantics and concurrency behavior.
- Review `pkg/gojahttp/auth/sessionauth/sessionauth_test.go` for the expected behavior matrix.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/sessionauth -count=1
  go test ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1
  ```

### Technical details
- Session manager setup for localhost HTTP:
  ```go
  manager, err := sessionauth.New(sessionauth.Config{
      Store:             sessionauth.NewMemoryStore(),
      AllowInsecureHTTP: true,
  })
  ```
- Production-shaped setup keeps secure defaults:
  ```go
  manager, err := sessionauth.New(sessionauth.Config{
      Store:       postgresSessionStore,
      ActorLoader: appUserActorLoader,
  })
  ```
- Planned-route integration:
  ```go
  host := gojahttp.NewHost(gojahttp.HostOptions{
      Auth: manager.AuthOptions(),
  })
  ```
