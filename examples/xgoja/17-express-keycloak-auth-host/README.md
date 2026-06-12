# Express Keycloak auth host example

This example is the production-oriented companion to `../16-express-auth-host`. It uses a Docker Compose Keycloak realm for login and the optional host-auth packages:

- `keycloakauth` for OIDC Authorization Code + PKCE login/callback/logout,
- `sessionauth` for opaque app session cookies and CSRF verification,
- `appauth` for app-owned resource resolution and explicit authorization,
- `audit` for JSON audit logging,
- `modules/express` and `pkg/gojahttp` for planned route declarations/enforcement.

The browser receives an app session cookie. Keycloak tokens stay server-side during the callback flow.

## Automated smoke

The smoke starts Docker Compose Keycloak, starts the Go host, drives the Keycloak login form with the demo user, verifies the app session, checks CSRF enforcement, updates a project route, logs out, and tears Keycloak down again:

```bash
make -C examples/xgoja/17-express-keycloak-auth-host smoke
```

It uses only Python standard-library HTTP/form handling; no browser driver is required. Set `KEEP_KEYCLOAK=1` to leave the Keycloak container running after the smoke, or `KEYCLOAK_PORT=18081` if port `18080` is already in use.

## Start Keycloak manually

```bash
make -C examples/xgoja/17-express-keycloak-auth-host keycloak-up
```

Keycloak runs at:

```text
http://127.0.0.1:18080
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
make -C examples/xgoja/17-express-keycloak-auth-host serve
```

Open:

```text
http://127.0.0.1:8790/
```

Click **Login with Keycloak**, sign in with the demo credentials, then try:

```bash
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

## Stop Keycloak

```bash
make -C examples/xgoja/17-express-keycloak-auth-host keycloak-down
```

## Notes

This is still an example, not a complete production deployment. For production:

- use HTTPS,
- use secure cookies, not `AllowInsecureHTTP`,
- use persistent session, transaction, user, membership, resource, and audit stores,
- review Keycloak realm/client settings,
- add a shared transaction store for multi-instance callback handling,
- keep application authorization in app-owned Go code or a chosen policy engine.
