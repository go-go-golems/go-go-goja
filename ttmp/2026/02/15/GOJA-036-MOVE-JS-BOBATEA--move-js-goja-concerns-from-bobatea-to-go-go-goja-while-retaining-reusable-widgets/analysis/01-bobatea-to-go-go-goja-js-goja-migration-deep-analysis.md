---
Title: Bobatea to go-go-goja JS/Goja Migration Deep Analysis
Ticket: GOJA-036-MOVE-JS-BOBATEA
Status: active
Topics:
    - goja
    - bobatea
    - architecture
    - repl
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: bobatea/pkg/repl/evaluators/javascript/evaluator.go
      Note: |-
        Current JS-specific evaluator implementation in bobatea
        Removed from bobatea per ownership split
    - Path: bobatea/pkg/repl/model.go
      Note: Generic REPL shell and widget wiring retained in bobatea
    - Path: bobatea/pkg/tui/widgets/contextbar/widget.go
      Note: Generic context help bar widget reuse target
    - Path: bobatea/pkg/tui/widgets/contextpanel/widget.go
      Note: Generic context drawer widget reuse target
    - Path: bobatea/pkg/tui/widgets/suggest/widget.go
      Note: Generic completion widget reuse target
    - Path: go-go-goja/cmd/js-repl/main.go
      Note: Implemented command ownership move from bobatea example
    - Path: go-go-goja/cmd/repl/main.go
      Note: Current plain go-go-goja REPL command
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Current smalltalk-inspector model with local REPL pane
    - Path: go-go-goja/cmd/smalltalk-inspector/app/repl_widgets.go
      Note: Implemented widget reuse in inspector REPL
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Current REPL key/update flow integration point
    - Path: go-go-goja/pkg/repl/adapters/bobatea/javascript.go
      Note: Implemented adapter recommended by migration plan
    - Path: go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/analyze_bobatea_goja_boundary.sh
      Note: Experiment script for package coupling boundary
    - Path: go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/widget_reuse_matrix.sh
      Note: Experiment script for widget reuse matrix
ExternalSources: []
Summary: Deep migration plan to keep generic TUI widgets in bobatea while moving JavaScript/Goja-specific evaluator and js-repl ownership to go-go-goja.
LastUpdated: 2026-02-15T15:40:00-05:00
WhatFor: Define target architecture and phased execution for JS/Goja ownership boundaries between bobatea and go-go-goja, including smalltalk-inspector REPL integration opportunities.
WhenToUse: Use when implementing or reviewing GOJA-036 migration tasks and related inspector/repl refactors.
---



# GOJA-036 Deep Analysis: Moving JS/Goja-Specific REPL Pieces from Bobatea to go-go-goja

## Executive Summary

This audit shows a very clean boundary already exists in the codebase:

- `bobatea` is mostly generic TUI infrastructure.
- JS/Goja coupling in `bobatea/pkg` is concentrated in exactly one package: `bobatea/pkg/repl/evaluators/javascript`.
- `go-go-goja` currently imports only one Bobatea package (`mode-keymap`) in production code, even though it already has a Bobatea module dependency.

The highest-leverage plan is:

1. Move the JavaScript evaluator implementation from `bobatea/pkg/repl/evaluators/javascript` to a `go-go-goja`-owned package.
2. Move `bobatea/examples/js-repl` to a first-class `go-go-goja` command (`cmd/js-repl`) while keeping the existing plain `cmd/repl` intact.
3. Keep generic REPL/widget abstractions in Bobatea and consume them from go-go-goja where useful.
4. Incrementally enhance `smalltalk-inspector` REPL pane using Bobatea generic widgets (`suggest`, `contextbar`, `contextpanel`, `commandpalette`) rather than replacing the whole inspector UI with Bobatea REPL.

This gives correct ownership (JS logic in go-go-goja), avoids over-migration of generic code, and creates a controlled path to richer inspector REPL UX.

## User Intent Restatement

Requested outcome:

- Move all Goja-dependent / JS-relevant code out of Bobatea and into go-go-goja.
- Keep generic REPL/widget infrastructure in Bobatea.
- Reuse those generic components in go-go-goja, especially for `smalltalk-inspector` (which includes a REPL pane).
- Move JS REPL experience ownership to go-go-goja.
- Provide exhaustive, detailed migration analysis across widgets.

## Research Method and Evidence

### Static inventory

- `bobatea/pkg` and `go-go-goja` package trees were enumerated.
- Symbol and import scans were performed to identify coupling edges.

Key direct evidence:

- Only JS evaluator in Bobatea imports go-go-goja:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:13`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:14`
- go-go-goja imports Bobatea package code only via mode-keymap:
  - `go-go-goja/cmd/inspector/app/model.go:18`
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go:15`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go:11`

### Focused code reads

Deep reads covered:

- Bobatea REPL core and feature submodels:
  - `bobatea/pkg/repl/model.go`
  - `bobatea/pkg/repl/completion_model.go`
  - `bobatea/pkg/repl/helpbar_model.go`
  - `bobatea/pkg/repl/helpdrawer_model.go`
  - `bobatea/pkg/repl/command_palette_model.go`
- Bobatea JS evaluator:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go`
- Smalltalk inspector model + REPL pane flow:
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go`
  - `go-go-goja/cmd/smalltalk-inspector/app/view.go`
- go-go-goja terminal REPL command:
  - `go-go-goja/cmd/repl/main.go`

### Experiments saved in ticket scripts

Saved under:

- `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/analyze_bobatea_goja_boundary.sh`
- `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/widget_reuse_matrix.sh`

Generated outputs:

- `.../scripts/out/bobatea_pkg_summary.tsv`
- `.../scripts/out/bobatea_pkg_imports.tsv`
- `.../scripts/out/go_go_goja_imports_from_bobatea.txt`
- `.../scripts/out/widget_reuse_matrix.tsv`
- `.../scripts/out/go_go_goja_bobatea_widget_imports.tsv`

## Current Architecture Snapshot

## 1) Bobatea REPL architecture is already generic-first

Core model and orchestration are evaluator-agnostic:

- `bobatea/pkg/repl/model.go:21` defines a timeline-first REPL shell with generic evaluator interface (`repl.Evaluator`).
- It wires autocomplete/help/palette features through capability interfaces (`InputCompleter`, `HelpBarProvider`, `HelpDrawerProvider`) instead of hard-coding language logic:
  - `bobatea/pkg/repl/model.go:80`
  - `bobatea/pkg/repl/model.go:84`
  - `bobatea/pkg/repl/model.go:88`

Generic widget extractions are explicit:

- Completion is routed through `tui/widgets/suggest`:
  - `bobatea/pkg/repl/completion_model.go:64`
- Help bar is routed through `tui/widgets/contextbar`:
  - `bobatea/pkg/repl/helpbar_model.go:34`
- Help drawer is routed through `tui/widgets/contextpanel`:
  - `bobatea/pkg/repl/helpdrawer_model.go:43`
- Type aliases expose widget request/result types through REPL API:
  - `bobatea/pkg/repl/autocomplete_types.go:10`
  - `bobatea/pkg/repl/help_bar_types.go:10`
  - `bobatea/pkg/repl/help_drawer_types.go:10`

Conclusion: Bobatea REPL infrastructure is already in the right place (generic/shared).

## 2) JS/Goja specialization is concentrated in one Bobatea package

`bobatea/pkg/repl/evaluators/javascript/evaluator.go` carries all JS-specific runtime, completion, and contextual help behavior:

- Goja runtime ownership and synchronization:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:55`
- Direct Goja import:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:10`
- Direct go-go-goja engine + jsparse dependencies:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:13`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:14`
- Runtime-backed completion/help logic built on jsparse CST + resolver:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:221`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:317`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:368`

Conclusion: This package should be moved to go-go-goja ownership.

## 3) go-go-goja has two separate REPL experiences today

- A simple line REPL command:
  - `go-go-goja/cmd/repl/main.go:52` (`runInteractiveLoop`)
- A richer REPL pane inside smalltalk-inspector with custom local handling:
  - REPL input state in model: `go-go-goja/cmd/smalltalk-inspector/app/model.go:63`
  - REPL key handling: `go-go-goja/cmd/smalltalk-inspector/app/update.go:477`
  - REPL rendering: `go-go-goja/cmd/smalltalk-inspector/app/view.go:547`

This means migration should produce a coherent "go-go-goja-owned JS REPL layer" consumed by both standalone and inspector contexts.

## 4) Current cross-repo dependency state

- go-go-goja already depends on Bobatea module:
  - `go-go-goja/go.mod:13`
- Bobatea currently depends on go-go-goja module:
  - `bobatea/go.mod:15`

This bi-directional module awareness is already present; migration should reduce unnecessary circular ownership by moving JS-specific code to go-go-goja and leaving generic code in Bobatea.

## Detailed Widget-by-Widget Audit and Migration Position

This section explicitly evaluates all relevant Bobatea UI components for keep/move/reuse decisions.

Format:

- Purpose
- JS/Goja coupling
- Current go-go-goja usage
- Migration decision
- Recommended integration in go-go-goja

### A) `pkg/mode-keymap`

- Purpose: reflection-based mode-tag key enabling (`keymap-mode`), used for dynamic keymap states.
- Coupling: generic, no JS/Goja imports.
- Current usage: actively used by go-go-goja inspector UIs:
  - `go-go-goja/cmd/inspector/app/model.go:18`
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go:15`
- Decision: keep in Bobatea.
- Recommendation: continue consuming as shared utility; do not fork into go-go-goja.

### B) `pkg/tui/widgets/suggest`

- Purpose: async debounced completion widget with popup layout and navigation.
- Coupling: generic provider contract (`Provider.CompleteInput`), no JS runtime awareness.
- Current usage: indirectly used through Bobatea REPL completion model.
- Decision: keep in Bobatea.
- Recommendation: directly integrate into `smalltalk-inspector` REPL pane in phase 2.

### C) `pkg/tui/widgets/contextbar`

- Purpose: one-line contextual guidance widget with debounce and request timeout.
- Coupling: generic provider contract (`GetContextBar`), no JS runtime awareness.
- Current usage: indirectly used by Bobatea REPL help bar model.
- Decision: keep in Bobatea.
- Recommendation: directly integrate into `smalltalk-inspector` REPL pane.

### D) `pkg/tui/widgets/contextpanel`

- Purpose: richer contextual help drawer widget, toggle/pin/refresh flows, dock layout.
- Coupling: generic provider contract (`GetContextPanel`).
- Current usage: indirectly used by Bobatea REPL help drawer model.
- Decision: keep in Bobatea.
- Recommendation: integrate as optional inspector REPL overlay/panel to expose symbol docs and parse diagnostics.

### E) `pkg/commandpalette`

- Purpose: fuzzy command palette model with execution callback routing.
- Coupling: generic.
- Current usage: used by Bobatea REPL command palette submodel.
- Decision: keep in Bobatea.
- Recommendation: use for smalltalk inspector command mode unification; optionally replace ad hoc `:command` parsing flow.

### F) `pkg/overlay`

- Purpose: terminal overlay placement helper and whitespace handling.
- Coupling: generic rendering utility.
- Current usage: Bobatea overlays and panel layout helpers.
- Decision: keep in Bobatea.
- Recommendation: use when integrating suggest/help drawer into inspector REPL area to avoid custom placement bugs.

### G) `pkg/listbox`

- Purpose: compact selection list bubble component.
- Coupling: generic.
- Current usage: currently not used by go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: optional for globals/members list simplification if custom scroll logic in `smalltalk-inspector` is later reduced.

### H) `pkg/textarea`

- Purpose: high-capability multiline text area fork with memoization performance workaround.
- Coupling: generic text editing utility.
- Current usage: not used by go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: adopt for multiline REPL expressions in smalltalk-inspector once single-line textinput becomes limiting.

### I) `pkg/tui/inputhistory`

- Purpose: generic input history/navigation.
- Coupling: generic.
- Current usage: used by Bobatea REPL via aliasing.
- Decision: keep in Bobatea.
- Recommendation: if inspector REPL history behavior is expanded, use this package instead of bespoke history arrays.

### J) `pkg/tui/asyncprovider`

- Purpose: generic async provider runner with request IDs and timeout control.
- Coupling: generic.
- Current usage: used heavily by suggest/contextbar/contextpanel.
- Decision: keep in Bobatea.
- Recommendation: consume indirectly through widgets; avoid reimplementing debounce/timeout churn in inspector.

### K) `pkg/autocomplete`

- Purpose: suggestion data model and completion primitive types.
- Coupling: generic.
- Current usage: JS evaluator uses `autocomplete.Suggestion` for completion result translation.
- Decision: keep in Bobatea.
- Recommendation: acceptable as shared type dependency for Bobatea-oriented adapters.

### L) `pkg/timeline`, `pkg/timeline/renderers`, `pkg/eventbus`

- Purpose: event-driven timeline transcript UI and entity renderer stack.
- Coupling: generic UI/event infra, no JS-specific logic.
- Current usage: used by Bobatea REPL and js-repl example.
- Decision: keep in Bobatea.
- Recommendation: reuse in go-go-goja `cmd/js-repl` to preserve rich transcript behavior.

### M) `pkg/filepicker`

- Purpose: advanced file picker with history, preview, sorting, jail mode.
- Coupling: generic file UI.
- Current usage: currently not imported by go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: optional for inspector command `:load` UX improvement.

### N) `pkg/sparkline`

- Purpose: lightweight terminal sparkline chart model.
- Coupling: generic.
- Current usage: none in go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: optional for runtime/perf telemetry panes in future tooling.

### O) `pkg/diff`

- Purpose: data diff TUI component with filters/search/detail.
- Coupling: generic.
- Current usage: none in go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: potential future use for AST/runtime diff views; not part of GOJA-036 critical path.

### P) `pkg/buttons`

- Purpose: selection/confirmation button model.
- Coupling: generic.
- Current usage: none in go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: optional for interactive confirm flows in future commands.

### Q) `pkg/chat`

- Purpose: chat-focused model with timeline-backed interaction patterns.
- Coupling: generic/chat domain, not JS-runtime specific.
- Current usage: none in go-go-goja.
- Decision: keep in Bobatea.
- Recommendation: out of scope for GOJA-036, no move.

## Findings That Directly Determine Migration Scope

## Finding 1: Only one Bobatea package is truly JS/Goja-coupled

`bobatea/pkg/repl/evaluators/javascript` is the only package that both:

- imports `goja` and
- imports go-go-goja runtime/parser (`engine`, `pkg/jsparse`).

Evidence:

- `bobatea/pkg/repl/evaluators/javascript/evaluator.go:10`
- `bobatea/pkg/repl/evaluators/javascript/evaluator.go:13`
- `bobatea/pkg/repl/evaluators/javascript/evaluator.go:14`

Implication: the move set can be sharply targeted without uprooting generic REPL infrastructure.

## Finding 2: Bobatea REPL has already abstracted language-specific features via generic widgets

Completion/help/help-drawer flows are now adapters over generic widget packages.

Evidence:

- Completion widget adapter: `bobatea/pkg/repl/completion_model.go:64`
- Help bar widget adapter: `bobatea/pkg/repl/helpbar_model.go:34`
- Help drawer widget adapter: `bobatea/pkg/repl/helpdrawer_model.go:43`

Implication: these generic widgets should stay where they are and be consumed by go-go-goja features.

## Finding 3: smalltalk-inspector REPL currently has no generic widget integration

Current REPL in smalltalk-inspector:

- single-line `textinput`
- synchronous eval submit on enter
- no completion/help drawer/bar.

Evidence:

- REPL input construction: `go-go-goja/cmd/smalltalk-inspector/app/model.go:127`
- REPL key handler: `go-go-goja/cmd/smalltalk-inspector/app/update.go:477`
- REPL rendering path: `go-go-goja/cmd/smalltalk-inspector/app/view.go:547`

Implication: biggest UX gain is integrating generic Bobatea widgets into this REPL pane.

## Target Architecture Proposal

## Ownership split

### Keep in Bobatea (generic/shared)

- REPL framework and evaluator interfaces.
- Generic completion/context widgets.
- Command palette, overlay, timeline, event bus, mode-keymap, and other reusable TUI primitives.

### Move to go-go-goja (JS-specific)

- JavaScript/Goja evaluator implementation currently in Bobatea.
- JS-specific REPL command entrypoint and docs/examples ownership.
- JS-specific integration tests for parser-aware completion and contextual help behavior.

## Package shape recommendation

Recommended new package in go-go-goja:

- `go-go-goja/pkg/jsrepl/evaluator`

Recommended adapter strategy:

- Core evaluator API in go-go-goja should avoid hard dependency on Bobatea types.
- Add thin adapter package in go-go-goja for Bobatea REPL integration (can live under `pkg/jsrepl/bobateaadapter` or command-local code).

Reasoning:

- Keeps go-go-goja core reusable without UI dependencies.
- Still allows straightforward consumption by Bobatea generic REPL.
- Minimizes circular ownership and makes future non-TUI usage easier.

## Migration Plan (Phased, Low-Risk)

## Phase 0: Contract freeze and test baseline

Actions:

- Snapshot current behavior with tests in both modules.
- Capture current completion/help expectations from JS evaluator tests.

Key files:

- `bobatea/pkg/repl/evaluators/javascript/evaluator_test.go`
- `bobatea/pkg/repl/js_help_bar_integration_test.go`

Exit criteria:

- All current evaluator tests green before relocation work.

## Phase 1: Move JS evaluator code to go-go-goja

Actions:

- Create new evaluator package in go-go-goja and move code logic from:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go`
- Move or duplicate tests into go-go-goja test tree.
- Keep API parity for first pass.

Important compatibility choices:

- Keep feature toggles (`EnableModules`, `EnableCallLog`, etc.).
- Preserve completion/help-drawer semantics and request/result shapes.

Exit criteria:

- New go-go-goja evaluator package has equivalent behavior and test coverage.

## Phase 2: Introduce Bobatea REPL adapter from go-go-goja side

Actions:

- Implement adapter implementing Bobatea REPL interfaces:
  - `repl.Evaluator`
  - optionally `repl.InputCompleter`, `repl.HelpBarProvider`, `repl.HelpDrawerProvider`
- Adapter delegates to moved go-go-goja JS evaluator core.

Why adapter not direct type alias:

- Prevents core evaluator package from importing Bobatea types.
- Keeps interface glue isolated.

Exit criteria:

- Bobatea REPL can run with go-go-goja-owned evaluator through adapter.

## Phase 3: Move `js-repl` command ownership to go-go-goja

Actions:

- Port `bobatea/examples/js-repl/main.go` into `go-go-goja/cmd/js-repl/main.go`.
- Reuse Bobatea REPL + timeline/eventbus packages for rich TUI behavior.
- Keep current `go-go-goja/cmd/repl` (line REPL) unchanged to avoid breaking existing workflows.

Reference source for port:

- `bobatea/examples/js-repl/main.go:36`

Exit criteria:

- `go-go-goja` exposes a first-class JS TUI REPL command.
- Bobatea no longer owns the JS REPL executable example.

## Phase 4: Remove JS evaluator from Bobatea

Actions:

- Delete or deprecate:
  - `bobatea/pkg/repl/evaluators/javascript`
- Update Bobatea tests to avoid hard dependency on moved evaluator.
- Move JS-specific integration tests to go-go-goja.

Exit criteria:

- Bobatea module no longer imports go-go-goja in production package code due JS evaluator.
- Bobatea remains generic widget/REPL infra.

## Phase 5: smalltalk-inspector REPL enhancement using Bobatea generic widgets

Actions:

- Integrate `suggest.Widget` for completion in `FocusRepl` path.
- Integrate `contextbar.Widget` for one-line symbol insight.
- Integrate `contextpanel.Widget` for detailed drawer.
- Optionally integrate `commandpalette.Model` for command discoverability.

Minimal integration hooks:

- In `handleReplKey` (`go-go-goja/cmd/smalltalk-inspector/app/update.go:477`):
  - buffer-change debounce scheduling
  - shortcut triggers
  - overlay navigation precedence
- In `renderReplArea` (`go-go-goja/cmd/smalltalk-inspector/app/view.go:547`):
  - render bar/panel/popup overlays under current width constraints

Exit criteria:

- Inspector REPL has parser-aware completion/help parity with js-repl.
- No regression in existing inspector panes/focus navigation.

## Detailed File-Level Move Map

## Move from Bobatea to go-go-goja

Primary code moves:

- `bobatea/pkg/repl/evaluators/javascript/evaluator.go`
- `bobatea/pkg/repl/evaluators/javascript/evaluator_test.go`
- `bobatea/pkg/repl/evaluators/javascript/example_test.go`

Likely move/replace:

- `bobatea/examples/js-repl/main.go` -> `go-go-goja/cmd/js-repl/main.go`

Likely test relocation:

- `bobatea/pkg/repl/js_help_bar_integration_test.go` (or equivalent coverage in go-go-goja).

## Keep in Bobatea unchanged

- `bobatea/pkg/repl/*` (core framework, config, models)
- `bobatea/pkg/tui/widgets/*`
- `bobatea/pkg/mode-keymap`
- `bobatea/pkg/timeline/*`
- `bobatea/pkg/eventbus/*`

## Add in go-go-goja

- New JS evaluator core package (go-go-goja owned).
- Optional Bobatea REPL adapter package.
- New `cmd/js-repl` command.
- New inspector REPL integration glue for suggest/context widgets.

## smalltalk-inspector Integration Design (Practical)

## Why not replace inspector REPL with full Bobatea REPL model

`smalltalk-inspector` is currently a tightly integrated four-pane application with shared focus and source/member synchronization.

- Global/member/source/repl focus and mode transitions are custom and interconnected:
  - `go-go-goja/cmd/smalltalk-inspector/app/keymap.go:29`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go:243`
- Full-model replacement with Bobatea REPL would require major layout/state orchestration rewrite.

Recommendation:

- Keep inspector model as host shell.
- Integrate generic widgets into existing REPL sub-pane first.

## Suggested inspector-side interfaces

Introduce thin local provider adapters that call go-go-goja JS evaluator/session services:

- `InspectorReplCompleter` -> satisfies `suggest.Provider`
- `InspectorReplContextBarProvider` -> satisfies `contextbar.Provider`
- `InspectorReplContextPanelProvider` -> satisfies `contextpanel.Provider`

These providers should use:

- current REPL input buffer
- cursor position
- parsed source + runtime state (`inspectorapi`, `runtime.Session`, `jsparse`)

Outcome:

- Preserve current inspector architecture.
- Gain Bobatea generic widget capabilities incrementally.

## Risk Register and Mitigations

## Risk 1: Hidden behavior drift during evaluator move

- Symptom: completion/help output subtly changes after relocation.
- Mitigation:
  - preserve current tests and add parity tests before deleting old package.
  - compare candidate lists and help text snapshots in regression tests.

## Risk 2: Module dependency churn and potential cyclic ownership confusion

- Symptom: unclear boundaries if evaluator core directly imports Bobatea contracts.
- Mitigation:
  - keep evaluator core contracts local in go-go-goja.
  - isolate Bobatea interface adapter in a dedicated bridge package.

## Risk 3: Inspector REPL integration complexity

- Symptom: key routing conflicts (completion nav vs pane nav vs command mode).
- Mitigation:
  - enforce strict key-routing order in `handleReplKey`.
  - mirror precedence strategy already used in Bobatea REPL (`command palette`, `help drawer`, `completion`).

## Risk 4: UI overlay clipping in small terminals

- Symptom: help drawer/completion overlap with existing panes.
- Mitigation:
  - reuse widget overlay layout APIs and clamp behavior.
  - add viewport-size integration tests for narrow widths.

## Testing Strategy (Post-Migration)

## Unit tests

- Moved evaluator tests in go-go-goja:
  - evaluate/reset/module/help/completion behaviors.
- Adapter contract tests:
  - ensure Bobatea request/result types map cleanly.

## Integration tests

- `cmd/js-repl` smoke test with scripted input where feasible.
- smalltalk-inspector REPL widget routing tests:
  - completion open/apply/cancel
  - contextbar debounce visibility
  - drawer toggle/pin/refresh

## Manual validation script

- Load file in smalltalk inspector.
- Evaluate bindings/functions in REPL.
- Verify:
  - globals list refresh
  - REPL source highlighting
  - completion suggestions
  - help bar/drawer content.

## Proposed Execution Sequence (Concrete)

1. Create go-go-goja JS evaluator core package and copy logic from Bobatea JS evaluator.
2. Port tests and make them pass in go-go-goja.
3. Build Bobatea adapter package from go-go-goja side.
4. Add `go-go-goja/cmd/js-repl` using Bobatea REPL + moved evaluator adapter.
5. Update docs/help in go-go-goja to reference new js-repl command.
6. Integrate suggest/context widgets into `smalltalk-inspector` REPL pane.
7. Remove Bobatea JS evaluator package and relocate/replace remaining JS-specific tests.
8. Clean go.mod dependencies and run full test suite for both modules.

## What Should Not Move

To avoid over-coupling and unnecessary churn, do not move these from Bobatea into go-go-goja in GOJA-036:

- `pkg/repl` generic framework
- `pkg/tui/widgets/suggest`
- `pkg/tui/widgets/contextbar`
- `pkg/tui/widgets/contextpanel`
- `pkg/commandpalette`
- `pkg/mode-keymap`
- `pkg/overlay`
- `pkg/timeline` and `pkg/eventbus`

These are generic and already appropriately scoped as shared UI infrastructure.

## Final Recommendation

Adopt a strict ownership rule:

- Goja/JS semantics and evaluator intelligence live in go-go-goja.
- Generic REPL/TUI widgets stay in Bobatea.

Implement with adapter boundaries, not framework rewrites.

This gives:

- clean architectural ownership,
- low migration risk,
- immediate reuse of mature generic widgets,
- a practical path to improve smalltalk-inspector REPL without destabilizing inspector core.

## Appendix A: Key Evidence References

- Bobatea JS evaluator imports go-go-goja runtime/parser:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:13`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:14`
- Bobatea REPL wiring generic widgets:
  - `bobatea/pkg/repl/completion_model.go:64`
  - `bobatea/pkg/repl/helpbar_model.go:34`
  - `bobatea/pkg/repl/helpdrawer_model.go:43`
- Smalltalk inspector REPL implementation points:
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go:127`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go:477`
  - `go-go-goja/cmd/smalltalk-inspector/app/view.go:547`
- go-go-goja currently imports Bobatea mode-keymap only in app code:
  - `go-go-goja/cmd/inspector/app/model.go:18`
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go:15`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go:11`

## Appendix B: Script Artifacts for This Analysis

- Boundary scan script:
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/analyze_bobatea_goja_boundary.sh`
- Widget matrix script:
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/widget_reuse_matrix.sh`
- Generated outputs:
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/out/bobatea_pkg_summary.tsv`
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/out/widget_reuse_matrix.tsv`
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/out/go_go_goja_imports_from_bobatea.txt`
  - `go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/out/go_go_goja_bobatea_widget_imports.tsv`


## Appendix C: Exhaustive Package Matrix (Keep/Move/Reuse)

This appendix expands every relevant package decision into explicit actionability.

### `bobatea/pkg/repl`

- Role:
  - Generic REPL composition shell with evaluator abstraction.
  - Timeline integration, focus handling, keymap routing, history, command palette integration.
- Core evidence:
  - `bobatea/pkg/repl/model.go:21`
  - `bobatea/pkg/repl/model.go:60`
  - `bobatea/pkg/repl/model.go:80`
- Coupling profile:
  - No direct Goja/go-go-goja imports in core package.
  - Language-specific behavior delegated through interfaces.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - No move.
  - Ensure moved JS evaluator is consumable through Bobatea interfaces via adapter.

### `bobatea/pkg/repl/evaluators/javascript`

- Role:
  - Goja runtime evaluator with jsparse-driven completion/help and module features.
- Core evidence:
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:10`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:13`
  - `bobatea/pkg/repl/evaluators/javascript/evaluator.go:14`
- Coupling profile:
  - Strong JS-specific and go-go-goja-specific dependencies.
- Decision:
  - Move to go-go-goja.
- Action for GOJA-036:
  - Relocate package logic to go-go-goja-owned path.
  - Port tests and examples.

### `bobatea/pkg/tui/widgets/suggest`

- Role:
  - Debounced async completion behavior with popup layout and keyboard actions.
- Core evidence:
  - `bobatea/pkg/tui/widgets/suggest/widget.go:11`
  - `bobatea/pkg/tui/widgets/suggest/widget.go:109`
  - `bobatea/pkg/tui/widgets/suggest/widget.go:214`
- Coupling profile:
  - Generic provider API.
- Decision:
  - Keep in Bobatea; consume in go-go-goja inspector.
- Action for GOJA-036:
  - Add `suggest.Widget` to smalltalk inspector REPL state and render lifecycle.

### `bobatea/pkg/tui/widgets/contextbar`

- Role:
  - One-line context help pipeline with debounce and timeout.
- Core evidence:
  - `bobatea/pkg/tui/widgets/contextbar/widget.go:12`
  - `bobatea/pkg/tui/widgets/contextbar/widget.go:95`
  - `bobatea/pkg/tui/widgets/contextbar/widget.go:158`
- Coupling profile:
  - Generic provider API.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Add context bar to smalltalk inspector REPL for quick symbol signatures.

### `bobatea/pkg/tui/widgets/contextpanel`

- Role:
  - Toggleable rich context panel with pin/refresh and dock positions.
- Core evidence:
  - `bobatea/pkg/tui/widgets/contextpanel/widget.go:11`
  - `bobatea/pkg/tui/widgets/contextpanel/widget.go:191`
  - `bobatea/pkg/tui/widgets/contextpanel/types.go:11`
- Coupling profile:
  - Generic provider API and generic layout model.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Add help drawer/panel to smalltalk inspector REPL in incremental phase.

### `bobatea/pkg/commandpalette`

- Role:
  - Searchable command execution palette.
- Core evidence:
  - `bobatea/pkg/commandpalette/model.go:12`
  - `bobatea/pkg/commandpalette/model.go:124`
- Coupling profile:
  - Fully generic.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Optional inspector command UX upgrade to palette model.

### `bobatea/pkg/mode-keymap`

- Role:
  - Reflection-based key binding enable/disable by mode tags.
- Core evidence:
  - `bobatea/pkg/mode-keymap/mode-keymap.go:47`
  - `bobatea/pkg/mode-keymap/mode-keymap.go:136`
- Coupling profile:
  - Generic utility.
- Current go-go-goja usage:
  - `go-go-goja/cmd/inspector/app/model.go:18`
  - `go-go-goja/cmd/smalltalk-inspector/app/model.go:15`
  - `go-go-goja/cmd/smalltalk-inspector/app/update.go:11`
- Decision:
  - Keep in Bobatea and continue reusing.
- Action for GOJA-036:
  - No move.

### `bobatea/pkg/timeline` + `bobatea/pkg/eventbus`

- Role:
  - Generic event-driven transcript infrastructure used by Bobatea REPL.
- Core evidence:
  - `bobatea/pkg/timeline/shell.go:12`
  - `bobatea/pkg/timeline/controller.go:12`
  - `bobatea/pkg/eventbus/eventbus.go:19`
- Coupling profile:
  - Generic rendering/events; no JS semantics.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Reuse for `go-go-goja/cmd/js-repl` command.

### `bobatea/pkg/overlay`

- Role:
  - Overlay compositing utility for terminal text layers.
- Core evidence:
  - `bobatea/pkg/overlay/overlay.go:43`
- Coupling profile:
  - Generic rendering helper.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Use when overlaying context panel/suggest popup in inspector if needed.

### `bobatea/pkg/listbox`

- Role:
  - Lightweight selectable list component.
- Core evidence:
  - `bobatea/pkg/listbox/listbox.go:25`
- Coupling profile:
  - Generic.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Optional later refactor, not critical path.

### `bobatea/pkg/textarea`

- Role:
  - Rich multiline text editor widget with memoization optimization.
- Core evidence:
  - `bobatea/pkg/textarea/textarea.go:158`
- Coupling profile:
  - Generic.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Optional multiline REPL upgrade for inspector.

### `bobatea/pkg/filepicker`, `pkg/sparkline`, `pkg/diff`, `pkg/buttons`, `pkg/chat`

- Role:
  - Domain-generic utility widgets and screens.
- Coupling profile:
  - No JS-specific ownership concerns.
- Decision:
  - Keep in Bobatea.
- Action for GOJA-036:
  - Out of critical path.

## Appendix D: Concrete Adapter and Integration Sketches

## D1) go-go-goja evaluator core contract sketch

```go
// go-go-goja/pkg/jsrepl/core/types.go
package core

type CompletionRequest struct {
    Input      string
    CursorByte int
    Reason     string
    Shortcut   string
    RequestID  uint64
}

type CompletionResult struct {
    Show        bool
    Suggestions []Suggestion
    ReplaceFrom int
    ReplaceTo   int
}

type HelpBarRequest struct {
    Input      string
    CursorByte int
    Reason     string
    Shortcut   string
    RequestID  uint64
}

type HelpBarPayload struct {
    Show      bool
    Text      string
    Kind      string
    Severity  string
    Ephemeral bool
}
```

```go
// go-go-goja/pkg/jsrepl/core/evaluator.go
package core

type Evaluator interface {
    Evaluate(ctx context.Context, code string) (string, error)
    CompleteInput(ctx context.Context, req CompletionRequest) (CompletionResult, error)
    GetHelpBar(ctx context.Context, req HelpBarRequest) (HelpBarPayload, error)
    GetHelpDrawer(ctx context.Context, req HelpDrawerRequest) (HelpDrawerDocument, error)
}
```

## D2) Bobatea adapter sketch (go-go-goja-owned bridge)

```go
// go-go-goja/pkg/jsrepl/bobateaadapter/adapter.go
package bobateaadapter

import (
    bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
    "github.com/go-go-golems/go-go-goja/pkg/jsrepl/core"
)

type Adapter struct {
    Core core.Evaluator
}

func (a *Adapter) EvaluateStream(ctx context.Context, code string, emit func(bobarepl.Event)) error {
    out, err := a.Core.Evaluate(ctx, code)
    if err != nil {
        emit(bobarepl.Event{Kind: bobarepl.EventResultMarkdown, Props: map[string]any{"markdown": "Error: " + err.Error()}})
        return nil
    }
    emit(bobarepl.Event{Kind: bobarepl.EventResultMarkdown, Props: map[string]any{"markdown": out}})
    return nil
}

func (a *Adapter) CompleteInput(ctx context.Context, req bobarepl.CompletionRequest) (bobarepl.CompletionResult, error) {
    // map request to core, map result back
}

func (a *Adapter) GetHelpBar(ctx context.Context, req bobarepl.HelpBarRequest) (bobarepl.HelpBarPayload, error) {
    // map request to core, map result back
}

func (a *Adapter) GetHelpDrawer(ctx context.Context, req bobarepl.HelpDrawerRequest) (bobarepl.HelpDrawerDocument, error) {
    // map request to core, map result back
}
```

## D3) smalltalk-inspector REPL integration sketch

```go
// go-go-goja/cmd/smalltalk-inspector/app/model.go
// Add fields
suggestWidget     *suggest.Widget
contextBarWidget  *contextbar.Widget
contextPanelWidget *contextpanel.Widget
```

```go
// go-go-goja/cmd/smalltalk-inspector/app/update.go
func (m Model) handleReplKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    prevValue := m.replInput.Value()
    prevCursor := m.replInput.Position()

    // 1) command palette shortcut if added
    // 2) context panel shortcuts
    // 3) completion popup nav routing
    // 4) submit / escape / fallback input update

    var cmd tea.Cmd
    m.replInput, cmd = m.replInput.Update(msg)

    debounceCmds := []tea.Cmd{
        m.suggestWidget.OnBufferChanged(prevValue, prevCursor, m.replInput.Value(), m.replInput.Position()),
        m.contextBarWidget.OnBufferChanged(prevValue, prevCursor, m.replInput.Value(), m.replInput.Position()),
        m.contextPanelWidget.OnBufferChanged(prevValue, prevCursor, m.replInput.Value(), m.replInput.Position()),
    }

    return m, tea.Batch(append([]tea.Cmd{cmd}, debounceCmds...)...)
}
```

```go
// go-go-goja/cmd/smalltalk-inspector/app/view.go
func (m Model) renderReplArea() string {
    // existing repl line rendering
    // append context bar view under prompt
    // use overlay/panel layout helpers to render drawer and completion popup
}
```

## Appendix E: Acceptance Criteria Per Phase

## Phase 1 acceptance (JS evaluator relocation)

- New go-go-goja evaluator package compiles and tests pass.
- Completion/help outputs match previous behavior for representative fixtures.
- Bobatea no longer needs JS evaluator package for core generic REPL tests.

## Phase 2 acceptance (adapter)

- Bobatea REPL runs using go-go-goja-owned JS evaluator adapter.
- Key actions in REPL still work:
  - evaluate expression
  - autocomplete open/accept/cancel
  - help bar updates
  - help drawer toggle/refresh.

## Phase 3 acceptance (`cmd/js-repl`)

- `go-go-goja/cmd/js-repl` provides same rich TUI flow as former Bobatea example.
- Existing `go-go-goja/cmd/repl` remains functional and unchanged.

## Phase 4 acceptance (cleanup)

- Bobatea production package code no longer imports go-go-goja for JS evaluator ownership.
- JS-specific tests live under go-go-goja where evaluator now lives.

## Phase 5 acceptance (inspector enhancement)

- smalltalk-inspector REPL supports completion/help widgets without breaking pane navigation.
- Existing inspector tests still pass; add REPL widget routing tests.

## Appendix F: Rollback Strategy

If migration hits instability, rollback should be phase-bounded:

- If Phase 1 fails:
  - Keep existing Bobatea JS evaluator untouched and pause move.
- If Phase 2 adapter causes regressions:
  - Keep new go-go-goja evaluator package but stop wiring it into Bobatea REPL until parity gaps close.
- If Phase 5 inspector integration destabilizes key routing:
  - Feature-flag widget integrations and default to old local REPL behavior.

Rollback files to isolate quickly:

- New go-go-goja evaluator/adapter files.
- New `go-go-goja/cmd/js-repl` command wiring.
- Inspector REPL integration points (`model.go`, `update.go`, `view.go`).

This allows preserving shipped behavior while continuing migration behind flags.

## Appendix G: Suggested Task Breakdown for GOJA-036

1. Extract JS evaluator core into go-go-goja package and port tests.
2. Add Bobatea adapter layer in go-go-goja for REPL interfaces.
3. Add `go-go-goja/cmd/js-repl` with Bobatea REPL + adapter.
4. Move/retire `bobatea/examples/js-repl` and update references.
5. Move JS-specific integration tests from Bobatea to go-go-goja.
6. Add smalltalk-inspector completion widget integration.
7. Add smalltalk-inspector context bar and context panel integrations.
8. Run cross-module regression suite and dependency cleanup.

