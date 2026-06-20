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
    - Path: pkg/xgoja/providers/host/host.go
      Note: Key evidence for guarded fetch module placement
    - Path: pkg/xgoja/providers/hostauth/programmatic.go
      Note: Key evidence for Go-owned credential/auth builder style
    - Path: ttmp/2026/06/20/XGOJA-CLIENT-FETCH-AUTH-DESIGN--client-side-fetch-and-authenticated-api-client-design/design/01-client-side-fetch-and-authenticated-api-client-implementation-guide.md
      Note: Primary design deliverable created in Step 1
ExternalSources: []
Summary: Chronological diary for the client-side fetch and authenticated API client design ticket.
LastUpdated: 2026-06-20T11:45:00-04:00
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
