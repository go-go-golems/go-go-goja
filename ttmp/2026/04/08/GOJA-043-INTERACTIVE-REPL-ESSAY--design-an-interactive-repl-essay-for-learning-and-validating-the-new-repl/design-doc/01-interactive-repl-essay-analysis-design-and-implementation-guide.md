---
Title: Interactive REPL essay analysis, design, and implementation guide
Ticket: GOJA-043-INTERACTIVE-REPL-ESSAY
Status: active
Topics:
    - repl
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: Serve command and command-level entrypoints for future article hosting
    - Path: pkg/replapi/app.go
      Note: App-layer facade and session lifecycle API that the article should teach
    - Path: pkg/replapi/config.go
      Note: Profiles and session option model needed for profile-comparison sections
    - Path: pkg/replhttp/handler.go
      Note: Current HTTP route surface the article can exercise directly
    - Path: pkg/replsession/evaluate.go
      Note: Evaluation pipeline
    - Path: pkg/replsession/persistence.go
      Note: Persistence/export behavior for history and restore sections
    - Path: pkg/replsession/types.go
      Note: Core JSON response types that drive article panels
ExternalSources: []
Summary: Design guide for a Bret Victor-style interactive article that teaches and validates the new REPL by using the real API.
LastUpdated: 2026-04-08T20:18:24.8195301-04:00
WhatFor: Define the best possible interactive article concept before implementation.
WhenToUse: Use when planning or building a live teaching interface for the REPL/session system.
---


# Interactive REPL essay analysis, design, and implementation guide

## Executive Summary

This ticket proposes an interactive article or dynamic essay that teaches the new REPL by directly exercising the real system. The article should not be a passive explanation page with screenshots. It should create real sessions, evaluate real code, show real rewrite/runtime/persistence artifacts, and let the reader validate that the current implementation behaves as documented.

The strongest version of this article is not only educational. It is also a product-level validation harness. If the article can walk a reader through session creation, binding tracking, persistence, restore, history replay, and timeout recovery using the live API, then the article doubles as a human-readable acceptance test for the REPL stack.

The best implementation direction is an HTTP-backed article that sits close to the existing `goja-repl serve` JSON API. That is the shortest path to high signal because the backend already exposes the session lifecycle and evaluation response objects, and those response objects are unusually rich: they already include static analysis, rewrite information, execution results, runtime diffs, provenance, binding summaries, and history-oriented state.

The main design gap is that the current HTTP create-session route is fixed to the app default and does not accept creation-time session profile or policy overrides. A truly great article needs to compare raw, interactive, and persistent behavior side by side. That means either the HTTP API needs a create-session override body, or the article server needs a small extension layer that can create sessions with explicit options.

## Problem Statement

The new REPL is much stronger than the old mental model of "type JavaScript, get a result." It now has:

- multiple session profiles,
- an app-layer facade with optional auto-restore,
- a session service with different evaluation modes,
- rich cell reports,
- persistence and replay behavior,
- timeout and interruption semantics,
- and JSON/CLI surfaces that expose much of this state.

That is good engineering, but it makes the system harder to learn by reading only prose docs. The existing REPL usage doc explains the interactive CLI well enough for a user, but it does not deeply teach the session model, evaluation pipeline, or persistence/restore behavior. It also does not make validation easy.

The user request here is specifically aiming at a Bret Victor-like artifact. That implies a different teaching standard:

- show state changing as the reader interacts,
- keep cause and effect close together,
- make invisible transformations visible,
- and let the learner form and test hypotheses immediately.

In practical terms, the article should answer questions like these by demonstration, not only explanation:

- What is the difference between raw, interactive, and persistent sessions?
- What exactly gets rewritten before execution?
- What is being tracked statically versus at runtime?
- What persists across cells? Across process restarts? Across restore?
- What happens when evaluation times out?
- Which parts of the report are parser-derived, runtime-derived, or persistence-derived?

## Scope

This ticket is design-first. It does not implement the article. It defines:

- the best teaching goals,
- the article structure,
- the necessary interactive widgets,
- the current backend/API support,
- the missing capabilities,
- and the most realistic implementation options.

It does not decide final frontend technology beyond a recommendation hierarchy, and it does not yet change production code.

## Current System: What the Article Must Teach

### 1. App layer: `replapi.App`

The highest-level backend entrypoint is `replapi.App`. It combines the live session service with optional durable storage and provides a clean application facade.

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:15`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:22`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:59`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:74`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:85`

Important functions:

- `CreateSession`
- `CreateSessionWithOptions`
- `Evaluate`
- `Snapshot`
- `Restore`
- `DeleteSession`
- `ListSessions`
- `History`
- `Export`
- `Bindings`
- `Docs`
- `WithRuntime`

This means the article can be organized around real user stories instead of internal abstractions:

1. create a session
2. evaluate a cell
3. inspect the session snapshot
4. inspect history/bindings/docs/export
5. delete and restore as applicable

### 2. App configuration profiles

The current top-level app model already has named profiles:

- `raw`
- `interactive`
- `persistent`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go:12`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go:50`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go:74`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go:84`

That makes profile comparison one of the most important article sections. It is already a conceptual axis in the codebase, so the article should treat it as a first-class teaching dimension, not a footnote.

### 3. Session policy

Per-session behavior is defined in `replsession.SessionPolicy`, which groups:

- `EvalPolicy`
- `ObservePolicy`
- `PersistPolicy`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go:17`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go:25`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go:34`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go:43`

This is important for the article because it gives a clear teaching structure:

- evaluation behavior,
- observation behavior,
- persistence behavior.

The article should surface these policies visually, ideally as a "profile card" or "policy inspector" that maps toggles to visible outcomes.

### 4. Session summary and cell report types

The single most article-friendly backend feature is that the REPL already produces rich JSON-safe report types.

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:5`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:24`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:30`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:42`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:60`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:83`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go:103`

The article can use these directly:

- `SessionSummary`
- `EvaluateResponse`
- `CellReport`
- `ExecutionReport`
- `StaticReport`
- `RewriteReport`
- `RuntimeReport`
- `BindingView`
- `HistoryEntry`

This is the core reason a live essay is realistic: the backend already emits structured artifacts that are interesting enough to teach from.

### 5. HTTP API surface

The current JSON handler exposes a clean route set:

```text
GET    /api/sessions
POST   /api/sessions
GET    /api/sessions/:id
DELETE /api/sessions/:id
POST   /api/sessions/:id/evaluate
POST   /api/sessions/:id/restore
GET    /api/sessions/:id/history
GET    /api/sessions/:id/bindings
GET    /api/sessions/:id/docs
GET    /api/sessions/:id/export
```

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:14`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:21`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:41`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:75`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:103`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:114`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:136`

This is already enough for a high-value first article implementation.

### 6. CLI and server entrypoint

The canonical user-facing command already exposes a JSON server:

- `goja-repl serve`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:45`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:72`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:512`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:535`

That means the article does not need a brand-new backend from scratch. The likely future implementation path is either:

- extend this server with article/static asset handling, or
- build a sibling command that wraps the same app and handler surfaces.

## Behavioral Evidence the Article Should Demonstrate

The article should be structured around behaviors that are already tested.

### Session lifecycle and history

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler_test.go:18`

This test already proves:

- session creation,
- evaluation,
- history retrieval,
- export.

That sequence is almost already a chapter outline.

### Raw vs interactive differences

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:12`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:41`

These tests prove:

- raw sessions use direct execution and do not track bindings
- interactive sessions use the instrumented rewrite path and do track bindings

That should become a comparison widget in the article.

### Top-level await behavior

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:67`

The article should explicitly teach the current contract:

- expression-style top-level `await` can work
- declaration-style raw-mode top-level `await` still errors

This is perfect dynamic-essay material because it is exactly the kind of behavior a learner can test live.

### Timeout and recovery

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:108`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:135`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go:172`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go:31`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:409`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:445`

This is one of the strongest article sections because:

- it is easy to misunderstand from prose,
- it matters operationally,
- and the current implementation is now strong enough to demonstrate live.

## Proposed Article

### High-level design goal

The article should teach by letting the reader manipulate a real session and see multiple synchronized views of the same event:

```text
reader input
  -> real session API call
  -> cell report
  -> synchronized views:
       result
       rewrite
       runtime diff
       bindings
       history
       persistence/export
```

This is the core Bret Victor-like move: one action, many linked explanations.

### Recommended article shape

The article should be a linear guided experience, but every section should remain interactive and explorable after the reader passes it. In other words:

- read top to bottom for onboarding
- jump around later as a living reference

Recommended section list:

1. Why the new REPL is more than a prompt
2. Session profiles: raw vs interactive vs persistent
3. One cell, many views: source -> rewrite -> execution -> runtime
4. Bindings and globals
5. Persistence, history, and restore
6. Timeout and recovery
7. Docs and provenance
8. "What changed if I restart?" replay theater
9. Appendix: API contracts and real routes

## Interactive Widgets and Sections

This is the most important part of the ticket.

### Section 1: "Meet a session"

Educational goal:

- explain that a REPL session is a durable or in-memory entity with an ID, profile, policy, and evolving state

Widgets:

- profile card
- "Create session" button
- session metadata panel
- live JSON inspector for `SessionSummary`

Real API calls:

- `POST /api/sessions`
- `GET /api/sessions/:id`

Important note:

The current handler only creates sessions with the app's default configuration. For the article to fully teach profile differences, this route likely needs a request body for overrides or a sibling route.

### Section 2: "Profiles change behavior"

Educational goal:

- show that raw, interactive, and persistent are not marketing labels; they are different policies with visible consequences

Widgets:

- profile selector
- three synchronized source editors
- comparison table for:
  - rewrite mode
  - binding tracking
  - persistence enabled
  - static analysis enabled
  - top-level await support
  - timeout setting

Real backend fields:

- `SessionSummary.Profile`
- `SessionSummary.Policy`
- `CellReport.Rewrite.Mode`
- `SessionSummary.BindingCount`

Best demonstration snippet:

```javascript
const x = 1; x
```

Expected comparison:

- raw: direct execution, no binding tracking
- interactive: instrumented rewrite, binding tracking
- persistent: like interactive, plus durable writes

### Section 3: "What happened to my code?"

Educational goal:

- make the rewrite pipeline visible

Widgets:

- source editor
- transformed source diff
- rewrite operations timeline
- provenance badge strip

Real backend fields:

- `CellReport.Source`
- `CellReport.Rewrite.TransformedSource`
- `CellReport.Rewrite.Operations`
- `CellReport.Provenance`

Best demonstration snippets:

- `const x = 1; x`
- `await Promise.resolve(9)`
- `const x = await Promise.resolve(3); x`

### Section 4: "Static analysis vs runtime reality"

Educational goal:

- teach the difference between parser-derived information and runtime-derived information

Widgets:

- static report inspector
- AST/CST summary counters
- top-level bindings table
- runtime global diff panel
- "What was known before execution?" callout

Real backend fields:

- `CellReport.Static.Diagnostics`
- `CellReport.Static.TopLevelBindings`
- `CellReport.Static.Scope`
- `CellReport.Runtime.BeforeGlobals`
- `CellReport.Runtime.AfterGlobals`
- `CellReport.Runtime.Diffs`

Teaching point:

The article should make clear that these views come from different sources. This is already encoded in `ProvenanceRecord`, which makes the explanation easier and more honest.

### Section 5: "Bindings are the memory of the session"

Educational goal:

- explain how bindings persist across cells and how tracked bindings differ from leaked globals

Widgets:

- history scrubber
- current bindings table
- per-binding detail drawer
- "new / updated / removed" badges

Real backend fields:

- `SessionSummary.Bindings`
- `RuntimeReport.NewBindings`
- `RuntimeReport.UpdatedBindings`
- `RuntimeReport.RemovedBindings`
- `RuntimeReport.LeakedGlobals`

Best demonstration snippets:

```javascript
const answer = 41
answer + 1
```

```javascript
foo = 7
```

The second example is especially useful because it helps explain leaked globals versus declared bindings.

### Section 6: "Persistence and replay"

Educational goal:

- show exactly what persistent mode buys you

Widgets:

- session list browser
- history timeline
- export JSON viewer
- replay source list
- "kill and restore" button pair

Real API calls:

- `GET /api/sessions`
- `GET /api/sessions/:id/history`
- `GET /api/sessions/:id/export`
- `POST /api/sessions/:id/restore`

Best demonstration:

1. create persistent session
2. evaluate a few cells
3. show durable history
4. simulate restart or forced restore
5. show that the session summary comes back

### Section 7: "Timeouts are part of the contract"

Educational goal:

- demonstrate that timeouts are not just errors; they are managed recovery behavior

Widgets:

- canned examples list:
  - never-settling promise
  - `while (true) {}`
  - normal cell after timeout
- execution status panel
- timeline showing timeout then successful follow-up

Real backend fields:

- `ExecutionReport.Status`
- `ExecutionReport.Error`
- `ExecutionReport.Awaited`
- post-timeout `EvaluateResponse`

Best demonstration sequence:

```text
run infinite loop
observe timeout
run 1 + 1
observe successful recovery
```

This section is a very high-value validation tool because it exercises recently hardened behavior.

### Section 8: "Docs and provenance"

Educational goal:

- explain that the REPL can produce more than values; it can preserve documentation and provenance

Widgets:

- provenance inspector
- docs endpoint browser
- binding docs cross-reference panel

Real API calls and fields:

- `GET /api/sessions/:id/docs`
- `CellReport.Provenance`
- `BindingView.Provenance`

This section can also link back into the repo's existing docs ecosystem, which helps position the article as part of a larger documentation system rather than a one-off page.

### Section 9: "API appendix"

Educational goal:

- give the reader copy/paste power after the guided experience

Widgets:

- route list
- sample request/response pairs
- "Open this exact request in curl" snippets

This appendix is important because the article should not trap knowledge inside the UI.

## Article Interaction Model

The article should keep one shared client-side model:

```pseudo
state = {
  activeProfile,
  sessionID,
  lastCellReport,
  lastSessionSummary,
  history,
  bindings,
  docs,
  exportPayload,
  comparisonSessions,
}
```

When the reader edits source or presses a button:

```pseudo
if no session exists:
  create session

send evaluate request
receive EvaluateResponse
update:
  session summary
  cell report
  history
  bindings view
  runtime diff view
  rewrite view
```

This model matters because it keeps all panels synchronized. The article should feel like one living notebook, not separate demos glued together.

## Recommended Technical Approach

### Option A: Extend the existing HTTP server with article pages

This is the recommended first implementation.

Why:

- the backend already exists,
- the handler already exposes most of what the article needs,
- the article wants a real server-backed session model,
- and the repo already has lightweight server-rendered UI precedents.

Relevant precedents:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-perf/serve_command.go:63`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-perf/serve_command.go:120`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-perf/serve_command.go:217`

This file shows that the repo already accepts a simple local web-app pattern using:

- `net/http`
- inline templates
- Bootstrap
- HTMX

That would be enough for a very strong first article implementation.

### Option B: Add a richer bundled frontend

This is a good second choice if the article needs more advanced client-side synchronization, rich code editors, or linked animation.

Relevant precedent:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/bun-demo/main.go:19`

That example shows an embedded bundle pattern already exists in the repo. It is not the final article solution by itself, but it proves that shipping embedded frontend assets inside a Go command fits the repo.

### Option C: Run a REPL runtime in the browser

This is conceptually exciting and closer to the Bret Victor ideal of fully local feedback, but it is not the recommended first implementation.

Why not first:

- the current repo evidence shows server-backed Go/Goja runtime patterns, not browser-side Goja runtime deployment
- the article's strongest validation value comes from talking to the real backend implementation anyway

This option should be treated as an advanced future experiment, not the default path.

## Required Backend/Product Gaps

The article can do a lot with the current API, but not everything.

### Gap 1: profile selection at create time over HTTP

Current reality:

- `goja-repl serve` constructs the app with the persistent profile by default
- `POST /api/sessions` does not accept a body for session overrides

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:109`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go:145`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:21`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go:30`

Why it matters:

- the article wants raw vs interactive vs persistent comparisons
- the current HTTP surface cannot express that cleanly

Recommended future API extension:

```json
POST /api/sessions
{
  "profile": "interactive",
  "policy": {
    "eval": { "timeoutMs": 2500 }
  }
}
```

This should map to `replapi.CreateSessionWithOptions(...)`.

### Gap 2: article-specific "guided examples" metadata

The backend does not currently expose curated teaching scenarios. That is fine, but it means the first article implementation should ship its own scenario catalog on the UI side.

Example scenario model:

```json
{
  "id": "timeout-recovery",
  "title": "Timeout and recovery",
  "profile": "interactive",
  "steps": [
    "while (true) {}",
    "const x = 41; x + 1"
  ],
  "expected": [
    {"status": "timeout"},
    {"status": "ok", "result": "42"}
  ]
}
```

### Gap 3: optional richer restore/restart simulation

The current API supports restore, but a great article may want an explicit "simulate fresh process" affordance. That may require a small article-specific orchestration layer if the implementation wants to tear down and recreate app/server state on demand.

## Intern-Oriented Explanation of Where the Article's Data Comes From

This section matters because a new engineer implementing the article needs to know which backend file produces which panel.

### Source editor output -> `EvaluateResponse`

When the article submits code, the backend eventually calls:

- `replapi.App.Evaluate(...)`
- `replsession.Service.Evaluate(...)`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:74`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:27`

### Rewrite panel -> `buildRewriteReport(...)`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:96`

### Static analysis panel -> `buildStaticReport(...)`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:46`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:55`

### Runtime diff panel -> `evaluateInstrumented(...)` / `evaluateRaw(...)`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:146`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:241`

### Persistence/export/history panels -> `persistCell(...)` and store-backed app methods

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go:20`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:118`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:125`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:132`

### Restore/replay section -> `Restore(...)` and `RestoreSession(...)`

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go:85`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go:237`

### Timeout/recovery section -> `ErrEvaluationTimeout`, execution watcher, promise waiting

Evidence:

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go:31`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:355`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:409`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go:445`

## Suggested Implementation Phases

### Phase 0: choose the shell

Decide whether the first implementation is:

- HTMX/server-rendered
- richer bundled frontend

Recommendation: choose HTMX/server-rendered first unless the visual synchronization requirements immediately prove too awkward.

### Phase 1: build the teaching skeleton

Implement:

- article landing page
- shared layout
- one session card
- one editor
- one evaluate button
- result panel

Acceptance criteria:

- can create a session
- can evaluate one cell
- can show `SessionSummary` and `CellReport`

### Phase 2: add the comparison chapter

Implement:

- profile selector
- compare raw vs interactive vs persistent

Blocking need:

- HTTP create-session override support, or an article-specific backend wrapper

### Phase 3: add linked multi-panel explanations

Implement:

- rewrite view
- static/runtime split view
- bindings/history panels

Acceptance criteria:

- one evaluation updates all linked views

### Phase 4: add persistence and restore theater

Implement:

- session list
- export view
- restore path demonstration

### Phase 5: add timeout chapter

Implement:

- canned timeout scenarios
- visible recovery proof

Acceptance criteria:

- user can watch timeout happen and then validate subsequent successful evaluation

## Validation Strategy

The article should be treated as both documentation and system verification.

Validation layers:

1. UI-level manual validation
2. route-level contract validation
3. scenario-level acceptance validation

Example article validation checklist:

- create raw session and confirm no binding tracking
- create interactive session and confirm binding tracking
- create persistent session and confirm history/export/restore work
- run top-level `await` expression and observe expected behavior
- run declaration-style raw-mode `await` and observe expected error
- trigger timeout on a promise
- trigger timeout on a synchronous infinite loop
- run a normal cell immediately afterward and confirm session recovery

## Alternatives Considered

### Alternative 1: write a normal static tutorial

Rejected because it would explain behavior without proving it. The REPL now has enough complexity that screenshots and prose alone are lower-signal than live interaction.

### Alternative 2: build only a test harness, no article

Rejected because the user explicitly wants learning value. A pure harness validates behavior but does not optimize for understanding.

### Alternative 3: start with browser-only runtime execution

Rejected as a first step because it adds major technical risk while weakening the direct connection to the real deployed backend behavior.

## Open Questions

- Should the article live inside `goja-repl serve`, or under a new sibling command such as `goja-repl essay`?
- Is HTMX enough for synchronized panel linking, or will the article want a richer stateful frontend from the start?
- Should create-session profile/policy overrides be added directly to `pkg/replhttp`, or handled by a wrapper endpoint just for the article?
- Should the article support plugin/module demonstrations in the first version, or focus on the core session/evaluation model first?

## References

### Core code

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/policy.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/evaluate.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/persistence.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go`

### Tests

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler_test.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service_policy_test.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app_test.go`

### Existing documentation to link from the article

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/doc/04-repl-usage.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-040-PERSISTENCE-CORRECTNESS--fix-repl-persistence-correctness-and-sqlite-integrity/design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/design-doc/01-evaluation-control-analysis-design-and-implementation-guide.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-042-REPL-CLEANUP--refactor-session-kernel-and-api-shape-cleanup/design-doc/01-repl-cleanup-analysis-design-and-implementation-guide.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/08/repl-hardening-project-report.md`

### Relevant UI precedents

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-perf/serve_command.go`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-jsdoc/doc/01-jsdoc-system.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/bun-demo/main.go`
