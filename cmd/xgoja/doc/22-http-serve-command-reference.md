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

The HTTP provider contributes a `serve` command set for generated xgoja applications. The command keeps a JavaScript verb alive long enough for Express route declarations to register into a Go-owned `gojahttp.Host`. It can run with an xgoja-owned host, an external host supplied by the embedding Go program, or an auth-enabled host built from `hostauth` services.

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
  -> choose xgoja-owned host or external host
  -> create runtime with per-runtime host services
  -> execute the selected jsverb so it registers routes
  -> wait until context cancellation
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

If the embedding host installs `hostauth.ServiceFactoryKey`, the HTTP `serve` provider adds an `auth` Glazed section and later builds `hostauth.Services` after values are parsed.

```text
Command construction time:
  hostauth.LookupServiceFactory(ctx.Host)
  -> add auth section to the command schema

Command execution time:
  parse values
  -> factory.Build(ctx, values)
  -> hostOptionsWithAuth(httpSettings, authServices)
  -> gojahttp.NewHost(...)
```

This two-step design lets the command show auth flags without opening databases or resolving runtime settings during help generation.

`auth.mode=oidc` is not implemented in generated hostauth yet. Use the direct Keycloak example host when you need production-shaped OIDC today.

## External host integration

A Go program can provide an existing `*gojahttp.Host` with `httpprovider.ExternalHostService`. The provider then registers Express routes into that host instead of creating a new one.

Use an external host when a Go application already owns the `http.ServeMux`, WebSocket upgrades, middleware order, or auth handlers. This is the pattern used by hand-composed auth hosts.

## Hot reload

The serve command can run in hot-reload mode. The hot-reload section is named `http-serve` and includes fields such as:

| Field | Meaning |
| --- | --- |
| `hot-reload` | Enables blue/green route reload. |
| `watch` | Adds explicit watch roots. |
| `poll` | Uses polling instead of filesystem notifications. |
| `debounce` | Debounces reload events. |
| `smoke-path` | Optional path used as a reload smoke check. |

Hot reload creates a manager that can swap route snapshots while the listener stays alive. It still uses signal-aware shutdown for SIGINT/SIGTERM.

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
| `auth` flags are missing | No `hostauth.ServiceFactoryKey` was installed in host services. | Inject the factory before building the command set. |
| `auth.mode=oidc` fails | Generated hostauth OIDC is not implemented. | Use example 19 direct OIDC host or implement issue #82. |
| Hot reload starts but reload never triggers | Watch roots do not include the edited files. | Add explicit `--watch` roots or fix source roots. |

## See also

- `xgoja help xgoja-v2-reference`
- `xgoja help provider-runtime-config-and-host-services`
- `xgoja help hostauth-config-reference`
- `xgoja help express-auth-host-integration-guide`
