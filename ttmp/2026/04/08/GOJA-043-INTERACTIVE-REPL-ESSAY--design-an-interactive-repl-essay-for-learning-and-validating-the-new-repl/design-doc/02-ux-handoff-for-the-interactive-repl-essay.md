---
Title: UX handoff for the interactive REPL essay
Ticket: GOJA-043-INTERACTIVE-REPL-ESSAY
Status: active
Topics:
    - repl
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/replapi/app.go
      Note: Handoff references the app-level session lifecycle and evaluation entrypoints
    - Path: pkg/replhttp/handler.go
      Note: Current route surface and known HTTP gaps for the article
    - Path: pkg/replsession/policy.go
      Note: Policy card and profile behavior data come from these policy types
    - Path: pkg/replsession/types.go
      Note: Primary section-level data shapes come from these response types
    - Path: ttmp/2026/04/08/GOJA-043-INTERACTIVE-REPL-ESSAY--design-an-interactive-repl-essay-for-learning-and-validating-the-new-repl/design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md
      Note: UX handoff derives its section plan from the primary design guide
ExternalSources: []
Summary: UX handoff for the interactive REPL essay, including section-level visual guidance and backend data shape for each planned section.
LastUpdated: 2026-04-14T19:37:12.994113246-04:00
WhatFor: Help a UX designer design the interactive article ahead of implementation.
WhenToUse: Use when designing the visual/interaction layer for the interactive REPL essay.
---


# UX handoff for the interactive REPL essay

## Executive Summary

This document is for the UX designer who will shape the interactive REPL essay before the engineering work is fully implemented. The essay should feel like a live technical explainer: part article, part instrument panel, part truth surface for the backend. It should not look like a CRUD admin dashboard and it should not read like static documentation with screenshots.

The most important product idea is that every section teaches through real feedback. When the reader does something, the page should reveal multiple synchronized views of the same event: the friendly explanation, the compact summary, and the exact backend payload. This is how the essay becomes both understandable and trustworthy.

The first section should feel calm, focused, and concrete. Later sections can become more analytical and instrument-heavy, but the overall visual language should remain consistent: editorial layout, strong hierarchy, quiet surfaces, and one accent color for live state.

## Problem Statement

The REPL now exposes more than a prompt and a result. It has session identity, profile policy, runtime diffs, rewrite behavior, persistence, restore, and timeout semantics. That richness is valuable, but it is difficult to teach with a normal documentation page because many of the most important concepts are invisible until the user can see state changing in response to a real operation.

The design problem is to make these invisible mechanics visible without turning the experience into a cluttered debugger. The reader should feel guided, not buried.

## Proposed Solution

The essay should be designed as a sequence of focused chapters. Each chapter has:

- one core learning goal,
- one or two dominant interactions,
- one friendly explanatory view,
- one compact system-summary view,
- and one precise raw-data view.

That structure keeps the essay readable for newcomers and useful for implementers.

## Visual Direction

### Overall mood

- Technical but calm
- Editorial, not dashboard-heavy
- Confident, not playful
- Minimal color palette with one accent used for live/backend-driven state
- Strong typography hierarchy and generous spacing

### Visual metaphors

The page should feel like:

- a lab notebook,
- a systems essay,
- a guided instrument panel.

It should not feel like:

- an admin backend,
- a generic docs site,
- a BI dashboard,
- a chat interface.

### Layout principles

- Keep a clear reading flow from top to bottom
- Let the primary live interaction stay visible
- Use 2-3 strong surfaces per section, not 8 small cards
- Prefer linked panels over separate tabs when comparing one cause to multiple effects
- Use raw JSON as a trust surface, but not as the first surface

## Global UI Grammar

These component families should recur throughout the essay:

### 1. Editorial block

Used for:

- section title
- short prose
- margin-note style callouts

Purpose:

- keeps the experience feeling authored and guided

### 2. Live action block

Used for:

- primary button
- code editor
- canned scenario launcher

Purpose:

- makes the user feel they are driving the system

### 3. Summary block

Used for:

- compact session card
- policy card
- execution summary
- timeout outcome summary

Purpose:

- provides the “what just happened?” layer

### 4. Raw data block

Used for:

- JSON inspector
- export viewer
- route/request/response appendix

Purpose:

- provides verification and engineering trust

## Section-by-Section Handoff

## Section 1: Meet a Session

### UX intent

This section should answer one question very clearly: what is a session in this REPL, concretely?

### Rough layout

Desktop:

- top: short intro text
- middle left: primary `Create Session` action and session summary card
- middle right: policy card
- bottom: collapsible JSON inspector

Mobile:

- intro
- create button
- session summary card
- policy card
- JSON inspector

### Required states

- empty state
- loading state
- success state
- error state

### Data shape

Primary payload:

```ts
type Section1Data = {
  session: SessionSummary | null;
  error?: string;
}
```

Relevant backend shape:

```ts
type SessionSummary = {
  id: string;
  profile: string;
  policy: SessionPolicy;
  createdAt: string;
  cellCount: number;
  bindingCount: number;
  bindings: BindingView[];
  history: HistoryEntry[];
  currentGlobals: GlobalStateView[];
  provenance: ProvenanceRecord[];
}
```

Derived fields for design:

```ts
type SessionHeaderViewModel = {
  id: string;
  profileLabel: string;
  createdAtLabel: string;
  cellCount: number;
  bindingCount: number;
}

type PolicySummaryViewModel = {
  eval: { mode: string; timeoutMs: number; captureLastExpression: boolean; supportTopLevelAwait: boolean };
  observe: { staticAnalysis: boolean; runtimeSnapshot: boolean; bindingTracking: boolean; consoleCapture: boolean; jsdocExtraction: boolean };
  persist: { enabled: boolean; sessions: boolean; evaluations: boolean; bindingVersions: boolean; bindingDocs: boolean };
}
```

Backend routes:

- `POST /api/sessions`
- `GET /api/sessions/:id`

## Section 2: Profiles Change Behavior

### UX intent

This section should make profile differences visually obvious, not buried in prose.

### Rough layout

- top: a short explanation of the three profiles
- center: three aligned columns or one comparison matrix
- bottom: a shared snippet area plus response summaries

The section should visually read as a comparison chapter.

### Important design note

This section depends on session creation with profile selection. That is not fully exposed in the current HTTP create-session handler, so design can proceed, but engineering may need a new create-session override surface.

### Data shape

Primary UX model:

```ts
type Section2ProfileColumn = {
  profile: "raw" | "interactive" | "persistent";
  session: SessionSummary | null;
  lastCell?: CellReport;
  createError?: string;
  evalError?: string;
}
```

Most important fields:

```ts
type Section2ComparisonFields = {
  sessionProfile: string;
  evalMode: string;
  bindingCount: number;
  persistenceEnabled: boolean;
  staticAnalysisEnabled: boolean;
  topLevelAwaitEnabled: boolean;
  timeoutMs: number;
  rewriteMode?: string;
}
```

Derived from:

- `SessionSummary.profile`
- `SessionSummary.policy`
- `SessionSummary.bindingCount`
- `CellReport.rewrite.mode`

Likely needed request shape for the future:

```ts
type CreateSessionOverrideRequest = {
  profile?: "raw" | "interactive" | "persistent";
  policy?: Partial<SessionPolicy>;
}
```

## Section 3: What Happened to My Code?

### UX intent

Show that evaluation is not magic. The system may rewrite source before execution, and the reader should be able to see that plainly.

### Rough layout

- left: source editor
- center: transformed source diff
- right: rewrite steps / provenance
- bottom: compact result summary

This should feel like a “code transformation microscope.”

### Data shape

Primary payload:

```ts
type Section3Data = {
  source: string;
  response: EvaluateResponse | null;
  error?: string;
}
```

Most important sub-shape:

```ts
type RewriteReport = {
  mode: string;
  declaredNames: string[];
  helperNames: string[];
  lastHelperName: string;
  bindingHelperName: string;
  capturedLastExpr: boolean;
  transformedSource: string;
  operations: { kind: string; detail: string }[];
  warnings?: string[];
  finalExpressionSource?: string;
}
```

Important related fields:

- `CellReport.source`
- `CellReport.rewrite`
- `CellReport.execution.status`
- `CellReport.provenance`

Backend route:

- `POST /api/sessions/:id/evaluate`

Request shape:

```ts
type EvaluateRequest = {
  source: string;
}
```

## Section 4: Static Analysis vs Runtime Reality

### UX intent

This section should teach that some facts come from parsing and some come from runtime inspection. The visuals should make that split feel explicit.

### Rough layout

- top: explanation of “before execution” vs “after execution”
- left: static analysis panels
- right: runtime diff panels
- small provenance strip tying them together

### Data shape

Important sub-shapes:

```ts
type StaticReport = {
  diagnostics: DiagnosticView[];
  topLevelBindings: TopLevelBindingView[];
  unresolved: IdentifierUseView[];
  references: BindingReferenceGroup[];
  scope?: ScopeView;
  ast: ASTRowView[];
  astNodeCount: number;
  astTruncated: boolean;
  cst: CSTNodeView[];
  cstNodeCount: number;
  cstTruncated: boolean;
  finalExpression?: RangeView;
  summary: StaticSummaryFact[];
}

type RuntimeReport = {
  beforeGlobals: GlobalStateView[];
  afterGlobals: GlobalStateView[];
  diffs: GlobalDiffView[];
  newBindings: string[];
  updatedBindings: string[];
  removedBindings: string[];
  leakedGlobals: string[];
  persistedByWrap: string[];
  currentCellValue: string;
}
```

Design-facing derived view model:

```ts
type AnalysisVsRuntimeView = {
  diagnosticsCount: number;
  topLevelBindingCount: number;
  unresolvedCount: number;
  globalDiffCount: number;
  leakedGlobalCount: number;
  changedBindingCount: number;
}
```

## Section 5: Bindings Are the Memory of the Session

### UX intent

Teach that the session accumulates meaning over time through bindings, not just through a list of previous commands.

### Rough layout

- top: session timeline / scrubber
- left: current bindings table
- right: binding detail drawer
- bottom: new/updated/removed markers for the selected cell

### Data shape

Relevant session-level fields:

```ts
type BindingView = {
  name: string;
  kind: string;
  origin: string;
  declaredInCell: number;
  lastUpdatedCell: number;
  declaredLine?: number;
  declaredSnippet?: string;
  static?: BindingStaticView;
  runtime: BindingRuntimeView;
  provenance?: ProvenanceRecord[];
}

type HistoryEntry = {
  cellId: number;
  createdAt: string;
  sourcePreview: string;
  resultPreview: string;
  status: string;
}
```

Runtime detail fields used by the per-cell markers:

- `RuntimeReport.newBindings`
- `RuntimeReport.updatedBindings`
- `RuntimeReport.removedBindings`
- `RuntimeReport.leakedGlobals`

### Design note

This section should feel more like “inspect a living environment” than “look at a variable table.”

## Section 6: Persistence, History, and Restore

### UX intent

Show what persistent mode buys you and why it changes the mental model from “temporary REPL” to “recoverable session.”

### Rough layout

- left: durable session list
- center: selected session timeline / export snapshot
- right: restore/replay explanation

### Data shape

Current routes:

- `GET /api/sessions`
- `GET /api/sessions/:id/history`
- `GET /api/sessions/:id/export`
- `POST /api/sessions/:id/restore`

Expected UI model:

```ts
type SessionListItem = {
  sessionId: string;
  createdAt: string;
  updatedAt?: string;
  engineKind?: string;
}

type PersistedHistoryView = {
  sessionId: string;
  evaluations: unknown[]; // exact durable shape comes from repldb export/history payloads
}

type SessionExportView = unknown; // export payload should initially be rendered as structured JSON
```

### Design note

Treat export and history as trustworthy but potentially dense. Friendly summaries first, raw payload second.

## Section 7: Timeouts Are Part of the Contract

### UX intent

Make the timeout story feel safe and understandable. This should read as recovery behavior, not merely failure behavior.

### Rough layout

- top: canned scenario selector
- center: status timeline
- bottom: before/after comparison proving the session still works

### Data shape

Most important fields:

```ts
type ExecutionReport = {
  status: string;
  result: string;
  error?: string;
  durationMs: number;
  awaited: boolean;
  console: ConsoleEvent[];
  hadSideEffects: boolean;
  helperError: boolean;
}
```

Design-facing derived model:

```ts
type TimeoutScenarioResult = {
  scenarioId: string;
  firstRun: { status: string; error?: string; awaited: boolean; durationMs: number };
  recoveryRun?: { status: string; result?: string; error?: string };
}
```

### Design note

Show the recovery run very close to the timeout result. The success-after-timeout is the most important trust signal in the whole section.

## Section 8: Docs and Provenance

### UX intent

Teach that the REPL can explain where its insights came from and can preserve REPL-authored docs as part of the system.

### Rough layout

- left: docs browser
- right: provenance viewer
- top callout explaining “parser-derived / runtime-derived / persisted”

### Data shape

Relevant fields:

```ts
type ProvenanceRecord = {
  section: string;
  source: string;
  notes?: string[];
}
```

Current docs route:

- `GET /api/sessions/:id/docs`

The docs payload comes from `repldb.BindingDocRecord[]`, which should be treated as renderable structured data first and prettified later once the exact article implementation is underway.

## Section 9: API Appendix

### UX intent

Turn the article into a reusable engineering reference once the reader understands the system.

### Rough layout

- route list
- example request/response pairs
- copyable snippets
- compact schema references

### Data shape

This section is mostly documentation assembly rather than one live object. It should include:

- session creation response
- evaluate request/response
- snapshot response
- bindings response
- history response
- docs response
- export response

## Design Decisions

- Section 1 should be narrow and calm. It establishes trust before adding complexity.
- Every section should combine friendly summary plus raw backend truth.
- Raw JSON is mandatory, but never the primary visual surface.
- Profile comparison deserves a dedicated section, not an overloaded first screen.
- The UX can work ahead even where backend gaps remain, as long as the gap is explicit.

## Alternatives Considered

### Purely aesthetic handoff with no data contracts

Rejected because the UX designer needs to know which panels can rely on stable data and which ones depend on future backend work.

### Backend schema dump with no visual guidance

Rejected because the user explicitly wants something useful for design, not only an engineering appendix.

## Implementation Plan

1. Use this handoff to design all sections at low fidelity.
2. Start implementation with Section 1 only.
3. Reuse the section-level data shapes as the contract between design and backend work.
4. Refine the less-settled sections, especially persistence/docs/export, once the first implementation slice lands.

## Open Questions

- Will the article need explicit create-session profile selection before design can finalize Section 2?
- Should export/history get custom article-specific summary payloads, or should the first version render raw structured data directly?
- Does the final article want a richer client-side code editor, or is a simpler source input enough for the first implementation?

## References

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/08/GOJA-043-INTERACTIVE-REPL-ESSAY--design-an-interactive-repl-essay-for-learning-and-validating-the-new-repl/design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go`

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
