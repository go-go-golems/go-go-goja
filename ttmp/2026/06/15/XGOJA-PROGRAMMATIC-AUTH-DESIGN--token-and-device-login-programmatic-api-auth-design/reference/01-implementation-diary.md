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
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/design/01-token-and-device-login-programmatic-api-auth-implementation-guide.md
      Note: Primary implementation guide produced by this ticket.
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-PROGRAMMATIC-AUTH-DESIGN--token-and-device-login-programmatic-api-auth-design/sources/01-current-auth-surface.md
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
