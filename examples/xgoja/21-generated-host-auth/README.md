# xgoja generated-host auth example

This example demonstrates the generated runtime-package seam for host-owned Express auth.
The generated package does not hard-code auth stores or session policy in JavaScript.
Instead, the Go host imports the generated package and injects a lazy
`hostauth.ServiceFactoryKey` with `Options.ConfigureServices`.

Run the smoke test:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

The smoke test regenerates `internal/xgojaruntime`, starts the generated host's
`serve sites demo` command, checks public routes, and verifies that the planned
auth route returns `401 Unauthorized` without a session cookie.

## What this demonstrates

The xgoja spec selects the HTTP provider and embeds local jsverbs into a runtime
package artifact:

```yaml
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      config:
        reject-raw-routes: true
        dev-errors: false
artifacts:
  - id: runtime-package
    type: runtime-package
    output: internal/xgojaruntime
    package: xgojaruntime
    sources: [local-sites]
```

The host application wires auth services in Go:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(services *app.HostServices) {
        _ = services.SetHostService(
            hostauth.ServiceFactoryKey,
            hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: defaultAuthConfig()}),
        )
    },
})
```

The JavaScript route only declares intent:

```js
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => res.json({ actor: ctx.actor.id, action: ctx.action }));
```

The Go host owns the session manager, cookies, stores, audit sink, resource
resolver, and authorizer. This keeps Express as a route declaration layer rather
than an auth infrastructure owner.

## Store modes

By default the host uses in-memory stores, suitable for a fast local smoke:

```bash
go run ./examples/xgoja/21-generated-host-auth/cmd/host \
  serve sites demo --http-listen 127.0.0.1:8787
```

To demonstrate persistent stores, pass the auth store settings through the
Glazed-backed `serve` command flags. The host applies schema on startup for this
example:

```bash
go run ./examples/xgoja/21-generated-host-auth/cmd/host \
  serve sites demo \
  --http-listen 127.0.0.1:8787 \
  --auth-default-store-driver sqlite \
  --auth-default-store-dsn /tmp/xgoja-generated-host-auth.sqlite \
  --auth-default-store-apply-schema
```

The DSN is deliberately supplied as a Glazed command setting rather than being
read directly with `os.Getenv` in auth code. Postgres and OIDC/Keycloak
configuration remain follow-up work; this example focuses on the generated-host
seam and dev/session-store foundation.

## Manual checks

Start the server and then visit:

- <http://127.0.0.1:8787/> — public text route.
- <http://127.0.0.1:8787/healthz> — public JSON health route.
- <http://127.0.0.1:8787/me> — protected planned auth route, expected `401`
  without a session cookie.
