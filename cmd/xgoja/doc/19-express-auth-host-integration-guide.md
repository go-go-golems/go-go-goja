---
Title: "Express auth host integration guide"
Slug: express-auth-host-integration-guide
Short: "Compose a Go HTTP host with planned Express auth routes, OIDC handlers, stores, and graceful shutdown."
Topics:
- xgoja
- gojahttp
- express
- auth
- keycloak
- net-http
Commands:
- xgoja
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

A planned Express route describes security intent. The Go host supplies the infrastructure that makes the plan enforceable: session lookup, CSRF verification, resource resolution, authorization, audit logging, and identity-provider callbacks. This guide explains how to compose those pieces into a host binary that can run JavaScript route declarations while keeping the security boundary in Go.

Use this page when you are promoting an Express-auth example into a host application, or when you are reading generated route code and need to understand where authentication actually happens.

## The host owns infrastructure

JavaScript route code should not own login, cookies, database handles, or authorization services. It declares route-level requirements:

```javascript
app.patch("/orgs/:orgID/projects/:projectID")
  .auth(express.user().required())
  .resource(express.resource("project").idFromParam("projectID").tenantFromParam("orgID").mustExist())
  .csrf()
  .allow("project.update")
  .audit("project.update")
  .handle((ctx, res) => res.json({ updated: ctx.resource.id }))
```

The Go host turns that declaration into a request pipeline. It decides whether the session is valid, whether the CSRF token matches, which app resource is being addressed, whether the actor can perform the action, and where audit records are stored.

```text
HTTP request
  -> http.ServeMux
  -> gojahttp.Host route lookup
  -> planned route pipeline
  -> Authenticator
  -> CSRF verifier for unsafe routes
  -> Resource resolver
  -> Authorizer
  -> Audit sink
  -> JavaScript handler
```

This split is the main reason planned auth exists. Route files remain concise, and host applications keep deployment-specific infrastructure in Go.

## Compose `gojahttp.NewHost`

The central object is `*gojahttp.Host`. A production-shaped auth host constructs it with `RejectRawRoutes: true` and a complete `AuthOptions` block.

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:             true,
    RejectRawRoutes: true,
    Auth: gojahttp.AuthOptions{
        Authenticator: sessions,
        CSRF:          sessions,
        Resources:     appauth.Resolver{Store: appStores.store},
        Authorizer:    appauth.Authorizer{Memberships: appStores.store},
        Audit:         auditSink,
    },
})
```

Each field has a precise responsibility.

| Field | Runtime responsibility |
| --- | --- |
| `Authenticator` | Converts a request into an actor/session or rejects it. |
| `CSRF` | Verifies unsafe requests that declare `.csrf()`. |
| `Resources` | Loads route-declared resources from params, query, body, or constants. |
| `Authorizer` | Checks the actor, action, and resolved resource. |
| `Audit` | Records security-relevant route outcomes. |

`RejectRawRoutes: true` keeps the route surface reviewable. JavaScript code can register planned routes and approved static/generic mounts, but accidental raw route handlers are rejected instead of bypassing the planned-auth pipeline.

## Mount the Express module

The host is passed into the Express registrar before the runtime is created. This lets JavaScript call `require("express")` and register routes into the Go-owned host.

```go
factory, err := engine.NewRuntimeFactoryBuilder().
    UseModuleMiddleware(engine.MiddlewareOnly("timer")).
    WithModules(express.NewRegistrar(host)).
    Build()
if err != nil {
    return err
}

rt, err := factory.NewRuntime(
    engine.WithStartupContext(ctx),
    engine.WithLifetimeContext(ctx),
)
if err != nil {
    return err
}
defer rt.Close(ctx)

host.SetRuntime(rt.Owner)
```

The runtime owner matters because request handlers later call back into JavaScript. `goja.Runtime` access must remain serialized through the runtime owner; the host does not call the VM directly from arbitrary HTTP goroutines.

## Load the route script

The example 19 host reads a JavaScript route file and evaluates it once during startup.

```go
data, err := os.ReadFile(cfg.Script)
if err != nil {
    return err
}

_, err = rt.Owner.Call(ctx, "load-keycloak-auth-example", func(_ context.Context, vm *goja.Runtime) (any, error) {
    _, runErr := vm.RunString(string(data))
    return nil, runErr
})
if err != nil {
    return err
}
```

If route registration fails, the process should fail before it starts accepting traffic. That is preferable to serving a partially registered route set.

## Mount OIDC and session endpoints

OIDC login/callback/logout handlers are regular Go HTTP handlers. They are mounted beside the planned-route host on the same `http.ServeMux`.

```go
keycloakHandlers, err := keycloakauth.New(ctx, keycloakauth.Config{
    IssuerURL:      cfg.Issuer,
    ClientID:       cfg.ClientID,
    ClientSecret:   cfg.ClientSecret,
    RedirectURL:    cfg.RedirectURL,
    AfterLoginURL:  cfg.AfterLoginURL,
    AfterLogoutURL: cfg.AfterLogoutURL,
    SessionManager: sessions,
    UserNormalizer: normalizer,
})
if err != nil {
    return err
}

mux := http.NewServeMux()
mux.Handle("GET /auth/login", keycloakHandlers.LoginHandler())
mux.Handle("GET /auth/callback", keycloakHandlers.CallbackHandler())
mux.Handle("POST /auth/logout", keycloakHandlers.LogoutHandler())
mux.Handle("GET /auth/session", sessionHandler(sessions))
mux.Handle("/", indexPage(host))
```

The callback handler exchanges the OIDC code, normalizes claims into an app user, and creates a server-side app session. The browser receives an opaque app session cookie, not the Keycloak tokens.

## Configure public URLs explicitly

The bind address is not the browser origin. In Kubernetes, the process listens on `:8080`, while the user sees `https://goja-auth.yolo.scapegoat.dev`. OIDC redirect URLs must use the browser-visible origin.

```go
func resolveRedirectURL(settings serveSettings) (string, error) {
    if settings.RedirectURL != "" {
        return settings.RedirectURL, requireAllowedURLScheme(settings.RedirectURL, settings.AllowInsecureHTTP)
    }
    publicBase := strings.TrimRight(settings.PublicBaseURL, "/")
    if publicBase == "" {
        return "", errors.New("public-base-url or redirect-url is required")
    }
    if err := requireAllowedURLScheme(publicBase, settings.AllowInsecureHTTP); err != nil {
        return "", err
    }
    return publicBase + "/auth/callback", nil
}
```

Use `--public-base-url` for normal deployments. Use `--redirect-url` only when the callback path is intentionally not `<public-base-url>/auth/callback`. Allow HTTP only for localhost development with `--allow-insecure-http`.

## Use signal-aware shutdown

A host that starts `http.Server` should also stop it on SIGINT and SIGTERM. This affects local smoke tests and Kubernetes rollout behavior.

```go
func serveWithShutdown(ctx context.Context, server *http.Server) error {
    serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
    defer stop()

    errCh := make(chan error, 1)
    go func() {
        if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
            return
        }
        errCh <- nil
    }()

    select {
    case err := <-errCh:
        return err
    case <-serveCtx.Done():
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(shutdownCtx); err != nil {
        return err
    }
    return <-errCh
}
```

A smoke test that hangs after successful login is often a shutdown problem, not an auth problem. The example 19 cleanup instrumentation caught exactly this case.

## Minimal production checklist

Before deploying an auth host, confirm these conditions:

- The route script loads successfully before the HTTP server starts.
- The host uses `RejectRawRoutes: true` unless raw routes are deliberately allowed.
- All four stores have an explicit backend choice: session, audit, appauth, and capability.
- `public-base-url` is set to the HTTPS browser origin.
- `redirect-url` matches the Keycloak client's valid redirect URI.
- `allow-insecure-http` is false behind ingress.
- SIGTERM triggers `http.Server.Shutdown` within a bounded timeout.
- `/healthz` is public and returns HTTP 200 without requiring login.
- A browser-flow smoke test exercises login, session, CSRF, authorization, and logout.

## Troubleshooting

| Problem | Cause | Fix |
| --- | --- | --- |
| Keycloak returns invalid redirect URI | `public-base-url` or `redirect-url` does not match the client configuration. | Set `PUBLIC_BASE_URL=https://...` and configure `<public-base-url>/auth/callback` in Keycloak. |
| Login succeeds but authenticated routes still return 401 | The app session cookie is missing or rejected. | Check HTTPS, cookie security settings, and `allow-insecure-http`. Do not use insecure HTTP outside localhost. |
| Smoke passes functionally but process cleanup hangs | Server ignores SIGTERM. | Use `signal.NotifyContext` and bounded `server.Shutdown`. |
| Planned route returns 403 for unsafe mutation | Missing or stale CSRF token. | Fetch `/auth/session`, read `csrfToken`, and send `X-CSRF-Token`. |
| Route loads but authorization always fails | App membership/resource store is not seeded or persisted. | Check appauth store schema, seed data, and OIDC user normalizer. |

## See also

- `xgoja help go-planned-auth-api`
- `xgoja help express-route-auth-requirements`
- `xgoja help generated-auth-javascript-apis`
- `xgoja help programmatic-auth-javascript-apis`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-stores-reference`
- `xgoja help auth-host-production-runbook`
- `goja-repl help auth-module-guide`
- `goja-repl help express-auth-user-guide`
- `goja-repl help deploying-an-express-auth-host`
