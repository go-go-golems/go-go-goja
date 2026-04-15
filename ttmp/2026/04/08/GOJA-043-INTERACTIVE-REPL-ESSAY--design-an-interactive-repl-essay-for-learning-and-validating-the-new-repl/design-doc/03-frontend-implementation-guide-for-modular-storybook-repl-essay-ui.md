---
Title: Frontend implementation guide for modular Storybook REPL essay UI
Ticket: GOJA-043-INTERACTIVE-REPL-ESSAY
Status: active
Topics:
    - repl
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/features/meet-session/MeetSessionPage.tsx
      Note: Section 1 page composition and request-state orchestration
    - Path: web/src/app/api/essayApi.ts
      Note: RTK Query contract for article-scoped session bootstrap/create/snapshot
    - Path: web/src/components/primitives
      Note: Primitive component extraction from imported JSX mock
    - Path: web/src/mocks/handlers.ts
      Note: MSW backend contract emulation for Storybook and isolated frontend development
    - Path: ttmp/2026/04/08/GOJA-043-INTERACTIVE-REPL-ESSAY--design-an-interactive-repl-essay-for-learning-and-validating-the-new-repl/sources/local/repl-essay(1).jsx
      Note: Original UX designer mock used as extraction source
ExternalSources: []
Summary: Implementation guide for the new web frontend foundation that converts the imported mock into tokenized primitives, section components, RTK Query data flow, MSW-backed Storybook states, and phased integration steps.
LastUpdated: 2026-04-14T20:52:00-04:00
WhatFor: Turn the imported UX mock into a production-grade frontend architecture for GOJA-043.
WhenToUse: Use when implementing or reviewing the essay frontend in `web/` and wiring section pages to real API routes.
---

# Frontend implementation guide for modular Storybook REPL essay UI

## Executive Summary

The imported designer mock (`sources/local/repl-essay(1).jsx`) is a strong visual and interaction sketch, but it is a monolithic file with inline styles, mock data, and local state. It is useful as a design artifact, not as an implementation foundation.

This guide defines and applies a frontend architecture in `web/` that converts that sketch into a reusable and testable system:

- tokenized primitive UI components,
- section-level essay components,
- Redux Toolkit + RTK Query data contracts for real backend routes,
- MSW handlers for deterministic Storybook states,
- Storybook coverage for primitives and full page states.

The first delivered page is Section 1 (`MeetSessionPage`) with empty/loading/success/error behavior around the article-scoped backend routes.

## Problem Statement

The previous frontend implementation for Section 1 lived inside `pkg/replessay/handler.go` as an inline HTML/CSS/JS template. That was fast for backend validation, but it does not scale:

- no component boundaries for reuse in later sections,
- no theme/token contract,
- no Storybook surface to iterate with design,
- no Redux/RTK Query shape for growing section state and API usage,
- no MSW-backed deterministic UI states for review and regression checks.

At the same time, the imported mock already includes a rich component vocabulary (button styles, headings, windows/cards, tags, policy rows, JSON inspectors), but all of it is trapped in one file.

## Proposed Solution

Use `web/` as the frontend workspace and establish a layered system.

### 1. Primitives Layer

- `src/components/primitives/*`
- `Button`, `Typography`, `Card`, `Tag`, `Divider`, `CodeBlock`, `JsonViewer`
- tokenized CSS variables + `data-part` hooks

### 2. Feature Layer (Section 1)

- `src/features/meet-session/components/*`
- `SectionIntro`, `SessionSummaryCard`, `PolicyCard`, `SessionJsonPanel`, `SectionShell`
- `src/features/meet-session/MeetSessionPage.tsx` composes the section UI

### 3. State/Data Layer

- `src/app/store.ts` (Redux store)
- `src/app/api/essayApi.ts` (RTK Query API slice)
- `src/features/meet-session/meetSessionSlice.ts` (minimal durable UI state)
- endpoints:
  - `GET /api/essay/sections/meet-a-session`
  - `POST /api/essay/sections/meet-a-session/session`
  - `GET /api/essay/sections/meet-a-session/session/:id`

### 4. Design/Test Layer

- Storybook config in `.storybook/`
- MSW handlers in `src/mocks/handlers.ts`
- page stories with explicit failure states

### 5. Build Layer

- `vite` output: `web/dist/public`
- validated by:
  - `pnpm -C web check`
  - `pnpm -C web build`
  - `pnpm -C web build-storybook`

## Design Decisions

### 1) Keep frontend isolated in `web/`

The app is split away from Go template strings so UI iteration speed (Storybook/Vite) is independent from backend route/template edits.

### 2) Preserve visual intent, replace implementation style

The mock’s retro-editorial visual direction is preserved, but implementation moved from inline style objects into:

- CSS variables in `src/theme/tokens.css`,
- primitive styles in `src/theme/primitives.css`,
- page layout styles in `src/theme/essay.css`.

This creates a stable theming contract while staying faithful to the mock.

### 3) Use RTK Query for the API boundary from day one

Even for one section, introducing `essayApi` early avoids ad-hoc fetch patterns and gives a consistent data lifecycle as additional sections are built.

### 4) Keep Redux slice small and explicit

Only durable page UI state (`activeSessionId`) is in slice state. Request payload/status state remains in RTK Query.

### 5) Treat Storybook as a required development surface

Primitives and page composition both have stories. MSW handlers drive deterministic backend behavior, so review does not require a running Go server.

## Alternatives Considered

### Alternative A: Keep extending inline template in `handler.go`

Rejected because frontend complexity would continue to grow inside Go template strings with poor reuse/testing.

### Alternative B: Port full 9-section mock directly as one React file

Rejected because it preserves the monolith and does not establish a reusable primitive/component system.

### Alternative C: Use local state + direct fetch only

Rejected because section count and API breadth are expected to grow. RTK Query gives stronger request modeling and consistency.

## Implementation Plan

### Phase 0 (completed in this slice)

- Import UX source:
  - `sources/local/repl-essay(1).jsx`
- Scaffold `web/` React+TypeScript workspace with Vite, Storybook, Redux Toolkit, RTK Query, MSW.
- Extract primitives and Section 1 components.
- Implement `MeetSessionPage` with create/snapshot behavior against article routes.
- Add Storybook stories for primitives and page failure states.
- Validate:
  - `pnpm -C web check`
  - `pnpm -C web build`
  - `pnpm -C web build-storybook`

### Phase 1 (next)

- Wire Go essay route to serve compiled `web/dist/public` assets under `/static/...`.
- Replace template-rendered section body with React mount shell while keeping API routes unchanged.
- Add smoke coverage for static asset route and page shell.

### Phase 2

- Port Section 2 and Section 3 from the mock into feature modules.
- Extend `essayApi` for profile-selective create/evaluate flows as backend surfaces become available.
- Add section-specific MSW scenarios and Storybook docs pages.

### Phase 3

- Add global essay navigation and progressive section loading.
- Add visual regression checks for key Storybook stories.

## Open Questions

1. Should the essay frontend mount only from `goja-repl essay`, or also under `goja-repl serve`?
2. Should progression be a single long page or chapter-routed (`/essay/:section`)?
3. For profile comparison sections, should profile override happen at session creation or via dedicated backend routes?
4. Which theming API should be exposed first: CSS variable docs only, or runtime theme presets?

## References

- Imported UX mock:
  - `sources/local/repl-essay(1).jsx`
- New frontend workspace:
  - `web/`
- Existing design context:
  - `design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md`
  - `design-doc/02-ux-handoff-for-the-interactive-repl-essay.md`
