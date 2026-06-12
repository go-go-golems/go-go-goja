---
Title: Add proper authentication to Express HTTP module
Ticket: XGOJA-EXPRESS-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket for designing Go-owned authentication and authorization options for the go-go-goja Express HTTP module, including staged route plans and Express-style middleware stacks."
LastUpdated: 2026-06-12T15:05:00-04:00
WhatFor: "Use this ticket to track design and implementation of proper auth for modules/express and pkg/gojahttp."
WhenToUse: "Read when planning or reviewing Express HTTP auth, planned route metadata, or host-owned authorization services."
---

# Add proper authentication to Express HTTP module

## Overview

This ticket captures design options for adding proper authentication and authorization to the `go-go-goja` Express HTTP module. The current module supports lightweight raw route registration with `app.get`, `app.post`, and related helpers. The ticket now contains two complementary approaches: a staged planned-route API where JavaScript declares security intent directly, and an Express-style middleware/router API where Go-owned security middleware participates in a familiar `app.use(...); router.patch(..., middleware, handler)` pipeline.

The preliminary API exploration from `/tmp/auth.md` was imported into `sources/01-auth-preliminary-api-ideas.md` and reconciled into intern-oriented implementation guides.

## Key Links

- [MVP authentication API design and implementation guide](./design/01-mvp-authentication-api-design-and-implementation-guide.md)
- [Express-style middleware auth design and implementation guide](./design/02-express-style-middleware-auth-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Imported preliminary API ideas](./sources/01-auth-preliminary-api-ideas.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

## Topics

- goja
- http
- security
- xgoja

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
