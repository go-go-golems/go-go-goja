---
Title: Inspector Cleanup Review (GOJA-024 to GOJA-027)
Ticket: GOJA-028-CLEANUP-INSPECTOR
Status: active
Topics:
    - go
    - goja
    - tui
    - inspector
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Core state, globals/members logic, runtime merge, REPL source tracking
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: Message routing, key handling, eval flow, inspect/stack navigation
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Rendering pipeline and pane-specific UI behavior
    - Path: go-go-goja/cmd/smalltalk-inspector/app/keymap.go
      Note: Current keymap and help model integration constraints
    - Path: go-go-goja/pkg/inspector/runtime/introspect.go
      Note: Runtime object/property/prototype helpers reused by UI
    - Path: go-go-goja/pkg/inspector/runtime/errors.go
      Note: Stack trace parsing implementation and portability caveats
    - Path: go-go-goja/pkg/inspector/runtime/function_map.go
      Note: Runtime-function to static-source mapping strategy
    - Path: go-go-goja/pkg/jsparse/highlight.go
      Note: Syntax span model and character-class lookup path
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: GOJA-025 reusable component baseline for list/table/viewport/help/spinner/textinput
    - Path: go-go-goja/cmd/inspector/app/keymap.go
      Note: mode-keymap tags and mode-aware key binding model
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: Existing reusable list adapter pattern
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/reference/02-smalltalk-goja-inspector-interface-and-component-design.md
      Note: Original implementation blueprint and intended component decomposition
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-025-INSPECTOR-BUBBLES-REFACTOR--inspector-bubbles-component-refactor/reference/01-inspector-refactor-design-guide.md
      Note: Refactor baseline and reusable component decisions
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-026-INSPECTOR-BUGS--smalltalk-inspector-bug-fixes-empty-members-and-repl-definitions/reference/02-bug-report.md
      Note: Bug set and intended remediation details
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-026-INSPECTOR-BUGS--smalltalk-inspector-bug-fixes-empty-members-and-repl-definitions/design/01-fix-plan.md
      Note: Bug-fix sequencing and rationale
ExternalSources: []
Summary: Deep technical review of inspector work from GOJA-024 through GOJA-027, including architecture analysis, severity-ranked findings, and cleanup/refactor roadmap.
LastUpdated: 2026-02-15T17:10:00Z
WhatFor: Give implementers and maintainers a concrete, evidence-backed cleanup plan and architecture guide for the smalltalk-inspector code path.
WhenToUse: Use before further feature work on smalltalk-inspector or when planning refactors/shared component extraction.
---

# Inspector Cleanup Review (GOJA-024 to GOJA-027)

## Historical Status (Updated 2026-02-15)

This review is a historical snapshot from the pre-`pkg/inspectorapi` cutover phase. Several findings were intentionally resolved by later tickets (notably GOJA-029, GOJA-030, GOJA-034) that moved adapter orchestration into `pkg/inspectorapi` and integrated extracted inspector packages.

## Findings First (Severity-Ordered)

### Critical Findings

1. **Self-referential inheritance can crash the process via infinite recursion**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/model.go:345`, `go-go-goja/cmd/smalltalk-inspector/app/model.go:385`, `go-go-goja/cmd/smalltalk-inspector/app/model.go:387`  
   Evidence: `addInheritedMembers` recurses without cycle detection; `class A extends A {}` reproduces a stack overflow.  
   Reproduction run during this review:
   - `go test ./cmd/smalltalk-inspector/app -run TestTmpSelfExtends -count=1`
   - Runtime output included `fatal error: stack overflow` and recursive stack frames in `addInheritedMembers`.  
   Impact: any malformed/hostile input can hard-crash the inspector session.

2. **The main UI command has zero package-level tests despite high complexity**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app` (no `_test.go` files), `go test` output shows `[no test files]`.  
   Evidence: `go test ./cmd/smalltalk-inspector/... -count=1` reports no tests in command/app packages.  
   Impact: regressions in navigation, rendering, and cross-pane state transitions are currently caught only manually.

### High Findings

3. **Inspect and stack views do not support scrolling windows; selection can move off-screen**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/view.go:175`, `go-go-goja/cmd/smalltalk-inspector/app/view.go:253`, `go-go-goja/cmd/smalltalk-inspector/app/update.go:547`, `go-go-goja/cmd/smalltalk-inspector/app/update.go:639`  
   Evidence: render loops always start at index `0`; `inspectIdx`/`stackIdx` can exceed visible rows, but no scroll offset exists.  
   Impact: for large objects/stacks, users can navigate to items they cannot see.

4. **Syntax highlighting path is asymptotically expensive and likely to degrade on larger files**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/view.go:530`, `go-go-goja/pkg/jsparse/highlight.go:92`  
   Evidence: `renderSyntaxLine` iterates characters, and each character performs a full linear scan of all spans via `SyntaxClassAt`.  
   Complexity: roughly `O(chars_per_line * spans)` per line render.  
   Impact: scroll/render latency growth with file size and REPL history size.

5. **GOJA-025 reusable component baseline was not reused in GOJA-024/026/027 implementation path**  
   Where intended: `go-go-goja/cmd/inspector/app/keymap.go`, `go-go-goja/cmd/inspector/app/tree_list.go`, `go-go-goja/cmd/inspector/app/model.go`  
   Where implemented: `go-go-goja/cmd/smalltalk-inspector/app/{model,update,view,keymap}.go`  
   Evidence: no imports of `bubbles/list`, `bubbles/table`, `bubbles/viewport`, or `mode-keymap` in smalltalk-inspector app code (except a comment in keymap).  
   Impact: significant divergence, duplicate state/render logic, and reduced maintainability.

### Medium Findings

6. **Source scrolling logic uses file source length even when REPL source is active**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/update.go:391`, `go-go-goja/cmd/smalltalk-inspector/app/update.go:412`, `go-go-goja/cmd/smalltalk-inspector/app/model.go:727`  
   Evidence: `handleSourceKey` bounds to `len(m.sourceLines)`; REPL mode displays `replSourceLines`.  
   Impact: inconsistent scroll behavior when browsing REPL-rendered source.

7. **Command parsing for `:load` is not path-safe for filenames with spaces**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/update.go:498` to `go-go-goja/cmd/smalltalk-inspector/app/update.go:513`  
   Evidence: `strings.Fields` splits path tokens; no quoted path handling.  
   Impact: common filesystem paths can fail unexpectedly.

8. **Stack parser is brittle and non-portable for colon-bearing filenames (e.g., Windows paths)**  
   Where: `go-go-goja/pkg/inspector/runtime/errors.go:50`, `go-go-goja/pkg/inspector/runtime/errors.go:51`  
   Evidence: regex uses `([^:]+)` for filename; fails for `C:\...` and similar forms.  
   Impact: incorrect stack-frame parsing in cross-platform contexts.

9. **Magic numeric binding kinds in rendering are fragile and non-idiomatic**  
   Where: `go-go-goja/cmd/smalltalk-inspector/app/view.go:324`, `go-go-goja/cmd/smalltalk-inspector/app/view.go:327`  
   Evidence: literal `case 4` and `case 3` used instead of `jsparse.BindingClass`/`jsparse.BindingFunction`.  
   Impact: silent break risk if enum definitions change.

10. **REPL function-source fallback appends lines without rebuilding syntax spans**  
    Where: `go-go-goja/cmd/smalltalk-inspector/app/model.go:716` to `go-go-goja/cmd/smalltalk-inspector/app/model.go:724`  
    Evidence: path appends to `replSourceLines` but never calls `rebuildReplSyntaxSpans`.  
    Impact: inconsistent highlight behavior after fallback source injection.

11. **Runtime globals merge has correctness edge cases and stale-state behavior**  
    Where: `go-go-goja/cmd/smalltalk-inspector/app/model.go:796` to `go-go-goja/cmd/smalltalk-inspector/app/model.go:922`  
    Evidence:
    - `extractDeclaredNames` is token-based heuristic (not parser-based).
    - tracked names are added as `BindingConst` regardless of actual declaration kind.
    - no removal path for deleted bindings.
    Impact: global list can drift from actual runtime semantics over long sessions.

12. **`pkg/inspector/analysis` was not integrated in the reviewed snapshot (now superseded)**  
    Where at review time: `go-go-goja/pkg/inspector/analysis/*.go` with no references from `go-go-goja/cmd/smalltalk-inspector/app`  
    Status at update time (2026-02-15): superseded by the `pkg/inspectorapi` cutover and subsequent integration work.

### Low Findings

13. **Local helper duplication (`minInt`, `maxInt`, `padRight`) across command/runtime layers**  
    Where: `go-go-goja/cmd/smalltalk-inspector/app/model.go:781`, `go-go-goja/cmd/smalltalk-inspector/app/view.go:609`, `go-go-goja/pkg/inspector/runtime/session.go:124`, `go-go-goja/cmd/inspector/app/model.go:1379`  
    Impact: minor duplication noise; low immediate risk.

14. **Some ticket docs are incomplete relative to actual implementation status**  
    Where: `go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/changelog.md`, `go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/tasks.md`  
    Evidence: minimal changelog; one validation task remains unchecked despite code merged.  
    Impact: reduces long-term traceability and operational confidence.

15. **Known docmgr doctor warning remains for imported raw source without frontmatter**  
    Where: `go-go-goja/ttmp/2026/02/14/GOJA-024-SMALLTALK-INSPECTOR--smalltalk-inspector/sources/local/smalltalk-goja-inspector.md`  
    Impact: low; acknowledged intentional exception, but should be documented as policy.

---

## Scope and Methodology

This review covers work created from GOJA-024 onward:

- GOJA-024: design + implementation plan for smalltalk-inspector.
- GOJA-025: reusable component refactor in `cmd/inspector`.
- GOJA-026: bug-fix wave for smalltalk-inspector behavior.
- GOJA-027: syntax highlighting for source pane.

Evidence basis used for this review:

1. Code and docs inventory across ticket workspaces.
2. Commit chronology and touched-file stats.
3. Direct file-level inspection with line references.
4. Runtime validation:
   - `go test ./... -count=1` (green)
   - targeted reproduction of crash in inheritance recursion.

Quantitative context:

- Diff from `b1e9add` (pre-ticket baseline) to `HEAD`: **52 files changed, 8318 insertions, 115 deletions**.
- Non-ticket (`non-ttmp`) code/doc changes: **23 files, 4168 insertions, 115 deletions**.
- New smalltalk-inspector command footprint: ~**2469 LOC** in app package + main.
- New `pkg/inspector` footprint: ~**971 LOC**.

---

## What Was Built (By Ticket)

## GOJA-024 (Design Ticket)

Primary output was a detailed implementation and component-design blueprint. It established the intended decomposition and recommended reuse of GOJA-025 components (help/list/table/viewport/spinner/textinput/mode-keymap).

Strength:

- High-quality specification and screen-level mapping.

Gap:

- The implemented command path diverged from this decomposition and did not fully adopt the reusable baseline.

## GOJA-025 (Reusable Inspector Refactor)

Refactored `cmd/inspector` to introduce reusable Bubbles patterns and mode-aware key metadata.

What landed:

- `cmd/inspector/app/keymap.go`: mode tags and help integration.
- `cmd/inspector/app/tree_list.go`: list adapter for tree rows.
- `cmd/inspector/app/model.go`: spinner/help, list/viewport/table/textinput integration.

Assessment:

- Good foundational work and test-preserving refactor.
- This is the strongest candidate base for extracting shared components.

## GOJA-026 (Bug-Fix Wave)

Delivered meaningful behavior fixes in smalltalk-inspector:

- value globals get runtime member introspection.
- REPL-defined globals are merged into globals list.
- Enter-on-value opens inspect mode.
- runtime init ordering corrected.
- prototype chain footer improved.
- nav stack reset on new eval.

Assessment:

- Functional correctness improved significantly.
- Several fixes were tactical patches inside the existing monolith.

## GOJA-027 (Syntax Highlighting)

Added tree-sitter based syntax spans and rendering support for file + REPL source.

What landed:

- shared syntax types in `pkg/jsparse/highlight.go`.
- integration into smalltalk source pane rendering.

Assessment:

- Feature quality is visible and useful.
- Performance model and test coverage need hardening.

---

## How It Works Today (Architecture Map)

### Command Layer

`cmd/smalltalk-inspector/main.go` launches Bubble Tea alt-screen with a single root model.

### Root UI State/Flow

`cmd/smalltalk-inspector/app/model.go`, `update.go`, `view.go` currently implement a monolithic architecture:

- State store: globals/members/source/repl/inspect/stack and navigation state in one struct.
- Update flow: one large update router with pane-specific handlers.
- View flow: large render switch for empty/normal/inspect/error layouts.

### Domain Helpers

- `pkg/inspector/runtime/*`: runtime eval/introspection/stack parsing/function mapping.
- `pkg/inspector/analysis/*`: wrappers for static symbol/xref workflows (present, mostly unused by smalltalk command).

### Static Parsing/Highlighting

- `pkg/jsparse` handles parse/index/resolve and syntax spans.

### Reusable Baseline Not Yet Unified

`cmd/inspector/app/*` now includes reusable patterns (mode-keymap, list adapter, viewport, table, command input), but smalltalk-inspector re-implements many equivalents locally.

---

## Deep Technical Review

### 1) Correctness and Safety

#### 1.1 Inheritance recursion needs cycle guards

`addInheritedMembers` should track visited class names and stop recursion on repeated nodes. Without this, self-cycles and indirect cycles can overflow stack.

Recommended shape:

```go
func (m *Model) addInheritedMembers(className, source string, visited map[string]bool) {
    if visited[className] { return }
    visited[className] = true
    defer delete(visited, className)
    ...
}
```

#### 1.2 Mode transition correctness is mostly good, but render/selection coupling is brittle

Navigation state (`inspectIdx`, `stackIdx`) is not coupled to viewport offsets. This creates invisible selection states.

Recommended shape:

- Introduce `inspectScroll` and `stackScroll` with `ensureInspectVisible` / `ensureStackVisible`.
- Render loops should start from scroll index instead of `0`.

### 2) Architecture and Decomposition

#### 2.1 Monolith drift

Despite earlier design goals, most behavior still lives in three large files (`model.go` 932 lines, `update.go` 667 lines, `view.go` 615 lines).

Refactor direction:

- Keep a thin root model for orchestration.
- Split feature models:
  - globals/members/source/repl/inspect/stack/status.
- Reuse `cmd/inspector` adapters where possible.

#### 2.2 Unused domain package (`pkg/inspector/analysis`)

The package was correctly added but not integrated. This causes duplicated static-analysis access patterns in command-layer logic.

Refactor direction:

- Route binding/xref/symbol responsibilities through `pkg/inspector/analysis`.
- Keep UI layer focused on rendering and user interaction.

### 3) Component Reuse and Generalization Opportunities

High leverage opportunities:

1. **Mode-keymap bridge component**
- General-purpose wrapper used by both `cmd/inspector` and `cmd/smalltalk-inspector`.
- Avoid duplicated key-mode logic and help wiring.

2. **Selectable list model abstraction**
- General-purpose scrolling/select window logic shared by globals/members/inspect/stack.
- Could wrap `bubbles/list` with lightweight domain adapters.

3. **Source pane component**
- Shared code-view component with:
  - source buffer abstraction (file vs REPL)
  - highlight span cache
  - jump/center APIs
  - viewport-backed scrolling.

4. **Inspector object browser component**
- reusable property-table/list with symbol/property descriptors, proto jump, and drill-in stack.

5. **Status + command line module**
- standard status notices and command parsing including quoted arguments.

### 4) Performance Characteristics

#### 4.1 Highlighting path

Current char-by-char + span-scan model is simple but slow for larger inputs.

Recommended upgrade:

- Build per-line span slices once, then render by segments rather than per-char global lookup.
- Precompute visible-line styled content and invalidate on source/scroll changes.

#### 4.2 Runtime introspection in render path

Prototype footer currently can query runtime chain while rendering members pane.

Recommended upgrade:

- Cache proto chain per selected global until selection changes.
- Avoid runtime calls inside hot render loops.

### 5) API/UX Reliability

#### 5.1 Command parser needs quoting support

Current `strings.Fields` parsing breaks common path names. Replace with shell-like parser or command lexer.

#### 5.2 REPL source tracking should be structured, not substring-based

`showReplFunctionSource` matches by `strings.Contains` on expression/function strings. This can mis-associate sources and grows unbounded.

Recommended approach:

- maintain function identity map from eval result to source entry id when possible.
- optionally cap REPL source log size and provide clear/reset command.

### 6) Idiomatic Go and Maintainability

1. Replace manual sort loops with `sort.Slice` in globals sorting.
2. Replace numeric kind literals with typed constants.
3. Move large builtin maps to package-level variables to avoid repeated allocations.
4. Consolidate duplicate helper utilities in shared internal package.

---

## Deprecated or Unidiomatic Areas

1. **Magic numbers for enum kinds**: `view.go` uses raw numeric literals for binding kinds.  
2. **Comment claims mode-keymap integration, but implementation does not use it**: `keymap.go` comment is misleading.  
3. **Manual scroll/list rendering in many panes** despite reusable component baseline in GOJA-025.  
4. **String-field parsing for declarations** (`extractDeclaredNames`) instead of parser-backed approach.

---

## Refactor Plan (Pragmatic, Incremental)

### Phase A: Stabilize (Safety + Regression)

1. Add cycle detection to inheritance recursion.
2. Add test coverage in `cmd/smalltalk-inspector/app` for:
- globals/members selection sync
- inspect/stack scroll visibility
- command parser (`:load` with quoted paths)
- REPL source rendering and syntax-span updates.
3. Add crash/regression test for `class A extends A {}`.

### Phase B: Component Alignment

1. Introduce shared `internal/inspectorui` package with:
- keyed-mode helper
- scrollable selectable list model
- source view model.
2. Migrate smalltalk-inspector globals/members/inspect/stack to reusable list/table abstractions.
3. Move command parser to reusable module and add quoted args support.

### Phase C: Domain Layer Consolidation

1. Integrate `pkg/inspector/analysis` into smalltalk command path.
2. Keep `pkg/inspector/runtime` as single runtime truth source.
3. Remove duplicated static-analysis logic from command model.

### Phase D: Performance Hardening

1. Optimize syntax highlight lookup strategy.
2. Cache proto chains and descriptor rows per selection.
3. Add benchmark for source-pane rendering at large file sizes.

---

## Suggested Task Backlog for GOJA-028

1. Guard inherited-member recursion with cycle detection + tests.
2. Add inspect and stack scrolling state/visibility guarantees.
3. Fix source scrolling bounds when `showingReplSrc` is true.
4. Rebuild REPL syntax spans after runtime fallback source append.
5. Replace `strings.Fields` command parsing with quoted argument parser.
6. Replace magic binding kind literals with typed constants.
7. Introduce shared list/source/status components and migrate smalltalk panes.
8. Integrate `pkg/inspector/analysis` in smalltalk path.
9. Optimize syntax span lookup/rendering and add benchmark.
10. Complete GOJA-027 documentation closure and validation task.

---

## Open Questions / Assumptions

1. Should smalltalk-inspector intentionally remain separate from `cmd/inspector` abstractions, or is a shared `internal/inspectorui` module preferred?
2. Is cross-platform stack parsing (Windows paths) an explicit near-term requirement?
3. Should REPL source history persist across file loads, or reset on each `:load`?
4. Should descriptor UI for symbol properties be included in immediate cleanup scope, or deferred?

---

## Implementation Notes for New Developers

Start here in order:

1. `go-go-goja/cmd/smalltalk-inspector/app/update.go` (interaction and state transitions)
2. `go-go-goja/cmd/smalltalk-inspector/app/model.go` (data model and domain wiring)
3. `go-go-goja/cmd/smalltalk-inspector/app/view.go` (render behavior and scrolling limitations)
4. `go-go-goja/pkg/inspector/runtime/introspect.go` and `go-go-goja/pkg/inspector/runtime/errors.go`
5. `go-go-goja/cmd/inspector/app/keymap.go` and `go-go-goja/cmd/inspector/app/tree_list.go` for reusable component patterns

Recommended first command set:

```bash
cd go-go-goja

go test ./... -count=1

rg -n "addInheritedMembers|buildMembers|refreshRuntimeGlobals" cmd/smalltalk-inspector/app/model.go
rg -n "handleInspectKey|handleStackKey|handleSourceKey|executeCommand" cmd/smalltalk-inspector/app/update.go
rg -n "renderInspectPane|renderStackPane|renderSourcePane|renderSyntaxLine" cmd/smalltalk-inspector/app/view.go
```

---

## Review Appendix: Commands Executed During Analysis

```bash
# baseline test coverage and health
cd go-go-goja && go test ./... -count=1
cd go-go-goja && go vet ./...

# ticket and commit inventory
find go-go-goja/ttmp -maxdepth 4 -type d | rg 'GOJA-0(2[4-9]|3[0-9])'
git -C go-go-goja log --oneline -n 40

# key code surface
rg --files go-go-goja/cmd/smalltalk-inspector go-go-goja/pkg/inspector
wc -l go-go-goja/cmd/smalltalk-inspector/app/*.go go-go-goja/pkg/inspector/**/*.go

# crash reproduction (self extends)
# (performed via temporary test; output showed fatal stack overflow)
go test ./cmd/smalltalk-inspector/app -run TestTmpSelfExtends -count=1
```

---

## Final Assessment

The inspector effort since GOJA-024 delivered real user-facing capability quickly: the smalltalk-style browser works, runtime object inspection is functional, stack traces are viewable, and syntax coloring improved readability. GOJA-026 in particular fixed meaningful correctness gaps.

The main risk now is not missing features; it is **structural drift**:

- large monolithic command files,
- incomplete reuse of GOJA-025 component infrastructure,
- missing tests in the highest-change surface,
- and one proven crash class in inheritance recursion.

Addressing those items in GOJA-028 will convert the current implementation from feature-complete prototype quality into a stable, reusable inspector platform.
