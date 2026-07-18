---
Title: Implementation Diary
Ticket: XGOJA-CLIENT-FETCH-AUTH-DESIGN
Status: active
Topics:
    - xgoja
    - fetch
    - auth
    - javascript-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/22-programmatic-agent-auth/scripts/smoke.sh
      Note: Step 2 validation script for no-curl programmatic agent auth
    - Path: modules/fetch/fetch.go
      Note: Step 2 implementation file for guarded low-level fetch
    - Path: pkg/xgoja/providers/host/host.go
      Note: |-
        Key evidence for guarded fetch module placement
        Step 2 integration point for guarded fetch registration
    - Path: pkg/xgoja/providers/hostauth/programmatic.go
      Note: Key evidence for Go-owned credential/auth builder style
    - Path: ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/design/01-client-side-fetch-and-authenticated-api-client-implementation-guide.md
      Note: Primary design deliverable created in Step 1
ExternalSources: []
Summary: Chronological diary for the client-side fetch and authenticated API client design ticket.
LastUpdated: 2026-06-20T16:25:00-04:00
WhatFor: Use to understand what was investigated, why the fetch/auth design exists, and how to continue implementation.
WhenToUse: Read before implementing the fetch module, client auth builders, or programmatic agent smoke examples.
---



# Diary

## Goal

Capture the investigation and design work for a first-class xgoja client-side fetch API with framework-native authentication support.

## Step 1: Create the client-side fetch/auth design ticket and write the intern implementation guide

This step created a new ticket specifically for the client side of programmatic auth. The key decision was to avoid a curl-backed example entirely and instead design a proper guarded fetch module plus a fluent authenticated client API that matches the rest of the framework.

The resulting design is intentionally detailed for a new intern. It explains the relevant host provider, async module, jsverbs, Express route builder, hostauth, programauth, and generated-example pieces before proposing concrete APIs, pseudocode, implementation phases, and smoke tests.

### Prompt Context

**User prompt (verbatim):** "Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new ticket deliverable with a comprehensive design and implementation guide for a client-side fetch/auth API, then upload the document bundle to reMarkable.

**Inferred user intent:** Make the next implementation slice teachable and reviewable before coding, especially because canonical examples should not rely on `exec curl`.

**Commit (code):** N/A — design/documentation only.

### What I did

- Removed the untracked exploratory `examples/xgoja/22-programmatic-agent-auth` skeleton that used `exec.run("curl", ...)`.
- Created ticket `XGOJA-CLIENT-FETCH-AUTH-DESIGN`.
- Added a design document: `design/01-client-side-fetch-and-authenticated-api-client-implementation-guide.md`.
- Added this implementation diary.
- Updated ticket tasks with design completion and implementation phases.
- Gathered file-backed evidence from host provider modules, filesystem async APIs, Express builder stores, hostauth programmatic builders, API-token auth, route auth requirements, jsverbs Promise support, and generated host examples.
- Uploaded the design bundle to reMarkable at `/ai/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN`.

### Why

- The server side can now provision agents/tokens and protect routes with agent auth, but JavaScript agents need a framework-native outbound HTTP client.
- `exec curl` is a poor canonical example because it bypasses Go-owned request policy, redaction, and testability.
- A design-first ticket gives the implementation clear API boundaries, file-level starting points, and acceptance criteria.

### What worked

- Existing code already has strong patterns to copy: guarded host modules, Promise-returning async module methods, Go-owned fluent builders, and redacted auth projections.
- The design can reuse the current jsverbs Promise support for async agent verbs.
- The programauth API-token output shape gives a concrete first credential source for `fetch.auth.bearer().fromFile(...).jsonPath("token.value")`.

### What didn't work

- The exploratory example initially used `exec`/`curl`; this was explicitly rejected and removed before documenting or committing it.
- No implementation was attempted in this step because the user asked for a detailed guide first.

### What I learned

- The host provider is the right place for fetch because outbound network access is a host capability like filesystem, process execution, and database access.
- The low-level browser-like `fetch` API and high-level fluent authenticated client solve different problems and should both exist.
- Credential sources should follow the same Go-owned builder-store pattern as Express auth specs and hostauth grant builders.

### What was tricky to build

- The main design tension is generality versus framework opinion. A browser-like `fetch` is useful, but it does not by itself encode our security principles. The guide resolves this by proposing both `fetch.fetch(...)` and `fetch.client()`.
- Another tricky point is credential file access. Reading token files inside the fetch module keeps raw tokens out of user variables, but it expands module capability. The guide recommends gating file credential sources separately with `credentials.allowFiles` and optional `allowedFiles`.
- The API needs to avoid over-fitting to API tokens while still making API tokens excellent. The guide models bearer credential sources first and leaves device/refresh credentials as future builders.

### What warrants a second pair of eyes

- Whether the module should be named `fetch`, `http`, or `httpClient`; the guide recommends `fetch` with an opinionated `.client()` API.
- Whether `fetch.auth.bearer().fromFile(...)` should be included in the first implementation or deferred in favor of `fromEnv(...)` and `token(...)`.
- Whether fluent client non-2xx responses should throw by default in `expectJson()` / `expectText()` mode.

### What should be done in the future

- Implement Phase 1 through Phase 4 from the guide.
- Add the generated server+agent example only after the fetch/client auth APIs exist.
- Cross-link the completed implementation from the programmatic auth design ticket.

### Code review instructions

- Start with the design document, especially `Current-state architecture and evidence`, `Public API proposal`, and `Implementation plan`.
- Then inspect the referenced source files in the order listed under `New-intern review path`.
- Validate documentation hygiene with `docmgr doctor --ticket XGOJA-CLIENT-FETCH-AUTH-DESIGN --stale-after 30`.

### Technical details

Key commands and actions:

```bash
rm -rf examples/xgoja/22-programmatic-agent-auth

docmgr ticket create-ticket \
  --ticket XGOJA-CLIENT-FETCH-AUTH-DESIGN \
  --title "Client-side fetch and authenticated API client design" \
  --topics xgoja,fetch,auth,javascript-api

docmgr doc add --ticket XGOJA-CLIENT-FETCH-AUTH-DESIGN \
  --doc-type design \
  --title "Client-side Fetch and Authenticated API Client Implementation Guide"

docmgr doc add --ticket XGOJA-CLIENT-FETCH-AUTH-DESIGN \
  --doc-type reference \
  --title "Implementation Diary"

remarquee upload bundle \
  ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/design/01-client-side-fetch-and-authenticated-api-client-implementation-guide.md \
  ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/reference/01-implementation-diary.md \
  --name "XGOJA Client Fetch Auth Design" \
  --remote-dir "/ai/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN" \
  --toc-depth 2 \
  --non-interactive
# OK: uploaded XGOJA Client Fetch Auth Design.pdf -> /ai/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN
```

Primary files:

```text
ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/design/01-client-side-fetch-and-authenticated-api-client-implementation-guide.md
ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/reference/01-implementation-diary.md
ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/tasks.md
```

## Step 2: Implement guarded fetch, authenticated client builders, and the agent smoke example

This step turned the design into working code. The implementation adds a guarded host `fetch` module, a low-level Promise-returning HTTP API, a fluent `fetch.client()` builder, and Go-owned bearer credential sources so JavaScript agents can call authenticated routes without shelling out to `curl`.

It also adds a generated server+agent smoke example. The server provisions an automation agent and bootstrap API token, protects one route with `express.agent()`, protects another with `express.sessionUser()`, and the separate agent binary reads the token through `fetch.auth.bearer().fromFile(...).jsonPath(...)` before making the request.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the split implementation turn: finish the fetch/client-auth implementation, validate the generated example, and update ticket bookkeeping.

**Inferred user intent:** Complete the client-side fetch/auth work so programmatic auth has a canonical no-`exec` JavaScript agent example.

**Commit (code):** `c2cd764` — "modules: add guarded fetch client"; `5aa18ec` — "examples: add programmatic agent fetch auth demo"

### What I did

- Added `modules/fetch` with guarded outbound HTTP policy, Promise-returning `fetch.fetch(...)`, response helpers, fluent client builders, bearer credential source builders, TypeScript declarations, and tests.
- Registered the `fetch` host module in `pkg/xgoja/providers/host/host.go` with explicit `allow: true`, allowed origins, timeouts, response-size limits, and credential-source permissions.
- Added `examples/xgoja/22-programmatic-agent-auth` with separate generated server and agent specs.
- Added `scripts/smoke.sh` to build and run the no-curl smoke path: unauthenticated agent route returns `401`, the agent client succeeds, and the same token is rejected by the session-only route with `403`.
- Updated the design guide with an implementation status section and updated ticket tasks/changelog.

### Why

- A programmatic auth stack needs both a server-side route/auth model and a client-side way for agents to call those routes.
- A dedicated fetch module is narrower and more auditable than `exec curl`, and it lets Go own network policy, credential source policy, and redaction.
- Separate server and agent generated binaries keep server-only hostauth services out of the agent runtime.

### What worked

- The existing `runtimebridge`/Promise pattern fit the blocking HTTP client work.
- The host provider guard pattern fit outbound HTTP cleanly: fetch is disabled unless explicitly configured.
- The final generated smoke passed with `make -C examples/xgoja/22-programmatic-agent-auth smoke`.
- Targeted tests and the full suite passed during implementation:
  - `go test ./modules/fetch ./pkg/xgoja/providers/host`
  - `go test ./...`

### What didn't work

- The first smoke attempt passed `--http-listen` at the wrong command level:

```text
Error: unknown flag: --http-listen
unknown flag: --http-listen
```

- The agent verb initially used a hyphenated function name that jsverbs rejected:

```text
Error: scan jsverb source local-verbs: agent.js references unknown function "call-report"
scan jsverb source local-verbs: agent.js references unknown function "call-report"
```

- The agent verb initially declared unsupported `output: "json"`:

```text
Error: agent.js#callReport has unsupported output mode "json"
agent.js#callReport has unsupported output mode "json"
```

- A single generated binary that included both `hostauth` and agent `fetch` modules failed because the agent runtime tried to instantiate `auth` without hostauth services:

```text
Error: register module "xgoja:go-go-goja-hostauth.auth:auth": create module go-go-goja-hostauth.auth: auth module requires hostauth services
```

### What I learned

- jsverbs command names and JavaScript function identifiers need to stay compatible; a CLI command can be user-facing, but the underlying function reference cannot use a hyphenated identifier.
- Current jsverbs output modes are narrower than arbitrary JSON mode; returning JSON text is the compatible path for this example.
- Server and agent specs are cleaner as separate generated hosts because they need different host capabilities and service wiring.

### What was tricky to build

- The hardest edge was keeping credentials out of ordinary JavaScript object bags while still making the API ergonomic. The solution was to make `fetch.auth.bearer()` a Go-owned builder and reject plain object auth specs for sensitive credential input.
- Another tricky part was origin matching. The module needed to accept local generated examples such as `http://127.0.0.1:*` without silently opening all outbound HTTP.
- The example wiring exposed a service-lifetime boundary: hostauth services exist for the generated HTTP server runtime, not automatically for a separate jsverbs-only agent command. Splitting the xgoja specs resolved that boundary explicitly.

### What warrants a second pair of eyes

- Review `modules/fetch/config.go` and `pkg/xgoja/providers/host/host.go` for policy semantics, especially wildcard origin matching and credential file permissions.
- Review `modules/fetch/auth_builder.go` for redaction guarantees and whether raw bearer values can appear in returned errors.
- Review `examples/xgoja/22-programmatic-agent-auth` for whether the two-binary generated example is the right canonical shape.

### What should be done in the future

- Consider re-uploading the updated implementation guide and diary bundle to reMarkable if this ticket needs a post-implementation PDF snapshot.
- Add future credential builders for device/refresh-token flows after server-side refresh token families exist.
- Consider streaming response support only after the bounded-buffer v1 is stable.

### Code review instructions

- Start with `modules/fetch/config.go`, then `modules/fetch/fetch.go`, `modules/fetch/client_builder.go`, and `modules/fetch/auth_builder.go`.
- Check the integration point in `pkg/xgoja/providers/host/host.go`.
- Run:

```bash
go test ./modules/fetch ./pkg/xgoja/providers/host
go test ./...
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

### Technical details

Implemented and validated commands included:

```bash
GOWORK=off go run ./cmd/xgoja doctor -f examples/xgoja/22-programmatic-agent-auth/xgoja.yaml
GOWORK=off go run ./cmd/xgoja doctor -f examples/xgoja/22-programmatic-agent-auth/agent.xgoja.yaml
make -C examples/xgoja/22-programmatic-agent-auth smoke
docmgr task check --ticket XGOJA-CLIENT-FETCH-AUTH-DESIGN --id 6,7,8,9,10
```

Primary implementation files:

```text
modules/fetch/config.go
modules/fetch/fetch.go
modules/fetch/response.go
modules/fetch/client_builder.go
modules/fetch/auth_builder.go
modules/fetch/typescript.go
pkg/xgoja/providers/host/host.go
examples/xgoja/22-programmatic-agent-auth/xgoja.yaml
examples/xgoja/22-programmatic-agent-auth/agent.xgoja.yaml
examples/xgoja/22-programmatic-agent-auth/verbs/server.js
examples/xgoja/22-programmatic-agent-auth/verbs/agent.js
examples/xgoja/22-programmatic-agent-auth/scripts/smoke.sh
```
