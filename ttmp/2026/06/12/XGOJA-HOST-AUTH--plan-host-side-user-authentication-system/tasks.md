# Tasks

## Phase 0 — Ticket setup and planning

- [x] Import `/tmp/auth2.md` as source material.
- [x] Write production host auth implementation plan.
- [x] Define dev/demo host auth implementation plan.
- [x] Define package boundaries between Express core, `gojahttp` interfaces, and optional host auth kits.
- [x] Keep the investigation diary current after each implementation slice.
- [x] Keep the changelog and related-file links current after each implementation slice.

## Phase 1 — Dev/demo auth kit

Goal: provide a no-external-service auth kit that implements the existing `gojahttp.AuthOptions` interfaces and can replace the inline fake services in `examples/xgoja/16-express-auth-host`.

- [x] Create `pkg/gojahttp/auth/devauth` package.
- [x] Define dev seed data for users, tenants/roles, projects/resources, and default credentials.
- [x] Implement in-memory session creation, lookup, revocation, and CSRF token storage.
- [x] Implement `gojahttp.Authenticator` using a dev session cookie.
- [x] Implement `gojahttp.CSRFProtector` using a session-bound `X-CSRF-Token` header.
- [x] Implement `gojahttp.ResourceResolver` for demo project resources.
- [x] Implement `gojahttp.Authorizer` for `user.self.read`, `project.read`, and `project.update` demo actions.
- [x] Implement `gojahttp.AuditSink` with in-memory capture and optional logging.
- [x] Add `POST /auth/dev/login`, `POST /auth/dev/logout`, and `GET /auth/dev/session` handlers.
- [x] Add unit tests for login success/failure, authenticate success/failure, CSRF success/failure, resource resolution, authorization denial, logout, and audit capture.
- [x] Refactor `examples/xgoja/16-express-auth-host` to use `devauth` instead of inline fake services.
- [x] Extend example smoke coverage to exercise login, cookie-authenticated `/me`, CSRF denial, CSRF success, resource 404, logout, and post-logout 401.
- [x] Update the example README with login/logout/cookie/CSRF usage.
- [x] Validate with `go test ./pkg/gojahttp/auth/devauth ./examples/xgoja/16-express-auth-host/cmd/host ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1` and `make -C examples/xgoja/16-express-auth-host smoke`.

## Phase 2 — Shared session auth package

Goal: extract reusable session-cookie authentication and CSRF mechanics that can be used by both dev/demo and production Keycloak-backed hosts.

- [x] Create `pkg/gojahttp/auth/sessionauth` package.
- [x] Define `Session`, `Store`, and `UserLoader`/actor projection interfaces.
- [x] Implement secure cookie helpers with safe defaults and explicit development-mode escape hatches.
- [x] Implement CSPRNG session ID and CSRF token generation.
- [x] Implement idle timeout, absolute timeout, revocation, and session rotation semantics.
- [x] Implement a memory store for tests and demos.
- [x] Implement `gojahttp.Authenticator` backed by the session store.
- [x] Implement `gojahttp.CSRFProtector` backed by the session store.
- [x] Add tests for missing cookie, invalid cookie, expired session, revoked session, rotated session, actor projection, CSRF mismatch, CSRF success, and cookie clearing.
- [x] Decide whether `sessionauth` wraps `alexedwards/scs` directly or remains adapter-first.

## Phase 3 — Production Keycloak/OIDC package

Goal: provide an opinionated browser login/logout flow using Keycloak as IdP and server-side app sessions for planned routes.

- [x] Create `pkg/gojahttp/auth/keycloakauth` package.
- [x] Add configuration for issuer URL, client ID, optional client secret, redirect URL, scopes, and post-login/logout redirects.
- [x] Implement OIDC provider discovery and config validation.
- [x] Implement login handler with state, nonce, and PKCE verifier storage.
- [x] Implement callback handler with state verification, code exchange, ID token verification, nonce verification, and claim extraction.
- [x] Define `UserNormalizer` for mapping stable Keycloak `sub` values into app users.
- [x] Create/rotate app sessions through `sessionauth` after successful callback.
- [x] Implement logout handler that revokes the app session and clears the app cookie even if Keycloak end-session fails.
- [x] Add tests with a fake OIDC issuer/JWKS or httptest provider for success, bad state, bad nonce, expired token, wrong audience, and normalization failure.
- [x] Document Keycloak realm/client settings, including Authorization Code + PKCE and disabled password grant for browser clients.

## Phase 4 — App auth domain and explicit authorization

Goal: provide app-owned user, tenant, membership, resource, and action contracts without embedding a full policy engine.

- [x] Create `pkg/gojahttp/auth/appauth` package.
- [x] Define user, tenant, membership, resource, and action model types.
- [x] Define `UserStore`, `MembershipStore`, and `ResourceStore` interfaces.
- [x] Implement resource resolver helpers that map `gojahttp.ResourceRequest` into app resource lookups.
- [x] Implement explicit Go authorizer helpers with deny-by-default behavior.
- [x] Add action constants for common example actions.
- [x] Add negative authorization tests for cross-user, cross-tenant, missing membership, unknown action, missing resource, and backend error cases.
- [x] Document when to graduate from explicit Go checks to Casbin/OpenFGA/OPA/Keycloak Authorization Services.

## Phase 5 — Audit and capabilities

Goal: add reusable helpers for persistent audit records and narrow capability-token workflows.

- [x] Create `pkg/gojahttp/auth/audit` package.
- [x] Map `gojahttp.AuditEvent` into a normalized record shape.
- [x] Implement memory/log audit sinks.
- [x] Define an adapter interface for SQL/persistent audit sinks without forcing a DB library.
- [x] Add audit redaction tests ensuring tokens/session IDs/capability secrets are not logged.
- [x] Create `pkg/gojahttp/auth/capability` package.
- [x] Define capability token model with purpose, subject/resource, claims, expiry, single-use, revocation, and hashed token storage.
- [x] Implement issue, redeem, revoke, and atomic single-use semantics.
- [x] Add tests for expiry, wrong purpose, revocation, double redemption, token hashing, and audit hooks.
- [x] Implement one concrete example flow, preferably organization invite acceptance.

## Phase 6 — Production example, docs, and handoff

Goal: make the host auth system discoverable and runnable for both dev/demo and production-oriented users.

- [x] Add or update a dev/demo example that uses `devauth` and planned Express routes.
- [x] Add a production-oriented Keycloak host skeleton example, likely `examples/xgoja/17-express-keycloak-auth-host`.
- [x] Add a Docker Compose Keycloak setup for the production-oriented example and smoke testing.
- [x] Add docs explaining the boundary between Express planned routes, `gojahttp`, `devauth`, `sessionauth`, and `keycloakauth`.
- [ ] Add Glazed help pages for host auth setup and dev/demo auth usage.
- [ ] Add release/migration notes explaining that user stores remain host-owned.
- [ ] Validate `docmgr doctor --ticket XGOJA-HOST-AUTH --stale-after 30`.
- [x] Validate package tests and example smokes.
- [ ] Optionally upload the final ticket docs to reMarkable.
