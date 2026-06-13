---
Title: Investigation diary
Ticket: XGOJA-AUTH-STORES
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
Summary: "Chronological diary for production auth persistent store planning."
LastUpdated: 2026-06-12T20:29:30.353238909-04:00
WhatFor: "Use when resuming work on SQL/Postgres-backed stores for sessions, audit, capabilities, and app auth domain data."
WhenToUse: "Before implementing or reviewing persistent auth store work."
---

# Diary

## Goal

Capture the planning and implementation history for production persistent stores backing the gojahttp host-auth packages.

## Step 1: Create persistent auth store ticket

This step created a focused ticket for the first next auth work item: replacing in-memory host-auth stores with durable production stores. The ticket intentionally keeps storage behind existing Go interfaces so Express route declarations remain unchanged.

The ticket separates persistent storage from Keycloak/MFA hardening. That keeps the first implementation path concrete: sessions, audit, capabilities, and app-owned auth data need durable stores before the production Keycloak example can become a realistic deployment reference.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket for 1. Create a new ticket for 2 and 3.\n\nIs body schema relevant for auth / security?"

**Assistant interpretation:** Create one docmgr ticket for persistent auth stores, create a second ticket for production Keycloak hardening plus MFA flows, and answer whether body/schema validation belongs in the auth/security roadmap.

**Inferred user intent:** Split the next auth roadmap into actionable tickets while clarifying the security relevance of request body validation.

**Commit (code):** N/A — ticket setup only.

### What I did
- Created ticket `XGOJA-AUTH-STORES`.
- Added design document `design/01-persistent-auth-store-implementation-plan.md`.
- Added this investigation diary.
- Added phased tasks for store contracts, session SQL store, audit SQL store, capability SQL store, appauth SQL store, and example integration.

### Why
- The current host-auth examples use in-memory stores by design.
- Production deployments need durable session, audit, capability, user, tenant, membership, and resource state.
- Store work is a separable foundation for later Keycloak and MFA hardening.

### What worked
- The ticket workspace and initial docs were created successfully with docmgr.
- The design maps directly to existing package seams: `sessionauth.Store`, `audit.Store`, `capability.Store`, and appauth store interfaces.

### What didn't work
- N/A.

### What I learned
- The persistent-store work is best scoped as interface-backed adapters, not as changes to Express or JavaScript route planning.
- Capability token persistence has the strongest transactional requirement because single-use redemption must be atomic.

### What was tricky to build
- The main planning nuance was deciding whether appauth persistence should be generic or app-specific. The ticket currently proposes a starter store while preserving replaceable domain interfaces.

### What warrants a second pair of eyes
- Whether Postgres should be the default implementation target or whether the repo wants a different persistence abstraction.
- Whether appauth SQL should live in this repo or only in examples.
- Whether migrations should be embedded SQL files or Go-rendered DDL.

### What should be done in the future
- Implement store contract tests before writing SQL adapters.
- Decide on concrete migration tooling and test database strategy.

### Code review instructions
- Start with the design document and tasks in this ticket.
- Compare proposed stores against `pkg/gojahttp/auth/sessionauth`, `audit`, `capability`, and `appauth` interfaces.

### Technical details
- Primary design doc:
  ```text
  ttmp/2026/06/12/XGOJA-AUTH-STORES--production-host-auth-persistent-stores/design/01-persistent-auth-store-implementation-plan.md
  ```
