---
Title: Investigation diary
Ticket: XGOJA-AUTH-PROD-DOCS
Status: active
Topics:
    - goja
    - http
    - security
    - keycloak
    - oidc
    - architecture
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary for production auth deployment docs, policy adapter planning, and process cleanup."
LastUpdated: 2026-06-12T20:52:00-04:00
WhatFor: "Use when resuming production deployment guide or policy-adapter planning work."
WhenToUse: "Before writing production planned-auth docs or evaluating Casbin/OpenFGA/OPA adapters."
---

# Diary

## Goal

Capture the planning and implementation history for production planned-auth deployment documentation, optional policy adapter exploration, and docs/process cleanup.

## Step 1: Create production docs and policy adapter ticket

This step created the third follow-up ticket for the auth roadmap. The first follow-up ticket covers persistent stores; the second covers production Keycloak hardening and MFA flows; this ticket covers production deployment documentation, optional policy adapter evaluation, and process cleanup discovered while merging the xgoja/v2 example changes.

The important boundary is that policy adapters are not the next implementation task. They should remain optional and later, after app-owned users/tenants/memberships/resources and audit/session/capability stores have durable implementations that can feed a policy engine cleanly.

### Prompt Context

**User prompt (verbatim):** "yeah add a ticket, then work on the auth stores ticket. keep a detailed diary, commit at appropriate intervals."

**Assistant interpretation:** Create a docmgr ticket for auth roadmap item 4 and later, then begin implementation on `XGOJA-AUTH-STORES`, keeping diary entries and committing coherent chunks.

**Inferred user intent:** Preserve the remaining roadmap items as searchable ticket work while shifting active implementation focus to persistent auth stores.

**Commit (code):** N/A — ticket setup only.

### What I did
- Created ticket `XGOJA-AUTH-PROD-DOCS`.
- Added design document `design/01-production-auth-deployment-docs-and-policy-adapter-plan.md`.
- Added this investigation diary.
- Added tasks for production deployment docs, policy adapter evaluation, and docs/process cleanup.

### Why
- Production deployment guidance, policy adapters, and docs/process cleanup are useful but should not distract from persistent stores and Keycloak/MFA implementation.
- Capturing them in a ticket keeps the roadmap explicit and reviewable.

### What worked
- The ticket workspace was created successfully.
- The design doc captures dependencies on `XGOJA-AUTH-STORES` and `XGOJA-AUTH-KEYCLOAK-MFA`.

### What didn't work
- N/A.

### What I learned
- The production deployment guide should be checklist-oriented and operator-focused.
- Policy adapters should be framed as optional `gojahttp.Authorizer` implementations, not as core framework dependencies.

### What was tricky to build
- The tricky planning point was combining item 4+ without over-scoping implementation. The design now makes the production guide the concrete deliverable and keeps policy adapters as evaluation work gated on persistent stores.

### What warrants a second pair of eyes
- Whether policy adapter planning should remain in this ticket or be split into separate tickets later.
- Whether the production guide should be a Glazed help page, an example README, or both.

### What should be done in the future
- Write the production deployment guide after persistent stores and Keycloak/MFA flows are concrete enough to document accurately.

### Code review instructions
- Review the design doc for scope boundaries and dependencies.
- Confirm that policy adapters are documented as optional future work rather than immediate implementation requirements.

### Technical details
- Primary design doc:
  ```text
  ttmp/2026/06/12/XGOJA-AUTH-PROD-DOCS--production-auth-deployment-guide-and-policy-adapter-planning/design/01-production-auth-deployment-docs-and-policy-adapter-plan.md
  ```
