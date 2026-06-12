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
    - Path: examples/xgoja/17-express-keycloak-auth-host/cmd/host/main.go
      Note: Production-oriented Keycloak/session/appauth/audit host wiring (commit 852780d)
    - Path: examples/xgoja/17-express-keycloak-auth-host/docker-compose.yml
      Note: Docker Compose Keycloak service for local OIDC testing (commit 852780d)
    - Path: examples/xgoja/17-express-keycloak-auth-host/keycloak/realm-goja-demo.json
      Note: Imported demo realm/client/user configuration (commit 852780d)
    - Path: examples/xgoja/17-express-keycloak-auth-host/scripts/server.js
      Note: Planned Express routes protected by the Go host auth stack (commit 852780d)
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: App-owned resource resolver and explicit deny-by-default authorizer helpers (commit 952acb2)
    - Path: pkg/gojahttp/auth/appauth/appauth_test.go
      Note: Negative authorization and store behavior coverage (commit 952acb2)
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Audit record normalization
    - Path: pkg/gojahttp/auth/audit/audit_test.go
      Note: Audit normalization and secret-redaction tests (commit 4141b8a)
    - Path: pkg/gojahttp/auth/capability/capability.go
      Note: Capability issue/redeem/revoke service with hashed token storage (commit 4141b8a)
    - Path: pkg/gojahttp/auth/capability/invite.go
      Note: Concrete organization invite capability helper flow (commit 4141b8a)
    - Path: pkg/gojahttp/auth/devauth/devauth.go
      Note: Dev/demo auth kit implementation for planned routes (commit 38871dc)
    - Path: pkg/gojahttp/auth/devauth/devauth_test.go
      Note: Unit coverage for dev auth login/session/CSRF/resource/authz/audit behavior (commit 38871dc)
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: Keycloak/OIDC Authorization Code + PKCE handlers backed by sessionauth (commit f297487)
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth_test.go
      Note: Fake OIDC issuer/JWKS coverage for success and negative callback cases (commit f297487)
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


## Step 4: Add Keycloak/OIDC auth handlers

I implemented the production-oriented OIDC adapter package for Phase 3. The package is named `keycloakauth` because Keycloak is the intended IdP, but the implementation uses standard OIDC discovery, OAuth2 Authorization Code Flow, PKCE, ID-token verification, nonce checks, and a host-provided user normalizer.

The package deliberately creates an application session through `sessionauth` after OIDC verification. That keeps Keycloak tokens on the server side and lets planned routes continue authenticating with an opaque app session cookie.

### Prompt Context

**User prompt (verbatim):** "phase 3, phase 4"

**Assistant interpretation:** Continue sequential implementation with Phase 3 (Keycloak/OIDC host auth package) and Phase 4 (app auth domain helpers).

**Inferred user intent:** Move beyond demo/session foundations into the production IdP adapter and app-owned authorization layer.

**Commit (code):** f297487eec0b0faed15c01f321d0f8155526f756 — "Add Keycloak OIDC auth handlers"

### What I did
- Added `pkg/gojahttp/auth/keycloakauth` with:
  - `Config`,
  - `Handlers`,
  - `OIDCClaims`,
  - `UserSession`,
  - `UserNormalizer`,
  - `TransactionStore`,
  - `MemoryTransactionStore`.
- Implemented `New(ctx, Config)` with OIDC provider discovery via `coreos/go-oidc` and OAuth2 config setup.
- Implemented `GET /auth/login` handler:
  - generates state,
  - generates nonce,
  - generates PKCE verifier,
  - stores a login transaction,
  - redirects to the provider authorization endpoint with S256 PKCE challenge and nonce.
- Implemented `GET /auth/callback` handler:
  - rejects callback errors,
  - validates state,
  - exchanges code with the PKCE verifier,
  - verifies `id_token`,
  - checks nonce,
  - extracts claims,
  - calls `UserNormalizer`,
  - creates an app session through `sessionauth.Manager`,
  - sets the app session cookie,
  - redirects to the original safe return URL.
- Implemented logout handler:
  - revokes the app session if present,
  - clears the session cookie,
  - returns 204 for POST or redirects for GET.
- Added `pkg/gojahttp/auth/keycloakauth/README.md` documenting recommended Keycloak client settings:
  - Authorization Code Flow,
  - PKCE S256,
  - disabled password grant for browser clients,
  - disabled implicit flow,
  - app session cookie rather than browser-visible IdP tokens.
- Added a fake OIDC issuer/JWKS/token test provider covering:
  - successful login/callback/session creation,
  - bad state,
  - bad nonce,
  - expired ID token,
  - wrong audience,
  - user normalization failure,
  - logout session revocation.
- Added `sessionauth.Manager.RevokeRequestSession` so logout handlers can revoke the request session without knowing cookie internals.
- Added `go-oidc` and `oauth2` dependencies.
- Marked Phase 3 tasks complete.

### Why
- The production stack needs Keycloak/OIDC login but planned routes should not process raw OIDC tokens directly.
- OIDC callback handling is host/application infrastructure, not JavaScript route logic.
- A normalizer keeps the Keycloak `sub` to app-user mapping app-owned and stable, instead of treating email as identity.

### What worked
- Targeted tests passed:
  ```bash
  go test ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth -count=1
  ```
- Broader auth package tests passed during commit pre-hook.
- The commit pre-hook passed lint, `go generate ./...`, and `go test ./...`.

### What didn't work
- No command failures occurred in this phase.
- `go generate ./...` later produced a generated `logcopter.go` file for `keycloakauth`; I committed generated auth package loggers in a small follow-up commit `3c8dc11`.

### What I learned
- A fake OIDC provider with an RSA key, discovery endpoint, JWKS endpoint, auth endpoint, and token endpoint is enough to test the end-to-end login/callback flow without Docker or a live Keycloak.
- The package should keep its transaction storage pluggable. The memory transaction store is fine for tests and single-process demos, but production multi-instance hosts need shared storage.
- `return_to` must be constrained to local absolute paths; the login handler rejects empty/external-style values by falling back to `AfterLoginURL`.

### What was tricky to build
- The trickiest part was testing OIDC verification without introducing a live Keycloak dependency. The fake provider signs real RS256 JWTs and exposes JWKS so `go-oidc` verifies issuer, audience, expiry, and signature normally.
- Another subtle point was nonce verification. `go-oidc` verifies the token, but the application still has to compare the ID token nonce against the login transaction nonce.
- The handler must avoid returning Keycloak tokens to the browser. The package only uses the token response server-side and creates a separate app session.

### What warrants a second pair of eyes
- Review whether `MemoryTransactionStore` should accept an injected clock for deterministic expiry tests.
- Review whether logout should eventually redirect to Keycloak's end-session endpoint in addition to local app-session revocation.
- Review whether the normalizer should receive OAuth2 token metadata or only verified ID-token claims.

### What should be done in the future
- Add the Phase 6 Docker Compose Keycloak example/smoke as requested by the user.
- Add a production persistent transaction store if multiple host instances need to serve callbacks.
- Consider storing server-side access/refresh tokens only through an explicit, encrypted token-store interface if upstream API calls require them.

### Code review instructions
- Start with `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`, especially `handleLogin`, `handleCallback`, and `handleLogout`.
- Review `pkg/gojahttp/auth/keycloakauth/keycloakauth_test.go` for the fake OIDC issuer and negative verification cases.
- Review `pkg/gojahttp/auth/keycloakauth/README.md` for Keycloak client setup assumptions.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth -count=1
  ```

### Technical details
- Keycloak handler setup:
  ```go
  handlers, err := keycloakauth.New(ctx, keycloakauth.Config{
      IssuerURL:      "https://keycloak.example/realms/app",
      ClientID:       "goja-app",
      RedirectURL:    "https://app.example/auth/callback",
      SessionManager: sessions,
      UserNormalizer: normalizer,
  })
  ```
- Host routes:
  ```go
  mux.Handle("GET /auth/login", handlers.LoginHandler())
  mux.Handle("GET /auth/callback", handlers.CallbackHandler())
  mux.Handle("POST /auth/logout", handlers.LogoutHandler())
  ```


## Step 5: Add app-owned authorization domain helpers

I implemented Phase 4 by adding `appauth`, a small app-owned authorization helper package. It defines minimal user, tenant, membership, resource, and action contracts plus a deny-by-default authorizer that is intentionally boring Go code rather than a policy engine.

This package gives host applications and examples a concrete starting point for object/tenant authorization while keeping the design rule intact: Keycloak authenticates identity, and the Go application authorizes actions on specific resources.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Continue with Phase 4 after implementing the Keycloak/OIDC adapter.

**Inferred user intent:** Add the app-owned domain layer needed for resource resolution and authorization behind planned routes.

**Commit (code):** 952acb277dfdafcdcd5eb0986d16e1b559135e48 — "Add app auth domain helpers"

### What I did
- Added `pkg/gojahttp/auth/appauth` with:
  - `User`,
  - `Tenant`,
  - `Membership`,
  - `Resource`,
  - `UserStore`,
  - `MembershipStore`,
  - `ResourceStore`,
  - `Resolver`,
  - `Authorizer`,
  - `MemoryStore`.
- Added action constants:
  - `user.self.read`,
  - `user.self.update`,
  - `project.read`,
  - `project.update`,
  - `org.member.invite`.
- Implemented `Resolver` as a `gojahttp.ResourceResolver` that maps `gojahttp.ResourceRequest` into app resource lookup and tenant-bound `ResourceRef` projection.
- Implemented `Authorizer` as a `gojahttp.Authorizer` with deny-by-default behavior:
  - unknown action denies,
  - missing actor denies,
  - missing resource denies where required,
  - tenant membership required for `project.read`,
  - admin/editor required for `project.update`,
  - admin required for `org.member.invite`,
  - self-resource match required for `user.self.update`.
- Added `MemoryStore` for tests/examples with user lookup, Keycloak-sub lookup, OIDC upsert, memberships, role checks, and resource lookup.
- Added tests for:
  - resource resolution success,
  - tenant mismatch and missing resource denial,
  - allowed actions,
  - cross-user denial,
  - cross-tenant denial,
  - missing membership/unknown action/missing resource denial,
  - backend error propagation,
  - user lookup and OIDC upsert.
- Added `pkg/gojahttp/auth/appauth/README.md` documenting when to use explicit Go checks and when to graduate to a policy engine.
- Marked Phase 4 tasks complete.

### Why
- The OIDC package can authenticate a user, but it should not decide whether that user can update a project or invite a member.
- App-owned authorization must account for tenant membership, object ownership, resource type, and action, not just broad Keycloak roles.
- Starting with explicit Go checks and negative tests keeps the first production path auditable.

### What worked
- Package tests passed:
  ```bash
  go test ./pkg/gojahttp/auth/appauth ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp/auth/devauth -count=1
  ```
- The commit pre-hook passed lint, `go generate ./...`, and `go test ./...`.

### What didn't work
- No command failures occurred in this phase.
- `go generate ./...` produced generated `logcopter.go` files for `appauth` and `keycloakauth`; I committed them in follow-up commit `3c8dc11`.

### What I learned
- The `gojahttp.ResourceRequest` shape is sufficient for an app-owned resolver because it already carries resource type, resolved ID, tenant ID, actor, and request context.
- A useful default authorizer can stay very small if it is explicit about action names and resource types.
- Negative authorization tests are the most important tests in this layer; they prove guessed IDs and tenant mismatches do not accidentally pass.

### What was tricky to build
- The main tricky part was avoiding the temptation to design a full generic policy engine. `appauth` is deliberately a starter package and should remain easy to replace with application-specific policy code.
- Another subtlety was organization invites. An org resource may use its own ID as the tenant boundary, so the authorizer treats `ResourceRef{Type:"org", ID:"o1"}` as tenant `o1` when `TenantID` is empty.

### What warrants a second pair of eyes
- Review whether the built-in action constants are too example-specific for `appauth`, or whether they are useful enough as common defaults.
- Review whether `MemoryStore.UpsertFromOIDC` should generate IDs as `user:<sub>` or leave ID assignment entirely to application code.
- Review whether backend errors should be returned as errors, denied decisions, or both; the current implementation returns both a denied decision and the backend error.

### What should be done in the future
- Wire `appauth` into a production-oriented Keycloak example with Docker Compose Keycloak smoke testing.
- Add persistent store adapters only after the interface shape has been reviewed.
- Keep documenting when to move from explicit Go checks to Casbin/OpenFGA/OPA/Cedar.

### Code review instructions
- Start with `pkg/gojahttp/auth/appauth/appauth.go`, especially `Resolver.ResolveResource` and `Authorizer.Authorize`.
- Review `pkg/gojahttp/auth/appauth/appauth_test.go` for negative authorization coverage.
- Review `pkg/gojahttp/auth/appauth/README.md` for package positioning.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/appauth ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp/auth/devauth -count=1
  ```

### Technical details
- Host wiring sketch:
  ```go
  store := appauth.NewMemoryStore()
  host := gojahttp.NewHost(gojahttp.HostOptions{
      Auth: gojahttp.AuthOptions{
          Authenticator: sessions,
          CSRF:          sessions,
          Resources:     appauth.Resolver{Store: store},
          Authorizer:    appauth.Authorizer{Memberships: store},
      },
  })
  ```


## Step 6: Add audit and capability helper packages

I implemented Phase 5 by adding two reusable security-support packages: `audit` for normalized planned-route audit records, and `capability` for narrow hashed bearer-token delegation flows. These packages are still host-side infrastructure; they do not change the Express JavaScript API.

The audit package gives planned routes and auth helpers a storage-friendly record shape with redaction. The capability package covers one-time or scoped flows such as organization invite acceptance while ensuring raw tokens are returned once, stored only as hashes, and never placed into audit attributes.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue implementing the remaining planned phases after Phase 3 and Phase 4.

**Inferred user intent:** Keep moving through the host-auth task list, with commits and diary updates.

**Commit (code):** 4141b8a2cb9109ada057652c97cb5cdc840f471a — "Add audit and capability auth helpers"

### What I did
- Added `pkg/gojahttp/auth/audit` with:
  - normalized `Record` type,
  - `Normalizer`,
  - `Store` interface,
  - `Sink` adapter for persistent stores,
  - `MemorySink`,
  - `LogSink`,
  - recursive attribute redaction.
- Added audit tests covering:
  - actor/resource/route/request metadata normalization,
  - request ID, user agent, and IP hash mapping,
  - recursive redaction of token/session/cookie/authorization/capability-looking attributes,
  - memory sink,
  - store sink,
  - log sink redaction.
- Added `pkg/gojahttp/auth/capability` with:
  - `Capability`,
  - `IssueSpec`,
  - `IssueResult`,
  - `Store`,
  - `Service`,
  - `MemoryStore`,
  - token hashing,
  - issue/redeem/revoke operations,
  - audit hooks.
- Added concrete organization invite helpers:
  - `IssueOrgInvite`,
  - `AcceptOrgInvite`,
  - `PurposeOrgInviteAccept`,
  - `AcceptedOrgInvite`.
- Added capability tests covering:
  - issue and redeem,
  - single-use double redemption denial,
  - wrong purpose,
  - not found token,
  - expiry,
  - revocation,
  - hashed storage,
  - audit hook behavior,
  - org invite acceptance flow.
- Added README files for both packages.
- Marked Phase 5 tasks complete.
- Added generated logcopter files in follow-up commit `61c101e` — "Add generated audit capability loggers".

### Why
- Planned routes can emit audit events today, but production hosts need a reusable path from runtime events to storage records without leaking secrets.
- Capability tokens are a recurring auth-adjacent need for invites, email verification, password reset, API tokens, and one-time links.
- These helpers keep capability issuance/redeeming in Go, where expiry, revocation, single-use semantics, hashing, and audit can be enforced consistently.

### What worked
- Targeted package tests passed:
  ```bash
  go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/capability -count=1
  ```
- Broader auth package tests passed:
  ```bash
  go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/capability ./pkg/gojahttp/auth/appauth ./pkg/gojahttp/auth/keycloakauth ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp/auth/devauth -count=1
  ```
- The commit pre-hook passed lint, `go generate ./...`, and `go test ./...`.

### What didn't work
- No implementation command failed in this phase.
- As with the earlier auth packages, `go generate ./...` produced generated `logcopter.go` files after the main code commit, so I added them in a small follow-up commit.

### What I learned
- Audit redaction should operate on keys rather than values. A value like a random token cannot be reliably recognized after the fact, but keys such as `accessToken`, `sessionID`, `authorization`, or `capability` are strong signals.
- Capability services should return raw tokens only in the issue result. Stored and redeemed capability objects should redact `TokenHash` from the public result where possible.
- A concrete helper like `IssueOrgInvite` makes the generic capability service easier to understand and test without committing to a broad JavaScript capability API yet.

### What was tricky to build
- The main tricky part was balancing generic capability primitives with a concrete example. I added a generic `Service` plus one small org-invite helper so the package is usable without pretending to be a full capability algebra.
- Another subtle point is atomic single-use redemption. The `Store.Redeem` interface performs purpose/expiry/revocation/used checks and marks single-use tokens as used under the store lock in `MemoryStore`. A SQL store should preserve that atomicity with a transaction or conditional update.
- Audit events for denied capability redemption cannot include the stored capability when the token is invalid, so denied audit events include the requested purpose but no raw token.

### What warrants a second pair of eyes
- Review whether audit redaction keys are broad enough without hiding too much useful context.
- Review whether `capability.Store.Redeem` has the right signature for SQL implementations.
- Review whether `Capability.TokenHash` should ever be returned from `ByID`, or whether stores should always expose a redacted view through separate methods.
- Review whether audit event names should be package constants.

### What should be done in the future
- Add persistent SQL examples/adapters if a concrete database layer is chosen.
- Wire `audit.Sink` and `capability.Service` into the planned Docker Compose Keycloak example.
- Continue to Phase 6 with the requested Docker Compose Keycloak example and smoke testing.

### Code review instructions
- Start with `pkg/gojahttp/auth/audit/audit.go` and `pkg/gojahttp/auth/audit/audit_test.go` for audit normalization/redaction behavior.
- Then review `pkg/gojahttp/auth/capability/capability.go`, `invite.go`, and `capability_test.go` for issue/redeem/revoke semantics.
- Validate with:
  ```bash
  go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/capability -count=1
  ```

### Technical details
- Audit sink wiring:
  ```go
  host := gojahttp.NewHost(gojahttp.HostOptions{
      Auth: gojahttp.AuthOptions{
          Audit: audit.Sink{Store: sqlAuditStore},
      },
  })
  ```
- Organization invite capability flow:
  ```go
  issued, err := svc.IssueOrgInvite(ctx, capability.OrgInviteSpec{
      OrgID: "o1",
      Email: "new@example.test",
      Role:  "viewer",
      TTL:   7 * 24 * time.Hour,
  })

  accepted, err := svc.AcceptOrgInvite(ctx, issued.Token)
  ```


## Step 7: Add Docker Compose Keycloak host example

I implemented the production-oriented example phase by adding a new `17-express-keycloak-auth-host` example. It shows how the host-side auth packages fit together around planned Express routes: Keycloak authenticates with OIDC Authorization Code + PKCE, `sessionauth` issues the app cookie and CSRF token, `appauth` owns resource/tenant authorization, and `audit` records route outcomes.

This is intentionally a host skeleton, not a JavaScript auth framework. The JavaScript file still only declares planned routes and policy intent; all identity, session, resource, authorization, and audit decisions stay in Go.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue from the helper packages into the production-oriented Keycloak example requested earlier.

**Inferred user intent:** Produce a runnable reference for the production auth stack, using Docker Compose Keycloak as requested.

**Commit (code):** 852780dc5d6dd0d7d55ce0f27a5667a3b5abce55 — "Add Keycloak auth host example"

### What I did
- Added `examples/xgoja/17-express-keycloak-auth-host`.
- Added a Docker Compose Keycloak service importing a `goja-demo` realm.
- Added a realm export with:
  - public OIDC client `goja-app`,
  - Authorization Code flow enabled,
  - PKCE S256 requirement,
  - demo user `demo@example.test` / `demo-password`,
  - redirect URI `http://127.0.0.1:8790/auth/callback`.
- Added a Go host that wires:
  - `keycloakauth` login/callback/logout handlers,
  - `sessionauth` session cookies and CSRF verification,
  - `appauth` in-memory app resources and authorization,
  - `audit.LogSink`,
  - strict planned-route enforcement.
- Added a planned Express route script with:
  - public `/healthz`,
  - authenticated `/me`,
  - CSRF-protected project update route.
- Added README instructions and Makefile targets for Keycloak lifecycle and serving the host.
- Updated `examples/xgoja/README.md` and completed the relevant Phase 6 task checklist items.

### Why
- The earlier `16-express-auth-host` example demonstrates local dev/demo auth without external services.
- This new example demonstrates the intended production shape: Keycloak handles identity, the Go host creates opaque app sessions, and the application remains responsible for authorization.
- Docker Compose makes the example reproducible without requiring a pre-existing Keycloak installation.

### What worked
- The example host package compiled:
  ```bash
  go test ./examples/xgoja/17-express-keycloak-auth-host/cmd/host -count=1
  ```
- Docker Compose syntax validated:
  ```bash
  docker compose -f examples/xgoja/17-express-keycloak-auth-host/docker-compose.yml config >/tmp/keycloak-compose-config.txt
  ```
- The full auth helper package set passed:
  ```bash
  go test ./examples/xgoja/17-express-keycloak-auth-host/cmd/host ./pkg/gojahttp/auth/... -count=1
  ```
- The dev auth example smoke still passed:
  ```bash
  make -C examples/xgoja/16-express-auth-host smoke
  ```
- The commit pre-hook also ran lint, `go generate ./...`, and `go test ./...` successfully.

### What didn't work
- I did not run a full browser/form-login smoke against the Docker Compose Keycloak container in this step. The compose file is syntax-validated and the host compiles, but a complete automated login smoke still needs to be added.

### What I learned
- The production example reinforces the desired boundary: JavaScript route code names route-level requirements, but does not parse or own tokens, sessions, users, tenants, or resource permissions.
- `keycloakauth.UserNormalizer` is the right seam for mapping OIDC subject/email claims into app-owned users, memberships, and session claims.
- The example needs a tiny `/auth/session` host endpoint because CSRF tokens belong to the app session, not to Keycloak or Express route declarations.

### What was tricky to build
- The main tricky point was keeping the example realistic without turning it into a production deployment. I left persistent stores and TLS out, but documented them explicitly as production requirements.
- The host has to mount auth endpoints before delegating to the `gojahttp.Host`, while still letting the host serve planned Express routes for all other paths.
- The Keycloak client must be public and configured for standard flow + PKCE, not password grants, to match the desired production direction.

### What warrants a second pair of eyes
- Verify the imported Keycloak realm attributes for PKCE are correct for the pinned Keycloak image.
- Review whether the Docker Compose healthcheck is portable enough across developer environments.
- Review whether `appStore.AddMembership` inside the normalizer should be idempotent in a stricter way if the example is later backed by SQL.
- Review cookie/security settings before anyone copies the example into production; the example uses `AllowInsecureHTTP` for local development.

### What should be done in the future
- Add a true automated Keycloak smoke that starts Compose, completes browser/form login, verifies `/me`, fetches CSRF from `/auth/session`, and updates a project route.
- Add Glazed help pages for `devauth`, `sessionauth`, `keycloakauth`, `appauth`, `audit`, and `capability` package boundaries.
- Consider a persistent-store variant once the repository has an agreed database layer for sessions/audit/users.

### Code review instructions
- Start with `examples/xgoja/17-express-keycloak-auth-host/README.md` for intended flow and operator commands.
- Review `cmd/host/main.go` for package wiring and separation of Keycloak identity from app authorization.
- Review `scripts/server.js` to confirm JavaScript only declares planned route requirements.
- Validate with:
  ```bash
  docker compose -f examples/xgoja/17-express-keycloak-auth-host/docker-compose.yml config
  go test ./examples/xgoja/17-express-keycloak-auth-host/cmd/host -count=1
  ```

### Technical details
- Start Keycloak:
  ```bash
  make -C examples/xgoja/17-express-keycloak-auth-host keycloak-up
  ```
- Run host:
  ```bash
  make -C examples/xgoja/17-express-keycloak-auth-host serve
  ```
- Demo account:
  ```text
  demo@example.test / demo-password
  ```
