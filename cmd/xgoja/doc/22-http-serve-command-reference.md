---
Title: "HTTP serve command reference"
Slug: http-serve-command-reference
Short: "How the xgoja HTTP provider builds serve commands, mounts Express routes, and supports hot reload."
Topics:
- xgoja
- http
- express
- serve
- hot-reload
Commands:
- xgoja
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The HTTP provider contributes a `serve` command set for generated xgoja applications. The command owns the listener, `http.Server`, top-level mux, route-registration readiness, signal handling, and graceful shutdown. Express is only a JavaScript route-registration DSL; `app.listen()` is not supported in xgoja and users should run `xgoja serve` instead. The command can run with an xgoja-owned host, an external host supplied by the embedding Go program, or an auth-enabled host built from `hostauth` services.

## Where the command comes from

The HTTP provider registers two things:

```go
providerapi.Module{
    Name: "express",
    NewModuleFactory: ...,
}

providerapi.CommandSetProvider{
    Name:         "serve",
    DefaultMount: "serve",
    Description:  "Serve JavaScript verb-backed HTTP sites",
    NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        return newServeCommandSet(ctx)
    },
}
```

The module lets JavaScript `require("express")`. The command set creates CLI commands such as `serve sites demo`, scoped to jsverb sources declared on the command.

## Source scope

`serve` requires command-scoped runtime sources. It does not scan every source in the binary. This is intentional: a serve command should know exactly which JavaScript files can register routes.

```yaml
commands:
  - id: http-serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources:
      - site
```

At runtime, `serveCommandJSVerbSources` reads only the configured source IDs. If no jsverb sources are configured for the command, command construction fails.

## Runtime construction

The normal non-hot-reload path does the following work:

```text
serve command invoked
  -> parse Glazed HTTP/auth/hot-reload sections
  -> find selected jsverb
  -> build hostauth services if a hostauth factory is installed
  -> choose xgoja-owned app host or external host
  -> build the top-level handler/mux
       -> mount native auth handlers first
       -> mount serve/hot-reload helper routes
       -> mount the app host at /
  -> create runtime with per-runtime host services
  -> execute the selected jsverb so it registers routes
  -> net.Listen(http.listen)
  -> http.Server.Serve(listener)
  -> graceful Shutdown on context cancellation or SIGINT/SIGTERM
```

The route script must register routes while the runtime starts. The command then keeps the runtime alive; if the runtime exits immediately, the HTTP host would not have a live JavaScript owner for request callbacks.

## HTTP section

The provider exposes an `http` section for xgoja-owned HTTP modules.

| Field | Default | Meaning |
| --- | --- | --- |
| `enabled` | `true` | Start the xgoja-owned HTTP server for modules such as Express. |
| `listen` | `127.0.0.1:8787` | Listen address for the xgoja-owned server. |
| `dev-errors` | `false` | Return development JavaScript error details. |
| `reject-raw-routes` | `true` | Reject matched raw routes; planned routes and static mounts are unaffected. |

Generated CLI flags usually appear as prefixed fields such as `--http-listen` depending on command construction.

## Hostauth integration

If the generated runtime plan or embedding host installs `hostauth.ServiceFactoryKey`, the HTTP `serve` provider adds an `auth` Glazed section and later builds `hostauth.Services` after values are parsed.

```text
Command construction time:
  hostauth.LookupServiceFactory(ctx.Host)
  -> add auth section to the command schema

Command execution time:
  parse values
  -> factory.Build(ctx, values)
  -> hostOptionsWithAuth(httpSettings, authServices)
  -> gojahttp.NewHost(...)
  -> buildServeHandler(appHost, authServices)
```

This two-step design lets the command show auth flags without opening databases, discovering OIDC providers, or resolving runtime settings during help generation.

When `auth.mode=oidc`, `hostauth.Services.NativeHandlers` contains Go-owned OIDC routes. `serve` mounts them before the JavaScript app fallback:

```text
GET  /auth/login
GET  /auth/callback
GET  /auth/logout
POST /auth/logout
```

Use `--auth-oidc-public-base-url` for the browser-visible HTTPS origin; the callback defaults to `<public-base-url>/auth/callback`. Use `--auth-oidc-redirect-url` only as an advanced callback override. See `examples/xgoja/21-generated-host-auth` for a generated binary smoke fixture.

## External host integration

A Go program can provide an existing `*gojahttp.Host` with `httpprovider.ExternalHostService`. The provider then registers Express routes into that host instead of creating a new one.

Use an external host when a Go application already owns the app host or needs custom Go integrations. The `serve` command still owns the listener and top-level lifecycle; external-host mode only swaps the `gojahttp.Host` used for Express route registration.

## Hot reload

The serve command can run in hot-reload mode. The hot-reload section is named `http-serve` and includes fields such as:

| Field | Meaning |
| --- | --- |
| `hot-reload` | Enables blue/green route reload. |
| `watch` | Adds explicit watch roots. |
| `poll` | Uses polling instead of filesystem notifications. |
| `debounce` | Debounces reload events. |
| `smoke-path` | Optional path used as a reload smoke check. |

Hot reload creates one stable listener/top-level mux. Native auth handlers stay mounted in Go, and the hot-reload manager swaps only app route snapshots. It still uses signal-aware shutdown for SIGINT/SIGTERM.

## Minimal generated spec

```yaml
schema: xgoja/v2
name: demo

providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http

runtime:
  modules:
    - provider: http
      name: express
      as: express

sources:
  - id: site
    kind: jsverbs
    from:
      dir: .
    include:
      - server.js

commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources:
      - site
```

For generated OIDC login, add top-level `auth:`:

```yaml
auth:
  mode: oidc
  session:
    cookie:
      allow-insecure-http: false
  stores:
    default:
      driver: postgres
      dsn: postgres://user:pass@postgres:5432/app?sslmode=disable
      apply-schema: true
  oidc:
    issuer-url: https://auth.example.test/realms/demo
    client-id: demo-app
    public-base-url: https://demo.example.test
```

Run the generated command with:

```bash
./dist/demo serve sites demo --http-listen 127.0.0.1:8787
```

The exact command path depends on the jsverb namespace and command-set mount.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Command construction says serve requires runtime sources | The command set has no jsverb source IDs. | Add `sources:` to the provider command-set entry. |
| Express routes do not register | The selected jsverb did not require/register the route script. | Check the jsverb body and source filters. |
| `auth` flags are missing | No top-level `auth:` block exists and no `hostauth.ServiceFactoryKey` was installed in host services. | Add `auth:` to `xgoja.yaml` or inject the factory before building the command set. |
| `app.listen()` errors | Express is registration-only in xgoja. | Run the generated `serve` command; do not start listeners from JavaScript. |
| `/auth/login` is handled by the app route instead of OIDC | Native handlers were not built or mounted. | Check `--auth-mode oidc`, OIDC config, and startup logs/errors. |
| Hot reload starts but reload never triggers | Watch roots do not include the edited files. | Add explicit `--watch` roots or fix source roots. |

## See also

- `xgoja help xgoja-v2-reference`
- `xgoja help provider-runtime-config-and-host-services`
- `xgoja help hostauth-config-reference`
- `xgoja help express-auth-host-integration-guide`
