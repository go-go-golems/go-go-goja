---
Title: Generated host auth session and store configuration design
Ticket: XGOJA-GENERATED-HOST-AUTH-CONFIG
Status: active
Topics:
    - xgoja
    - auth
    - config
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl
      Note: Runtime-package ConfigureServices seam for generated-host auth service factories
    - Path: pkg/gojahttp/auth/sessionauth/sessionauth.go
      Note: Session manager config already exposes cookie and timeout knobs
    - Path: pkg/xgoja/app/factory.go
      Note: RuntimeFactory layers per-runtime host service overlays
    - Path: pkg/xgoja/app/host.go
      Note: HostOptions.ConfigureServices injects generated/custom host services
    - Path: pkg/xgoja/hostauth/config.go
      Note: Generated-host auth config and resolved config model
    - Path: pkg/xgoja/hostauth/lookup.go
      Note: Typed host-service lookup helpers for command/module consumers
    - Path: pkg/xgoja/hostauth/lookup_test.go
      Note: Host-service lookup tests for missing
    - Path: pkg/xgoja/hostauth/resolve.go
      Note: Config parser/default resolver for modes
    - Path: pkg/xgoja/hostauth/resolve_test.go
      Note: Resolver tests for defaults
    - Path: pkg/xgoja/hostauth/services.go
      Note: Hostauth service keys and typed service payloads
    - Path: pkg/xgoja/hostauth/stores.go
      Note: StoreBundle and memory/SQLite/Postgres auth store builders
    - Path: pkg/xgoja/hostauth/stores_test.go
      Note: Store builder tests for memory
    - Path: pkg/xgoja/providerapi/capabilities.go
      Note: HostServiceContributionCapability explains runtime/module host service contribution timing
    - Path: pkg/xgoja/providerapi/commands.go
      Note: CommandSetContext.Host makes host services visible to command providers
    - Path: pkg/xgoja/providerapi/module.go
      Note: ModuleSetupContext.Host and HostServiceLookup define module-side host service consumption
    - Path: pkg/xgoja/providers/http/http.go
      Note: HTTP provider external host service and Express loader integration
    - Path: pkg/xgoja/providers/http/serve.go
      Note: HTTP serve/hot-reload command path uses per-runtime external hosts
ExternalSources: []
Summary: 'Designs the next xgoja auth configuration phase: generated-host/session/store wiring, host services, command-visible auth services, and implementation tasks before OIDC.'
LastUpdated: 2026-06-14T22:30:00-04:00
WhatFor: Use this when implementing generated-host auth.session/auth.stores configuration and deciding how host services flow into xgoja commands and runtime modules.
WhenToUse: Read before adding session cookie config, store builders, host auth service keys, HTTP provider auth consumption, or generated-host templates.
---




# Generated host auth session and store configuration design

## Executive Summary

This document designs the next auth configuration phase after the first xgoja HTTP provider slice. The completed HTTP slice intentionally stayed small: it configures host-level HTTP behavior such as `enabled`, `listen`, `dev-errors`, and `reject-raw-routes`. It does not configure sessions, stores, OIDC, or authorization policy.

The next phase should add generated-host auth infrastructure configuration without moving application authorization into JavaScript or YAML. The generated Go host should build session managers, persistent stores, audit sinks, appauth stores, capability stores, and HTTP host services from typed configuration. JavaScript `express` route declarations should continue to declare intent only: `.public()`, `.auth(...)`, `.resource(...)`, `.allow(...)`, `.csrf()`, `.audit(...)`, and `.handle(...)`.

The important recent change is that host services are now visible to command providers through `providerapi.CommandSetContext.Host`. That makes the design cleaner: a generated host or auth service factory can expose typed host services once, and both command providers and module factories can consume those services through xgoja-native channels.

The proposed implementation is a new generated-host/auth-service layer with four responsibilities:

1. Parse secure-by-default auth configuration for sessions and stores.
2. Build typed Go services behind existing auth interfaces (`sessionauth.Store`, `audit.Store`, appauth store interfaces, `capability.Store`).
3. Expose those services through stable host-service keys so command providers and module factories can discover them.
4. Wire the HTTP provider to use supplied host/auth services when present, while preserving the current fallback xgoja-owned host behavior.

OIDC/Keycloak, MFA challenge flows, durable OIDC transaction stores, and policy DSLs remain deliberately out of scope for this ticket. This ticket creates the session/store foundation those features need.

## Problem Statement

xgoja now has a strict Express planned-route auth model and reusable Go host auth packages. The current examples prove three modes:

- no-auth public routes,
- local development auth,
- production-shaped Keycloak auth backed by SQL stores.

However, those examples are still custom Go hosts. Generated xgoja binaries currently cannot express the shared auth infrastructure in a production-shaped way. The HTTP provider can configure a small set of HTTP host options, but it does not know how to build session managers, SQL stores, audit sinks, appauth stores, or capability stores.

The missing piece is a generated-host configuration and wiring layer that lets generated binaries say:

```yaml
auth:
  mode: dev

  session:
    cookie:
      allow-insecure-http: false
      name: ""
      same-site: lax
      path: /
    idle-timeout: 30m
    absolute-timeout: 12h

  stores:
    default:
      driver: postgres
      dsn-env: APP_AUTH_DATABASE_URL
      apply-schema: false

    session: {}
    audit: {}
    appauth: {}
    capability: {}
```

The implementation must preserve these boundaries:

1. The Go host owns authentication, session cookies, CSRF, audit, stores, and lifecycle.
2. Express route plans declare route intent and authorization requirements, not storage schemas or secrets.
3. Application authorization remains app-owned Go (or a future chosen policy engine), not an xgoja YAML DSL.
4. DSNs and secrets should come from environment/config references rather than committed production YAML.
5. OIDC and Keycloak configuration should wait until the session/store foundation is stable.

## Current-State Analysis

### Command providers can now see host services

`providerapi.CommandSetContext` includes a `Host providerapi.HostServices` field. The command factory receives this context when xgoja attaches provider-owned commands.

Evidence:

- `pkg/xgoja/providerapi/commands.go:69-82` defines `CommandSetContext` and includes `Host` at line 77.
- `pkg/xgoja/app/command_providers.go:78-89` passes `Host: h.Services` into `provider.NewCommandSet(...)`.

This matters because auth is not only module setup. `serve`, future auth migration/check commands, and OIDC callback command wiring may need the same host-owned services that modules consume.

### Module factories already receive host services

`providerapi.ModuleSetupContext` includes `Host providerapi.HostServices`; `providerapi.HostServiceLookup` defines arbitrary opaque service lookup by key.

Evidence:

- `pkg/xgoja/providerapi/module.go:13-23` defines `ModuleSetupContext` with `Host` at line 20.
- `pkg/xgoja/providerapi/module.go:29-39` defines `HostServices` and `HostServiceLookup`.

The existing HTTP provider uses this path: the Express module factory decodes module config and calls `capability.newExpressLoader(ctx.Host, cfg)`.

Evidence:

- `pkg/xgoja/providers/http/http.go:33-58` registers the `express` module and `serve` command set.
- `pkg/xgoja/providers/http/http.go:41-47` passes `ctx.Host` into `newExpressLoader`.

### Runtime factories can receive per-runtime host-service overlays

`RuntimeFactory.NewRuntimeFromSectionsWithHostServices(...)` accepts a host-service overlay and layers it over the factory's base services before module setup.

Evidence:

- `pkg/xgoja/app/factory.go:78-86` accepts per-runtime `hostServices` and computes runtime services.
- `pkg/xgoja/app/factory.go:147-152` layers base services with per-runtime services.
- `pkg/xgoja/app/factory.go:171-176` runs `HostServiceContributionCapability` before module setup.

This is necessary for hot reload and command execution because a command may build or select a request-specific host, store, or auth bundle after command-line/config values are parsed.

### Host-service contribution currently targets runtime/module setup

`HostServiceContributionCapability` is explicitly described as a way for selected packages to contribute services before provider modules are set up. It is not currently a command-construction hook.

Evidence:

- `pkg/xgoja/providerapi/capabilities.go:70-78` describes contribution before runtime construction.
- `pkg/xgoja/providerapi/capabilities.go:91-98` defines `HostServiceContributionCapability`.

Implication: `CommandSetContext.Host` is useful for generated-host-provided services and service factories that exist before commands are built. It does not automatically expose per-command parsed values, because command construction happens before command execution.

### Generated hosts already have an injection seam

`app.HostOptions.ConfigureServices` lets generated/custom hosts add services to `app.HostServices` before command attachment and runtime factory creation.

Evidence:

- `pkg/xgoja/app/host.go:25-32` defines `HostOptions.ConfigureServices`.
- `pkg/xgoja/app/host.go:38-43` builds the host-service bag and calls `ConfigureServices`.
- `pkg/xgoja/app/host.go:48-55` stores those services on the host and passes them into the runtime factory.

Generated runtime-package templates expose this seam through `Options.ConfigureServices`.

Evidence:

- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl:44-48` defines `Options.ConfigureServices`.
- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl:85-99` passes `ConfigureServices` into `app.NewHostWithOptions`.

### The HTTP provider already supports external Go hosts

The HTTP provider defines `go-go-goja-http.host` and accepts an `ExternalHostService{Host, OwnsListen}`. When supplied, Express route registration uses that external host instead of creating an xgoja-owned host.

Evidence:

- `pkg/xgoja/providers/http/http.go:24-31` defines `HostServiceKey` and `ExternalHostService`.
- `pkg/xgoja/providers/http/http.go:177-207` creates the Express loader and chooses between an external host and a new `gojahttp.Host`.
- `pkg/xgoja/providers/http/http.go:209-225` validates the external host service payload.

Hot reload already demonstrates a per-runtime overlay pattern where a candidate host is passed to the runtime factory as a host service.

Evidence:

- `pkg/xgoja/providers/http/serve.go:138-142` requires `RuntimeFactoryWithHostServices` for hot reload.
- `pkg/xgoja/providers/http/serve.go:175-180` sets `go-go-goja-http.host` in a runtime overlay and creates the candidate runtime.

### Sessionauth already has the needed knobs

`sessionauth.Config` already exposes store, cookie name, path, `SameSite`, idle timeout, absolute timeout, and `AllowInsecureHTTP`.

Evidence:

- `pkg/gojahttp/auth/sessionauth/sessionauth.go:76-87` defines these config fields.
- `pkg/gojahttp/auth/sessionauth/sessionauth.go:103-105` documents secure-by-default cookies.
- `pkg/gojahttp/auth/sessionauth/sessionauth.go:125-129` supplies default idle and absolute timeouts.

The next phase does not need to invent a new session model. It needs to parse generated-host config and map it into this existing Go config.

## Gap Analysis

The current codebase has most runtime primitives, but not the generated-host composition layer.

| Gap | Why it matters | Desired outcome |
| --- | --- | --- |
| No typed auth config structs for generated hosts | Examples hand-wire session/store settings in Go | `auth.session` and `auth.stores` parse into typed Go structs with validation |
| No store inheritance model | Production configs need one DSN/driver for multiple auth stores | `auth.stores.default` can be inherited by `session`, `audit`, `appauth`, and `capability` |
| No shared store builder | SQL store constructors are package-specific | One builder dispatches to memory/sqlite/postgres adapters behind existing interfaces |
| No auth service host-service contract | HTTP and future auth commands need stable service keys | Typed services are discoverable from `HostServiceLookup` |
| Command providers see host services but not parsed command values at construction time | Store DSNs may come from Glazed config/env at execution time | Use a lazy auth service factory service, then build concrete services during command execution |
| HTTP provider only consumes external host service today | It cannot yet consume an auth bundle or build host options from it | HTTP provider can receive a preconfigured `gojahttp.Host` or construct one from an auth service bundle |
| No generated-host template integration | Runtime-package artifacts expose `ConfigureServices`, but binary templates do not yet generate auth wiring | Generated artifacts can opt into auth service factory construction |
| No lifecycle ownership model for DB handles/stores | SQL stores require closing DBs safely | Host-service builder records closers and runtime/host ownership explicitly |

## Proposed Architecture

### Architecture overview

Add a small xgoja host-auth layer that lives outside the Express JavaScript module. The preferred package name is one of:

- `pkg/xgoja/hostauth` for provider-neutral service keys, config, builders, and tests; or
- `pkg/xgoja/providers/authhost` if we want provider packaging, config sections, and commands in one place.

The safer first implementation is `pkg/xgoja/hostauth` plus optional provider wiring. That avoids a dependency cycle between the HTTP provider and an auth provider. The HTTP provider can import `hostauth`, but `hostauth` should not import the HTTP provider.

Core components:

1. `hostauth.Config`
   - typed representation of `auth.mode`, `auth.session`, and `auth.stores`.
2. `hostauth.ResolveConfig`
   - merges static defaults, config-file/env/Glazed values, and secure defaults.
3. `hostauth.BuildServices`
   - builds stores, session manager, audit sink, optional appauth/capability stores, and closers.
4. `hostauth.ServiceFactory`
   - a lazy factory that command providers can discover at command-construction time and invoke at command-execution time with parsed values.
5. `hostauth.Services`
   - concrete auth services built for one host/runtime/command execution.
6. HTTP provider consumption
   - use a concrete `gojahttp.Host` if supplied via `go-go-goja-http.host`, otherwise optionally build a host from `hostauth.Services`.

### Configuration model

First-phase schema:

```yaml
auth:
  mode: none # none | dev; oidc deferred

  session:
    cookie:
      allow-insecure-http: false
      name: ""
      same-site: lax # lax | strict | none | default
      path: /
    idle-timeout: 30m
    absolute-timeout: 12h

  stores:
    default:
      driver: memory # memory | sqlite | postgres
      dsn: ""       # allowed for local/dev; prefer dsn-env for production
      dsn-env: ""   # read DSN from environment at runtime
      apply-schema: false

    session: {}
    audit: {}
    appauth: {}
    capability: {}
```

Rules:

1. `auth.mode=none` builds no authenticator by default, but can still allow public planned routes.
2. `auth.mode=dev` builds dev/session infrastructure suitable for local demos.
3. `auth.mode=oidc` is reserved and should produce a clear “not implemented yet” error in this ticket if specified.
4. `auth.session.cookie.allow-insecure-http=false` is the default and production-safe value.
5. `allow-insecure-http=true` should be accepted only when explicitly configured and clearly documented as local/dev-only.
6. Empty cookie name means “use `sessionauth.New` defaults”.
7. Empty cookie path means `/`.
8. Empty `same-site` means `lax` for generated-host auth config, even though `sessionauth` has its own internal defaulting.
9. Empty `idle-timeout` and `absolute-timeout` use `sessionauth` defaults unless the config layer intentionally wants to materialize the defaults.
10. Store-specific blocks inherit from `auth.stores.default` field-by-field.
11. Store-specific `driver: memory` ignores DSN fields and never applies SQL schema.
12. `postgres` and `sqlite` require a DSN from either `dsn`, `dsn-env`, or a future secret reference.
13. `apply-schema=true` is acceptable for demos/tests but should be documented carefully for production migrations.

### Store inheritance

Resolved store config should be explicit after merging:

```go
type StoresConfig struct {
    Default    StoreConfig `yaml:"default"`
    Session    StoreConfig `yaml:"session"`
    Audit      StoreConfig `yaml:"audit"`
    AppAuth    StoreConfig `yaml:"appauth"`
    Capability StoreConfig `yaml:"capability"`
}

type StoreConfig struct {
    Driver      string `yaml:"driver"`
    DSN         string `yaml:"dsn"`
    DSNEnv      string `yaml:"dsn-env"`
    ApplySchema bool   `yaml:"apply-schema"`
}
```

The resolver should produce:

```go
type ResolvedStoresConfig struct {
    Session    ResolvedStoreConfig
    Audit      ResolvedStoreConfig
    AppAuth    ResolvedStoreConfig
    Capability ResolvedStoreConfig
}

type ResolvedStoreConfig struct {
    Name        string // session | audit | appauth | capability
    Driver      StoreDriver
    DSN         string
    ApplySchema bool
}
```

Field-level inheritance is important. For example:

```yaml
auth:
  stores:
    default:
      driver: postgres
      dsn-env: APP_AUTH_DATABASE_URL
      apply-schema: false
    audit:
      apply-schema: true
```

This should resolve audit to `driver=postgres`, `dsn` from `APP_AUTH_DATABASE_URL`, and `apply-schema=true` while the other stores inherit `apply-schema=false`.

### Host service keys and payloads

Use typed payloads. xgoja core treats service values as opaque, so correctness depends on stable keys and validation.

Proposed keys:

```go
const (
    ServicesKey       = "go-go-goja-auth.services"
    ServiceFactoryKey = "go-go-goja-auth.service-factory"
)
```

Proposed payloads:

```go
type ServiceFactory interface {
    BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error)
}

type Services struct {
    Config      ResolvedConfig
    AuthOptions gojahttp.AuthOptions

    SessionManager *sessionauth.Manager
    SessionStore   sessionauth.Store

    AuditSink  gojahttp.AuditSink
    AuditStore audit.Store

    AppAuthUsers       appauth.UserStore
    AppAuthMemberships appauth.MembershipStore
    AppAuthResources   appauth.ResourceStore

    CapabilityStore capability.Store

    Closers []func(context.Context) error
}
```

The first implementation may keep this struct smaller and add fields as wiring matures. It should still reserve the service key and package boundary now.

### CommandSetContext.Host usage

`CommandSetContext.Host` should be used to discover a lazy service factory, not necessarily already-open DB stores.

Reason: command providers are built when the root command tree is assembled. At that time, command-line flags and config-file layers have not necessarily been parsed into `values.Values`. DSNs may come from env/config fields attached to the command. Creating DB connections during command construction would make error reporting and command help worse.

Preferred command flow:

```go
func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    authFactory, _ := hostauth.LookupServiceFactory(ctx.Host)

    cmd := &serveCommand{
        runtimeFactory: ctx.RuntimeFactory,
        authFactory:    authFactory,
    }
    return commandSet(cmd), nil
}

func (c *serveCommand) Run(ctx context.Context, parsed *values.Values) error {
    var runtimeServices app.HostServices

    if c.authFactory != nil {
        authServices, err := c.authFactory.BuildHostAuthServices(ctx, parsed)
        if err != nil { return err }
        defer closeAll(authServices.Closers)

        gojaHost := gojahttp.NewHost(gojahttp.HostOptions{
            RejectRawRoutes: true,
            Auth:            authServices.AuthOptions,
        })
        runtimeServices.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{Host: gojaHost})
        runtimeServices.SetHostService(hostauth.ServicesKey, authServices)
    }

    rt, err := c.runtimeFactory.NewRuntimeFromSectionsWithHostServices(ctx, parsed, runtimeServices)
    if err != nil { return err }
    defer rt.Close(ctx)

    return invokeJSVerb(rt)
}
```

This pattern keeps the command factory lightweight but still benefits from `CommandSetContext.Host`: the command provider knows whether auth factory services are available and can shape commands/help/errors accordingly.

### ModuleSetupContext.Host usage

Modules should consume concrete services, not parse auth configuration. The Express module should continue to register routes against a `gojahttp.Host`.

Preferred module flow:

1. HTTP provider checks for `go-go-goja-http.host` first.
2. If present, it uses that host exactly as today.
3. If absent, it may check for `hostauth.ServicesKey` and create an xgoja-owned `gojahttp.Host` with `AuthOptions` from those services.
4. If no auth services are present, it falls back to current behavior: create a plain xgoja-owned host from HTTP settings.

This preserves backward compatibility while allowing generated-host auth wiring.

### Generated host integration

Generated `runtime-package` artifacts already expose `Options.ConfigureServices`. A custom application can inject the service factory manually:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(services *app.HostServices) {
        _ = services.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.Options{
            Config: authConfig,
        }))
    },
})
```

For generated binaries, the generator has two possible paths:

1. Generate a host-auth template when the build spec enables auth.
2. Use a provider package that exposes Glazed config sections and a service factory.

The recommended implementation path is staged:

- Phase 1: library-first `hostauth` package and manual `ConfigureServices` example/test.
- Phase 2: HTTP provider consumes `hostauth.ServiceFactoryKey` / `hostauth.ServicesKey` in `serve` command paths.
- Phase 3: generated runtime-package example wires `ConfigureServices` explicitly.
- Phase 4: add build-spec/generator sugar only after the library and provider paths are stable.

This avoids making the xgoja/v2 schema carry a large auth block before the ownership model is proven.

### Relationship to xgoja config files

`xgoja.yaml` is currently a build/runtime-plan spec with `providers`, `runtime.modules`, `sources`, `commands`, and `artifacts`. It does not currently include a top-level `auth` field.

This design should not immediately force a large schema change. Instead, support the config shape in a host-auth package and expose it through one or more of these surfaces:

1. `app.configFile` runtime config, if the generated command path already loads Glazed sections.
2. Provider command config sections for `auth.*` fields.
3. Manual `ConfigureServices` in runtime-package/custom hosts.
4. A later `xgoja/v2` schema addition only if the preceding surfaces are too awkward.

The important invariant is that auth config remains host infrastructure config, not route behavior config.

## Proposed API Sketch

### Config parsing package

```go
package hostauth

type Mode string

const (
    ModeNone Mode = "none"
    ModeDev  Mode = "dev"
    ModeOIDC Mode = "oidc" // reserved
)

type Config struct {
    Mode    Mode          `yaml:"mode"`
    Session SessionConfig `yaml:"session"`
    Stores  StoresConfig  `yaml:"stores"`
}

type SessionConfig struct {
    Cookie          CookieConfig `yaml:"cookie"`
    IdleTimeout     string       `yaml:"idle-timeout"`
    AbsoluteTimeout string       `yaml:"absolute-timeout"`
}

type CookieConfig struct {
    AllowInsecureHTTP bool   `yaml:"allow-insecure-http"`
    Name              string `yaml:"name"`
    SameSite          string `yaml:"same-site"`
    Path              string `yaml:"path"`
}

type ResolveOptions struct {
    LookupEnv func(string) (string, bool)
}

func ResolveConfig(cfg Config, opts ResolveOptions) (ResolvedConfig, error)
```

### Store builder package

```go
type StoreBundle struct {
    Session    sessionauth.Store
    Audit      audit.Store
    AppAuth    AppAuthStores
    Capability capability.Store
    Closers     []func(context.Context) error
}

type AppAuthStores struct {
    Users       appauth.UserStore
    Memberships appauth.MembershipStore
    Resources   appauth.ResourceStore
}

func BuildStores(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error)
```

### Session manager builder

```go
func BuildSessionManager(cfg ResolvedSessionConfig, store sessionauth.Store, loader sessionauth.ActorLoader) (*sessionauth.Manager, error) {
    return sessionauth.New(sessionauth.Config{
        Store:             store,
        ActorLoader:       loader,
        AllowInsecureHTTP: cfg.Cookie.AllowInsecureHTTP,
        CookieName:        cfg.Cookie.Name,
        Path:              cfg.Cookie.Path,
        SameSite:          cfg.Cookie.SameSite,
        IdleTimeout:       cfg.IdleTimeout,
        AbsoluteTimeout:   cfg.AbsoluteTimeout,
    })
}
```

### Auth service builder

```go
type Builder struct {
    Config      Config
    LookupEnv   func(string) (string, bool)
    ActorLoader sessionauth.ActorLoader
    Now         func() time.Time
}

func (b Builder) Build(ctx context.Context) (*Services, error) {
    resolved, err := ResolveConfig(b.Config, ResolveOptions{LookupEnv: b.LookupEnv})
    if err != nil { return nil, err }

    stores, err := BuildStores(ctx, resolved.Stores)
    if err != nil { return nil, err }

    sessionManager, err := BuildSessionManager(resolved.Session, stores.Session, b.ActorLoader)
    if err != nil { closeAll(stores.Closers); return nil, err }

    return &Services{
        Config:         resolved,
        SessionManager: sessionManager,
        SessionStore:   stores.Session,
        AuditStore:     stores.Audit,
        AuditSink:      audit.Sink{Store: stores.Audit},
        AuthOptions: gojahttp.AuthOptions{
            Authenticator: sessionManager,
            CSRF:          sessionManager,
            Audit:         audit.Sink{Store: stores.Audit},
        },
        Closers: stores.Closers,
    }, nil
}
```

## Key Flows

### Flow 1: generated binary, no auth

1. User configures HTTP provider with `reject-raw-routes: true`.
2. No hostauth factory is installed.
3. HTTP provider creates an xgoja-owned host with HTTP settings.
4. Public planned routes work.
5. Authenticated routes fail clearly because no authenticator is configured.

### Flow 2: generated binary with dev auth

1. Generated host installs `hostauth.ServiceFactoryKey` through `ConfigureServices`.
2. `serve` command provider discovers the factory through `CommandSetContext.Host`.
3. At command execution, parsed config values are available.
4. The service factory builds memory stores and a `sessionauth.Manager` with dev-safe settings.
5. The command creates a `gojahttp.Host` with `AuthOptions` and passes it as `go-go-goja-http.host` to runtime creation.
6. Express registers planned routes against that host.
7. Host-owned auth enforces sessions, CSRF, resources, authorization, and audit.

### Flow 3: generated binary with Postgres stores

1. Config sets `auth.stores.default.driver=postgres` and `dsn-env=APP_AUTH_DATABASE_URL`.
2. Store-specific overrides inherit the default DSN.
3. Service builder opens DB connections and constructs SQL stores.
4. `apply-schema` controls whether `ApplySchema(ctx)` runs.
5. Store closers are registered and run on command/runtime shutdown.
6. Keycloak/OIDC can later reuse the same stores without changing Express route declarations.

### Flow 4: hot reload

1. `serve --hot-reload` creates a candidate `gojahttp.Host` for each reload generation.
2. The auth service factory builds or reuses host-level auth services depending on lifecycle policy.
3. The hot reload loader passes the candidate HTTP host through `go-go-goja-http.host` as it does today.
4. It also passes `hostauth.ServicesKey` if modules need concrete auth services.
5. Candidate runtime closes route registrations cleanly; shared DB stores are not closed until the outer command exits.

## Design Decisions

### Decision: Keep auth config out of the Express module

- **Context:** The Express module is used by JavaScript route code. Auth infrastructure requires cookies, stores, OIDC, audit, and Go lifecycle.
- **Options considered:** Put auth config under `runtime.modules[].config` for `express`; put it in route builders; put it in generated-host services.
- **Decision:** Keep `express` as route declaration and registration. Build auth infrastructure in generated/custom Go host services.
- **Rationale:** This matches the existing boundary: route plans declare intent, Go host enforces. It avoids JS access to secrets/DSNs and avoids coupling route behavior to SQL schemas.
- **Consequences:** More Go/provider plumbing is needed, but the security model stays clear.
- **Status:** accepted

### Decision: Use a lazy service factory for command-visible auth

- **Context:** `CommandSetContext.Host` is available during command construction, before command-line/config values are parsed for a specific command invocation.
- **Options considered:** Build DB stores during command construction; use global package variables; expose a lazy factory through host services.
- **Decision:** Put a lazy `hostauth.ServiceFactory` in host services. Command providers discover it early and invoke it during command execution with parsed `values.Values`.
- **Rationale:** This preserves help/command construction behavior and lets DSNs come from env/config at runtime.
- **Consequences:** The factory interface becomes an important stable seam and needs tests for missing/malformed services.
- **Status:** proposed

### Decision: Add a provider-neutral `hostauth` package before generator sugar

- **Context:** HTTP provider can consume auth services, but auth service construction should not depend on the HTTP provider.
- **Options considered:** Put everything in `pkg/xgoja/providers/http`; add a new auth provider that imports HTTP; add provider-neutral `pkg/xgoja/hostauth`.
- **Decision:** Start with `pkg/xgoja/hostauth` for config, keys, builders, and lookup helpers.
- **Rationale:** This avoids dependency cycles and keeps auth service semantics reusable by generated hosts, command providers, and future OIDC packages.
- **Consequences:** A later provider may wrap this package for config sections/commands.
- **Status:** proposed

### Decision: Implement store inheritance with `auth.stores.default`

- **Context:** Most deployments use one DB for all host auth stores, with occasional per-store overrides.
- **Options considered:** Require full config for every store; use one global DSN only; implement field-level inheritance from `default`.
- **Decision:** Use `auth.stores.default` with field-level inheritance for each store.
- **Rationale:** It keeps config concise without hiding which stores exist.
- **Consequences:** Resolver tests must cover partial overrides and clear error messages.
- **Status:** accepted

### Decision: Keep OIDC deferred

- **Context:** OIDC requires issuer/client config, callback routes, transaction stores, PKCE/nonce handling, logout semantics, user normalization, and MFA freshness updates.
- **Options considered:** Add OIDC config now; reserve OIDC mode and return a clear error; implement session/store foundation first.
- **Decision:** Reserve `auth.mode=oidc`, but do not implement it in this ticket.
- **Rationale:** Session and store config are prerequisites and can be validated independently.
- **Consequences:** Keycloak examples remain custom-host until the OIDC follow-up.
- **Status:** accepted

### Decision: Do not add a YAML authorization policy DSL

- **Context:** Express routes declare `.allow("action")`, but actual business authorization requires app-owned tenants, memberships, resources, and policy decisions.
- **Options considered:** Add YAML action/resource rules; expose a policy engine adapter; keep authorization app-owned Go.
- **Decision:** Do not add a YAML authorization DSL in this ticket.
- **Rationale:** Policy DSL design is separate and riskier than infrastructure config. The appauth stores provide data, but apps still decide policy.
- **Consequences:** Generated-host auth can provide default dev/demo authorizers, but production authorization remains custom Go or future adapter work.
- **Status:** accepted

## Alternatives Considered

### Alternative: Put everything under `runtime.modules[].config` for `express`

Rejected. This would make the JavaScript import path responsible for host infrastructure. It also fails for command-only flows and OIDC handlers that live beside Express routes.

### Alternative: Add top-level `auth` to `xgoja/v2` immediately

Deferred. A top-level schema addition may be the right end state, but it should follow a working library/provider model. The current v2 schema does not have `auth`, and adding it prematurely would lock in representation before the host-service lifecycle is proven.

### Alternative: Require custom Go hosts for all production auth

Rejected as the only path. Custom hosts should remain supported, but generated binaries need a paved path for common auth infrastructure.

### Alternative: Use one monolithic SQL store package

Rejected for now. The existing implementation intentionally uses small subpackages: `sessionauth/sqlstore`, `audit/sqlstore`, `capability/sqlstore`, and `appauth/sqlstore`. The generated-host builder should compose those adapters rather than replace them.

## Implementation Plan

### Phase 0: Ticket and design handoff

1. Close `XGOJA-HTTP-AUTH-CONFIG` as the completed first-slice HTTP provider config ticket.
2. Keep this ticket focused on generated-host auth session/store config.
3. Relate this design to the key host-service, HTTP provider, generated template, and auth package files.

### Phase 1: `hostauth` package skeleton

1. Create `pkg/xgoja/hostauth`.
2. Add service keys and lookup helpers.
3. Add `Config`, `SessionConfig`, `CookieConfig`, `StoresConfig`, and `StoreConfig`.
4. Add `ResolvedConfig`, `ResolvedSessionConfig`, and `ResolvedStoresConfig`.
5. Add parse helpers for durations, `SameSite`, mode, and store driver.
6. Add clear error types/messages with config paths.
7. Add unit tests for defaults, invalid values, and error messages.

### Phase 2: store resolver and builders

1. Implement field-level inheritance from `auth.stores.default`.
2. Implement memory store builders.
3. Implement SQLite store builders using existing SQL store subpackages.
4. Implement Postgres store builders using existing SQL store subpackages.
5. Add `apply-schema` behavior per store.
6. Register DB closers.
7. Add tests for memory, SQLite, and Postgres config resolution. Postgres constructor tests can avoid live DBs initially; live smoke can follow.

### Phase 3: session manager builder

1. Map resolved session config to `sessionauth.Config`.
2. Preserve secure cookie defaults.
3. Add tests for `allow-insecure-http=false` default behavior.
4. Add tests for local/dev `allow-insecure-http=true` behavior.
5. Add tests for `same-site` parsing.
6. Add tests for timeout parsing and defaulting.

### Phase 4: concrete service builder

1. Implement `Builder.Build(ctx)` returning `hostauth.Services`.
2. Wire `AuthOptions.Authenticator` and `AuthOptions.CSRF` to the session manager.
3. Wire audit sink when audit store is configured.
4. Expose appauth and capability stores in `Services`.
5. Ensure partial construction failures close opened DB handles.
6. Add tests for cleanup on failure.

### Phase 5: HTTP provider consumption

1. Teach `newServeCommandSet` to look up `hostauth.ServiceFactoryKey` from `CommandSetContext.Host`.
2. At serve command execution, build auth services after values are parsed.
3. Create a `gojahttp.Host` from HTTP settings plus auth options.
4. Pass that host as `go-go-goja-http.host` to `NewRuntimeFromSectionsWithHostServices`.
5. Preserve existing behavior when no auth factory is present.
6. Add tests for command construction with and without auth factory.
7. Add tests for malformed host service payloads.

### Phase 6: runtime/module service visibility

1. Pass `hostauth.ServicesKey` into runtime overlays when concrete services are built.
2. Decide whether the Express loader should consume `hostauth.ServicesKey` directly or only consume `go-go-goja-http.host`.
3. Prefer using `go-go-goja-http.host` for route registration and keeping `hostauth.ServicesKey` for future modules/tools.
4. Add tests that modules see the concrete services when passed.

### Phase 7: generated runtime-package example

1. Add an example using generated runtime-package + manual `ConfigureServices` injection.
2. Demonstrate memory/dev mode.
3. Demonstrate SQLite store mode if feasible.
4. Include a smoke script.
5. Document how this differs from custom Go host examples 18/19/20.

### Phase 8: docs and migration guidance

1. Update xgoja provider runtime config docs.
2. Update Express auth guide to point to generated-host auth config.
3. Document secure defaults and local-only insecure cookie mode.
4. Document store inheritance.
5. Document why OIDC is deferred.
6. Document how to move from custom host to generated host where possible.

### Phase 9: validation

1. Run `go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/app -count=1`.
2. Run relevant auth store tests.
3. Run full `go test ./... -count=1` before final push.
4. Run example smokes affected by HTTP provider changes.
5. Run docmgr doctor for this ticket.

## Test Strategy

### Unit tests

- Config defaults.
- Mode parsing.
- Duration parsing.
- SameSite parsing.
- Store driver parsing.
- DSN env resolution.
- Store inheritance.
- Secure cookie default preservation.
- Invalid config path errors.
- Host service lookup helpers.
- Service factory construction and cleanup on failure.

### Integration tests

- HTTP provider serve command without auth factory preserves current behavior.
- HTTP provider serve command with auth factory builds an external host.
- Express planned route registration uses the supplied host.
- Hot reload candidate runtime still receives external host overlays.
- Runtime-package `ConfigureServices` exposes auth factory to command providers.

### Store tests

- Memory builder returns working stores.
- SQLite builder applies schema and passes existing store contract tests through built stores.
- Postgres builder validates dialect/DSN and is covered by Keycloak or a later containerized smoke.

### Smoke tests

- Existing `examples/xgoja/13-http-serve-jsverbs` smoke still passes.
- Dev auth generated-host example smoke passes.
- Existing custom-host examples 18, 19, and 20 still pass.

## Risks

1. **Command construction vs command execution timing:** `CommandSetContext.Host` is available early, but parsed command values are only available at execution. The lazy factory pattern is intended to avoid this risk.
2. **Lifecycle leaks:** SQL DB handles must close on construction failures and command/runtime shutdown.
3. **Config surface sprawl:** Adding top-level xgoja schema too early could create migration burden.
4. **Security default regressions:** Cookie defaults must remain secure; local insecure mode must be explicit.
5. **Provider dependency cycles:** Keep `hostauth` provider-neutral so HTTP provider can consume it safely.
6. **Authorization confusion:** Store config is not policy. Docs must repeat that app authorization remains app-owned.

## Open Questions

1. Should `hostauth` become a standalone package only, or also a provider with Glazed config sections and auth helper commands?
2. Should the first generated-host example use runtime-package `ConfigureServices` or a binary template extension?
3. Should `auth.mode=dev` include a default dev login handler, or only build session/store infrastructure?
4. Should `apply-schema` be allowed in production mode, or should docs merely warn?
5. Should DSN config support `dsn-file` or a generic secret-reference type in this phase?
6. Should HTTP provider consume `hostauth.ServicesKey` directly, or should all HTTP integration go through `go-go-goja-http.host`?
7. How should host-level appauth seed data be expressed for dev examples without becoming a production policy DSL?

## References

- `pkg/xgoja/providerapi/commands.go:69-82` — command providers receive `CommandSetContext.Host`.
- `pkg/xgoja/app/command_providers.go:78-89` — xgoja passes host services into command-set construction.
- `pkg/xgoja/providerapi/module.go:13-39` — modules receive host services and can use `HostServiceLookup`.
- `pkg/xgoja/providerapi/capabilities.go:70-98` — runtime host-service contribution capability.
- `pkg/xgoja/app/factory.go:78-86` and `pkg/xgoja/app/factory.go:147-176` — per-runtime host-service overlays and contribution flow.
- `pkg/xgoja/app/host.go:25-55` — generated/custom host service injection seam.
- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl:44-99` — runtime-package `ConfigureServices` seam.
- `pkg/xgoja/providers/http/http.go:24-31`, `177-225` — HTTP provider external host service.
- `pkg/xgoja/providers/http/serve.go:138-180` — hot reload runtime overlay using an external HTTP host.
- `pkg/gojahttp/auth/sessionauth/sessionauth.go:76-87` — session manager config knobs.
- `pkg/gojahttp/auth/sessionauth/sqlstore` — SQL session store.
- `pkg/gojahttp/auth/audit/sqlstore` — SQL audit store.
- `pkg/gojahttp/auth/appauth/sqlstore` — SQL appauth store.
- `pkg/gojahttp/auth/capability/sqlstore` — SQL capability store.
- `ttmp/2026/06/14/XGOJA-HTTP-AUTH-CONFIG--design-xgoja-http-auth-provider-configuration/design-doc/01-http-auth-provider-configuration-analysis-and-implementation-guide.md` — previous first-slice HTTP provider config design.
