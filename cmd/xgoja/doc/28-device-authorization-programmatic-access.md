---
Title: "Device authorization for programmatic access"
Slug: device-authorization-programmatic-access
Short: "Use generated-host native device-code endpoints to obtain access and refresh tokens for limited-input clients."
Topics:
- xgoja
- auth
- programmatic-auth
- device-flow
- oauth
Commands:
- xgoja
- xgoja build
- xgoja serve
Flags:
- --auth-mode
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

Generated xgoja hosts with hostauth enabled mount Go-owned device authorization endpoints. Use them when a CLI, appliance, or automation worker cannot safely handle an interactive browser login but can show a short user code and poll for completion.

The device flow is separate from API-token bootstrap. API tokens are long-lived credentials issued directly to an agent. Device authorization starts with a short-lived `device_code`, requires browser-session approval, and returns an access/refresh-token pair after approval.

## Endpoints

The built-in native handlers are mounted before application routes:

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/auth/device/start` | Create a device code, user code, verification URI, expiry, and poll interval. |
| `POST` | `/auth/device/token` | Poll using the device code. Returns `authorization_pending` or `slow_down` until approval, then returns tokens. |
| `POST` | `/auth/device/approve` | Browser-session + CSRF protected approval endpoint. Narrows grants and marks the code approved. |

## Start a device authorization

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"clientName":"local-cli","tenantId":"o1","actions":["user.self.read"]}' \
  http://127.0.0.1:8787/auth/device/start
```

The response contains the raw `device_code` for the polling client and a shorter `user_code` for the human:

```json
{
  "device_code": "ggdc_...",
  "user_code": "ABCD-EFGH",
  "verification_uri": "/auth/device",
  "verification_uri_complete": "/auth/device?user_code=ABCD-EFGH",
  "expires_in": 600,
  "interval": 5
}
```

Treat `device_code` as a secret. Display only the `user_code` and verification URI to the user.

## Poll for tokens

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"grant_type":"urn:ietf:params:oauth:grant-type:device_code","device_code":"ggdc_..."}' \
  http://127.0.0.1:8787/auth/device/token
```

Before approval, the handler returns OAuth-style errors:

```json
{"error":"authorization_pending","error_description":"authorization is pending","interval":5}
```

Polling too quickly returns `slow_down` and increases the interval. Expired, denied, consumed, or unknown device codes return `expired_token`, `access_denied`, or `invalid_grant`.

After approval, the next valid poll consumes the device code and returns tokens:

```json
{
  "access_token": "ggat_...",
  "refresh_token": "ggrt_...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "o1:user.self.read"
}
```

Only `ggat_...` access tokens authenticate planned routes. Refresh tokens are accepted only by refresh-token service code and are never valid route credentials.

## Approve from a browser session

`POST /auth/device/approve` requires a valid app session cookie and `X-CSRF-Token`. Approval may pass explicit actions to narrow the requested grants:

```json
{
  "user_code": "ABCD-EFGH",
  "tenantId": "o1",
  "actions": ["user.self.read"]
}
```

If actions are omitted, the requested grants are approved as-is. If actions are present, the service intersects requested grants with approved grants so approval cannot broaden privilege.

## Example and validation

See `examples/xgoja/22-programmatic-agent-auth`. Its smoke test verifies generated host wiring for:

- native device start,
- pending device-token polling,
- API-token agent route access through `fetch.client()`,
- route auth restrictions that reject API tokens on session-only routes.

Run:

```bash
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

## See also

- `xgoja help programmatic-auth-javascript-apis`
- `xgoja help express-route-auth-requirements`
- `xgoja help guarded-fetch-client-api`
- `xgoja help auth-stores-reference`
