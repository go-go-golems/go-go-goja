---
Title: Express Auth Examples
Slug: express-auth-examples
Short: Run the dev-auth, generated-host, and Keycloak examples for planned Express authentication.
Topics:
- http
- express
- auth
- examples
- keycloak
- goja
Commands:
- xgoja
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

The Express auth examples show the same planned-route API at three levels of operational realism. The dev-auth example runs without external services and is best for learning the host interfaces. The generated-host example demonstrates the runtime-package `ConfigureServices` seam with reusable `hostauth` session/store builders. The Keycloak example uses Docker Compose and proves the production-shaped browser login flow with OIDC Authorization Code + PKCE, app sessions, CSRF, app-owned authorization, and audit.

## Choose the right example

Start with the dev-auth example when you want to understand planned routes or test route declarations quickly. It keeps all identity, sessions, resources, authorization, and audit state in memory so you can run a complete smoke without Keycloak or a database.

Use the generated-host example when you want a generated runtime package and provider-owned `serve` commands, but still want the Go host to own sessions, stores, cookies, and app authorization services.

Use the Keycloak example when you need to validate the production boundary: Keycloak authenticates the browser user, the Go host creates an opaque app session, and planned routes authorize against app-owned resources and memberships.

| Example | Directory | Best for | External services |
| --- | --- | --- | --- |
| Dev auth host | `examples/xgoja/18-express-auth-host` | Local route-authoring, demos, fast smoke tests. | None. |
| Generated-host auth | `examples/xgoja/21-generated-host-auth` | Runtime-package host-service injection, memory/SQLite store demos. | None. |
| Keycloak auth host | `examples/xgoja/19-express-keycloak-auth-host` | Production-shaped OIDC/session/authz smoke. | Docker Compose Keycloak. |

## Dev-auth smoke

The dev-auth smoke starts the Go host in smoke mode and exits after exercising public routes, login, authenticated routes, CSRF denial, CSRF success, missing resources, audit events, logout, and post-logout denial.

```bash
make -C examples/xgoja/18-express-auth-host smoke
```

A successful run prints status lines like this:

```text
ok public health            200
ok me before login          401
ok login                    200
ok session after login      200
ok project missing csrf     403
ok project update           200
ok logout                   204
{"auditEvents":10,"status":"PASS"}
```

The host uses `devauth` because the goal is a self-contained development backend. It still wires the same `gojahttp.AuthOptions` interface shape that production hosts use.

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Dev:             true,
    RejectRawRoutes: true,
    Auth: gojahttp.AuthOptions{
        Authenticator: kit,
        CSRF:          kit,
        Resources:     kit,
        Authorizer:    kit,
        Audit:         kit,
    },
})
```

## Generated-host auth smoke

The generated-host smoke regenerates a runtime package, starts the package host's provider-owned `serve sites demo` command, checks public routes, and verifies that a protected planned route returns `401 Unauthorized` without a session cookie. The host injects a lazy `hostauth.ServiceFactoryKey` through `xgojaruntime.Options.ConfigureServices`; JavaScript still only declares route intent.

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

By default the example uses memory stores. To demonstrate persistent local stores, set `XGOJA_AUTH_STORE=sqlite` and provide the DSN through the environment:

```bash
XGOJA_AUTH_STORE=sqlite \
XGOJA_AUTH_SQLITE_DSN=/tmp/xgoja-generated-host-auth.sqlite \
go run ./examples/xgoja/21-generated-host-auth/cmd/host \
  serve sites demo --http-listen 127.0.0.1:8787
```

The DSN is intentionally an environment value, not committed YAML. OIDC/Keycloak generated-host config remains deferred; use the Keycloak host example below for the current production-shaped browser login smoke.

## Keycloak smoke

The Keycloak smoke starts Docker Compose Keycloak, waits for the OIDC discovery document, builds and starts the Go host, drives the Keycloak login form with the demo account, verifies planned routes with the resulting app session, and tears Keycloak down again.

```bash
make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

A successful run includes the login and CSRF assertions:

```text
ok keycloak form login          200
ok login redirected to host     http://127.0.0.1:8790/
ok me after login               200
ok session after login          200
ok project missing csrf         403
ok project update               200
ok logout                       204
{"status": "PASS", "actorId": "user:...", "csrfChecked": true}
```

The imported demo realm uses these local credentials:

```text
realm:    goja-demo
client:   goja-app
username: demo@example.test
password: demo-password
```

If port `18080` is busy, move Keycloak to another host port. The Makefile derives the issuer URL from `KEYCLOAK_PORT`.

```bash
KEYCLOAK_PORT=18081 make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

If you want to inspect Keycloak after a failed run, keep the container running.

```bash
KEEP_KEYCLOAK=1 make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

## What both examples prove

Both examples load JavaScript planned routes that look like normal Express-style route declarations but require explicit security stages before `.handle(...)`.

```javascript
app.patch("/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(
    express.resource("project")
      .idFromParam("projectId")
      .tenantFromParam("orgId")
      .mustExist()
  )
  .csrf()
  .allow("project.update")
  .audit("project.updated")
  .handle((ctx, res) => {
    const project = ctx.resource("project")
    res.json({ updated: project.id })
  })
```

The JavaScript route names the action and resource binding. The Go host authenticates the request, resolves the resource, verifies CSRF when required, authorizes the action, emits audit outcomes, and only then calls the JavaScript handler.

## Local Keycloak cookie detail

The Keycloak smoke uses a small Python standard-library script instead of Playwright or Selenium. Keycloak dev mode may set `Secure; SameSite=None` login cookies even when served on `http://127.0.0.1`; Python's `CookieJar` correctly refuses to send those cookies over plain HTTP. The smoke explicitly sends the Keycloak login cookies on the local form POST so the no-browser smoke matches local browser behavior.

That workaround is limited to the smoke client. It does not change the Go host or production authentication behavior.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Dev-auth smoke returns 401 after login | The session cookie was not preserved between requests. | Use the provided smoke target or a `curl` cookie jar with `-c` and `-b`. |
| Project update returns 403 | The planned route declares `.csrf()` and the request lacks the app session CSRF token. | Fetch the session endpoint and send `X-CSRF-Token`. |
| Project update returns 404 | The route's tenant/resource parameters do not match seeded data. | Use `/orgs/o1/projects/p1` in the examples or update the seeded store. |
| Keycloak smoke times out waiting for discovery | Docker is still pulling/starting Keycloak or the container failed. | Run `docker compose -f examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml logs keycloak`. |
| `address already in use` | A previous host or Keycloak process is still using the default port. | Stop the old process, use `KEYCLOAK_PORT=18081`, or override the host `LISTEN` address. |
| Keycloak callback fails | Issuer, redirect URI, or client settings no longer match the imported realm. | Use the default Makefile values or update the realm redirect URI and host flags together. |
| Generated-host `/me` returns 500 | The host did not inject `hostauth.ServiceFactoryKey` or the HTTP provider did not receive the generated auth host. | Use the provided `cmd/host` or configure services with `hostauth.NewServiceFactory`. |

## See Also

- `express-auth-user-guide` — Main planned-auth route authoring and host-wiring guide.
- `migrate-express-apps-to-planned-auth` — Migration tutorial for old raw handlers.
- `express-module` — General Express-style HTTP module reference.
- Source: `examples/xgoja/18-express-auth-host`, `examples/xgoja/19-express-keycloak-auth-host`, and `examples/xgoja/21-generated-host-auth`.
