---
Title: Syntax Highlighting Algorithm Research
Ticket: GOJA-030-SYNTAX-HIGHLIGHTING-IMPROVEMENTS
Status: active
Topics:
    - go
    - goja
    - tui
    - inspector
    - refactor
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/jsparse/highlight.go
      Note: Current syntax span model and hot lookup function (`SyntaxClassAt`)
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: Parser behavior, full reparse strategy, and TS node coordinate model
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Source-pane render loop and current per-character highlight path
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: File/REPL span rebuild flow and REPL fallback append behavior
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-030-SYNTAX-HIGHLIGHTING-IMPROVEMENTS--syntax-highlighting-improvements/design/01-syntax-highlighting-implementation-plan.md
      Note: Baseline implementation plan being refined by this research
ExternalSources: []
Summary: Deep algorithm research memo covering current highlighting architecture, known bottlenecks, candidate algorithm families, recommended target design, and phased migration guidance for GOJA-030.
LastUpdated: 2026-02-15T15:06:00Z
WhatFor: Provide a defensible algorithm choice and concrete implementation details before performance and correctness refactors.
WhenToUse: Read before implementing GOJA-030 tasks 4-14 and when reviewing future highlighting architecture changes.
---

# Syntax Highlighting Algorithm Research

## Executive Summary

This memo evaluates the current syntax-highlighting pipeline in `go-go-goja` and recommends a concrete algorithm stack that improves both latency and maintainability while preserving the existing visual behavior.

The current implementation is functionally correct for many cases and already uses tree-sitter as its parser foundation. However, rendering cost scales poorly because each visible character performs a global linear scan over all syntax spans. This creates an avoidable `O(visible_chars * total_spans)` hot path.

Recommended direction:

1. Keep tree-sitter parsing and existing `SyntaxClass` taxonomy in place.
2. Replace global span scans with per-line indexed spans and binary lookup.
3. Move from per-character styling to segment-based line rendering.
4. Add lightweight, invalidation-aware styled-line caching.
5. Separate core highlighting engine code from Bubble Tea rendering adapters so the same core can later be reused by CLI batch output or REST APIs.

This recommendation provides strong expected gains with low migration risk because it can be introduced in compatibility phases and verified with parity tests.

## Scope and Constraints

This research focuses on syntax-highlighting algorithm choices, not on re-theming colors or redesigning the UI. It addresses both file source and REPL source, including runtime fallback snippets.

Constraints assumed:

1. Language support remains JavaScript via tree-sitter.
2. Output remains terminal-oriented (ANSI/lipgloss today).
3. Existing `SyntaxClass` values and visible color behavior should remain stable during early phases.
4. We need an architecture that can be reused outside Bubble Tea in a later phase.

## Current Architecture (as implemented)

### Parser and syntax spans

Current flow:

1. `pkg/jsparse/treesitter.go` creates a JavaScript tree-sitter parser (`NewTSParser`).
2. `TSParser.Parse` currently reparses from scratch and snapshots a `TSNode` tree.
3. `pkg/jsparse/highlight.go` traverses leaf nodes and maps node kinds to `SyntaxClass` via `ClassifySyntaxKind`.
4. `BuildSyntaxSpans` emits a flat `[]SyntaxSpan` with 1-based start/end line/column coordinates.

Important detail: highlighting is tree-sitter based today. It is not regex-only, and it is not using tree-sitter query captures yet. It classifies by node kind names from the CST walk.

### Render path in the inspector

In `cmd/smalltalk-inspector/app/view.go`, `renderSourcePane` calls `renderSyntaxLine` for non-target lines. `renderSyntaxLine` loops every character in the line and calls `jsparse.SyntaxClassAt(spans, lineNo, colNo)`.

`SyntaxClassAt` scans the full span slice linearly and returns the first span that contains the position.

That means visible rendering cost is:

`visible_chars * linear_scan(all_spans)`

This is the primary performance bottleneck.

### Span rebuild lifecycle

In `cmd/smalltalk-inspector/app/model.go`:

1. File load triggers `rebuildFileSyntaxSpans`.
2. REPL append via `appendReplSource` triggers `rebuildReplSyntaxSpans`.
3. Runtime fallback append path in `showReplFunctionSource` appends lines, but does not rebuild REPL spans in that path.

The fallback path creates a correctness and consistency gap: newly appended runtime snippet lines may show stale/no highlighting until a later rebuild event occurs.

## Problem Analysis

## 1) Time complexity hotspot

The renderer repeatedly performs global span lookup for each character, causing worst-case behavior proportional to full document span count even when only a small viewport is visible.

Symptoms:

1. Larger files and long REPL histories produce slower redraws.
2. Scrolling and focus transitions can feel heavy due to repeated recomputation.
3. Allocation and CPU both trend up with input size.

## 2) Missing structural index

Spans are stored flat and unindexed. The lookup path does not exploit:

1. line boundaries,
2. sorted ranges,
3. contiguous style runs.

Without line locality, rendering redoes avoidable work.

## 3) Rendering granularity mismatch

The model captures token ranges, but rendering works per character. For terminal output, line-segment rendering is a better fit: apply style to contiguous ranges once, concatenate segments, and avoid per-character style decisions.

## 4) Invalidation behavior gaps

There is no styled-line cache, and fallback REPL source append currently misses span rebuild in one branch. This combination increases both stale-display risk and redundant compute.

## 5) Test/benchmark coverage gap

There is currently little dedicated benchmark coverage for highlight hot paths. Without baseline + after metrics, algorithm changes are harder to justify and tune.

## Candidate Algorithm Families

This section compares algorithm families that are relevant to the current codebase and expected workloads.

### A. Flat spans + linear scan (current baseline)

Approach:

1. Keep one flat slice of spans.
2. For each character, scan all spans until a match.

Complexity:

1. Lookup per char: `O(total_spans)`
2. Visible line render: `O(visible_chars * total_spans)`

Assessment:

1. Simple but non-scalable.
2. Should not remain the hot path.

### B. Per-line buckets + binary search

Approach:

1. Pre-index spans by line.
2. Each line holds sorted spans by start column.
3. Lookup class at column via binary search, then local containment check.

Complexity:

1. Index build: `O(total_spans + multi_line_expansion)`
2. Per char lookup: `O(log line_spans)` (often very small)
3. Line render: `O(line_chars * log line_spans)`

Assessment:

1. Strong near-term improvement with moderate implementation cost.
2. Works cleanly with current span model.
3. Good first migration step.

### C. Segment-based line renderer (run-length styling)

Approach:

1. Convert each line into ordered style segments.
2. Render each segment once (unstyled prefix + styled runs).

Complexity:

1. Segment build per line: near `O(line_spans + line_chars_for_boundaries)`
2. Render: `O(segments)` style applications, not `O(chars)`

Assessment:

1. Large constant-factor wins for terminal rendering.
2. Pairs well with per-line index.
3. Should be phase-two after indexing.

### D. Interval tree over global spans

Approach:

1. Build an interval tree for all spans.
2. Query intervals containing a given coordinate.

Complexity:

1. Query: ~`O(log n + k)`
2. Build and memory overhead higher than line buckets.

Assessment:

1. Powerful but likely unnecessary for this workload.
2. More complex than needed because rendering is line-oriented and viewport-limited.
3. Better fit if global random-access querying dominates (not the case here).

### E. Sweep-line event model

Approach:

1. For each line, precompute start/end events by column.
2. Walk left-to-right maintaining active style.

Complexity:

1. Line build: `O(events + line_chars)`
2. Useful with overlapping layers.

Assessment:

1. Good for richer multi-layer semantics.
2. Adds complexity not required for current single-class output.
3. Could be an evolution if semantic layers are added later.

### F. Tree-sitter query captures

Approach:

1. Use query files/captures instead of node-kind mapping.
2. Map captures to style categories.

Benefits:

1. More semantically precise highlighting.
2. Easier language-specific tuning as query sets grow.

Tradeoffs:

1. Higher up-front complexity in query authoring.
2. Requires robust capture-priority rules and test coverage.

Assessment:

1. Valuable long-term correctness enhancement.
2. Not required for immediate performance optimization.

### G. Incremental parse + partial invalidation

Approach:

1. Track edits and dirty ranges.
2. Reparse incrementally.
3. Rebuild spans/segments only for affected lines.

Assessment:

1. Very useful for true editor scenarios.
2. Current inspector workload (load + occasional append) may not justify immediate complexity.
3. Good phase-three/four direction after baseline optimization.

## Recommended Target Design

Recommended stack:

1. Tree-sitter parse remains source of truth.
2. Span extraction remains compatible with existing `SyntaxClass`.
3. Add line index to avoid global scans.
4. Add segment renderer for contiguous styling.
5. Add optional styled-line cache with explicit invalidation.
6. Encapsulate this in a core package independent from Bubble Tea.

### Why this is the best fit now

1. It directly addresses the measured bottleneck path.
2. It avoids premature complexity (no interval tree required).
3. It can be implemented in small, reviewable slices.
4. It creates a reusable core for future REST/CLI use.

## Core/UI Separation Design

To satisfy the requirement of separating Bubble Tea UI concerns from reusable logic, split responsibilities as follows.

### Core package (non-UI, reusable)

Proposed package:

`pkg/highlightcore` (or `pkg/jsparse/highlightcore`)

Responsibilities:

1. Token class model (`SyntaxClass` reused or aliased).
2. Span extraction adapters from TS nodes.
3. Line index construction.
4. Segment generation from line text + indexed spans.
5. Styled-line cache keys and invalidation policy.
6. Pure data output (segments with class labels), no lipgloss calls.

Interfaces:

1. `BuildDocumentIndex(source []byte, root *TSNode) (DocumentIndex, error)`
2. `SegmentsForLine(index DocumentIndex, lineNo int, line string) []Segment`
3. `InvalidateLines(cache *Cache, startLine, endLine int)`

### UI adapter package (Bubble Tea specific)

Current likely location:

`cmd/smalltalk-inspector/app` plus possible helpers in `internal/inspectorui`

Responsibilities:

1. Convert segment classes to styled strings using lipgloss.
2. Integrate with viewport/list/table rendering.
3. Manage pane focus, scroll offsets, and update loop messages.

Outcome:

1. Inspector uses core engine.
2. Future CLI/REST can call the same core APIs and apply different formatting adapters.

## Data Structures

### Document index

Proposed types:

```go
type LineSpan struct {
    StartCol int // 1-based inclusive
    EndCol   int // 1-based exclusive
    Class    SyntaxClass
}

type DocumentIndex struct {
    // lines[1] => spans for line 1
    Lines [][]LineSpan
}
```

Construction rules:

1. Expand multi-line spans into line-local segments.
2. Normalize bounds to `[1, lineLen+1]` where line length semantics are explicit.
3. Sort by `StartCol`, then `EndCol`.
4. Merge adjacent spans when class and boundaries permit.

### Render segment

```go
type Segment struct {
    Text  string
    Class SyntaxClass
}
```

Segment production invariant:

1. Concatenated `Segment.Text` reproduces the input line exactly.
2. Segment classes are maximal contiguous runs.

### Cache key

```go
type StyledLineKey struct {
    SourceVersion uint64
    LineNo        int
    ThemeHash     uint64
}
```

Cache invalidation:

1. File reload increments `SourceVersion`, invalidating all lines implicitly.
2. REPL append invalidates appended line range only.
3. Theme changes invalidate by hash mismatch.

## Algorithms in Detail

### Algorithm 1: Build line index from flat spans

Pseudo-flow:

1. Allocate line bucket slice from line count.
2. For each span:
3. For each line touched by span:
4. Clip span to line-local start/end.
5. Append `LineSpan` to that line bucket.
6. Post-process each line: sort and coalesce.

Pseudo-code:

```text
for span in spans:
  for line = span.StartLine .. span.EndLine:
    localStart = if line == span.StartLine then span.StartCol else 1
    localEnd   = if line == span.EndLine then span.EndCol else lineMaxCol(line)
    if localStart < localEnd:
      lines[line].append(LineSpan{localStart, localEnd, span.Class})

for each line bucket:
  sort by (StartCol, EndCol)
  normalize/merge compatible neighbors
```

### Algorithm 2: Lookup class at column with binary search

Per line:

1. Binary search nearest span with `StartCol <= col`.
2. Check containment `col < EndCol`.
3. Return class or none.

This eliminates full-document scans and keeps line-local lookups cheap.

### Algorithm 3: Segment renderer

For a given line:

1. Start at column 1.
2. Walk spans in order.
3. Emit unstyled segment for gap before span.
4. Emit styled segment for span intersection.
5. Continue until end of line.

This reduces style calls and string-builder churn relative to per-character rendering.

### Algorithm 4: Styled-line cache with invalidation

1. Build styled line once on first view.
2. Cache by `{sourceVersion, lineNo, themeHash}`.
3. On re-render, return cached string if key matches.
4. On edits/appends/theme changes, invalidate relevant keys.

This provides stable scroll latency and avoids repeated recomputation for unchanged viewport lines.

## Correctness Considerations

### Span overlap precedence

Current implementation effectively uses first-match precedence from DFS span order. After indexing/segmenting, precedence must remain deterministic.

Recommended rule:

1. Primary: smallest containing span wins when categories overlap.
2. Secondary tie-break: earlier start, then shorter range, then stable insertion order.

If behavior parity with current order is required initially, preserve current extraction order and apply only monotonic transforms.

### Multi-line token handling

Multi-line comments and template strings require careful clipping during line expansion. Tests should include:

1. block comments spanning many lines,
2. template literals with interpolations,
3. escaped backticks and nested `${}`.

### Column semantics and Unicode

Tree-sitter columns are byte-oriented. Go string iteration in `range` also returns byte indices, which currently aligns for column mapping, but display width can still diverge for wide glyphs.

Recommendation:

1. Keep byte-based indexing for syntax correctness.
2. Document that display alignment for non-ASCII wide characters may require a later terminal-width reconciliation layer.

### REPL fallback source path

In `showReplFunctionSource`, runtime fallback append currently does not call `rebuildReplSyntaxSpans`. This should be fixed as part of invalidation work, and verified with a regression test to ensure newly appended fallback snippets are highlighted immediately.

## Benchmarks and Profiling Plan

To make algorithm choice evidence-based, add repeatable benchmarks:

1. Small file (200-500 lines)
2. Medium file (2k-5k lines)
3. Large file (20k+ lines or synthetic equivalent)
4. REPL accumulation (append 500-2k expressions)

Measure:

1. ns/op for line render path
2. allocations/op
3. bytes/op
4. end-to-end pane redraw latency in representative viewports

Profile checkpoints:

1. baseline current algorithm
2. + line index
3. + segment rendering
4. + cache

Success criteria:

1. 3x+ speedup on medium/large render benchmarks
2. clear allocation reduction
3. no visible regression in highlight output parity tests

## Migration Strategy (Incremental and Safe)

### Phase 1: Introduce index behind compatibility API

1. Keep `BuildSyntaxSpans` public behavior unchanged.
2. Add `BuildLineIndex(spans, sourceLines)` and `SyntaxClassAtIndexed`.
3. Add tests proving parity with `SyntaxClassAt` for sampled coordinates.

### Phase 2: Add segment renderer

1. Introduce `RenderSyntaxLineSegments`.
2. Keep old per-char renderer behind a feature flag or test helper.
3. Snapshot-test representative lines to verify output class parity.

### Phase 3: Add cache + invalidation

1. Cache only in source pane path first.
2. Invalidate on file load, REPL append, and theme changes.
3. Add targeted tests for invalidation correctness.

### Phase 4: Correctness expansion

1. Expand token kind mapping coverage.
2. Add multiline/template/operator edge tests.
3. Optionally evaluate tree-sitter query captures for semantic improvements.

## What a Coworker Research Spike Should Cover

A coworker research spike is still worthwhile for one focused question: if/when to move from node-kind mapping to tree-sitter query-capture highlighting for semantic precision.

Requested deliverables:

1. Comparison of query-capture systems in editor-like tools.
2. Recommended capture-priority model for this codebase.
3. Migration cost estimate from kind-based mapping.
4. Test methodology for ensuring semantic correctness.

This spike is optional for performance optimization itself, but valuable before deeper correctness/semantics phases.

## Risks and Mitigations

Risk: behavioral drift while changing lookup order.
Mitigation: parity tests comparing old and new class outputs on fixtures.

Risk: cache invalidation bugs causing stale lines.
Mitigation: explicit invalidation API and versioned cache keys.

Risk: increased complexity in core package boundaries.
Mitigation: small interfaces and strict separation between pure core and UI adapters.

Risk: over-optimizing before baseline measurement.
Mitigation: require benchmark baseline and checkpoint reports before merging each phase.

## Detailed Implementation Mapping (File-by-File)

This section maps the algorithm work to concrete files and symbols so a developer can execute without rediscovering architecture.

### `pkg/jsparse/highlight.go`

Current responsibilities:

1. Class taxonomy (`SyntaxClass`)
2. Span extraction (`BuildSyntaxSpans`)
3. Point lookup (`SyntaxClassAt`)
4. Char-level rendering helpers (`RenderSyntaxChar`)

Recommended additions:

1. Keep existing exported types for compatibility.
2. Add a new line-index type and builder.
3. Add lookup functions that use the index.
4. Add line-to-segment converter that produces class-labeled segments.

Expected refactor shape:

1. `BuildSyntaxSpans` remains as source-compatible baseline.
2. `BuildLineIndex(spans []SyntaxSpan, lineCount int) *LineIndex`
3. `ClassAt(index *LineIndex, lineNo, colNo int) SyntaxClass`
4. `SegmentsForLine(index *LineIndex, line string, lineNo int) []Segment`

`RenderSyntaxChar` can stay for compatibility, but the main render path should stop calling it in a character loop.

### `cmd/smalltalk-inspector/app/view.go`

Current hot path:

1. `renderSourcePane` loops lines
2. `renderSyntaxLine` loops each character
3. each character calls `SyntaxClassAt` over global spans

Recommended change:

1. Build or receive a line index from model state.
2. For each visible line, request precomputed segments.
3. Style per segment, not per char.

This preserves the Bubble Tea `viewport` usage and pane logic. The optimization is isolated to content generation before `vp.SetContent`.

### `cmd/smalltalk-inspector/app/model.go`

Current responsibilities include source lifecycle and span rebuild calls.

Required updates:

1. Model fields should include indexed highlight artifacts, not only flat spans.
2. `rebuildFileSyntaxSpans` and `rebuildReplSyntaxSpans` should rebuild both flat spans (if still needed) and line index.
3. `showReplFunctionSource` runtime fallback append path must call rebuild/invalidate.
4. Add source-version counters for cache keying.

This is also where invalidation boundaries are best computed because model code already owns file-load and REPL-append state transitions.

### `pkg/jsparse/treesitter.go`

No immediate parser changes are required for phase-one and phase-two optimization.

However, document assumptions here:

1. parse-from-scratch behavior is currently intentional.
2. column coordinates are byte-based.
3. a future incremental parse phase would start here.

### Tests and benchmarks

Recommended additions:

1. `pkg/jsparse/highlight_bench_test.go` for synthetic and fixture-based benchmarks.
2. `pkg/jsparse/highlight_index_test.go` for index correctness and parity tests.
3. inspector integration tests for REPL fallback source append highlighting behavior.

## Worked Example: Why Segment Rendering Helps

Consider line:

```js
const msg = `Hello ${user.name}!`; // greet
```

Current approach:

1. Iterate every visible character.
2. For each character, scan all global spans.
3. Style one character at a time.

Even on this short line, style decisions are repeated for many adjacent characters that share the same class. On a medium file with many spans, this repeats thousands of unnecessary span checks per redraw.

Segment approach:

1. Convert the line into runs:
2. `const` keyword run
3. whitespace run
4. identifier run
5. operator run
6. template literal runs
7. punctuation run
8. comment run

Then style each run once and append.

The segment method reduces both lookup calls and style object work. It also composes naturally with line caches because segment output is deterministic for `(line text, line spans, theme)`.

## Benchmark Blueprint (Concrete)

To avoid subjective optimization, create benchmarks with controlled fixtures.

### Benchmark groups

1. `BenchmarkHighlight_LineRender_Small`
2. `BenchmarkHighlight_LineRender_Medium`
3. `BenchmarkHighlight_LineRender_Large`
4. `BenchmarkHighlight_REPLAppend_Rebuild`
5. `BenchmarkHighlight_REPLAppend_InvalidateOnly`

### Data setup strategy

1. Use a few real JS fixture files committed under testdata.
2. Add one synthetic stress fixture with repeated constructs.
3. Generate a REPL transcript fixture with many appended snippets.

### Metrics to capture

1. time/op
2. allocs/op
3. bytes/op
4. optional profile for large-case hot path

### Benchmark acceptance gate

A practical merge gate:

1. no regression on small fixtures,
2. at least 2x improvement on medium case,
3. at least 3x improvement on large case,
4. noticeable allocation drop.

## Correctness Test Matrix

The optimization should keep visible behavior stable unless explicitly improving classification.

### Category A: parity tests

Compare class outputs between old and new lookup for randomly sampled coordinates across fixtures.

### Category B: edge-structure tests

Include:

1. nested template literals
2. multiline block comments
3. arrow functions and chained calls
4. optional chaining and nullish coalescing
5. destructuring patterns and shorthand properties
6. regex literals if represented by tree-sitter kinds in current grammar version

### Category C: REPL-specific tests

Include:

1. expression append triggers rebuild/invalidation,
2. runtime fallback append highlights immediately,
3. switching between file and REPL sources keeps correct line mapping.

### Category D: stability tests

Ensure no panics when:

1. spans are empty,
2. line index has sparse lines,
3. requested line/column out of bounds,
4. malformed node spans appear due to parser error nodes.

## Cache Design Deep Dive

Caching should be conservative and easy to reason about.

### Key design

Use three dimensions:

1. source version
2. line number
3. theme hash

### Why not cache whole pane strings

Whole-pane caching becomes fragile with viewport offsets and mode banners. Line-level caching is simpler and reuses well across scroll operations.

### Invalidation rules

1. file load: bump source version (invalidate all implicitly),
2. REPL append: invalidate appended lines only, keep earlier lines,
3. theme change: new theme hash invalidates style layer while reusing structural index.

### Memory bounds

Use bounded map or small LRU:

1. cap by number of lines or bytes,
2. evict least recently used lines,
3. expose simple metrics for hit rate.

For this inspector, even a plain map with periodic reset may be enough at first. Add LRU only if memory profiling justifies it.

## Common Syntax Highlighting Algorithms (General Landscape)

This section answers the broader question of common algorithms beyond this codebase.

### 1. Regex/tokenizer-based lexers

Examples include Pygments-like token streams. Fast for simple grammars and static lexers, but weaker for nested language constructs and incremental parsing complexity.

### 2. Parser-based AST/CST classification

The current code is in this family: parse with tree-sitter, then classify nodes. Good structural correctness and robust for real code.

### 3. Query-capture semantic highlighting

Tree-sitter queries capture semantic nodes (`function.name`, `property`, `type`, etc.) and map captures to styles. Rich semantics, language-specific tuning, more configuration surface.

### 4. Incremental region-based highlighters

Used by editors that update only dirty regions after edits. Strong for interactive editing but requires edit tracking and careful invalidation.

### 5. Rope/piece-table coupled highlighters

Deep editor architectures co-locate text storage with highlight metadata for efficient edits. Probably unnecessary here unless the inspector evolves into a full editor.

### 6. Hybrid lexical + semantic layers

Common in modern IDEs: lexical base colors plus semantic overlays from language servers. Powerful but significantly more complex.

For this project’s scope, parser-based classification plus indexed rendering is the right middle ground.

## Why Bubble Tea `viewport` Should Stay

The existing `viewport` component is not the problem. It efficiently handles scrolling and clipping, and it integrates with the inspector’s layout model.

The expensive part is the string-generation path that feeds the viewport.

Therefore:

1. keep `viewport`,
2. optimize the highlight generation pipeline,
3. optionally cache rendered lines before joining into viewport content.

This keeps UX behavior stable while solving core compute cost.

## Addressing “Issue 5” and Relationship to “Issue 3”

Based on prior ticket discussion context:

1. issue 5 maps to highlight lookup/render inefficiency,
2. issue 3 concerns viewport/render structure concerns.

Does solving issue 5 address issue 3?

Partially:

1. It addresses perceived viewport sluggishness by reducing content-generation cost.
2. It does not replace viewport architecture, but it removes the dominant hot path under it.

If issue 3 includes broader pane-lifecycle concerns, those remain separate. But for rendering performance in source pane, this plan is the direct fix.

## New Developer Orientation: Where to Look First

A new contributor should traverse the code in this order:

1. `cmd/smalltalk-inspector/app/model.go`
   This is source lifecycle control: load, REPL append, mode transitions, rebuild hooks.
2. `cmd/smalltalk-inspector/app/view.go`
   This is where highlight output is consumed and piped into viewport content.
3. `pkg/jsparse/highlight.go`
   This is the current highlight engine and will host the index/segment refactor.
4. `pkg/jsparse/treesitter.go`
   This clarifies parser semantics and coordinate conventions.
5. ticket docs in `ttmp/.../GOJA-030.../design`
   These capture why the architecture is changing and in what phase order.

Suggested review sequence for pull requests:

1. inspect benchmark deltas,
2. inspect parity/correctness tests,
3. inspect model invalidation hooks,
4. inspect renderer path changes.

## Prototype Outline for `scripts/` (Optional)

If we want quick empirical checks before full integration, add two scripts under GOJA-030:

1. `scripts/highlight_bench_driver.go`
   Loads fixture text, builds spans/index, runs timed render loops.
2. `scripts/highlight_parity_check.go`
   Compares old vs new class lookup on sampled coordinates and reports mismatches.

These can be temporary but are useful during algorithm iterations before final benchmark tests are stabilized.

## Final Recommendation

Implement a hybrid of:

1. per-line indexed spans,
2. segment-based rendering,
3. bounded styled-line caching,

while retaining tree-sitter parse + current syntax class taxonomy in the short term.

This combination is the best tradeoff for GOJA-030 because it gives immediate and substantial performance gains, keeps migration risk manageable, and creates the core/UI separation needed for future REST/CLI exposure.

Short answer to active architecture questions:

1. Yes, syntax highlighting is currently tree-sitter-based.
2. Yes, Bubble Tea `viewport` remains the right scrolling primitive; optimize the line-generation pipeline feeding it.
3. The plan directly addresses issue 5, and it resolves the performance aspect of issue 3 without requiring viewport replacement.
4. The REPL fallback span-rebuild gap is separate and must be fixed explicitly in `model.go` invalidation flow.

## Appendix A: End-to-End Pipeline Pseudocode

This appendix spells out a concrete pipeline with clear ownership boundaries between core engine and UI adapter. The objective is to make implementation order unambiguous and keep side effects out of the core.

### Core data flow

```text
source bytes
  -> parse tree (tree-sitter)
  -> flat spans (existing BuildSyntaxSpans-compatible)
  -> line index (new)
  -> per-line segments (new)
  -> styled line cache lookup/write (new)
  -> styled string output for viewport
```

### Suggested core API

```go
type Engine struct {
    parser      *jsparse.TSParser
    classMapper ClassMapper
}

type Document struct {
    Version uint64
    Source  []string
    Spans   []SyntaxSpan
    Index   *LineIndex
}

type Renderer struct {
    Theme Theme
    Cache *LineCache
}
```

### Build document operation

```text
func BuildDocument(sourceText string) Document:
  lines = splitLines(sourceText)
  root = parser.Parse([]byte(sourceText))
  spans = BuildSyntaxSpans(root)
  index = BuildLineIndex(spans, len(lines), lineLengths(lines))
  return Document{Version: nextVersion(), Source: lines, Spans: spans, Index: index}
```

Notes:

1. `lineLengths(lines)` should be computed once and reused in line clipping operations.
2. If source is empty, return a document with empty index and still-valid version.
3. Keep build operation pure from UI state to simplify testing.

### Render line operation

```text
func RenderLine(doc, lineNo):
  key = {doc.Version, lineNo, themeHash}
  if cache has key:
    return cache[key]
  lineText = doc.Source[lineNo-1]
  spans = doc.Index.Lines[lineNo]
  segments = SegmentsForLine(spans, lineText)
  styled = styleSegments(segments, theme)
  cache[key] = styled
  return styled
```

The above can be adapted to both file-source and REPL-source documents. The UI only asks for visible lines and handles vertical composition.

## Appendix B: Comparison Matrix With Practical Tradeoffs

This matrix expands algorithm differences in terms that matter during implementation and code review.

### Flat span scan

Engineering properties:

1. Minimal code size.
2. Easy to understand.
3. No up-front indexing cost.

Operational drawbacks:

1. Repeatedly does the same global search work.
2. Cost spikes as spans increase, even when rendering a small viewport.
3. Hard to cache effectively because work happens at per-character granularity.

Where it still fits:

1. Quick prototypes.
2. Very tiny files.
3. Debug baseline function for parity tests.

### Per-line index + binary lookup

Engineering properties:

1. Moderate code size.
2. Straightforward deterministic behavior.
3. Easy to unit-test with table-driven inputs.

Operational advantages:

1. Excellent locality.
2. Dramatically fewer candidate spans per lookup.
3. Natural base for segment generation and line-level caching.

Potential pitfalls:

1. Off-by-one bugs during line clipping.
2. Overlap precedence ambiguities if spans conflict.
3. Need to define behavior for invalid/out-of-range coordinates.

### Segment renderer

Engineering properties:

1. Slightly more complex than char-loop rendering.
2. Requires robust string slicing rules.
3. Needs strong unit tests for line reconstruction invariants.

Operational advantages:

1. Fewer style operations.
2. Better allocation behavior with builders.
3. Pairs naturally with cache.

Potential pitfalls:

1. Incorrect segment boundaries can drop or duplicate characters.
2. Must preserve exact line text including whitespace.
3. Must handle empty lines and trailing newline semantics consistently.

### Interval tree

Engineering properties:

1. High complexity relative to present needs.
2. Harder to reason about and debug.
3. Harder onboarding for contributors unfamiliar with interval indexes.

Operational advantages:

1. Strong for arbitrary random-access point/range queries across whole doc.
2. Good foundation when many cross-line queries are needed.

Why not now:

1. The source pane is line-and-viewport oriented.
2. Complexity tax likely exceeds practical gain at this stage.

## Appendix C: Coordinate and Boundary Semantics

Syntax bugs in editors and inspectors often come from inconsistent coordinate conventions. This appendix defines explicit rules to avoid drift.

### Coordinate system

Adopt and document:

1. Line numbers: 1-based external API.
2. Columns: 1-based, end-exclusive.
3. Internal slices: Go string byte indices when mapping from tree-sitter.

### Required invariants

1. `StartCol >= 1`
2. `EndCol > StartCol`
3. For any line-local span, `EndCol <= lineByteLength+1`
4. For multi-line expansion, each expanded line span remains end-exclusive.

### Multi-line expansion rules

Given span `(startLine, startCol) -> (endLine, endCol)`:

1. First line: `[startCol, lineLen+1)`
2. Intermediate lines: `[1, lineLen+1)`
3. Last line: `[1, endCol)`

Corner cases:

1. zero-length spans should be dropped,
2. malformed spans where end precedes start should be ignored with debug count,
3. spans on missing lines should be clamped or dropped depending on strictness mode.

### Unicode and width

Tree-sitter gives byte columns; terminals render display cells. Those are different dimensions.

Strategy:

1. Keep syntax boundaries byte-based for correctness.
2. Keep display width handling as a separate concern in the UI layer.
3. If needed later, add a byte-to-cell mapping adapter for wide glyph scenarios.

## Appendix D: Integration Guide for New Contributors

This section gives concrete starting points for developers newly joining the codebase.

### Step 1: Understand model lifecycle

Read `cmd/smalltalk-inspector/app/model.go` and follow:

1. file load path (`MsgFileLoaded` handling),
2. REPL result path (`MsgEvalResult`),
3. source switching path (`activeSourceLines`, `showingReplSrc`),
4. syntax rebuild hooks (`rebuildFileSyntaxSpans`, `rebuildReplSyntaxSpans`).

Goal:

Understand where source text mutates and where highlight artifacts should be rebuilt or invalidated.

### Step 2: Understand rendering hot path

Read `cmd/smalltalk-inspector/app/view.go`:

1. `renderSourcePane`,
2. `renderSyntaxLine`,
3. viewport setup (`vp.SetContent`).

Goal:

Understand where highlight work impacts frame-time.

### Step 3: Understand parser + spans

Read:

1. `pkg/jsparse/treesitter.go`,
2. `pkg/jsparse/highlight.go`.

Goal:

Understand how token spans are produced and where indexing should hook in.

### Step 4: Implement in safe slices

Recommended pull request progression:

1. PR1: add index types + tests, no UI path change yet.
2. PR2: add indexed lookup path under flag + parity tests.
3. PR3: add segment rendering and switch source pane.
4. PR4: add cache and invalidation.
5. PR5: add benchmark report and cleanup.

This keeps review focused and rollback simple.

### Step 5: Validate behavior manually

Manual checklist:

1. load small JS file,
2. load large JS file,
3. evaluate several REPL expressions,
4. inspect runtime object to trigger fallback source display,
5. scroll aggressively in source pane,
6. verify no stale highlights after new REPL append.

## Appendix E: Failure Modes and Debugging Playbook

When refactoring highlighting code, certain bugs recur. This playbook lists symptoms and likely root causes.

### Symptom: lines render with missing tail characters

Likely cause:

1. segment slicing dropped suffix after last span.

Fix:

1. ensure segment builder always emits trailing unstyled slice from current column to line end.

### Symptom: highlighting shifts right or left on some lines

Likely cause:

1. mismatch between byte indices and rune/cell indices.

Fix:

1. keep all syntax boundaries in byte coordinates,
2. avoid mixing rune position counters in boundary math.

### Symptom: stale highlighting after REPL runtime fallback

Likely cause:

1. append path updated text but not spans/index/cache.

Fix:

1. ensure fallback append triggers `rebuildReplSyntaxSpans` or line-range incremental update,
2. invalidate relevant cache keys.

### Symptom: random class mismatches after indexing

Likely cause:

1. overlap precedence changed unintentionally.

Fix:

1. encode precedence explicitly,
2. run parity tests against old lookup for known fixtures.

### Symptom: performance improvement is smaller than expected

Likely causes:

1. style-rendering dominates after lookup optimization,
2. excessive rebuilding rather than incremental invalidation,
3. benchmark fixture too small.

Fix:

1. profile with realistic medium/large fixtures,
2. ensure line cache is actually being hit,
3. inspect allocations in segment/styling path.

## Appendix F: Concrete Task Expansion for GOJA-030

This section turns ticket tasks into implementation-ready substeps.

### Task 4: per-line span index

Substeps:

1. add `LineSpan`, `LineIndex` types,
2. add span expansion/clipping logic,
3. add sorting and merge logic,
4. unit-test with single-line and multi-line fixtures.

Definition of done:

1. all index tests pass,
2. parity with baseline on sampled lookups,
3. no behavior change in current renderer path yet.

### Task 5: replace global lookup in renderer

Substeps:

1. wire indexed lookup into `renderSyntaxLine`,
2. keep fallback path for testing,
3. add benchmarks comparing old/new lookup.

Definition of done:

1. parity tests pass,
2. benchmark shows lookup improvement.

### Task 6: segment renderer

Substeps:

1. add segment builder,
2. replace char loop in source pane path,
3. add line reconstruction tests and class-boundary tests.

Definition of done:

1. output parity on fixtures,
2. fewer allocations in benchmark output.

### Task 7 and 8: cache + REPL fallback invalidation

Substeps:

1. add source version counters,
2. add line cache in model or renderer adapter,
3. call rebuild/invalidate in fallback append path,
4. add regression test for immediate post-fallback highlighting.

Definition of done:

1. no stale render after fallback append,
2. repeated scroll hits cache effectively.

### Tasks 9-11: correctness and behavior validation

Substeps:

1. add edge-case fixtures,
2. verify behavior in file-source and REPL-source modes,
3. run full tests including inspector command path.

Definition of done:

1. zero regressions in test suite,
2. explicit coverage for known prior failure modes.

### Tasks 12-14: final documentation and recommendation loop

Substeps:

1. summarize chosen algorithm and measured outcomes,
2. update plan document and ticket changelog,
3. capture follow-ups for optional query-capture phase.

Definition of done:

1. implementation and research docs aligned,
2. future work clearly separated from completed scope.

## Appendix G: Tree-sitter Query Capture Migration Notes

This appendix is forward-looking and intentionally optional for the immediate optimization.

### Why consider query captures later

Kind-based classification is stable but coarse. Query captures can identify richer categories such as:

1. function declaration names,
2. method definitions,
3. property keys versus variable identifiers,
4. parameter names and type-like identifiers (where grammar allows).

### Migration strategy (if pursued)

1. Keep existing kind mapper as fallback.
2. Add capture layer that can override or refine class assignment.
3. Introduce capture priority table to resolve conflicts.
4. Add fixture tests for semantic distinctions.

### Example priority rule

If a token matches both generic identifier and function-name capture:

1. use function-name style class,
2. otherwise fallback to identifier class.

### Risk profile

1. Higher complexity than current scope,
2. requires strong test coverage per language grammar version,
3. can introduce visual drift if capture sets are incomplete.

Recommendation:

Treat query-capture migration as a separate ticket after GOJA-030 performance phases land.

## Appendix H: Validation Commands and Review Checklist

Use these commands during implementation milestones:

```bash
go test ./pkg/jsparse -count=1
go test ./cmd/smalltalk-inspector/... -count=1
go test ./... -count=1
go test ./pkg/jsparse -bench Highlight -benchmem
```

When performance claims are made:

```bash
go test ./pkg/jsparse -bench Highlight -benchmem -cpuprofile /tmp/highlight.cpu
go tool pprof /tmp/highlight.cpu
```

Reviewer checklist:

1. Does the new index preserve coordinate semantics?
2. Do parity tests cover the old and new paths?
3. Is REPL fallback invalidation wired correctly?
4. Are benchmark fixtures representative and reproducible?
5. Is core logic free of Bubble Tea dependencies?
6. Is UI adapter code thin and clearly separated?
7. Are docs updated with algorithm choice and residual risks?

## Appendix I: Scenario Modeling and Cost Estimation

This appendix provides rough operation-count modeling to explain why each phase should produce measurable gains. The numbers are not exact runtime measurements, but they are useful for intuition and planning benchmark expectations.

### Scenario 1: Medium file, normal viewport

Assumptions:

1. file has 3,000 lines,
2. average line length in source pane is 80 bytes,
3. visible viewport renders 45 lines,
4. total spans across file are 25,000.

Current method estimated checks per frame:

1. visible characters = `45 * 80 = 3,600`,
2. each char calls linear scan of up to 25,000 spans,
3. rough containment checks per frame = `3,600 * 25,000 = 90,000,000`.

Even if branch prediction and early exits reduce practical work, the ceiling is very high.

Indexed method (line buckets + binary lookup):

Assume average spans per visible line bucket = 20.

1. per-char lookup ~`log2(20)` comparisons (~5),
2. checks per frame ~`3,600 * 5 = 18,000` plus light overhead.

This is several orders of magnitude lower than the global scan upper bound.

Segment method:

Assume average 14 segments per line.

1. style applications per frame = `45 * 14 = 630`,
2. boundary handling and string joins scale with segment count, not raw char count,
3. additional speedup expected from lower style invocation frequency.

### Scenario 2: Long REPL history

Assumptions:

1. REPL log has 1,200 entries,
2. each entry averages 3 code lines plus separators,
3. total REPL source lines near 5,000,
4. total spans near 35,000.

Current approach:

1. every frame still scans global spans for each visible character,
2. cost rises as history grows even if viewport remains constant height.

Indexed + cached approach:

1. append invalidates only appended lines,
2. unchanged lines remain cache hits,
3. steady-state scrolling cost primarily reflects cache misses for newly viewed lines.

This is critical for REPL-heavy sessions where users inspect historical snippets repeatedly.

### Scenario 3: Frequent focus switching

When users move between panes and source pane rerenders:

Current method:

1. recomputes highlight classification for all visible lines each time.

With cache:

1. repeated line renders are mostly cache hits,
2. focus switching overhead drops to pane composition + minimal string joins.

### Scenario 4: Worst-case token density

Some generated code has dense punctuation/operators and many short tokens.

Current char-loop scan worsens because:

1. many spans,
2. short adjacent spans,
3. frequent class changes.

Segment renderer still helps because:

1. classification work is done from indexed line spans,
2. even with many segments, segment-level styling beats global scan-per-char.

### Modeling takeaway

The largest gain source is eliminating global span scans. Segment rendering and caching provide additional multiplicative improvements on top, especially during repeated redraw patterns.

## Appendix J: Reference Implementation Sketch

This appendix outlines a concrete implementation skeleton that contributors can map directly to production code.

### Core engine package sketch

```go
package highlightcore

type SyntaxClass int

type Span struct {
    StartLine int
    StartCol  int
    EndLine   int
    EndCol    int
    Class     SyntaxClass
}

type LineSpan struct {
    StartCol int
    EndCol   int
    Class    SyntaxClass
}

type LineIndex struct {
    Lines [][]LineSpan // 1-based logical access, 0 slot unused or mapped
}

type Segment struct {
    StartCol int
    EndCol   int
    Class    SyntaxClass
}
```

### Index build function sketch

```go
func BuildLineIndex(spans []Span, lineLens []int) *LineIndex {
    idx := &LineIndex{Lines: make([][]LineSpan, len(lineLens)+1)}
    for _, sp := range spans {
        if sp.EndLine < sp.StartLine {
            continue
        }
        for line := sp.StartLine; line <= sp.EndLine; line++ {
            if line <= 0 || line > len(lineLens) {
                continue
            }
            start := 1
            end := lineLens[line-1] + 1
            if line == sp.StartLine {
                start = sp.StartCol
            }
            if line == sp.EndLine {
                end = sp.EndCol
            }
            if start < 1 {
                start = 1
            }
            if end > lineLens[line-1]+1 {
                end = lineLens[line-1] + 1
            }
            if start >= end {
                continue
            }
            idx.Lines[line] = append(idx.Lines[line], LineSpan{
                StartCol: start,
                EndCol:   end,
                Class:    sp.Class,
            })
        }
    }
    normalizeIndex(idx)
    return idx
}
```

### Normalization sketch

`normalizeIndex` should:

1. sort by `(StartCol, EndCol)`,
2. merge adjacent spans with same class where `prev.EndCol == next.StartCol`,
3. preserve deterministic order for overlaps.

### Segment conversion sketch

```go
func SegmentsForLine(lineText string, spans []LineSpan) []Segment {
    // treat lineText in byte coordinates
    // emit full coverage from 1..lineLen+1
}
```

Expected behavior:

1. all bytes are covered by exactly one segment class (including `SyntaxNone` gaps),
2. no overlaps in output segments,
3. output order strictly increasing by start col.

### UI adapter sketch

```go
type Styler interface {
    Render(class SyntaxClass, text string) string
}

func RenderStyledLine(line string, segs []Segment, styler Styler) string {
    // slice by byte ranges and apply style per segment
}
```

Bubble Tea integration:

1. source pane asks renderer for each visible line,
2. renderer returns styled strings,
3. viewport receives joined lines.

No Bubble Tea types should appear in core engine signatures.

### Cache sketch

```go
type LineKey struct {
    Version uint64
    LineNo  int
    Theme   uint64
}

type LineCache struct {
    Data map[LineKey]string
    // optional LRU metadata
}
```

Cache API:

1. `Get(key) (string, bool)`
2. `Put(key, value)`
3. `InvalidateVersion(version)`
4. `InvalidateRange(version, startLine, endLine)`

This is sufficient for initial rollout.

## Appendix K: Rollout, Observability, and Maintenance

Algorithm work succeeds long-term only if it remains observable and maintainable after merge.

### Rollout strategy

Use staged rollout with guardrails:

1. introduce new index and renderer behind internal switch,
2. run CI parity tests with both paths,
3. default to new path once benchmarks and parity are stable,
4. remove old path after one stabilization cycle.

### Observability hooks

Add lightweight counters in debug builds:

1. cache hit/miss ratio,
2. lines rendered per frame,
3. index rebuild count,
4. REPL append invalidation count.

These counters help diagnose regressions without full profiling sessions.

### Maintenance guidelines

1. keep coordinate semantics documented near types,
2. do not merge style-layer logic into core index code,
3. require benchmark snippets in PR descriptions for hot-path changes,
4. keep fixture corpus representative of real use cases,
5. keep parser-kind mapping and tests updated as tree-sitter grammar versions evolve.

### Dependency and upgrade considerations

Since parsing depends on tree-sitter JavaScript bindings:

1. grammar updates can change node kinds,
2. classification mappings may need updates,
3. semantic drift must be caught by fixture tests.

A robust test suite around classification and segment parity reduces upgrade risk substantially.

### Long-term options after GOJA-030

Once the core optimization lands and stabilizes, future options include:

1. query-capture semantic highlighting,
2. optional incremental parsing for live-edit workflows,
3. language-agnostic highlight core interfaces if multi-language support is desired,
4. exporting highlight metadata as JSON for REST consumers.

These should be treated as separate phases to keep GOJA-030 focused and shippable.
