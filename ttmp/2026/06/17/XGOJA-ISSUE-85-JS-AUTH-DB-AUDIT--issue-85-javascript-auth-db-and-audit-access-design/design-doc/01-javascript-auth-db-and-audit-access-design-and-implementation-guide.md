---
Title: JavaScript Auth DB and Audit Access Design and Implementation Guide
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
      Note: Target JavaScript app for JS-owned audit route
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
    - Path: modules/database/database.go
      Note: Existing JavaScript SQL module to reuse for phase 2 raw handles
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Audit record model and memory store
    - Path: pkg/gojahttp/auth/audit/sqlstore/sqlstore.go
      Note: SQL audit store and query implementation starting point
    - Path: pkg/xgoja/hostauth/builder.go
    - Path: pkg/xgoja/hostauth/services.go
      Note: Defines hostauth.Services and ServicesKey exposed to runtime modules
    - Path: pkg/xgoja/hostauth/stores.go
      Note: Builds hostauth stores and is the future SQL handle registry seam
    - Path: pkg/xgoja/providers/host/host.go
      Note: Current guarded host database module configuration
    - Path: pkg/xgoja/providers/http/serve.go
      Note: Injects hostauth services into serve runtimes
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/85
    - https://github.com/go-go-golems/go-go-goja/issues/86
Summary: Design and implementation guide for exposing safe JavaScript auth/audit access in generated xgoja OIDC hosts.
LastUpdated: 2026-06-17T16:25:00-04:00
WhatFor: 'Use this when implementing GitHub issue #85 or onboarding an engineer to xgoja hostauth, host services, database modules, and JS-owned audit routes.'
WhenToUse: Before changing hostauth stores, exposing DB handles to JavaScript, moving /auth/audit out of native handlers, or adding high-level auth/audit JS APIs.
---


# JavaScript Auth DB and Audit Access Design and Implementation Guide

## Executive summary

GitHub issue [#85](https://github.com/go-go-golems/go-go-goja/issues/85) asks for a safe way for generated `xgoja` hosts to let JavaScript application code inspect host-owned auth data, especially audit logs. The immediate product motivation is simple: the demo currently has a native `GET /auth/audit` endpoint, but the app should own audit presentation and authorization policy in JavaScript. The broader platform need is larger: generated hosts should be able to expose selected host resources to JS without leaking raw secrets, session internals, or unrestricted SQL.

The safest first implementation is **not** to expose raw database handles as the primary API. The first implementation should add a high-level JavaScript module, tentatively `require("auth")`, with a narrow `auth.audit.query(...)` method. That method should call Go-owned audit query interfaces, enforce maximum result limits, normalize filters, and return typed JSON-safe audit records. This gives example 21 enough power to move `/auth/audit` into JS and then lets issue [#86](https://github.com/go-go-golems/go-go-goja/issues/86) remove demo endpoints from generic OIDC native handlers.

Raw named database handles should be phase 2. They are valuable, but they are also the sharp edge. If implemented, they should use an explicit `hostHandle` configuration, read-only wrappers by default, query timeouts, row limits, and table/statement guardrails. No DSN should be visible to JavaScript, and no host/auth DB should be exposed unless the application opts in through `xgoja.yaml`.

This guide explains the current system, the exact files involved, the proposed APIs, the implementation order, the security model, and a test plan. It is written for a new intern who needs to implement the feature without first knowing how `xgoja`, `hostauth`, host services, generated command providers, or the existing database module fit together.

## The problem in one diagram

Today the generated OIDC host owns the auth stores and native auth handlers. JavaScript owns application routes, but it cannot directly query audit records.

```text
Browser
  |
  v
xgoja serve top-level mux
  |
  +-- native OIDC handlers from hostauth
  |     /auth/login
  |     /auth/callback
  |     /auth/logout
  |     /auth/session
  |     /auth/audit          <-- demo/native today, too broad
  |     /orgs/o1/invites    <-- demo/native today, hard-coded
  |     /org-invites/accept <-- demo/native today, hard-coded
  |
  +-- JavaScript app host fallback
        /
        /healthz
        /me
        /orgs/:orgId/projects/:projectId
```

The desired end state after #85 and #86 is:

```text
Browser
  |
  v
xgoja serve top-level mux
  |
  +-- native OIDC/session lifecycle only
  |     /auth/login
  |     /auth/callback
  |     /auth/logout
  |     /auth/session
  |
  +-- JavaScript app host fallback
        /
        /healthz
        /me
        /auth/audit or /orgs/:orgId/audit
          -> .auth(...)
          -> .allow("audit.read")
          -> require("auth").audit.query(...)
```

The key idea is that JavaScript should declare the route and authorization policy, while Go should keep responsibility for safe access to host-owned auth data.

## Problem statement and scope

### What this ticket should solve

This ticket should make it possible for JavaScript application code in a generated xgoja host to implement an authorized audit-log route. The first target API should be narrow and high-level:

```js
const auth = require("auth");

app.get("/orgs/:orgId/audit")
  .auth(express.user().required())
  .allow("audit.read")
  .handle((ctx, res) => {
    const records = auth.audit.query({
      tenantId: ctx.request.params.orgId,
      outcome: ctx.request.query.outcome || undefined,
      limit: 50,
    });
    res.json({ records });
  });
```

The implementation should work for the current hostauth stores:

- memory audit store,
- SQL audit store backed by SQLite,
- SQL audit store backed by Postgres.

The implementation should not require JavaScript to know database DSNs or raw table schemas for the common audit case.

### What this ticket should not solve first

This ticket should not begin by exposing unrestricted raw SQL access to all auth databases. That would be powerful but unsafe. The session, appauth, capability, and audit stores contain sensitive data. A narrow high-level audit API provides enough capability to unblock #86 while leaving raw host DB handles for a later phase.

This ticket also should not solve full admin UI design, audit retention, RBAC modeling, cross-tenant query semantics, or general SQL sandboxing. Those are related but larger concerns.

## Current-state architecture

### Generated xgoja hosts and runtime modules

A generated xgoja binary is built from `xgoja.yaml`. Example 21 selects providers, runtime modules, sources, commands, artifacts, and auth config in `examples/xgoja/21-generated-host-auth/xgoja.yaml`.

Evidence:

- `examples/xgoja/21-generated-host-auth/xgoja.yaml:1-10` defines the generated app name, env prefix, Go module, and workspace mode.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml:11-20` registers the core, host, and HTTP providers.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml:21-42` wires runtime modules including `timer`, `fs:assets`, and `express`.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml:43-61` configures top-level OIDC auth defaults.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml:62-90` embeds JS verb sources and dashboard assets into the generated binary.

The JavaScript app routes live in `examples/xgoja/21-generated-host-auth/verbs/sites.js`. That file currently demonstrates planned auth route declarations such as `.auth(...)`, `.allow(...)`, `.csrf()`, `.resource(...)`, and `.audit(...)` on JS routes.

Evidence:

- `examples/xgoja/21-generated-host-auth/verbs/sites.js:5-15` creates the Express-style app, loads embedded assets, and serves `/`.
- `examples/xgoja/21-generated-host-auth/verbs/sites.js:18-34` declares public health and async demo routes.
- `examples/xgoja/21-generated-host-auth/verbs/sites.js:37-42` declares protected `/me` with `.auth(express.user().required())`, `.allow("user.self.read")`, and `.audit("user.self.read")`.
- `examples/xgoja/21-generated-host-auth/verbs/sites.js:45-61` declares a protected project update route with resource lookup, CSRF, authorization, and audit annotation.

### `xgoja serve` and host services

The HTTP provider's `serve` command owns normal server lifecycle for generated hosts. It builds auth services, creates or reuses a `gojahttp.Host`, creates runtime services, starts a JavaScript runtime, invokes the selected JS verb, and then starts the Go HTTP server unless an external host owns the listener.

Evidence:

- `pkg/xgoja/providers/http/serve.go:145-151` requires a runtime factory capable of per-runtime host services.
- `pkg/xgoja/providers/http/serve.go:152-164` creates a generated host or reuses an injected external host.
- `pkg/xgoja/providers/http/serve.go:165-176` creates runtime host services and starts a runtime with `NewRuntimeFromSectionsWithHostServices`.
- `pkg/xgoja/providers/http/serve.go:178-182` initializes selected modules.
- `pkg/xgoja/providers/http/serve.go:184-186` invokes the JS verb so route declarations register on the host.
- `pkg/xgoja/providers/http/serve.go:188-191` supports external no-listen mode.
- `pkg/xgoja/providers/http/serve.go:193-205` builds the top-level handler, binds `net.Listen`, and serves HTTP when xgoja owns the listener.

The serve command passes auth services into the runtime using host services. This is the seam #85 should reuse.

Evidence:

- `pkg/xgoja/providers/http/serve.go:475-486` builds `app.HostServices`, optionally sets `go-go-goja-http.host`, and sets `hostauth.ServicesKey` when auth services exist.

### Host services are the cross-language dependency injection mechanism

Provider modules receive host services through `providerapi.ModuleSetupContext`. This lets Go-owned services be made available to modules without putting them into global variables.

Evidence:

- `pkg/xgoja/providerapi/module.go:19-25` defines `ModuleSetupContext` with `Host providerapi.HostServices`.
- `pkg/xgoja/providerapi/module.go:29-32` defines `HostServices` as the interface exposing `AssetResolver()`.
- `pkg/xgoja/providerapi/module.go:36-39` defines `HostServiceLookup`, which allows lookup of arbitrary keyed host services.
- `pkg/xgoja/providerapi/capabilities.go:91-98` defines `HostServiceContributionCapability`, which lets provider packages contribute host services before runtime module setup.

At the app layer, `HostServices` stores assets plus arbitrary services keyed by string.

Evidence:

- `pkg/xgoja/app/assets.go:21-24` defines `HostServices` with `Assets` and `Services` fields.
- `pkg/xgoja/app/assets.go:64-78` defines `SetHostService`.
- `pkg/xgoja/app/assets.go:80-92` defines `AddHostService`.
- `pkg/xgoja/app/assets.go:94-108` defines single and multi-value host service lookup.
- `pkg/xgoja/app/host_services.go:102-119` defines layered host services, where per-runtime services can overlay base services.
- `pkg/xgoja/app/host_services.go:149-190` defines contributed host services that preserve a base service lookup.

For #85, the important point is: **we already have a typed injection path from Go services into JavaScript modules**. We should use it rather than inventing a parallel runtime global.

### `hostauth.Services` is already available to runtime modules

The hostauth package already defines a runtime service object that groups auth configuration, middleware options, stores, services, and native handlers.

Evidence:

- `pkg/xgoja/hostauth/services.go:5` defines `ServicesKey` as `go-go-goja-hostauth.services`.
- `pkg/xgoja/hostauth/services.go:46-61` defines `Services` with config, auth options, session manager, stores, native handlers, and closers.
- `pkg/xgoja/hostauth/services.go:63-73` defines `Close`.
- `pkg/xgoja/providers/http/serve.go:475-486` injects `hostauth.Services` into runtime host services.

That means a new JS module can discover `hostauth.Services` with `HostServiceLookup`, then expose safe JavaScript functions backed by the existing Go stores.

### Hostauth stores and SQL DB ownership

`hostauth.BuildStores` builds the concrete session, audit, appauth, and capability stores from resolved config. It already shares SQL DB handles when store configs resolve to the same driver and DSN.

Evidence:

- `pkg/xgoja/hostauth/stores.go:23-30` defines `StoreBundle` with logical stores and closers.
- `pkg/xgoja/hostauth/stores.go:41-48` constructs a `storeBuilder` with `dbs: map[sqlDBKey]*sql.DB{}`.
- `pkg/xgoja/hostauth/stores.go:54-79` builds session, audit, appauth, and capability stores and returns the bundle.
- `pkg/xgoja/hostauth/stores.go:82-105` builds memory or SQL session stores.
- `pkg/xgoja/hostauth/stores.go:108-129` builds memory or SQL audit stores.
- `pkg/xgoja/hostauth/stores.go:131-153` builds memory or SQL appauth stores.
- `pkg/xgoja/hostauth/stores.go:156-177` builds memory or SQL capability stores.
- `pkg/xgoja/hostauth/stores.go:179-194` opens and reuses SQL DB handles by driver and DSN.

The store builder is a good future place to register named raw DB handles. However, for phase 1, we can avoid raw DB handles by using the existing logical audit store.

### Existing audit store capabilities

The audit package defines normalized records and a write interface.

Evidence:

- `pkg/gojahttp/auth/audit/audit.go:23-42` defines `Record` fields such as event, outcome, route name, action, actor, tenant, resource, request metadata, attributes, and created time.
- `pkg/gojahttp/auth/audit/audit.go:45-47` defines `Store` with `InsertAuditRecord`.
- `pkg/gojahttp/auth/audit/audit.go:49-58` defines `Sink.Record`, which writes records if a store exists.
- `pkg/gojahttp/auth/audit/audit.go:72-92` defines `MemoryStore` and its insert behavior.
- `pkg/gojahttp/auth/audit/audit.go:94-99` defines `MemoryStore.Snapshot()`.

The SQL audit store already has read methods, but they are not represented in the base `audit.Store` interface.

Evidence:

- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go:24-31` defines SQL audit store config and fields.
- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go:33-47` validates dialect and constructs the store.
- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go:62-88` inserts normalized audit records.
- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go:104-122` implements `Snapshot(ctx)`.
- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go:124-151` implements `QueryByOutcome(ctx, outcome, limit)`.

For #85, introduce a first-class query interface rather than relying on ad hoc `Snapshot()` type assertions.

### Existing database module

The repository already has a generic `modules/database` module. It can be configured with a DSN or preconfigured with a Go-provided DB-like interface.

Evidence:

- `modules/database/database.go:17-25` defines `QueryExecer` and `QueryExecerContext` interfaces.
- `modules/database/database.go:29-38` defines transaction interfaces.
- `modules/database/database.go:63-70` defines `WithPreconfiguredDB`.
- `modules/database/database.go:80-86` defines `WithConfigureEnabled`.
- `modules/database/database.go:99-113` constructs the module with default name `database` and configurable behavior.
- `modules/database/database.go:221-237` exposes JS functions `configure`, `query`, `exec`, `begin`, and `close`.
- `modules/database/database.go:279-323` implements `QueryContext` with context propagation and row-to-record conversion.
- `modules/database/database.go:326-361` implements `ExecContext`.
- `modules/database/database.go:364-380` starts transactions.

The host provider already exposes a guarded `database` module that can open a preconfigured DB from static module config.

Evidence:

- `pkg/xgoja/providers/host/host.go:51-55` defines `DatabaseConfig` with `allowConfigure`, `driverName`, and `dataSourceName`.
- `pkg/xgoja/providers/host/host.go:59-68` registers `database` and `db` modules in the host provider.
- `pkg/xgoja/providers/host/host.go:221-233` decodes database module config and creates the module.
- `pkg/xgoja/providers/host/host.go:235-250` either returns a configurable DB module or opens a preconfigured DB from `driverName` and `dataSourceName`.

This existing database module should be reused for phase 2 raw DB handles, but phase 1 should not depend on raw SQL.

## Gap analysis

### Gap 1: JavaScript cannot ask hostauth for audit records safely

`hostauth.Services` contains `AuditStore`, but no JS-facing module uses it today. Native Go code can type-assert the store and query records, as the current demo native `/auth/audit` handler does, but JavaScript routes cannot call that behavior.

### Gap 2: The audit read interface is informal

`audit.Store` only supports insert. `MemoryStore` has `Snapshot()`, and SQL store has `Snapshot(ctx)` plus `QueryByOutcome(ctx, outcome, limit)`. The generic native handler currently type-asserts to those shapes. That works for a demo, but it is not a stable API contract for a generated host platform.

### Gap 3: Raw DB access exists but is DSN-oriented

The host provider can create a database module from `driverName` and `dataSourceName`, but issue #85 is about host-owned auth DBs. Those DBs are already configured under `auth.stores.*` and should not require a second DSN in the runtime module config. A future raw DB path needs a host-handle registry.

### Gap 4: Native demo endpoints block JS ownership

Issue #86 exists because generic OIDC services currently include demo-specific native endpoints. The safe migration path is to implement #85 first, then move audit browsing into JS and remove the generic native audit endpoint in #86.

## Proposed architecture

### Phase 1: High-level `auth` JavaScript module

Create a new provider module, likely under `pkg/xgoja/providers/hostauth` or as part of the existing hostauth package/provider surface, that exposes a JavaScript module named `auth`.

Initial JS API:

```ts
interface AuthAuditQuery {
  tenantId?: string;
  outcome?: string;
  actorId?: string;
  resourceType?: string;
  resourceId?: string;
  limit?: number;
  offset?: number;
}

interface AuthAuditRecord {
  event: string;
  outcome: string;
  reason?: string;
  statusCode?: number;
  routeName?: string;
  method?: string;
  pattern?: string;
  action?: string;
  actorId?: string;
  actorKind?: string;
  tenantId?: string;
  resourceType?: string;
  resourceId?: string;
  requestId?: string;
  ipHash?: string;
  userAgent?: string;
  attributes?: Record<string, unknown>;
  createdAt: string;
}

interface AuthAuditAPI {
  query(query?: AuthAuditQuery): AuthAuditRecord[];
}

interface AuthModule {
  audit: AuthAuditAPI;
}
```

Usage in JavaScript:

```js
const auth = require("auth");

app.get("/orgs/:orgId/audit")
  .auth(express.user().required())
  .resource(express.resource("org").idFromParam("orgId").mustExist())
  .allow("audit.read")
  .audit("audit.records.read")
  .handle((ctx, res) => {
    const records = auth.audit.query({
      tenantId: ctx.request.params.orgId,
      outcome: ctx.request.query.outcome || undefined,
      limit: Number(ctx.request.query.limit || 50),
    });
    res.json({ records });
  });
```

The module should discover `hostauth.Services` from host services:

```go
lookup, ok := ctx.Host.(providerapi.HostServiceLookup)
if !ok { return error }
raw, ok := lookup.HostService(hostauth.ServicesKey)
if !ok { return error }
services, ok := raw.(*hostauth.Services)
if !ok { return error }
```

Then it should expose `services.AuditStore` through a typed, guarded query facade.

### Phase 1 Go interfaces

Add a query contract to the audit package:

```go
type Query struct {
    TenantID     string
    Outcome      string
    ActorID      string
    ResourceType string
    ResourceID   string
    Limit        int
    Offset       int
}

type QueryStore interface {
    QueryAuditRecords(ctx context.Context, query Query) ([]Record, error)
}
```

Recommended defaults:

- default limit: 50,
- max limit: 100,
- negative limits rejected or normalized to default,
- negative offsets rejected or normalized to zero,
- empty query allowed only if max limit applies.

Implement `QueryAuditRecords` for:

- `audit.MemoryStore`, by filtering an in-memory snapshot,
- `audit/sqlstore.Store`, by building a small parameterized SQL query.

For the SQL store, do not expose free-form where clauses. Build the query from known fields only.

Pseudocode:

```go
func (s *Store) QueryAuditRecords(ctx context.Context, q audit.Query) ([]audit.Record, error) {
    q = audit.NormalizeQuery(q)

    builder := newSQLBuilder(s.dialect)
    builder.Select(auditColumns...).From("audit_records")

    if q.TenantID != "" {
        builder.Where("tenant_id = ?", q.TenantID)
    }
    if q.Outcome != "" {
        builder.Where("outcome = ?", q.Outcome)
    }
    if q.ActorID != "" {
        builder.Where("actor_id = ?", q.ActorID)
    }
    if q.ResourceType != "" {
        builder.Where("resource_type = ?", q.ResourceType)
    }
    if q.ResourceID != "" {
        builder.Where("resource_id = ?", q.ResourceID)
    }

    builder.OrderBy("created_at desc")
    builder.Limit(q.Limit)
    builder.Offset(q.Offset)

    rows, err := s.db.QueryContext(ctx, builder.SQL(), builder.Args()...)
    ... scan rows ...
}
```

### Phase 1 runtime wiring

A new provider module should be selected in `xgoja.yaml` only when an app wants the module. Example 21 can add it when #85 is implemented.

Possible YAML:

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
          max-limit: 100
```

Alternative: register the module from the existing HTTP or host provider. This is less clean because audit/auth is not an HTTP or host filesystem concern. A small dedicated provider makes ownership obvious.

### Phase 2: Guarded host DB handles

Phase 2 can add raw DB handles using a dedicated registry. This is useful for internal admin tools or app-specific queries that are not worth adding to the high-level `auth` module.

The registry should live in a small package such as `pkg/xgoja/hostdb`:

```go
package hostdb

const ServicesKey = "go-go-goja-hostdb.registry"

type SQLHandle struct {
    Name     string
    DB       *sql.DB
    Driver   string
    Logical  string
    ReadOnly bool
}

type Registry struct {
    mu      sync.RWMutex
    handles map[string]SQLHandle
}

func (r *Registry) Register(handle SQLHandle) error
func (r *Registry) Handle(name string) (SQLHandle, bool)
func (r *Registry) List() []SQLHandle
```

`hostauth.StoreBundle` can own one registry and register logical handles as stores are built:

```go
func (b *storeBuilder) buildAuditStore(ctx context.Context, cfg ResolvedStoreConfig) (audit.Store, error) {
    switch cfg.Driver {
    case StoreDriverSQLite, StoreDriverPostgres:
        db, err := b.openDB(cfg)
        ...
        b.sql.Register(hostdb.SQLHandle{Name: "auth.audit", DB: db, Driver: string(cfg.Driver), Logical: "audit"})
    }
}
```

Then the host database provider can support:

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: database
      as: auditdb
      config:
        hostHandle: auth.audit
        access: read-only
        maxRows: 100
        timeout: 2s
        allowedTables: [audit_records]
```

The existing `modules/database` module should be reused with a guarded wrapper that implements `databasemod.QueryExecerContext`.

Pseudocode:

```go
type guardedDB struct {
    db      *sql.DB
    policy  Policy
}

func (g *guardedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
    if err := g.policy.AllowQuery(query); err != nil {
        return nil, err
    }
    ctx, cancel := context.WithTimeout(ctx, g.policy.Timeout)
    defer cancel()
    return g.db.QueryContext(ctx, g.policy.RewriteLimit(query), args...)
}

func (g *guardedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
    if !g.policy.AllowWrites {
        return nil, fmt.Errorf("database handle is read-only")
    }
    ...
}
```

Phase 2 is deliberately separate from the first #85 implementation. It should not block moving `/auth/audit` into JS.

## API reference

### Proposed JavaScript API: `require("auth")`

#### `auth.audit.query(query?)`

Returns audit records matching a bounded filter.

Input:

```js
{
  tenantId?: string,
  outcome?: string,
  actorId?: string,
  resourceType?: string,
  resourceId?: string,
  limit?: number,
  offset?: number,
}
```

Output:

```js
[
  {
    event: "project.updated",
    outcome: "allowed",
    routeName: "...",
    action: "project.update",
    actorId: "user:...",
    actorKind: "user",
    tenantId: "o1",
    resourceType: "project",
    resourceId: "p1",
    requestId: "...",
    attributes: {},
    createdAt: "2026-06-17T...Z"
  }
]
```

Errors:

- no hostauth services available,
- no audit store configured,
- audit store does not implement query interface,
- invalid query shape,
- store query failure.

#### `auth.audit.maxLimit()` or config-only max

This is optional. If exposed, it should only return configured max limit. The simpler implementation can keep max limit as an internal module config.

### Proposed Go API: `audit.QueryStore`

```go
type Query struct {
    TenantID     string
    Outcome      string
    ActorID      string
    ResourceType string
    ResourceID   string
    Limit        int
    Offset       int
}

type QueryStore interface {
    QueryAuditRecords(ctx context.Context, query Query) ([]Record, error)
}
```

### Proposed provider API: `pkg/xgoja/providers/hostauth`

```go
func Register(registry *providerapi.ProviderRegistry) error
```

Registers module:

```go
providerapi.Module{
    Name: "auth",
    DefaultAs: "auth",
    Description: "Safe JavaScript access to generated host auth services",
    NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) { ... },
}
```

Module config sketch:

```go
type AuthModuleConfig struct {
    Audit AuditModuleConfig `json:"audit"`
}

type AuditModuleConfig struct {
    MaxLimit int `json:"maxLimit,omitempty"`
}
```

## Detailed implementation plan

### Phase 0: Add tests that describe the desired behavior

Start with tests. They will force the design into the runtime seams that already exist.

Suggested tests:

1. `pkg/gojahttp/auth/audit`: memory query filtering.
2. `pkg/gojahttp/auth/audit/sqlstore`: SQL query filtering by tenant/outcome/limit.
3. `pkg/xgoja/providers/hostauth`: module setup fails clearly without `hostauth.Services`.
4. `pkg/xgoja/providers/hostauth`: module exposes `auth.audit.query` when services are present.
5. `pkg/xgoja/providers/http`: generated serve runtime can require the module and return audit records from a JS route.
6. `examples/xgoja/21-generated-host-auth`: smoke proves `/auth/audit` is JS-owned once #86 moves the route.

### Phase 1: Add `audit.Query` and `audit.QueryStore`

File: `pkg/gojahttp/auth/audit/audit.go`.

Add:

```go
type Query struct { ... }
type QueryStore interface { QueryAuditRecords(context.Context, Query) ([]Record, error) }
func NormalizeQuery(Query, maxLimit int) (Query, error)
```

Memory implementation can live in the same file.

Pseudocode:

```go
func (s *MemoryStore) QueryAuditRecords(_ context.Context, query Query) ([]Record, error) {
    s.mu.Lock()
    records := append([]Record(nil), s.records...)
    s.mu.Unlock()

    query = NormalizeQuery(query)
    out := []Record{}
    for i := len(records)-1; i >= 0; i-- {
        if matches(records[i], query) {
            out = append(out, records[i])
            if len(out) == query.Limit { break }
        }
    }
    reverse(out) // if API wants ascending; or document descending
    return out, nil
}
```

Recommendation: return newest first for audit browser convenience and document it.

### Phase 2: Implement SQL audit query

File: `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go`.

Add `QueryAuditRecords(ctx, query)` and small dialect-aware placeholder helper. The store already has dialect information and scan logic. Reuse existing `scanRecord`.

Do not accept arbitrary SQL fragments. Every filter should be a known field.

### Phase 3: Add the `auth` provider module

New package suggestion:

```text
pkg/xgoja/providers/hostauth/
  hostauth.go
  auth_module.go
  auth_module_test.go
```

The module should:

- decode config,
- find `hostauth.Services` from `ctx.Host`,
- require an audit store for `auth.audit.query`,
- expose JS functions through `modules.SetExport`, consistent with existing native modules.

Loader pseudocode:

```go
func (m *AuthModule) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    auditObj := vm.NewObject()
    modules.SetExport(auditObj, "auth.audit", "query", func(raw map[string]any) ([]map[string]any, error) {
        query, err := decodeAuditQuery(raw)
        if err != nil { return nil, err }
        records, err := m.audit.Query(runtimebridge.CurrentOwnerContext(vm), query)
        if err != nil { return nil, err }
        return recordsToJS(records), nil
    })
    _ = exports.Set("audit", auditObj)
}
```

Use `runtimebridge.CurrentOwnerContext(vm)` so request/runtime cancellation propagates.

### Phase 4: Register the provider in example 21

Update `examples/xgoja/21-generated-host-auth/xgoja.yaml`:

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
          maxLimit: 100
```

Update `examples/xgoja/21-generated-host-auth/verbs/sites.js`:

```js
const auth = require("auth");

app.get("/auth/audit")
  .auth(express.user().required())
  .allow("audit.read")
  .audit("audit.records.read")
  .handle((_ctx, res) => {
    res.json({ records: auth.audit.query({ limit: 50 }) });
  });
```

For production safety, prefer a tenant-scoped path:

```js
app.get("/orgs/:orgId/audit") ...
```

But to preserve the current dashboard UX during transition, `/auth/audit` can temporarily exist as JS-owned demo route. The important part is that it is not a generic native handler.

### Phase 5: Prepare #86 removal

After #85 lands, issue #86 can remove native demo handlers from `pkg/xgoja/hostauth/builder.go`:

- `GET /auth/audit`,
- `POST /orgs/o1/invites`,
- `POST /org-invites/accept`.

Audit can be ported to JS using the new `auth.audit.query` API. Invite endpoints may still need a high-level capability API or can be removed from the demo until such an API exists.

## Security model

### Principle 1: JavaScript owns app policy; Go owns host data safety

JS should decide who can call a route:

```js
.auth(express.user().required()).allow("audit.read")
```

Go should decide how audit data can be queried safely:

- bounded result set,
- known filters only,
- context timeout,
- no arbitrary SQL in phase 1.

### Principle 2: no hidden DB exposure

Adding the `auth` module should be explicit in `xgoja.yaml`. A generated host with `auth.mode=oidc` should not automatically expose raw stores to JS.

### Principle 3: no raw session/capability tables in phase 1

Audit records are already intended to be read operationally. Session and capability stores are much more sensitive. Do not expose them through this phase.

### Principle 4: raw SQL only behind explicit host handles

When phase 2 adds raw DB handles, require configuration such as:

```yaml
hostHandle: auth.audit
access: read-only
```

No automatic `require("sessiondb")`, no automatic `require("capabilitydb")`, and no DSNs in JavaScript.

## Decision records

### Decision: implement high-level audit API before raw DB handles

- **Context:** The user wants JavaScript to control audit access. Raw DB handles are powerful but risky because auth DBs contain sensitive data.
- **Options considered:** Expose raw DBs first; expose a high-level `auth.audit.query` first; keep native `/auth/audit` until a full DB registry exists.
- **Decision:** Implement high-level `auth.audit.query` first, defer raw DB handles to phase 2.
- **Rationale:** This directly enables JS-owned audit routes and #86 cleanup while avoiding a broad SQL footgun.
- **Consequences:** Some custom queries will need more Go API surface later, but the first implementation is safer and easier to review.
- **Status:** proposed.

### Decision: use host services for module wiring

- **Context:** Generated runtimes already use host services to inject auth and HTTP host state into provider modules.
- **Options considered:** Global variables, singleton registries, context values, or existing host services.
- **Decision:** Use `providerapi.HostServiceLookup` and `hostauth.ServicesKey`.
- **Rationale:** This matches current `serve` architecture and keeps generated runtime instances isolated.
- **Consequences:** The new module must fail clearly when hostauth services are absent.
- **Status:** proposed.

### Decision: add a formal audit query interface

- **Context:** SQL and memory audit stores have read helpers, but `audit.Store` only defines inserts.
- **Options considered:** Type-assert current `Snapshot` methods; add query methods only to SQL store; add a formal `audit.QueryStore` interface.
- **Decision:** Add `audit.QueryStore` with bounded filters.
- **Rationale:** A formal interface gives the JS module a stable contract and avoids ad hoc type assertions.
- **Consequences:** Existing stores need small implementations; future stores can opt in explicitly.
- **Status:** proposed.

### Decision: keep raw DB handles as phase 2

- **Context:** Issue #85 includes guarded host/auth DB handles as a broader goal.
- **Options considered:** Implement registry and raw DB handles immediately; split into phases.
- **Decision:** Split raw DB handles into phase 2 after `auth.audit.query`.
- **Rationale:** The high-level audit API is sufficient to unblock route ownership and is much safer.
- **Consequences:** The ticket can land incrementally; phase 2 still needs careful policy design.
- **Status:** proposed.

## Test strategy

### Unit tests: audit query

Memory store:

- inserts records with multiple tenants/outcomes,
- queries by tenant,
- queries by outcome,
- applies default limit,
- clamps max limit,
- returns newest first or documented order.

SQL store:

- applies schema,
- inserts records,
- queries by tenant/outcome/resource,
- verifies placeholders work for SQLite and Postgres dialect builders,
- validates limit behavior.

### Unit tests: `auth` module

- missing host services returns clear error,
- wrong `hostauth.ServicesKey` type returns clear error,
- missing audit store returns clear error,
- audit store without query support returns clear error,
- `auth.audit.query({ limit: 1000 })` clamps or rejects according to spec,
- result objects serialize expected fields.

### Integration tests: generated serve runtime

Build a runtime with:

- HTTP provider,
- hostauth services,
- new hostauth/auth provider module,
- JS route that calls `require("auth").audit.query`.

Then verify:

- unauthorized request still gets 401 before handler,
- authorized route can return audit records,
- route-level `.allow("audit.read")` still controls access,
- native `/auth/audit` is not required for the app route to work.

### Example 21 smoke

After #85 and during #86:

- update dashboard to call JS-owned audit route,
- update smoke to verify the route is app-owned where practical,
- stop relying on generic native `/auth/audit`.

## Risks and mitigations

### Risk: audit records leak cross-tenant data

Mitigation: Make the JS route tenant-scoped and recommend `tenantId` filters. The Go API should support tenant filters. For a generic `/auth/audit` demo route, document that it is a demo and require an explicit permission.

### Risk: raw DB handles become the default escape hatch

Mitigation: Do not implement raw handles first. When implemented, require explicit `hostHandle` config and read-only guards.

### Risk: API grows into a full admin SDK too early

Mitigation: Start with `auth.audit.query` only. Add appauth/capability helpers only when concrete JS routes need them.

### Risk: route auth and module access are confused

Mitigation: Document that `require("auth")` is a data access primitive, not an authorization check. JS routes must still use `.auth(...)`, `.resource(...)`, and `.allow(...)`.

## Intern implementation checklist

1. Read `pkg/xgoja/providers/http/serve.go` to understand when runtimes get host services.
2. Read `pkg/xgoja/hostauth/services.go` to understand what hostauth services contain.
3. Read `pkg/gojahttp/auth/audit/audit.go` and `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go` to understand audit records and existing query helpers.
4. Add `audit.Query` and `audit.QueryStore`.
5. Implement query support for memory and SQL audit stores.
6. Add a new provider module exposing `require("auth")`.
7. Use `runtimebridge.CurrentOwnerContext(vm)` in JS-callable functions.
8. Add unit tests before wiring example 21.
9. Update example 21 only after the module is tested.
10. Run focused tests, then full tests.

## File reference map

| Area | File | Why it matters |
|---|---|---|
| Generated OIDC example config | `examples/xgoja/21-generated-host-auth/xgoja.yaml` | Shows provider/module/source/artifact/auth config for the target demo. |
| Generated OIDC JS routes | `examples/xgoja/21-generated-host-auth/verbs/sites.js` | Destination for JS-owned audit route. |
| Serve runtime wiring | `pkg/xgoja/providers/http/serve.go` | Builds auth services and injects host services into each runtime. |
| Hostauth service bundle | `pkg/xgoja/hostauth/services.go` | Existing object containing audit store and session/auth services. |
| Store construction | `pkg/xgoja/hostauth/stores.go` | Builds memory/SQL stores and reuses SQL DB handles. |
| Native handlers | `pkg/xgoja/hostauth/builder.go` | Current location of generic native `/auth/audit`; #86 will remove it. |
| Host services | `pkg/xgoja/app/assets.go`, `pkg/xgoja/app/host_services.go` | Dependency injection mechanism for runtime modules. |
| Provider API | `pkg/xgoja/providerapi/module.go`, `pkg/xgoja/providerapi/capabilities.go` | Contracts used by modules and host service contributors. |
| Existing DB module | `modules/database/database.go` | Reusable raw SQL module for phase 2. |
| Host provider DB config | `pkg/xgoja/providers/host/host.go` | Existing DSN-oriented DB module config; future `hostHandle` extension point. |
| Audit model | `pkg/gojahttp/auth/audit/audit.go` | Audit record and write store interface. |
| SQL audit store | `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go` | SQL persistence and existing snapshot/query methods. |

## Recommended implementation sequence for #85 and #86

The clean order is:

1. Implement #85 phase 1: `auth.audit.query`.
2. Update example 21 to use JS-owned audit route.
3. Implement #86: remove generic native demo `/auth/audit` from hostauth.
4. Later implement #85 phase 2: guarded raw `hostHandle` DB modules.

This order avoids breaking the demo while still moving toward the secure architecture.

## References

- GitHub issue #85: <https://github.com/go-go-golems/go-go-goja/issues/85>
- GitHub issue #86: <https://github.com/go-go-golems/go-go-goja/issues/86>
- `pkg/xgoja/providers/http/serve.go`
- `pkg/xgoja/hostauth/services.go`
- `pkg/xgoja/hostauth/stores.go`
- `pkg/gojahttp/auth/audit/audit.go`
- `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go`
- `modules/database/database.go`
- `pkg/xgoja/providers/host/host.go`
- `examples/xgoja/21-generated-host-auth/xgoja.yaml`
- `examples/xgoja/21-generated-host-auth/verbs/sites.js`
