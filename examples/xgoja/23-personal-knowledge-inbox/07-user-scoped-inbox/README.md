# Step 07: user-scoped inbox

This step copies Step 06 and makes authenticated identity affect application data. Alice and Bob still log in through local Keycloak, but `/api/inbox`, `/api/capture`, and `/api/inbox/:id/archive` now operate only on rows owned by the current browser session user.

The teaching point is that login is not enough by itself. Once a route is authenticated, handlers must still scope reads and writes to the authenticated principal or an app-owned tenant/resource boundary.

## Tutorial users

| User | Password |
| --- | --- |
| `alice` | `alice-password` |
| `bob` | `bob-password` |

## What changed from Step 06

- The browser API still requires `express.sessionUser()`.
- Captured browser rows store `submitted_by_kind = 'sessionUser'` and `submitted_by_id = ctx.actor.id`.
- `GET /api/inbox` returns only rows for `ctx.actor.id`.
- `POST /api/inbox/:id/archive` only archives rows for `ctx.actor.id`.
- The UI labels the inbox as the current user's inbox.
- Direct CLI commands still access the local SQLite file directly for tutorial/debugging use.

## Run fast smoke

```bash
make smoke
```

## Run Keycloak smoke

```bash
make keycloak-smoke
```

This starts Keycloak on port `18087`, starts the generated app on `18794`, verifies the OIDC redirect, verifies static assets, and verifies that logged-out API access returns `401`.

## Manual run

```bash
make build
make keycloak-up
./dist/personal-knowledge-inbox-user-scoped-inbox \
  serve inbox server \
  --db /tmp/personal-inbox-user-scoped.sqlite
```

Then open <http://127.0.0.1:18794/>.

Try this manually:

1. Log in as Alice.
2. Capture one item.
3. Log out.
4. Log in as Bob.
5. Confirm Bob does not see Alice's item.
6. Capture a Bob item.
7. Log out and log back in as Alice.
8. Confirm Alice still sees only Alice's item.

Stop Keycloak with:

```bash
make keycloak-down
```

Later device/programmatic steps will grant non-browser clients access to this same user/resource boundary instead of reopening public API access.

## Run tinyidp smoke

```bash
make tinyidp-smoke
```

This replaces the local Keycloak container with tinyidp for the browser login path, then verifies that the generated app creates an authenticated session and can read the protected inbox route.
