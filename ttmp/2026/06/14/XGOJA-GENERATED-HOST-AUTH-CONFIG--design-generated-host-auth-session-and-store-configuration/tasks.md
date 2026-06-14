# Tasks

## TODO

### A. Ticket setup and design

- [x] Task 1: Create `XGOJA-GENERATED-HOST-AUTH-CONFIG` as the focused follow-up ticket for generated-host auth session/store configuration.
- [x] Task 2: Add the primary design document for generated-host auth configuration.
- [x] Task 3: Relate all design-critical source files to the primary design document with `docmgr doc relate`.
- [x] Task 4: Add a strict implementation diary entry for ticket creation and design handoff.
- [x] Task 5: Close `XGOJA-HTTP-AUTH-CONFIG` after transferring its next-phase backlog into this ticket.

### B. Package boundary and naming

- [x] Task 6: Decide final package name for provider-neutral auth host helpers: prefer `pkg/xgoja/hostauth` unless dependency review finds a better location.
- [x] Task 7: Create the package skeleton with `doc.go`, `config.go`, `services.go`, `lookup.go`, and initial tests.
- [x] Task 8: Define stable host service keys for concrete services and lazy service factories.
- [x] Task 9: Add lookup helpers that validate `providerapi.HostServiceLookup` payload types and return clear errors.
- [x] Task 10: Add tests for missing service keys, wrong payload types, nil service factories, and duplicate/multi-value behavior.

### C. Auth config model

- [x] Task 11: Define `hostauth.Config` with `mode`, `session`, and `stores` sections.
- [x] Task 12: Define `hostauth.SessionConfig` with cookie and timeout subfields.
- [x] Task 13: Define `hostauth.CookieConfig` with `allow-insecure-http`, `name`, `same-site`, and `path`.
- [x] Task 14: Define `hostauth.StoresConfig` with `default`, `session`, `audit`, `appauth`, and `capability` blocks.
- [x] Task 15: Define `hostauth.StoreConfig` with `driver`, `dsn`, `dsn-env`, and `apply-schema`.
- [x] Task 16: Define resolved config types that remove ambiguity after defaulting and inheritance.
- [x] Task 17: Add config path metadata or error helpers so parse failures point to fields such as `auth.session.cookie.same-site`.

### D. Config parsing and defaults

- [x] Task 18: Implement mode parsing for `none`, `dev`, and reserved `oidc`.
- [x] Task 19: Make `auth.mode=oidc` return a clear not-yet-implemented error in this ticket.
- [x] Task 20: Implement duration parsing for `idle-timeout` and `absolute-timeout` using Go duration syntax.
- [x] Task 21: Decide whether empty durations materialize defaults in resolved config or delegate defaulting to `sessionauth.New`.
- [x] Task 22: Implement `SameSite` parsing for `lax`, `strict`, `none`, and `default`.
- [x] Task 23: Make generated-host config default `same-site` to `lax` unless explicitly set otherwise.
- [x] Task 24: Default empty cookie path to `/`.
- [x] Task 25: Preserve empty cookie name as “use `sessionauth` secure default”.
- [x] Task 26: Keep `allow-insecure-http=false` as the default and require explicit opt-in for dev/local HTTP.
- [x] Task 27: Add unit tests for all session/cookie defaults.
- [x] Task 28: Add unit tests for invalid modes, durations, same-site values, empty paths, and unsafe cookie combinations.

### E. Store inheritance and DSN resolution

- [x] Task 29: Implement field-level inheritance from `auth.stores.default` into each store-specific block.
- [x] Task 30: Decide final semantics for empty store-specific blocks: inherit all fields from default.
- [x] Task 31: Decide final semantics for explicit `driver: memory`: ignore inherited DSN fields or reject conflicting DSNs.
- [x] Task 32: Implement DSN resolution from `dsn-env` with injectable `LookupEnv` for tests.
- [x] Task 33: Reject configs that set both `dsn` and `dsn-env` unless a documented precedence is chosen.
- [x] Task 34: Reject `postgres` and `sqlite` stores with no resolved DSN.
- [x] Task 35: Permit `memory` stores with no DSN.
- [x] Task 36: Add table-driven tests for default inheritance, partial overrides, DSN env lookup, missing env vars, and driver errors.
- [x] Task 37: Document that production DSNs should use env/config refs and should not be committed in example YAML.

### F. Store builders

- [ ] Task 38: Define a `StoreBundle` struct for session, audit, appauth, capability, and closers.
- [ ] Task 39: Implement memory store builders for session, audit, appauth, and capability.
- [ ] Task 40: Implement SQLite store builders using `sessionauth/sqlstore`, `audit/sqlstore`, `appauth/sqlstore`, and `capability/sqlstore`.
- [ ] Task 41: Implement Postgres store builders using the same SQL store packages and Postgres dialects.
- [ ] Task 42: Decide whether separate DB handles are opened per store or shared when resolved DSNs/dialects match.
- [ ] Task 43: If sharing DB handles, implement deterministic ownership and one closer per shared handle.
- [ ] Task 44: Implement `apply-schema` dispatch per store.
- [ ] Task 45: Add cleanup on partial construction failure.
- [ ] Task 46: Add tests that `apply-schema=false` does not create tables.
- [ ] Task 47: Add SQLite integration tests that built stores pass existing store contract helpers where practical.
- [ ] Task 48: Add constructor-level Postgres tests that validate dialect/DSN behavior without requiring a live server.
- [ ] Task 49: Add optional containerized Postgres smoke coverage if it can be kept fast and non-flaky.

### G. Session manager and auth options builder

- [ ] Task 50: Implement `BuildSessionManager` that maps resolved config into `sessionauth.Config`.
- [ ] Task 51: Ensure `sessionauth.ActorLoader` can be injected by custom/generated hosts.
- [ ] Task 52: Provide a safe default actor loader only for dev/demo mode if appropriate.
- [ ] Task 53: Wire `SessionManager` as both `gojahttp.Authenticator` and `gojahttp.CSRFProtector`.
- [ ] Task 54: Wire audit store through `audit.Sink` and `gojahttp.AuthOptions.Audit`.
- [ ] Task 55: Decide how default development authorizers/resources are represented, if at all.
- [ ] Task 56: Keep production authorization app-owned; do not add YAML authorization policy in this ticket.
- [ ] Task 57: Add tests that session config maps to secure cookie behavior.
- [ ] Task 58: Add tests that `allow-insecure-http=true` only changes cookie security when explicitly configured.

### H. Auth service factory

- [ ] Task 59: Define `ServiceFactory` interface with a build method that accepts context and parsed values.
- [ ] Task 60: Implement a default service factory that resolves config and builds concrete services at command execution time.
- [ ] Task 61: Add `Services` struct with resolved config, auth options, stores, session manager, audit sink, and closers.
- [ ] Task 62: Add service factory lookup helper for `CommandSetContext.Host` consumers.
- [ ] Task 63: Add concrete services lookup helper for runtime/module consumers.
- [ ] Task 64: Add tests proving factories are discoverable during command construction but build services later.
- [ ] Task 65: Add tests that closers run on build failures and normal shutdown.

### I. HTTP provider integration

- [ ] Task 66: Update `pkg/xgoja/providers/http` to optionally import and consume `pkg/xgoja/hostauth`.
- [ ] Task 67: Teach `newServeCommandSet` to discover `hostauth.ServiceFactoryKey` from `CommandSetContext.Host`.
- [ ] Task 68: Preserve existing serve behavior exactly when no hostauth factory is present.
- [ ] Task 69: At serve command execution, build auth services after Glazed values are parsed.
- [ ] Task 70: Construct a `gojahttp.Host` from HTTP settings plus `hostauth.Services.AuthOptions`.
- [ ] Task 71: Pass the constructed host through `go-go-goja-http.host` as `httpprovider.ExternalHostService`.
- [ ] Task 72: Pass concrete auth services through `hostauth.ServicesKey` for future module/tool consumers.
- [ ] Task 73: Decide whether the Express loader should ever create a host from `hostauth.ServicesKey` if no external HTTP host is present.
- [ ] Task 74: Add tests for malformed auth host service payloads.
- [ ] Task 75: Add tests that HTTP provider config (`dev-errors`, `reject-raw-routes`) still applies when hostauth creates the host.
- [ ] Task 76: Add tests that external custom `go-go-goja-http.host` still wins over hostauth-generated hosts.

### J. Hot reload integration

- [ ] Task 77: Review `serveVerbHotReload` lifecycle with auth service factories.
- [ ] Task 78: Decide whether hot reload reuses one auth service bundle across candidate runtimes or rebuilds per candidate.
- [ ] Task 79: Prefer sharing DB-backed auth services across candidates while creating per-candidate HTTP hosts.
- [ ] Task 80: Ensure candidate runtime overlays include `go-go-goja-http.host` and any needed auth services.
- [ ] Task 81: Ensure candidate runtime close does not close shared command-level DB handles prematurely.
- [ ] Task 82: Add hot reload tests for auth service lifecycle if feasible.

### K. Generated host / runtime-package integration

- [ ] Task 83: Add a runtime-package example that manually injects `hostauth.ServiceFactoryKey` with `ConfigureServices`.
- [ ] Task 84: Demonstrate `auth.mode=dev` with memory stores in that example.
- [ ] Task 85: Demonstrate SQLite stores in that example if the setup remains simple.
- [ ] Task 86: Add a smoke script for the generated-host auth example.
- [ ] Task 87: Document how runtime-package `ConfigureServices` differs from custom host examples 18/19/20.
- [ ] Task 88: Decide whether to extend generated binary templates directly in this ticket or leave that for a follow-up.
- [ ] Task 89: If extending templates, update `cmd/xgoja/internal/generate/templates/main.go.tmpl` or related template generation code with minimal auth service factory injection.
- [ ] Task 90: Add generator tests for any new template output.

### L. Documentation

- [ ] Task 91: Update `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` with generated-host auth service factory guidance.
- [ ] Task 92: Update `cmd/xgoja/doc/17-xgoja-v2-reference.md` with the supported auth configuration surface or explicit deferral if schema remains unchanged.
- [ ] Task 93: Update `pkg/doc/29-express-auth-user-guide.md` to mention generated-host session/store configuration.
- [ ] Task 94: Update `pkg/doc/31-express-auth-examples.md` with the new example and smoke command.
- [ ] Task 95: Document store inheritance with concrete YAML snippets.
- [ ] Task 96: Document cookie security defaults and local-only `allow-insecure-http` usage.
- [ ] Task 97: Document that OIDC/Keycloak config is deferred until this foundation is stable.
- [ ] Task 98: Document that app authorization remains app-owned Go, not YAML policy.

### M. Validation and release hygiene

- [ ] Task 99: Run focused tests for `pkg/xgoja/hostauth`, `pkg/xgoja/providers/http`, and `pkg/xgoja/app`.
- [ ] Task 100: Run focused tests for all auth store packages.
- [ ] Task 101: Run affected xgoja example smokes.
- [ ] Task 102: Run `go test ./... -count=1` before final push.
- [ ] Task 103: Run targeted security scanner if SQL/cookie/http server code changes.
- [ ] Task 104: Run `docmgr doctor --ticket XGOJA-GENERATED-HOST-AUTH-CONFIG --stale-after 30`.
- [ ] Task 105: Update changelog and diary after each implementation slice.
- [ ] Task 106: Commit in focused slices and push to `task/goja-express-auth`.

### N. Explicitly deferred follow-ups

- [ ] Task 107: Defer `auth.mode=oidc` implementation to a later OIDC/Keycloak config ticket.
- [ ] Task 108: Defer durable OIDC transaction store design to the OIDC follow-up.
- [ ] Task 109: Defer MFA freshness update flows to the Keycloak/MFA ticket.
- [ ] Task 110: Defer YAML authorization policy DSL design to a separate policy adapter ticket.
- [ ] Task 111: Defer secret-manager integrations beyond `dsn-env` unless needed by the first generated-host example.
