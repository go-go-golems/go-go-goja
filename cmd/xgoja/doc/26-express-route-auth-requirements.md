---
Title: "Express route auth requirements"
Slug: express-route-auth-requirements
Short: "Declare whether planned Express routes accept browser users, automation agents, or explicit alternatives."
Topics:
- xgoja
- express
- auth
- gojahttp
- rate-limits
- javascript
Commands:
- xgoja
- xgoja build
- xgoja doctor
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Planned Express routes declare who may enter before handler code runs. The host still owns authentication, CSRF verification, authorization, resource loading, audit, and rate-limit enforcement; JavaScript only declares route intent.

Use this page when choosing between `express.agent()`, `express.sessionUser()`, `express.anyOf(...)`, and route-level `.rateLimit(...)`. For broader host composition and OIDC setup, start with `express-auth-host-integration-guide` and `hostauth-config-reference`.

## Choose the right route auth builder

The route auth builder narrows acceptable principal types after the host has authenticated the request.

| Builder | Accepts | Typical use |
| --- | --- | --- |
| `express.agent()` | Authenticated automation agent principal. | API-token driven jobs and integrations. |
| `express.sessionUser()` | Browser app-session user principal. | UI/API routes that require cookie sessions and CSRF for mutations. |
| `express.user().required()` | Legacy/general authenticated user requirement. | Existing session-user routes; prefer `sessionUser()` when the route must reject agent tokens. |
| `express.anyOf(...)` | One of several explicit alternatives. | Routes intentionally shared by browser users and agents. |

Do not distinguish agents by manually inspecting `Authorization` headers in handlers. The Go enforcer parses bearer tokens, builds `ctx.auth`, intersects token grants with route actions, and rejects requests before the handler when requirements do not match.

## Agent-only route

Use `express.agent()` for routes meant to be called by programmatic clients.

```javascript
const express = require("express")
const app = express.app()

app.get("/agent/reports/:reportId")
  .auth(express.agent())
  .rateLimit(express.rateLimit("agent-report-read").perMinute(60).byActor().byRoute())
  .allow("report.read")
  .audit("agent.report.read")
  .handle((ctx, res) => {
    res.json({
      ok: true,
      reportId: ctx.params.reportId,
      actor: ctx.actor,
      auth: ctx.auth
    })
  })
```

A missing token returns `401`. A browser session or other principal kind that does not satisfy the route requirement returns `403`.

## Browser-session-only route

Use `express.sessionUser()` when an API-token agent must not enter a route, even if it has a matching action grant.

```javascript
app.post("/profile/email")
  .auth(express.sessionUser())
  .csrf()
  .allow("profile.email.update")
  .audit("profile.email.update")
  .handle((ctx, res) => {
    res.json({ ok: true, user: ctx.actor.id })
  })
```

Unsafe browser-session routes should normally call `.csrf()`. API-token authentication does not require CSRF because it does not rely on ambient browser cookies, but the route requirement above rejects API-token agents before the handler.

## Explicit alternatives

Use `express.anyOf(...)` only when the product decision is explicit. It keeps mixed routes reviewable.

```javascript
app.get("/reports/:reportId")
  .auth(express.anyOf(
    express.sessionUser(),
    express.agent()
  ))
  .allow("report.read")
  .audit("report.read")
  .handle((ctx, res) => {
    res.json({
      reportId: ctx.params.reportId,
      principalKind: ctx.auth.principalKind,
      authMethod: ctx.auth.method
    })
  })
```

Avoid using `anyOf` as a shortcut for uncertainty. If a route should only serve browsers, use `sessionUser()`. If it should only serve automation, use `agent()`.

## Read `ctx.auth` in handlers

`ctx.auth` is a redacted authentication result. It is safe for diagnostics and response metadata, but it is not a place to make the primary authorization decision.

```typescript
interface AuthInfo {
  method: "none" | "session" | "apiToken" | "accessToken" | string
  principalKind?: "user" | "agent" | "service" | string
  principalId?: string
  credentialId?: string
  credentialHint?: string
  scopes: string[]
}
```

Common values:

| Field | Meaning |
| --- | --- |
| `method` | How the host authenticated the request, such as `session` or `apiToken`. |
| `principalKind` | The kind of principal that entered the route, such as `user` or `agent`. |
| `principalId` | The app user id, agent id, or service id. |
| `credentialHint` | Redacted display hint for the credential. |
| `scopes` | Effective scopes/grants carried by the credential. |

Use `.allow("action.name")` for authorization and use `ctx.auth` for logging, display, and branch logic that is not a security boundary.

## Rate limits are route policy

Rate limits are planned-route primitives like `.audit(...)` and `.allow(...)`. Declare them next to the route they protect.

```javascript
app.post("/auth/device")
  .public()
  .rateLimit(express.rateLimit("device-start").perMinute(10).byIP().byRoute())
  .audit("device.start")
  .handle(startDeviceFlow)

app.get("/agent/reports/:reportId")
  .auth(express.agent())
  .rateLimit(express.rateLimit("agent-report-read").perMinute(60).byActor().byRoute())
  .allow("report.read")
  .audit("agent.report.read")
  .handle(readReport)
```

The key builder controls quota identity:

| Method | Key component |
| --- | --- |
| `byIP()` | Client IP address. Useful before authentication. |
| `byRoute()` | Route identity. Useful for per-route quotas. |
| `byActor()` | Authenticated actor. Useful after authentication. |
| `byParam(name)` | Route parameter value. |
| `byTenantParam(name)` | Tenant parameter value. |
| `byHeader(name)` | Request header value. Use sparingly. |
| `byBodyField(name)` | Parsed body field. Use sparingly. |
| `byResource(name)` | Resolved resource identity. |

Pre-auth limits should use stable request data such as IP and route. Post-auth limits can use actor, tenant, or resource keys. Avoid header/body-field keys for secrets or user-controlled high-cardinality values unless you intentionally want that behavior.

## Validation and status codes

The planned route pipeline runs before JavaScript handlers:

```text
request
  -> rate limits that can run before auth
  -> authentication
  -> route auth requirement check
  -> CSRF for session-backed unsafe routes
  -> resource resolution
  -> grant/action authorization
  -> rate limits that need actor/resource data
  -> audit bookkeeping
  -> JavaScript handler
```

Expected status patterns:

| Request | Typical status |
| --- | --- |
| No credential on required route | `401 Unauthorized` |
| Valid credential with wrong principal kind | `403 Forbidden` |
| Valid API token without required grant/action | `403 Forbidden` |
| Session mutation without CSRF | `403 Forbidden` |
| Rate limit exceeded | `429 Too Many Requests` |

## Local validation

The programmatic-agent example validates the route requirement behavior through generated binaries:

```bash
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

The smoke proves that an agent token can enter an `express.agent()` route and that the same token is rejected by an `express.sessionUser()` route.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Agent token gets `403` on an agent route | Token grants do not include the route `.allow(...)` action, or the token is revoked/expired. | Inspect issued token scopes and route action names. |
| Agent token can enter a browser-only route | The route uses generic auth instead of a session-specific requirement. | Change `.auth(express.user().required())` to `.auth(express.sessionUser())` when agents must be excluded. |
| Browser session gets `403` on unsafe mutation | CSRF is required and the request did not send a valid token. | Read `/auth/session` and send the current CSRF token. |
| Rate limit groups unrelated clients together | The key builder is too broad. | Add `byActor()`, `byTenantParam(...)`, `byResource(...)`, or route-specific policies as appropriate. |
| A public route unexpectedly requires auth | The route called `.auth(...)` instead of `.public()`. | Review the staged route declaration before `.handle(...)`. |

## See also

- `xgoja help programmatic-auth-javascript-apis`
- `xgoja help guarded-fetch-client-api`
- `xgoja help generated-auth-javascript-apis`
- `xgoja help express-auth-host-integration-guide`
- `xgoja help go-planned-auth-api`
- `examples/xgoja/22-programmatic-agent-auth`
