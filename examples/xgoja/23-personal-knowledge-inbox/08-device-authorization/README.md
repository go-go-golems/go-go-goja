# Step 08: device authorization

This step copies Step 07 and adds the first programmatic login flow. A CLI can start a device authorization request, a logged-in browser user approves the displayed user code, and the CLI polls for access/refresh tokens. The access token can then call a programmatic capture route. Browser identity is standard OIDC; device credentials are application-owned `programauth` tokens, not tiny-idp bearer tokens.

The browser session remains the approval authority. The CLI does not receive the user's browser cookie or password.

## Tutorial users

| User | Password |
| --- | --- |
| `alice` | `alice-password` |
| `bob` | `bob-password` |

## What changed from Step 07

- Added guarded `fetch` for CLI calls to native device endpoints.
- Added CLI verbs:
  - `verbs inboxctl device-start`
  - `verbs inboxctl device-token`
  - `verbs inboxctl token-capture`
- Added browser approval UI that posts a user code to `/auth/device/approve` with CSRF.
- Added `/api/programmatic/capture`, protected with `express.agent()` and an access-token grant.
- Programmatic capture stores rows under the approving user's owner id from the device-created agent.

## Run fast smoke

```bash
make smoke
```

## Run Keycloak/device smoke

```bash
make keycloak-smoke
```

This starts Keycloak on port `18088`, starts the generated app on `18795`, verifies login redirect behavior, verifies logged-out API denial, starts a device code, and verifies polling is pending before approval.

## Manual device flow

Start services:

```bash
make build
make keycloak-up
./dist/personal-knowledge-inbox-device-authorization \
  serve inbox server \
  --db /tmp/personal-inbox-device.sqlite
```

In another terminal, start device authorization:

```bash
./dist/personal-knowledge-inbox-device-authorization \
  verbs inboxctl device-start \
  --base-url http://127.0.0.1:18795
```

Copy the `user_code` and `device_code` from the response.

In the browser:

1. Open <http://127.0.0.1:18795/>.
2. Log in as Alice or Bob.
3. Paste the `user_code` into **Device approval**.
4. Click **Approve device**.

Back in the CLI, poll with the raw `device_code`:

```bash
./dist/personal-knowledge-inbox-device-authorization \
  verbs inboxctl device-token \
  --base-url http://127.0.0.1:18795 \
  --device-code 'ggdc_...'
```

Use the returned `access_token` to capture through the programmatic route:

```bash
./dist/personal-knowledge-inbox-device-authorization \
  verbs inboxctl token-capture \
  --base-url http://127.0.0.1:18795 \
  --access-token 'ggat_...' \
  --title 'CLI device capture' \
  --url 'https://example.com/device'
```

Refresh the browser inbox for the approving user. The item appears in that user's scoped inbox.

## Run tinyidp smoke

```bash
make tinyidp-smoke
```

This uses tinyidp for the browser login/approval authority and the native xgoja device endpoints for device-code start, approval, token polling, and token-authenticated programmatic capture. It approves one device token as Alice and one as Bob, captures one item with each token, then verifies each browser user only sees the capture produced by their own approved device token.

The tinyidp smoke uses `../tinyidp-users.yaml` so Alice and Bob have stable seeded `sub` values and inbox-specific claims.

## Run strict tiny-idp production-contract smoke

```bash
make strict-tinyidp-smoke TINYIDP_ROOT=/absolute/path/to/tiny-idp
```

This provisions tiny-idp through `serve-production` with TLS, a generated signing key, exact redirect and post-logout registrations, a public PKCE client, and Alice/Bob fixture users. It then runs the generated application with disposable SQLite databases, checks the CLI’s pre-approval device poll, verifies browser/device isolation for both users, rotates and revokes refresh-token families, and runs a Playwright UI login/device-approval smoke.

The application is deliberately loopback HTTP only inside this disposable test. A real deployment must use the `hostauth` `single-node` profile, durable stores with pre-applied migrations, an HTTPS public origin behind a reverse proxy, and exact tiny-idp registrations. See the Phase 3 ticket runbook for the complete two-layer model and release checklist.

## Refresh and revoke device credentials

```bash
bin=./dist/personal-knowledge-inbox-device-authorization
$bin verbs inboxctl device-refresh --base-url http://127.0.0.1:18795 --refresh-token 'ggrt_...'
$bin verbs inboxctl device-revoke --base-url http://127.0.0.1:18795 --refresh-token 'ggrt_...'
```

Refresh rotates the refresh credential. Reusing a rotated credential revokes its family. Revocation disables the refresh family; previously issued access tokens retain their bounded expiry rather than receiving an unverified claim of immediate invalidation.
