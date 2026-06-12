---
Title: Preliminary auth API ideas
Ticket: XGOJA-EXPRESS-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/auth.md
Summary: "Imported preliminary analysis of possible authentication API ideas for go-go-goja Express."
LastUpdated: 2026-06-12T14:25:00-04:00
WhatFor: "Source material for reconciling broad auth API ideas with the current Express HTTP module design."
WhenToUse: "Read when comparing the MVP design against the original API exploration."
---

A good design would make the JS API feel ergonomic, but make the **security-critical parts declarative and Go-owned**.

The main rule: user scripts should **describe intent**, not perform raw auth checks themselves. Go should compile that intent into a route plan, validate it, and enforce it on every request. This matches OWASP’s access-control guidance: deny by default and validate authorization on every request. ([OWASP Cheat Sheet Series][1])

Also, since goja is an ECMAScript 5.1+ engine embedded in Go, do not assume Node/Express semantics unless you provide them. Expose a narrow host API. ([GitHub][2])

---

## Core shape

Instead of this:

```js
app.post("/users/:id", function (req, res) {
  if (req.user.role === "admin") {
    db.users.update(req.params.id, req.body);
  }
});
```

Aim for this:

```js
api.post("/users/:userId")
  .name("user.update")
  .auth(user().required().mfaFresh("10m"))
  .resource(userResource().fromParam("userId").withinTenant())
  .allow("user.update")
  .body(schema.userUpdate())
  .audit("user.updated")
  .handle(function (ctx) {
    return ctx.users
      .update(ctx.resource("user"))
      .setProfile(ctx.body)
      .commit();
  });
```

The important part is that `.handle()` is not registering arbitrary code on a naked route. It registers a route with a compiled policy envelope:

```js
{
  method: "POST",
  path: "/users/:userId",
  auth: {
    kind: "user",
    required: true,
    mfaFreshWithin: "10m"
  },
  resource: {
    type: "user",
    source: "param:userId",
    tenantBound: true
  },
  action: "user.update",
  bodySchema: "userUpdate",
  audit: "user.updated"
}
```

The handler only runs after Go has:

```text
loaded session
loaded actor
loaded resource
checked tenant boundary
checked permission
validated body
created audit context
```

---

# Idea 1: staged route builder

Make routes impossible to register unless they choose one of:

```text
.public()
.auth(...)
.system(...)
.capability(...)
```

Example:

```js
api.get("/me")
  .auth(user().required())
  .allow("user.self.read")
  .handle(function (ctx) {
    return ctx.users.me().profile();
  });

api.post("/logout")
  .auth(user().required())
  .csrf()
  .handle(function (ctx) {
    return ctx.session.logout();
  });

api.get("/health")
  .public()
  .cache("30s")
  .handle(function () {
    return { ok: true };
  });
```

Bad patterns should fail at route-registration time:

```js
api.get("/admin").handle(fn);
// Error: route must declare .public(), .auth(), .system(), or .capability()

api.post("/users/:id")
  .auth(user().required())
  .handle(fn);
// Error: mutating authenticated route must declare .allow(action)
```

This is probably the simplest “make wrong things hard” layer.

Implementation idea in Go:

```go
type RouteNeedsAuth struct {
    plan *RoutePlan
}

type RouteNeedsPolicy struct {
    plan *RoutePlan
}

type RouteNeedsHandler struct {
    plan *RoutePlan
}

func (r *RouteNeedsAuth) Auth(spec AuthSpec) *RouteNeedsPolicy
func (r *RouteNeedsAuth) Public() *RouteNeedsHandler

func (r *RouteNeedsPolicy) Allow(action string) *RouteNeedsHandler

func (r *RouteNeedsHandler) Handle(call goja.Callable) error
```

In JavaScript this is dynamic, so it is not compile-time type safety. But you can expose different Go-backed objects at each stage so the wrong method simply is not present.

---

# Idea 2: contract-first route API

This is less Express-like, but more auditable.

```js
api.route("POST", "/orgs/:orgId/invites")
  .contract({
    name: "org.invite_user",

    auth: user()
      .required()
      .emailVerified()
      .mfaFresh("15m"),

    tenant: tenant()
      .fromParam("orgId"),

    resource: org()
      .fromParam("orgId"),

    allow: "org.member.invite",

    body: schema.object({
      email: schema.email(),
      role: schema.enum("viewer", "editor")
    }),

    audit: "org.invite.created"
  })
  .handle(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .expiresIn("7d")
      .send();
  });
```

Pros:

```text
very clear
easy to lint
easy to generate docs from
easy to test
easy to reject unsafe routes
```

Cons:

```text
less fluent
less “Express-like”
large route definitions can feel verbose
```

I would consider this for admin APIs, billing APIs, or anything security-sensitive.

---

# Idea 3: operation builders instead of database builders

Do not expose:

```js
ctx.db.users.update(userId, patch);
ctx.db.sessions.delete(sessionId);
ctx.db.roles.insert(...);
```

Expose operations:

```js
ctx.users.me()
  .changeDisplayName("Manuel")
  .commit();

ctx.users.byId(ctx.params.userId)
  .require("user.suspend")
  .suspend()
  .reason(ctx.body.reason)
  .commit();

ctx.users.byEmail(ctx.body.email)
  .withinTenant(ctx.tenant)
  .invite()
  .role("editor")
  .expiresIn("7d")
  .send();
```

The operation builder can enforce required fields:

```js
ctx.users.byId(id)
  .suspend()
  .commit();
// Error: missing reason

ctx.users.byId(id)
  .delete();
// Error: hard delete unavailable; use .deactivate() or .scheduleDeletion()
```

For user management, this is a strong pattern. Make “dangerous” things explicit:

```js
ctx.users.byId(id)
  .dangerouslyRevokeAllSessions()
  .because("account compromise")
  .auditAs("security.response")
  .commit();
```

That makes reviewers notice the operation.

---

# Idea 4: resource-first authorization

This is good when most bugs are object-ownership bugs.

```js
api.patch("/projects/:projectId")
  .auth(user().required())
  .resource(project().fromParam("projectId").withinTenant())
  .allow("project.update")
  .body(schema.projectPatch())
  .handle(function (ctx) {
    return ctx.projects
      .update(ctx.resource("project"))
      .patch(ctx.body)
      .commit();
  });
```

Here, `ctx.resource("project")` is not just `{ id: "123" }`. It is a Go-owned object that has already been loaded and authorized.

Avoid:

```js
ctx.projects.update(ctx.params.projectId, ctx.body);
```

Prefer:

```js
ctx.projects.update(ctx.resource("project")).patch(ctx.body).commit();
```

This prevents the common bug where the handler checks access to one object but mutates another.

---

# Idea 5: current-user API

For “me” operations, remove target ambiguity.

```js
api.patch("/me/profile")
  .auth(user().required())
  .allow("user.self.update")
  .body(schema.profilePatch())
  .handle(function (ctx) {
    return ctx.me()
      .profile()
      .patch(ctx.body)
      .commit();
  });
```

For email change:

```js
api.post("/me/email-change")
  .auth(user().required().mfaFresh("10m"))
  .allow("user.self.email.change.request")
  .body(schema.object({
    email: schema.email()
  }))
  .handle(function (ctx) {
    return ctx.me()
      .email()
      .changeTo(ctx.body.email)
      .requiresConfirmation()
      .send();
  });
```

Do not expose this as:

```js
ctx.users.updateEmail(ctx.user.id, email);
```

Make the semantic operation explicit:

```js
ctx.me().email().changeTo(email).requiresConfirmation().send();
```

---

# Idea 6: capability builder

Capabilities should be separate from normal permissions. They are useful for email verification, password reset, invites, magic links, scoped API tokens, file downloads, and one-time approvals.

Example: email verification.

```js
ctx.capabilities.issue("user.email.verify")
  .subject(ctx.actor)
  .bind("email", ctx.body.email)
  .ttl("30m")
  .singleUse()
  .deliver("email")
  .commit();
```

Example: organization invite.

```js
ctx.capabilities.issue("org.invite.accept")
  .resource(ctx.resource("org"))
  .claim("email", ctx.body.email)
  .claim("role", ctx.body.role)
  .ttl("7d")
  .singleUse()
  .revocable()
  .commit();
```

Example: API token.

```js
ctx.apiTokens.issue()
  .forUser(ctx.actor)
  .name(ctx.body.name)
  .allow("project.read")
  .on(ctx.resource("project"))
  .expiresIn("90d")
  .commit();
```

Rules I would enforce:

```text
no capability without purpose
no capability without resource or subject
no capability without expiry
single-use by default for links
revocable by default
token secret returned once
token value never logged
```

Bad:

```js
ctx.capabilities.issue("*").forever();
```

Better: do not expose that method.

---

# Idea 7: route-level intent / command API

This is my preferred design for serious apps.

Routes bind HTTP to named commands:

```js
api.command("InviteOrgMember")
  .auth(user().required().mfaFresh("10m"))
  .resource(org().fromParam("orgId"))
  .allow("org.member.invite")
  .body(schema.inviteOrgMember())
  .audit("org.member.invited")
  .run(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .send();
  });

api.post("/orgs/:orgId/invites")
  .useCommand("InviteOrgMember");
```

The route becomes thin. The command becomes the security boundary.

Pros:

```text
easy to test commands outside HTTP
clear audit model
good for background jobs too
easy to version
easy to generate docs
```

A command can also declare its side effects:

```js
api.command("SuspendUser")
  .auth(user().required().mfaFresh("10m"))
  .resource(userResource().fromParam("userId").withinTenant())
  .allow("user.suspend")
  .body(schema.object({
    reason: schema.string().min(10)
  }))
  .effects([
    "user.status.change",
    "session.revoke",
    "audit.write"
  ])
  .run(function (ctx) {
    return ctx.users.byResource(ctx.resource("user"))
      .suspend()
      .reason(ctx.body.reason)
      .revokeSessions()
      .commit();
  });
```

This is more opinionated than Express, but much safer.

---

# Idea 8: tenant-scoped builder

If the app is multi-tenant, make tenant context mandatory.

```js
api.post("/tenants/:tenantId/users")
  .auth(user().required())
  .tenant(tenant().fromParam("tenantId"))
  .allow("tenant.user.invite")
  .body(schema.inviteUser())
  .handle(function (ctx) {
    return ctx.tenant.users
      .invite(ctx.body.email)
      .role(ctx.body.role)
      .send();
  });
```

Inside the handler, all tenant operations are already scoped:

```js
ctx.tenant.users.list();
ctx.tenant.projects.create(...);
ctx.tenant.audit.write(...);
```

Avoid global APIs:

```js
ctx.users.list();
ctx.projects.get(id);
```

Prefer scoped APIs:

```js
ctx.tenant.users.list();
ctx.tenant.projects.get(id);
```

For cross-tenant operations, force explicit naming:

```js
ctx.platform.tenants
  .dangerouslyCrossTenant()
  .require("platform.support.impersonate")
  .because(ctx.body.reason)
  .impersonate(targetTenant)
  .commit();
```

---

# Idea 9: relationship-based permissions

If your app has sharing, teams, folders, projects, inherited permissions, or “user can view document if they can view parent folder,” consider modeling authorization as relationships.

OpenFGA’s model is built around answering checks based on relationships between users and objects, and relationship-based access control handles cases like access inherited from parent objects. ([openfga.dev][3]) ([openfga.dev][4])

Your JS API could stay simple:

```js
api.get("/documents/:docId")
  .auth(user().required())
  .resource(document().fromParam("docId"))
  .allow("document.read")
  .handle(function (ctx) {
    return ctx.documents.read(ctx.resource("document"));
  });
```

But Go maps that to:

```text
check:
  subject = user:123
  relation/action = reader
  object = document:456
```

This lets you change the policy backend without changing plugin authors’ route code.

---

# Idea 10: policy registry

Another style: define policies once, reference them everywhere.

```js
security.policy("CanInviteOrgMember")
  .actor(user().required().mfaFresh("10m"))
  .resource(org().fromParam("orgId"))
  .allow("org.member.invite")
  .audit("org.member.invited");

api.post("/orgs/:orgId/invites")
  .usePolicy("CanInviteOrgMember")
  .body(schema.inviteOrgMember())
  .handle(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .send();
  });
```

Pros:

```text
less repetition
central policy review
good for larger apps
```

Cons:

```text
route behavior becomes less local
policy indirection can hide important details
```

Good compromise: allow `.usePolicy(...)`, but render expanded route plans in an admin/debug view.

---

# What I would not expose

Avoid these in the JS API:

```js
ctx.rawAccessToken
ctx.rawRefreshToken
ctx.sessionId
ctx.setCookie(...)
ctx.db.query(...)
ctx.user.roles.includes("admin")
ctx.auth.can("thing") // no resource
ctx.users.delete(id)
ctx.users.setRole(id, role)
ctx.impersonate(id)
```

Prefer:

```js
ctx.actor
ctx.session.logout()
ctx.resource("project")
ctx.authorized("project.update", ctx.resource("project"))
ctx.users.byResource(user).deactivate().reason(...).commit()
ctx.tenant.memberships.changeRole(...).because(...).commit()
```

The difference is that the second group preserves context, resource binding, auditability, and intent.

---

# A possible final API shape

This is the style I would probably ship:

```js
var api = require("app").api;
var schema = require("app").schema;
var user = require("app").user;
var org = require("app").org;

api.post("/orgs/:orgId/invites")
  .name("org.invite")
  .auth(user.session()
    .required()
    .emailVerified()
    .mfaFresh("10m"))
  .tenant()
    .fromParam("orgId")
  .resource("org")
    .fromParam("orgId")
    .mustExist()
  .allow("org.member.invite")
  .body(schema.object({
    email: schema.email(),
    role: schema.enum("viewer", "editor")
  }).closed())
  .rateLimit("org.invite", {
    perUser: "20/hour",
    perOrg: "100/day"
  })
  .audit("org.member.invited")
  .handle(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .expiresIn("7d")
      .send();
  });
```

For “me”:

```js
api.patch("/me/profile")
  .name("me.profile.update")
  .auth(user.session().required())
  .allow("user.self.update")
  .body(schema.object({
    displayName: schema.string().min(1).max(80)
  }).closed())
  .audit("user.profile.updated")
  .handle(function (ctx) {
    return ctx.me()
      .profile()
      .setDisplayName(ctx.body.displayName)
      .commit();
  });
```

For admin:

```js
api.post("/users/:userId/suspend")
  .name("user.suspend")
  .auth(user.session()
    .required()
    .mfaFresh("5m"))
  .resource("user")
    .fromParam("userId")
    .withinTenant()
    .mustExist()
  .allow("user.suspend")
  .body(schema.object({
    reason: schema.string().min(10)
  }).closed())
  .audit("user.suspended")
  .handle(function (ctx) {
    return ctx.users
      .byResource(ctx.resource("user"))
      .suspend()
      .reason(ctx.body.reason)
      .revokeSessions()
      .commit();
  });
```

For capability redemption:

```js
api.post("/invites/accept")
  .name("org.invite.accept")
  .capability("org.invite.accept")
    .fromBody("token")
    .singleUse()
    .notExpired()
  .body(schema.object({
    token: schema.string()
  }).closed())
  .audit("org.invite.accepted")
  .handle(function (ctx) {
    return ctx.capability
      .redeem()
      .createMembership()
      .commit();
  });
```

---

# Go-side enforcement model

Internally, I would compile every route into something like:

```go
type RoutePlan struct {
    Name       string
    Method     string
    Path       string

    Auth       AuthRequirement
    Tenant     TenantRequirement
    Resource   ResourceRequirement
    Action     string
    BodySchema SchemaRef
    RateLimit  *RateLimitSpec
    Audit      AuditSpec

    Handler    goja.Callable
}
```

Then every request follows the same pipeline:

```go
func ServeRoute(plan RoutePlan, w http.ResponseWriter, r *http.Request) {
    session := authenticate(plan.Auth, r)
    actor := loadActor(session)
    tenant := resolveTenant(plan.Tenant, r, actor)
    resource := resolveResource(plan.Resource, r, tenant)

    authorize(actor, plan.Action, resource, tenant)

    body := validateBody(plan.BodySchema, r)
    ctx := buildScriptContext(actor, tenant, resource, body)

    result := callGojaHandler(plan.Handler, ctx)

    writeResponse(result)
}
```

The JS handler should never be responsible for remembering the universal security sequence.

---

# Builder rules that make misuse hard

Enforce these at registration time:

```text
Every route must be public, authenticated, system-authenticated, or capability-authenticated.

Every non-public mutating route must have an action.

Every action must be tied to a resource, unless explicitly marked global.

Every tenant app route must resolve tenant context.

Every body-consuming route must declare a schema.

Every sensitive route must declare an audit event.

Every capability must have a purpose, expiry, and subject or resource.

Every admin route must require MFA freshness.

Every dangerous operation must require .because(...).
```

Reject route registration if the plan is incomplete.

---

# Best default design

I would combine four ideas:

```text
1. Staged route builder
2. Resource-first authorization
3. Operation builders instead of DB access
4. Capability builder for links/tokens/invites
```

So the mental model for users is:

```text
declare who can call this
declare what resource it touches
declare what action is allowed
declare input shape
write handler using safe operation builders
```

The security model stays in Go. The JS API stays pleasant. The wrong thing becomes either impossible, unavailable, or noisy.

[1]: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html?utm_source=chatgpt.com "Authorization - OWASP Cheat Sheet Series"
[2]: https://github.com/dop251/goja?utm_source=chatgpt.com "dop251/goja: ECMAScript/JavaScript engine in pure Go"
[3]: https://openfga.dev/docs/concepts?utm_source=chatgpt.com "Concepts"
[4]: https://openfga.dev/docs/authorization-concepts?utm_source=chatgpt.com "Fine-Grained Authorization, ReBAC, ABAC & Zanzibar ..."


--- 

A nice MVP should be **small, opinionated, and hard to bypass**.

Do not start with Casbin/OpenFGA/OPA, a full policy language, or generic capability algebra. Start with:

```text
Keycloak login
server-side app sessions
route builder
resource loading
simple Go authorization checks
safe user/org operations
audit logging
```

The MVP should prove one thing: **a JS route author cannot forget authentication, tenant scoping, input validation, or authorization.**

---

# MVP shape

## 1. Route builder with mandatory security mode

Every route must choose exactly one:

```js
.public()
.auth(...)
.capability(...)
.system(...)
```

Example:

```js
api.get("/me")
  .auth(user.session().required())
  .allow("user.self.read")
  .handle(function (ctx) {
    return ctx.me().profile();
  });
```

This should fail:

```js
api.get("/me")
  .handle(function (ctx) {
    return ctx.me().profile();
  });
```

Error:

```text
Route GET /me has no security mode.
Use .public(), .auth(), .capability(), or .system().
```

That one rule catches a large class of mistakes.

---

# 2. Minimal route API

I would expose only this at first:

```js
api.get(path)
api.post(path)
api.patch(path)
api.delete(path)
```

With these builder methods:

```js
.name(name)
.public()
.auth(authSpec)
.resource(resourceSpec)
.allow(action)
.body(schema)
.audit(eventName)
.handle(fn)
```

MVP example:

```js
api.patch("/users/:userId")
  .name("user.update")
  .auth(user.session().required())
  .resource(userResource().fromParam("userId").withinTenant())
  .allow("user.update")
  .body(schema.object({
    displayName: schema.string().min(1).max(80)
  }).closed())
  .audit("user.updated")
  .handle(function (ctx) {
    return ctx.users
      .byResource(ctx.resource("user"))
      .updateProfile()
      .displayName(ctx.body.displayName)
      .commit();
  });
```

The route author never receives a raw `userId` and then manually updates it. They receive a loaded, authorized resource.

---

# 3. Minimal resource model

Support maybe three resource types at first:

```text
user
org
project
```

Resource builders:

```js
userResource().fromParam("userId")
orgResource().fromParam("orgId")
projectResource().fromParam("projectId")
```

With modifiers:

```js
.mustExist()
.withinTenant()
.ownedByActor()
```

Examples:

```js
.resource(projectResource()
  .fromParam("projectId")
  .withinTenant()
  .mustExist())
.allow("project.update")
```

Go should resolve the resource before the JS handler runs.

---

# 4. Minimal authorization model

Start with hardcoded Go checks.

No policy engine yet.

Something like:

```go
type Action string

const (
    UserSelfRead    Action = "user.self.read"
    UserUpdate      Action = "user.update"
    OrgInvite       Action = "org.invite"
    ProjectRead     Action = "project.read"
    ProjectUpdate   Action = "project.update"
)

func Can(ctx context.Context, actor Actor, action Action, resource Resource) (bool, error) {
    switch action {
    case UserSelfRead:
        return actor.Authenticated, nil

    case UserUpdate:
        return actor.IsTenantAdmin(resource.TenantID), nil

    case OrgInvite:
        return actor.HasTenantRole(resource.ID, "admin"), nil

    case ProjectRead:
        return actor.MemberOfTenant(resource.TenantID), nil

    case ProjectUpdate:
        return actor.HasTenantRole(resource.TenantID, "admin", "editor"), nil

    default:
        return false, nil
    }
}
```

This is boring. Good. Boring authz is easier to test.

---

# 5. Minimal session model

Use Keycloak for login, then issue your own app session.

Browser sees:

```text
__Host-app-session=<opaque random session id>
```

JS handlers see:

```js
ctx.actor
ctx.session
ctx.me()
```

They should not see:

```js
ctx.rawAccessToken
ctx.rawRefreshToken
ctx.sessionId
```

MVP session context:

```js
ctx.actor.id
ctx.actor.email
ctx.actor.emailVerified
ctx.actor.tenantIds
```

Maybe:

```js
ctx.auth.hasMfa
ctx.auth.mfaAge
```

But do not overdo it.

---

# 6. Minimal safe operation builders

Expose semantic operations, not raw DB.

For current user:

```js
ctx.me().profile()
ctx.me().email()
ctx.me().sessions()
```

For users:

```js
ctx.users.byResource(resource)
ctx.users.invite()
```

For orgs:

```js
ctx.orgs.byResource(resource)
ctx.org.members()
```

Example:

```js
ctx.me()
  .profile()
  .setDisplayName("Manuel")
  .commit();
```

Example invite:

```js
ctx.users.invite()
  .toOrg(ctx.resource("org"))
  .email(ctx.body.email)
  .role(ctx.body.role)
  .expiresIn("7d")
  .send();
```

Do not expose this in MVP:

```js
ctx.db
ctx.sql
ctx.users.setRole(userId, role)
ctx.users.delete(userId)
ctx.users.update(id, patch)
```

---

# 7. Minimal schema system

You do not need full Zod.

Start with:

```js
schema.object({...}).closed()
schema.string()
schema.email()
schema.enum(...)
schema.boolean()
schema.number()
schema.array(...)
```

Example:

```js
.body(schema.object({
  email: schema.email(),
  role: schema.enum("viewer", "editor")
}).closed())
```

Make `.closed()` the default if possible. Unknown fields should be rejected unless explicitly allowed.

---

# 8. Minimal audit

Only require audit for mutating authenticated routes.

```js
.audit("org.member.invited")
```

Go records:

```text
event_name
actor_id
tenant_id
resource_type
resource_id
route_name
request_id
timestamp
ip_hash maybe
user_agent maybe
outcome
```

Do not let JS write arbitrary security audit records in MVP. Let JS declare the event name; Go emits it.

---

# 9. MVP route examples

## Public route

```js
api.get("/health")
  .public()
  .handle(function () {
    return { ok: true };
  });
```

## Current user

```js
api.get("/me")
  .name("me.read")
  .auth(user.session().required())
  .allow("user.self.read")
  .handle(function (ctx) {
    return ctx.me().profile();
  });
```

## Update own profile

```js
api.patch("/me/profile")
  .name("me.profile.update")
  .auth(user.session().required())
  .allow("user.self.update")
  .body(schema.object({
    displayName: schema.string().min(1).max(80)
  }).closed())
  .audit("user.profile.updated")
  .handle(function (ctx) {
    return ctx.me()
      .profile()
      .setDisplayName(ctx.body.displayName)
      .commit();
  });
```

## Invite org member

```js
api.post("/orgs/:orgId/invites")
  .name("org.invite")
  .auth(user.session().required())
  .resource(orgResource().fromParam("orgId").mustExist())
  .allow("org.member.invite")
  .body(schema.object({
    email: schema.email(),
    role: schema.enum("viewer", "editor")
  }).closed())
  .audit("org.member.invited")
  .handle(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .expiresIn("7d")
      .send();
  });
```

## Update project

```js
api.patch("/projects/:projectId")
  .name("project.update")
  .auth(user.session().required())
  .resource(projectResource()
    .fromParam("projectId")
    .withinTenant()
    .mustExist())
  .allow("project.update")
  .body(schema.object({
    name: schema.string().min(1).max(120)
  }).closed())
  .audit("project.updated")
  .handle(function (ctx) {
    return ctx.projects
      .byResource(ctx.resource("project"))
      .rename(ctx.body.name)
      .commit();
  });
```

---

# 10. MVP registration validation

At startup, compile all JS routes into Go `RoutePlan`s.

Reject invalid plans.

Rules:

```text
Every route must have a name.
Every route must have exactly one security mode.
Every authenticated mutating route must have .allow(...).
Every route with a request body must have .body(...).
Every mutating authenticated route must have .audit(...).
Every .allow(...) action must be known to Go.
Every .resource(...) type must be known to Go.
Every route path param referenced by a resource must exist.
No duplicate route names.
No duplicate method/path pairs.
```

This gives immediate feedback before the server starts.

---

# 11. Go internal model

Something like:

```go
type RoutePlan struct {
    Name      string
    Method    string
    Path      string

    Security  SecurityMode
    Resource  *ResourceSpec
    Action    *Action
    Body      *SchemaSpec
    Audit     *AuditSpec

    Handler   goja.Callable
}
```

Request pipeline:

```go
func Serve(plan RoutePlan, w http.ResponseWriter, r *http.Request) {
    actor := authenticate(plan.Security, r)

    params := parsePathParams(plan.Path, r)
    body := validateBody(plan.Body, r)

    resource := resolveResource(plan.Resource, params, actor)

    authorize(actor, plan.Action, resource)

    ctx := buildJSContext(actor, resource, body, params)

    result := runHandler(plan.Handler, ctx)

    writeResult(w, result)
}
```

The JS handler is the last step, not the place where core security is assembled.

---

# 12. What I would deliberately skip in MVP

Skip:

```text
custom IdP
custom password auth
generic policy language
OpenFGA
OPA
Casbin
dynamic permission editor UI
nested group inheritance
field-level permissions
user impersonation
OAuth client management
long-lived API tokens
multi-region session replication
complex capability system
```

Maybe include one simple capability type:

```text
org invite token
```

But not a generic capability API yet.

---

# 13. The smallest useful MVP

The smallest version I would build:

```text
1. Keycloak login
2. Server-side sessions
3. JS route registration through goja
4. Route builder with .public() / .auth()
5. Schema validation
6. Resource resolver for user/org/project
7. Go-owned Can(actor, action, resource)
8. Safe ctx.me(), ctx.users, ctx.orgs, ctx.projects APIs
9. Audit for mutating routes
10. Startup validation that rejects unsafe routes
```

That is enough to validate the whole architecture.

The MVP should feel like this to users:

```js
api.post("/orgs/:orgId/invites")
  .auth(user.session().required())
  .resource(orgResource().fromParam("orgId"))
  .allow("org.member.invite")
  .body(schema.inviteMember())
  .audit("org.member.invited")
  .handle(function (ctx) {
    return ctx.users.invite()
      .toOrg(ctx.resource("org"))
      .email(ctx.body.email)
      .role(ctx.body.role)
      .send();
  });
```

And internally it should enforce this:

```text
No auth declaration, no route.
No resource, no resource action.
No permission, no handler execution.
No schema, no body.
No audit, no sensitive mutation.
```

That is a strong MVP.

