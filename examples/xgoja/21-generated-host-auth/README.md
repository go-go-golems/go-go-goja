# xgoja generated OIDC host auth example

This example demonstrates a self-contained `xgoja.yaml` generated HTTP host with
server-side OIDC auth. The YAML carries the top-level `auth:` block, the HTTP
provider exposes Glazed/env-backed `--auth-*` settings, and the generated binary
runs `serve` without a hand-written Go host shell.

Run the smoke test:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

The smoke test builds `dist/generated-oidc-host-auth`, starts a tiny local OIDC
discovery endpoint, runs the generated `serve sites demo` command, verifies
public JavaScript routes, verifies that `/auth/login` is handled by the native
Go OIDC handler, and verifies that `/me` remains protected without a session.
The route script also keeps the public async routes and protected project route
shape used by the production demo so the generated image is closer to a drop-in
platform replacement.

## What this demonstrates

The xgoja spec selects the HTTP and host providers, embeds local jsverbs plus
split dashboard assets, builds a binary, and configures OIDC entirely through
YAML:

```yaml
auth:
  mode: oidc
  session:
    cookie:
      allow-insecure-http: true
  stores:
    default:
      driver: memory
  oidc:
    issuer-url: http://localhost:18080/realms/generated-oidc-host-auth
    client-id: generated-oidc-host-auth
    public-base-url: http://localhost:18789

sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./verbs
  - id: dashboard-assets
    kind: assets
    from:
      dir: ./assets

artifacts:
  - id: binary
    type: binary
    output: dist/generated-oidc-host-auth
    sources: [local-sites, dashboard-assets]
  - id: embedded-dashboard-assets
    type: embedded-assets
    sources: [dashboard-assets]
```

The route script uses the embedded asset filesystem directly:

```js
const assets = require("fs:assets");
app.staticFromAssetsModule("/static", assets, "/app/public");
app.get("/")
  .public()
  .handle((_ctx, res) => res.type("text/html").send(
    assets.readFileSync("/app/public/index.html", "utf8")
  ));
```

`serve` owns the listener and mux. Native auth handlers are mounted before the
JavaScript app host:

- `GET /auth/login`
- `GET /auth/callback`
- `GET /auth/logout`
- `POST /auth/logout`
- `GET /auth/session`

The JavaScript route only declares application authorization intent:

```js
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .handle((ctx, res) => res.json({ actor: ctx.actor.id, action: ctx.action }));
```

## Manual run

Build the generated binary:

```bash
make -C examples/xgoja/21-generated-host-auth build
```

Run it against a real OIDC issuer:

```bash
examples/xgoja/21-generated-host-auth/dist/generated-oidc-host-auth \
  serve sites demo \
  --http-listen 127.0.0.1:18789 \
  --auth-oidc-issuer-url http://localhost:18080/realms/generated-oidc-host-auth \
  --auth-oidc-client-id generated-oidc-host-auth \
  --auth-oidc-client-secret "$OIDC_CLIENT_SECRET" \
  --auth-oidc-public-base-url http://127.0.0.1:18789
```

Then visit:

- <http://127.0.0.1:18789/> — public embedded HTML dashboard.
- <http://127.0.0.1:18789/static/app.js> — embedded dashboard JavaScript.
- <http://127.0.0.1:18789/static/styles.css> — embedded dashboard CSS.
- <http://127.0.0.1:18789/healthz> — public JSON health route.
- <http://127.0.0.1:18789/async-return?name=demo> — public async return route.
- <http://127.0.0.1:18789/async-send?name=demo> — public async JSON route.
- <http://127.0.0.1:18789/auth/login> — native OIDC login redirect.
- <http://127.0.0.1:18789/auth/session> — native app-session metadata after login.
- <http://127.0.0.1:18789/me> — protected route, expected `401` without a
  session cookie.

Production deployments should keep `allow-insecure-http` false and use an HTTPS
`public-base-url`; `redirect-url` is only needed when the callback is not
`<public-base-url>/auth/callback`.

## tinyidp smoke

The generated host can also be tested against tinyidp instead of Docker Compose Keycloak:

```bash
make -C examples/xgoja/21-generated-host-auth tinyidp-smoke
```

The target starts Postgres from the existing compose file, starts tinyidp as the OIDC issuer, runs the generated host, logs in through tinyidp, seeds demo appauth rows, and exercises CSRF, audit, and invite routes.
