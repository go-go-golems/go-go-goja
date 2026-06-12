---
Title: Investigation diary
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
RelatedFiles:
    - Path: ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md
      Note: Primary design output produced during Step 1
    - Path: ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md
      Note: Source material imported at ticket setup
ExternalSources:
    - ../sources/01-auth-preliminary-api-ideas.md
Summary: Chronological diary for the Express authentication design investigation.
LastUpdated: 2026-06-12T14:25:00-04:00
WhatFor: Use this to resume or review the ticket research and design work.
WhenToUse: Read before continuing implementation work for XGOJA-EXPRESS-AUTH.
---


# Diary

## Goal

This diary records the creation of the `XGOJA-EXPRESS-AUTH` ticket, the import of the preliminary authentication API ideas, and the first reconciliation pass against the current go-go-goja Express HTTP implementation.

## Step 1: Create ticket and write MVP auth design

I created a new docmgr ticket for adding proper authentication to the Express HTTP module and imported the user's preliminary API exploration into the ticket sources directory. I then inspected the current Express/gojahttp/xgoja HTTP provider code paths to understand where route registration, route matching, session handling, request DTO construction, response writing, and provider-owned host lifecycle currently live.

The resulting design chooses a small staged secure-route builder as the MVP. It keeps compatibility with current `app.get(pattern, handler)` routes while introducing a planned route path where JavaScript declares security intent and Go enforces authentication, resource resolution, authorization, body validation, CSRF, and audit hooks before invoking the handler.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add proper authentication to express in go-go-goja/.

Import the document /tmp/auth.md into the sources/ directory of the tickte, which is a preliminary analysis i did of potential api ideas. 

Reconcile those with our current express design, and make a new design document for an MVP auth API for our express http module.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a docmgr research/design ticket, import `/tmp/auth.md` as source material, analyze the existing go-go-goja Express HTTP module, write a detailed intern-oriented MVP auth API design, and upload the resulting bundle to reMarkable.

**Inferred user intent:** Produce a durable implementation guide that turns preliminary API sketches into an actionable, evidence-backed design for adding proper authentication and authorization to the Express HTTP module.

**Commit (code):** N/A — documentation-only ticket work in this step.

### What I did

- Ran `docmgr status --summary-only` from the workspace to confirm the docmgr root.
- Created ticket `XGOJA-EXPRESS-AUTH` with title `Add proper authentication to Express HTTP module` and topics `goja,http,security,xgoja`.
- Copied `/tmp/auth.md` to `ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md`.
- Added the design document `design/01-mvp-authentication-api-design-and-implementation-guide.md`.
- Added this diary document at `reference/01-investigation-diary.md`.
- Inspected the current implementation files:
  - `modules/express/express.go`
  - `modules/express/typescript.go`
  - `modules/express/express_integration_test.go`
  - `pkg/gojahttp/host.go`
  - `pkg/gojahttp/route_registry.go`
  - `pkg/gojahttp/request_response.go`
  - `pkg/gojahttp/session.go`
  - `pkg/xgoja/providers/http/http.go`
  - `pkg/xgoja/providers/http/serve.go`
  - `pkg/doc/18-express-module.md`
- Wrote the design around concrete file evidence and current runtime constraints.

### Why

- The preliminary source document proposed several strong API ideas, but many are broader than a first implementation in this codebase.
- The current Express module is intentionally minimal and not full Express-compatible, so a middleware-first design would not fit the current architecture.
- The central enforcement point is `gojahttp.Host.ServeHTTP`, not the JavaScript handler, because the host owns route matching, session creation, runtime calls, and response completion.

### What worked

- `docmgr ticket create-ticket` created the full ticket workspace with `design`, `reference`, `sources`, `scripts`, and bookkeeping files.
- `docmgr doc add --doc-type design` and `--doc-type reference` created correctly frontmattered documents.
- The existing implementation was small and cohesive enough to map into one request lifecycle.
- The imported `/tmp/auth.md` already contained a clear MVP recommendation section, which made reconciliation straightforward.

### What didn't work

- No command failures occurred during ticket creation, import, or initial design writing.
- I did not run Go tests because no production Go code was changed in this step.

### What I learned

- `modules/express` only exports `app()` today; all raw route methods directly call `Host.Register` with a `goja.Callable`.
- `pkg/gojahttp.Registry` stores only method, pattern, and handler today, so a real auth implementation needs to carry route metadata in or next to the route.
- The current session layer is intentionally just an opaque cookie ID and must not be treated as authentication by itself.
- The xgoja HTTP provider already supports external host injection, which is the right path for production applications to provide auth services.

### What was tricky to build

- The main design tension was preserving the existing simple Express-style API while adding security that is hard to bypass. Removing or redefining `app.get` would be disruptive, but relying on manual checks inside those handlers would fail the security goal.
- The proposed compromise is to keep legacy raw routes and add a new planned route path with staged registration. This makes adoption incremental while giving production hosts a future diagnostic or rejection point for raw routes.
- Another subtle point is hot reload: auth plans and services must be host/runtime scoped because the HTTP provider can build candidate hosts during hot reload before swapping them live.

### What warrants a second pair of eyes

- The staged builder API names and whether secure handlers should receive `(ctx, res)` or only `ctx`.
- The default behavior for missing auth services: the design recommends failing closed with 500 for authenticated planned routes, but reviewers should confirm this is the desired developer experience.
- The CSRF MVP boundary: the design leaves room for a `CSRFProtector` hook but asks whether unsafe session-authenticated routes should require CSRF from day one.

### What should be done in the future

- Implement `RoutePlan` and planned route registration in `pkg/gojahttp`.
- Add the staged `app.route(...).public/auth(...).handle(...)` builder in `modules/express`.
- Add host auth service interfaces and test fakes.
- Update `pkg/doc/18-express-module.md` after implementation.
- Add a production hardening option to reject legacy raw routes.

### Code review instructions

- Start with the design document: `ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md`.
- Compare the proposed enforcement points with:
  - `pkg/gojahttp/host.go:94-152`
  - `pkg/gojahttp/route_registry.go:10-32`
  - `modules/express/express.go:132-146`
- Validate doc hygiene with `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30`.
- Once implementation begins, validate code with `go test ./modules/express ./pkg/gojahttp ./pkg/xgoja/providers/http`.

### Technical details

The key design decision is to add a planned route alongside the current raw route:

```go
type Route struct {
    Method  string
    Pattern string
    Handler goja.Callable
    Plan    *RoutePlan // nil for legacy raw routes
}
```

Secure dispatch should happen after route matching and request DTO creation but before JavaScript handler invocation:

```text
match route
  -> build request DTO
  -> authenticate actor
  -> resolve resources
  -> authorize action/resource
  -> validate body
  -> build secure ctx
  -> call JS handler(ctx, res)
```


## Step 2: Validate and upload the documentation bundle

I validated the ticket with `docmgr doctor`, fixed the source import so it had docmgr frontmatter and a numeric prefix, and uploaded the document bundle to reMarkable. The first upload succeeded but surfaced a Mermaid rendering warning caused by parentheses in a flowchart node label, so I corrected the diagram label and re-uploaded the bundle with `--force`.

The final upload completed successfully at `/ai/2026/06/12/XGOJA-EXPRESS-AUTH` with the name `XGOJA EXPRESS AUTH MVP Design.pdf`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish ticket hygiene validation and deliver the design bundle to reMarkable.

**Inferred user intent:** Ensure the design is both stored in docmgr and available on the reMarkable device for reading/review.

**Commit (code):** N/A — documentation-only ticket work in this step.

### What I did

- Ran `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30`.
- Fixed the imported source by renaming it to `sources/01-auth-preliminary-api-ideas.md` and adding docmgr frontmatter.
- Re-ran `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30` until all checks passed.
- Ran a reMarkable dry run for the bundle.
- Ran the actual reMarkable upload.
- Fixed the Mermaid flowchart label that produced a rendering warning.
- Re-uploaded the corrected bundle with `--force`.
- Checked task 5 after upload completion.

### Why

- `docmgr doctor` requires every Markdown file in the ticket to have valid frontmatter and source files should use numeric prefixes.
- The ticket-research workflow requires validation before reMarkable delivery.
- The first upload succeeded, but the Mermaid warning indicated the generated PDF might not contain the intended diagram, so the local design and uploaded PDF needed correction.

### What worked

- After adding frontmatter and numeric prefix to the imported source, `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30` reported `✅ All checks passed`.
- The dry run showed all intended files would be included in the bundle.
- The final upload printed:
  - `OK: uploaded XGOJA EXPRESS AUTH MVP Design.pdf -> /ai/2026/06/12/XGOJA-EXPRESS-AUTH`

### What didn't work

- The first `docmgr doctor` run failed because the imported source file had no frontmatter and no numeric prefix:

```text
1) [error] YAML/frontmatter syntax error
File: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/auth-preliminary-api-ideas.md
Problem: frontmatter delimiters '---' not found

## Doctor Report (2 findings)

### XGOJA-EXPRESS-AUTH

- [ERROR] invalid_frontmatter — Failed to parse frontmatter: taxonomy: docmgr.frontmatter.parse/yaml_syntax: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/auth-preliminary-api-ideas.md frontmatter delimiters '---' not found (path=/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/auth-preliminary-api-ideas.md)
- [WARNING] missing_numeric_prefix — file without numeric prefix (path=/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/auth-preliminary-api-ideas.md)
```

- The first real upload produced a Mermaid warning even though it uploaded successfully:

```text
WARNING: failed to render Mermaid block 1: mmdc failed: Generating single mermaid chart

Error: Parse error on line 11:
...  J --> K[JS handler(req, res)]  K --> 
-----------------------^
```

### What I learned

- Imported source Markdown should be treated as a docmgr document when it lives under a ticket workspace; adding frontmatter avoids validation failures.
- Mermaid flowchart node labels with parentheses can fail in the upload rendering pipeline unless quoted or simplified.
- Re-uploading with `--force` is appropriate after correcting a generated PDF rendering problem for the same remote document name.

### What was tricky to build

- The source import was initially a direct copy of `/tmp/auth.md`, which satisfied the user's import request but did not satisfy docmgr validation. The fix was to preserve the content while wrapping it with source-document frontmatter and renaming it with a numeric prefix.
- The Mermaid warning appeared only during the real upload rendering path, not during the dry run. I traced the parser error to the `JS handler(req, res)` flowchart label and replaced it with `JS handler with req and res`.

### What warrants a second pair of eyes

- The uploaded PDF should be read on reMarkable to confirm the diagrams and long code blocks are legible.
- The design document is intentionally comprehensive; reviewers should confirm that the MVP remains small enough before implementation begins.

### What should be done in the future

- If more source files are imported into docmgr tickets, add frontmatter and numeric prefixes at import time.
- Consider adding a small pre-upload Markdown check for Mermaid parse errors if this recurs.

### Code review instructions

- Review ticket hygiene with `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30`.
- Review the upload bundle contents from the dry-run file list in the terminal transcript if needed.
- Start design review at `design/01-mvp-authentication-api-design-and-implementation-guide.md`.

### Technical details

Final upload command shape:

```bash
remarquee upload bundle \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/index.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md \
  --name "XGOJA EXPRESS AUTH MVP Design" \
  --remote-dir "/ai/2026/06/12/XGOJA-EXPRESS-AUTH" \
  --toc-depth 2 \
  --force \
  --non-interactive
```


## Step 3: Add Express-style middleware and router auth alternative

I added a second design document that approaches authentication from a more Express-compatible direction. Instead of making a new staged route builder the primary API, this design adds `app.use`, middleware stacks, `next`, error middleware, `express.Router`, router mounting, and route-local middleware arrays, then implements authentication as Go-owned middleware factories.

The key design compromise is to keep Express-style composition while preserving the strongest security insight from the earlier research: authentication, resource loading, authorization, CSRF, body validation, and audit should be native Go-owned middleware with machine-readable security metadata. That metadata lets the host validate effective route coverage so production deployments can reject unsafe routes that forgot `auth.required()` or `allow(action)`.

### Prompt Context

**User prompt (verbatim):** "ok, now do a separate document that actually approaches the problem fromt he direction of actuallly sticking more closely to the express approach by adding middleware stacks, possibly routers, and maybe taking the best out of the current research and design you did, and add that too. 

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Add a separate ticket design that explores a more Express-like middleware/router approach to auth, make it detailed and intern-friendly, store it in the existing ticket, and upload an updated reMarkable bundle.

**Inferred user intent:** Compare the previous staged route-plan design with an alternative that feels closer to real Express while still taking the best security ideas from the research.

**Commit (code):** N/A — documentation-only ticket work in this step.

### What I did

- Created `design/02-express-style-middleware-auth-design-and-implementation-guide.md` with `docmgr doc add`.
- Re-inspected the current Express and gojahttp implementation points that matter for middleware/router support:
  - `modules/express/express.go:100-187`
  - `pkg/gojahttp/host.go:94-152`
  - `pkg/gojahttp/route_registry.go:10-112`
  - `pkg/gojahttp/request_response.go:16-180`
  - `pkg/gojahttp/session.go:16-84`
  - `pkg/doc/18-express-module.md:18-125`
  - `pkg/xgoja/providers/http/http.go:124-169`
  - `pkg/xgoja/providers/http/serve.go:119-216`
- Wrote an alternative design centered on:
  - `app.use(...)`
  - `express.Router()`
  - `(req, res, next)` middleware
  - `(err, req, res, next)` error middleware
  - route-local middleware arrays
  - Go-owned security middleware factories
  - strict route coverage validation
- Updated the ticket index with a link to the new document.
- Added and later checked ticket tasks for the new design and updated reMarkable upload.
- Related the new design doc to the source files and companion design that shaped it.

### Why

- The first design is intentionally more declarative and security-first, but it is not the closest mental model for developers familiar with Express.
- Adding middleware and routers would make the module easier to use for web developers and easier to port small Express examples.
- Middleware auth can fail open if a route forgets middleware, so the design adds Go-owned security middleware metadata and host route coverage validation to recover much of the safety of the staged design.

### What worked

- The current implementation has a clear single-handler dispatch path, which made it straightforward to identify where a middleware executor would replace or wrap current behavior.
- The existing request/response DTOs already map well to Express-style `(req, res)` handlers.
- The existing xgoja external host service pattern gives production applications a place to inject auth services.

### What didn't work

- No command failures occurred while creating the second design document.
- I did not run Go tests because no production Go code was changed.

### What I learned

- The current documentation explicitly calls out the lack of middleware stacks, routers, and `next()`, so this alternative is a real shift in module scope rather than a small auth-only extension.
- Supporting middleware makes async/error semantics much more complex than the current single-handler model.
- A middleware approach needs explicit route coverage validation if it is going to satisfy the original security goal.

### What was tricky to build

- The tricky design issue was preserving existing go-go-goja return-value auto-send behavior while adding Express middleware semantics. Express normally ignores return values, but current go-go-goja handlers can return strings or values and have the host finish the response. The design resolves this by preserving auto-send only for the terminal route handler, not for non-terminal middleware.
- Another tricky issue was deciding how strict validation fits the current lifecycle. The provider currently starts the HTTP server on route use, but full coverage validation needs to see the whole mounted router stack. The design recommends `Host.ValidateRoutes()` plus a strict first-request fallback and xgoja serve validation after JS route bootstrap.

### What warrants a second pair of eyes

- Whether `next("route")` is needed in the MVP or can be deferred.
- Whether static mounts should remain before all middleware or become part of the middleware stack.
- Whether strict coverage should be available only to embedding applications first or exposed as an xgoja provider flag.
- Whether `express.secure(spec)` should return an array of middleware or one native middleware that runs a mini policy envelope.

### What should be done in the future

- Decide which design should be implemented first: staged RoutePlan, Express middleware/router stack, or a hybrid where `express.secure(spec)` compiles to Go-owned middleware.
- If middleware wins, start with router/layer data structures and keep all existing single-handler route tests green before adding auth middleware.

### Code review instructions

- Review the new design document at `design/02-express-style-middleware-auth-design-and-implementation-guide.md`.
- Compare it against `design/01-mvp-authentication-api-design-and-implementation-guide.md` before selecting an implementation direction.
- Validate ticket hygiene with `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30`.

### Technical details

The proposed middleware route protection model is:

```js
api.patch(
  "/orgs/:orgId/projects/:projectId",
  express.loadResource("project", { type: "project", id: "param:projectId", tenant: "param:orgId" }),
  express.allow("project.update", { resource: "project" }),
  express.validateBody("project.patch"),
  express.audit("project.updated"),
  handler,
)
```

The proposed Go-side layer model is:

```go
type Layer struct {
    Kind     LayerKind
    Method   string
    Pattern  string
    Handlers []HandlerSpec
    Router   *Router
    Security SecurityTags
}
```


## Step 4: Upload updated design-options bundle

I validated the ticket after adding the middleware/router design, ran a reMarkable dry run, and uploaded a new bundle that includes both design alternatives. I used a new bundle name so the earlier MVP-only PDF remains available while the updated design-options PDF contains the staged RoutePlan design, the Express-style middleware design, the diary, index, and imported source.

The final upload completed successfully at `/ai/2026/06/12/XGOJA-EXPRESS-AUTH` with the name `XGOJA EXPRESS AUTH Design Options.pdf`.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Deliver the expanded ticket package, including the new Express-style middleware/router design, to reMarkable.

**Inferred user intent:** Make the alternative design available for review on the reMarkable alongside the original staged-auth design.

**Commit (code):** N/A — documentation-only ticket work in this step.

### What I did

- Ran `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30` after adding the new design.
- Ran a dry-run bundle upload including:
  - `index.md`
  - `design/01-mvp-authentication-api-design-and-implementation-guide.md`
  - `design/02-express-style-middleware-auth-design-and-implementation-guide.md`
  - `reference/01-investigation-diary.md`
  - `sources/01-auth-preliminary-api-ideas.md`
- Uploaded the bundle to `/ai/2026/06/12/XGOJA-EXPRESS-AUTH`.
- Checked the ticket task for the updated reMarkable upload.

### Why

- The user explicitly asked to store the new document in the ticket and upload it to reMarkable.
- A new bundle name keeps the previous MVP-only upload intact and makes it clear that the new PDF contains multiple design options.

### What worked

- `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30` passed.
- The dry run listed all intended documents.
- The actual upload printed:
  - `OK: uploaded XGOJA EXPRESS AUTH Design Options.pdf -> /ai/2026/06/12/XGOJA-EXPRESS-AUTH`

### What didn't work

- No upload, validation, or rendering failures occurred in this step.

### What I learned

- Plain text diagrams avoided the Mermaid rendering issue from the prior upload.
- The reMarkable bundle workflow handled the larger package with both design documents cleanly.

### What was tricky to build

- The main decision was whether to overwrite the previous bundle or upload a new one. I chose a new name, `XGOJA EXPRESS AUTH Design Options`, because it better describes the expanded content and preserves the earlier MVP-only artifact.

### What warrants a second pair of eyes

- Confirm on reMarkable that the larger bundle remains readable and the two design documents are easy to distinguish in the table of contents.

### What should be done in the future

- If the team chooses one design direction, publish a smaller implementation-focused bundle for that selected path.

### Code review instructions

- Validate docs with `docmgr doctor --ticket XGOJA-EXPRESS-AUTH --stale-after 30`.
- Review both design documents before implementation planning.

### Technical details

Final upload command shape:

```bash
remarquee upload bundle \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/index.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/01-mvp-authentication-api-design-and-implementation-guide.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/design/02-express-style-middleware-auth-design-and-implementation-guide.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/reference/01-investigation-diary.md \
  ttmp/2026/06/12/XGOJA-EXPRESS-AUTH--add-proper-authentication-to-express-http-module/sources/01-auth-preliminary-api-ideas.md \
  --name "XGOJA EXPRESS AUTH Design Options" \
  --remote-dir "/ai/2026/06/12/XGOJA-EXPRESS-AUTH" \
  --toc-depth 2 \
  --non-interactive
```


## Step 5: Select Go-backed fluent builders and break implementation into phases

I recorded the final implementation direction after the design discussion: use the fluent staged route builder as the primary API, with builder/spec objects implemented on the Go side. This keeps the JavaScript surface ergonomic while avoiding defensive parsing of arbitrary JavaScript maps for security-critical auth/resource declarations.

The implementation plan now treats `express.resource("project").idFromParam("projectId").tenantFromParam("orgId")` as HTTP adapter binding rather than domain authorization. The route plan extracts typed values from the HTTP route, but actual resource resolution and authorization remain behind Go interfaces.

### Prompt Context

**User prompt (verbatim):** "ok, add detailed phases and for each phase, detailed list of tasks to be built for this project.

Then implement it task by task, committing at appropriate intervals, keeping a diary as you work."

**Assistant interpretation:** Expand the ticket into concrete implementation phases and begin implementing the selected Go-backed fluent builder approach, committing focused increments and keeping the diary current.

**Inferred user intent:** Move from design/research into execution with a reviewable sequence of tasks, commits, validation, and diary notes.

**Commit (code):** pending — this step prepares the planning update before the first implementation commit.

### What I did

- Re-read the existing ticket task list.
- Replaced the coarse task list with detailed phases:
  - Phase 0: implementation direction and planning
  - Phase 1: `RoutePlan` model and host auth interfaces
  - Phase 2: planned route dispatch and secure context
  - Phase 3: Express Go-backed fluent builders
  - Phase 4: TypeScript declarations and user docs
  - Phase 5: validation, examples, and provider integration
  - Future/out-of-MVP body schema, CSRF, audit, raw-route strict mode, and middleware/router support
- Kept the chosen API direction explicit: Go-backed staged fluent builders are the implementation foundation.

### Why

- The selected direction needs a concrete build order because route-plan data structures, host dispatch, JavaScript builder objects, TypeScript declarations, and docs all depend on each other.
- Separating HTTP value extraction from domain authorization keeps the auth model cleaner than making access control parse raw `req.params`.

### What worked

- The ticket now has a task breakdown detailed enough to drive incremental commits and code review.
- The plan preserves the out-of-MVP concerns without letting body schema, CSRF, audit, or middleware routers block the core auth implementation.

### What didn't work

- No command failures occurred in this planning step.

### What I learned

- The first implementation should not add body schema, CSRF, or audit yet unless the auth core lands cleanly. They remain important, but they are separable from Go-backed auth/resource/allow/handle.

### What was tricky to build

- The tricky part was translating a broad ticket into phases that are small enough to commit independently while still leaving the repository in a compiling/testable state after each phase.
- The selected sequence starts with `pkg/gojahttp` data types and validation because the Express builder should compile to host-level route plans rather than owning auth itself.

### What warrants a second pair of eyes

- Whether `.allow(...)` should be required for every `.auth(...)` route in the first implementation, including current-user routes.
- Whether the compatibility aliases `fromParam` and `withinTenantParam` should be included immediately or delayed to avoid API clutter.

### What should be done in the future

- Implement Phase 1 and commit it before touching JavaScript builder objects.
- Update this diary after each implementation phase with exact tests and failures.

### Code review instructions

- Review `tasks.md` first to understand the implementation sequence.
- Review the first implementation commit against Phase 1 only; do not expect the JS API to exist until Phase 3.

### Technical details

The selected builder style is:

```js
app.route("PATCH", "/orgs/:orgId/projects/:projectId")
  .auth(express.user().required())
  .resource(
    express.resource("project")
      .idFromParam("projectId")
      .tenantFromParam("orgId")
      .mustExist()
  )
  .allow("project.update")
  .handle(handler)
```


## Step 6: Implement Phase 1 route-plan model and registry support

I implemented the first code phase: the host-level route-plan model, auth service interfaces, planned route registration, and route-plan validation. This phase deliberately does not change request dispatch yet; it creates the typed Go foundation that the later Express fluent builders will compile into.

The new model keeps HTTP adapter binding explicit with `ValueSource` and `ResourceSpec`, while the actual authentication, resource lookup, and authorization decisions are delegated to host-provided Go interfaces.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Start implementing the selected Go-backed fluent builder design task by task, beginning with host-level plan data structures and validation.

**Inferred user intent:** Make the auth design concrete in the codebase through small, reviewable commits.

**Commit (code):** pending — Phase 1 code is ready to commit after this diary update.

### What I did

- Added `pkg/gojahttp/auth_plan.go` with:
  - `RoutePlan`
  - `SecuritySpec`
  - `ResourceSpec`
  - `ValueSource`
  - `Actor`
  - `ResourceRef`
  - `AuthOptions`
  - `Authenticator`, `ResourceResolver`, and `Authorizer` interfaces
  - `ErrUnauthenticated`, `ErrForbidden`, and `ErrNotFound`
  - `ValidateRoutePlan`
- Extended `HostOptions` with `Auth AuthOptions`.
- Added `auth AuthOptions` to `Host`.
- Extended `Route` with `Plan *RoutePlan`.
- Added `Registry.AddPlanned`.
- Added `Host.RegisterPlanned`.
- Added `pkg/gojahttp/auth_plan_test.go` with validation and planned registration tests.
- Ran:

```bash
gofmt -w pkg/gojahttp/auth_plan.go pkg/gojahttp/auth_plan_test.go pkg/gojahttp/host.go pkg/gojahttp/route_registry.go
go test ./pkg/gojahttp -count=1
```

### Why

- The Express fluent builder should compile to a typed Go `RoutePlan`; it should not parse arbitrary JavaScript maps during request handling.
- Host-level interfaces keep access control independent of raw HTTP request parsing.
- Planned routes need to coexist with current raw routes, so `RegisterPlanned` was added beside `Register` instead of replacing it.

### What worked

- `go test ./pkg/gojahttp -count=1` passed.
- Existing registry behavior still works because `Host.Register` and `Registry.Add` keep their raw route path.
- Planned route validation catches missing security mode, missing `.allow(action)` on user routes, and resource specs that reference missing path params.

### What didn't work

- No test or compile failures occurred in this phase.

### What I learned

- The existing registry shape was easy to extend with `Plan *RoutePlan` while preserving route matching behavior.
- Keeping `ValueSource` as a small typed struct is enough for `idFromParam`/`tenantFromParam` without coupling the future `ResourceResolver` to raw `req.params`.

### What was tricky to build

- The main tradeoff was how strict `ValidateRoutePlan` should be before the builder exists. I made user-auth routes require an action immediately, because the selected staged builder is supposed to force `.allow(action)` before `.handle(...)`.
- I kept body schema, CSRF, audit, system auth, and capability auth out of the Phase 1 model to avoid designing unused types before the core path compiles.

### What warrants a second pair of eyes

- Whether `SecuritySpec.Required` should be derived for public/user modes as it is now, or left exactly as the builder set it.
- Whether user-auth routes with no action should ever be allowed for current-user endpoints, or whether `.allow("user.self.read")` should always be required.

### What should be done in the future

- Phase 2 should add planned-route dispatch in `Host.ServeHTTP` and use the new auth interfaces.
- Phase 3 should add the JavaScript fluent builders that produce these typed plans.

### Code review instructions

- Start with `pkg/gojahttp/auth_plan.go` and review the type boundaries.
- Then review `Host.RegisterPlanned` in `pkg/gojahttp/host.go` and `Registry.AddPlanned` in `pkg/gojahttp/route_registry.go`.
- Validate with `go test ./pkg/gojahttp -count=1`.

### Technical details

A planned route is stored as the same matched route shape as legacy routes, with an optional plan:

```go
type Route struct {
    Method  string
    Pattern string
    Handler goja.Callable
    Plan    *RoutePlan
}
```
