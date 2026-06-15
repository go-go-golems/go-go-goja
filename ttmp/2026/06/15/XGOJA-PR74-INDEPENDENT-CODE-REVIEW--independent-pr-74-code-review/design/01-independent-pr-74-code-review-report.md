---
Title: Independent PR 74 Code Review Report
Ticket: XGOJA-PR74-INDEPENDENT-CODE-REVIEW
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/doc/17-xgoja-v2-reference.md
      Note: |-
        Documentation/API drift review for removed Express handler overload.
        Stale removed Express overload documentation finding
    - Path: modules/express/auth_builders.go
      Note: |-
        JavaScript planned-route builder API and trusted object boundary review.
        Planned route builder API review
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: |-
        Session and CSRF enforcement review; contains the main blocking CSRF finding.
        CSRF/session enforcement blocking finding
    - Path: pkg/gojahttp/planned_dispatch.go
      Note: |-
        Planned-route secure context construction and audit flow review.
        Secure context and audit flow review
    - Path: pkg/xgoja/hostauth/resolve.go
      Note: |-
        Generated-host auth configuration resolution review.
        Generated-host auth config behavior finding
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/pull/74
Summary: Independent review of PR 74's planned Express auth and generated-host auth work. Targeted and full Go tests pass, but the review found one blocking CSRF hardening issue and several important follow-ups around immutable secure-context data, disabled-auth config behavior, docs drift, and session rotation contract tests.
LastUpdated: 2026-06-15T16:45:00-04:00
WhatFor: Use this as an independent code review report for PR 74, separate from the shared methodology ticket and colleague diary work.
WhenToUse: Use before approving or merging PR 74, especially to validate the CSRF/session hardening fixes and documentation updates.
---


# Independent PR 74 Code Review Report

## Scope and environment

- **Repository:** `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja`
- **Branch:** `task/goja-express-auth`
- **Base:** `origin/main` at `d406577f97866c816a4bd0fd0d2c5284143c2cc0`
- **Head:** `b66baea869583d79db6b0e8ec5007e0fad0e5ef7`
- **Diff size:** `186 files changed, 28570 insertions(+), 119 deletions(-)`
- **Go version observed:** `go1.26.1 linux/amd64`
- **Review isolation:** I did not read the existing ticket's diary, sources, or colleague work. I used the requested methodology guide as the starting instructions and generated new scripts/evidence under this ticket only.

## Commands and validation run

Evidence files created under this ticket:

- `scripts/01-independent-inventory.sh` -> `sources/01-independent-inventory.md`
- `scripts/02-independent-validation.sh` -> `sources/02-independent-validation.md`
- `scripts/03-independent-static-probes.sh` -> `sources/03-independent-static-probes.md`
- `scripts/04-probe-goja-map-mutation/main.go` -> `sources/07-goja-map-mutation-probe.md`
- `scripts/05-probe-csrf-empty-token/main.go` -> `sources/08-csrf-empty-token-probe.md`
- `scripts/06-probe-mode-none-store-validation/main.go` -> `sources/09-mode-none-store-validation-probe.md`

Validation results:

```text
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp/auth/... -count=1
GOFLAGS=-buildvcs=false go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/20-express-hello-world/cmd/host ./examples/xgoja/21-generated-host-auth/cmd/host -count=1
make -C examples/xgoja/18-express-auth-host smoke
go vet ./pkg/gojahttp/... ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http
GOFLAGS=-buildvcs=false go test ./... -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

All commands above passed. The generated-host auth smoke regenerated `examples/xgoja/21-generated-host-auth/internal/xgojaruntime` and left that example subtree clean afterward.

I did not run the Keycloak Docker smoke. Docker is available on the host, but the review already produced a blocking CSRF finding; the Keycloak smoke should still be run after fixes if PR 74 is close to merge.

## Architecture summary

PR 74 introduces a coherent planned-route security model: JavaScript declares route intent through `modules/express`, the builder compiles that intent into `gojahttp.RoutePlan`, and `gojahttp.Host` performs authentication, CSRF verification, resource resolution, authorization, and audit before invoking a JavaScript handler. The generated-host side adds `pkg/xgoja/hostauth` for lazy auth service construction and wires those services into the `http serve` provider.

The broad direction is sound. The staged builder API prevents accidental raw `app.get(path, handler)` registration, trusted Go-backed builder objects prevent plain JavaScript object spoofing for `.auth()` and `.resource()`, and the main planned-dispatch path generally fails closed when a host dependency is missing. The remaining issues are mostly hardening and consistency problems around edge cases that are easy to miss in normal happy-path tests.

## Merge recommendation

**Request changes before merge.**

The main reason is the CSRF empty-token edge case in `sessionauth.Manager.VerifyCSRF`: if a persisted session row has an empty CSRF token, a request with no `X-CSRF-Token` header is accepted. Manager-created sessions generate non-empty tokens, but the store/schema currently allow malformed sessions, and the verifier should fail closed even with corrupted, migrated, manually inserted, or buggy store data.

## Blocking issues

### 1. Empty persisted CSRF tokens make missing CSRF headers pass

**Severity:** blocking / security hardening

**Problem:** `VerifyCSRF` only checks constant-time equality between the request header and stored session token. In Go, `subtle.ConstantTimeCompare([]byte(""), []byte("")) == 1`, so a malformed session with `CSRFToken == ""` accepts a request that omits `X-CSRF-Token`.

**Where to look:**

- `pkg/gojahttp/auth/sessionauth/sessionauth.go:222-234`
- `pkg/gojahttp/auth/sessionauth/sessionauth.go:358-365`
- `pkg/gojahttp/auth/sessionauth/sqlstore/schema.go:5-17` and `:27-39`
- Probe: `sources/08-csrf-empty-token-probe.md`

**Example:**

```go
func (m *Manager) VerifyCSRF(ctx context.Context, req gojahttp.CSRFRequest) error {
    session, err := m.SessionFromRequest(ctx, req.HTTPRequest)
    if err != nil {
        return authError(err)
    }
    if req.Actor != nil && req.Actor.ID != session.UserID {
        return errors.New("session actor mismatch")
    }
    if !constantTimeEqual(req.HTTPRequest.Header.Get(CSRFHeaderName), session.CSRFToken) {
        return errors.New("missing or invalid X-CSRF-Token")
    }
    return nil
}
```

The independent probe output was:

```text
VerifyCSRF accepted missing header when stored csrf token is empty
```

**Why it matters:** CSRF checks must fail closed against malformed persisted state. This bug is not triggered by `Manager.NewSession`, but it can be triggered by direct store use, manual SQL repair/import, a future migration, a buggy test fixture promoted into an example, or a compromised writer that can create invalid rows but not forge a real CSRF token. The SQL schema uses `csrf_token TEXT NOT NULL`, but `NOT NULL` still permits `''`.

**Suggested fix:**

```go
func constantTimeTokenEqual(header, stored string) bool {
    if header == "" || stored == "" {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(header), []byte(stored)) == 1
}

func validateSession(session *Session, now time.Time) error {
    if session.UserID == "" || session.CSRFToken == "" {
        return ErrInvalidCookie
    }
    // existing revoked/expiry checks...
}
```

Also validate `Session` in `MemoryStore.Create`, SQL `Store.Create`/`insert`, and `Rotate`, and add schema `CHECK (csrf_token <> '')` for newly created SQL tables if migrations permit it.

**Regression tests needed:**

- `VerifyCSRF` rejects empty stored token + missing header.
- `VerifyCSRF` rejects empty stored token + non-empty header.
- `Store.Create` rejects empty `CSRFToken` for memory and SQL stores, or `SessionFromRequest` treats such sessions as invalid before CSRF comparison.

## Important non-blocking issues

### 2. Planned-handler context exposes mutable Go-owned maps and slices to JavaScript

**Severity:** important non-blocking / security-boundary hardening

**Problem:** `secureEnvelope.JSObject` builds JavaScript-visible objects using Go maps/slices from `Actor.Claims`, `Actor.TenantIDs`, and `ResourceRef.Claims`. Goja-backed objects can mutate the original Go map and slice. That means a JavaScript handler can modify data that the host considers host-owned security context.

**Where to look:**

- `pkg/gojahttp/planned_dispatch.go:258-314`
- `pkg/gojahttp/auth/appauth/appauth.go:340-347`
- Probe: `sources/07-goja-map-mutation-probe.md`

**Example:**

```go
func actorJSMap(actor *Actor) map[string]any {
    if actor == nil {
        return nil
    }
    return map[string]any{
        "id":        actor.ID,
        "kind":      actor.Kind,
        "tenantIds": actor.TenantIDs,
        "claims":    actor.Claims,
    }
}
```

The independent Goja probe showed that JavaScript assignments mutate the original Go data:

```json
{
  "claims": {
    "extra": "new",
    "nested": { "level": "two" },
    "role": "mutated"
  },
  "tenantIDs": ["mutated-tenant", "o2"]
}
```

**Why it matters:** The authorization decision has already happened before the handler runs, so this is not an immediate auth bypass in the current dispatch path. It is still the wrong default for a security envelope. It weakens the boundary between host-owned identity/resource facts and JavaScript-owned response logic, and it can corrupt later audit enrichment or future hooks that reuse `envelope.Actor` or `envelope.Resources` after handler execution.

**Suggested fix:** Deep-copy host-owned values before exposing them to Goja. Consider freezing the JS objects if handlers are only supposed to read them.

```go
func actorJSMap(actor *Actor) map[string]any {
    if actor == nil { return nil }
    return map[string]any{
        "id": actor.ID,
        "kind": actor.Kind,
        "tenantIds": append([]string(nil), actor.TenantIDs...),
        "claims": deepCloneAnyMap(actor.Claims),
    }
}

func resourceRefJSMap(resource *ResourceRef) map[string]any {
    if resource == nil { return nil }
    return map[string]any{
        "name": resource.Name,
        "type": resource.Type,
        "id": resource.ID,
        "tenantId": resource.TenantID,
        "claims": deepCloneAnyMap(resource.Claims),
    }
}
```

**Regression tests needed:**

- A planned handler mutates `ctx.actor.claims`, `ctx.actor.tenantIds`, and `ctx.resources.<name>.claims`; subsequent host-side actor/resource structs remain unchanged.
- If immutability is desired, assert that `Object.isFrozen(ctx.actor)` or mutation attempts are rejected/ignored.

### 3. `auth.mode=none` still validates store DSNs and env vars

**Severity:** important non-blocking / generated-host usability

**Problem:** `ResolveConfig` parses `mode`, but even when `mode == none`, it continues resolving session and all store configuration. A disabled-auth config can therefore fail because a Postgres DSN or env var is absent, even though `BuildHostAuthServices` will return no auth options and should not open stores.

**Where to look:**

- `pkg/xgoja/hostauth/resolve.go:39-56`
- `pkg/xgoja/hostauth/resolve.go:136-158`
- Probe: `sources/09-mode-none-store-validation-probe.md`

**Example:**

```go
func ResolveConfig(cfg Config, opts ResolveOptions) (ResolvedConfig, error) {
    mode, err := parseMode(cfg.Mode)
    if err != nil {
        return ResolvedConfig{}, configError("auth.mode", err)
    }
    if mode == ModeOIDC {
        return ResolvedConfig{}, configError("auth.mode", ErrOIDCNotImplemented)
    }

    session, err := resolveSessionConfig(cfg.Session)
    // stores are resolved even when mode == ModeNone
    stores, err := resolveStoresConfig(cfg.Stores, opts.LookupEnv)
    // ...
}
```

Probe output:

```text
mode=none still validates store config and failed: auth.stores.session.dsn: dsn or dsn-env is required for non-memory stores
```

**Why it matters:** This makes it harder to keep one config template across environments where auth may be disabled locally or in tests. More importantly, it violates the lazy/no-open mental model: disabled auth should not require auth persistence configuration to be valid.

**Suggested fix:** Return early after mode parsing for `ModeNone`, or explicitly document that store config is validated even when auth is disabled. I recommend early return:

```go
if mode == ModeNone {
    session, err := resolveSessionConfig(cfg.Session) // optional, only if callers need cookie defaults
    if err != nil { return ResolvedConfig{}, err }
    return ResolvedConfig{Mode: ModeNone, Session: session}, nil
}
```

If callers do not need resolved session defaults in none mode, skip session resolution too.

**Regression test needed:** `ResolveConfig(Config{Mode: ModeNone, Stores: postgres-without-dsn})` should not fail if the chosen contract is that disabled auth ignores stores.

### 4. The xgoja v2 reference still teaches the removed `app.get(path, handler)` overload

**Severity:** important non-blocking / docs and migration contract

**Problem:** PR 74 intentionally removes the two-argument Express route overload, and `modules/express/express.go` panics with a migration-oriented error if `app.get(pattern, handler)` is used. The main xgoja v2 reference still includes TypeScript snippets and route-pattern bullets using the removed form.

**Where to look:**

- `cmd/xgoja/doc/17-xgoja-v2-reference.md:512-519`
- `cmd/xgoja/doc/17-xgoja-v2-reference.md:528-539`
- `cmd/xgoja/doc/17-xgoja-v2-reference.md:547-551`
- Runtime enforcement: `modules/express/express.go:184-194`

**Example:**

```ts
export function demo() {
  const app = express.app()
  app.get("/", (_req, res) => res.send(message()))
}
```

**Why it matters:** This PR is a breaking API change. The docs are the migration contract, and stale docs will cause users to copy code that fails at runtime. This is especially likely because `17-xgoja-v2-reference.md` is a broad reference page, not a niche auth page.

**Suggested fix:** Update all reference snippets to planned-route syntax:

```ts
app.get("/").public().handle((_ctx, res) => res.send(message()))
app.get("/healthz").public().handle((_ctx, res) => res.send("ok"))
```

For route pattern bullets, use builder syntax too:

```md
- `app.get("/users/:id").public().handle(...)` captures one segment as `ctx.params.id`.
```

**Regression check:** Run `rg -n 'app\.(get|post|put|patch|delete|all)\([^\n]*,' pkg/doc cmd/xgoja/doc examples -S` and review every hit as either an explicitly labeled old-code migration example or a stale snippet.

### 5. Session rotation does not prove the old session existed before creating the replacement

**Severity:** non-blocking / session-store contract hardening

**Problem:** Both memory and SQL rotation paths allow `Rotate(oldID, next)` to create `next` even if `oldID` does not exist. The memory store simply deletes `oldID` and inserts `next`; the SQL store deletes without checking `RowsAffected`, then inserts.

**Where to look:**

- `pkg/gojahttp/auth/sessionauth/sessionauth.go:392-400`
- `pkg/gojahttp/auth/sessionauth/sqlstore/sqlstore.go:96-112`
- Contract gap: `pkg/gojahttp/auth/internal/sessionauthtest/store_contract.go:90-113`

**Example:**

```go
func (s *MemoryStore) Rotate(_ context.Context, oldID string, next Session) error {
    if next.ID == "" {
        return fmt.Errorf("next session id is required")
    }
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.sessions, oldID)
    s.sessions[next.ID] = cloneSession(next)
    return nil
}
```

**Why it matters:** Rotation is usually used to prevent session fixation or to move a session through a higher-assurance state. A rotate operation that can create a replacement from a missing old session is closer to `Create`. That may be fine if it is deliberate, but the method name and contract imply old-session replacement.

**Suggested fix:** Decide the contract explicitly. I recommend `Rotate` return `ErrInvalidCookie` when the old session is absent, and only insert `next` after proving the old row existed. SQL should check `RowsAffected` on the delete/update. Memory should check map membership before delete.

**Regression tests needed:** Add a shared store-contract test for `Rotate("missing", next)` returning `ErrInvalidCookie` and not creating `next`.

## Security notes

- The staged Express builder is a strong improvement over raw route registration. `.handle(...)` is unavailable until `.public()` or `.auth(...).allow(...)` reaches the handler stage, and plain JavaScript objects are rejected by `.auth(...)` and `.resource(...)`.
- `gojahttp.Host` generally fails closed for missing authenticator, CSRF protector, resource resolver, and authorizer in planned routes.
- Audit sinks intentionally ignore audit write errors. That is acceptable for best-effort audit, but production deployments that require strict audit should get an explicit strict mode rather than changing the default silently.
- `keycloakauth` correctly uses state, nonce, PKCE verifier, OIDC verifier, and server-side app sessions. Follow-ups worth considering: memory transaction cleanup for abandoned logins and whether GET logout should remain enabled without CSRF in production-shaped examples.
- Store packages usually clone top-level maps/slices on input/output. Several clone helpers are shallow for nested `map[string]any`, so nested claim structures can still be shared. This is the same family of issue as the planned-context mutability finding.

## Test coverage notes

What is well covered:

- Planned-route registration and validation.
- Builder staging and trusted builder object checks.
- CSRF dispatch order and denial behavior in planned routes.
- Session expiration/revocation/rotation happy paths.
- Audit redaction of secret-looking attributes.
- Memory and SQLite-backed store contracts for appauth/audit/capability/sessionauth.
- Generated-host service construction and HTTP serve/hot-reload integration.

Coverage gaps to close before merge or immediately after:

1. Empty persisted CSRF token must fail CSRF verification.
2. Planned `ctx.actor`/`ctx.resources` mutation must not mutate Go-owned actor/resource structures.
3. `auth.mode=none` should have an explicit test for whether stores are ignored or validated.
4. Session `Rotate` should test missing old session behavior.
5. Docs should be grep-checked for stale two-argument Express route snippets outside explicit migration-before examples.
6. Run the Keycloak Docker smoke after security fixes, because it exercises the production-shaped OIDC/session/appauth/capability path.

## Documentation and migration notes

The dedicated auth docs look mostly aligned with the planned-route API. The migration guide intentionally includes old snippets as before/after examples and explains the removed overload well. The stale xgoja v2 reference snippets are the main doc issue found in this pass.

The generated-host docs correctly state that OIDC/Keycloak generated-host config is deferred and that the current generated-host example covers dev mode with memory/SQLite stores.

## Suggested approval path

1. Fix the CSRF empty-token behavior and add regression tests.
2. Deep-copy or freeze planned-context actor/resource data before exposing it to JS.
3. Decide and test `auth.mode=none` config behavior.
4. Decide and test session rotation's missing-old-session contract.
5. Update stale `cmd/xgoja/doc/17-xgoja-v2-reference.md` snippets.
6. Re-run:

```bash
GOFLAGS=-buildvcs=false go test ./pkg/gojahttp/auth/sessionauth ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http -count=1
GOFLAGS=-buildvcs=false go test ./... -count=1
make -C examples/xgoja/18-express-auth-host smoke
make -C examples/xgoja/21-generated-host-auth smoke
```

7. If Docker ports are available, run `make -C examples/xgoja/19-express-keycloak-auth-host smoke` and capture logs.

## References

- `sources/01-independent-inventory.md`
- `sources/02-independent-validation.md`
- `sources/03-independent-static-probes.md`
- `sources/04-full-go-test.md`
- `sources/05-generated-host-smoke.md`
- `sources/06-line-evidence.md`
- `sources/07-goja-map-mutation-probe.md`
- `sources/08-csrf-empty-token-probe.md`
- `sources/09-mode-none-store-validation-probe.md`
- `sources/10-docs-api-grep.md`
- `sources/11-coverage-grep.md`
