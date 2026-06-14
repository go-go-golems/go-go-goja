# Express Keycloak auth host example

This example is the production-oriented companion to `../18-express-auth-host`. It uses a Docker Compose Keycloak realm for login and the optional host-auth packages:

- `keycloakauth` for OIDC Authorization Code + PKCE login/callback/logout,
- `sessionauth` for opaque app session cookies and CSRF verification,
- `appauth` for app-owned resource resolution and explicit authorization,
- `appauth/sqlstore` for persistent app users, memberships, tenants, and resources,
- `capability/sqlstore` for persistent hashed invite tokens,
- `audit` for JSON audit logging,
- `modules/express` and `pkg/gojahttp` for planned route declarations/enforcement.

The browser receives an app session cookie. Keycloak tokens stay server-side during the callback flow.

## Automated smoke

The smoke starts Docker Compose Keycloak plus a Postgres database for persistent app sessions, audit records, appauth users/resources, and capability token hashes. It starts the Go host, drives the Keycloak login form with the demo user, verifies the app session, checks CSRF enforcement, updates a project route, issues and accepts a single-use org invite capability, verifies persisted SQL rows, logs out, and tears the containers down again:

```bash
make -C examples/xgoja/19-express-keycloak-auth-host smoke
```

It uses only Python standard-library HTTP/form handling; no browser driver is required. Set `KEEP_KEYCLOAK=1` to leave the containers running after the smoke, `KEYCLOAK_PORT=18081` if port `18080` is already in use, or `POSTGRES_PORT=15433` if port `15432` is already in use.

The JavaScript route script also includes public async planned routes. They use the host-provided `timer` module to prove the request waits for promises:

```js
const timer = require("timer")

app.get("/async-return")
  .public()
  .handle(async (ctx, _res) => {
    await timer.sleep(5)
    return `async return ${ctx.request.query.name}`
  })

app.get("/async-send")
  .public()
  .handle(async (ctx, res) => {
    await timer.sleep(5)
    res.json({ ok: true, mode: "send", name: ctx.request.query.name })
  })
```

The host awaits returned promises for planned routes. Fulfilled string return values are sent directly, `res.json(...)` can be called after `await` for structured data, and rejected promises become handler errors.

## Start Keycloak and Postgres manually

```bash
make -C examples/xgoja/19-express-keycloak-auth-host keycloak-up
```

Keycloak runs at:

```text
http://127.0.0.1:18080
```

Postgres is exposed for the Go host's app sessions, audit records, appauth rows, and capability hashes at:

```text
postgres://goja:goja@127.0.0.1:15432/goja_auth?sslmode=disable
```

Imported realm/client/user:

```text
realm:    goja-demo
client:   goja-app
username: demo@example.test
password: demo-password
```

## Start the Go host

```bash
make -C examples/xgoja/19-express-keycloak-auth-host serve
```

Open:

```text
http://127.0.0.1:8790/
```

Click **Login with Keycloak**, sign in with the demo credentials, then try:

```bash
curl -i 'http://127.0.0.1:8790/async-return?name=manual'
curl -i 'http://127.0.0.1:8790/async-send?name=manual'
curl -i -b /tmp/goja-keycloak.cookie http://127.0.0.1:8790/me
```

For command-line testing, use a browser or run `scripts/keycloak_smoke.py` with a cookie jar managed by the script to complete the Keycloak form login. After login, `GET /auth/session` returns the app session's CSRF token:

```bash
curl -i -b /tmp/goja-keycloak.cookie http://127.0.0.1:8790/auth/session
```

Then use that CSRF token on unsafe planned routes:

```bash
curl -i -X PATCH \
  -b /tmp/goja-keycloak.cookie \
  -H 'X-CSRF-Token: TOKEN' \
  http://127.0.0.1:8790/orgs/o1/projects/p1
```

## Stop Keycloak and Postgres

```bash
make -C examples/xgoja/19-express-keycloak-auth-host keycloak-down
```

## Notes

This is still an example, not a complete production deployment. For production:

- use HTTPS,
- use secure cookies, not `AllowInsecureHTTP`,
- keep the Postgres-backed app session, audit, appauth, and capability stores,
- review Keycloak realm/client settings,
- add a shared transaction store for multi-instance callback handling,
- keep application authorization in app-owned Go code or a chosen policy engine.
