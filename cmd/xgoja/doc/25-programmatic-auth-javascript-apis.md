---
Title: "Programmatic auth JavaScript APIs"
Slug: programmatic-auth-javascript-apis
Short: "Provision automation agents and API tokens from generated xgoja JavaScript using Go-owned auth builders."
Topics:
- xgoja
- auth
- programmatic-auth
- agents
- api-tokens
- javascript
Commands:
- xgoja
- xgoja build
- xgoja doctor
Flags:
- --auth-mode
- --auth-default-store-driver
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

Generated xgoja hosts can expose a JavaScript `auth` module for programmatic API access. Use it when a server route needs to create durable automation identities, issue API tokens, list or revoke those tokens, and return only redacted credential metadata after issuance.

This page covers server-side JavaScript APIs such as `auth.agents.create(...)`, `auth.grants()`, and `auth.tokens.api.*`. It complements `generated-auth-javascript-apis`, which focuses on browser-session OIDC, audit queries, and capability-token flows.

## Mental model

Programmatic auth separates three concepts that are easy to confuse:

| Concept | Meaning | JavaScript API |
| --- | --- | --- |
| Agent | Durable automation principal such as a CI bot or integration. | `auth.agents.create(...)` |
| API token | Bearer credential that authenticates as an agent or subject. | `auth.tokens.api.issue(...)` |
| Grant | Typed permission attached to an agent or token. | `auth.grants().allow(...).done()` |

The raw API token value is available only once, in the issuance result. Store it like a secret. Later list and revoke operations return redacted metadata such as token id, prefix, credential hint, timestamps, and scopes.

## Enable the auth module

Programmatic builders live in the generated hostauth `auth` module. The generated server needs the hostauth provider, the `auth` runtime module, and top-level `auth:` config so concrete services exist at runtime.

```yaml
providers:
  - id: go-go-goja-hostauth
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/hostauth
    register: Register
runtime:
  modules:
    - provider: go-go-goja-hostauth
      name: auth
      as: auth
      config:
        audit:
          maxLimit: 50
auth:
  mode: dev
  stores:
    default:
      driver: memory
```

For production-shaped hosts, use durable stores rather than memory. The current generated store configuration covers sessions, audit, appauth, and capability stores. SQL-backed programmatic token/agent stores should be reviewed before multi-process production use.

## Create an agent and issue its first token

The most common bootstrap flow creates an agent and issues a token in one builder chain.

```javascript
const auth = require("auth")
const fs = require("fs:host")

const issued = auth.agents.create("daily-report-bot")
  .kind("ci")
  .tenantId("o1")
  .createdBy("server-bootstrap")
  .grants(
    auth.grants()
      .tenant("o1")
      .allow("user.self.read")
      .done()
  )
  .issueApiToken("daily-report-token")
  .expiresInDays(30)
  .run()

fs.writeFileSync("/run/secrets/daily-report-token.json", JSON.stringify({
  agent: issued.agent,
  token: issued.token
}, null, 2), "utf8")
```

The output shape is intentionally asymmetric:

```json
{
  "agent": {
    "id": "...",
    "name": "daily-report-bot",
    "kind": "ci",
    "tenantId": "o1",
    "scopes": ["tenant:o1:user.self.read"]
  },
  "token": {
    "id": "...",
    "name": "daily-report-token",
    "value": "ggpat_...",
    "tokenPrefix": "...",
    "credentialHint": "ggpat_...",
    "scopes": ["tenant:o1:user.self.read"]
  }
}
```

`token.value` appears only in the issue response. Do not log the full object in production unless your log sink is secret-safe.

## Build grants explicitly

Use `auth.grants()` when a token or agent needs more than a single action string.

```javascript
const grants = auth.grants()
  .tenant("o1")
  .resource("report", "daily")
  .allow("report.read")
  .allow("report.export")
  .done()
```

The builder is Go-owned. JavaScript passes the resulting builder object back to `grants(...)`; it does not construct arbitrary grant maps for security-sensitive policy.

For a simple agent policy, `.allow(action)` on the agent builder is also supported:

```javascript
auth.agents.create("audit-bot")
  .tenantId("o1")
  .allow("audit.read")
  .createdBy("admin")
  .run()
```

## Issue, list, and revoke API tokens

Issue a token for an existing agent:

```javascript
const issued = auth.tokens.api.issue("nightly-job-token")
  .agent(agentId)
  .createdBy("operator")
  .grants(auth.grants().tenant("o1").allow("job.run").done())
  .expiresInDays(7)
  .run()
```

List active tokens for an agent:

```javascript
const active = auth.tokens.api.list()
  .agent(agentId)
  .run()
```

Include revoked tokens when an operator screen needs historical state:

```javascript
const all = auth.tokens.api.list()
  .agent(agentId)
  .includeRevoked(true)
  .run()
```

Revoke by token id, not by raw token value:

```javascript
const revoked = auth.tokens.api.revoke()
  .id(tokenId)
  .run()
```

List and revoke responses never include the raw token value.

## Authenticate with the token

A server route should not manually parse programmatic tokens. It should declare route intent with Express route auth requirements and let the Go enforcer authenticate `Authorization: Bearer ...`.

```javascript
const express = require("express")
const app = express.app()

app.get("/agent/reports/:reportId")
  .auth(express.agent())
  .allow("report.read")
  .audit("agent.report.read")
  .handle((ctx, res) => {
    res.json({
      ok: true,
      actor: ctx.actor,
      auth: ctx.auth,
      reportId: ctx.params.reportId
    })
  })
```

When an API token authenticates successfully, `ctx.auth.method` is `"apiToken"` and `ctx.auth.principalKind` is usually `"agent"`. Use `ctx.auth.credentialHint` for display. Do not expect raw token values to appear in handler context.

## Local validation

The generated programmatic-agent example builds a real server binary and a real agent binary:

```bash
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

The smoke verifies these auth properties:

- the server provisions an agent and writes a bootstrap token file;
- the agent-only route returns `401` without a token;
- the JavaScript agent authenticates through `fetch.auth.bearer().fromFile(...).jsonPath(...)`;
- the route context reports `authMethod: "apiToken"` and `principalKind: "agent"`;
- the same API token is rejected by a `sessionUser()` route with `403`.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `require("auth")` fails | The hostauth provider/runtime module is not selected. | Add the `go-go-goja-hostauth` provider and `runtime.modules[].name: auth`. |
| `auth module requires hostauth services` | The runtime has the `auth` module but the command did not build hostauth services. | Run through a generated HTTP serve host with top-level `auth:` config, or split client-only agents into a separate xgoja spec. |
| API token works on no routes | The token grants do not intersect the route `.allow(...)` action. | Check `issued.token.scopes` and the route action string. |
| Agent token reaches a route that should be browser-only | The route accepts generic user auth instead of `express.sessionUser()`. | Use route auth requirements for browser-session-only routes. |
| A raw token appears in logs | The issuance result was logged directly. | Log only `token.id`, `tokenPrefix`, or `credentialHint`; treat `token.value` as a secret. |

## See also

- `xgoja help express-route-auth-requirements`
- `xgoja help guarded-fetch-client-api`
- `xgoja help device-authorization-programmatic-access`
- `xgoja help generated-auth-javascript-apis`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-stores-reference`
- `examples/xgoja/22-programmatic-agent-auth`
