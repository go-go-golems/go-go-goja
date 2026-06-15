---
Title: PR 74 Code Review Report
Ticket: XGOJA-PR74-CODE-REVIEW-PLAN
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: modules/express/auth_builders.go
      Note: |-
        Fluent route builder and trusted Go-backed object boundary; reviewed in Phase 1-3
        confirmed trusted-builder boundary + copy-on-auth
    - Path: pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go
      Note: |-
        SQL appauth store; finding F3 (UpsertFromOIDC race) lives here
        F2 UpsertFromOIDC SELECT-then-INSERT race vs UNIQUE keycloak_sub
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: |-
        Audit normalization and RedactMap; finding F1 lives here
        F1 over-redaction in secretKey() substring match list
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: |-
        OIDC adapter; finding F4 (doc accuracy) and Phase 6 review target
        F3 doc accuracy re IdP token retention; full OIDC review
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Session/CSRF manager; reviewed in Phase 3
    - Path: pkg/gojahttp/auth_plan.go
      Note: RoutePlan contract and ValidateRoutePlan; reviewed in Phase 1-2
    - Path: pkg/gojahttp/planned_dispatch.go
      Note: |-
        Runtime enforcement pipeline; every denial path traced in Phase 2-3
        traced full denial matrix in buildSecureEnvelope/servePlannedRoute
    - Path: pkg/xgoja/hostauth/builder.go
      Note: |-
        Lazy generated-host auth service factory; reviewed in Phase 4
        confirmed lazy BuildHostAuthServices + failure-closing cleanup
    - Path: pkg/xgoja/providers/http/serve.go
      Note: serve and hot reload lifecycle; reviewed in Phase 4/9
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/pull/74
Summary: Line-anchored code review of PR 74 (Express planned auth + host auth), conducted per the methodology in design/01-...; records findings, confirms invariants, and gives a merge recommendation.
LastUpdated: 2026-06-15T16:30:00-04:00
WhatFor: The actual review deliverable for PR 74; read alongside the methodology guide.
WhenToUse: Use to triage PR 74 findings, drive fixes, and decide merge readiness.
---


# PR 74 Code Review Report

## Scope and environment

- **Branch:** `task/goja-express-auth`
- **Head:** `b66baea869583d79db6b0e8ec5007e0fad0e5ef7`
- **Base:** `origin/main` (`d406577f97866c816a4bd0fd0d2c5284143c2cc0`)
- **PR:** https://github.com/go-go-golems/go-go-goja/pull/74 — "Add planned Express auth and host auth examples"
- **Toolchain:** `go1.26.1 linux/amd64`, `GOFLAGS=-buildvcs=false`
- **Diff scale (local `git diff origin/main...HEAD`):** 186 files / +28570 / −119. Note: a large fraction of the line volume is `ttmp/` ticket history (design docs + diaries), **not** product code. Product code is concentrated in `pkg/gojahttp`, `pkg/gojahttp/auth/...`, `modules/express`, `pkg/xgoja/hostauth`, `pkg/xgoja/providers/http`, the four `examples/xgoja/1[89]-*`, `20-*`, `21-*` dirs, and `pkg/doc/2[9]-*`/`3[01]-*`. This review focuses on product code.

### Commands run (this review)

```bash
# Static
GOFLAGS=-buildvcs=false go vet ./pkg/gojahttp/... ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http   # clean (exit 0)

# Targeted tests (all PASS)
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp/auth/... -count=1

# Example smoke (PASS, exercises 401/403/404/200/204 denial + success paths)
GOFLAGS=-buildvcs=false go run ./examples/xgoja/18-express-auth-host/cmd/host -smoke

# Reviewer verification harness (ticket-local, does not modify product code)
GOFLAGS=-buildvcs=false go run ./ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/03-verify-behaviors.go
```

### Smokes run / skipped

- ✅ `examples/xgoja/18-express-auth-host` in-process smoke (PASS).
- ⏭️ `examples/xgoja/21-generated-host-auth` smoke — skipped by choice; its Makefile regenerates committed runtime files. Recommend author/reviewer run from a clean worktree and confirm `git status --short` is empty afterward.
- ⏭️ `examples/xgoja/19-express-keycloak-auth-host` Keycloak/Postgres smoke — skipped; requires Docker + free ports. Reviewed statically instead (see Phase 6).

### Triage note

The PR bundles a large amount of `ttmp/` planning history. That is harmless to the build (it is not Go code), but it inflates the diff and makes the GitHub file list hard to read. **Recommendation:** keep or split per repo convention, but do not let the doc volume obscure the product review below. This is not a code defect.

## Architecture summary (confirmed)

PR 74 implements a host-owned planned-auth model with a clean trust boundary. JavaScript declares route **intent**; Go owns every enforcement step. The intended model from the design articles is realized in the code:

1. `modules/express` fluent builders (`auth_builders.go`) stage a route declaration and only call `Host.RegisterPlanned` after an explicit `.public()` or `.auth(...).allow(...)` choice.
2. Builders compile intent into a `pkg/gojahttp.RoutePlan` (`auth_plan.go`), which `ValidateRoutePlan` normalizes and rejects if incomplete (missing security mode, missing `.allow(action)` for user routes, resources referencing undeclared `:param`s).
3. `pkg/gojahttp.Host.ServeHTTP` → `servePlannedRoute` → `buildSecureEnvelope` (`planned_dispatch.go`) authenticates, verifies CSRF, resolves resources, authorizes, and audits **before** the JavaScript handler runs.
4. Auth helper packages under `pkg/gojahttp/auth/...` (`sessionauth`, `audit`, `capability`, `appauth`, `devauth`, `keycloakauth`) implement the host-owned interfaces.
5. `pkg/xgoja/hostauth` builds concrete services from config lazily (factory discovered at command construction; stores opened at execution time).
6. `pkg/xgoja/providers/http/serve.go` wires generated-host auth services into the `serve` and hot-reload lifecycles.

The central review question — *does PR 74 create a clear, fail-closed, testable path from JS route declarations to Go-owned enforcement, without unsafe lifecycle/persistence/generated-host behavior?* — is answered **yes**, with the caveats in the findings below.

## Blocking issues

**None.** No issue found rises to a blocking/security-bypass level. The fail-closed invariants all hold (see "Security notes"). The findings below are non-blocking or nits.

## Non-blocking issues

### F1 — Audit `RedactMap` over-redacts, defeating capability/session correlation

- **Location:** `pkg/gojahttp/auth/audit/audit.go`, `secretKey()` (substring match list includes `"capability"`, `"session"`, `"token"`, `"code"`).
- **Evidence:** Verification harness `scripts/03-verify-behaviors.go`, behavior C. An audit event with attributes `{capabilityId: "cap_123", purpose: "invite", subjectId: "u1", sessionId: "sess_xyz"}` normalizes to `{capabilityId: "[REDACTED]", sessionId: "[REDACTED]", purpose: "invite", subjectId: "u1"}`.
- **Why it matters:** `capability.go` `Service.record()` explicitly puts `capabilityId` (the database **id**, not the raw token) into audit attributes so operators can correlate which capability was issued/redeemed/revoked. The substring redaction erases exactly that identifier, and also erases any `*Id`/`*SessionId` attribute that merely *mentions* a sensitive concept but is itself non-secret. The raw token is never stored (it is hashed — see D), so there is no leak risk from these identifiers; the redaction removes operational value without adding safety.
- **Expected:** Identify real secrets (raw tokens, passwords, cookie *values*, Authorization *header values*) without nuking identifiers.
- **Suggested fix:** Move from substring-on-key to either (a) a precise deny-list of exact/normalized secret key names (`password`, `secret`, `token`, `authorization`, `cookie`, `refresh_token`, …) **plus** an explicit opt-in `redact` marker on fields, or (b) keep substring matching but exempt keys ending in `Id`/`Type`/`Name`/`Count` and document the rule. At minimum, special-case `capabilityId` since the capability package relies on it.
- **Severity:** non-blocking, but should be fixed before audit logs are relied on for forensics.
- **Test to add:** assert `RedactMap` keeps `capabilityId`, redacts `token`/`password`/`cookie`.

### F2 — `appauth` SQL `UpsertFromOIDC` has a SELECT-then-plain-INSERT race

- **Location:** `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go`, `UpsertFromOIDC`.
- **Evidence:** Schema (`schema.go`) declares `keycloak_sub TEXT UNIQUE` for both SQLite and Postgres. `UpsertFromOIDC` does `scanUser(... userBySubQuery ...)`; on `ErrNotFound` it runs `insertUserQuery()` (a plain `INSERT`, **not** the upsert), inside a transaction. Under the default `READ COMMITTED` isolation, two concurrent OIDC callbacks for the *same brand-new* `keycloak_sub` both observe "not found", both `INSERT`, and the loser surfaces a raw unique-constraint violation instead of a clean upsert.
- **Why it matters:** This is the Keycloak first-login path. The race is low-likelihood (two simultaneous first logins for the exact same subject) but the failure mode is a 500 to the user and a confusing error in the OIDC callback, when an idempotent upsert would have succeeded. The update path and the fixture `AddUser` already use the proper `ON CONFLICT ... DO UPDATE` upsert; only the OIDC create path is inconsistent.
- **Expected:** First login for a subject is idempotent under concurrency.
- **Suggested fix:** Use `upsertUserQuery()` (`ON CONFLICT(id) DO UPDATE ...`) for the create branch as well, or switch to `INSERT ... ON CONFLICT(keycloak_sub) DO NOTHING RETURNING ...` / `DO UPDATE` keyed on `keycloak_sub`, then re-read. Keep the transaction.
- **Severity:** non-blocking (concurrency edge case), worth fixing because it is in the security-critical login path and the fix is small and consistent with the rest of the file.
- **Test to add:** two goroutines calling `UpsertFromOIDC` with the same new `sub` concurrently; assert both succeed or one returns the created row without an unwrapped driver error.

### F3 — `keycloakauth` package doc is inaccurate about token retention

- **Location:** `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`, package comment: *"It keeps Idp tokens server-side and creates an opaque application session."*
- **Evidence:** Reading `handleCallback`: after `oauth2Config.Exchange(...)`, only the `id_token` is verified and its claims normalized; the returned access/refresh tokens are **never stored anywhere** (not server-side, not in the session). The opaque cookie holds only the app session id (`sessionauth`). The actual behavior is stricter/better than documented — zero IdP token retention.
- **Why it matters:** A reviewer/operator reading the doc may assume tokens are persisted and look for (or implement) token refresh/storage that does not exist. For a security-sensitive package, the doc should describe the real (good) behavior.
- **Expected:** Doc states that IdP tokens are used ephemerally to establish the app session and are then discarded; only the opaque app session is persisted.
- **Suggested fix:** Rephrase to e.g. *"It exchanges the authorization code for OIDC tokens, verifies the ID token, derives an opaque application session, and discards the raw IdP tokens; only the app session id is stored in the cookie."*
- **Severity:** nit (documentation accuracy in a security package).

## Nits

### N1 — No dispatch-level test for the "authenticator returns `(nil actor, nil error)`" contract

- **Location:** `pkg/gojahttp/planned_dispatch.go` (the `if actor == nil { return ..., http.StatusUnauthorized, ErrUnauthenticated }` branch) vs `pkg/gojahttp/planned_dispatch_test.go`.
- **Evidence:** The implementation **correctly** maps `(nil, nil)` → 401 (verified by `scripts/03-verify-behaviors.go` behavior A: `status=401 body="Unauthorized"`). However `TestPlannedUserRouteReturns401WhenUnauthenticated` only exercises the `ErrUnauthenticated` *error* path, not the success-but-nil-actor path. The methodology checklist explicitly lists "authenticator returning nil actor returns 401" as an expected negative test.
- **Why it matters:** This is a bug-prone interface contract. A future refactor could drop the `actor == nil` guard; a test pins it. The behavior is correct today; this is purely a coverage gap.
- **Suggested fix:** Add a dispatch test that registers an `Authenticator` returning `(nil, nil)` and asserts 401 + handler-not-invoked.
- **Severity:** nit (test coverage).

### N2 — `devauth` is correctly dev-only but ships weak defaults

- **Location:** `pkg/gojahttp/auth/devauth/devauth.go` (`DefaultUsername`/`DefaultPassword`, `secureCookie` defaults to `false`).
- **Evidence:** Package doc and all call sites mark it dev/example-only. Example 18 (`18-express-auth-host`) uses it behind an in-process smoke and a manual server with `Dev: true`. Constant-time credential compare is used (`constantTimeEqual`).
- **Why it matters:** Not a defect — it is honestly scoped. Flagging only so docs/READMEs make the non-production status unmissable (they largely do). Consider a startup log line when a `devauth.Kit` is constructed in non-`Dev` mode, as defense-in-depth against accidental production wiring.
- **Severity:** nit (defense-in-depth / docs).

### N3 — `devauth.sessionFromRequest` uses `time.Now()` directly

- **Location:** `pkg/gojahttp/auth/devauth/devauth.go`, `sessionFromRequest`.
- **Evidence:** Unlike `sessionauth.Manager` (which accepts an injectable `Now`), `devauth` checks `time.Now().After(session.ExpiresAt)` with the wall clock, which makes expiry-related dev behavior harder to test deterministically.
- **Severity:** nit (testability), dev-only package.

## Security notes (Phase 3 + Phase 6)

All ten invariants from the methodology were checked. Results:

| # | Invariant | Result | Evidence |
|---|---|---|---|
| 1 | Security mode required before `.handle()` | ✅ Holds | `auth_builders.go` staged state machine; `.handle()` only on `needsHandlerObject()` which follows `.public()`/`.allow()`. `ValidateRoutePlan` rejects empty mode (`auth_plan.go`). |
| 2 | Protected route requires `.allow(action)` | ✅ Holds | `ValidateRoutePlan` returns error for `SecurityModeUser` + empty action. |
| 3 | Trusted builder objects only (no spoofing) | ✅ Holds | `authSpecs`/`resourceSpecs` `sync.Map` keyed by `*goja.Object`; `.auth({required:true})` and `.resource({type:...})` rejected. Spec is **value-copied** on `.auth()` (`return *spec, nil`), so later JS mutation of the builder does not retroactively change a registered plan (copy-on-auth). |
| 4 | No handler before security choice | ✅ Holds | `needsSecurityObject()` exposes no `.handle()`. |
| 5 | CSRF before mutation handler | ✅ Holds | `buildSecureEnvelope` runs CSRF *before* resource resolution and authorization. Verified by `TestPlannedRouteVerifiesCSRFBeforeHandler` and `TestPlannedRouteCSRFErrorBlocksHandler` (403, handler not run). |
| 6 | Resource resolver required for resource routes | ✅ Holds | `if h.auth.Resources == nil { return ..., 500 }` before any resource lookup. |
| 7 | Authorizer required for protected action routes | ✅ Holds | `if h.auth.Authorizer == nil { return ..., 500 }`. |
| 8 | Audit is host-owned | ✅ Holds | JS declares `plan.Audit.Event`; Go emits allowed/denied/completed/failed outcomes in `recordAudit`. Best-effort (`_ = RecordAudit(...)`), cannot change a response. |
| 9 | Raw routes can be rejected | ✅ Holds | `RejectRawRoutes` rejects matched unplanned routes while planned routes + static mounts still work. `http.go` defaults `reject-raw-routes: true`. |
| 10 | Error details gated by dev mode | ✅ Holds | Verified: `dev=true` leaks `err.Error()` for 5xx; `dev=false` returns `http.StatusText(500)`. Generated-host `dev-errors` defaults to `false`. |

**Denial-path status table (traced in `buildSecureEnvelope` / `servePlannedRoute`):**

| Condition | Status | Audit | Handler |
|---|---|---|---|
| Missing plan (nil) | 500 | denied | no |
| Missing authenticator | 500 | denied | no |
| Authentication failure (sentinel) | 401/403/404/500 per `statusForAuthError` | denied | no |
| Authenticator returns nil actor, nil error | **401** | denied | no |
| Missing CSRF protector | 500 | denied | no |
| CSRF failure | 403 | denied | no |
| Missing resource resolver | 500 | denied | no |
| Resource not found (`ErrNotFound`) | 404 | denied | no |
| Missing authorizer | 500 | denied | no |
| Authorizer denied | 403 | denied | no |
| Handler error | 500 | failed | started then aborted |
| Handler success | handler status | completed | yes |

**Session/CSRF (`sessionauth`):** secure-by-default cookie (`__Host-app`, HttpOnly, `Secure` unless `AllowInsecureHTTP`, SameSite Lax default), 32-byte `crypto/rand` session ids and CSRF tokens, constant-time CSRF compare (`subtle.ConstantTimeCompare`), idle + absolute expiry, rotation deletes old id, revoked/expired sessions rejected everywhere, MFA-freshness uses trusted `session.MFAAt`. Memory store deep-copies sessions (claims/tenantIDs/timestamps) on read and write.

**Capability tokens:** raw token returned exactly once from `Issue`; stored only as SHA-256 hash (`HashToken`); `IssueResult.Capability` has `TokenHash` redacted; purpose checked on redeem; expired/revoked/used rejected; **single-use redemption is atomic in SQL** (`UPDATE ... SET used_at=? WHERE id=? AND used_at IS NULL` inside a transaction + `requireAffected`; row locking makes concurrent double-redeem return `ErrUsed`). Memory store uses constant-time hash compare.

**OIDC (`keycloakauth`):** state/nonce/PKCE verifier are 32-byte `crypto/rand`; state is **single-use** (`MemoryTransactionStore.Take` deletes before use) with a TTL; PKCE S256 challenge; ID token verified by `go-oidc` `IDTokenVerifier` (issuer, audience=client id, expiry, signature); **nonce bound to the transaction and checked** after verification; `return_to` open-redirect guard rejects `//`-prefixed and non-`/`-rooted targets; browser receives only the opaque app session cookie. Logout revokes the session and clears the cookie.

**Audit (`audit`):** `Record` deliberately excludes cookies, Authorization header, session ids, and raw tokens from the stored shape; only `RequestID`/`UserAgent`/hashed IP are derived from the request. (The over-redaction defect F1 is about *attributes*, not the core record shape.) `LogSink` is intentionally even more minimal.

## Lifecycle notes (Phase 4 + Phase 9)

Generated-host auth service construction is correct and matches the design claim ("discover at construction, open at execution"):

- `newServeCommandSet` calls `hostauth.LookupServiceFactory` during command construction **only** for type validation/discovery (`lookup.go` checks interface + nil via reflect). No DB opens.
- `buildServeAuthServices` (serve.go) calls `factory.BuildHostAuthServices` only inside `serveVerb`/`serveVerbHotReload`, i.e. at execution time, **after** Glazed values are parsed. `ResolveConfig` performs env lookups (`os.LookupEnv`) here, not earlier.
- `BuildStores` shares `*sql.DB` handles for identical `(driver, dsn)` pairs (`storeBuilder.dbs` map) and registers a single closer per unique handle; `closeAll` closes in LIFO order with `errors.Join`. Verified by `TestServiceFactoryUsesEnvLookupAtBuildTime` (asserts `len(services.Closers) == 1` for a shared sqlite).
- Failure-path cleanup: `BuildHostAuthServices` defers `stores.Close` while `success == false`, so a partially-built bundle is closed.
- `serveVerb` defers `authServices.Close` and `rt.Close`; hot reload defers `manager.Close`.
- Hot reload **shares** auth services across candidate reloads (built once, passed into the `Load` closure per candidate as `serveRuntimeServices(candidate.Host, authServices, false, true)`), so session state is stable across reloads and stores are closed exactly once on exit. This is the desired behavior for blue/green reload.
- External host ownership is preserved: when an external host is supplied, `includeGeneratedHost=false` and `ownsListen` is respected (`serveRuntimeServices`).
- `RejectRawRoutes` propagates through generated-host hosts via `hostOptionsWithAuth` → `HostOptions.RejectRawRoutes`, and `http.go` defaults it to `true`.

**`sql.Open` does not connect immediately** (noted in the methodology as an open question). Schema application (`ApplySchema`) is the first real I/O and will surface a bad/unreachable DSN there when `apply-schema: true`. This is acceptable and explicit; just be aware that a misconfigured `dsn` with `apply-schema: false` will only fail on first query, not at startup. Recommend documenting that operators who want fail-fast should enable `apply-schema`.

## Store / persistence notes (Phase 5)

- Memory stores (`sessionauth`, `appauth`, `audit`, `capability`) deep-copy mutable maps/slices on input and output. ✅
- SQL stores round-trip nullable fields (`sql.NullString`/`NullTime`), empty slices (normalized to `[]`/`{}` JSON), and timestamps consistently across SQLite and Postgres. ✅
- Schemas are idempotent (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`). ✅
- Dialect selection is explicit per store; placeholder syntax is branched correctly (`?` vs `$n`). ✅
- Transactions used where atomicity matters: session rotation (`Rotate`), capability redemption (`Redeem`), OIDC user upsert (`UpsertFromOIDC`). ✅ (subject to F2)
- `closeAll` uses `errors.Join` so one failing close does not swallow others. ✅
- Shared DB handles closed exactly once (see Lifecycle). ✅

## Documentation and migration notes (Phase 8)

- Migration doc `pkg/doc/30-migrate-...` correctly states the legacy `app.get(path, handler)` overload is **removed**, shows explicit `.public()` for public routes and `.auth(...).allow(...)` before `.handle(...)` for protected routes, and explains the staged builder states. Aligned with implementation.
- User guide `pkg/doc/29-express-auth-user-guide.md` matches the builder API and the security-first staging. Aligned.
- `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` and `17-xgoja-v2-reference.md` cover the generated-host auth config and host-service bag; spot-checked consistent with `hostauth.Config` / `ServicesKey`.
- `reject-raw-routes` default `true` is documented in the `http` config field help. ✅
- Example READMEs distinguish dev-only (`18`, `20`) from production-shaped (`19` Keycloak/Postgres) and generated-host (`21`). ✅
- TypeScript declarations (`modules/express/typescript.go`) were updated alongside the verb-helper cutover. (Not deeply diffed in this pass; recommend a focused check that the `.d.ts` matches the staged builder method set.)

## Test coverage notes

**Well covered:**
- Plan validation (`auth_plan_test.go`): public/user/resource-param/missing-action/normalization.
- Dispatch denial paths (`planned_dispatch_test.go`): raw-route rejection, public dispatch, auth+authz, 401, CSRF-before-handler, CSRF `ALL`-method semantics, CSRF error→403, audit allowed/denied/completed, resource resolution+authz, 404 mapping, authorizer-error→500.
- Builder staging (`auth_builders_integration_test.go`): public/auth/resource/csrf/audit builders, plain-object rejection, legacy-overload rejection, handle-before-security rejection.
- Hostauth lifecycle (`builder_test.go`, `stores_test.go`, `lookup_test.go`, `resolve_test.go`): mode none/dev, env-deferred DSN, shared DB handles, config inheritance, `ApplySchema` pointer semantics.
- Serve lifecycle (`serve_test.go`): malformed factory rejection, factory integration, external-host preservation, hot reload auth integration, HTTP-settings preservation.

**Gaps to consider:**
- N1: `(nil actor, nil error)` dispatch branch has no direct test.
- F1: no test asserting `RedactMap` keeps identifiers while redacting secrets.
- F2: no concurrency test for `UpsertFromOIDC`.
- No dispatch-level test for a handler that returns a **rejected** promise (the `awaitAndFinishPromise` reject path) — recommend adding one.
- Keycloak/Postgres end-to-end smoke not run in this environment (static review only).

## Merge recommendation

**Approve with non-blocking follow-ups.** No blocking correctness or security-bypass issue was found. The fail-closed invariants, trusted-builder boundary, session/CSRF/capability/OIDC security properties, and generated-host lifecycle all hold and are well tested. The diff is large but the product code is focused and the doc/test quality is high.

Recommended pre-or-post-merge follow-ups, in priority order:
1. **F1** — tighten audit `RedactMap` so `capabilityId`/`*Id` identifiers survive while real secrets are still redacted (forensics value).
2. **F2** — make `appauth` SQL `UpsertFromOIDC` idempotent under concurrency (use the upsert query for the create path).
3. **N1** — add the `(nil,nil)`-authenticator dispatch test.
4. **F3** — correct the `keycloakauth` package doc to describe actual token handling.
5. Run the generated-host (`21`) smoke from a clean worktree and the Keycloak (`19`) Docker smoke before/around merge to close the two skipped validations.

## References

- Methodology: `design/01-pr-74-code-review-methodology-and-intern-guide.md`
- Verification harness: `scripts/03-verify-behaviors.go`
- Inventory: `sources/01-pr74-inventory.md`
- PR: https://github.com/go-go-golems/go-go-goja/pull/74
