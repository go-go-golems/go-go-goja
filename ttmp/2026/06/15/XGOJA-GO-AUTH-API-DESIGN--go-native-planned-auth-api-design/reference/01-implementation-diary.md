---
Title: Implementation Diary
Ticket: XGOJA-GO-AUTH-API-DESIGN
Status: active
Topics:
    - goja
    - xgoja
    - auth
    - security
    - architecture
    - rest-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/design/01-go-native-planned-auth-api-intern-implementation-guide.md
      Note: Main intern-oriented implementation guide.
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/sources/01-current-gojahttp-auth-surface.md
      Note: Current code evidence used by the guide.
ExternalSources: []
Summary: Diary for creating the Go-native planned auth API design ticket and uploading it to reMarkable.
LastUpdated: 2026-06-15T22:18:00-04:00
WhatFor: Preserve context for the Go-side planned auth API design work.
WhenToUse: Read before implementing SecureContext, RegisterPlannedHTTP, Go fluent route builders, middleware, or hostauth.AuthKit.
---

# Diary

## Goal

Capture the creation of a new ticket and intern-oriented guide for making the planned auth framework usable from traditional Go hosts, not only JavaScript Goja Express routes.

## Step 1: Create the Go-native planned auth API guide

This step turns the design discussion into a concrete implementation guide. The guide explains the current route-plan/auth system, identifies the coupling to Goja handlers, and proposes a Go-native API built around `SecureContext`, `RegisterPlannedHTTP`, a Go fluent builder, standard `net/http` middleware, and a reusable `hostauth.AuthKit`.

The intended reader is a new intern. The document therefore emphasizes concepts, current file references, sequence diagrams, pseudocode, staged implementation phases, and tests rather than only listing API signatures.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new ticket and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
 for building this go side API."

**Assistant interpretation:** Create a separate docmgr ticket, write a clear intern-oriented technical implementation guide for building a Go-side planned auth API, include diagrams/pseudocode/API and file references, store the guide in the ticket, and upload it to reMarkable.

**Inferred user intent:** Provide an implementation-ready onboarding document so an intern can understand the current system and safely add Go-native APIs that reuse the existing planned auth framework.

**Commit (code):** N/A — ticket documentation only.

### What I did

- Created ticket `XGOJA-GO-AUTH-API-DESIGN`.
- Added design doc `design/01-go-native-planned-auth-api-intern-implementation-guide.md`.
- Added diary doc `reference/01-implementation-diary.md`.
- Added evidence doc `sources/01-current-gojahttp-auth-surface.md` with line anchors from current product code.
- Wrote the implementation guide in an explanatory style with prose paragraphs, diagrams, tables, API sketches, pseudocode, phased tasks, and tests.

### Why

- Traditional Go programs can already assemble `gojahttp.HostOptions.Auth`, but they do not yet get the same ergonomic planned-route enforcement path as JavaScript routes.
- The central design is to make planned auth a `gojahttp` framework feature and make JavaScript Express just one frontend to `RoutePlan`.

### What worked

- The current code already has the important reusable pieces: `RoutePlan`, `AuthOptions`, `sessionauth`, `appauth`, `audit`, `hostauth`, and `gojahttp.Host`.
- The design can be implemented incrementally by extracting `SecureContext` before adding new public APIs.

### What didn't work

- N/A.

### What I learned

- The key missing abstraction is a reusable secure context/enforcer that is independent of JavaScript object construction.
- `hostauth` should be repositioned as a general Go auth kit while preserving generated-host service factory behavior.

### What was tricky to build

- The guide had to preserve the existing JavaScript API and explain that Go and JS need identical semantics, not identical syntax.
- The route registry change needs special care because it currently stores a single `goja.Callable` handler shape.

### What warrants a second pair of eyes

- Whether the route registry should use separate handler fields, an interface, or separate route tables.
- Whether `SecureContext` should be exported immediately or introduced as internal first.
- Whether `hostauth.AuthKit` belongs in `pkg/xgoja/hostauth` or a more neutral package later.

### What should be done in the future

- Implement the guide in phases:
  1. Extract `SecureContext`.
  2. Add `RegisterPlannedHTTP`.
  3. Add Go fluent builder.
  4. Add planned middleware.
  5. Add `hostauth.NewKit`.
  6. Add examples/docs.

### Code review instructions

- Start with `pkg/gojahttp/planned_dispatch.go` and verify the extraction preserves current JS planned-route behavior.
- Review route registry changes carefully so raw Goja routes, planned Goja routes, and planned Go routes dispatch predictably.
- Validate with tests for public, authenticated, CSRF-denied, resource-denied, authorization-denied, handler-failed, and mixed Go/JS hosts.

### Technical details

Primary files created:

```text
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/design/01-go-native-planned-auth-api-intern-implementation-guide.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/reference/01-implementation-diary.md
ttmp/2026/06/15/XGOJA-GO-AUTH-API-DESIGN--go-native-planned-auth-api-design/sources/01-current-gojahttp-auth-surface.md
```
