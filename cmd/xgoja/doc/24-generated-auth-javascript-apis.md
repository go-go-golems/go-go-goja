---
Title: "Generated auth JavaScript APIs"
Slug: generated-auth-javascript-apis
Short: "Use require(\"auth\") in generated xgoja OIDC hosts for audit queries and capability-token flows."
Topics:
- xgoja
- auth
- oidc
- javascript
- audit
- capabilities
Commands:
- xgoja
- xgoja build
- xgoja doctor
Flags:
- --auth-mode
- --auth-oidc-issuer-url
- --auth-oidc-public-base-url
- --auth-default-store-driver
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

Generated OIDC hosts can expose a JavaScript `auth` module alongside the Express planned-route module. The module gives route code safe access to high-level auth services: bounded audit queries and generic capability-token operations. It does not expose raw database handles, session stores, Keycloak tokens, or authorization internals.

Use this page when you are building an `xgoja.yaml` host with `auth.mode=oidc`, writing JavaScript routes for that host, or reviewing the generated example in `examples/xgoja/21-generated-host-auth`.

## Provider and runtime configuration

The generated binary must import the hostauth provider and select its `auth` runtime module. Example 21 uses this shape:

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
```

The module requires concrete auth services. In a generated HTTP serve host, those services come from top-level `auth:` config and the `serve` command's `--auth-*` flags.

## Native routes versus application routes

The generated hostauth layer owns OIDC and session lifecycle handlers:

```text
GET  /auth/login
GET  /auth/callback
GET  /auth/logout
POST /auth/logout
GET  /auth/session
```

Application-specific routes belong in JavaScript. Example 21 owns these routes in `verbs/sites.js`:

```text
GET  /orgs/:orgId/audit
POST /orgs/:orgId/invites
POST /org-invites/accept
```

This split keeps reusable auth core small. Login, callback, logout, and session introspection are host lifecycle concerns. Audit dashboards, org invites, project updates, and other domain actions are application concerns that should be visible in route code and protected by planned-route policy.

## Audit query route

A generated host can expose an app audit view without giving JavaScript direct database access:

```javascript
const express = require("express")
const auth = require("auth")
const app = express.app()

app.get("/orgs/:orgId/audit")
  .auth(express.user().required())
  .resource(
    express.resource("org")
      .idFromParam("orgId")
      .mustExist()
  )
  .allow("audit.read")
  .audit("audit.records.read")
  .handle((ctx, res) => {
    const org = ctx.resource("org")
    const records = auth.audit.query()
      .tenantId(org.id)
      .outcome(ctx.request.query.outcome || "")
      .limit(Number(ctx.request.query.limit || 50))
      .run()
    res.json({ records, count: records.length })
  })
```

The planned route handles authentication, resource loading, and `audit.read` authorization. The `auth.audit.query()` builder only runs after those checks pass.

## Capability issue route

A capability token is a constrained bearer token for a future action. Issue it from a protected route, usually with CSRF and an authorization action:

```javascript
app.post("/orgs/:orgId/invites")
  .auth(express.user().required())
  .resource(
    express.resource("org")
      .idFromParam("orgId")
      .mustExist()
  )
  .csrf()
  .allow("org.member.invite")
  .audit("org.invite.issued")
  .handle((ctx, res) => {
    const org = ctx.resource("org")
    const body = ctx.body || {}
    const issued = auth.capabilities.issue("org-invite")
      .resource("org", org.id)
      .tenantId(org.id)
      .claimString("email", body.email || "")
      .claimString("role", body.role || "viewer")
      .ttlSeconds(900)
      .singleUse(true)
      .createdBy(ctx.actor.id)
      .run()
    res.json({ token: issued.token, expiresAt: issued.capability.expiresAt })
  })
```

The returned `token` is the only time the raw token is available. Persisted stores keep a hash, not the raw token.

## Capability accept route

Public accept routes should consume tokens and map expected failures to client errors:

```javascript
app.post("/org-invites/accept")
  .public()
  .audit("org.invite.accepted")
  .handle((ctx, res) => {
    const body = ctx.body || {}
    try {
      const accepted = auth.capabilities.consume(body.token || "")
        .expectedType("org-invite")
        .expectedResource("org", "o1")
        .run()
      res.json({
        capabilityId: accepted.id,
        orgId: accepted.resourceId,
        email: accepted.claims.email,
        role: accepted.claims.role,
      })
    } catch (err) {
      const message = String((err && err.message) || err || "capability rejected")
      const status = message.includes("already used") ? 409 : 400
      res.status(status).json({ error: message })
    }
  })
```

Use `.validate(token)` when the route needs a preview without consuming a single-use token. Use `.consume(token)` when the route is performing the action and should mark the token used.

## Store and seed data requirements

The `auth` module uses the stores selected by generated hostauth. For production-shaped generated hosts, use PostgreSQL for session, audit, appauth, and capability stores:

```bash
generated-oidc-host-auth serve sites demo \
  --auth-mode oidc \
  --auth-default-store-driver postgres \
  --auth-default-store-dsn "$DATABASE_URL" \
  --auth-default-store-apply-schema \
  --auth-oidc-issuer-url "$KEYCLOAK_ISSUER" \
  --auth-oidc-client-id "$KEYCLOAK_CLIENT_ID" \
  --auth-oidc-public-base-url "$PUBLIC_BASE_URL"
```

Generic OIDC normalization upserts the app user and projects existing memberships into the session. It does not grant demo roles. Seed tenants, resources, and memberships as application data. The example 21 compose smoke demonstrates this explicitly by inserting demo `o1` rows after login.

## Local validation

Run the fast generated-host smoke first. It validates the xgoja plan, build, embedded assets, fake OIDC discovery, and unauthenticated route behavior:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

Run the compose smoke when you want real Keycloak/Postgres coverage:

```bash
make -C examples/xgoja/21-generated-host-auth compose-smoke
```

The compose smoke reuses the example 19 Keycloak realm and Postgres compose stack, starts example 21 with Postgres-backed auth stores, drives browserless Keycloak login, seeds demo appauth rows, exercises JS-owned audit and invite routes, and verifies reused invite tokens return `409`.

## Deployment guidance

For production deployments, keep these boundaries stable:

- `--auth-oidc-public-base-url` is the browser-visible HTTPS origin.
- `--auth-oidc-redirect-url` is only an advanced override.
- Never derive redirect URLs from `--http-listen`; listen addresses are process-local.
- Use HTTPS outside localhost. Only use insecure HTTP for local smoke tests.
- Keep replicas at one until OIDC transaction/session durability is reviewed for multi-pod behavior.
- Do not bake issuer URLs, client ids, client secrets, DSNs, or public URLs into images.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `require("auth")` fails | The hostauth provider was not selected, or hostauth services were not built. | Add the provider/runtime module and run through a generated host with top-level `auth:` config. |
| `/auth/login` works but protected app routes return `403` | The user authenticated but appauth memberships/resources are missing. | Seed application tenants, memberships, and resources. Do not add demo grants to the generic OIDC normalizer. |
| Reused invite token returns `500` | The route lets capability errors escape as generic handler errors. | Catch capability errors and map `already used` to `409`, other invalid-token cases to `400`. |
| Compose smoke fails with invalid redirect URI | The Keycloak client redirect URI does not match the host public base URL. | Use the example 19 realm with `http://127.0.0.1:8790/auth/callback`, or update the realm/client. |
| Audit query returns too many records | The route did not set a limit, or the configured maximum is too high. | Use `.limit(...)` and set `runtime.modules[].config.audit.maxLimit`. |

## See also

- `goja-repl help auth-module-guide`
- `goja-repl help express-auth-user-guide`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-stores-reference`
- `xgoja help auth-host-production-runbook`
- `examples/xgoja/21-generated-host-auth`
