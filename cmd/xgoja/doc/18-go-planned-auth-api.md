---
Title: "Go planned auth API"
Slug: go-planned-auth-api
Short: "How Go hosts register planned auth routes without JavaScript Express."
Topics:
- xgoja
- gojahttp
- auth
- net-http
Commands:
- xgoja
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Go planned auth API

`gojahttp` planned auth is available to Go hosts directly. JavaScript Express routes, generated-host routes, and Go handlers all use the same `RoutePlan` contract: the route declares who may enter, which resources are loaded, whether CSRF is required, which action is authorized, and which audit event is emitted. The host remains responsible for authentication, authorization, sessions, CSRF, and audit.

Use the Go API when a traditional Go program wants the planned auth framework without writing an Express module. Use JavaScript Express when the route behavior should stay in scripts.

## Core types

| Type or function | Purpose |
| --- | --- |
| `gojahttp.RoutePlan` | Shared planned-route contract used by JavaScript and Go routes. |
| `gojahttp.SecureContext` | Immutable-by-convention handler input populated after auth, resource resolution, and authorization. |
| `gojahttp.PlannedHTTPHandler` | Go handler signature for planned routes. |
| `(*gojahttp.Host).RegisterPlannedHTTP` | Low-level registration API for generated code or explicit `RoutePlan` construction. |
| `gojahttp.NewApp(host)` | Fluent Go route builder for application code. |
| `gojahttp.PlannedMiddleware` | Standard `net/http` middleware for programs that already route with `http.ServeMux` or another router. |

`SecureContext` exposes the same security result that JavaScript planned handlers receive through their secure context: route params, query/body DTO data, session data, the authenticated actor, the primary resource, resolved resources by name, the authorized action, and the original plan.

## Configure host-owned services

A planned route only declares intent. The host provides concrete services through `gojahttp.AuthOptions` and `gojahttp.SessionOptions`:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Sessions: gojahttp.SessionOptions{
        CookieName: "sid",
        // Store, signer, and cookie options go here.
    },
    Auth: gojahttp.AuthOptions{
        Authenticator: myAuthenticator,
        Resources:     myResourceResolver,
        Authorizer:    myAuthorizer,
        CSRF:          myCSRFVerifier,
        Audit:         myAuditSink,
    },
})
```

The authenticator decides who the request is. The resource resolver loads domain resources from plan-declared value sources. The authorizer decides whether the actor may perform the route action on the resolved resource. The handler runs only after those checks pass.

## Fluent Go route builder

The fluent builder is the normal hand-written Go API. It stages route declaration so callers choose `Public()` or `Auth(...)` before a handler can be registered.

```go
app := gojahttp.NewApp(host)

err := app.Get("/healthz").
    Public().
    HandleJSON(func(ctx context.Context, sec *gojahttp.SecureContext) (any, error) {
        return map[string]bool{"ok": true}, nil
    })
if err != nil {
    return err
}

err = app.Get("/orgs/:orgID/projects/:projectID").
    Auth(gojahttp.User().Required()).
    Resource(
        gojahttp.Resource("project").
            IDFromParam("projectID").
            TenantFromParam("orgID").
            MustExist(),
    ).
    Audit("project.read").
    Allow("project.read").
    HandleJSON(func(ctx context.Context, sec *gojahttp.SecureContext) (any, error) {
        return map[string]any{
            "actor":   sec.Actor.ID,
            "project": sec.Resource.ID,
            "tenant":  sec.Resource.TenantID,
        }, nil
    })
if err != nil {
    return err
}
```

Builder rules:

- `Public()` routes do not require an action.
- `Auth(...)` routes must call `Allow(action)` before `Handle` or `HandleJSON`.
- `CSRF()` can be declared on authenticated routes before or after `Allow(...)`.
- `Resource(...)` may be called multiple times. The first resolved resource is available as `sec.Resource`; all resources are also available through `sec.Resources`.
- `HandleJSON` sets `Content-Type: application/json` and encodes the returned value. Use `Handle` when the route needs direct `http.ResponseWriter` control.

## Low-level registration

Generated code or framework adapters can build `RoutePlan` directly and register a Go handler:

```go
err := host.RegisterPlannedHTTP(gojahttp.RoutePlan{
    Method:  http.MethodPost,
    Pattern: "/projects/:projectID/archive",
    Security: gojahttp.SecuritySpec{
        Mode:     gojahttp.SecurityModeUser,
        Required: true,
    },
    Resources: []gojahttp.ResourceSpec{
        gojahttp.Resource("project").IDFromParam("projectID").MustExist(),
    },
    CSRF:   gojahttp.CSRFSpec{Required: true},
    Action: "project.archive",
    Audit:  gojahttp.AuditSpec{Event: "project.archive"},
}, func(ctx context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, r *http.Request) error {
    // Domain mutation runs after authentication, CSRF, resource resolution,
    // authorization, and audit-start bookkeeping.
    w.WriteHeader(http.StatusNoContent)
    return nil
})
```

This API is intentionally verbose. Prefer `NewApp(host)` for hand-written application routes.

## Standard net/http middleware

Use `PlannedMiddleware` when another router owns request dispatch. The route plan still uses gojahttp `:param` value-source names; `ParamFunc` maps those names to values from the active router.

With Go 1.22 `http.ServeMux`:

```go
handler, err := gojahttp.PlannedMiddleware(gojahttp.MiddlewareOptions{
    Auth: authOptions,
    Sessions: sessionOptions,
    ParamFunc: func(r *http.Request, name string) string {
        return r.PathValue(name)
    },
}, gojahttp.RoutePlan{
    Method:   http.MethodGet,
    Pattern:  "/orgs/:orgID/projects/:projectID",
    Security: gojahttp.User().Required(),
    Resources: []gojahttp.ResourceSpec{
        gojahttp.Resource("project").
            IDFromParam("projectID").
            TenantFromParam("orgID").
            MustExist(),
    },
    Action: "project.read",
}, func(ctx context.Context, sec *gojahttp.SecureContext, w http.ResponseWriter, r *http.Request) error {
    return json.NewEncoder(w).Encode(sec.Resource)
})
if err != nil {
    return err
}

mux := http.NewServeMux()
mux.Handle("GET /orgs/{orgID}/projects/{projectID}", handler)
```

If `ParamFunc` is omitted, the middleware falls back to gojahttp's own `:param` path matcher and can be served directly for matching paths.

## Choosing the API

| Use case | Recommended API |
| --- | --- |
| Hand-written Go app that can use `gojahttp.Host` as router | `gojahttp.NewApp(host)` |
| Generated Go code that already has a serialized plan | `host.RegisterPlannedHTTP(plan, handler)` |
| Existing `net/http` app with its own mux/router | `gojahttp.PlannedMiddleware(...)` |
| JavaScript-authored behavior | Express planned route builder (`.public()`, `.auth(...)`, `.allow(...)`, `.handle(...)`) |

## Security model

Do not put security decisions in the handler. Handlers may rely on `SecureContext` because it is populated after host-owned checks. Keep these responsibilities in Go services:

- identity proof and session/token validation in `Authenticator`,
- resource lookup and tenant binding in `ResourceResolver`,
- permission decisions in `Authorizer`,
- mutation protection in `CSRFVerifier`,
- audit persistence in `AuditSink`.

The planned route declaration is allowed to say “this route requires a user and action `project.read`”. It should not embed secrets, store DSNs, OIDC client configuration, or application policy rules.
