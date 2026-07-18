---
Title: Production tiny-idp integration and durable xgoja OIDC host auth
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - xgoja
    - auth
    - oidc
    - security
    - http
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation plan for a production-grade tiny-idp OIDC integration in generated xgoja hosts, including durable login transactions, production validation, application-owned device login, and a future native-token resource-server boundary."
LastUpdated: 2026-07-15T15:10:02.590001737-04:00
WhatFor: ""
WhenToUse: ""
---

# Production tiny-idp integration and durable xgoja OIDC host auth

## Overview

This ticket turns the OIDC host support introduced for the personal-inbox tutorial into a design that can support a real small application. The immediate objective is not to replace tiny-idp or to make every application speak OAuth directly. It is to give a generated xgoja host a correct, durable relying-party boundary when tiny-idp is the external OpenID Provider.

The ticket is deliberately split into two product surfaces. The first is the implementation target: browser sign-in through tiny-idp, an application-owned session, planned Express routes, and application-owned device credentials. The second is a separately scoped future capability: accepting tiny-idp-issued device credentials at an xgoja API as a formal OAuth resource server. These surfaces must not be merged accidentally because they have different token semantics, revocation contracts, and ownership rules.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **Phase 0 and Phase 1 implemented and validated; Phase 2 production-profile work remains**

## Topics

- xgoja
- auth
- oidc
- security
- http
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts

## Primary Documents

- [Design and implementation guide](./design-doc/01-production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth-design-and-implementation-guide.md) is the intern-facing technical design, API reference, phase plan, and test strategy.
- [Diary](./reference/01-diary.md) records the evidence used to form the design and provides the continuation point for implementation work.
