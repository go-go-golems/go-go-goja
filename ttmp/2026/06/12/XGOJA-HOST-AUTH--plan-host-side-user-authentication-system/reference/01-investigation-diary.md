---
Title: Investigation diary
Ticket: XGOJA-HOST-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
    - keycloak
    - oidc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md
      Note: Wrapped-up Obsidian project report before opening this follow-up ticket.
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/design-doc/01-host-side-user-auth-system-implementation-plan.md
      Note: Primary design plan for production and dev/demo host auth packages.
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/sources/01-keycloak-oidc-session-authz-host-notes.md
      Note: Imported /tmp/auth2.md source material.
ExternalSources:
    - ../sources/01-keycloak-oidc-session-authz-host-notes.md
Summary: Chronological diary for planning the host-side user/auth system after Express planned auth cutover.
LastUpdated: 2026-06-12T17:00:00-04:00
WhatFor: Use this to resume or review the host-side auth system planning ticket.
WhenToUse: Read before implementing dev/demo auth, session auth, Keycloak/OIDC, app authorization, audit, or capabilities.
---

# Diary

## Goal

This diary records the creation of `XGOJA-HOST-AUTH`, a follow-up ticket for the actual host-side user/auth system behind the Express planned-route framework.

## Step 1: Wrap up Express auth article and create host-auth planning ticket

I first updated the Obsidian vault article for the completed Express planned-auth cutover, because the previous report still described the earlier MVP state. The wrapped-up article now records the hard verb-helper cutover, strict raw-route rejection, CSRF/audit support, runnable host example, remaining generated-build caveat, and the boundary between Express route planning and future host-side auth packages.

After that, I created a new docmgr ticket for the host-side user/auth system. The new plan separates an opinionated production stack from a simple dev/demo stack while keeping both behind the existing `gojahttp.AuthOptions` interfaces.

### Prompt Context

**User prompt (verbatim):** "ok, let's make a new ticket and plan out the implementation of the actual host side user/auth system. While I do want a production ready yet opinionated (on tech choices) setup, we should also have a simple version for dev/demo purposes. After creating the new ticket, give me an update on what that would lok like.

But before you do any of that, let's update and wrap up the project report we wrote in the obsidian vault, as we have finished implementing the express cutover. /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md

Only then continue with the new ticket."

**Assistant interpretation:** First update the existing Obsidian article to reflect the completed Express cutover, then create a new docmgr ticket for planning the concrete host-side auth/user system with both production and dev/demo implementations.

**Inferred user intent:** Close out the Express route-planning phase cleanly and start a separate implementation track for real authentication, sessions, users, authorization, and demo tooling.

**Commit (code):** N/A — documentation/ticket planning only in this step.

### What I did
- Rewrote the Obsidian article at `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md` to reflect the final Express planned-auth implementation state.
- Committed the Obsidian article update in the vault as `7defced` — `Wrap up go-go-goja Express auth article`.
- Created docmgr ticket `XGOJA-HOST-AUTH` with title `Plan host-side user authentication system` and topics `goja,http,security,xgoja,keycloak,oidc`.
- Imported `/tmp/auth2.md` into the new ticket as `sources/01-keycloak-oidc-session-authz-host-notes.md`.
- Created the primary design document `design-doc/01-host-side-user-auth-system-implementation-plan.md`.
- Wrote the design around two tracks:
  - production: Keycloak/OIDC, server-side sessions, app user normalization, app-owned authorization, audit, capabilities;
  - dev/demo: in-memory users/sessions/resources/authz/CSRF/audit with login/logout smoke support.
- Added and checked ticket task 1 for importing source material.

### Why
- The Express planned-auth work is now a framework boundary, not the entire auth product. A separate ticket keeps production identity/session/user-store work from being conflated with the Express route API.
- `/tmp/auth2.md` specifically argues for Keycloak/OIDC, app sessions, app-owned authorization, and capabilities. Those concerns need their own plan because they carry a much larger security and maintenance burden than route builders.
- A dev/demo implementation is needed so examples and tests remain runnable without Keycloak, Redis, Postgres, or browser redirects.

### What worked
- `docmgr ticket create-ticket` created the new workspace successfully.
- The imported source validated under `docmgr doctor` after frontmatter was added.
- The new design document was able to reuse the existing `gojahttp` interfaces instead of proposing a new enforcement boundary.
- The Obsidian report now matches the current state of the branch instead of the earlier MVP state.

### What didn't work
- No command failures occurred in this planning step.
- No Go tests were run because no production Go code was changed.

### What I learned
- The host-side work should be split into optional packages rather than moved into `modules/express`.
- The likely package boundary is `pkg/gojahttp/auth/...`, with packages such as `sessionauth`, `keycloakauth`, `appauth`, `capability`, `audit`, and `devauth`.
- The dev/demo stack should model the same interfaces as production, not bypass them, so it remains useful as onboarding material and smoke coverage.

### What was tricky to build
- The main tricky part was deciding how much opinion to put into the host layer without turning `go-go-goja` into a full application framework. The design resolves this by making Keycloak/session/authz packages optional and host-side, while keeping `modules/express` focused on route declarations.
- Another subtlety was the production/dev split. If the dev system is too different from production, examples teach the wrong model. If it is too production-like, it becomes hard to run. The proposed `devauth` package uses in-memory stores but implements the same `gojahttp` interfaces.

### What warrants a second pair of eyes
- Review whether the proposed package names and boundaries are right before implementation begins.
- Review whether `sessionauth` should wrap `alexedwards/scs` directly or expose a smaller store interface with optional adapters.
- Review whether `keycloakauth` belongs in this repo long-term or should eventually become a separate module once stabilized.

### What should be done in the future
- Implement Phase 1 first: extract a reusable `devauth` package from the current runnable example and add login/logout/session-cookie smoke coverage.
- Then implement `sessionauth` as the reusable base for production and demo session-backed authentication.
- Keep Keycloak/OIDC production work behind optional package imports.

### Code review instructions
- Start with `ttmp/2026/06/12/XGOJA-HOST-AUTH--plan-host-side-user-authentication-system/design-doc/01-host-side-user-auth-system-implementation-plan.md`.
- Compare the proposed package boundaries against:
  - `pkg/gojahttp/auth_plan.go`,
  - `pkg/gojahttp/planned_dispatch.go`,
  - `examples/xgoja/16-express-auth-host/cmd/host/main.go`.
- Validate ticket hygiene with:
  - `docmgr doctor --ticket XGOJA-HOST-AUTH --stale-after 30`.

### Technical details
- Proposed optional host-auth package layout:
  ```text
  pkg/gojahttp/auth/sessionauth
  pkg/gojahttp/auth/keycloakauth
  pkg/gojahttp/auth/appauth
  pkg/gojahttp/auth/capability
  pkg/gojahttp/auth/audit
  pkg/gojahttp/auth/devauth
  ```
- Proposed first implementation target:
  ```text
  devauth: in-memory users, sessions, resources, authorization, CSRF, audit, login/logout handlers
  ```
- Proposed production foundation:
  ```text
  Keycloak OIDC Authorization Code + PKCE -> app user normalization -> server-side session -> gojahttp Authenticator/CSRFProtector -> ResourceResolver -> Authorizer -> AuditSink
  ```
