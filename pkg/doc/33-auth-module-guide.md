---
Title: Auth module guide
Slug: auth-module-guide
Short: Use the generated-host auth module from JavaScript to query audit records and issue, validate, consume, or revoke capability tokens.
Topics:
- goja
- javascript
- auth
- audit
- capabilities
- express
Commands:
- xgoja
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `auth` module gives JavaScript route code narrow access to host-owned authentication services. It is designed for generated and embedded hosts that already configured `hostauth` stores and planned Express routes. JavaScript can query audit records and operate on capability tokens, but it does not receive raw database handles or permission to bypass the route authorization pipeline.

Use this module when a route has already declared its security policy with the Express planned-auth DSL and then needs to perform a small auth-related application action. Common examples are an admin dashboard that reads recent audit records, an invite route that issues a single-use token, and a public accept route that consumes that token.

## Runtime shape

A host exposes this module with `require("auth")` when the generated runtime plan selects the hostauth provider module:

```javascript
const auth = require("auth")
```

The module currently exposes two namespaces:

| Namespace | Purpose |
| --- | --- |
| `auth.audit` | Query host audit records through a bounded fluent query builder. |
| `auth.capabilities` | Issue, validate, consume, and revoke bearer-like capability tokens. |

The module depends on host-owned services. If the host has no audit query store or capability store, module setup fails when the runtime starts. That failure is intentional because JavaScript code should not silently fall back to in-memory or fake security data.

## Relationship to planned routes

The `auth` module is not an authorization bypass. A protected route should still declare who may call it, what resource it touches, and which action is required:

```javascript
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
      .limit(50)
      .run()
    res.json({ records, count: records.length })
  })
```

The route plan authenticates the actor and authorizes `audit.read`. The `auth.audit.query()` call only performs the data lookup after that policy has passed.

## Query audit records

`auth.audit.query()` returns a fluent builder. Each setter records one filter, and `.run()` executes the query on the host-owned audit store.

```javascript
const records = auth.audit.query()
  .tenantId("o1")
  .outcome("denied")
  .resource("project", "p1")
  .limit(25)
  .offset(0)
  .run()
```

Available setters:

| Method | Effect |
| --- | --- |
| `.tenantId(id)` | Filters records by tenant id. |
| `.outcome(value)` | Filters records by outcome such as `completed` or `denied`. |
| `.actorId(id)` | Filters records by actor id. |
| `.resource(type, id)` | Sets both resource type and resource id. |
| `.resourceType(type)` | Filters by resource type only. |
| `.resourceId(id)` | Filters by resource id only. |
| `.limit(n)` | Limits returned records. The host clamps this to its configured maximum. |
| `.offset(n)` | Skips records for pagination. |
| `.run()` | Executes the query and returns an array of records. |

Returned records use JavaScript-friendly lower-camel-case fields such as `event`, `outcome`, `actorId`, `tenantId`, `resourceType`, `resourceId`, `createdAt`, and optional `attributes`.

## Issue capability tokens

A capability token represents a constrained future action. The token is returned once from `.run()` and should be treated like a secret. The stored record keeps only a hash.

```javascript
const issued = auth.capabilities.issue("org-invite")
  .resource("org", org.id)
  .tenantId(org.id)
  .claimString("email", body.email || "")
  .claimString("role", body.role || "viewer")
  .ttlSeconds(900)
  .singleUse(true)
  .createdBy(ctx.actor.id)
  .run()

res.json({
  token: issued.token,
  capabilityId: issued.capability.id,
  expiresAt: issued.capability.expiresAt
})
```

Available issue setters:

| Method | Effect |
| --- | --- |
| `.subject(kind, id)` | Sets the subject id for `"id"` or `"user"`; other kinds are stored as string claims named `subject.<kind>`. |
| `.subjectId(id)` | Sets the first-class capability subject id. |
| `.resource(type, id)` | Sets the resource type and id the capability applies to. |
| `.tenantId(id)` | Stores the tenant id as a string claim. |
| `.claimString(key, value)` | Adds one string claim. |
| `.ttlSeconds(seconds)` | Sets expiry relative to issue time. |
| `.expiresAt(rfc3339)` | Sets an absolute expiry. |
| `.singleUse(true)` | Marks the token as consuming only once. |
| `.createdBy(actorId)` | Records who issued the capability. |
| `.run()` | Stores the capability and returns `{ token, capability }`. |

Use capability claims for application payload that is needed later, such as invite email and role. Keep claims small and non-secret unless the storage and audit path for that data is acceptable.

## Validate versus consume

Validation checks a token without marking it used. Consumption validates and then marks a single-use token used. This distinction matters when you need to check expected type or resource before burning a token.

```javascript
const preview = auth.capabilities.validate(token)
  .expectedType("org-invite")
  .expectedResource("org", "o1")
  .run()
```

```javascript
const accepted = auth.capabilities.consume(token)
  .expectedType("org-invite")
  .expectedResource("org", "o1")
  .run()
```

Available validate/consume setters:

| Method | Effect |
| --- | --- |
| `.expectedType(type)` | Requires the stored capability purpose/type to match. |
| `.expectedResource(type, id)` | Requires the stored resource type and id to match before consume. |
| `.run()` | Returns the redacted capability record, or throws a Go-backed error. |

A consumed single-use token has `usedAt` in the returned record. Reusing the same token throws an error equivalent to `capability.ErrUsed`.

## Revoke capabilities

Revocation invalidates a capability by id. The current JavaScript builder accepts a reason for readability, but the underlying service stores only the revoked timestamp.

```javascript
auth.capabilities.revoke()
  .id(capabilityId)
  .reason("user_request")
  .run()
```

A successful revoke returns `{ revoked: true, id }`.

## Map errors deliberately

Capability errors are normal application outcomes. A public accept route should not expose every failure as a server error:

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
      res.json({ orgId: accepted.resourceId, email: accepted.claims.email })
    } catch (err) {
      const message = String((err && err.message) || err || "capability rejected")
      const status = message.includes("already used") ? 409 : 400
      res.status(status).json({ error: message })
    }
  })
```

A typical mapping is:

| Condition | Suggested status | Why |
| --- | --- | --- |
| Token already used | `409 Conflict` | The request conflicts with current token state. |
| Token missing, malformed, expired, revoked, wrong type, or wrong resource | `400 Bad Request` | The public accept request cannot be honored. |
| Protected issue route lacks actor, CSRF, resource, or authorization | Let planned auth return `401`, `403`, or `404` | The route plan should enforce these checks before handler code runs. |

## Example 21 route ownership

`examples/xgoja/21-generated-host-auth` is the canonical full generated-host example. Native Go handlers own only OIDC/session lifecycle routes. The JavaScript file owns application-specific routes such as audit dashboards and org invites.

Run the fast generated-host smoke:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

Run the real local Keycloak/Postgres smoke:

```bash
make -C examples/xgoja/21-generated-host-auth compose-smoke
```

The compose smoke logs in through the example Keycloak realm, seeds demo appauth resources and membership rows, exercises the JavaScript-owned audit/invite routes, and verifies that a reused invite token returns `409`.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `require("auth")` fails during runtime setup | The host did not install hostauth services, or the runtime plan did not select the hostauth provider module. | Add the hostauth provider and `runtime.modules` entry, and run through a generated host that builds auth services. |
| Audit query fails at startup | The configured audit store does not implement query access. | Use the generated audit store configuration or a store that implements `audit.QueryStore`. |
| Capability issue fails with missing subject or resource | The issue builder did not set either `.subjectId(...)` or `.resource(...)`. | Add a resource for resource-scoped capabilities, or a subject for user-scoped capabilities. |
| Capability consume returns an error after expected-resource checks | The token is missing, expired, revoked, already used, wrong type, or wrong resource. | Map expected application failures to `400` or `409` and log enough context to debug. |
| Protected invite route returns `403` after login | The appauth membership/resource store lacks demo authorization rows. | Seed tenants, resources, and memberships outside generic OIDC normalization. |

## See also

- `goja-repl help express-auth-user-guide`
- `goja-repl help express-auth-examples`
- `xgoja help generated-auth-javascript-apis`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-stores-reference`
