# Step 06: browser login with local Keycloak

This step copies Step 05 and adds local OIDC browser login with Keycloak. It also changes the CLI commands back to direct SQLite access, because the browser API is now protected by session authentication and CSRF rather than being a public API for CLI automation.

The tutorial users are seeded by `keycloak/realm-personal-inbox.json`:

| User | Password |
| --- | --- |
| `alice` | `alice-password` |
| `bob` | `bob-password` |

Human credentials live in Keycloak. xgoja receives OIDC claims, upserts an app-local user record, and creates an opaque app-session cookie.

## What changed from Step 05

- Added `compose.yaml` with a local Keycloak container.
- Added `keycloak/realm-personal-inbox.json` with Alice, Bob, and a public PKCE OIDC client.
- Added top-level `auth:` config in `xgoja.yaml`.
- Added the `go-go-goja-hostauth` provider so generated `serve` can mount native auth routes.
- Protected the browser API with `express.sessionUser()`.
- Added CSRF checks for browser mutations.
- Updated browser JavaScript to read `/auth/session` and send `X-CSRF-Token`.
- Replaced fetch-backed CLI verbs with direct SQLite verbs using `verbs/lib/inbox_store.js`.

## Run fast smoke

```bash
make smoke
```

This validates the generated binary and direct SQLite CLI commands. It does not start Docker.

## Run Keycloak smoke

```bash
make keycloak-smoke
```

This starts Keycloak, starts the generated app, verifies the OIDC login redirect, verifies the UI assets, and verifies that unauthenticated API access returns `401`.

## Run tinyidp smoke

```bash
make tinyidp-smoke
```

This runs the same generated hostauth OIDC login path against `tinyidp`, a small local mock IdP, instead of Docker Compose Keycloak. It starts `tinyidp`, starts the generated app with a root issuer URL, drives `/auth/login` through the HTML login form as `alice`, and asserts that `/auth/session` returns `alice@example.test` plus a CSRF token.

By default the target expects `tinyidp` in the workspace sibling directory:

```text
../2026-06-22--mock-oidc-idp
```

Override it with `TINYIDP_ROOT=/path/to/tinyidp make tinyidp-smoke`. The smoke intentionally uses a root issuer such as `http://127.0.0.1:19087`; Keycloak-style realm-path issuer compatibility is a separate tinyidp feature.

## Manual run

```bash
make build
make keycloak-up # waits until OIDC discovery is reachable
./dist/personal-knowledge-inbox-browser-login-keycloak \
  serve inbox server \
  --db /tmp/personal-inbox-ui.sqlite
```

The Step 06 `xgoja.yaml` sets the HTTP default to `127.0.0.1:18793`, matching the Keycloak redirect URI. Then open <http://127.0.0.1:18793/> and log in as Alice or Bob.

If you start Keycloak with raw `docker compose up -d`, wait for discovery before starting the app:

```bash
make keycloak-wait
```

Stop Keycloak with:

```bash
make keycloak-down
```

## Native auth endpoints introduced here

The generated host mounts these Go-owned routes before JavaScript routes:

- `GET /auth/login`
- `GET /auth/callback`
- `GET /auth/session`
- `POST /auth/logout`

Later steps will use this browser session to approve device authorization and issue programmatic credentials.

The tinyidp smoke uses `../tinyidp-users.yaml` so Alice and Bob have stable seeded `sub` values and inbox-specific claims.
