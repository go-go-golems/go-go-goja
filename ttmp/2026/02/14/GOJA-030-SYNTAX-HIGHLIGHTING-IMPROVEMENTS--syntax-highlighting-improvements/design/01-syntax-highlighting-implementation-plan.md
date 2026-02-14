---
Title: Syntax Highlighting Improvement Plan
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
      Note: Current syntax span model and lookup algorithm
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: Span rebuild flow for file and REPL source
    - Path: go-go-goja/cmd/smalltalk-inspector/app/view.go
      Note: Current per-character syntax rendering path
    - Path: go-go-goja/pkg/jsparse/treesitter_parser.go
      Note: Parser baseline used for syntax spans
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-027-SYNTAX-HIGHLIGHT--syntax-highlighting-for-smalltalk-inspector-source-pane/tasks.md
      Note: Original highlighting feature implementation context
    - Path: go-go-goja/ttmp/2026/02/14/GOJA-028-CLEANUP-INSPECTOR--cleanup-inspector/reference/01-inspector-cleanup-review.md
      Note: Performance findings motivating this ticket
ExternalSources: []
Summary: Detailed plan to improve highlighting correctness and performance with measurable benchmarks and incremental algorithm upgrades.
LastUpdated: 2026-02-14T19:02:00Z
WhatFor: Provide a concrete implementation and benchmarking roadmap for upgrading current tree-sitter span rendering.
WhenToUse: Use as the execution guide for GOJA-030.
---

# Syntax Highlighting Improvement Plan

## Goal

Improve syntax highlighting performance and correctness for source panes (file + REPL) without regressing visual quality.

## Current Behavior Summary

Current flow:

1. Parse source with tree-sitter.
2. Build flat span list (`[]SyntaxSpan`).
3. During render, iterate each character in each visible line.
4. For each character, find class using linear span scan (`SyntaxClassAt`).

This is simple but costly for larger files and long REPL histories.

## Problem Statement

Primary issues:

1. Span lookup cost scales poorly (`O(chars * spans)` on visible region).
2. No per-line span indexing.
3. No styled-line cache; repeated rendering recomputes same output.
4. REPL fallback source append path can desync spans if rebuild is skipped.

## Design Principles

1. Keep tree-sitter as parser source of truth.
2. Separate parse/index work from render work.
3. Make rendering operate on pre-indexed line segments, not global span scans.
4. Add benchmarks before and after each major change.
5. Preserve current color mapping initially; optimize algorithm first.

## Common Algorithm Options (Decision Set)

## Option A: Flat spans + per-char scan (current)

Pros:
- simplest implementation.

Cons:
- poor scaling.

Decision:
- replace.

## Option B: Per-line span buckets + binary search

Data model:

- `map[int][]LineSpan` or `[][]LineSpan` indexed by line.
- each line span sorted by start column.

Lookup:

- binary search span containing column.

Complexity:

- near `O(chars * log(lineSpans))`.

Decision:
- adopt in Phase 1.

## Option C: Segment-based line rendering (run-length style)

Approach:

- convert each visible line into style segments.
- render whole segments instead of per-character lookups.

Complexity:

- near `O(segments + lineLength)` with lower constant factors.

Decision:
- adopt in Phase 2 after Phase 1 baseline.

## Option D: Incremental parse/highlight with dirty ranges

Approach:

- re-parse only changed content ranges.
- invalidate only affected line caches.

Decision:
- optional Phase 3; likely worthwhile for heavy REPL workflows.

## Option E: Tree-sitter query-based semantic highlighting

Approach:

- use query captures for richer token categories (function names, types, properties).

Decision:
- defer unless correctness requirements exceed current class-based mapping.

## Implementation Plan

## Phase 0: Baseline Measurement

1. Add benchmark(s):
- highlight render on small, medium, large JS source.
- REPL source accumulation scenario.
2. Capture CPU and allocation profiles for current path.

Deliverable:

- baseline numbers committed in benchmark outputs/docs.

## Phase 1: Per-line Span Index

1. Add line-indexed span structure in `pkg/jsparse/highlight.go`.
2. Build index once after `BuildSyntaxSpans`.
3. Replace global linear lookup in `SyntaxClassAt` hot path for rendering.

Deliverable:

- measurable perf improvement with same visible output.

## Phase 2: Segment Renderer

1. Add line segment generation utility:
- input raw line + line spans
- output style segments.
2. Replace per-char rendering loop in `renderSyntaxLine`.
3. Add tests comparing output parity with current renderer on fixture lines.

Deliverable:

- further perf and allocation reduction.

## Phase 3: Cache and Invalidation

1. Add styled-line cache keyed by:
- source identity/version
- line number
- active theme palette hash.
2. Invalidate only dirty ranges on source changes.
3. Ensure REPL append path triggers appropriate incremental updates.

Deliverable:

- stable latency during repeated scrolling.

## Phase 4: Correctness Enhancements

1. Expand token classification cases where needed.
2. Add explicit tests for multiline tokens, template strings, comments.
3. Verify REPL fallback source path always rebuilds spans.

Deliverable:

- stronger correctness with test coverage.

## Benchmark and Verification Plan

Required commands:

```bash
cd go-go-goja
go test ./pkg/jsparse -bench Highlight -benchmem
go test ./cmd/smalltalk-inspector/... -count=1
go test ./... -count=1
```

Add profiles (if benchmark indicates hotspots remain):

```bash
go test ./pkg/jsparse -bench Highlight -benchmem -cpuprofile /tmp/highlight.cpu.out
go tool pprof /tmp/highlight.cpu.out
```

## Should a Coworker Do Algorithm Research?

Yes, for a short focused spike (1-2 days) it is worthwhile.

What to ask them to deliver:

1. Compare editor-grade highlight approaches for tree-sitter-backed TUIs.
2. Recommend one near-term algorithm for this codebase (likely line-index + segment rendering).
3. Provide expected complexity, implementation cost, and risks.
4. Include at least one reference implementation pattern we can mirror.

This is especially useful before Phase 3/Phase 4 decisions.

## Definition of Done

1. Highlighting render performance improved with benchmark evidence.
2. Visible output quality maintained or improved.
3. REPL and file-source highlighting both stable.
4. Tests cover key token classes and multiline edge cases.
5. No regressions in smalltalk-inspector interaction behavior.
