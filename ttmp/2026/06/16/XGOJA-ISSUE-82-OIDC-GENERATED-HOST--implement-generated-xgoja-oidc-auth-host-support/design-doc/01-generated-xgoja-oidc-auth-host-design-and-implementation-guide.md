---
Title: Generated xgoja OIDC Auth Host Design and Implementation Guide
Ticket: XGOJA-ISSUE-82-OIDC-GENERATED-HOST
Status: active
Topics:
    - xgoja
    - auth
    - oidc
    - http
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Generated binary template that should install hostauth service factory
    - Path: cmd/xgoja/internal/specv2/types.go
      Note: xgoja/v2 schema extension point for top-level auth
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Production-proven hand-written host reference implementation
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: OIDC adapter to promote into generated hostauth
    - Path: pkg/xgoja/hostauth/builder.go
      Note: Service factory that should build OIDC handlers
    - Path: pkg/xgoja/hostauth/config.go
      Note: Current generated-host auth config shape and ModeOIDC enum
    - Path: pkg/xgoja/hostauth/resolve.go
      Note: Current ErrOIDCNotImplemented hard stop to remove
    - Path: pkg/xgoja/providers/http/serve.go
      Note: Generated serve command that discovers hostauth and must mount native auth handlers
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-16T18:01:38.160535714-04:00
WhatFor: ""
WhenToUse: ""
---


# Generated xgoja OIDC Auth Host Design and Implementation Guide

## Executive summary

Issue [#82](https://github.com/go-go-golems/go-go-goja/issues/82) asks for production OIDC support in the fully generated `xgoja serve` host. The current production demo at `https://goja-auth.yolo.scapegoat.dev` works because example 19 uses a hand-written Go host shell that calls `keycloakauth.New`, wires stores, mounts `/auth/login`, `/auth/callback`, and `/auth/logout`, and then loads a JavaScript route script. That shape proved the runtime path in production, but it is not the final xgoja developer experience.

The desired final state is a self-contained `xgoja.yaml` example: the user writes route declarations in JavaScript, declares OIDC/session/store configuration in xgoja configuration, runs `xgoja generate` or `xgoja build`, and starts the generated binary with `serve sites demo`. No example-specific Go `cmd/host/main.go` should be required for Keycloak/OIDC login.

The implementation should promote the proven example 19 wiring into reusable generated-host infrastructure:

- extend `hostauth.Config` with an OIDC block;
- remove the `auth.mode=oidc` hard stop in `ResolveConfig`;
- let `hostauth.Builder` construct `keycloakauth.Handlers` and a default appauth-backed `UserNormalizer`;
- extend `hostauth.Services` so HTTP providers can mount OIDC handlers;
- teach the HTTP `serve` command to mount auth endpoints around the generated Express host;
- add a top-level `auth:` block to `xgoja/v2` and the runtime plan so generated binaries can install `hostauth.ServiceFactoryKey` without a custom Go host shell;
- convert the production demo into a generated binary built from `xgoja.yaml`.

The security rule from the production rollout remains central: redirect URLs are derived from an explicit browser-visible `public-base-url` or from an explicit advanced `redirect-url`; they must never be derived from the in-pod listen address.

## Problem statement and scope

### What problem are we solving?

The codebase has the pieces required for a Keycloak-backed auth host, but the pieces are not available through a self-contained generated xgoja binary.

Current production shape:

```text
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
  -> parses OIDC/env flags
  -> builds session/appauth/audit/capability stores
  -> calls keycloakauth.New
  -> mounts /auth/login, /auth/callback, /auth/logout
  -> constructs gojahttp.Host with AuthOptions
  -> evaluates scripts/server.js
  -> serves HTTP
```

Desired generated shape:

```text
xgoja.yaml
  -> declares providers, express module, serve command, route source, auth config
  -> xgoja build/generate embeds route source and runtime plan
  -> generated binary installs hostauth.ServiceFactoryKey from embedded auth plan
  -> HTTP serve command builds hostauth.Services from Glazed values
  -> HTTP serve command mounts OIDC handlers and Express host
  -> browser completes Keycloak login
```

### In scope

- `auth.mode=oidc` support in generated hostauth.
- OIDC config in `hostauth.Config`, Glazed flags/config/env mapping, and generated runtime plan shape.
- OIDC handler construction and mounting in the provider-owned `serve` command.
- A default OIDC user normalizer backed by `appauth.UserStore.UpsertFromOIDC`.
- A generated `xgoja.yaml` example that can replace the hand-written example 19 production demo.
- Unit, integration, smoke, and documentation updates.

### Out of scope for the first implementation

- Enterprise policy engines or YAML authorization policy DSLs. Planned routes remain JavaScript route policy declarations; authorization data remains app-owned stores.
- Multi-replica OIDC transaction storage beyond the existing memory transaction store. The design must leave a seam for durable transaction storage, but first production parity may keep replicas at 1.
- Keycloak MFA policy itself. That belongs to the Keycloak hardening/MFA work; this issue is about generated host plumbing.
- Removing example 19 immediately. The safer migration is to add a generated version, validate it, then retire or simplify the hand-written host after parity.

## Glossary for a new intern

- **xgoja:** The generator/runtime system that turns an `xgoja.yaml` plan into a Go binary or runtime package that embeds JavaScript sources and selected Go modules.
- **goja:** The JavaScript VM used by the project.
- **Express module:** The JavaScript-facing route DSL implemented by Go package `modules/express` and backed by `pkg/gojahttp`.
- **Planned route:** A route declaration that records auth/resource/CSRF/audit intent before a JavaScript handler runs.
- **gojahttp.Host:** The Go-owned HTTP router/runtime bridge that enforces planned routes and invokes JavaScript handlers through the runtime owner.
- **hostauth:** The generated-host auth infrastructure package under `pkg/xgoja/hostauth`. It builds sessions, stores, authorization, audit, and eventually OIDC handlers.
- **appauth:** A small app-owned user/tenant/membership/resource store and authorizer used by demos and as a starting point for monoliths.
- **keycloakauth:** The OIDC adapter that performs discovery, authorization-code + PKCE login, ID-token verification, app-session creation, and logout.
- **Glazed section:** A typed CLI/config/env surface. `hostauth.GlazedConfigSection` exposes flat `--auth-*` fields for generated commands.
- **Host service:** A Go service object installed into an xgoja host before provider command sets or runtime modules are constructed. The HTTP `serve` command discovers `hostauth.ServiceFactoryKey` here.

## Current-state architecture

### The generated host can already expose auth flags

`pkg/xgoja/hostauth/glazed.go` defines the flat `auth` command section. It already exposes `auth-mode` with choices `none`, `dev`, and `oidc` (`pkg/xgoja/hostauth/glazed.go:57`). It also exposes session cookie settings and store settings as `--auth-*` fields (`pkg/xgoja/hostauth/glazed.go:18-44`, `57-75`).

The flattening layer turns nested Go config into CLI-friendly settings (`pkg/xgoja/hostauth/glazed.go:84-127`) and parses values back into `hostauth.Config` (`pkg/xgoja/hostauth/glazed.go:129-173`). The missing piece is not the existence of an `auth` section. The missing piece is that no OIDC-specific fields exist in that section.

### `auth.mode=oidc` is an explicit hard stop

The enum exists:

```go
const (
    ModeNone Mode = "none"
    ModeDev  Mode = "dev"
    ModeOIDC Mode = "oidc"
)
```

Evidence: `pkg/xgoja/hostauth/config.go:10-14`.

But `ResolveConfig` rejects OIDC before store/session services are built:

```go
if mode == ModeOIDC {
    return ResolvedConfig{}, configError("auth.mode", ErrOIDCNotImplemented)
}
```

Evidence: `pkg/xgoja/hostauth/resolve.go:11`, `42-43`. The test `TestResolveConfigRejectsOIDCModeForThisPhase` currently asserts the rejection (`pkg/xgoja/hostauth/resolve_test.go:126-129`).

### The builder can build sessions and authorization, not OIDC handlers

`hostauth.Builder.BuildHostAuthServices` already performs the correct lazy sequence for generated commands:

1. read command-time Glazed values into `hostauth.Config`;
2. resolve defaults and validate config;
3. return empty services for `ModeNone`;
4. build session/audit/appauth/capability stores;
5. build a `sessionauth.Manager`;
6. construct `gojahttp.AuthOptions`;
7. return `hostauth.Services` and closers.

Evidence: `pkg/xgoja/hostauth/builder.go:44-85`.

The returned `Services` type contains `AuthOptions`, `SessionManager`, stores, audit sink, appauth stores, capability store, and closers (`pkg/xgoja/hostauth/services.go:35-50`). It does not contain OIDC handlers or handler mount metadata.

### The HTTP provider can see a hostauth factory before command construction

The HTTP `serve` provider calls `hostauth.LookupServiceFactory(ctx.Host)` while building command sets (`pkg/xgoja/providers/http/serve.go:65`). If a factory exists, `serveAuthSection` adds the generated `auth` Glazed section (`pkg/xgoja/providers/http/serve.go:524-531`).

At command execution time, `serveVerb` calls `buildServeAuthServices`, creates an auth-enabled `gojahttp.Host`, passes it as `go-go-goja-http.host`, and passes concrete auth services as `hostauth.ServicesKey` (`pkg/xgoja/providers/http/serve.go:136-164`, `409-444`).

This is the right high-level architecture. The gap is that `serve` currently only routes to the Express host. It does not know how to mount Go-owned auth handlers around that host.

### The HTTP server is started by the Express module capability

The Express loader starts the HTTP server when the JavaScript module is used. The provider creates or reuses a `gojahttp.Host` (`pkg/xgoja/providers/http/http.go:177-207`), starts a server in `capability.start`, and uses that host as the HTTP handler (`pkg/xgoja/providers/http/http.go:220-260`).

Because server startup is owned by the HTTP capability, OIDC endpoint mounting must be integrated into the host/handler before the server begins accepting traffic. That can happen in one of two ways:

- make `gojahttp.Host` itself aware of native auth endpoint handlers; or
- wrap the host in an outer `http.ServeMux` and make the HTTP capability serve the wrapper.

The second option requires changing `ExternalHostService` or `runtimeEntry` to carry a handler distinct from `*gojahttp.Host`. The first option is simpler if `gojahttp.Host` has or can receive native route mounting support.

### The Keycloak adapter is complete enough for a generated host

`pkg/gojahttp/auth/keycloakauth/keycloakauth.go` exposes exactly the needed runtime surface:

```go
Config{
    IssuerURL,
    ClientID,
    ClientSecret,
    RedirectURL,
    Scopes,
    AfterLoginURL,
    AfterLogoutURL,
    SessionManager,
    UserNormalizer,
    TransactionStore,
}
```

Evidence: `pkg/gojahttp/auth/keycloakauth/keycloakauth.go:20-33`.

`New` validates required settings, discovers the OIDC provider, constructs OAuth2 config, and returns `LoginHandler`, `CallbackHandler`, and `LogoutHandler` (`pkg/gojahttp/auth/keycloakauth/keycloakauth.go:94-139`). The callback path verifies state, exchanges the code with PKCE, verifies the ID token, checks nonce, normalizes the user, creates an app session, and sets the opaque app cookie (`pkg/gojahttp/auth/keycloakauth/keycloakauth.go:175-233`).

The default transaction store is memory-backed (`pkg/gojahttp/auth/keycloakauth/keycloakauth.go:109-111`, `290-321`). This is acceptable for a one-replica demo, but the design must keep a future durable transaction-store seam.

### Example 19 is the reference implementation

The hand-written production example exposes all operator-facing OIDC settings as Glazed flags with env defaults (`examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:89-109`). It resolves redirect URL from `public-base-url` or explicit `redirect-url` (`main.go:160-177`) and rejects insecure public URLs unless explicitly allowed for localhost (`main.go:179-194`).

It constructs Keycloak handlers directly (`main.go:247-267`), creates `gojahttp.Host` with auth options (`main.go:271-281`), loads `server.js` (`main.go:294-300`), then mounts OIDC endpoints and the app host on a `http.ServeMux` (`main.go:303-309`). This is exactly the behavior to promote into generated hostauth/serve.

### Example 21 shows the generated-host seam but still needs a Go shell

`examples/xgoja/21-generated-host-auth/cmd/host/main.go` uses a generated runtime package and injects `hostauth.ServiceFactoryKey` in `Options.ConfigureServices` (`main.go:15-21`). The example proves that the HTTP `serve` command can discover and use auth services, but it still requires a hand-written Go binary wrapper. Its default config uses `ModeDev`, not OIDC (`main.go:43-56`).

The proposed work should remove the need for this Go shell by making `xgoja.yaml` carry hostauth configuration and making generated binary templates install the service factory automatically.

## Gap analysis

| Gap | Current evidence | Required change |
| --- | --- | --- |
| OIDC mode rejects | `ResolveConfig` returns `ErrOIDCNotImplemented` for `ModeOIDC`. | Resolve OIDC config, validate required fields, and return `ResolvedOIDCConfig`. |
| No OIDC config block | `Config` has only `Mode`, `Session`, `Stores`. | Add `OIDC OIDCConfig` to `hostauth.Config` and resolved config. |
| No OIDC Glazed fields | `GlazedSettings` only covers mode/session/stores. | Add `--auth-oidc-*` and `--auth-public-base-url` / `--auth-redirect-url` fields. |
| No OIDC handler in services | `Services` has no Keycloak handlers. | Add `OIDCHandlers` or generic `NativeHandlers` to `Services`. |
| `serve` only creates auth-enabled host | `serveVerb` passes `AuthOptions` but mounts no OIDC routes. | Mount `/auth/login`, `/auth/callback`, `/auth/logout`, and optionally `/auth/session`. |
| Generated binary cannot install factory from YAML | Current schema lacks top-level `auth:`; example 21 injects factory in Go. | Add `auth:` to spec/runtime plan and generated host construction. |
| User normalizer is hand-written | Example 19 normalizer lives inline in `main.go`. | Provide default appauth-backed normalizer and optional extension seam. |
| Production demo is not self-contained | Example 19 has custom `cmd/host/main.go`. | Convert/add generated `xgoja.yaml` example and production Dockerfile path. |

## Proposed architecture

### Architecture diagram

```text
┌────────────────────────────────────────────────────────────────────┐
│ xgoja.yaml                                                         │
│                                                                    │
│ providers: go-go-goja-http                                         │
│ runtime.modules: express                                           │
│ sources: jsverbs server.js                                         │
│ commands: provider.command-set serve                               │
│ auth:                                                              │
│   mode: oidc                                                       │
│   public-base-url: https://goja-auth.yolo.scapegoat.dev            │
│   oidc: issuer/client-id/client-secret/after-login/after-logout    │
│   stores: postgres DSNs/apply-schema                               │
└──────────────────────────────┬─────────────────────────────────────┘
                               │ xgoja build/generate
                               ▼
┌────────────────────────────────────────────────────────────────────┐
│ generated binary main                                              │
│                                                                    │
│ decode runtime plan                                                │
│ app.NewHostWithOptions(... ConfigureServices:                      │
│   SetHostService(hostauth.ServiceFactoryKey,                       │
│     hostauth.NewServiceFactory(hostauth.BuilderOptions{Config}))   │
│ )                                                                  │
│ host.AttachDefaultCommands(root)                                   │
└──────────────────────────────┬─────────────────────────────────────┘
                               │ serve sites demo
                               ▼
┌────────────────────────────────────────────────────────────────────┐
│ HTTP serve command                                                 │
│                                                                    │
│ Lookup hostauth.ServiceFactoryKey                                  │
│ Add auth Glazed section                                            │
│ BuildHostAuthServices                                              │
│   -> stores + sessions + appauth + audit + keycloakauth handlers   │
│ Create gojahttp.Host with AuthOptions                              │
│ Mount native auth handlers + app host                              │
│ Start server through HTTP capability                               │
└──────────────────────────────┬─────────────────────────────────────┘
                               │ browser
                               ▼
┌────────────────────────────────────────────────────────────────────┐
│ Keycloak flow                                                      │
│                                                                    │
│ GET /auth/login -> redirect to Keycloak                            │
│ GET /auth/callback -> verify code, nonce, ID token                 │
│ UserNormalizer -> appauth.UpsertFromOIDC                           │
│ sessionauth.NewSession -> opaque app cookie                        │
│ planned Express routes now see ctx.actor                           │
└────────────────────────────────────────────────────────────────────┘
```

### New xgoja.yaml shape

Add a top-level `auth:` block to `schema: xgoja/v2`. The top-level placement is deliberate: auth host services must exist before provider command sets are constructed. Runtime module config is too late because provider command sets discover `hostauth.ServiceFactoryKey` from `h.Services` during `Host.AttachCommandProvider`.

Example target shape:

```yaml
schema: xgoja/v2
name: goja-auth-host-demo
app:
  name: goja-auth-host-demo
  envPrefix: GOJA_AUTH_HOST_DEMO
  configFile:
    enabled: true
    fileName: goja-auth-host-demo.yaml

go:
  module: github.com/go-go-golems/go-goja-auth-host-demo
  version: "1.26"

providers:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register

runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      config:
        reject-raw-routes: true
        dev-errors: false

sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./scripts
    include:
      - server.js
    language: javascript

commands:
  - id: http-serve
    type: provider.command-set
    name: serve
    mount: serve
    provider: go-go-goja-http
    sources:
      - local-sites

auth:
  mode: oidc
  public-base-url: https://goja-auth.yolo.scapegoat.dev
  session:
    cookie:
      allow-insecure-http: false
      same-site: lax
      path: /
  oidc:
    issuer-url: https://auth.yolo.scapegoat.dev/realms/goja-auth-host-demo
    client-id: goja-auth-host-demo
    client-secret: ${KEYCLOAK_CLIENT_SECRET}
    after-login-url: /
    after-logout-url: /
    scopes: [openid, profile, email]
  stores:
    default:
      driver: postgres
      dsn: ${DATABASE_URL}
      apply-schema: true

artifacts:
  - id: binary
    type: binary
    output: dist/goja-auth-host-demo
    sources:
      - local-sites
```

The `${...}` notation above is conceptual. The implementation should not introduce a new env interpolation engine inside `hostauth.ResolveConfig` unless xgoja already has one at the config layer. The safer first version is:

- defaults may be committed in `xgoja.yaml` when non-secret;
- secrets and DSNs come from Glazed config/env/flags;
- generated commands expose `--auth-oidc-client-secret` and `--auth-default-store-dsn` so existing Glazed middleware can source them from env/config.

### `hostauth.Config` API sketch

```go
type Config struct {
    Mode          Mode          `yaml:"mode" json:"mode"`
    PublicBaseURL string        `yaml:"public-base-url" json:"public-base-url"`
    RedirectURL   string        `yaml:"redirect-url" json:"redirect-url"`
    Session       SessionConfig `yaml:"session" json:"session"`
    Stores        StoresConfig  `yaml:"stores" json:"stores"`
    OIDC          OIDCConfig    `yaml:"oidc" json:"oidc"`
}

type OIDCConfig struct {
    IssuerURL      string   `yaml:"issuer-url" json:"issuer-url"`
    ClientID       string   `yaml:"client-id" json:"client-id"`
    ClientSecret   string   `yaml:"client-secret" json:"client-secret"`
    Scopes         []string `yaml:"scopes" json:"scopes"`
    AfterLoginURL  string   `yaml:"after-login-url" json:"after-login-url"`
    AfterLogoutURL string   `yaml:"after-logout-url" json:"after-logout-url"`
}

type ResolvedConfig struct {
    Mode          Mode
    PublicBaseURL string
    RedirectURL   string
    Session       ResolvedSessionConfig
    Stores        ResolvedStoresConfig
    OIDC          ResolvedOIDCConfig
}

type ResolvedOIDCConfig struct {
    IssuerURL      string
    ClientID       string
    ClientSecret   string
    RedirectURL    string
    Scopes         []string
    AfterLoginURL  string
    AfterLogoutURL string
}
```

`PublicBaseURL` and `RedirectURL` belong at hostauth root rather than inside `OIDCConfig` because future auth modes may also need public origin knowledge. `OIDC.RedirectURL` in the resolved shape is computed from either explicit root `RedirectURL` or `<PublicBaseURL>/auth/callback`.

### Redirect URL validation rules

Production rollout showed that deriving the redirect URL from `--listen` is wrong. The generated resolver should enforce this invariant.

Pseudocode:

```go
func resolveOIDCRedirectURL(cfg Config, allowInsecureHTTP bool) (string, error) {
    if strings.TrimSpace(cfg.RedirectURL) != "" {
        return cfg.RedirectURL, requireAllowedURLScheme("auth.redirect-url", cfg.RedirectURL, allowInsecureHTTP)
    }
    base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
    if base == "" {
        return "", configError("auth.public-base-url", errors.New("required for auth.mode=oidc unless auth.redirect-url is set"))
    }
    if err := requireAllowedURLScheme("auth.public-base-url", base, allowInsecureHTTP); err != nil {
        return "", err
    }
    return base + "/auth/callback", nil
}

func requireAllowedURLScheme(path, raw string, allowInsecureHTTP bool) error {
    parsed, err := url.Parse(raw)
    if err != nil { return configError(path, err) }
    switch parsed.Scheme {
    case "https": return nil
    case "http":
        if allowInsecureHTTP && isLocalhost(parsed.Hostname()) { return nil }
        return configError(path, errors.New("must use https unless insecure localhost is explicitly allowed"))
    default:
        return configError(path, errors.New("must use http or https"))
    }
}
```

Use `auth.session.cookie.allow-insecure-http` as the local-development override. Do not add a second differently named insecure flag unless there is a clear reason.

### OIDC services in `hostauth.Services`

Add fields that let HTTP providers mount native endpoints without importing `keycloakauth` directly if possible.

Option A: expose concrete Keycloak handlers:

```go
type Services struct {
    // existing fields...
    OIDCHandlers *keycloakauth.Handlers
}
```

Option B: expose generic native handlers:

```go
type NativeHandler struct {
    Method  string
    Path    string
    Handler http.Handler
}

type Services struct {
    // existing fields...
    NativeHandlers []NativeHandler
}
```

Prefer Option B for the HTTP provider boundary. `hostauth` can internally use `keycloakauth`, but `serve` only needs to mount method/path/handler triples.

Proposed helper:

```go
func (s *Services) AuthNativeHandlers() []NativeHandler {
    if s == nil { return nil }
    return append([]NativeHandler(nil), s.NativeHandlers...)
}
```

For OIDC mode, the builder returns:

```go
[]NativeHandler{
    {Method: http.MethodGet, Path: "/auth/login", Handler: handlers.LoginHandler()},
    {Method: http.MethodGet, Path: "/auth/callback", Handler: handlers.CallbackHandler()},
    {Method: http.MethodPost, Path: "/auth/logout", Handler: handlers.LogoutHandler()},
    {Method: http.MethodGet, Path: "/auth/logout", Handler: handlers.LogoutHandler()}, // optional parity with adapter
}
```

The adapter accepts GET logout today (`keycloakauth.handleLogout`, `pkg/gojahttp/auth/keycloakauth/keycloakauth.go:235-247`). The production example currently mounts only POST logout. For generated parity, document and test whichever route set is chosen.

### Default appauth-backed UserNormalizer

Issue #82 calls out the need for a `UserNormalizer` strategy. The generated host cannot require user-authored Go for the default case, so the builder should provide a normalizer when appauth users are available.

Pseudocode:

```go
func BuildDefaultOIDCUserNormalizer(stores AppAuthStores) keycloakauth.UserNormalizer {
    return keycloakauth.UserNormalizerFunc(func(ctx context.Context, claims keycloakauth.OIDCClaims) (keycloakauth.UserSession, error) {
        if stores.Users == nil {
            return keycloakauth.UserSession{}, fmt.Errorf("hostauth: appauth user store is required for oidc normalizer")
        }
        user, err := stores.Users.UpsertFromOIDC(ctx, claims.Subject, claims.Email, claims.EmailVerified)
        if err != nil {
            return keycloakauth.UserSession{}, err
        }
        memberships, err := membershipsForUser(ctx, stores.Memberships, user.ID)
        if err != nil {
            return keycloakauth.UserSession{}, err
        }
        return keycloakauth.UserSession{
            UserID:        user.ID,
            Email:         user.Email,
            EmailVerified: user.EmailVerified,
            TenantIDs:     tenantIDs(memberships),
            Claims: map[string]any{
                "keycloakSub": claims.Subject,
                "preferredUsername": claims.PreferredUsername,
                "groups": claims.Groups,
            },
        }, nil
    })
}
```

Important behavior:

- Upsert by stable OIDC `sub`, not email.
- Do not auto-grant admin membership in generic generated hostauth. Example 19 does this for demo convenience, but production generated hostauth should not silently grant roles. Seed data should be explicit route/example setup.
- Include existing memberships as `TenantIDs` so planned routes that check tenant membership can work.

For the production demo, seed membership/resource data through one of these approaches:

1. keep explicit JS/HTTP fixture routes that create app data;
2. add a small optional appauth seed facility under the example, not generic hostauth;
3. keep the current Postgres bootstrap/data seed if already externalized.

Do not build a broad app policy language into this ticket.

### HTTP `serve` mounting strategy

The current HTTP provider server starts by serving a `*gojahttp.Host`. To mount native handlers, use one of these designs.

#### Option A: Add native handler support to `gojahttp.Host`

API sketch:

```go
type NativeRoute struct {
    Method string
    Path   string
    Handler http.Handler
}

func (h *Host) MountNative(method, path string, handler http.Handler) error
```

`Host.ServeHTTP` checks native routes before planned/raw route resolution.

Pros:

- Minimal changes to HTTP provider shape.
- Works in normal serve and hot reload because candidate hosts are still `*gojahttp.Host`.
- Avoids changing `ExternalHostService` and `runtimeEntry` to carry arbitrary handlers.

Cons:

- Extends `gojahttp.Host` with a second route table.
- Must ensure native routes cannot accidentally shadow planned routes without review.

#### Option B: Let HTTP provider serve an outer handler

API sketch:

```go
type ExternalHostService struct {
    Host       *gojahttp.Host
    Handler    http.Handler
    OwnsListen bool
}
```

When auth services exist, create:

```go
mux := http.NewServeMux()
mountNativeHandlers(mux, authServices.NativeHandlers)
mux.Handle("/", serveHost)
ExternalHostService{Host: serveHost, Handler: mux, OwnsListen: true}
```

Pros:

- Keeps `gojahttp.Host` focused on app routes.
- Mirrors example 19 exactly.

Cons:

- Larger HTTP provider refactor.
- Hot reload manager currently expects candidate hosts and serves them directly.
- More surfaces to keep consistent between normal and hot-reload serve.

Decision recommendation: implement Option A first. It is smaller and fits the current provider ownership model. If `gojahttp.Host` already has a native mount primitive not shown in the inspected files, use that instead of inventing a new one.

### BuildHostAuthServices for OIDC

Proposed flow:

```go
func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
    cfg := ConfigFromValues(vals, b.options.Config)
    resolved := ResolveConfig(cfg)

    if resolved.Mode == ModeNone {
        return &Services{Config: resolved}, nil
    }

    stores := BuildStores(ctx, resolved.Stores)
    sessions := BuildSessionManager(resolved.Session, stores.Session, b.options.ActorLoader, b.options.Now)
    auditSink := audit.Sink{Store: stores.Audit}
    authOptions := BuildAuthOptions(sessions, stores, auditSink)

    services := &Services{...existing fields...}

    if resolved.Mode == ModeOIDC {
        normalizer := b.options.OIDCUserNormalizer
        if normalizer == nil {
            normalizer = BuildDefaultOIDCUserNormalizer(services.AppAuth)
        }
        handlers, err := keycloakauth.New(ctx, keycloakauth.Config{
            IssuerURL:      resolved.OIDC.IssuerURL,
            ClientID:       resolved.OIDC.ClientID,
            ClientSecret:   resolved.OIDC.ClientSecret,
            RedirectURL:    resolved.OIDC.RedirectURL,
            Scopes:         resolved.OIDC.Scopes,
            AfterLoginURL:  resolved.OIDC.AfterLoginURL,
            AfterLogoutURL: resolved.OIDC.AfterLogoutURL,
            SessionManager: sessions,
            UserNormalizer: normalizer,
            TransactionStore: transactionStoreFromOptionsOrDefault,
        })
        services.NativeHandlers = oidcNativeHandlers(handlers)
    }

    return services, nil
}
```

### Generated binary host construction

The generated binary currently uses `app.NewRootCommand(app.Options{...})` for simple generated artifacts, or `app.NewHostWithOptions` for adapter/cobra targets. The runtime-package template exposes `Options.ConfigureServices`, which example 21 uses manually.

The generated main template should be extended so when the runtime plan contains auth config it constructs the host with a service factory.

Pseudocode:

```go
func buildRoot() (*cobra.Command, error) {
    registry := providerapi.NewProviderRegistry()
    must(http.Register(registry))
    runtimePlan := decodeRuntimePlan()

    host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{
        EmbeddedJSVerbs: embeddedJSVerbs,
        EmbeddedHelp: embeddedHelp,
        EmbeddedAssets: embeddedAssets,
        ConfigureServices: func(services *app.HostServices) {
            if runtimePlan.Auth != nil {
                must(services.SetHostService(
                    hostauth.ServiceFactoryKey,
                    hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: runtimePlan.Auth.ToHostAuthConfig()}),
                ))
            }
        },
    })

    root := &cobra.Command{Use: runtimePlan.AppName()}
    host.AttachDefaultCommands(root)
    return root, nil
}
```

Do not make the HTTP provider invent the auth config by reading `runtime.modules[].config`; auth service discovery must happen at host construction, before command sets are attached.

## Decision records

### Decision: Add top-level `auth:` to xgoja/v2

- **Context:** The generated `serve` command discovers auth factories during command construction through `CommandSetContext.Host`. Runtime module config is not available early enough.
- **Options considered:** Use runtime module config; use command-set config; add top-level auth config; keep requiring custom Go host shell.
- **Decision:** Add top-level `auth:` to xgoja/v2 and runtime plan, and make generated hosts install `hostauth.ServiceFactoryKey` automatically.
- **Rationale:** This matches the existing host-service architecture and removes the custom host shell, which is the user-visible goal.
- **Consequences:** Requires schema, plan, generator, docs, and tests. Keeps auth as host infrastructure, not JavaScript route policy.
- **Status:** proposed.

### Decision: Promote example 19 wiring, not example 19 binary

- **Context:** Example 19 is production-proven but hand-written. Example 21 is generated-host-shaped but dev-auth-only and still has a Go shell.
- **Options considered:** Keep deploying example 19; modify example 21 only; create a new generated Keycloak example; convert example 19 in place.
- **Decision:** Promote the reusable wiring from example 19 into hostauth/serve, then convert or add a generated Keycloak example that uses `xgoja.yaml` directly.
- **Rationale:** The production behavior remains anchored to proven code, while the final user experience becomes generated.
- **Consequences:** During transition, keep the hand-written example until generated smoke parity passes.
- **Status:** proposed.

### Decision: Default appauth-backed normalizer, no auto-admin grant

- **Context:** OIDC callback must map Keycloak `sub` into an app user/session. Example 19 also grants an admin membership for demo convenience.
- **Options considered:** Require Go normalizer; default to `appauth.UpsertFromOIDC`; add YAML mapping DSL; auto-grant demo admin membership.
- **Decision:** Provide a default `appauth.UpsertFromOIDC` normalizer and derive tenant IDs from existing memberships. Do not auto-grant roles generically.
- **Rationale:** Generated host must work without Go code, but authorization state is application data and should not be silently invented.
- **Consequences:** Examples must seed membership/resource data explicitly.
- **Status:** proposed.

### Decision: Keep secrets in Glazed input layer

- **Context:** OIDC client secret and DSNs are sensitive. `xgoja.yaml` should be commit-safe.
- **Options considered:** Env interpolation in hostauth resolver; committed placeholders; Glazed flags/config/env; custom secret provider.
- **Decision:** Expose flat Glazed fields and let the existing CLI/config/env middleware supply secrets at command time.
- **Rationale:** This matches the current generated command model and avoids building a new interpolation system in auth resolver.
- **Consequences:** Docs and examples must show env/config usage clearly.
- **Status:** proposed.

### Decision: First-class `public-base-url`

- **Context:** Production proved that the in-pod listen address is not the browser origin.
- **Options considered:** Derive callback from listen; require explicit redirect URL only; support public base URL plus redirect override.
- **Decision:** Add `auth.public-base-url` as the normal setting and `auth.redirect-url` as an advanced override.
- **Rationale:** It is safer and clearer for Kubernetes/ingress deployments.
- **Consequences:** `auth.mode=oidc` requires one of these fields and rejects insecure non-local HTTP.
- **Status:** proposed.

## Implementation plan

### Phase 1: Extend hostauth config and validation

Files:

- `pkg/xgoja/hostauth/config.go`
- `pkg/xgoja/hostauth/resolve.go`
- `pkg/xgoja/hostauth/resolve_test.go`
- `pkg/xgoja/hostauth/glazed.go`
- `pkg/xgoja/hostauth/glazed_test.go`

Tasks:

1. Add root `PublicBaseURL` and `RedirectURL` fields.
2. Add `OIDCConfig` and `ResolvedOIDCConfig`.
3. Add `GlazedSettings` fields:
   - `auth-public-base-url`
   - `auth-redirect-url`
   - `auth-oidc-issuer-url`
   - `auth-oidc-client-id`
   - `auth-oidc-client-secret`
   - `auth-oidc-scope`
   - `auth-oidc-after-login-url`
   - `auth-oidc-after-logout-url`
4. Update `FlattenConfig` and `ToConfig`.
5. Replace `TestResolveConfigRejectsOIDCModeForThisPhase` with tests that assert:
   - OIDC requires issuer, client ID, and public base or redirect URL;
   - derived redirect URL is `<public-base-url>/auth/callback`;
   - explicit redirect URL overrides public base;
   - HTTPS is required unless insecure localhost is allowed;
   - `ModeNone` still ignores store DSN requirements.

### Phase 2: Build OIDC services

Files:

- `pkg/xgoja/hostauth/builder.go`
- `pkg/xgoja/hostauth/services.go`
- `pkg/xgoja/hostauth/builder_test.go`
- possibly `pkg/xgoja/hostauth/oidc.go` for new helpers.

Tasks:

1. Add `OIDCUserNormalizer keycloakauth.UserNormalizer` and optional transaction store hook to `BuilderOptions`.
2. Add `NativeHandler` or `OIDCHandlers` to `Services`.
3. Implement `BuildDefaultOIDCUserNormalizer`.
4. In `BuildHostAuthServices`, after session manager creation, call `keycloakauth.New` for `ModeOIDC`.
5. Add native handlers to the returned services.
6. Ensure all partially built stores close if OIDC discovery or handler construction fails.

Tests:

- Fake OIDC provider unit test for builder if practical.
- Builder returns native `/auth/login`, `/auth/callback`, `/auth/logout` handlers for OIDC mode.
- Builder closes stores on OIDC construction error.
- Default normalizer upserts user and projects tenant IDs from memberships.

### Phase 3: Mount native auth handlers in HTTP serve

Files:

- `pkg/gojahttp` host implementation files.
- `pkg/xgoja/providers/http/serve.go`
- `pkg/xgoja/providers/http/http.go`
- `pkg/xgoja/providers/http/serve_test.go`
- `pkg/xgoja/providers/http/http_test.go`

Tasks:

1. Add a native handler mount mechanism to `gojahttp.Host`, or implement equivalent outer handler support.
2. In `serveVerb`, after `gojahttp.NewHost(hostOptionsWithAuth(...))`, mount `authServices.NativeHandlers` before runtime creation.
3. In external host mode, set auth options and mount native handlers on the external host if supported.
4. In hot reload mode, ensure each candidate host has native auth handlers mounted.
5. Add tests that a generated serve command with OIDC services exposes the auth section and routes `/auth/login` to the native handler before planned routes.

Pseudocode:

```go
func applyAuthServicesToHost(host *gojahttp.Host, services *hostauth.Services) error {
    if host == nil || services == nil { return nil }
    host.SetAuthOptions(services.AuthOptions)
    for _, route := range services.NativeHandlers {
        if err := host.MountNative(route.Method, route.Path, route.Handler); err != nil {
            return err
        }
    }
    return nil
}
```

### Phase 4: Add xgoja.yaml auth plan support

Files:

- `cmd/xgoja/internal/specv2/types.go`
- `cmd/xgoja/internal/specv2/specv2_test.go`
- `cmd/xgoja/internal/plan/plan.go` if compiled plan has its own type.
- `cmd/xgoja/internal/generate/templates.go`
- `cmd/xgoja/internal/generate/templates/main.go.tmpl`
- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl`
- `pkg/xgoja/app/runtime_plan.go`
- `pkg/xgoja/app/host.go` or helper file for installing runtime-plan auth.

Tasks:

1. Add `Auth hostauth.Config` or a schema-local equivalent to `specv2.Config`.
2. Add `Auth *hostauth.Config` or runtime-plan-safe equivalent to `app.RuntimePlan`.
3. Render auth config from xgoja config into runtime plan JSON.
4. Update generated binary template imports to include `hostauth` only when auth config is present.
5. Construct the generated host with `ConfigureServices` that sets `hostauth.ServiceFactoryKey`.
6. For runtime-package artifacts, preserve `Options.ConfigureServices` as an override/extension and apply generated auth first or compose both in deterministic order.

Composition rule:

```go
func configureGeneratedServices(runtimePlanAuth *hostauth.Config, userHook func(*app.HostServices)) func(*app.HostServices) {
    return func(services *app.HostServices) {
        if runtimePlanAuth != nil {
            must(services.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: *runtimePlanAuth})))
        }
        if userHook != nil {
            userHook(services)
        }
    }
}
```

### Phase 5: Convert the production demo into self-contained generated serve

Files:

- `examples/xgoja/19-express-keycloak-auth-host/xgoja.yaml`
- `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js`
- `examples/xgoja/19-express-keycloak-auth-host/Makefile`
- `examples/xgoja/19-express-keycloak-auth-host/README.md`
- `Dockerfile.auth-host`
- `.github/workflows/publish-auth-host-image.yaml`
- possibly `deploy/gitops-targets.json`.

Tasks:

1. Add an `xgoja.yaml` with top-level `auth.mode=oidc`, HTTP provider, Express module, `serve` command, and binary artifact.
2. Build generated binary with `xgoja build` instead of `go build ./examples/.../cmd/host`.
3. Replace Dockerfile build target with generated binary artifact.
4. Keep the route script and smoke script.
5. Decide what to do with invite/capability demo endpoints that are currently Go-only in example 19 (`/orgs/o1/invites`, `/org-invites/accept`). Either:
   - move them into planned JavaScript routes backed by exposed modules; or
   - explicitly mark them as out-of-scope for the first generated OIDC parity; or
   - provide a small native handler contribution mechanism for example-only demo extras.

The cleanest first production demo may focus on login/session/planned route auth and leave invite/capability endpoints for a follow-up if they require custom native handlers.

### Phase 6: Update docs

Files:

- `cmd/xgoja/doc/20-hostauth-config-reference.md`
- `cmd/xgoja/doc/22-http-serve-command-reference.md`
- `cmd/xgoja/doc/23-auth-host-production-runbook.md`
- `pkg/doc/32-deploying-an-express-auth-host.md`
- `examples/xgoja/README.md`

Tasks:

1. Remove “OIDC is not implemented” language after implementation.
2. Document the new `auth:` YAML block.
3. Document the new `--auth-oidc-*` flags.
4. Explain migration from hand-written example 19 host to generated binary.
5. Preserve warnings about public URL, HTTPS, secure cookies, and one replica until transaction state is durable.

## Testing strategy

### Unit tests

- `ResolveConfig` OIDC validation tests.
- `GlazedConfigSection` includes all OIDC fields and round-trips config.
- `BuildDefaultOIDCUserNormalizer` maps claims to appauth user/session.
- `BuildHostAuthServices` returns native handlers for OIDC.
- HTTP provider mounts native handlers before app routes.
- Generated template tests assert auth imports and `ConfigureServices` code appear when `auth:` exists.

### Integration tests

- Use `httptest` OIDC provider from `keycloakauth` tests to drive login/callback without a real Keycloak.
- Build a generated host from fixture `xgoja.yaml`, invoke the `serve` command with OIDC config pointed at the test provider, and verify:
  - `/auth/login` redirects to provider;
  - `/auth/callback` creates an app session;
  - protected planned route changes from 401 to 200 with session cookie;
  - unsafe route requires CSRF.

### Example smoke tests

- Local Docker Compose Keycloak smoke for example 19 generated host.
- Public yolo smoke against `https://goja-auth.yolo.scapegoat.dev` after image/GitOps migration.

### Regression tests

- `auth.mode=none` still produces no auth section behavior.
- `auth.mode=dev` still works for example 21.
- Postgres store inheritance remains unchanged.
- Hot reload with auth still mounts native auth handlers after reload.

## Production migration plan

1. Implement OIDC generated-host support behind tests.
2. Add a new generated example, for example `22-generated-keycloak-auth-host`, or convert example 19 only after parity is proven.
3. Run local Compose Keycloak smoke.
4. Build and publish a new auth-host image from generated binary.
5. Update K3s GitOps image tag only; keep existing Keycloak/Vault/Postgres resources.
6. Confirm Argo `Synced Healthy`.
7. Run public smoke with a valid Vault token.
8. Remove or deprecate the hand-written `cmd/host` once generated binary is production-proven.

## Risks and open questions

- **Native handler mounting shape:** Adding native routes to `gojahttp.Host` is likely easiest, but reviewers should confirm it does not violate planned-route invariants.
- **Capability invite endpoints:** Example 19 includes Go-only capability demo endpoints. A purely generated example may need to defer these or expose a generic way for JS routes to issue/redeem capabilities.
- **Durable OIDC transactions:** `keycloakauth` defaults to memory transactions. Keep replicas at 1 until this is durable or sticky.
- **Secret sourcing:** Do not commit client secrets or DSNs in `xgoja.yaml`. The implementation must use Glazed command/config/env input.
- **Generated template coupling:** Adding `hostauth` imports to generated templates must be conditional so non-auth generated binaries do not gain unnecessary imports.
- **Backward compatibility:** Existing runtime plans without `auth` must continue to decode and run.

## References

### GitHub issue

- [go-go-goja issue #82: xgoja support production OIDC in fully generated serve host](https://github.com/go-go-golems/go-go-goja/issues/82)

### Core files

- `pkg/xgoja/hostauth/config.go` — current auth config shape and mode enum.
- `pkg/xgoja/hostauth/resolve.go` — current OIDC hard stop and validation path.
- `pkg/xgoja/hostauth/builder.go` — generated-host service builder.
- `pkg/xgoja/hostauth/services.go` — concrete service bundle returned to providers.
- `pkg/xgoja/hostauth/glazed.go` — flat `--auth-*` command/config/env surface.
- `pkg/xgoja/providers/http/serve.go` — provider-owned `serve` command, auth factory discovery, auth service construction.
- `pkg/xgoja/providers/http/http.go` — Express loader and HTTP server startup.
- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` — OIDC adapter to promote into generated hostauth.
- `pkg/gojahttp/auth/appauth/appauth.go` — default user/session normalizer backing store.
- `cmd/xgoja/internal/specv2/types.go` — xgoja/v2 schema input.
- `pkg/xgoja/app/runtime_plan.go` — embedded runtime plan shape.
- `cmd/xgoja/internal/generate/templates.go` and `templates/main.go.tmpl` — generated binary output path.

### Examples and docs

- `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` — production-proven hand-written OIDC host.
- `examples/xgoja/19-express-keycloak-auth-host/scripts/server.js` — planned-route JavaScript app to keep.
- `examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py` — browser-flow smoke script.
- `examples/xgoja/21-generated-host-auth/cmd/host/main.go` — current generated-host auth seam with Go service injection.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml` — generated runtime-package auth example.
- `cmd/xgoja/doc/20-hostauth-config-reference.md` — current documentation of the OIDC hard stop.
- `cmd/xgoja/doc/22-http-serve-command-reference.md` — current serve/auth factory docs.
- `pkg/doc/32-deploying-an-express-auth-host.md` — production deployment tutorial that should be updated after issue #82 lands.
