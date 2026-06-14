---
Title: HTTP auth provider configuration analysis and implementation guide
Ticket: XGOJA-HTTP-AUTH-CONFIG
Status: active
Topics:
    - xgoja
    - http
    - auth
    - configuration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Secure-by-default cookie settings for future auth phase
    - Path: pkg/gojahttp/host.go
      Note: HostOptions fields targeted by first slice
    - Path: pkg/xgoja/app/factory.go
      Note: xgoja module config merge path
    - Path: pkg/xgoja/providers/http/http.go
      Note: Current HTTP provider config and host construction seam
    - Path: pkg/xgoja/providerutil/sections.go
      Note: Provider config parsing and merge helpers
ExternalSources: []
Summary: Design and phased implementation guide for xgoja HTTP/provider auth configuration.
LastUpdated: 2026-06-14T20:49:00-04:00
WhatFor: Use when adding xgoja.yaml/provider configuration for HTTP host options, auth sessions, OIDC, stores, and production-ready generated hosts.
WhenToUse: When deciding which auth-related behavior belongs in provider config versus JavaScript route declarations or custom Go hosts.
---


# HTTP auth provider configuration analysis and implementation guide

## Executive Summary

The Express auth work established a strong boundary: JavaScript declares route intent, while the Go host owns HTTP serving, authentication, sessions, cookies, resources, authorization, CSRF, audit, and persistence. The next configuration step should keep that boundary intact while making the simple cases easy in `xgoja.yaml`.

This guide recommends a phased design:

1. **Start small in the existing `go-go-goja-http` provider** with host-safe HTTP options that are already implemented by `gojahttp.HostOptions`: listen address, dev error behavior, and strict raw-route rejection.
2. **Use `runtime.modules[].config` for xgoja.yaml static provider configuration**, because generated v2 apps already pass that map into provider module setup.
3. **Keep public Glazed `http` flags/config as command-time overrides** for the same fields, so generated binaries can override listen/dev settings without rebuilding.
4. **Design but defer production auth wiring** (`auth.mode`, session cookie hardening, OIDC, transaction store, SQL stores) until the provider has enough host-service seams or template generation support to own those resources responsibly.
5. **Route JavaScript remains policy-intent only**: `.public()`, `.auth(...)`, `.resource(...)`, `.csrf()`, `.allow(...)`, `.audit(...)`, and handler logic. It should not configure cookies or OIDC.

The first implementation slice should be intentionally small and safe:

```yaml
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      config:
        enabled: true
        listen: 127.0.0.1:8787
        dev-errors: false
        reject-raw-routes: true
```

This immediately gives generated no-auth and public-route apps a production-shaped host default (`reject-raw-routes: true`) without forcing Keycloak/session configuration into the generic Express provider.

## Problem Statement

The current examples demonstrate three host modes:

- `20-express-hello-world`: no auth infrastructure, public planned routes only.
- `18-express-auth-host`: custom Go host with development auth.
- `19-express-keycloak-auth-host`: custom Go host with Keycloak, sessions, appauth, audit, capability, and SQL stores.

Generated xgoja apps can select the HTTP provider and `express`, but provider configuration is currently minimal. Users need a coherent answer to:

- What can I configure in `xgoja.yaml`?
- Which values are safe defaults for production?
- How do command flags/config/env override static xgoja.yaml values?
- Where do cookie hardening, Keycloak, and auth stores belong?
- When should I stop using generic provider YAML and switch to a generated/custom Go host template?

The risk is overcorrecting in either direction:

- If YAML config is too small, every serious app must immediately become a custom Go host.
- If YAML config becomes a policy DSL or app model DSL, the provider becomes a brittle application framework and undermines app-owned Go authorization.

## Current State Evidence

### Existing HTTP provider configuration

`pkg/xgoja/providers/http/http.go` currently exposes a public Glazed section:

```go
type settings struct {
    Enabled bool   `glazed:"enabled"`
    Listen  string `glazed:"listen"`
}
```

The fields become command/config/env values under the `http` section with `http-` flag prefixes. The provider starts a `gojahttp.Host` when Express is used.

### Existing provider configuration machinery

`cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` describes two phases:

```text
xgoja.yaml runtime.modules[].config
  -> provider XGojaConfigSection
  -> ModuleSetupContext.Config

Glazed command/config/env/flag values
  -> provider GlazedConfigSections
  -> XGojaConfigFromGlazed override
  -> final ModuleSetupContext.Config
```

This is the right mechanism for xgoja.yaml provider config because it already handles static config plus runtime overrides.

### Existing host auth packages

The auth work produced reusable Go packages:

- `sessionauth` for opaque server-side app sessions and CSRF.
- `keycloakauth` for OIDC login/callback/logout.
- `appauth` for app-owned users, tenants, memberships, resources, and simple authorization.
- `audit` for normalized audit records.
- `capability` for single-purpose bearer capabilities.
- SQL stores for session, audit, appauth, and capability.

Those packages are production-shape building blocks, but wiring all of them from a generic provider config requires secret handling, migrations, transaction stores, store lifecycles, and app-specific authorization hooks. That should be phased, not bolted into the first provider config pass.

## Configuration Boundary

### YAML/provider config should own infrastructure

Provider or generated-host config may own:

- HTTP listen address and ownership.
- development error verbosity.
- raw-route rejection.
- session cookie hardening.
- session idle/absolute timeouts.
- store backend selection and DSNs.
- OIDC issuer/client/redirect settings.
- OIDC transaction store backend.
- optional demo seed data.

### JavaScript should own route intent only

Express scripts should own:

- route patterns,
- `.public()`,
- `.auth(express.user().required())`,
- `.resource(...)`,
- `.csrf()`,
- `.allow("action")`,
- `.audit("event")`,
- handler code.

JavaScript should not configure `Secure`, `SameSite`, OIDC client secrets, SQL DSNs, or application authorization policy.

### Go host/application code should own real policy

Application authorization should stay in Go services or a chosen policy engine. YAML should not become this:

```yaml
authorization:
  rules:
    - action: project.update
      roles: [admin]
```

That is too close to a new policy language and too far from the app's data invariants. Provider config may select `authorization.mode: appauth` or `custom`, and it may seed demo tenants/resources, but production authorization rules should be Go-owned.

## Proposed Configuration Model

### Phase 1: HTTP host config in the existing provider

The first implemented provider config should be flat and limited:

```yaml
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      as: express
      config:
        enabled: true
        listen: 127.0.0.1:8787
        dev-errors: false
        reject-raw-routes: true
```

Fields:

| Field | Default | Meaning |
| --- | --- | --- |
| `enabled` | `true` when values/config are present | Start the xgoja-owned HTTP server when Express registers a route. |
| `listen` | `127.0.0.1:8787` | Listen address for the xgoja-owned server. |
| `dev-errors` | `false` | Use development error responses from `gojahttp.Host`. |
| `reject-raw-routes` | `true` | Reject matched raw/unplanned routes; planned `.public()`/`.auth()` routes and static mounts still work. |

The public Glazed `http` section should expose the same fields as command-time overrides:

```bash
my-app run site.js --http-listen 127.0.0.1:9000 --http-dev-errors --http-reject-raw-routes
```

Implementation notes:

- Add `XGojaConfigSectionCapability` to the HTTP provider capability.
- Keep `GlazedConfigSectionCapability` for public flags/config/env.
- Map public `http` fields into the internal xgoja config in `XGojaConfigFromGlazed`.
- Decode final `ModuleSetupContext.Config` in the Express module factory.
- Use those settings when constructing the internal `gojahttp.Host`.
- If an external host service is provided, do not override that host's options; external hosts own their own hardening.

### Phase 2: Session cookie config behind a generated host seam

Once Phase 1 is stable, introduce a session config shape, preferably in a generated-host template or a provider host-service contribution that can own stores and closers:

```yaml
auth:
  mode: none # none | dev | oidc

  session:
    cookie:
      allow-insecure-http: false
      name: ""        # default: __Host-app when secure
      same-site: lax   # lax | strict | none
      path: /
    idle-timeout: 30m
    absolute-timeout: 12h
```

Production defaults:

- `allow-insecure-http: false`
- `name: __Host-app` by default
- `Secure: true`
- `HttpOnly: true`
- `SameSite: lax`
- `Path: /`

The current `sessionauth.Config` already supports these knobs except that SameSite parsing and duration parsing would need a config adapter.

### Phase 3: Store config under `auth.stores`

Store config should live under `auth`, not a separate top-level `authStores`, and should use a default store that individual stores inherit:

```yaml
auth:
  stores:
    default:
      driver: postgres # memory | sqlite | postgres
      dsn: ${AUTH_DB_DSN}
      apply-schema: false

    session: {}
    audit: {}
    appauth: {}
    capability: {}
```

Why `auth.stores.default` instead of `postgresDsn`:

- It does not bake the driver into the field name.
- It works for SQLite/Postgres/memory and future Redis transaction stores.
- It clearly answers who consumes it: auth store builders.
- It supports per-store overrides without repeating boilerplate.

Example inheritance:

```yaml
auth:
  stores:
    default:
      driver: postgres
      dsn: ${AUTH_DB_DSN}
      apply-schema: false
    audit:
      dsn: ${AUDIT_DB_DSN}
```

Session/appauth/capability use `AUTH_DB_DSN`; audit uses `AUDIT_DB_DSN`.

### Phase 4: OIDC/Keycloak config

Use generic `oidc` naming unless a field is Keycloak-specific:

```yaml
auth:
  mode: oidc

  oidc:
    issuer-url: https://keycloak.example.com/realms/app
    client-id: goja-app
    client-secret:
      env: OIDC_CLIENT_SECRET
    redirect-url: https://app.example.com/auth/callback
    after-login-url: /
    after-logout-url: /
    scopes: [openid, profile, email]

  oidc-transaction:
    store:
      driver: postgres # memory for dev/single-process only
      dsn: ${AUTH_DB_DSN}
    ttl: 10m
```

The shared transaction store is needed when `/auth/login` and `/auth/callback` can land on different replicas. A memory store is acceptable for localhost demos and single-process tests only.

### Phase 5: Generated Go host template integration

Advanced apps should graduate to generated or custom Go hosts. The template should generate the explicit Go wiring that production apps need:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:             cfg.HTTP.DevErrors,
    RejectRawRoutes: cfg.HTTP.RejectRawRoutes,
    Auth: gojahttp.AuthOptions{
        Authenticator: sessions,
        CSRF:          sessions,
        Resources:     resolver,
        Authorizer:    authorizer,
        Audit:         auditSink,
    },
})
```

The template can load config, create stores, run migrations or refuse `apply-schema` in production, configure OIDC handlers, and expose app-specific authorization seams. This avoids stuffing all production behavior into the generic `express` module.

## Design Decisions

### Decision 1: Start with HTTP host options, not full auth

Status: accepted.

The smallest useful provider config is `enabled`, `listen`, `dev-errors`, and `reject-raw-routes`. These fields map directly to existing provider/host behavior and do not require app-specific policy.

### Decision 2: Use `runtime.modules[].config` for static xgoja.yaml provider config

Status: accepted.

This is already the v2-native way to configure selected provider modules. It keeps provider config near module selection and avoids inventing unsupported arbitrary top-level build-spec sections.

### Decision 3: Public Glazed `http` values override static xgoja config

Status: accepted.

Generated binaries need runtime overrides for listen address and local debugging. The provider config machinery already supports this merge path.

### Decision 4: Keep `auth.stores.default` for future store inheritance

Status: proposed.

A default store block is clearer and more extensible than `authStores.postgresDsn` or `postgresDsn` fields.

### Decision 5: No YAML authorization DSL in this track

Status: accepted.

App authorization remains app-owned Go code or an explicitly chosen policy engine. Provider config may choose/store/seed, but not encode real business policy rules.

## Alternatives Considered

### Top-level `authStores`

Rejected. It separates stores from the auth system that consumes them and encourages one-off shorthand fields like `postgresDsn`.

### JavaScript import-time auth configuration

Rejected. `require("express")` and `express.app()` should not configure cookies, OIDC, or SQL stores. Those belong to the Go host/provider.

### Full Keycloak config in the first provider slice

Deferred. Keycloak/OIDC needs transaction store, secret references, callback handlers, and app user normalization. That should be implemented in a host template or a later provider phase with stronger seams.

## First Implementation Slice

Implement now:

1. Add `dev-errors` and `reject-raw-routes` to the HTTP provider settings.
2. Add an internal xgoja config section for the HTTP provider so `runtime.modules[].config` can set those fields.
3. Map public Glazed `http` fields into that internal config so command-time values override static config.
4. Decode final provider config in the Express module factory.
5. Use the resulting values when constructing the xgoja-owned `gojahttp.Host`.
6. Add tests proving:
   - static config is accepted and exposed in module listing/config,
   - public values are mapped into provider config,
   - dev error mode changes handler error responses,
   - `reject-raw-routes` is set on internally-created hosts without affecting external host services.
7. Update provider docs/reference examples.

Do not implement yet:

- `auth.mode`,
- session cookie config,
- OIDC config,
- transaction store config,
- SQL auth store config in generated provider YAML,
- YAML policy rules.

## Validation Strategy

Run targeted tests while implementing:

```bash
go test ./pkg/xgoja/providers/http ./pkg/xgoja/app -count=1
```

Run a generated HTTP example smoke if impacted:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
```

Run full validation before push:

```bash
go test ./... -count=1
```

## Open Questions

1. Should the long-term production auth config live in the generic `go-go-goja-http` provider, a sibling `go-go-goja-auth` provider, or generated host templates?
2. Should secret references use a shared `secretRef` struct across xgoja providers?
3. Should `apply-schema` be allowed from generated binaries, or restricted to examples/tests?
4. Should OIDC transaction stores share the `auth.stores` inheritance mechanism or have their own `auth.oidc-transaction.store` block?

## References

- `pkg/xgoja/providers/http/http.go` — current HTTP provider settings and host creation.
- `pkg/xgoja/providers/http/http_test.go` — existing HTTP provider tests.
- `pkg/xgoja/app/factory.go` — xgoja module config merge path.
- `pkg/xgoja/providerutil/sections.go` — xgoja config section parsing/merging helpers.
- `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md` — provider config model.
- `pkg/gojahttp/host.go` — host options (`Dev`, `RejectRawRoutes`, `Auth`).
- `pkg/gojahttp/auth/sessionauth/sessionauth.go` — secure-by-default session cookie config.
- `examples/xgoja/20-express-hello-world` — no-auth public-route custom host.
- `examples/xgoja/19-express-keycloak-auth-host` — production-shaped custom auth host.
