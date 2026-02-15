---
Title: 'tmux analysis: tree pane width/scroll and REPL symbol source regression'
Ticket: GOJA-035-INSPECTOR-UI-REGRESSIONS
Status: active
Topics:
    - go
    - goja
    - inspector
    - bugfix
    - ui
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Tree/source split layout and tree pane rendering details.
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: AST tree list delegate behavior and item shaping.
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: REPL source log, jump-to-source behavior, fallback logic.
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: REPL eval pipeline and REPL source tracking calls.
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model_members_test.go
      Note: Regression coverage for REPL binding source fallback.
    - Path: go-go-goja/cmd/inspector/app/model_test.go
      Note: Tree pane width and list title clamping coverage.
    - Path: go-go-goja/ttmp/2026/02/15/GOJA-035-INSPECTOR-UI-REGRESSIONS--inspector-ui-regressions-tree-pane-width-scroll-and-repl-source-mapping/scripts/repro-tmux-goja-035.sh
      Note: Repro automation script used for tmux captures.
    - Path: go-go-goja/ttmp/2026/02/15/GOJA-035-INSPECTOR-UI-REGRESSIONS--inspector-ui-regressions-tree-pane-width-scroll-and-repl-source-mapping/scripts/smalltalk-globals-bottom.txt
      Note: tmux capture proving REPL symbol selected while source pane still shows file source (pre-fix).
ExternalSources: []
Summary: Root-cause analysis and fix plan for two UI regressions: oversized/awkward inspector tree pane behavior and missing REPL source fallback for REPL-defined globals.
LastUpdated: 2026-02-15T17:44:00Z
WhatFor: Provide a reproducible, evidence-backed implementation guide to fix both regressions quickly and safely.
WhenToUse: Use when validating GOJA-035 behavior or reworking tree/list/source mapping in inspector UIs.
---

# GOJA-035 tmux Analysis

## Scope

This ticket covers two regressions:

1. `cmd/inspector`: AST tree pane feels too wide/awkward and scrolling quality degrades under long node labels.
2. `cmd/smalltalk-inspector`: selecting a REPL-defined global symbol no longer shows REPL source in the Source pane.

## Reproduction Setup

- Environment: tmux 3.4
- Commands used:
  - `go run ./cmd/inspector /tmp/long-tree.js`
  - `go run ./cmd/smalltalk-inspector testdata/inspector-test.js`
- Capture automation script: `scripts/repro-tmux-goja-035.sh`

## Evidence: Issue 1 (Tree Pane Width/Scroll)

From `scripts/inspector-narrow-initial.txt` (pre-fix):

```text
=== narrow initial ===
  4       deeplyNestedPropertyWithVerbos    ▼ LexicalDeclaration const
  5         message: "this is a ridiculo
  6       }
  7     }                                     ▶ Binding
  8   }
  9 };
 10                                         ▼ FunctionDeclaration "makeMonsterF…
 11 function makeMonsterFunctionWithExtr
 12   const localBindingWithLongName = p
 13   return {                                ▶ FunctionLiteral "makeMonsterFun…
```

From `scripts/inspector-narrow-scroll.txt` (pre-fix):

```text
=== narrow scroll ===
 13   return {                          │     ▶ BlockStatement
 14     localBindingWithLongName,       │
 15     nested: {
 16       desc: "another very long descr
```

Observations:

- Tree rows and source rows interleave visually in narrow tmux widths.
- Long row labels create clutter; visual boundaries are weak.
- With a 50/50 split, tree pane dominates real estate for a secondary pane.

## Evidence: Issue 2 (REPL Symbol Source Fallback)

From `scripts/smalltalk-globals-bottom.txt` (pre-fix):

```text
Globals ───────────────────── zzzReplFn: Members ────────── Source ────────────────────────────────────────────────────
...
▸ ●  zzzReplFn                                               13 class Shape {
...
```

From `scripts/smalltalk-after-enter-zzz.txt` (pre-fix):

```text
REPL Result: zzzReplFn ──────────────────────────────────── Source ────────────────────────────────────────────────────
▸ ·  length : 1  number                                      67     case "rectangle":
...
```

Observations:

- `zzzReplFn` exists in globals list (runtime merge works).
- Source pane remains on file source, not REPL source.
- Regression affects both plain selection and enter-to-inspect flow.

## Root Cause Analysis

### Issue 1

- `cmd/inspector/app/model.go` used a strict 50/50 source/tree split.
- Tree rows were displayed with default list delegate visuals and verbose labels; this increased noise in narrow terminals.
- Tree metadata section consumed several lines (`metaHeight=5`), reducing effective tree viewport and perceived scroll quality.

### Issue 2

- `cmd/smalltalk-inspector/app/model.go:jumpToSource()` always resets `showingReplSrc=false` before binding lookup.
- `jumpToBinding()` only looked for static declaration lines via `BindingDeclarationLine`.
- REPL-defined names have no file-backed declaration in static analysis, so lookup returns not found and source stays in file mode.
- `replSourceLog` tracked expressions but did not track declared symbol names, so no deterministic fallback from global name -> REPL source entry was possible.

## Fix Plan

1. **Smalltalk REPL source mapping**
- Track declared symbol names per REPL expression entry.
- Add fallback `showReplSourceForBinding(name)` that maps selected global names to REPL source log entries.
- Update `jumpToBinding` to use fallback when static declaration lookup fails.
- Ensure REPL source appends include parser-backed declarations.

2. **Inspector tree pane ergonomics**
- Reduce tree pane width share from 50% to ~40% (source gets ~60%).
- Disable description rendering in tree list delegate for cleaner rows.
- Clamp tree row titles with ellipsis to prevent long-label spillover.
- Reduce tree metadata height (4 -> 2 on constrained content) to increase visible tree rows.

## Validation Plan

1. Unit tests:
- `go test ./cmd/smalltalk-inspector/app ./cmd/inspector/app -count=1`
- Add explicit regression test for REPL fallback mapping.
- Add explicit tests for tree pane width policy and title clamping.

2. tmux manual validation:
- Re-run `scripts/repro-tmux-goja-035.sh`.
- Confirm `Source (REPL)` appears when selecting REPL-defined global.
- Confirm tree pane is narrower and long labels clamp cleanly.

## Acceptance Criteria

- Selecting REPL-defined globals shows REPL source context in Source pane.
- Enter on REPL-defined function keeps REPL source context visible.
- Tree pane no longer dominates layout at typical tmux widths.
- Long tree labels are clamped and easier to scan; scrolling remains stable.
