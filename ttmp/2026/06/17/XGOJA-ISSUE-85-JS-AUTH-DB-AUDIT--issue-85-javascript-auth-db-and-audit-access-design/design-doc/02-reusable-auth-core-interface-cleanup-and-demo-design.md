---
Title: Reusable auth core interface cleanup and demo design
Ticket: XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT
Status: active
Topics:
    - xgoja
    - auth
    - audit
    - database
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/21-generated-host-auth/verbs/sites.js
      Note: Target demo for self-contained app-domain routes
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: Current starter app authorization policy that mixes reusable interfaces with demo actions
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Go audit Query and QueryStore contracts retained behind fluent JS builder
    - Path: pkg/gojahttp/auth/capability/capability.go
      Note: |-
        Capability primitives to generalize beyond org invites
        Future generic capability service backing fluent capability builders
    - Path: pkg/xgoja/hostauth/builder.go
      Note: Current generic native handler list includes demo routes to remove under cleanup
    - Path: pkg/xgoja/providers/hostauth/hostauth.go
      Note: |-
        Current auth JS module to extend with generic capabilities
        Current object-bag auth.audit.query implementation to refactor into fluent builder
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/85
    - https://github.com/go-go-golems/go-go-goja/issues/86
Summary: Design v2 for cleaning xgoja auth interfaces into a reusable opinionated core, replacing object-bag JS APIs with fluent Go-backed builders, and moving demo-specific org/project/invite behavior into demos.
LastUpdated: 2026-06-17T14:50:00-04:00
WhatFor: 'Use this when implementing issue #86 and follow-up auth API cleanup so generic hostauth remains reusable while examples remain rich.'
WhenToUse: Before changing hostauth native handlers, appauth action policy, capability APIs, or example 21 demo routes.
---



# Reusable auth core interface cleanup and demo design

## Executive summary

The current generated OIDC auth host has grown a useful but mixed auth surface. Some parts are clearly reusable platform primitives: OIDC login/callback/logout/session lifecycle, server-side sessions, CSRF verification, audit recording/querying, route-level `.auth()`, `.resource()`, `.allow()`, and `.audit()` hooks, and capability token storage. Other parts are demo-specific: hard-coded organization invite endpoints, project update actions, fixed organization ID `o1`, role names such as `admin`/`editor`/`viewer`, and native demo routes mounted by generic `hostauth.BuildNativeHandlers`.

The desired end state is a **small, opinionated, reusable auth core** plus a **self-contained demo** that showcases how to build real applications on top of it. The core should provide safe primitives that real users can compose without needing raw DB access for common auth workflows. The demo should live entirely in `examples/xgoja/21-generated-host-auth` (or a clearly marked demo helper package) and should use the same public JavaScript-facing APIs that users would use.

This document proposes a cleanup plan:

1. Remove demo-native routes from generic hostauth (#86).
2. Promote platform-level APIs as **fluent Go-backed builders** instead of object bags:
   - `auth.audit.query().tenantId(...).limit(...).run()`
   - `auth.capabilities.issue(type).resource(...).ttlSeconds(...).run()`
   - `auth.capabilities.validate(token).expectedType(...).run()`
   - `auth.capabilities.consume(token).expectedType(...).run()`
   - `auth.capabilities.revoke().id(...).reason(...).run()`
3. Keep route authorization policy in the route DSL (`.auth`, `.csrf`, `.resource`, `.allow`, `.audit`) rather than hiding authorization inside token helpers.
4. Split current `appauth` concepts into reusable interfaces and demo/starter policy.
5. Rebuild example 21 as a rich demo that uses the reusable APIs to implement org invites, project authorization, audit browsing, share links, and one-time token flows in JavaScript.

The goal is not to make the core fully generic or policy-engine-heavy. The goal is a pragmatic auth toolkit: safe by default, small enough to understand, and flexible enough that users do not immediately reach for raw database handles.

## Current-state problem

### Generic hostauth currently mounts demo routes

`pkg/xgoja/hostauth/builder.go` currently installs native handlers for OIDC lifecycle and demo application behavior from the same generic function:

```text
GET  /auth/login
GET  /auth/callback
POST /auth/logout
GET  /auth/logout
GET  /auth/session
GET  /auth/audit              <-- demo/admin UI behavior
POST /orgs/o1/invites         <-- hard-coded demo org invite behavior
POST /org-invites/accept      <-- hard-coded demo invite acceptance behavior
```

The first five routes are generic OIDC/session lifecycle. The last three are application behavior. They should not be mounted by the reusable hostauth package because:

- they hard-code paths and domain concepts,
- they bypass JavaScript route ownership,
- they can shadow application routes because native handlers mount before app fallback,
- they encode demo policy in generic Go,
- they make it harder for users to see which routes their app owns.

### Appauth mixes useful interfaces with app-specific policy

`pkg/gojahttp/auth/appauth/appauth.go` contains useful concepts:

- user store interface,
- membership store interface,
- resource store interface,
- resource resolver,
- authorizer shape,
- authorization decision shape.

It also contains app-specific action names and policy:

```go
ActionUserSelfRead   = "user.self.read"
ActionProjectRead    = "project.read"
ActionProjectUpdate  = "project.update"
ActionOrgInvite      = "org.member.invite"
ActionAuditRead      = "audit.read"
```

The hard-coded switch that says project updates require `admin`/`editor`, org invites require `admin`, and audit reads require org `admin` is fine for the demo, but it is not a universal auth model. Real users will have actions such as:

- `invoice.read`,
- `workspace.member.invite`,
- `document.share`,
- `billing.plan.update`,
- `dataset.export`,
- `admin.impersonate`.

The generic core should make it easy to plug those actions into route authorization, not prescribe project/org semantics.

### Capability currently contains org-invite-specific helpers

The capability package is conceptually reusable: it stores issued, expiring, revocable, consumable tokens. But the current public helpers are centered on org invites:

```go
IssueOrgInvite(...)
AcceptOrgInvite(...)
OrgInviteSpec
```

Org invites are a good demo use case, but capability tokens are broader. Real users will want:

- email verification links,
- passwordless login links,
- invite links,
- share links,
- temporary upload/download grants,
- API bootstrap tokens,
- one-time approval tokens,
- cross-device pairing codes,
- account recovery tokens.

The core should expose generic capability primitives and let demo code implement org invites as one possible token type.

## Design goals

1. **Reusable core, not a demo framework.** Core packages should expose primitives that are useful across many app domains.
2. **Opinionated safety.** APIs should prefer bounded queries, explicit TTLs, explicit token types, explicit resource scopes, and no raw secrets in JS.
3. **Composable route ownership.** Application routes should be in JavaScript or app code. Native hostauth should only own host lifecycle endpoints.
4. **No accidental global admin endpoints.** Audit browsing and token issuance must be app routes with app-specific authorization.
5. **No raw DB as first resort.** Raw DB handles can exist later behind explicit guarded config, but common workflows should have safer APIs.
6. **Demos showcase real patterns.** Example 21 should be a high-quality cookbook for building auth flows using the reusable APIs.
7. **Small enough to learn.** Avoid introducing a full policy engine unless a concrete need appears.

## Proposed package boundaries

### Generic core should keep

| Area | Keep in reusable core? | Reason |
|---|---:|---|
| OIDC login/callback/logout/session handlers | Yes | Host lifecycle concern, not app-domain behavior. |
| Session manager/store | Yes | Core auth infrastructure. |
| CSRF verification | Yes | Core web auth safety primitive. |
| Audit record model and safe query | Yes | Operational platform concern. |
| Generic capability token store | Yes | Reusable token primitive. |
| Route DSL `.auth`, `.csrf`, `.resource`, `.allow`, `.audit` | Yes | Core app integration surface. |
| Host service injection | Yes | Runtime composition primitive. |
| JavaScript `auth.audit.query` | Yes | Safe high-level read API. |
| JavaScript `auth.capabilities.*` | Yes | Safe high-level token API. |

### Demo/starter code should own

| Area | Move out of generic hostauth? | Destination |
|---|---:|---|
| Native `/auth/audit` | Yes | JS route in example 21. |
| Native `/orgs/o1/invites` | Yes | JS route in example 21. |
| Native `/org-invites/accept` | Yes | JS route in example 21. |
| `IssueOrgInvite` / `AcceptOrgInvite` convenience helpers | Prefer yes | Demo helper or JS wrapper over generic capabilities. |
| `ActionProjectUpdate`, `ActionOrgInvite`, demo role policy | Prefer yes | Demo/starter policy package or example-local JS/Go. |
| Hard-coded resources `org`, `project`, `o1`, `p1` | Yes | Example fixtures. |

## Proposed JavaScript core API v2: fluent Go-backed builders

The first #85 implementation exposed an object-bag API:

```js
auth.audit.query({ tenantId: "o1", outcome: "denied", limit: 50 })
```

That proved the architecture, but it should not be the long-term shape. The reusable auth core should expose **fluent builder APIs backed by Go objects**. JavaScript should call typed methods one field at a time, then call `.run()` to execute. This keeps the public API ergonomic while avoiding defensive map/object decoding on the Go side.

The main benefits are:

- method names become the API surface instead of arbitrary object keys,
- Go receives typed method arguments instead of `map[string]any`,
- TypeScript declarations are straightforward and discoverable,
- validation errors happen close to the incorrect setter call,
- the builder owns defaulting, clamping, and final validation,
- object casing mismatches such as `tenantId` versus `TenantID` cannot silently widen a query,
- large or weird nested JS objects do not get passed through unless an explicit setter accepts them.

Go still validates inputs. Fluent builders do **not** make JavaScript statically safe at runtime. They do, however, reduce the attack surface and accidental misuse from "any object shape" to a small list of typed methods.

### Audit query cleanup: `auth.audit.query()` builder

Current API to replace:

```js
const records = auth.audit.query({
  tenantId: org.id,
  outcome: ctx.request.query.outcome,
  limit: 50,
});
```

Target API:

```js
const records = auth.audit.query()
  .tenantId(org.id)
  .outcome(ctx.request.query.outcome || "")
  .limit(50)
  .run();
```

Full example:

```js
const records = auth.audit.query()
  .tenantId(ctx.params.orgId)
  .outcome(ctx.request.query.outcome || "")
  .actorId(ctx.request.query.actorId || "")
  .resource("project", ctx.request.query.projectId || "")
  .limit(Number(ctx.request.query.limit || 50))
  .offset(Number(ctx.request.query.offset || 0))
  .run();
```

TypeScript shape:

```ts
interface AuthModule {
  audit: AuditAPI;
  capabilities: CapabilitiesAPI;
}

interface AuditAPI {
  query(): AuditQueryBuilder;
}

interface AuditQueryBuilder {
  tenantId(id: string): this;
  outcome(outcome: string): this;
  actorId(id: string): this;
  resource(type: string, id: string): this;
  resourceType(type: string): this;
  resourceId(id: string): this;
  limit(limit: number): this;
  offset(offset: number): this;
  run(): AuditRecord[];
}
```

Go implementation sketch:

```go
func newAuditQueryBuilder(vm *goja.Runtime, store audit.QueryStore, maxLimit int) *goja.Object {
    var q audit.Query
    obj := vm.NewObject()

    _ = obj.Set("tenantId", func(id string) *goja.Object {
        q.TenantID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("outcome", func(outcome string) *goja.Object {
        q.Outcome = strings.TrimSpace(outcome)
        return obj
    })
    _ = obj.Set("actorId", func(id string) *goja.Object {
        q.ActorID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("resource", func(typ, id string) *goja.Object {
        q.ResourceType = strings.TrimSpace(typ)
        q.ResourceID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("resourceType", func(typ string) *goja.Object {
        q.ResourceType = strings.TrimSpace(typ)
        return obj
    })
    _ = obj.Set("resourceId", func(id string) *goja.Object {
        q.ResourceID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("limit", func(limit int) *goja.Object {
        q.Limit = limit
        return obj
    })
    _ = obj.Set("offset", func(offset int) *goja.Object {
        q.Offset = offset
        return obj
    })
    _ = obj.Set("run", func() goja.Value {
        normalized := audit.NormalizeQuery(q, maxLimit)
        records, err := store.QueryAuditRecords(runtimebridge.CurrentOwnerContext(vm), normalized)
        if err != nil {
            panic(vm.NewGoError(err))
        }
        return vm.ToValue(recordsForJS(records))
    })
    return obj
}
```

Module export:

```go
_ = auditObj.Set("query", func() *goja.Object {
    return newAuditQueryBuilder(vm, queryStore, maxLimit)
})
```

Cleanup required from the current #85 implementation:

1. Keep `audit.Query` and `audit.QueryStore`; those are good internal Go contracts.
2. Replace `auth.audit.query(object)` in `pkg/xgoja/providers/hostauth/hostauth.go` with `auth.audit.query()` returning a builder.
3. Delete `queryFromValue`, `optionalString`, and `optionalInt`; they exist only for object-bag decoding.
4. Update `pkg/xgoja/providers/hostauth/hostauth_test.go` to call the builder chain.
5. Update `examples/xgoja/21-generated-host-auth/verbs/sites.js` to call the builder chain.
6. Add TypeScript declarations for the builder shape when the provider starts emitting DTS.

### Generic capability issue builder

Current object-bag idea to avoid:

```js
auth.capabilities.issue({
  type: "org-invite",
  subject: "email:invitee@example.test",
  resource: { type: "org", id: orgId, tenantId: orgId },
  claims: { role: "viewer" },
  ttlSeconds: 900,
  createdBy: ctx.actor.id,
});
```

Target API:

```js
const issued = auth.capabilities.issue("org-invite")
  .subject("email", email)
  .resource("org", orgId)
  .tenantId(orgId)
  .claimString("role", role)
  .ttlSeconds(900)
  .createdBy(ctx.actor.id)
  .run();
```

A different workflow uses the same generic builder:

```js
const share = auth.capabilities.issue("project-share-link")
  .resource("project", project.id)
  .tenantId(project.tenantId)
  .claimString("permission", "project.read")
  .claimBool("anonymous", true)
  .ttlSeconds(24 * 60 * 60)
  .createdBy(ctx.actor.id)
  .run();
```

TypeScript shape:

```ts
interface CapabilitiesAPI {
  issue(type: string): CapabilityIssueBuilder;
  validate(token: string): CapabilityValidateBuilder;
  consume(token: string): CapabilityConsumeBuilder;
  revoke(): CapabilityRevokeBuilder;
}

interface CapabilityIssueBuilder {
  subject(kind: string, value: string): this;
  subjectRaw(subject: string): this;
  resource(type: string, id: string): this;
  tenantId(id: string): this;
  claimString(key: string, value: string): this;
  claimNumber(key: string, value: number): this;
  claimBool(key: string, value: boolean): this;
  metadataString(key: string, value: string): this;
  ttlSeconds(seconds: number): this;
  createdBy(actorId: string): this;
  run(): IssuedCapability;
}
```

Go implementation sketch:

```go
func newCapabilityIssueBuilder(vm *goja.Runtime, service capability.Service, cfg CapabilityConfig, capType string) *goja.Object {
    spec := capability.IssueSpec{
        Type:     strings.TrimSpace(capType),
        Claims:   map[string]any{},
        Metadata: map[string]any{},
    }
    obj := vm.NewObject()

    _ = obj.Set("subject", func(kind, value string) *goja.Object {
        spec.Subject = strings.TrimSpace(kind) + ":" + strings.TrimSpace(value)
        return obj
    })
    _ = obj.Set("subjectRaw", func(subject string) *goja.Object {
        spec.Subject = strings.TrimSpace(subject)
        return obj
    })
    _ = obj.Set("resource", func(typ, id string) *goja.Object {
        spec.ResourceType = strings.TrimSpace(typ)
        spec.ResourceID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("tenantId", func(id string) *goja.Object {
        spec.TenantID = strings.TrimSpace(id)
        return obj
    })
    _ = obj.Set("claimString", func(key, value string) *goja.Object {
        setClaim(spec.Claims, key, value)
        return obj
    })
    _ = obj.Set("claimNumber", func(key string, value float64) *goja.Object {
        setClaim(spec.Claims, key, value)
        return obj
    })
    _ = obj.Set("claimBool", func(key string, value bool) *goja.Object {
        setClaim(spec.Claims, key, value)
        return obj
    })
    _ = obj.Set("ttlSeconds", func(seconds int) *goja.Object {
        spec.TTL = time.Duration(seconds) * time.Second
        return obj
    })
    _ = obj.Set("createdBy", func(actorID string) *goja.Object {
        spec.CreatedBy = strings.TrimSpace(actorID)
        return obj
    })
    _ = obj.Set("run", func() goja.Value {
        issued, err := service.Issue(runtimebridge.CurrentOwnerContext(vm), normalizeIssueSpec(spec, cfg))
        if err != nil {
            panic(vm.NewGoError(err))
        }
        return vm.ToValue(issuedForJS(issued))
    })
    return obj
}
```

Setter validation should reject empty claim keys, unsupported token types, invalid TTLs, or oversized metadata/claims early where possible. Final validation still runs in `.run()` because some invariants require the full spec.

### Capability validate / consume builders

Validation checks a token without consuming it:

```js
const preview = auth.capabilities.validate(token)
  .expectedType("org-invite")
  .expectedResource("org", orgId)
  .run();
```

Consumption atomically marks a one-time token used:

```js
const accepted = auth.capabilities.consume(token)
  .expectedType("org-invite")
  .expectedResource("org", orgId)
  .run();
```

TypeScript shape:

```ts
interface CapabilityValidateBuilder {
  expectedType(type: string): this;
  expectedResource(type: string, id: string): this;
  run(): CapabilityValidation;
}

interface CapabilityConsumeBuilder {
  expectedType(type: string): this;
  expectedResource(type: string, id: string): this;
  run(): ConsumedCapability;
}
```

`validate(...).run()` should return structured invalid results rather than throw for normal invalid-token states:

```js
{ valid: false, reason: "expired" }
```

`consume(...).run()` can throw structured errors for states the caller must handle:

```js
try {
  const accepted = auth.capabilities.consume(token).expectedType("org-invite").run();
} catch (e) {
  if (e.code === "capability_used") ...
  if (e.code === "capability_expired") ...
}
```

### Capability revoke builder

```js
auth.capabilities.revoke()
  .id(capabilityId)
  .reason("user_request")
  .run();
```

or:

```js
auth.capabilities.revoke()
  .token(token)
  .reason("rotated")
  .run();
```

TypeScript shape:

```ts
interface CapabilityRevokeBuilder {
  id(id: string): this;
  token(token: string): this;
  reason(reason: string): this;
  run(): void;
}
```

### Capability listing remains optional and should also be fluent

Listing capabilities creates more information-disclosure risk than issuing or consuming a specific token. Delay it until a concrete UI needs it. If added, keep it fluent and bounded:

```js
const links = auth.capabilities.list()
  .type("project-share-link")
  .resource("project", project.id)
  .includeExpired(false)
  .limit(50)
  .run();
```

## Proposed Go core API

The current capability store/service should be generalized around a shape like this.

```go
type Capability struct {
    ID           string
    Type         string
    Subject      string
    ResourceType string
    ResourceID   string
    TenantID     string
    Claims       map[string]any
    Metadata     map[string]any
    CreatedBy    string
    CreatedAt    time.Time
    ExpiresAt    time.Time
    UsedAt       *time.Time
    RevokedAt    *time.Time
}

type IssueSpec struct {
    Type       string
    Subject    string
    Resource   ResourceRef
    Claims     map[string]any
    Metadata   map[string]any
    TTL        time.Duration
    CreatedBy  string
}

type ValidateSpec struct {
    Token        string
    ExpectedType string
}

type ConsumeSpec struct {
    Token        string
    ExpectedType string
}

type RevokeSpec struct {
    ID     string
    Token  string
    Reason string
}
```

Service methods:

```go
type Service struct { Store Store }

func (s Service) Issue(ctx context.Context, spec IssueSpec) (Issued, error)
func (s Service) Validate(ctx context.Context, spec ValidateSpec) (Validation, error)
func (s Service) Consume(ctx context.Context, spec ConsumeSpec) (Consumed, error)
func (s Service) Revoke(ctx context.Context, spec RevokeSpec) error
```

Store methods:

```go
type Store interface {
    Insert(ctx context.Context, cap Capability, tokenHash string) error
    ByTokenHash(ctx context.Context, tokenHash string) (*Capability, error)
    MarkUsed(ctx context.Context, id string, usedAt time.Time) (*Capability, error)
    Revoke(ctx context.Context, id string, revokedAt time.Time, reason string) error
}
```

Implementation details:

- Continue storing only token hashes, never raw tokens.
- Return raw token only from `Issue`.
- Use atomic `MarkUsed` for one-time consumption where possible.
- Preserve current SQL/memory stores but rename org-invite helpers into demo wrappers.

## App authorization cleanup

### Keep the generic route contract

The route DSL should remain the center of authorization composition:

```js
app.post("/workspaces/:workspaceId/invites")
  .auth(express.user().required())
  .resource(express.resource("workspace").idFromParam("workspaceId").mustExist())
  .csrf()
  .allow("workspace.member.invite")
  .audit("workspace.invite.issued")
  .handle(...)
```

This pattern is good because it separates concerns:

- `.auth(...)` establishes actor/session.
- `.resource(...)` establishes resource context.
- `.csrf()` protects state-changing browser requests.
- `.allow(...)` delegates policy to the configured authorizer.
- `.audit(...)` records the action and outcome.
- handler code performs business behavior.

### Split `appauth` into reusable interfaces and starter policy

Recommended package split:

```text
pkg/gojahttp/auth/policy/          # reusable interfaces and request/decision helpers
pkg/gojahttp/auth/appauth/         # maybe retained for starter app stores
pkg/gojahttp/auth/appauth/starter/ # optional demo/starter hard-coded policy
examples/xgoja/21.../demoauth/     # example-specific policy/fixtures, if Go helper needed
```

Possible reusable policy package:

```go
type Authorizer interface {
    Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error)
}

type ResourceStore interface {
    GetResource(ctx context.Context, typ, id string) (*Resource, error)
}

type MembershipStore interface {
    MembershipsForUser(ctx context.Context, userID string) ([]Membership, error)
    IsMember(ctx context.Context, userID, tenantID string) (bool, error)
    HasRole(ctx context.Context, userID, tenantID string, roles ...string) (bool, error)
}
```

Starter/demo policy can still provide:

```go
const (
    ActionProjectUpdate = "project.update"
    ActionOrgInvite     = "org.member.invite"
    ActionAuditRead     = "audit.read"
)
```

But those constants should not be presented as the universal auth model.

## Example 21 demo architecture

The demo should become a showcase of real-world patterns built on reusable core APIs.

### Demo goals

Example 21 should demonstrate:

1. OIDC login/logout/session.
2. Protected self route (`/me`).
3. Resource-scoped project route.
4. Audit browser route using `auth.audit.query`.
5. One-time org invite flow using `auth.capabilities.issue` and `consume`.
6. Validate-before-consume invite preview using `auth.capabilities.validate`.
7. Optional share link flow to demonstrate a second capability type.
8. Dashboard UI backed by embedded split assets.

### Demo code organization

Keep demo logic in example-local JavaScript modules:

```text
examples/xgoja/21-generated-host-auth/
  verbs/
    sites.js                 # route registration
    demo-policy.js           # action names, role checks if JS-side helpers exist
    invites.js               # org invite helper over auth.capabilities
    share-links.js           # optional second capability example
  assets/public/
    index.html
    app.js
    styles.css
```

If route DSL policy still uses Go `appauth.Authorizer`, then fixture/policy setup remains in Go store config. But path/domain behavior should still be in JS routes.

### Demo invite helper in JavaScript

```js
function issueOrgInvite(auth, ctx) {
  const body = ctx.request.body || {};
  const orgId = ctx.params.orgId;
  const role = normalizeInviteRole(body.role);
  return auth.capabilities.issue("org-invite")
    .subject("email", String(body.email || "").trim().toLowerCase())
    .resource("org", orgId)
    .tenantId(orgId)
    .claimString("role", role)
    .metadataString("demo", "true")
    .ttlSeconds(15 * 60)
    .createdBy(ctx.actor.id)
    .run();
}

function acceptOrgInvite(auth, token, orgId) {
  const consumed = auth.capabilities.consume(token)
    .expectedType("org-invite")
    .expectedResource("org", orgId)
    .run();
  return {
    orgId: consumed.capability.resource.id,
    role: consumed.capability.claims.role,
    subject: consumed.capability.subject,
  };
}
```

Routes:

```js
app.post("/orgs/:orgId/invites")
  .auth(express.user().required())
  .resource(express.resource("org").idFromParam("orgId").mustExist())
  .csrf()
  .allow("org.member.invite")
  .audit("org.invite.issued")
  .handle((ctx, res) => res.json(issueOrgInvite(auth, ctx)));

app.get("/org-invites/preview")
  .public()
  .audit("org.invite.previewed")
  .handle((ctx, res) => {
    const result = auth.capabilities.validate(ctx.request.query.token)
      .expectedType("org-invite")
      .run();
    res.json(result);
  });

app.post("/org-invites/accept")
  .public()
  .audit("org.invite.accepted")
  .handle((ctx, res) => res.json(acceptOrgInvite(auth, ctx.request.body.token, ctx.request.body.orgId)));
```

### Demo share link flow

A second capability type would prove the API is not invite-specific:

```js
app.post("/orgs/:orgId/projects/:projectId/share-links")
  .auth(express.user().required())
  .resource(express.resource("project").idFromParam("projectId").tenantFromParam("orgId").mustExist())
  .csrf()
  .allow("project.share")
  .audit("project.share_link.issued")
  .handle((ctx, res) => {
    const project = ctx.resource("project");
    const issued = auth.capabilities.issue("project-share-link")
      .resource("project", project.id)
      .tenantId(project.tenantId)
      .claimString("permission", "project.read")
      .claimBool("anonymous", true)
      .ttlSeconds(24 * 60 * 60)
      .createdBy(ctx.actor.id)
      .run();
    res.json(issued);
  });

app.get("/share/:token")
  .public()
  .audit("project.share_link.viewed")
  .handle((ctx, res) => {
    const result = auth.capabilities.validate(ctx.params.token)
      .expectedType("project-share-link")
      .run();
    if (!result.valid) return res.status(404).json({ error: result.reason });
    res.json({ projectId: result.capability.resource.id, claims: result.capability.claims });
  });
```

This demonstrates reusable tokens without adding a new Go helper for every domain concept.

## Raw DB handles remain a separate advanced feature

Raw host DB handles still have a place, especially for reporting, migrations, admin tooling, and application-specific queries. But they should not be the primary answer for auth workflows.

If implemented, raw handles should be explicit and guarded:

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: database
      as: auditdb
      config:
        hostHandle: auth.audit
        access: read-only
        allowedTables: [auth_audit_records]
        maxRows: 100
        timeout: 2s
```

The important principle: requesting raw DB access means the user is advanced, but it does not mean the resulting route is safe. Safer high-level APIs should cover common workflows first.

## Implementation plan

### Phase 0: Clean up the current audit query API to a fluent builder

The current implementation has already proven the host service lookup, audit store query contract, and JS module wiring. Before expanding the auth module with capabilities, refactor the audit API from object-bag decoding to a fluent builder:

```js
// Replace this
auth.audit.query({ tenantId: org.id, limit: 50 });

// With this
auth.audit.query().tenantId(org.id).limit(50).run();
```

Implementation tasks:

1. Add `newAuditQueryBuilder` in `pkg/xgoja/providers/hostauth/hostauth.go`.
2. Change `auth.audit.query` to return a builder object instead of accepting an options object.
3. Remove JS object decoding helpers from the module.
4. Keep `audit.Query`/`audit.QueryStore` as the Go-side execution contract.
5. Update provider tests and example 21 route code.
6. Document the builder API in generated TypeScript metadata when available.

This should happen before adding `auth.capabilities.*`, so all auth module APIs share the same builder style.

### Phase 1: #86 native route cleanup

Remove from `pkg/xgoja/hostauth/builder.go` generic native handlers:

- `GET /auth/audit`,
- `POST /orgs/o1/invites`,
- `POST /org-invites/accept`.

Keep:

- `GET /auth/login`,
- `GET /auth/callback`,
- `POST /auth/logout`,
- `GET /auth/logout`,
- `GET /auth/session`.

Update tests in `pkg/xgoja/hostauth/builder_test.go` to expect only lifecycle/session handlers.

Example 21 already has JS-owned audit browsing through `/orgs/:orgId/audit`.

### Phase 2: Generic capability service API

Refactor `pkg/gojahttp/auth/capability` to support generic issue/validate/consume/revoke operations.

Preserve store compatibility where possible, but move org-invite naming out of the generic API.

Suggested migration:

1. Add generic types and methods alongside existing org-invite helpers.
2. Reimplement org-invite helpers as thin wrappers or move them to demo helpers.
3. Update tests to cover multiple token types.
4. Mark old org-invite helpers as demo-only or remove if no external compatibility is required.

### Phase 3: Expose `auth.capabilities.*` in JS

Extend `pkg/xgoja/providers/hostauth/hostauth.go`:

```js
auth.capabilities.issue(type).resource(...).ttlSeconds(...).run()
auth.capabilities.validate(token).expectedType(...).run()
auth.capabilities.consume(token).expectedType(...).run()
auth.capabilities.revoke().id(...).reason(...).run()
```

Config:

```yaml
runtime:
  modules:
    - provider: go-go-goja-hostauth
      name: auth
      as: auth
      config:
        audit:
          maxLimit: 50
        capabilities:
          maxTTLSeconds: 86400
          maxClaimsBytes: 8192
          allowedTypes:
            - org-invite
            - project-share-link
```

### Phase 4: Move invite demo into example JS

Update `examples/xgoja/21-generated-host-auth/verbs/sites.js` or helper modules:

- `POST /orgs/:orgId/invites` implemented in JS with `auth.capabilities.issue(...).run()`.
- `GET /org-invites/preview` implemented in JS with `auth.capabilities.validate(...).run()`.
- `POST /org-invites/accept` implemented in JS with `auth.capabilities.consume(...).run()`.

Update dashboard JS to call those routes.

### Phase 5: Appauth split / renaming

Decide whether to:

1. Keep `appauth` as a starter appauth package but document it clearly as starter/demo policy, or
2. Split reusable policy interfaces into a more neutral package and move hard-coded actions into a starter/demo package.

Recommended short-term approach:

- keep current interfaces to avoid churn,
- move hard-coded action constants and authorizer switch out of the generic path once example/demo has its own policy package,
- document `appauth.Authorizer` as a starter policy, not a universal policy engine.

### Phase 6: Rich demo polish

Enhance example 21 dashboard to show:

- current session,
- protected `/me`,
- project update success/failure,
- audit browser with outcome filter,
- issue invite,
- preview invite,
- accept invite,
- issue/share project link,
- validate/share link.

The demo should clearly show that all app-domain routes are JS-owned.

## Decision records

### Decision: native hostauth owns only auth lifecycle routes

- **Context:** Generic hostauth currently mounts demo routes.
- **Decision:** Keep only OIDC/session lifecycle native routes in hostauth.
- **Rationale:** App-domain behavior belongs to applications and demos.
- **Consequence:** Demos must implement audit/invite/share routes themselves using public APIs.
- **Status:** proposed.

### Decision: generic capabilities before raw DB handles

- **Context:** Users may ask for raw DB access to build token workflows.
- **Decision:** Provide `auth.capabilities.*` first for common safe token workflows.
- **Rationale:** Capability issuance/consumption is a common auth primitive that can be made safer than raw SQL.
- **Consequence:** Raw DB handles remain advanced/optional.
- **Status:** proposed.

### Decision: invite helpers are demo sugar, not core API

- **Context:** `IssueOrgInvite` is convenient but domain-specific.
- **Decision:** Implement invites in demo code on top of generic capabilities.
- **Rationale:** Real applications need arbitrary token types and claims.
- **Consequence:** Example 21 gets slightly more JS helper code, but the core becomes cleaner.
- **Status:** proposed.

### Decision: route DSL remains the authorization boundary

- **Context:** Capability APIs could hide authorization checks internally.
- **Decision:** Keep authorization explicit on routes with `.auth`, `.resource`, `.csrf`, `.allow`, `.audit`.
- **Rationale:** It is clearer, composable, auditable, and matches current xgoja route planning.
- **Consequence:** JS handlers must be written with the route chain; capability APIs validate token mechanics, not caller permissions.
- **Status:** proposed.

### Decision: JS auth APIs use fluent Go-backed builders instead of object bags

- **Context:** Object-bag APIs are easy to call but force Go to defensively decode arbitrary JS objects and nested maps.
- **Decision:** Use fluent builders for `auth.audit.query`, `auth.capabilities.issue`, `validate`, `consume`, and `revoke`.
- **Rationale:** Builders give users readable APIs while letting Go accept typed setter arguments, enforce method-level validation, and avoid casing/nesting bugs.
- **Consequence:** The API is a little more verbose, but safer, easier to document in TypeScript, and consistent with the existing route-builder style.
- **Status:** proposed.

## Test strategy

### Core tests

- `hostauth.BuildNativeHandlers` only returns lifecycle/session routes.
- `auth.capabilities.issue(type).ttlSeconds(...).run()` clamps TTL and rejects invalid type/claims.
- `auth.capabilities.validate(token).expectedType(...).run()` returns structured invalid reasons.
- `auth.capabilities.consume(token).expectedType(...).run()` is one-time and maps used-token errors.
- `auth.capabilities.revoke().id(...).run()` prevents future validation/consumption.
- Memory and SQL capability stores satisfy the same contract.

### JS module tests

- Runtime can `require("auth").capabilities.issue("type").ttlSeconds(...).run()`.
- JS cannot issue a token with disallowed type if config has `allowedTypes`.
- JS cannot exceed max TTL, even when setting TTL through the builder.
- JS receives no token hash or store internals.
- JS errors include stable `code` fields.

### Example tests

- Example 21 `xgoja doctor` resolves providers.
- Example 21 build succeeds.
- Smoke verifies dashboard references JS-owned routes.
- Unauthenticated audit/invite issue routes return `401`.
- Public invite preview/accept routes are app-owned.
- Optional authenticated smoke verifies real invite issue/accept if session setup is available.

## Risks and mitigations

### Risk: API becomes too generic and unsafe

Mitigation: Keep capability APIs method-based, field-based, and policy-bounded. Avoid arbitrary SQL, arbitrary store access, object-bag passthrough, or unbounded listing.

### Risk: API becomes too specific again

Mitigation: Keep domain terms out of core method names. Token type and claims are caller-defined.

### Risk: users confuse token validation with authorization

Mitigation: Document that `auth.capabilities.*` validates token mechanics only. Route authorization remains `.auth/.resource/.allow`.

### Risk: appauth churn breaks existing examples

Mitigation: Move in phases. First remove native routes. Then add generic capability API. Then move starter policy.

## Open questions

1. Should `auth.capabilities.validate(token).run()` throw errors or return `{ valid:false, reason }`? Recommendation: return structured invalid result for validate, throw for issue/consume/revoke failures.
2. Should `auth.capabilities.consume(token)` require expected resource as well as expected type? Recommendation: support optional `.expectedResource(type, id)` and encourage demos to use it.
3. Should capability claims be arbitrary JSON or restricted to typed setter methods initially? Recommendation: start with typed setters (`claimString`, `claimNumber`, `claimBool`) and add explicit JSON support later only if needed.
4. Should appauth be renamed/split now or after #86? Recommendation: after #86 and generic capabilities, to avoid too much churn at once.
5. Should demo helper modules be JS files only or a Go demo provider package? Recommendation: JS first; only add Go demo helpers if JS cannot express setup/fixtures cleanly.

## References

- GitHub issue #85: <https://github.com/go-go-golems/go-go-goja/issues/85>
- GitHub issue #86: <https://github.com/go-go-golems/go-go-goja/issues/86>
- `pkg/xgoja/hostauth/builder.go`
- `pkg/xgoja/providers/hostauth/hostauth.go`
- `pkg/gojahttp/auth/appauth/appauth.go`
- `pkg/gojahttp/auth/capability/capability.go`
- `pkg/gojahttp/auth/audit/audit.go`
- `examples/xgoja/21-generated-host-auth/verbs/sites.js`
- `examples/xgoja/21-generated-host-auth/xgoja.yaml`
