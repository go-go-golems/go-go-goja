---
Title: Investigation diary
Ticket: XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT
Status: active
Topics:
    - xgoja
    - auth
    - audit
    - database
    - javascript
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/21-generated-host-auth/verbs/sites.js
      Note: |-
        Example route calls auth audit query from JavaScript commit b7f85cc
        Updated example audit route to fluent auth audit query builder (commit eedfdb7)
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
      Note: Registers hostauth provider and auth module commit b7f85cc
    - Path: pkg/gojahttp/auth/appauth/appauth.go
      Note: Added audit.read appauth action for JS-owned audit route commit b7f85cc
    - Path: pkg/gojahttp/auth/audit/audit.go
      Note: Added audit.Query
    - Path: pkg/gojahttp/auth/audit/sqlstore/sqlstore.go
      Note: Added bounded SQL audit query implementation (commit a0a2eeb)
    - Path: pkg/xgoja/hostauth/builder.go
      Note: Removed generic native demo handlers; hostauth now owns lifecycle/session routes only (commit e094279)
    - Path: pkg/xgoja/hostauth/builder_test.go
      Note: Updated native handler test to assert demo endpoints are absent (commit e094279)
    - Path: pkg/xgoja/providers/hostauth/hostauth.go
      Note: Refactored auth.audit.query from object-bag decoder to fluent builder (commit eedfdb7)
    - Path: pkg/xgoja/providers/hostauth/hostauth_test.go
      Note: |-
        Runtime tests for auth audit query module commit 53156f5
        Updated runtime test to use fluent audit query builder (commit eedfdb7)
    - Path: ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/01-javascript-auth-db-and-audit-access-design-and-implementation-guide.md
      Note: Primary deliverable described by diary
    - Path: ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md
      Note: |-
        Step 6 cleanup design deliverable
        Step 8 fluent-builder v2 design and reMarkable upload
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/85
Summary: Chronological record for the Issue 85 JavaScript auth DB and audit access design ticket.
LastUpdated: 2026-06-17T16:25:00-04:00
WhatFor: Use this to understand how the Issue 85 design document was prepared and validated.
WhenToUse: Before resuming or implementing the JavaScript auth DB/audit access work.
---








# Diary

## Goal

This diary captures the creation of a docmgr ticket and intern-oriented design guide for GitHub issue #85: exposing safe JavaScript access to host/auth database-backed audit data in generated xgoja hosts.

## Step 1: Create the Issue 85 design ticket and evidence-backed guide

I created a dedicated docmgr ticket for issue #85 and wrote a long-form design and implementation guide. The guide maps the current generated OIDC host architecture, explains how host services and auth stores flow into runtime modules, and proposes a phased implementation that starts with a safe high-level `require("auth").audit.query(...)` API before adding guarded raw DB handles.

The design intentionally targets a new intern as the reader. It includes architecture diagrams, API sketches, pseudocode, decision records, file references, and implementation/test phases so the implementation can proceed without rediscovering the surrounding xgoja runtime model.

### Prompt Context

**User prompt (verbatim):** "ok, create a new ticket for #85, then Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a docmgr ticket for GitHub issue #85, write a comprehensive intern-friendly architecture/design/implementation guide, store it in that ticket workspace, validate the ticket, and upload the resulting document bundle to reMarkable.

**Inferred user intent:** Turn the GitHub issue into an actionable engineering design package that a less experienced contributor can use to implement the feature safely.

**Commit (code):** N/A — documentation ticket only at this step.

### What I did
- Created docmgr ticket `XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT`.
- Created design doc `design-doc/01-javascript-auth-db-and-audit-access-design-and-implementation-guide.md`.
- Created diary doc `reference/01-investigation-diary.md`.
- Read the ticket-research writing style and deliverable checklist references.
- Read the reMarkable upload skill and followed its minimal upload workflow guidance.
- Gathered evidence from GitHub issue #85 and current source files, including:
  - `pkg/xgoja/hostauth/stores.go`
  - `pkg/xgoja/hostauth/services.go`
  - `pkg/xgoja/hostauth/builder.go`
  - `pkg/xgoja/providers/http/serve.go`
  - `pkg/xgoja/providers/host/host.go`
  - `modules/database/database.go`
  - `pkg/gojahttp/auth/audit/audit.go`
  - `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go`
  - `examples/xgoja/21-generated-host-auth/xgoja.yaml`
  - `examples/xgoja/21-generated-host-auth/verbs/sites.js`
- Wrote the guide with current-state analysis, proposed architecture, decision records, implementation phases, testing strategy, risks, and file references.

### Why
- GitHub issue #85 is broad and security-sensitive. A rushed implementation could expose raw auth/session/capability database data to JavaScript.
- The design guide makes the safer path explicit: implement a high-level audit API first, then defer raw DB handles to a guarded second phase.
- This sequencing also supports issue #86, because JS-owned audit routes can replace native demo audit endpoints after #85 lands.

### What worked
- `docmgr ticket create-ticket` created the new ticket workspace cleanly.
- `docmgr doc add` created the design and diary docs with appropriate topics and doc types.
- The source code already has strong seams for this feature:
  - host services can carry Go-owned services into runtime modules,
  - `hostauth.Services` already includes the audit store,
  - the SQL audit store already has read helpers,
  - the existing database module can be reused later for guarded raw DB handles.

### What didn't work
- N/A. No command failures occurred during ticket setup or writing.

### What I learned
- The safest implementation path is narrower than the full issue title: start with `auth.audit.query` rather than raw DB handles.
- The existing host service mechanism is the correct dependency injection path for the new JS module.
- Raw DB handles are still useful, but they should be phase 2 with explicit `hostHandle`, read-only guards, row limits, and timeouts.

### What was tricky to build
- The main design challenge is separating route-level authorization from data-access safety. JavaScript should own `.auth(...)`, `.resource(...)`, and `.allow(...)`, but Go should still constrain what audit data can be queried and how much data can be returned.
- Another tricky point is sequencing #85 and #86. Doing #85 first is cleaner because it gives example 21 a JS-owned audit implementation before #86 removes the generic native audit route.

### What warrants a second pair of eyes
- Review the decision to start with a high-level `require("auth")` module rather than raw DB handles.
- Review whether `auth.audit.query` should allow unscoped queries in the demo or require tenant filters by default.
- Review where the provider should live: a new `pkg/xgoja/providers/hostauth` package versus extending an existing provider.

### What should be done in the future
- Implement the phase 1 `auth.audit.query` API.
- Use it to port `/auth/audit` into example 21 JavaScript.
- Then implement issue #86 to remove demo native endpoints from generic OIDC services.
- Consider phase 2 guarded raw host DB handles only after the high-level API is working.

### Code review instructions
- Start with the design doc:
  - `ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/01-javascript-auth-db-and-audit-access-design-and-implementation-guide.md`
- Verify the key current-state references against:
  - `pkg/xgoja/providers/http/serve.go`
  - `pkg/xgoja/hostauth/services.go`
  - `pkg/xgoja/hostauth/stores.go`
  - `pkg/gojahttp/auth/audit/audit.go`
  - `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go`
  - `modules/database/database.go`
- Validate ticket hygiene with:

```bash
docmgr doctor --ticket XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT --stale-after 30
```

### Technical details
- Proposed phase 1 JS API:

```js
const auth = require("auth");
const records = auth.audit.query({ tenantId: "o1", outcome: "allowed", limit: 50 });
```

- Proposed Go query interface:

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

- Proposed implementation sequence:
  1. Add formal audit query interface.
  2. Implement memory and SQL audit queries.
  3. Add JS `auth` module backed by `hostauth.Services`.
  4. Update example 21 to use JS-owned audit route.
  5. Use #86 to remove generic native demo endpoints.


## Step 2: Validate and upload the Issue 85 design bundle

I validated the new docmgr ticket and uploaded the ticket bundle to reMarkable. The bundle includes the ticket index, primary design guide, investigation diary, and task checklist so the implementation context is available as one PDF.

The first doctor run found missing vocabulary entries for the new `audit` and `database` topics. I added both topic vocabulary entries and reran doctor successfully.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete ticket hygiene and deliver the Issue 85 design package to reMarkable.

**Inferred user intent:** Make the design package reviewable outside the repository and ensure docmgr metadata is healthy.

**Commit (code):** N/A — documentation ticket only at this step.

### What I did
- Added docmgr vocabulary topics `audit` and `database`.
- Ran `docmgr doctor --ticket XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT --stale-after 30` successfully.
- Uploaded a reMarkable bundle named `XGOJA Issue 85 JavaScript Auth DB Audit Design.pdf`.
- Marked validation and upload tasks complete.

### Why
- The ticket should pass docmgr hygiene checks before handoff.
- The user explicitly asked to upload the design package to reMarkable.

### What worked
- Upload succeeded:

```text
OK: uploaded XGOJA Issue 85 JavaScript Auth DB Audit Design.pdf -> /ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT
```

### What didn't work
- Initial `docmgr doctor` reported unknown topics:

```text
unknown topics: [audit database]
```

I resolved this with:

```bash
docmgr vocab add --category topics --slug audit --description "Audit logging, audit records, and audit access patterns"
docmgr vocab add --category topics --slug database --description "Database access, SQL handles, stores, and persistence design"
```

### What I learned
- The repository vocabulary did not yet have generic `audit` or `database` topic slugs, even though both are appropriate for this ticket.

### What was tricky to build
- The upload workflow is intentionally minimal; the successful upload output is the verification signal, so no separate cloud listing was required.

### What warrants a second pair of eyes
- Review whether adding broad `audit` and `database` topic vocabulary entries is preferable to using existing narrower topics only.

### What should be done in the future
- N/A.

### Code review instructions
- Review `ttmp/vocabulary.yaml` for the new topic entries.
- Review the ticket bundle contents under `ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design`.

### Technical details
- reMarkable destination: `/ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT`.
- Uploaded document: `XGOJA Issue 85 JavaScript Auth DB Audit Design.pdf`.


## Step 3: Implement bounded audit query stores

I implemented the first code slice for issue #85: a formal, safe audit query contract at the store layer. This gives JavaScript-facing code a stable Go API to call later, without exposing raw SQL or auth database internals.

The implementation added `audit.Query`, `audit.QueryStore`, bounded limit normalization, memory-store filtering, and SQL-store filtering. The SQL implementation builds parameterized predicates from known fields only and preserves newest-first ordering for audit browser use cases.

### Prompt Context

**User prompt (verbatim):** "implement step by step, commit at appropriate intervals, keep a detailed diary"

**Assistant interpretation:** Start implementing the Issue 85 design incrementally, commit each coherent slice, and maintain a detailed chronological diary.

**Inferred user intent:** Convert the design guide into production code safely while preserving reviewable commits and continuation-friendly documentation.

**Commit (code):** a0a2eeb — "auth: add bounded audit query stores"

### What I did
- Added `audit.Query` and `audit.QueryStore` in `pkg/gojahttp/auth/audit/audit.go`.
- Added `audit.NormalizeQuery` with default/max limits and whitespace normalization.
- Implemented `MemoryStore.QueryAuditRecords` with field filtering, offset, bounded limit, and newest-first ordering.
- Implemented `sqlstore.Store.QueryAuditRecords` with parameterized SQL built from known audit fields.
- Rewired `QueryByOutcome` to call the new general query API.
- Added memory and SQL tests covering filtering, ordering, and bounds.
- Ran focused validation:

```bash
go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore -count=1
```

- The pre-commit hook also ran full lint and test successfully during `git commit`.

### Why
- The high-level JavaScript `auth.audit.query(...)` module needs a stable Go query interface that works for both memory and SQL auth stores.
- Keeping queries field-based avoids exposing raw SQL predicates or auth database internals to JavaScript.

### What worked
- Focused audit tests passed:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit	0.006s
ok  	github.com/go-go-golems/go-goja/pkg/gojahttp/auth/audit/sqlstore	0.037s
```

- The repository pre-commit hook passed lint and full tests before creating commit `a0a2eeb`.

### What didn't work
- N/A. This slice did not hit compile or test failures.

### What I learned
- The existing SQL audit store already had enough scan and persistence infrastructure to support a general query method with minimal change.
- Reusing the existing `QueryByOutcome` behavior through the new query API keeps old operational helpers intact while creating a broader contract.

### What was tricky to build
- The SQL builder needed to preserve parameterization across SQLite (`?`) and Postgres (`$1`, `$2`, ...), while avoiding any user-provided SQL fragments.
- Memory-store ordering needed to be newest-first without mutating stored insertion order or leaking mutable `Attributes` maps.

### What warrants a second pair of eyes
- Confirm that clamping excessive limits is preferable to returning validation errors.
- Confirm newest-first ordering is the desired stable API contract for all audit query stores.
- Review whether `Offset` is needed for v1 or should remain internal until pagination UI exists.

### What should be done in the future
- Add the JavaScript-facing `require("auth")` provider module that consumes `audit.QueryStore` through `hostauth.Services`.
- Add integration tests showing JS can query audit records through host services.

### Code review instructions
- Start with `pkg/gojahttp/auth/audit/audit.go`, especially `Query`, `NormalizeQuery`, and `MemoryStore.QueryAuditRecords`.
- Then review `pkg/gojahttp/auth/audit/sqlstore/sqlstore.go`, especially `QueryAuditRecords` and `queryAuditRecordsQuery`.
- Validate with:

```bash
go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore -count=1
```

### Technical details
- `audit.QueryStore` contract:

```go
type QueryStore interface {
    QueryAuditRecords(ctx context.Context, query Query) ([]Record, error)
}
```

- Supported filters in this slice:
  - `tenantId`
  - `outcome`
  - `actorId`
  - `resourceType`
  - `resourceId`
  - `limit`
  - `offset`


## Step 4: Expose high-level `require("auth").audit.query(...)`

I implemented the JavaScript-facing auth module for issue #85. The new provider package `pkg/xgoja/providers/hostauth` registers a guarded `auth` module that looks up `hostauth.Services` from runtime host services and exposes a single high-level API: `auth.audit.query(...)`.

This slice intentionally does not expose raw database handles. JavaScript can query audit records through a bounded, field-based API, while Go keeps ownership of auth stores, query normalization, and result shaping.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue the staged implementation by adding the JS module layer after the store query contract.

**Inferred user intent:** Make the audit query capability usable from generated xgoja JavaScript without opening unrestricted auth DB access.

**Commit (code):** 53156f5 — "xgoja: expose high-level auth audit module"

### What I did
- Added package `pkg/xgoja/providers/hostauth`.
- Registered provider package ID `go-go-goja-hostauth` with module `auth`.
- Added module config for `audit.maxLimit` / `audit.max-limit`.
- Looked up `hostauth.Services` via `hostauth.LookupServices(ctx.Host)`.
- Required `Services.AuditStore` to implement `audit.QueryStore`.
- Exposed `auth.audit.query(query)` to JavaScript.
- Returned lower-camel-case JavaScript record objects instead of raw Go struct field names.
- Added tests for:
  - provider registration,
  - missing hostauth service error,
  - real runtime `require("auth").audit.query(...)` with max-limit clamping.
- Made audit query stores tolerate nil contexts defensively after the JS module test surfaced that edge.

### Why
- This is the core #85 high-level API needed before moving the demo `/auth/audit` route into JavaScript.
- It lets application routes own authorization policy while avoiding raw SQL or direct table access.

### What worked
- Focused validation passed:

```bash
go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore ./pkg/xgoja/providers/hostauth -count=1
```

- The pre-commit hook passed full lint and full `go test ./...` before commit `53156f5`.

### What didn't work
- First provider compile failed because I initially followed the design-doc shorthand and tried to read `services.Stores.AuditStore`. The actual `hostauth.Services` shape exposes `AuditStore` directly:

```text
pkg/xgoja/providers/hostauth/hostauth.go:64:16: services.Stores undefined (type *"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth".Services has no field or method Stores)
pkg/xgoja/providers/hostauth/hostauth_test.go:63:87: unknown field Stores in struct literal of type "github.com/go-go-golems/go-goja/pkg/xgoja/hostauth".Services
```

- The first JS runtime test returned the wrong tenant because `goja.ExportTo` did not map JS `tenantId` to Go `TenantID` through JSON tags:

```text
state missing "event":"new denied": {"count":1,"event":"other tenant","tenantId":"o2","resourceId":"p3"}
```

I fixed this by explicitly decoding JS object fields (`tenantId`, `outcome`, `actorId`, `resourceType`, `resourceId`, `limit`, `offset`).

- The next runtime test panicked on nil optional fields / nil context before the final fix:

```text
runtimeowner hostauth.test: runtime call panicked: runtime error: invalid memory address or nil pointer dereference
```

I fixed this by making optional JS value decoding nil-safe and by making `QueryAuditRecords` tolerate nil contexts.

### What I learned
- The design direction was right, but the exact service shape is `Services.AuditStore`, not a nested store bundle.
- JSON tags are not a reliable way to decode JS objects into Go structs through goja; explicit property decoding is safer for stable JavaScript APIs.
- Store-level query methods should defensively handle nil contexts because module/runtime bridges can be exercised from tests or hosts that do not install owner call contexts exactly as expected.

### What was tricky to build
- The sharpest edge was JavaScript-to-Go field decoding. The symptom was a query that ignored `tenantId`, which could become a serious data isolation bug. The fix was to decode every supported public JS field by exact property name instead of relying on Go reflection.
- Another tricky point was result shaping. Returning raw `audit.Record` values risks Go field names leaking into JS. The module now builds lower-camel-case maps deliberately.

### What warrants a second pair of eyes
- Review the explicit JS query decoder for all supported fields and nil/undefined/null behavior.
- Review the decision to clamp max limits rather than throw when JS asks for too many records.
- Review whether the provider package should eventually carry TypeScript declarations for `require("auth")`.

### What should be done in the future
- Wire example 21 to register the new provider and use `require("auth")` from its JS route file.
- Add an HTTP/serve-level integration test if the example smoke does not fully cover generated host wiring.
- Implement issue #86 after the JS-owned audit route exists.

### Code review instructions
- Start at `pkg/xgoja/providers/hostauth/hostauth.go`.
- Pay special attention to:
  - `LookupServices` usage,
  - `queryFromValue`,
  - `recordsForJS`,
  - `effectiveMaxLimit`.
- Then review `pkg/xgoja/providers/hostauth/hostauth_test.go` for runtime-level proof that `require("auth")` works.
- Validate with:

```bash
go test ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore ./pkg/xgoja/providers/hostauth -count=1
```

### Technical details
- JavaScript API implemented in this step:

```js
const auth = require("auth");
const records = auth.audit.query({ tenantId: "o1", outcome: "denied", limit: 50 });
```

- The module returns audit records as plain JS objects with keys such as `event`, `outcome`, `tenantId`, `actorId`, `resourceType`, `resourceId`, and `createdAt`.


## Step 5: Wire example 21 to the JS-owned audit query route

I wired the generated OIDC example to use the new high-level JavaScript audit API. The example now registers the `go-go-goja-hostauth` provider, requires `auth` in `verbs/sites.js`, and exposes an application-owned `/orgs/:orgId/audit` route that calls `auth.audit.query(...)`.

I also added an explicit `audit.read` appauth action and made it require an admin role on an organization resource. This keeps the example route from becoming another "any authenticated user can read audit records" path, while still leaving raw native demo endpoint removal to issue #86.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue staged implementation by wiring the new JS audit module into the generated OIDC demo and validating it through the example build/smoke path.

**Inferred user intent:** Prove the #85 API is usable in the real generated-host example before removing the generic native demo endpoint in #86.

**Commit (code):** b7f85cc — "examples: use JS auth audit query route"

### What I did
- Added `ActionAuditRead = "audit.read"` to `pkg/gojahttp/auth/appauth/appauth.go`.
- Updated `appauth.Authorizer` so `audit.read` requires an organization resource and admin role.
- Updated appauth tests for allowed/denied `audit.read` cases.
- Added provider `go-go-goja-hostauth` to example 21 `xgoja.yaml`.
- Added runtime module `auth` with `audit.maxLimit: 50`.
- Updated `examples/xgoja/21-generated-host-auth/verbs/sites.js`:
  - `const auth = require("auth")`
  - new `/orgs/:orgId/audit` route
  - route requires user auth, org resource resolution, `audit.read`, and calls `auth.audit.query(...)`
- Updated dashboard `app.js` to fetch `/orgs/o1/audit` instead of `/auth/audit`.
- Updated the example smoke test to verify the dashboard references `/orgs/o1/audit` and unauthenticated requests to that JS route return `401`.
- Included generated `pkg/xgoja/providers/hostauth/logcopter.go` created by the repository's `go generate` pre-commit flow.

### Why
- The new module is only useful if generated applications can declare it in `xgoja.yaml` and call it from route code.
- Adding `audit.read` makes the JS-owned audit route use an explicit permission rather than reusing unrelated `user.self.read` or allowing every authenticated user.

### What worked
- Focused tests passed:

```bash
go test ./pkg/gojahttp/auth/appauth ./pkg/xgoja/providers/hostauth -count=1
```

- Example doctor/build passed:

```bash
make -C examples/xgoja/21-generated-host-auth doctor build
```

- Full example smoke completed successfully:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
```

- Pre-commit hook passed full lint and full `go test ./...` before commit `b7f85cc`.

### What didn't work
- N/A for this slice. The build and smoke path passed on the first run after edits.

### What I learned
- The generated example can import a new provider package cleanly through `xgoja.yaml`; `xgoja doctor` resolved the workspace provider without extra generator changes.
- The example smoke target is mostly quiet on success because the OIDC smoke recipe is `@`-prefixed; successful completion with no shell output after build is expected.

### What was tricky to build
- The route authorization needed a real action. Reusing `user.self.read` would have made the demo work but would also encode the wrong security model. Adding `audit.read` to appauth keeps the route aligned with the design and issue #86 hardening goals.
- The native `/auth/audit` route still exists and still mounts before JS. To avoid route shadowing before #86, the JS-owned route uses `/orgs/:orgId/audit` and the dashboard now calls that route.

### What warrants a second pair of eyes
- Review `audit.read` semantics: it currently requires an admin role on an org resource.
- Review whether `/orgs/:orgId/audit` should filter by `tenantId: org.id` or use a different tenant derivation for organizations that have separate resource IDs and tenant IDs.
- Review whether the old native `/auth/audit` should now be removed immediately under #86.

### What should be done in the future
- Implement issue #86: remove generic native `/auth/audit`, invite issue, and invite accept demo endpoints from `hostauth.BuildNativeHandlers`.
- Add an authenticated integration smoke if a local fake OIDC flow can cheaply establish a session and seed `audit.read` permissions.

### Code review instructions
- Review appauth policy changes in `pkg/gojahttp/auth/appauth/appauth.go` and tests.
- Review example wiring in `examples/xgoja/21-generated-host-auth/xgoja.yaml` and `verbs/sites.js`.
- Validate with:

```bash
go test ./pkg/gojahttp/auth/appauth ./pkg/xgoja/providers/hostauth -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

### Technical details
- New example route:

```js
app.get("/orgs/:orgId/audit")
  .auth(express.user().required())
  .resource(express.resource("org").idFromParam("orgId").mustExist())
  .allow("audit.read")
  .audit("audit.records.read")
  .handle((ctx, res) => {
    const org = ctx.resource("org");
    const records = auth.audit.query({ tenantId: org.id, outcome: ctx.request.query.outcome || undefined, limit: Number(ctx.request.query.limit || 50) });
    res.json({ records, count: records.length });
  });
```


## Step 6: Design reusable auth core cleanup and self-contained demo split

I added a second design document that captures the desired cleanup direction after implementing the high-level audit API. The new design separates reusable hostauth primitives from demo-specific org/project/invite behavior, and proposes a generic capability API that real applications can use without raw database access.

The design also lays out how example 21 should become a rich, self-contained showcase: JavaScript-owned app routes using `auth.audit.query(...)`, future `auth.capabilities.*`, route-level `.auth/.csrf/.resource/.allow/.audit`, and demo-local helper code for org invites and share links.

### Prompt Context

**User prompt (verbatim):** "Add detailed design doc on the cleanup of the interfaces in order to make a really nice reusable core (yet simple and opinoinated), and then a demo taking fulla dvantage of these to show case the different possibilities."

**Assistant interpretation:** Add a detailed design document explaining how to clean up current auth interfaces into a reusable opinionated core, move demo-specific behavior into demos, and design a showcase demo that exercises the resulting APIs.

**Inferred user intent:** Prevent the generic auth package from accumulating demo-specific abstractions while still offering real users useful high-level APIs and a compelling example.

**Commit (code):** N/A — documentation/design only.

### What I did
- Added design doc `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md`.
- Defined which auth pieces should remain generic core versus demo/starter code.
- Proposed generic JavaScript capability APIs:
  - `auth.capabilities.issue(...)`
  - `auth.capabilities.validate(...)`
  - `auth.capabilities.consume(...)`
  - `auth.capabilities.revoke(...)`
- Proposed moving org invite semantics out of core and into example-local JavaScript helpers.
- Proposed splitting `appauth` reusable interfaces from hard-coded starter/demo action policy.
- Added a phased implementation plan for #86 and follow-up core cleanup.
- Added test strategy, risks, decision records, and example 21 demo architecture.

### Why
- The previous implementation is enough to remove native `/auth/audit`, but invite/capability flows still risk either remaining demo-native or becoming too domain-specific in core.
- Real users need reusable primitives such as generic capability tokens, not only `issueOrgInvite` helpers.
- The demo should showcase how to compose the primitives rather than hide app-domain behavior inside generic hostauth.

### What worked
- The new design doc gives a concrete migration sequence:
  1. remove generic native demo routes,
  2. add generic capability service/API,
  3. expose `auth.capabilities.*` to JS,
  4. port invite/share flows into example JS,
  5. split or clarify `appauth` starter policy.

### What didn't work
- N/A. This was a documentation/design step.

### What I learned
- The cleanest abstraction is neither raw DB access nor domain-specific `auth.invites.*` as core. A mid-level capability API gives real users room to build app-specific safe flows.
- `appauth` should be treated carefully because it contains both broadly useful interfaces and demo/starter policy decisions.

### What was tricky to build
- The hard part was drawing the boundary between useful opinionated core and demo-specific sugar. The resulting rule is: core APIs should talk about generic actors, resources, audit records, capabilities, sessions, and policy checks; demos should talk about orgs, projects, invites, roles, and share links.

### What warrants a second pair of eyes
- Review whether `auth.capabilities.validate` should return `{ valid:false, reason }` while `consume` throws structured errors.
- Review whether `appauth` should be split into a new package or simply documented as starter policy first.
- Review the proposed module config for allowed capability types, max TTL, and max claims size.

### What should be done in the future
- Use this design to implement #86 native route cleanup.
- Add generic capability service and JavaScript APIs.
- Port invite and share-link demos into example 21 JavaScript.

### Code review instructions
- Start with `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md`.
- Cross-check the current demo-specific native handlers in `pkg/xgoja/hostauth/builder.go`.
- Cross-check current JS module design in `pkg/xgoja/providers/hostauth/hostauth.go`.

### Technical details
- Proposed generic capability JS surface:

```js
auth.capabilities.issue({ type, subject, resource, claims, ttlSeconds, createdBy });
auth.capabilities.validate({ token, expectedType });
auth.capabilities.consume({ token, expectedType });
auth.capabilities.revoke({ id, token, reason });
```

- Proposed demo-local invite helper:

```js
function issueOrgInvite(auth, ctx) {
  return auth.capabilities.issue({
    type: "org-invite",
    subject: `email:${ctx.request.body.email}`,
    resource: { type: "org", id: ctx.params.orgId, tenantId: ctx.params.orgId },
    claims: { role: ctx.request.body.role },
    ttlSeconds: 900,
    createdBy: ctx.actor.id,
  });
}
```


## Step 7: Upload reusable auth core cleanup design to reMarkable

I uploaded the new reusable auth core cleanup design document to the existing Issue 85 reMarkable folder. This makes the follow-up #86/core-cleanup design available alongside the earlier Issue 85 implementation guide.

The upload was a single direct `remarquee upload bundle` call, following the reMarkable upload workflow guidance to avoid separate status or post-upload listing calls.

### Prompt Context

**User prompt (verbatim):** "update new doc to remarkable as well"

**Assistant interpretation:** Upload the newly created reusable auth core cleanup design document to reMarkable too.

**Inferred user intent:** Keep the reMarkable review set current with the latest design document.

**Commit (code):** N/A — documentation/upload bookkeeping only.

### What I did
- Uploaded `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md` as a standalone reMarkable bundle/PDF.
- Recorded the upload result in the diary and changelog.
- Ran `docmgr doctor` for ticket hygiene.

### Why
- The new cleanup design is a key follow-up document for #86 and capability API planning, so it should be available on reMarkable for review.

### What worked
- Upload succeeded:

```text
OK: uploaded XGOJA Issue 85 Reusable Auth Core Cleanup Design.pdf -> /ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT
```

### What didn't work
- N/A.

### What I learned
- N/A.

### What was tricky to build
- N/A; this was a straightforward upload/bookkeeping step.

### What warrants a second pair of eyes
- N/A.

### What should be done in the future
- If the cleanup design changes materially after #86/capability implementation, upload a refreshed version or a full ticket bundle.

### Code review instructions
- Review the uploaded source document at `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md`.
- Validate ticket hygiene with:

```bash
docmgr doctor --ticket XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT --stale-after 30
```

### Technical details
- reMarkable destination: `/ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT`.
- Uploaded document: `XGOJA Issue 85 Reusable Auth Core Cleanup Design.pdf`.


## Step 8: Revise cleanup design around fluent Go-backed auth builders

I updated the reusable auth core cleanup design to v2. The major change is replacing object-bag JavaScript APIs with fluent Go-backed builders for both audit queries and future capability APIs.

This revision explicitly calls out that the current `auth.audit.query({ ... })` implementation should be cleaned up before expanding the auth module. The target shape is `auth.audit.query().tenantId(...).limit(...).run()`, with similar builder chains for `auth.capabilities.issue(...)`, `validate(...)`, `consume(...)`, and `revoke(...)`.

### Prompt Context

**User prompt (verbatim):** "update the design doc, and add a section to cleanup the audit query part (and others if necessary). upload as v2 to remarkable once done"

**Assistant interpretation:** Revise the reusable auth core cleanup design to specify fluent builder APIs, include an explicit audit query cleanup section, update any related capability API designs, and upload the revised document to reMarkable as version 2.

**Inferred user intent:** Avoid long-term JS object-map decoding in the auth module and make the reusable core safer, more typed, and more idiomatic through fluent Go-backed builders.

**Commit (code):** N/A — documentation/design only.

### What I did
- Updated `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md`.
- Marked the document as v2 in the summary and updated `LastUpdated`.
- Replaced the object-bag JavaScript API section with `Proposed JavaScript core API v2: fluent Go-backed builders`.
- Added explicit audit query cleanup guidance:
  - replace `auth.audit.query(object)` with `auth.audit.query().tenantId(...).run()`,
  - keep `audit.Query` / `audit.QueryStore` as the Go-side execution contract,
  - remove object decoding helpers from the current provider implementation,
  - update provider tests and example 21 route code.
- Redesigned capability APIs as fluent builders:
  - `auth.capabilities.issue(type).resource(...).ttlSeconds(...).run()`,
  - `auth.capabilities.validate(token).expectedType(...).run()`,
  - `auth.capabilities.consume(token).expectedType(...).run()`,
  - `auth.capabilities.revoke().id(...).reason(...).run()`.
- Updated demo invite/share-link snippets to use the fluent builder style.
- Added an implementation `Phase 0` for audit query cleanup before #86/generic capabilities.
- Added a decision record for fluent Go-backed builders instead of object bags.
- Uploaded the revised document to reMarkable as v2.

### Why
- Object-bag APIs force Go to defensively decode arbitrary JavaScript objects and nested maps.
- Fluent builders let Go accept typed setter arguments one field at a time, improving runtime safety and TypeScript documentation while preserving a nice JavaScript authoring experience.
- The audit query API already exists, so documenting its cleanup now prevents capability APIs from copying the less-safe object-bag shape.

### What worked
- reMarkable upload succeeded:

```text
OK: uploaded XGOJA Issue 85 Reusable Auth Core Cleanup Design v2.pdf -> /ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT
```

### What didn't work
- N/A.

### What I learned
- The fluent builder style aligns well with the existing xgoja route-builder style: `.auth(...).resource(...).allow(...).handle(...)`.
- The key cleanup is not the Go store contract; `audit.Query` and `audit.QueryStore` remain useful. The cleanup is specifically the JavaScript-facing object decoder.

### What was tricky to build
- The tricky part was keeping the design generic without making it abstract. The new shape keeps concrete typed methods (`tenantId`, `resource`, `claimString`, `ttlSeconds`) while avoiding domain-specific methods such as `issueOrgInvite` in the core.

### What warrants a second pair of eyes
- Review whether capability claims should start with only typed setters (`claimString`, `claimNumber`, `claimBool`) or also expose an explicit JSON setter later.
- Review whether `validate(...).run()` should return structured invalid results while `consume(...).run()` throws structured errors.
- Review whether the audit query builder should expose `.timeRange(...)` before or after the first cleanup implementation.

### What should be done in the future
- Refactor the current implemented `auth.audit.query(object)` to the fluent builder style before adding capability APIs.
- Then implement #86 native route cleanup and generic capability builders.

### Code review instructions
- Review `design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md`, especially:
  - `Proposed JavaScript core API v2: fluent Go-backed builders`,
  - `Audit query cleanup: auth.audit.query() builder`,
  - implementation `Phase 0`,
  - the new fluent-builder decision record.
- Validate ticket hygiene with:

```bash
docmgr doctor --ticket XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT --stale-after 30
```

### Technical details
- Target audit API:

```js
auth.audit.query()
  .tenantId(org.id)
  .outcome("denied")
  .limit(50)
  .run();
```

- Target capability issue API:

```js
auth.capabilities.issue("org-invite")
  .subject("email", email)
  .resource("org", orgId)
  .tenantId(orgId)
  .claimString("role", role)
  .ttlSeconds(900)
  .createdBy(ctx.actor.id)
  .run();
```

- reMarkable destination: `/ai/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT`.
- Uploaded document: `XGOJA Issue 85 Reusable Auth Core Cleanup Design v2.pdf`.


## Step 9: Refactor audit query API to a fluent Go-backed builder

I refactored the current `auth.audit.query` JavaScript API from an object-bag function into a fluent Go-backed builder. The public JavaScript shape is now `auth.audit.query().tenantId(...).outcome(...).limit(...).run()`, matching the v2 design direction and avoiding arbitrary JS object decoding in the provider.

The Go-side `audit.Query` and `audit.QueryStore` contracts remain unchanged. The cleanup is strictly at the JavaScript module boundary: the provider now builds an internal `audit.Query` through typed setter methods and executes it only when `.run()` is called.

### Prompt Context

**User prompt (verbatim):** "add detailed tasks to the ticket so that we can granularly track your progress, and then implementation them one by one. Commit at appropriate intervals, and keep a detaield diary as you work."

**Assistant interpretation:** Add granular docmgr tasks for the planned auth cleanup work, then implement each slice incrementally with code commits and detailed diary updates.

**Inferred user intent:** Track progress precisely while turning the fluent-builder design into implementation safely and reviewably.

**Commit (code):** eedfdb7 — "xgoja: make auth audit query fluent"

### What I did
- Added granular ticket tasks for:
  - fluent audit query cleanup,
  - provider tests,
  - example 21 update,
  - validation,
  - #86 native route cleanup,
  - future generic capability service/API work.
- Refactored `pkg/xgoja/providers/hostauth/hostauth.go`:
  - `auth.audit.query` now returns a builder object,
  - added builder methods `.tenantId`, `.outcome`, `.actorId`, `.resource`, `.resourceType`, `.resourceId`, `.limit`, `.offset`, `.run`,
  - removed object-bag decoding helpers `queryFromValue`, `optionalString`, and `optionalInt`.
- Updated `pkg/xgoja/providers/hostauth/hostauth_test.go` to use the fluent chain.
- Updated `examples/xgoja/21-generated-host-auth/verbs/sites.js` to use the fluent chain in `/orgs/:orgId/audit`.
- Ran focused validation:

```bash
go test ./pkg/xgoja/providers/hostauth ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

- Pre-commit hook also ran lint and full tests before commit `eedfdb7`.

### Why
- The object-bag API required Go to defensively decode arbitrary JavaScript maps/objects and had already exposed field-name mismatch risk during the first implementation.
- Fluent builder methods keep the API ergonomic while making the Go boundary narrower and more typed.
- This establishes the style future `auth.capabilities.*` APIs should follow.

### What worked
- Focused tests passed:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/hostauth
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit
ok  	github.com/go-go-golems/go-goja/pkg/gojahttp/auth/audit/sqlstore
```

- Example 21 smoke passed after the route update.
- Pre-commit lint and full tests passed before the code commit.

### What didn't work
- N/A. This slice passed focused validation on the first run after edits.

### What I learned
- The builder pattern fits naturally into the existing xgoja API style because route registration already uses chained builders.
- Keeping `audit.Query` as an internal Go value behind the builder gives a good separation: JS gets fluent ergonomics, Go keeps the strongly shaped query contract.

### What was tricky to build
- The main subtlety is that the builder object closes over mutable query state. This is fine for one-shot builder chains, but reviewers should confirm that returning the same object from each setter is acceptable and that users should call `auth.audit.query()` for each independent query.
- Another subtlety is that `.outcome("")` intentionally clears/omits the filter after normalization. This supports simple route code using `ctx.request.query.outcome || ""` without reintroducing object-bag `undefined` handling.

### What warrants a second pair of eyes
- Review the builder methods in `newAuditQueryBuilder` for naming and completeness.
- Confirm that `.resource(type, id)` plus `.resourceType` / `.resourceId` is the right combination.
- Confirm that `.run()` should normalize/clamp silently rather than returning explicit validation errors for large limits.

### What should be done in the future
- Implement #86 native demo route removal next.
- Use the same fluent builder pattern when adding `auth.capabilities.*`.
- Add TypeScript declarations for `AuditQueryBuilder` once provider DTS is wired.

### Code review instructions
- Start at `pkg/xgoja/providers/hostauth/hostauth.go`, especially `newAuditQueryBuilder`.
- Review runtime proof in `pkg/xgoja/providers/hostauth/hostauth_test.go`.
- Review example usage in `examples/xgoja/21-generated-host-auth/verbs/sites.js`.
- Validate with:

```bash
go test ./pkg/xgoja/providers/hostauth ./pkg/gojahttp/auth/audit ./pkg/gojahttp/auth/audit/sqlstore -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

### Technical details
- New JavaScript API:

```js
const records = auth.audit.query()
  .tenantId(org.id)
  .outcome(ctx.request.query.outcome || "")
  .limit(Number(ctx.request.query.limit || 50))
  .run();
```

- Internal Go execution still uses:

```go
normalized := audit.NormalizeQuery(query, maxLimit)
records, err := queryStore.QueryAuditRecords(runtimebridge.CurrentOwnerContext(vm), normalized)
```


## Step 10: Remove native demo endpoints from generic hostauth

I removed the generic native demo endpoints from `BuildNativeHandlers`, leaving native hostauth responsible only for OIDC and session lifecycle routes. This closes the main issue #86 architecture debt: the generic Go auth host no longer owns example-specific audit listing or org-invite semantics.

The example already owns `/orgs/:orgId/audit` through the JavaScript `auth.audit` module. Invite routes are intentionally not reintroduced yet because the generic fluent capability API does not exist; until that lands, the generic host should not expose hard-coded `o1` invite behavior.

### Prompt Context

**User prompt (verbatim):** (same as Step 9)

**Assistant interpretation:** Continue implementing the granular ticket tasks one by one, committing code and updating the diary after each coherent slice.

**Inferred user intent:** Incrementally harden the auth host by separating reusable auth lifecycle mechanics from demo-specific application semantics.

**Commit (code):** e094279 — "hostauth: remove native demo endpoints"

### What I did
- Updated `pkg/xgoja/hostauth/builder.go`:
  - removed native `GET /auth/audit`,
  - removed native `POST /orgs/o1/invites`,
  - removed native `POST /org-invites/accept`,
  - removed helper implementations for audit snapshots and demo invite issue/accept.
- Updated `pkg/xgoja/hostauth/builder_test.go`:
  - expected only lifecycle/session handlers,
  - asserted the three removed demo routes are not registered.
- Ran focused validation:

```bash
go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/hostauth -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

- Pre-commit hook also ran lint and full tests before commit `e094279`.

### Why
- Native handlers mount before the JavaScript app fallback, so generic demo endpoints can shadow application routes.
- `/auth/audit` has a safe JavaScript-owned replacement via the high-level audit module.
- Hard-coded org invite endpoints (`o1`) are demo sugar and should not live in reusable auth core.

### What worked
- Focused tests passed:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/hostauth
```

- Example 21 smoke still passed, confirming generated host build/doctor remains healthy after route removal.
- Pre-commit lint and full `go test ./...` passed before the code commit.

### What didn't work
- N/A. The removal was straightforward after the fluent audit route was already in example JavaScript.

### What I learned
- The previous generic hostauth builder had two responsibilities mixed together: reusable OIDC/session lifecycle and demo application behavior. Removing the demo routes makes the boundary obvious and easier to review.
- The remaining native handler list is now small enough to audit visually: login, callback, logout, session.

### What was tricky to build
- The tricky part was sequencing, not code complexity. Removing `/auth/audit` before JavaScript audit support would have broken the demo dashboard; doing Step 9 first made this slice safe.
- Invite endpoints require a future replacement API because the current reusable capability service methods are still org-invite-specific. I deliberately did not move them into JavaScript yet because that would either require raw DB handles or preserve demo-specific Go helpers.

### What warrants a second pair of eyes
- Confirm that no production smoke or dashboard path still depends on native `/auth/audit`.
- Confirm it is acceptable for invite smoke coverage to be deferred until generic `auth.capabilities.*` exists.
- Review whether `/auth/session` should remain native or eventually become part of a route DSL/session info API.

### What should be done in the future
- Implement generic capability service methods and fluent JS builders.
- Rebuild invite demo routes in example 21 over `auth.capabilities.*`.
- Add smoke coverage that proves removed native endpoints are either absent or application-owned.

### Code review instructions
- Start at `pkg/xgoja/hostauth/builder.go`, `BuildNativeHandlers`.
- Review `pkg/xgoja/hostauth/builder_test.go`, `TestServiceFactoryOIDCBuildsNativeHandlers`.
- Validate with:

```bash
go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/hostauth -count=1
make -C examples/xgoja/21-generated-host-auth smoke
```

### Technical details
- Remaining native OIDC routes:

```text
GET  /auth/login
GET  /auth/callback
POST /auth/logout
GET  /auth/logout
GET  /auth/session
```

- Removed generic demo routes:

```text
GET  /auth/audit
POST /orgs/o1/invites
POST /org-invites/accept
```
