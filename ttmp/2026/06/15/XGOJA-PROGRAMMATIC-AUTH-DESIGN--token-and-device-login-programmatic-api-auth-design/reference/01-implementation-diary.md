---
Title: Implementation Diary
Ticket: XGOJA-PROGRAMMATIC-AUTH-DESIGN
Status: active
Topics:
    - goja
    - xgoja
    - auth
    - security
    - rest-api
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: modules/express/auth_builders.go
      Note: Express fluent rate-limit builders implemented in commit 1486dbb
    - Path: modules/express/typescript.go
      Note: PlannedContext AuthInfo TypeScript declarations added in commit 1add4b5
    - Path: pkg/gojahttp/api_token_integration_test.go
      Note: End-to-end planned-route API-token behavior (commit 00a1e86)
    - Path: pkg/gojahttp/auth/programauth/agent.go
      Note: Programmatic Agent model/service and actor projection added in commit 5800dd7
    - Path: pkg/gojahttp/auth/programauth/composite.go
      Note: Bearer-first/session-fallback composite authenticator (commit 00a1e86)
    - Path: pkg/gojahttp/auth/programauth/logcopter.go
      Note: Generated log metadata committed after go generate hook (commit 5412cc6)
    - Path: pkg/gojahttp/auth/programauth/memory_store.go
      Note: In-memory AgentStore with clone isolation/listing/disablement added in commit 5800dd7
    - Path: pkg/gojahttp/auth/programauth/memory_token_store.go
      Note: In-memory API-token store with prefix lookup/list/revoke/touch behavior (commit 00a1e86)
    - Path: pkg/gojahttp/auth/programauth/token.go
      Note: API-token service
    - Path: pkg/gojahttp/auth_plan.go
      Note: |-
        AuthResult
        AuthResult GrantSet field added in commit 5800dd7
    - Path: pkg/gojahttp/enforcer.go
      Note: |-
        Pre/post route limit enforcement implemented in commit 1486dbb
        ResultAuthenticator integration
        Grant intersection enforcement for AuthResult.Grants (commit 00a1e86)
    - Path: pkg/gojahttp/grants.go
      Note: Typed GrantSet model
    - Path: pkg/gojahttp/planned_dispatch.go
      Note: SecureContext Auth field and redacted ctx.auth JavaScript projection added in commit 1add4b5
    - Path: pkg/gojahttp/ratelimit.go
      Note: Core rate-limit model and in-memory limiter implemented in commit 1486dbb
    - Path: pkg/xgoja/hostauth/services.go
      Note: Generated hostauth rate-limiter wiring implemented in commit 1486dbb
    - Path: ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md
      Note: Primary implementation guide produced by this ticket.
    - Path: ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/sources/01-current-auth-surface.md
      Note: Current code evidence used to ground the guide.
ExternalSources: []
Summary: Diary for creating the token/device-login programmatic API auth design ticket and reMarkable bundle.
LastUpdated: 2026-06-15T22:00:00-04:00
WhatFor: Track what was created, why, and how to continue implementation.
WhenToUse: Read before resuming implementation of token-based or device-login-based programmatic API access.
---






# Diary

## Goal

Capture the creation of a new docmgr ticket and implementation guide for token-based and device-login-based programmatic API access for planned go-go-goja REST routes.

## Step 1: Create the programmatic auth design ticket

I created a new ticket dedicated to the follow-up design rather than adding the work to the independent PR 74 review ticket. This keeps the code review deliverable separate from the forward-looking implementation plan.

The ticket contains a primary design/implementation guide, a diary, and a source evidence file with line anchors from the current PR 74 auth surfaces.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket and a detailed analysis + implementation guide in that ticket eto build token based / device login based programmatic api access auth. Then upload to remarkable. COmmit all ticket stuff to disk."

**Assistant interpretation:** Create a separate docmgr ticket, write a detailed design and implementation guide for API-token/device-login/refresh-token programmatic auth, persist all files, upload the bundle to reMarkable, and keep ticket bookkeeping current.

**Inferred user intent:** Turn the earlier discussion into an implementation-ready design package that can be read offline and used to start building the feature.

**Commit (code):** N/A — ticket documentation only at this step.

### What I did

- Created docmgr ticket `XGOJA-PROGRAMMATIC-AUTH-DESIGN`.
- Added design doc `design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md`.
- Added diary doc `reference/01-implementation-diary.md`.
- Added source evidence doc `sources/01-current-auth-surface.md`.
- Wrote the implementation guide with package-level, API-level, store-level, endpoint-level, and test-level detail.

### Why

- The programmatic auth design spans multiple packages and should be implemented in phases.
- The current PR 74 planned-route architecture provides the right foundation, but token/device auth needs a clear contract for auth method, scopes, refresh-token rotation, device code approval, and generated-host wiring.

### What worked

- The current code has clear extension points:
  - `gojahttp.AuthOptions`
  - `gojahttp.Authenticator`
  - `planned_dispatch.go`
  - `sessionauth.Manager`
  - `hostauth.StoreBundle`
  - `hostauth.Services`
  - HTTP provider `serve` host-service wiring.
- The design can preserve the current JS route declaration model while adding Go-owned credential handling.

### What didn't work

- N/A.

### What I learned

- The main missing abstraction is an auth result that carries actor, method, credential ID, scopes, and CSRF behavior.
- API tokens and refresh tokens should not be conflated: PAT/service API tokens should be directly revocable credentials, while device/OAuth-style clients should use short-lived access tokens plus rotating refresh tokens.

### What was tricky to build

- The design needed to balance minimal JS API changes with future route-level credential restrictions. The guide recommends no required JS changes for v1 and optional builder sugar later.
- Refresh-token support needed explicit reuse detection and family revocation; otherwise token rotation can give a false sense of security.

### What warrants a second pair of eyes

- Whether `AuthResult` should be introduced as an optional compatibility interface or as a breaking replacement for `Authenticator`.
- Whether generated-host auth should mount token/device handlers automatically or require explicit host opt-in.
- Whether service-account API tokens should be included in v1 or added after personal API tokens.

### What should be done in the future

- Implement the guide in the recommended PR split:
  1. AuthResult plumbing.
  2. API tokens.
  3. Generated-host API-token wiring.
  4. Access/refresh tokens.
  5. Device login.
  6. Optional JS builder sugar and docs.

### Code review instructions

- Start with the `Executive summary`, `Proposed architecture`, and `Implementation phases` sections of the guide.
- Validate implementation work against the `Security invariants` and `Testing strategy` sections.
- Use `sources/01-current-auth-surface.md` to locate current integration points.

### Technical details

Important paths:

```text
ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md
ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/reference/01-implementation-diary.md
ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/sources/01-current-auth-surface.md
```

## Step 2: Review the plan against external best practice and current code

I revisited the original programmatic auth plan as a design review rather than as an implementation task. The goal was to test the plan against OWASP, IETF, NIST, and practical fine-grained token guidance, then update the ticket with a more opinionated design for JavaScript-authored programmatic agent APIs.

The main outcome is a new long-form review document. It keeps the original plan's strongest ideas, but shifts the implementation target from `planned_dispatch.go` to the current `gojahttp.Enforcer`, adds a first-class programmatic agent model, and recommends Go-backed fluent builders for `auth.agents`, `auth.tokens`, `auth.device`, and typed grant construction.

### Prompt Context

**User prompt (verbatim):** "The plan was written by your little brother. Can you do some research on best practice (say, owasp and co), using defuddle to download the resources to the sources/ directory (could be that they already did that), and then review their plan in depth and figure out if not only things are correct and well written out, but reflecting on potential missed opportunities to create a simple yet opiinoinated way to build javascript APIs around programmatic agent design. We want to have opinionated but also flexible / general APIs to use, with some core principles like minimizing the use of js maps / object sto avoid overly defensive programming and using fluent, builder api style where the core objects are under go control. 

Create  a detailed analysis / design / code review that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Research external API/OAuth/session/token best practice, save sources into the ticket, deeply review the existing programmatic auth guide, and write a new intern-friendly design/code-review document emphasizing opinionated Go-owned fluent JavaScript APIs for programmatic agents.

**Inferred user intent:** Improve the quality of the original design before implementation so the next engineer builds a secure, ergonomic, reusable programmatic auth surface rather than a pile of token endpoints.

**Commit (code):** N/A — documentation and ticket-source updates only.

### What I did

- Downloaded OWASP API Security, OWASP cheat sheets, IETF RFCs, NIST session guidance, and GitHub fine-grained PAT guidance into `sources/`.
- Used `defuddle parse --md` for the standard web pages.
- Stored RFC 8628 and RFC 6750 from RFC Editor plain text after defuddle could not extract their HTML pages.
- Added `design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md`.
- Related the new design doc to current implementation files:
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go`
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go`
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go`
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/hostauth/hostauth.go`
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go`
  - `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/verbs/sites.js`
- Added and checked ticket tasks for source download, review writing, and reMarkable upload.
- Dry-ran and uploaded the bundle `XGOJA Programmatic Auth Best Practice Review` to `/ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN`.
- Fixed a Mermaid label that broke rendering and re-uploaded the bundle with `--force`.

### Why

- The original plan was written before later auth work moved enforcement into `gojahttp.Enforcer` and before the `require("auth")` fluent audit/capability APIs existed.
- External guidance makes rate limiting, header-only bearer transport, refresh-token rotation, object-level authorization, and device-code polling semantics more explicit than the original plan did.
- Programmatic auth should be designed as an agent/API product surface, not just token storage and endpoints.

### What worked

- The current codebase already has the right API style precedent in `auth.audit.query()` and `auth.capabilities.issue()`.
- The staged `modules/express` builder pattern already demonstrates how to avoid object-bag defensive programming.
- The existing generated-host service factory is a good seam for future token/device stores and native handlers.

### What didn't work

- `defuddle parse https://www.ietf.org/rfc/rfc8628.html --md` failed with:
  `Error: No content could be extracted from https://www.ietf.org/rfc/rfc8628.html`
- `defuddle parse https://www.ietf.org/rfc/rfc6750.html --md` failed with:
  `Error: No content could be extracted from https://www.ietf.org/rfc/rfc6750.html`
- Retrying the RFC Editor HTML pages also failed with the same extraction error.
- I stored the RFC Editor `.txt` versions for RFC 8628 and RFC 6750 with a `FetchNote` explaining the fallback.
- The first real `remarquee upload bundle` succeeded, but emitted Mermaid rendering warnings:
  `Error: Parse error on line 24: ... JSAuthModule[require("auth") fluent APIs ... got 'PS'`
- I changed the Mermaid subgraph label to `auth module fluent APIs`, reran a dry-run, and re-uploaded with `--force`; the second upload completed without warnings.

### What I learned

- The original plan's security instincts are mostly correct, but it under-specifies the product model around durable automation identities.
- OWASP API2 and RFC 8628 make rate limiting part of the auth design, not merely an operational note.
- NIST's session guidance is useful for explaining why bearer access tokens must not be treated as proof of active browser-session presence.
- GitHub fine-grained PAT guidance is a useful practical model: token access is constrained by owner access and further narrowed by token permissions.

### What was tricky to build

- The review had to distinguish between what was wrong in the old plan and what simply became stale after later auth work. The key example is `planned_dispatch.go`: it still projects JavaScript context, but `gojahttp.Enforcer` now owns the enforcement algorithm and must be the primary AuthResult integration target.
- The JavaScript API proposal needed to stay flexible without collapsing into free-form object maps. The solution was to recommend high-level `auth.agents` builders and lower-level `auth.tokens` builders, all backed by Go-owned state and explicit `.run()` side-effect boundaries.

### What warrants a second pair of eyes

- Whether the implementation should use one cohesive `programauth` package or keep separate `apitoken`, `tokenauth`, and `deviceauth` packages with a shared grant/agent package.
- Whether `express.agent()` and `express.sessionUser()` should ship in the first AuthResult PR or wait until after API-token authentication works.
- Whether production generated-host config should fail hard without a configured rate limiter when device login is enabled, or start with warnings plus a no-op dev limiter.

### What should be done in the future

- Update the original `design/01-...` guide to mark stale integration points and link to the new review.
- Implement Phase 1 from the new review: `AuthResult` in `gojahttp.Enforcer` and safe `ctx.auth` projection.
- N/A.

### Code review instructions

- Start with `design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md`.
- Pay special attention to the sections `Review of the existing plan`, `Opinionated JavaScript API design`, `Revised enforcement algorithm`, and `Concrete implementation phases`.
- Validate the claims against the cited files and downloaded sources in `sources/`.

### Technical details

Important commands and outcomes:

```bash
defuddle parse https://owasp.org/API-Security/editions/2023/en/0x11-t10/ --md | fold -w 100 -s > sources/02-owasp-api-security-top-10-2023.md
defuddle parse https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html --md | fold -w 100 -s > sources/10-owasp-oauth2-cheat-sheet.md
defuddle parse https://www.ietf.org/rfc/rfc8628.html --md
# Error: No content could be extracted from https://www.ietf.org/rfc/rfc8628.html
curl -fsSL https://www.rfc-editor.org/rfc/rfc8628.txt > sources/12-ietf-rfc8628-device-authorization-grant.md
remarquee upload bundle ... --name "XGOJA Programmatic Auth Best Practice Review" --remote-dir "/ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN" --toc-depth 2 --dry-run --non-interactive
remarquee upload bundle ... --name "XGOJA Programmatic Auth Best Practice Review" --remote-dir "/ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN" --toc-depth 2 --force --non-interactive
# OK: uploaded XGOJA Programmatic Auth Best Practice Review.pdf -> /ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN
```

Primary new document:

```text
ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md
```

## Step 3: Promote rate limiting to a planned-route primitive

I updated the revised programmatic auth review to treat rate limiting as a first-class planned-route primitive, parallel to `.audit(...)` and `.csrf()`. The design now says rate limits should apply to all planned routes, with pre-auth limits for cheap IP/route keys and post-auth limits for actor, tenant, and resource-aware quotas.

This is a meaningful correction to the previous auth design. Programmatic auth makes the omission more visible because token, refresh, and device endpoints are obvious abuse targets, but OWASP API4 applies to every API route. The right abstraction is therefore not a few special auth-endpoint throttles; it is a general `RateLimitSpec` compiled into `RoutePlan` and enforced by `gojahttp.Enforcer`.

### Prompt Context

**User prompt (verbatim):** "- can the rate limiting be applied to all routes now? I must admit I forgot that in my previous express auth design that we are building upon. It probably makes sense to make rate limiting a fundamental primitive like audit() ."

**Assistant interpretation:** Confirm whether rate limiting should become a general route primitive and update the programmatic auth design accordingly.

**Inferred user intent:** Avoid designing rate limiting as a narrow auth-flow afterthought; make it part of the core Express/planned-route security DSL so all routes can declare and inherit request budgets.

**Commit (code):** N/A — documentation update only.

### What I did

- Added a `Route-level rate limiting primitive` section to the best-practice review.
- Updated the design principles to list `rateLimit` next to `auth`, `resource`, `allow`, `csrf`, and `audit`.
- Added JavaScript API sketches for `.rateLimit(express.rateLimit(...))` on public, authenticated, and auth-sensitive routes.
- Added Go API sketches for `RateLimitSpec`, `RateLimitKeyPart`, `RateLimiter`, and `AuthOptions.RateLimiter`.
- Updated the enforcer pseudocode with pre-auth and post-auth rate-limit checks.
- Added security invariants, implementation-phase tasks, tests, code-review checklist items, and a decision record for rate limiting.
- Re-uploaded the revised bundle to reMarkable with `--force` at `/ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN`.

### Why

- OWASP API4 is about unrestricted resource consumption across APIs, not just authentication endpoints.
- OWASP API2 and RFC 8628 require stricter throttling for auth/device flows, but that should be a specialization of a general primitive.
- Route authors know the business cost of a route; the host knows normalized keys such as IP, actor, tenant, principal, and resource. A planned-route rate-limit primitive lets both cooperate without unsafe JavaScript key concatenation.

### What worked

- The existing planned-route DSL already has the right shape: `.audit(...)` is available on public and authenticated stages, so `.rateLimit(...)` can follow the same pattern.
- The current staged builder design can add `express.rateLimit(policy)` as another Go-backed builder object stored in `builderStore`.
- `gojahttp.Enforcer` is the right enforcement point because it can run cheap pre-auth limits before expensive auth work and actor/tenant/resource limits after authentication/resource extraction.

### What didn't work

- N/A.

### What I learned

- A single limiter pass is not enough. Pre-auth limits protect public/login/device endpoints before expensive work; post-auth limits support actor/tenant/resource quotas.
- Rate-limit failures should be audited as `429` denials with redacted policy/key hints.
- Generated hosts should support default route-limit policies even when JavaScript authors forget to call `.rateLimit(...)`.

### What was tricky to build

- The design needed to avoid making rate limits either too early or too late in the pipeline. If limits only run after authentication, public routes and login/device endpoints remain exposed. If limits only run before authentication, per-actor and per-tenant quotas are impossible. The proposed solution is a compiled two-stage limiter plan.

### What warrants a second pair of eyes

- Whether route-level rate-limit specs should live directly in `RoutePlan` or in a separate policy registry referenced by route policy name.
- Whether production generated-host config should fail closed for all limiter outages or only for auth-sensitive routes.
- Whether limiter defaults should be injected globally by generated-host config even if route plans do not declare `.rateLimit(...)`.

### What should be done in the future

- Implement `RateLimitSpec` in the same PR as `AuthResult` or as the immediately following PR, because both belong in `gojahttp.Enforcer`.

### Code review instructions

- In `design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md`, review `Route-level rate limiting primitive`, `Revised enforcement algorithm`, `Security invariants`, `Concrete implementation phases`, and the rate-limit decision record.
- Validate future code by adding tests for public pre-auth limits, authenticated actor limits, tenant/resource limits, generated defaults, and `429` audit redaction.

### Technical details

Primary changed document:

```text
ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/02-best-practice-review-and-opinionated-javascript-api-design-for-programmatic-auth.md
```

Upload command:

```bash
remarquee upload bundle ... --name "XGOJA Programmatic Auth Best Practice Review" --remote-dir "/ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN" --toc-depth 2 --force --non-interactive
# OK: uploaded XGOJA Programmatic Auth Best Practice Review.pdf -> /ai/2026/06/18/XGOJA-PROGRAMMATIC-AUTH-DESIGN
```

## Step 4: Implement planned-route rate limiting

I implemented the first programmatic-auth implementation phase by making rate limiting a compiled planned-route primitive instead of a handler-local convention. Routes can now declare one or more rate-limit policies in Go or JavaScript, and `gojahttp.Enforcer` evaluates those policies before or after authentication depending on the key material they need.

The implementation keeps the security-sensitive pieces Go-owned: JavaScript receives fluent builders, route plans store typed specs, generated hosts get a default in-memory limiter for development, and production hosts can replace it through `AuthOptions.RateLimiter` without changing route declarations.

### Prompt Context

**User prompt (verbatim):** "Create all phases/tasks for the programmatic auth work, then fully implement the rate-limiting phases, keep a detailed diary, and commit at appropriate intervals."

**Assistant interpretation:** Turn the programmatic-auth design into docmgr phase tasks, complete the rate-limiting implementation phases, validate them, record the work in the diary, and commit focused changes.

**Inferred user intent:** Move from design into a reviewable first implementation slice while keeping ticket bookkeeping and commit history clean enough for later auth/token/device phases.

**Commit (code):** 1486dbbba2a840d879c4087be2f28faf67986458 — "gojahttp: add planned route rate limiting"

### What I did

- Added docmgr tasks for Phase 1A through Phase 1E and later Phase 2 through Phase 9 programmatic-auth work.
- Added `pkg/gojahttp/ratelimit.go` with typed `RateLimitSpec`, key parts, stage selection, `RateLimiter`, `RateLimitRequest`, `RateLimitDecision`, `RateLimitError`, validation, deterministic key construction, and a fixed-window `MemoryRateLimiter`.
- Added `RateLimits []RateLimitSpec` to `RoutePlan` and `RateLimiter RateLimiter` to `AuthOptions`.
- Integrated pre-auth and post-auth checks into `gojahttp.Enforcer`, including `429 Too Many Requests` status mapping and `Retry-After` response handling.
- Added Go builder support through `gojahttp.RateLimit(policy)` and `.RateLimit(...)` methods on planned route policy/handler phases.
- Added Express support through `express.rateLimit(policy)`, fluent builder methods such as `.perMinute(...)`, `.byIP()`, `.byActor()`, `.byParam(...)`, `.failOpen(...)`, and planned-route `.rateLimit(...)` methods.
- Updated generated-host auth services so `BuildHostAuthServices` creates a default `gojahttp.NewMemoryRateLimiter()` and `BuildAuthOptions` wires it into `gojahttp.AuthOptions`.
- Added tests for validation, pre-auth enforcement, post-auth enforcement, Go builder usage, Express builder integration, plain-object rejection, and generated-host service wiring.
- Marked tasks 10 through 14 complete and updated the ticket changelog.

### Why

- Rate limiting is now part of the route security contract alongside auth, resource resolution, authorization, CSRF, and audit.
- Keeping limiter keys as typed Go-owned route-plan data avoids making JavaScript handlers concatenate security-sensitive bucket identifiers.
- Two-stage enforcement protects public/auth endpoints before expensive auth work while still allowing actor/resource-aware quotas after authentication.

### What worked

- The staged planned-route builders already had natural extension points for `.RateLimit(...)` and `.rateLimit(...)`.
- `gojahttp.Enforcer` had enough request, actor, resource, and route-plan context to centralize limiter enforcement cleanly.
- Focused package tests and the full repository test suite passed after the lint fix.
- The pre-commit hook passed lint, glazed lint, go generation, and `go test ./...` before creating commit `1486dbb`.

### What didn't work

- A focused test run initially failed after `RouteDescriptor.RateLimitPolicies` was introduced as `[]string`; existing tests compared route descriptors with non-comparable struct values. I changed `RateLimitPolicies` to a comma-separated `string` populated with `strings.Join(...)`.
- The first commit attempt failed in `golangci-lint` exhaustive checking:
  `pkg/gojahttp/ratelimit.go:230:3: missing cases in switch of type gojahttp.RateLimitKeyKind: gojahttp.RateLimitKeyIP, gojahttp.RateLimitKeyRoute, gojahttp.RateLimitKeyParam, gojahttp.RateLimitKeyTenantParam, gojahttp.RateLimitKeyHeader, gojahttp.RateLimitKeyBodyField (exhaustive)`
- I fixed the lint error by making `rateLimitStage` explicitly continue for pre-auth key kinds.

### What I learned

- The rate-limit stage decision must be explicit enough for the repository's exhaustive linter, even when the default behavior is pre-auth.
- Route descriptor fields used in test struct comparisons should remain comparable or tests need to change shape.
- Generated-host defaults are useful for development, but the route declaration still fails closed if a plan declares limits and the host has no limiter configured.

### What was tricky to build

- The main sharp edge was preserving a small JavaScript API while still supporting multiple keying strategies. The solution was to use Go-owned builder objects stored in `builderStore`, reject arbitrary object bags, and copy typed specs into `RoutePlan`.
- The enforcement order mattered. Pre-auth limit checks need route and request information only; post-auth checks need actor/resource data. I implemented stage inference from key parts so route authors do not need to choose the stage manually.
- Fixed-window memory limiting needed deterministic keys and safe concurrency. The memory limiter keeps buckets under a mutex, resets windows when they expire, and returns `RetryAfter` based on the current bucket window.

### What warrants a second pair of eyes

- Whether the first production limiter interface needs richer metadata such as cost/weight before non-dev backends are implemented.
- Whether `RateLimitKeyBodyField` should remain in the public builder before body parsing semantics are finalized for planned routes.
- Whether `FailOpen` should be allowed on auth-sensitive routes or require host-level policy restrictions.

### What should be done in the future

- Implement Phase 2: `AuthResult`, `ResultAuthenticator`, and safe redacted `ctx.auth` projection.
- Add production/distributed limiter examples after the in-memory development limiter has been exercised.
- Consider generated-host default policies for auth/device/token endpoints once those endpoints exist.

### Code review instructions

- Start with `pkg/gojahttp/ratelimit.go` for the model, validation, key construction, and memory limiter.
- Then review `pkg/gojahttp/enforcer.go` for enforcement order and error/status handling.
- Review `pkg/gojahttp/app.go` and `modules/express/auth_builders.go` for Go and JavaScript builder ergonomics.
- Review `pkg/xgoja/hostauth/services.go` and `pkg/xgoja/hostauth/builder.go` for generated-host wiring.
- Validate with `go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth` and `go test ./...`.

### Technical details

Key commands and outcomes:

```bash
gofmt -w pkg/gojahttp/route_registry.go modules/express/ratelimit_integration_test.go
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

go test ./...
# ok for full repository before commit

git commit -m "gojahttp: add planned route rate limiting"
# first attempt failed exhaustive lint on rateLimitStage switch

gofmt -w pkg/gojahttp/ratelimit.go
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

git commit -m "gojahttp: add planned route rate limiting"
# pre-commit lint/test passed; commit 1486dbbba2a840d879c4087be2f28faf67986458
```

Primary files:

```text
pkg/gojahttp/ratelimit.go
pkg/gojahttp/enforcer.go
pkg/gojahttp/auth_plan.go
pkg/gojahttp/app.go
pkg/gojahttp/planned_dispatch.go
pkg/gojahttp/route_registry.go
modules/express/auth_builders.go
modules/express/express.go
modules/express/typescript.go
pkg/xgoja/hostauth/services.go
pkg/xgoja/hostauth/builder.go
```

## Step 5: Add AuthResult and redacted ctx.auth projection

I continued past rate limiting into the next implementation phase: authentication now returns a non-secret `AuthResult` when the host authenticator supports it. The enforcer still accepts the existing `Authenticator` interface, but adapts legacy session authenticators into an `AuthResult` so existing hosts keep working.

This gives future API-token, access-token, device-login, and agent work a stable place to put credential method, principal identity, credential IDs/hints, scopes, and CSRF behavior without overloading `Actor.Claims` or exposing secrets to JavaScript.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue with the next programmatic-auth implementation phase after the completed rate-limiting slice.

**Inferred user intent:** Keep progressing through the ticket tasks in order, committing focused implementation work and preserving the implementation diary.

**Commit (code):** 1add4b5ebbd58fbabe6974465ecefe8978cd370f — "gojahttp: expose auth results in planned contexts"

### What I did

- Added `AuthMethod` constants for `none`, `session`, `apiToken`, and `accessToken`.
- Added `PrincipalKind` constants for `user`, `agent`, and `service`.
- Added `AuthResult` with actor, method, principal, credential ID/hint, scopes, and `CSRFRequired` metadata.
- Added `ResultAuthenticator` as an optional richer interface while preserving the existing `Authenticator` interface.
- Updated `gojahttp.Enforcer` to prefer `ResultAuthenticator`, normalize returned results, and adapt legacy authenticators as session/user results.
- Added `Auth AuthResult` to `SecureContext`.
- Projected a redacted `ctx.auth` object into planned JavaScript handlers.
- Made route CSRF enforcement conditional on auth method metadata for authenticated routes, while preserving public-route CSRF checks.
- Added redacted auth metadata to route audit attributes.
- Updated Express TypeScript declarations with `AuthInfo`.
- Added tests for legacy-session compatibility, redacted `ctx.auth`, scope-copy safety, and CSRF skipping for token-style auth results.
- Marked task 15 complete and updated the changelog.

### Why

- Programmatic credentials need method and credential metadata that should not be mixed into actor identity.
- JavaScript handlers should be able to inspect non-secret caller metadata, but should never receive raw tokens, token hashes, refresh token IDs, or device codes.
- API tokens should not have to satisfy browser-session CSRF checks; session-based auth remains CSRF-required through the compatibility adapter.

### What worked

- The existing `Authenticator` interface could remain intact by adding `ResultAuthenticator` as an optional extension interface.
- `SecureContext` was already the right carrier for enforcement output, so adding `Auth` required only local changes in the enforcer and JavaScript projection.
- Focused package tests and `go test ./...` passed before committing.
- The pre-commit hook passed lint, glazed lint, generation, and the full test suite before creating commit `1add4b5`.

### What didn't work

- N/A. No compile or test failures occurred in this step.

### What I learned

- CSRF behavior is easier to reason about when it is attached to the authentication result instead of inferred from actor kind or route security mode alone.
- The compatibility adapter gives legacy session auth the right defaults: method `session`, principal kind `user`, principal ID from `Actor.ID`, and `CSRFRequired=true`.
- `ctx.auth` should remain a projection, not the source of authorization truth; route authorization still happens before the handler runs.

### What was tricky to build

- The subtle part was preserving existing public-route CSRF behavior. Authenticated token requests can opt out with `CSRFRequired=false`, but public unsafe routes that declare `.csrf()` still need the CSRF verifier to run because they have no `AuthResult` to consult.
- Another edge was mutation safety. `AuthResult.Scopes` is copied during normalization and copied again during JavaScript projection so handler mutations do not alter host-owned scope slices.

### What warrants a second pair of eyes

- Whether `CredentialID` should be exposed to JavaScript or only to audit; it is intended to be a stable non-secret ID, but future token implementations must keep it that way.
- Whether `AuthResult` should grow typed grants in this struct or keep grants as a separate Phase 3 abstraction with scopes as the current wire/debug view.
- Whether audit attributes should include `principalId` for every route or only for auth-sensitive events.

### What should be done in the future

- Implement Phase 3: typed grants and first-class agent model.
- Implement composite bearer/session authentication once API-token and access-token authenticators exist.
- Add docs/examples that demonstrate `ctx.auth` as diagnostic metadata rather than an authorization mechanism.

### Code review instructions

- Start with `pkg/gojahttp/auth_plan.go` for the new public auth metadata types.
- Review `pkg/gojahttp/enforcer.go` for compatibility behavior, normalization, CSRF gating, and audit attributes.
- Review `pkg/gojahttp/planned_dispatch.go` for the redacted JavaScript projection.
- Review `pkg/gojahttp/planned_dispatch_test.go` for the intended `ctx.auth` and CSRF behavior.
- Validate with `go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth` and `go test ./...`.

### Technical details

Key commands and outcomes:

```bash
gofmt -w pkg/gojahttp/auth_plan.go pkg/gojahttp/enforcer.go pkg/gojahttp/planned_dispatch.go modules/express/typescript.go
go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

go test ./...
# ok for full repository

git commit -m "gojahttp: expose auth results in planned contexts"
# pre-commit lint/test passed; commit 1add4b5ebbd58fbabe6974465ecefe8978cd370f
```

Primary files:

```text
pkg/gojahttp/auth_plan.go
pkg/gojahttp/enforcer.go
pkg/gojahttp/planned_dispatch.go
pkg/gojahttp/enforcer_test.go
pkg/gojahttp/planned_dispatch_test.go
modules/express/typescript.go
```

## Step 6: Add typed grants and first-class agents

I added the first durable programmatic-auth domain model: typed grants and automation agents. Grants are now represented as structured Go data with deterministic normalization, matching, cloning, and scope-string serialization, while agents provide a lifecycle object that future API tokens and device clients can attach to.

This step deliberately stops before token issuance. The goal is to create the policy and principal foundation first, so later token work can authenticate credentials into an agent-backed `AuthResult` instead of inventing token-specific policy strings or anonymous automation identities.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Continue with the next programmatic-auth implementation phase after AuthResult plumbing.

**Inferred user intent:** Keep building the feature in small reviewable slices, with tests, diary notes, task bookkeeping, and focused commits.

**Commit (code):** 5800dd7aa1f086acc6fbdd645b2ef85bbcf54a84 — "gojahttp: add programmatic grants and agents"; 5412cc6dbd990712801270c3d44665172e69d332 — "programauth: add generated log metadata"

### What I did

- Added `gojahttp.Grant` and `gojahttp.GrantSet`.
- Implemented grant normalization, validation, deduplication, deterministic sorting, clone isolation, scope-string serialization, and action/tenant/resource matching.
- Added `Grants GrantSet` to `AuthResult`; `normalizeAuthResult` now normalizes grants and derives scope strings when explicit scopes are not provided.
- Added `pkg/gojahttp/auth/programauth` with:
  - `AgentKind` constants for personal, service, device, CI, and integration agents.
  - `Agent` as a durable automation identity with owner, tenant, disablement, timestamps, and grant policy.
  - `AgentCreateSpec`, `AgentQuery`, `AgentStore`, and `AgentService`.
  - `MemoryAgentStore` for tests, examples, and local generated hosts.
  - actor projection from agent to `gojahttp.Actor` with `PrincipalKindAgent`.
- Added tests for grant normalization/matching/wildcards, agent creation/validation, list filtering, disablement, actor projection, and store clone isolation.
- Committed generated `programauth/logcopter.go` metadata produced by repository generation hooks.
- Marked task 16 complete and updated the changelog.

### Why

- Programmatic credentials should reference durable principals; otherwise long-lived tokens become anonymous blobs that are hard to list, disable, audit, or explain.
- Typed grants let Go own policy semantics while still allowing OAuth-style scope strings at storage/protocol boundaries.
- The in-memory agent store provides a low-friction implementation for tests and development while preserving a store interface for SQL-backed production work later.

### What worked

- Keeping grants in `gojahttp` lets `AuthResult` carry typed policy without introducing an import cycle.
- Keeping the agent service in `pkg/gojahttp/auth/programauth` lets the programmatic-auth model evolve separately from the core enforcer.
- Focused package tests and `go test ./...` passed before committing.
- The pre-commit hook passed lint, glazed lint, generation, and the full test suite before creating commit `5800dd7`.

### What didn't work

- N/A. No compile or test failures occurred in this step.

### What I learned

- `GrantSet` needs both a typed matcher and a deterministic string projection. The matcher is the source of truth; scope strings are for debugging, wire protocols, and storage.
- Agent policy clone isolation matters even for an in-memory dev store because tests and future JavaScript APIs may mutate returned data.
- Agent disablement should be modeled on the service boundary: the store can return disabled records for admin listing, while `AgentService.GetAgent` rejects disabled agents for active-auth use.

### What was tricky to build

- The grant matching semantics needed careful wildcard behavior. Empty tenant/resource fields act as wildcards, but populated fields must match exactly; the action `*` is allowed while still respecting tenant/resource restrictions.
- Package placement was the main design edge. Putting `GrantSet` only in `programauth` would make `gojahttp.AuthResult` depend on a subpackage and create an import cycle risk. I put `Grant`/`GrantSet` in `gojahttp` and kept agent lifecycle in `programauth`.

### What warrants a second pair of eyes

- Whether scope-string format should be finalized now or treated as provisional until API-token storage is implemented.
- Whether `AgentService.GetAgent` should return disabled agents plus a status instead of returning `ErrAgentDisabled`; the current behavior is auth-path oriented.
- Whether agent IDs should use a host-provided ID generator in production rather than the current secure random default.

### What should be done in the future

- Implement Phase 4: API-token issue/list/revoke/authenticate and bearer planned-route auth.
- Add a SQL-backed `programauth` agent store before production use.
- Wire generated hostauth services to construct a programauth agent store once token issuance is added.

### Code review instructions

- Start with `pkg/gojahttp/grants.go` and `pkg/gojahttp/grants_test.go` for grant semantics.
- Review `pkg/gojahttp/auth/programauth/agent.go` for the agent model and service boundary.
- Review `pkg/gojahttp/auth/programauth/memory_store.go` and `agent_test.go` for clone isolation, listing, and disablement behavior.
- Review `pkg/gojahttp/enforcer.go` only for the small `AuthResult.Grants` normalization/scope derivation change.
- Validate with `go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth` and `go test ./...`.

### Technical details

Key commands and outcomes:

```bash
gofmt -w pkg/gojahttp/grants.go pkg/gojahttp/grants_test.go pkg/gojahttp/auth_plan.go pkg/gojahttp/enforcer.go pkg/gojahttp/auth/programauth/agent.go pkg/gojahttp/auth/programauth/memory_store.go pkg/gojahttp/auth/programauth/agent_test.go
go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

go test ./...
# ok for full repository

git commit -m "gojahttp: add programmatic grants and agents"
# pre-commit lint/test passed; commit 5800dd7aa1f086acc6fbdd645b2ef85bbcf54a84

git commit -m "programauth: add generated log metadata"
# pre-commit lint/test passed; commit 5412cc6dbd990712801270c3d44665172e69d332
```

Primary files:

```text
pkg/gojahttp/grants.go
pkg/gojahttp/grants_test.go
pkg/gojahttp/auth_plan.go
pkg/gojahttp/enforcer.go
pkg/gojahttp/auth/programauth/agent.go
pkg/gojahttp/auth/programauth/memory_store.go
pkg/gojahttp/auth/programauth/agent_test.go
pkg/gojahttp/auth/programauth/logcopter.go
```

## Step 7: Add API-token issue, storage, authentication, and bearer route enforcement

I implemented the first credential family on top of the new agent and grant foundation. Programauth can now issue opaque API tokens, store only hashes plus redacted prefixes, list/revoke records without returning secrets, and authenticate `Authorization: Bearer ...` requests into `AuthResult` values backed by agents and typed grants.

This completes the first end-to-end programmatic route-auth path: a token can call a planned route, the enforcer skips browser CSRF for bearer auth, still resolves resources, intersects the route action with token grants, and then runs the application authorizer.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Continue with the next programmatic-auth implementation phase after grants and agents.

**Inferred user intent:** Keep progressing through the implementation plan in focused, tested commits while maintaining ticket docs and diary continuity.

**Commit (code):** 00a1e866e75c17cb07121463021e03ca1d046725 — "programauth: add API token authentication"

### What I did

- Added API-token records, safe list views, issue specs, query specs, and issue results in `programauth`.
- Added SHA-256 token hashing plus optional peppered HMAC hashing.
- Implemented opaque token generation with `ggpat_<prefix>_<secret>` format; the prefix is usable for lookup and diagnostics, while the full token is returned only on issuance.
- Added strict `Authorization: Bearer <token>` parsing that rejects malformed/duplicate authorization headers and `access_token` query parameters.
- Added `APITokenStore` and `MemoryAPITokenStore` with create, prefix lookup, list, revoke, and touch operations.
- Added `APITokenService` issue/list/revoke/authenticate methods.
- Added `CompositeAuthenticator`, which tries header bearer auth first and falls back to a configured session authenticator when no bearer token is present.
- Updated `gojahttp.Enforcer` to deny authenticated routes when `AuthResult.Grants` are present and do not allow the planned route action/resource.
- Added planned-route integration coverage: an API-token request can call a CSRF-declared route without browser CSRF, and a disallowed action is denied before the handler runs.
- Added unit tests for issue/auth/list/revoke, expiry, disabled-agent denial, and bearer parser rejection cases.
- Marked task 17 complete and updated the changelog.

### Why

- API tokens are the simplest programmatic credential family and validate that `AuthResult`, typed grants, agents, and route enforcement fit together.
- Header-only bearer parsing keeps token transport predictable and avoids leaking credentials through query strings, logs, or caches.
- Prefix lookup avoids scanning all token hashes while still requiring a constant-time full hash comparison before authentication succeeds.

### What worked

- The agent service cleanly gates token authentication: disabling the agent immediately makes future token auth fail.
- The grant intersection in the enforcer is small because `GrantSet` already knows how to match action/tenant/resource dimensions.
- Focused package tests and `go test ./...` passed before commit.
- The pre-commit hook passed after fixing named return lint failures.

### What didn't work

- The first commit attempt failed `golangci-lint` because the new token helpers used named returns:
  `pkg/gojahttp/auth/programauth/token.go:297:1: named return "value" with type "string" found (nonamedreturns)`
  `pkg/gojahttp/auth/programauth/token.go:361:1: named return "token" with type "string" found (nonamedreturns)`
- I changed `newRawAPIToken` and `BearerFromHeader` to use unnamed return values and reran focused tests before committing.

### What I learned

- API-token auth becomes straightforward once agents and grants exist; most complexity is around safe token handling and lifecycle redaction.
- The planned-route enforcer is the right place to enforce the intersection of token grants and route action/resource, before app-specific authorization.
- Query-parameter bearer rejection should happen in the shared parser so all future bearer credential families inherit the transport rule.

### What was tricky to build

- The sharpest boundary is between stored token records and list/detail projections. Store records need hashes, but list responses and JavaScript APIs must never expose hashes or raw values. I introduced `APITokenView` so service list/revoke/issue metadata paths are redacted by construction.
- Another tricky detail is prefix lookup. The prefix narrows candidates, but never authenticates by itself; authentication still hashes the raw token and uses constant-time comparison against candidate hashes.
- The enforcer grant check had to run after resource resolution so tenant/resource-bound grants can be evaluated, but before the app authorizer and handler.

### What warrants a second pair of eyes

- Whether the initial `ggpat_<prefix>_<secret>` format is the desired long-term public token format.
- Whether `TokenHash` should be moved behind unexported store-only structs before JavaScript-facing token APIs are added.
- Whether the enforcer should add `WWW-Authenticate` bearer challenge headers for malformed/expired/insufficient-scope cases in this PR or a follow-up.

### What should be done in the future

- Wire generated hostauth services with agent/API-token stores and composite auth.
- Add JavaScript `auth.agents` and `auth.tokens` fluent builders.
- Add SQL-backed token/agent stores before production use.
- Add bearer challenge headers and OAuth-style error codes.

### Code review instructions

- Start with `pkg/gojahttp/auth/programauth/token.go` for token format, hashing, parser, service, and authenticator behavior.
- Review `pkg/gojahttp/auth/programauth/memory_token_store.go` for clone/redaction-sensitive store semantics.
- Review `pkg/gojahttp/auth/programauth/composite.go` for bearer-first/session-fallback ordering.
- Review `pkg/gojahttp/enforcer.go` for grant intersection enforcement.
- Review `pkg/gojahttp/api_token_integration_test.go` for the intended route-level behavior.
- Validate with `go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth` and `go test ./...`.

### Technical details

Key commands and outcomes:

```bash
gofmt -w pkg/gojahttp/auth/programauth/token.go pkg/gojahttp/auth/programauth/memory_token_store.go pkg/gojahttp/auth/programauth/composite.go pkg/gojahttp/auth/programauth/token_test.go pkg/gojahttp/api_token_integration_test.go pkg/gojahttp/enforcer.go
go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

go test ./...
# ok for full repository

git commit -m "programauth: add API token authentication"
# first attempt failed nonamedreturns lint on token.go
gofmt -w pkg/gojahttp/auth/programauth/token.go
go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./modules/express ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/providers/hostauth
# ok for all focused packages

git commit -m "programauth: add API token authentication"
# pre-commit lint/test passed; commit 00a1e866e75c17cb07121463021e03ca1d046725
```

Primary files:

```text
pkg/gojahttp/auth/programauth/token.go
pkg/gojahttp/auth/programauth/memory_token_store.go
pkg/gojahttp/auth/programauth/composite.go
pkg/gojahttp/auth/programauth/token_test.go
pkg/gojahttp/api_token_integration_test.go
pkg/gojahttp/enforcer.go
```
