---
Title: Investigation diary
Ticket: XGOJA-AUTH-KEYCLOAK-MFA
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary for production Keycloak hardening and MFA flow planning."
LastUpdated: 2026-06-12T20:29:31.081833056-04:00
WhatFor: "Use when resuming Keycloak production hardening, OIDC transaction storage, logout, and MFA freshness work."
WhenToUse: "Before changing keycloakauth/sessionauth behavior or the Keycloak host example."
---

# Diary

## Goal

Capture the planning and implementation history for production Keycloak/OIDC hardening and concrete MFA flows that integrate with planned Express route security.

## Step 1: Create Keycloak hardening and MFA ticket

This step created a combined ticket for the second and third next auth work items: hardening the Keycloak/OIDC production path and adding a concrete way to refresh `Session.MFAAt` after MFA completion. These two belong together because MFA freshness is enforced at the session-auth boundary, and production Keycloak deployments influence how MFA is satisfied.

The design keeps the same architectural boundary as the current implementation. Keycloak authenticates identity, `sessionauth` owns app sessions and freshness checks, application/host code owns MFA challenge flows, and Express route declarations only request freshness with `.mfaFresh(...)`.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket for 1. Create a new ticket for 2 and 3.\n\nIs body schema relevant for auth / security?"

**Assistant interpretation:** Create one docmgr ticket for persistent auth stores, create a second ticket for production Keycloak hardening plus MFA flows, and answer whether body/schema validation belongs in the auth/security roadmap.

**Inferred user intent:** Split the next auth roadmap into actionable tickets while clarifying the security relevance of request body validation.

**Commit (code):** N/A — ticket setup only.

### What I did
- Created ticket `XGOJA-AUTH-KEYCLOAK-MFA`.
- Added design document `design/01-keycloak-production-hardening-and-mfa-implementation-plan.md`.
- Added this investigation diary.
- Added phased tasks for production Keycloak settings, durable OIDC transaction storage, logout hardening, MFA session update primitives, MFA example flow, and production smoke/docs.
- Included a design note explaining where body/schema validation fits in the security roadmap.

### Why
- The current Keycloak example is production-shaped but still local/demo in its storage and deployment assumptions.
- `.mfaFresh(...)` is now enforced by `sessionauth`, but there is not yet a host-owned flow for completing MFA and updating `Session.MFAAt`.
- Keycloak hardening and MFA step-up design should be planned together so assurance/freshness semantics are consistent.

### What worked
- The ticket workspace and initial docs were created successfully with docmgr.
- The design ties directly to existing files: `keycloakauth.go`, `sessionauth.go`, and `examples/xgoja/19-express-keycloak-auth-host`.

### What didn't work
- N/A.

### What I learned
- Body/schema validation is security-relevant, but it should not be folded into the immediate Keycloak/MFA ticket. It is broader request validation and authorization safety work.
- MFA freshness is authentication state. The authorizer should receive an actor only after the authenticator has checked route-level freshness requirements.

### What was tricky to build
- The main planning nuance was separating Keycloak-enforced MFA, app-owned step-up MFA, and Keycloak step-up prompt flows. The ticket keeps all three visible while recommending a narrow host-owned `Session.MFAAt` update primitive first.

### What warrants a second pair of eyes
- Whether missing/stale MFA should remain a 401 unauthenticated response or use a richer step-up signal.
- Whether Keycloak end-session support belongs in `keycloakauth` or should remain application-owned.
- Whether MFA completion should always rotate the session ID.

### What should be done in the future
- Implement durable OIDC transaction storage after persistent store decisions are made.
- Add a demo MFA flow that proves route denial before MFA and success after `Session.MFAAt` update.

### Code review instructions
- Start with the design document and tasks in this ticket.
- Review against `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`, `pkg/gojahttp/auth/sessionauth/sessionauth.go`, and `examples/xgoja/19-express-keycloak-auth-host`.

### Technical details
- Primary design doc:
  ```text
  ttmp/2026/06/12/XGOJA-AUTH-KEYCLOAK-MFA--production-keycloak-hardening-and-mfa-flows/design/01-keycloak-production-hardening-and-mfa-implementation-plan.md
  ```
