---
Title: 'Intern Research Guide: F and G'
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
RelatedFiles: []
ExternalSources:
    - https://tree-sitter.github.io/tree-sitter/
    - https://tree-sitter.github.io/tree-sitter/using-parsers/queries/
    - https://github.com/tree-sitter/tree-sitter/blob/master/lib/include/tree_sitter/api.h
    - https://unpkg.com/monaco-editor@latest/esm/vs/editor/editor.api.d.ts
    - https://github.com/microsoft/vscode/blob/main/src/vs/editor/common/model/textModelTokens.ts
    - https://code.visualstudio.com/blogs/2017/02/08/syntax-highlighting-optimizations
    - https://tree-sitter.github.io/tree-sitter/creating-parsers#using-grammarjs
    - https://tree-sitter.github.io/tree-sitter/3-syntax-highlighting.html
Summary: Internet-only research guide for an intern to investigate F and G with no prior context, including explicit pattern catalogs (F0-F4, G0-G4), architectural primers, and implementation-oriented deliverables based on public APIs/references.
LastUpdated: 2026-02-15T16:10:00Z
WhatFor: Provide a concrete research plan when intern access is limited to public internet resources.
WhenToUse: Use when assigning API/reference deep research without private repository access.
---

# Intern Research Guide: F and G (Internet-Only)

## Constraint

The intern has **no access to our private codebase**.  
This guide is intentionally designed for internet-only execution.

## Purpose

Research two technical tracks and deliver implementation-ready guidance:

1. **F:** tree-sitter query/capture semantic highlighting.
2. **G:** incremental parsing + dirty-range invalidation.

The output should be a decision package we can map onto our code later.

## Definitions (use these exactly)

1. **F (semantic capture track):** classification quality.
   Use tree-sitter query captures to identify semantic categories beyond simple token kinds.
2. **G (incremental update track):** update efficiency.
   Recompute only changed/affected regions rather than full-document highlight passes.
3. **F and G are complementary.**
   F improves correctness/semantic expressiveness, G improves latency/throughput.

## Zero-Context Primer (Read This First)

If you are new to syntax highlighting, start here.

### What syntax highlighting is

Syntax highlighting is the process of:

1. reading source text,
2. deciding what each piece of text *means* (keyword, function, comment, property, etc.),
3. mapping those meanings to style categories,
4. rendering the styled output.

### Why there are two tracks (F and G)

There are two orthogonal problems:

1. **Classification quality problem (F):** are we assigning the right meaning/class to each token?
2. **Update cost problem (G):** how much work do we do after changes/edits?

You can have:

1. good classifications but slow updates,
2. fast updates but poor classifications,
3. or both good if F and G are designed together.

### Minimal architecture vocabulary

Use these terms consistently in your report:

1. **Parser:** builds syntax structure from text.
2. **Token/classifier:** assigns categories to ranges.
3. **Span/segment index:** organizes ranges for fast lookup/rendering.
4. **Renderer:** converts classified ranges to styled output.
5. **Invalidation engine:** decides which parts must be recomputed after change.
6. **Cache:** stores previously computed render/classification artifacts.

### Typical pipeline (conceptual)

1. input text,
2. parse/capture/classify,
3. build line/range index,
4. render visible region,
5. on edits: invalidate affected ranges only.

### What commonly goes wrong

1. ambiguous classification precedence,
2. full-document recomputation on small edits,
3. stale cache after partial updates,
4. byte/column and Unicode alignment mistakes,
5. semantic layer and lexical layer conflicts.

## Explicit Pattern Catalog: F (Classification Patterns)

These are the concrete F patterns you must analyze. Use these exact IDs in your report.

## F0: Kind-Based Classification (baseline)

What it is:

1. classify tokens directly from parser node kinds/types.

Strengths:

1. simple and fast to implement,
2. low conceptual overhead.

Weaknesses:

1. semantics are coarse,
2. hard to distinguish context-specific roles cleanly.

When it fits:

1. initial implementation,
2. low semantic precision requirements.

## F1: Query Capture Overlay on Top of Baseline

What it is:

1. keep F0 baseline,
2. add tree-sitter query captures for higher-value semantic distinctions,
3. captures override or refine baseline where matched.

Strengths:

1. incremental adoption,
2. preserves fallback behavior.

Weaknesses:

1. overlap/precedence policy must be explicit,
2. possible dual-path complexity.

When it fits:

1. migration from coarse to richer semantics with low risk.

## F2: Capture-First Classification

What it is:

1. primary classification comes from query captures,
2. node-kind fallback only for uncovered tokens.

Strengths:

1. strongest semantic flexibility,
2. language-specific tuning via queries.

Weaknesses:

1. higher maintenance burden,
2. gaps in query coverage can cause inconsistencies.

When it fits:

1. mature highlighting systems with strong query discipline.

## F3: Multi-Layer Semantic Model

What it is:

1. lexical/base classes plus semantic overlay classes,
2. rendering merges layers by priority.

Strengths:

1. robust fallback path,
2. can evolve semantics without destabilizing lexical baseline.

Weaknesses:

1. layer merge logic can be complex,
2. conflicts must be deterministic.

When it fits:

1. systems needing high resilience and gradual semantic rollout.

## F4: External Semantic Provider Protocol

What it is:

1. semantic classifications delivered by a provider protocol (for example semantic tokens with legends/deltas),
2. combined with local lexical tokenization.

Strengths:

1. can leverage rich external analyzers,
2. supports dynamic semantic updates.

Weaknesses:

1. provider latency/availability concerns,
2. integration complexity.

When it fits:

1. IDE-like environments with language-service infrastructure.

## F Pattern Research Requirements

For each F pattern (F0-F4), collect:

1. required APIs,
2. precedence/conflict model,
3. performance impact expectations,
4. correctness risks,
5. migration feasibility from simple baseline.

## Explicit Pattern Catalog: G (Update/Invalidation Patterns)

These are the concrete G patterns you must analyze. Use these exact IDs in your report.

## G0: Full Rebuild on Any Change

What it is:

1. parse and re-highlight entire document on every update.

Strengths:

1. simplest correctness model,
2. lowest implementation complexity.

Weaknesses:

1. poor scalability,
2. unnecessary work for local edits.

When it fits:

1. very small documents or early prototypes.

## G1: Line-State Retokenization With Stop-on-Equal-State

What it is:

1. tokenize line-by-line,
2. each line produces an end state,
3. after edits, retokenize forward until end state matches prior state.

Strengths:

1. strong practical efficiency in many editors,
2. predictable invalidation boundaries.

Weaknesses:

1. state model must be correct and comparable (`equals` semantics),
2. can still cascade far in worst-case edits.

When it fits:

1. top-down tokenization architectures.

## G2: Incremental Parse Tree + Dirty Ranges

What it is:

1. apply edit metadata to previous parse tree,
2. reparse incrementally,
3. derive changed ranges and invalidate only impacted highlights.

Strengths:

1. potentially strong performance for large documents,
2. tightly coupled with syntax structure changes.

Weaknesses:

1. edit bookkeeping complexity,
2. invalidation mapping from tree ranges to render ranges must be robust.

When it fits:

1. parser infrastructure supports incremental edits well.

## G3: Semantic Token Delta Updates

What it is:

1. semantic layer returns deltas/edits against previous token stream,
2. renderer applies only token differences.

Strengths:

1. efficient semantic refresh path,
2. lower transport/work for unchanged regions.

Weaknesses:

1. delta protocol correctness is subtle,
2. requires stable identity/versioning (`resultId`-like flows).

When it fits:

1. systems with semantic provider protocols.

## G4: Multi-Tier Cache With Range Invalidation

What it is:

1. separate caches for parse artifacts, classified segments, and styled lines,
2. invalidate by range and version keys.

Strengths:

1. high practical speed for repeated views,
2. keeps rendering responsive under scrolling/navigation.

Weaknesses:

1. stale-cache bugs if invalidation policy is incomplete,
2. memory management needed.

When it fits:

1. high-frequency render/update pipelines.

## G Pattern Research Requirements

For each G pattern (G0-G4), collect:

1. invalidation trigger model,
2. stop conditions,
3. cache key strategy,
4. asymptotic and practical performance implications,
5. failure modes and mitigations.

## F+G Composition Patterns (Integration Patterns)

Use these integration IDs in your comparison matrix.

## FG-A: Fast Lexical Baseline + Semantic Overlay (Recommended Default Pattern to Evaluate First)

1. lexical/high-speed path provides stable base colors quickly,
2. semantic layer refines classes asynchronously or incrementally,
3. invalidation primarily handled by G1/G4 or G2/G4 combinations.

Why important:

1. balances responsiveness and semantic quality,
2. robust fallback behavior.

## FG-B: Unified Semantic-First Pipeline

1. semantic captures/provider drive most classification directly,
2. lexical layer is minimal fallback.

Why important:

1. potentially cleaner semantics,
2. higher correctness risk if semantic coverage is incomplete.

## FG-C: Adaptive Hybrid by Document Size/Mode

1. lightweight mode for large or transient buffers,
2. richer semantic mode for stable/important views,
3. mode switching based on policy.

Why important:

1. operationally pragmatic for mixed workloads.

## Mandatory Comparison Output for Patterns

Your report must include one table with rows:

1. F0, F1, F2, F3, F4
2. G0, G1, G2, G3, G4
3. FG-A, FG-B, FG-C

Columns must include:

1. implementation complexity,
2. correctness risk,
3. expected latency impact,
4. maintenance cost,
5. recommended rollout order.

## What the Intern Should Not Do

1. Do not reference private files or guess private architecture details.
2. Do not claim repo-specific facts without explicit evidence from maintainers.
3. Do not produce only narrative summaries without concrete API references and design artifacts.

## Primary Research Corpus (Public Sources)

Minimum required sources:

1. Tree-sitter docs:
   parser API, query language, captures, edits/incremental update behavior.
2. Monaco public APIs:
   token providers, semantic tokens providers, tokenizer state interfaces.
3. VS Code tokenization architecture:
   line-state invalidation and retokenization model.
4. At least 2 additional editor/reference implementations (for example Helix/Neovim/Zed or similar) with public docs/source explaining F and/or G patterns.

Every design claim must include source URLs.

## Core Research Questions

### F-track

1. What capture models are common for JavaScript/TypeScript (function, property, parameter, type-like constructs)?
2. How do mature systems resolve capture overlap/priority?
3. How are semantic captures mapped to theme/token categories?
4. What is the minimum viable capture set for high impact?
5. What are typical failure modes (grammar drift, noisy captures, precedence bugs)?

### G-track

1. How do systems represent edit deltas and dirty ranges?
2. How do they determine recompute stop conditions?
3. How do they combine line-state invalidation with background tokenization?
4. What cache key/invalidation strategies are common?
5. What are tradeoffs between full-rebuild simplicity and incremental complexity?

### F+G integration

1. Where do F and G compose cleanly?
2. Which layering works best:
   lexical baseline + semantic overlay, or single capture-driven path?
3. What phased rollout pattern minimizes correctness regression risk?

## Required Outputs (Internet-Only)

The intern must deliver:

1. **API Reference Pack** (concise, source-linked):
   key APIs/signatures for tree-sitter, Monaco, and VS Code-relevant concepts.
2. **Architecture Comparison Matrix**:
   F/G mechanisms across at least 3 systems.
3. **Algorithm Option Memo**:
   3-5 concrete algorithm options with complexity, risk, and expected payoff.
4. **Integration Blueprint Template**:
   repo-agnostic blueprint with placeholders we can map to our codebase.
5. **Validation Plan**:
   measurable benchmarks/tests we should run once mapped internally.
6. **Final Recommendation Report (6+ pages)**:
   phased implementation order, risks, and open questions for maintainers.

If one of these is missing, research is incomplete.

## Deliverable Format Requirements

### 1) API Reference Pack

For each relevant API:

1. exact name/signature,
2. where it lives (URL),
3. what it solves (F/G),
4. caveats/version notes,
5. why it matters to implementation.

### 2) Comparison Matrix

Columns:

1. system/tool,
2. F mechanism,
3. G mechanism,
4. invalidation model,
5. cache model,
6. performance implications,
7. complexity/risk notes.

### 3) Blueprint Template

Must include:

1. component boundaries (parser, classifier, indexer, renderer, cache),
2. data flow (source -> parse -> classify -> index -> render),
3. invalidation flow (edit -> dirty range -> recompute),
4. rollout phases and rollback points.

No private file names are required; use abstract component names.

## Suggested Technical Focus Areas

1. Tree-sitter query syntax and capture precedence patterns.
2. Tree-sitter incremental parse model and edit propagation.
3. Semantic token delta protocols (`resultId`, edits).
4. Line-state tokenization with end-state equality stop conditions.
5. Segment-based rendering and line-level caching patterns.
6. Correctness safeguards for multiline constructs and Unicode/byte-column semantics.

## Research Method (Step-by-Step)

### Step 1: Build the source map

Create a bibliography table with:

1. URL,
2. source type (official docs/source/blog),
3. reliability level,
4. what topic it informs (F, G, or both).

### Step 2: Extract API-level facts

Pull only verifiable facts:

1. function/interface names,
2. required inputs/outputs,
3. update/invalidation semantics,
4. constraints and edge behavior.

### Step 3: Build model diagrams

Create two diagrams:

1. F-only pipeline,
2. G-only pipeline,

then a merged F+G pipeline.

### Step 4: Derive options

Produce 3-5 candidate architectures and compare with weighted criteria:

1. expected performance,
2. implementation complexity,
3. correctness risk,
4. testability,
5. maintainability.

### Step 5: Produce phase recommendation

Recommend phased rollout:

1. what should be implemented first,
2. what should be deferred,
3. why.

## Evidence Rules

1. No uncited claims.
2. Prefer primary sources (official API docs and source code).
3. When source behavior is inferred, label it as inference.
4. Include at least one concrete source quote or signature reference per major claim.

## Quality Bar

Recommendation quality is acceptable only if:

1. evidence-backed,
2. API-specific,
3. architecture-level (not vague),
4. phased and testable,
5. explicit about unknowns that require maintainer input.

## Questions the Intern Must Leave for Maintainers

Since intern cannot see private code, final report must include a section:

1. “Assumptions requiring confirmation”
2. “Information needed from maintainers before implementation”
3. “Potential integration blockers”

This section is mandatory.

## 10-Day Timeline (Internet-Only)

### Day 1-2

1. build bibliography,
2. extract API facts for tree-sitter/Monaco/VS Code.

### Day 3-4

1. deep dive F mechanisms,
2. draft capture precedence and mapping options.

### Day 5-6

1. deep dive G mechanisms,
2. draft invalidation/cache options.

### Day 7

1. compare F/G/F+G architectures,
2. score options.

### Day 8

1. draft 6+ page final report.

### Day 9

1. tighten citations,
2. clarify assumptions and open questions.

### Day 10

1. deliver final package,
2. present short walkthrough.

## Final Report Template

Use this section order:

1. Scope and constraints.
2. Source corpus and reliability.
3. F deep dive.
4. G deep dive.
5. F+G integration patterns.
6. Option matrix and scoring.
7. Recommended phased plan.
8. Risks and mitigations.
9. Maintainer questions and required confirmations.
10. Appendix with API references and citations.

## Common Failure Modes

1. Over-indexing on one editor implementation only.
2. Mixing semantic quality claims with no performance discussion.
3. Mixing performance claims with no invalidation model.
4. Giving repo-specific advice without repo access.
5. Recommending a final architecture without rollback strategy.

## Mentor Acceptance Checklist

Accept research only if:

1. all required outputs are present,
2. sources are primary and cited,
3. recommendation is phased and testable,
4. assumptions are explicit,
5. integration questions for maintainers are clear.
