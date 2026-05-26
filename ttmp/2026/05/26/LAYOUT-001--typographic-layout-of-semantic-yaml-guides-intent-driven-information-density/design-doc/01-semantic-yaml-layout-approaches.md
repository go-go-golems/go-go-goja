---
Title: Semantic YAML layout approaches
Ticket: LAYOUT-001
Status: active
Topics:
    - typography
    - layout
    - semantic-yaml
    - information-design
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/03-semantic-document-yaml-schema-spec.md
      Note: Schema spec defining the semantic YAML structure that layouts must consume
    - Path: ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml
      Note: Real semantic YAML document serving as primary test case for layout experiments
ExternalSources: []
Summary: Catalog of layout approaches for rendering intent-encoded semantic YAML into high-density, scannable documents. Each approach exploits a different combination of the semantic signals (block type, importance, severity, intent, section role) to maximize information absorption.
LastUpdated: 2026-05-26T09:30:00-04:00
WhatFor: Give designers and implementers concrete layout strategies to try when rendering semantic YAML into PDF, web, or print formats where information density and rapid comprehension matter.
WhenToUse: Use when designing a renderer/converter for semantic-document/v1 YAML, choosing a layout strategy for a specific audience or reading mode, or prototyping how the goja context ownership guide should look in a new format.
---

# Semantic YAML layout approaches

## Executive summary

The semantic YAML schema separates *what content means* from *how it looks*. This document catalogs concrete layout approaches that exploit the semantic signals — block type, importance, severity, intent, section role — to produce documents that let readers absorb dense technical material quickly. Each approach has a distinct information-density strategy, a different idea of what "quickly" means, and different tradeoffs.

The test case is the goja context ownership guide (2377 lines of semantic YAML, 20 sections, 234 blocks spanning prose, principle, code, diagram, risk, decision, table, bullet_list, ordered_list, and checklist). That document is high-density, multi-audience (intern + maintainer + package author + designer), and supports three reading modes (deep read, implementation reference, code review checklist). A good layout approach should serve at least two of those modes well.

## Problem statement

Markdown renders everything as flat text with heading-level hierarchy. A principle, a risk, and a transitional paragraph look the same. A definition and a narrative explanation get the same typographic treatment. For a 20-section technical guide with code, diagrams, API sketches, risks, and implementation phases, flat Markdown forces the reader to parse every word to find what they need.

The semantic YAML already encodes the distinctions. The layout layer needs to exploit them. The question is *how* — which spatial, typographic, and structural strategies make intent-encoded content faster to absorb?

## Design constraints from the source document

The goja context ownership guide has these structural properties that any layout must accommodate:

1. **Deep nesting**: 4 levels of section hierarchy (H1→H2→H3→H4), with meaningful content at every level.
2. **Mixed block types**: prose, principle, definition, risk, decision, code, command, diagram, table, bullet_list, ordered_list, checklist — often interleaved within a single section.
3. **Importance spread**: critical (executive summary, proposed architecture, vocabulary), high (most sections), medium (implementation phases, sub-structures), low (minor asides).
4. **Severity signals**: risk blocks carry severity (high/medium/low) that is independent of importance.
5. **Intent diversity**: orient-reader, explain, define-core-term, remember, warn, compare-or-classify, show-api-or-pseudocode, show-js-example, show-command, explain-flow, enumerate, validate, implement.
6. **Long code/diagram blocks**: some blocks are 30+ lines of Go pseudocode or ASCII sequence diagrams.
7. **Multi-audience**: an intern reads linearly; a maintainer jumps to API reference; a reviewer wants the checklist.
8. **Reading modes**: deep_read (sequential), implementation_reference (jump-to), code_review_checklist (scan-only).

---

## Layout approaches

### Approach 1: Layered margin sidebar (the "Tufte margin" approach)

**Core idea**: Use the right margin as a semantic sidebar. Principles, definitions, decisions, and risks float into the margin alongside the main prose flow. The central column becomes a clean narrative; the margin carries the "sticky" information a reader should remember.

**How semantic signals map to layout**:

| Signal | Layout treatment |
| --- | --- |
| `type: principle` + `importance: critical` | Full-width pull-quote box spanning main + margin |
| `type: principle` + `importance: high` | Right-margin note with rule icon |
| `type: definition` | Margin card with term bolded |
| `type: risk` | Margin panel with severity-colored left border (red=high, amber=medium) |
| `type: decision` | Margin box with "Decision" label |
| `type: code` | Full-width code block in main column |
| `type: diagram` | Full-width, slightly indented |
| `type: prose` | Main column body text |
| `role: gap-analysis` | Section gets a left-border accent stripe |
| `role: proposed-architecture` | Section gets a different accent stripe |

**Information density strategy**: The margin absorbs everything that is "remember this" or "watch out for this," leaving the main column as a readable narrative. A scanner reads the margin notes to get the 80% takeaway without reading the prose. A deep reader gets both in parallel.

**Strengths**:
- Natural for technical books (Tufte, O'Reilly sidebars)
- Margins work in PDF and print; web scroll can reflow them as callouts
- Clean separation of "story" vs. "rules"
- Works well for the intern's deep_read mode

**Weaknesses**:
- Margin space is limited (especially on narrow screens)
- If a section has many principles/risks, margin stacks vertically and loses the parallel-reading benefit
- Doesn't help the "jump to API reference" reading mode without additional navigation
- Code blocks that sit next to margin notes can feel cramped

**When to use**: Documents where the narrative flow matters and the audience reads linearly or near-linearly. Best when the ratio of "sticky" blocks (principle, definition, risk, decision) to prose is moderate (1:3 to 1:5).

---

### Approach 2: Block-type color banding (the "newspaper layout" approach)

**Core idea**: Every block gets a thin colored left border or top band based on its type. Sections get background tinting based on their role. The page becomes a color-coded map of information types. A reader can scan the left edge and know: "this stretch is all code," "here are three principles," "this section is all risks."

**How semantic signals map to layout**:

| Signal | Layout treatment |
| --- | --- |
| `type: principle` | Blue left border + ☑ icon + bold first sentence |
| `type: definition` | Purple left border + 📖 icon + term in bold |
| `type: risk` | Red/orange left border (severity maps to red→orange→yellow) + ⚠ icon |
| `type: decision` | Green left border + ✦ icon |
| `type: code` | Dark gray background block, no left border, language label |
| `type: diagram` | Monospace block with light background, "Diagram" label |
| `type: table` | Standard table with header row, alternating row tint |
| `type: checklist` | Checkbox cards, each item as a card row |
| `role: implementation-phase` | Light green section background |
| `role: risk-analysis` | Light red section background |
| `role: concept-foundation` | Light blue section background |
| `role: authoring-guidelines` | Light yellow section background |
| `importance: critical` | Block gets a top accent bar in addition to left border |

**Information density strategy**: Visual scanning replaces reading. The color language lets a reviewer's eye jump to all risks (red), all principles (blue), all decisions (green). Section background tinting creates a spatial map of the document's rhetorical structure.

**Strengths**:
- Extremely scannable — a reader can locate every risk in 2 seconds
- Scales to any page width (borders don't need margin space)
- Works well for code_review_checklist mode
- Easy to implement in CSS/PDF renderers
- Dense — no whitespace cost for the color coding

**Weaknesses**:
- Can look busy or "traffic-light" if overused
- Doesn't help with *understanding* the narrative flow — it's a visual index, not a reading aid
- Color alone doesn't communicate meaning to colorblind readers (needs icons/patterns)
- Section backgrounds can collide with block borders visually

**When to use**: Documents where scanning is the primary reading mode, where the audience already knows the domain and needs to find specific things fast. Best for code_review_checklist and implementation_reference modes.

---

### Approach 3: Two-panel split (the "Dash/Zeal doc" approach)

**Core idea**: Split the page into a narrow left navigation panel and a wide right content panel. The navigation panel shows the section hierarchy with semantic role icons and importance indicators. Clicking/tapping a section scrolls the content panel. The content panel renders blocks with minimal decoration — just enough to distinguish types.

**How semantic signals map to layout**:

| Signal | Layout treatment |
| --- | --- |
| Section hierarchy | Left panel tree with role-based icons (⚙ architecture, ⚠ risk, ✏ authoring, 📋 implementation, 🔍 gap-analysis, 💡 concept) |
| `importance: critical` | Section name in bold in nav + ★ icon |
| `importance: high` | Normal weight in nav |
| `importance: medium/low` | Dimmed in nav |
| `type: principle` | Content: boxed paragraph with subtle background |
| `type: code` | Content: full-width code block |
| `type: risk` + `severity: high` | Content: red-topped warning box |
| `type: diagram` | Content: full-width diagram panel |
| Block `intent` | Shown as small tag above the block (e.g., `[define-core-term]`, `[show-api]`) |

**Information density strategy**: The navigation panel provides a persistent map. A reader never loses their position in a 20-section document. The content panel is clean because navigation is handled externally. Importance and role signals in the nav let a reader skip entire sections they don't need.

**Strengths**:
- Best for jump-to reading (implementation_reference mode)
- Navigation is always visible — no scrolling back to the top
- Section-level semantic signals (role, importance) are leveraged for navigation, not just decoration
- Familiar pattern (Dash, Zeal, many API doc sites)

**Weaknesses**:
- The nav panel eats 20-30% of horizontal space
- On narrow screens, the split collapses and you're back to a single column with a hamburger menu
- Doesn't improve the *content* layout itself — blocks in the content panel still need their own treatment
- Not great for linear deep reading (nav panel is a distraction)

**When to use**: Documents with many sections (10+) where the audience wants to jump directly to specific parts. Best for implementation_reference mode. Pair with Approach 1 or 2 for the content panel itself.

---

### Approach 4: Progressive disclosure cards (the "dashboard" approach)

**Core idea**: Render each block as a card. Cards have a header showing the block type (as an icon + label) and a collapsed preview (first line or summary). Critical/high-importance cards are expanded by default; medium/low cards are collapsed. The reader expands what they need. Section headers are card-group headers with the role badge and importance indicator.

**How semantic signals map to layout**:

| Signal | Layout treatment |
| --- | --- |
| `importance: critical` | Card expanded by default, bold border, ★ badge |
| `importance: high` | Card expanded by default |
| `importance: medium` | Card collapsed by default, shows first 80 chars |
| `importance: low` | Card collapsed, dimmed header |
| `type: principle` | Card icon: ☑, blue accent |
| `type: risk` + `severity: high` | Card icon: ⛔, red accent, expanded by default regardless of importance |
| `type: code` | Card icon: `</>`, shows language + first 5 lines when collapsed |
| `type: diagram` | Card icon: 📐, shows caption when collapsed |
| `type: table` | Card icon: 📊, shows header row when collapsed |
| `role: implementation-phase` | Section header: green badge "Phase" + phase number |
| `role: gap-analysis` | Section header: amber badge "Gap" |

**Information density strategy**: The default view shows only critical content + section headers. A reader can absorb the full document's shape in one screen. Then they selectively expand the blocks they need. For code_review_checklist mode, start with only `type: principle` + `type: risk` + `type: checklist` expanded.

**Strengths**:
- Highest initial information density — the whole document's shape fits on one screen
- Reader controls density level interactively
- Perfect for the "I need the 5-minute version" reader
- Natural fit for web (CSS collapsible) and PDF (two renderings: full and summary)
- Importance and severity directly control initial state

**Weaknesses**:
- Collapsed cards hide content that *might* matter — the reader has to guess from the 80-char preview
- Can feel "clicky" for deep reading (expand, expand, expand)
- Doesn't work well in print unless you render a "full" version
- Card chrome (headers, icons, borders) adds visual noise when many cards are expanded

**When to use**: Documents where the audience is time-constrained and needs to choose what to read. Best for multi-audience documents where different readers want different subsets. Pairs well with reading_mode filtering.

---

### Approach 5: Reading-mode lanes (the "swim lane" approach)

**Core idea**: Use the `reading_modes` field from the document frontmatter to produce multiple renderings of the same content, each optimized for one mode. Each lane filters and reorders blocks based on intent relevance to that reading mode.

**How semantic signals map to layout**:

For `deep_read`:
- Render all blocks in document order
- Use Approach 1 (Tufte margin) for block type differentiation
- Full section hierarchy visible

For `implementation_reference`:
- Filter to blocks with `intent` in [show-api-or-pseudocode, show-js-example, show-command, implement, define-core-term]
- Include `type: principle` with `importance: critical` as pull quotes
- Suppress `intent: orient-reader` and `intent: explain` prose blocks
- Reorder: API blocks first, then principles, then risks

For `code_review_checklist`:
- Filter to blocks with `type` in [principle, risk, decision, checklist]
- Group by section role
- Render as a numbered checklist with section context
- Include code blocks only when they appear inside a principle or decision

**Information density strategy**: Instead of one document that tries to serve all readers, produce mode-specific documents. Each lane is sparse by design because irrelevant content is filtered out.

**Strengths**:
- Best reading experience per mode — no compromises
- Exploits the intent and reading_modes signals directly
- Each lane can use a different sub-approach (Tufte for deep read, color bands for checklist)
- Produces natural deliverables: a "quick reference card," a "review checklist," a "full guide"

**Weaknesses**:
- Requires multiple rendering passes or a mode selector
- Content that is filtered out is invisible — a reader in checklist mode might miss a critical explanation
- Requires accurate intent tagging in the YAML (garbage in, garbage out)
- "Which lane am I in?" can confuse readers who expect one document

**When to use**: Documents that already declare multiple reading modes and have well-tagged intents. Best when the audience is clearly segmented. Works as a top-level strategy that composes with any of the other approaches for each lane.

---

### Approach 6: Semantic grid with spatial zones (the "newspaper front page" approach)

**Core idea**: Lay out sections on a grid where position carries semantic meaning. The top-left zone is for `role: summary` and `role: problem-statement`. The top-right is for `role: proposed-architecture` and `role: architecture-map`. The left column is for `role: concept-foundation` and `role: exposition`. The right column is for `role: gap-analysis`, `role: risk-analysis`, and `role: open-questions`. The bottom spans the full width for `role: implementation-plan` and `role: migration-guide`.

**How semantic signals map to layout**:

| Section role | Grid position |
| --- | --- |
| `summary`, `problem-statement` | Top banner, full width |
| `concept-foundation` | Left column |
| `proposed-architecture`, `architecture-map` | Right column, top |
| `gap-analysis`, `risk-analysis` | Right column, middle |
| `authoring-guidelines`, `authoring-rule` | Left column, middle |
| `implementation-plan`, `implementation-phase` | Bottom, full width, numbered |
| `diagram-section` | Center inset or full width |
| `test-strategy`, `open-questions` | Right column, bottom |
| `conclusion` | Bottom banner |

Within each zone, blocks use Approach 2 (color banding) for block-type differentiation.

**Information density strategy**: The spatial arrangement means a reader knows *where to look* for a certain kind of content. Risks are always on the right. Concepts are always on the left. Implementation is always at the bottom. The document becomes a map, not a scroll.

**Strengths**:
- Extremely high information density — many sections visible at once
- Spatial memory aids navigation (after reading once, you know where things are)
- Exploits section_role signals, which are often ignored in simpler layouts
- Works well as a poster/one-page overview

**Weaknesses**:
- Requires a large canvas (A3 print or large screen) — doesn't work on phone or narrow PDF
- Sections have varying heights; grid alignment is hard without manual tuning
- Not suited to sequential reading — the left-right split means the reading order is ambiguous
- Hard to automate — requires a layout engine that understands spatial semantics
- Sections with many nested subsections may not fit their assigned zone

**When to use**: Documents where an overview/survey reading is the primary need, and the output is a large-format print or a dashboard screen. Best as a companion to a linear rendering, not a replacement.

---

### Approach 7: Layered overlay with importance filtering (the "x-ray" approach)

**Core idea**: Render the document in layers. The base layer is all content at `importance: low`. Each successive layer adds higher importance content with progressively stronger visual treatment. A reader can "dial" the importance filter to see only critical content, critical+high, critical+high+medium, or everything.

**How semantic signals map to layout**:

| Filter level | Visible content | Visual treatment |
| --- | --- | --- |
| Critical only | `importance: critical` blocks | Bold, colored, boxed, larger type |
| Critical + High | + `importance: high` blocks | Normal body type |
| + Medium | + `importance: medium` blocks | Slightly smaller, muted |
| All | + `importance: low` blocks | Footnote-style, smallest type |

Within any filter level, block types get Approach 2 color banding.

**Information density strategy**: The importance slider lets a reader choose their density. At "critical only," a 20-section document becomes a 1-page executive brief. At "all," it's the full guide.

**Strengths**:
- Directly exploits the importance signal, which is the most underused semantic field
- Natural for documents with a clear importance hierarchy
- Works in PDF (render multiple versions) and web (interactive slider)
- Scales from 1-page summary to full reference

**Weaknesses**:
- Importance is relative within a document — a "medium" block in a critical section may be more important than a "high" block in a minor section
- Removes narrative context when low-importance explanatory prose is hidden
- Requires careful importance assignment in the source YAML
- The "all" view is identical to a simpler layout — this approach only adds value at the filtered levels

**When to use**: Documents where importance has been carefully assigned and the audience needs both a summary and a deep dive from the same source. Pairs well with Approach 5 (reading-mode lanes).

---

### Approach 8: Timeline/circuit diagram (the "flow map" approach)

**Core idea**: Treat the document as a directed graph of concepts. Sections become nodes on a vertical timeline or circuit. Edges represent logical dependencies (e.g., "Vocabulary" depends on "Conceptual model"; "Implementation plan" depends on "Proposed API direction"). Each node expands to show its blocks when focused. The overall shape gives a structural overview; expanding a node gives the detail.

**How semantic signals map to layout**:

| Signal | Layout treatment |
| --- | --- |
| `section.role` | Node color/shape (rectangle=exposition, diamond=risk, hexagon=architecture, circle=concept) |
| `section.importance` | Node size |
| `section.summary` | Node tooltip/preview |
| Block types within a section | Shown when node is expanded, using Approach 2 color banding |
| `role: implementation-phase` | Nodes on the timeline portion with numbered sequence |
| Cross-references between sections | Directed edges in the graph |

**Information density strategy**: The graph view gives a structural understanding in seconds. The expanded view gives full content. The two levels match the two cognitive modes: "where am I?" and "what does this say?"

**Strengths**:
- Best for documents with strong section dependencies
- Visualizes the "why this order" question that flat documents don't answer
- Works well for onboarding (the intern can see the conceptual map before reading)
- Exploits section-level semantics (role, importance, summary) at the overview level

**Weaknesses**:
- Hard to lay out automatically without a layout algorithm (dot/Graphviz, dagre)
- Not all documents have clean dependency graphs — some sections are just parallel
- Doesn't work in print without a foldout
- Implementation complexity is high
- Reading the full content still requires expanding every node, which is effectively linear

**When to use**: Documents with clear section dependencies (concept → architecture → plan → phases). Best as a companion overview to a linear rendering.

---

## Comparison matrix

| Approach | Primary mode | Density | Print | Web | Implementation | Best for |
| --- | --- | --- | --- | --- | --- | --- |
| 1. Tufte margin | deep_read | Medium | ✅ | ✅ | Medium | Narrative guides, books |
| 2. Color banding | code_review, impl_ref | High | ✅ | ✅ | Easy | Scanning, checklists |
| 3. Two-panel split | impl_ref | Medium | ⚠️ | ✅ | Medium | API docs, long guides |
| 4. Progressive cards | all modes | Very high (collapsed) | ⚠️ | ✅ | Medium | Multi-audience, dashboards |
| 5. Reading-mode lanes | mode-specific | Mode-dependent | ✅ | ✅ | Medium | Clearly segmented audiences |
| 6. Semantic grid | overview/survey | Very high | ⚠️ (large) | ✅ (large) | Hard | Posters, one-page overviews |
| 7. Importance filter | summary→full | Adjustable | ✅ | ✅ | Easy | Executive briefs, layered docs |
| 8. Flow map | onboarding | Low (overview) | ❌ | ✅ | Hard | Conceptual onboarding |

## Recommended combinations for the test case

The goja context ownership guide has 20 sections, mixed block types, multi-audience, and three declared reading modes. The most effective strategy would combine:

1. **Approach 5 (reading-mode lanes) as the top-level strategy** — produce three renderings for the three declared reading modes.

2. **For the `deep_read` lane**: Approach 1 (Tufte margin) — the narrative flow matters, and the many principle/risk/decision blocks deserve margin treatment.

3. **For the `implementation_reference` lane**: Approach 3 (two-panel split) + Approach 2 (color banding in the content panel) — the maintainer needs fast navigation and block-type scanning.

4. **For the `code_review_checklist` lane**: Approach 7 (importance filter at critical+high) + Approach 2 (color banding) — the reviewer wants to see only what matters and spot risks/principles instantly.

5. **As an optional companion**: Approach 8 (flow map) showing the section dependency graph — useful for the intern's first encounter with the document.

## Open questions

1. Should the layout rules be encoded in the YAML itself (as `layout_rules` in the schema spec), in a separate layout definition file, or in the renderer code? The schema spec shows `layout_rules` as an example but scopes them out.
2. Can a single YAML source produce all recommended combinations without manual per-section tuning?
3. What is the minimum viable renderer? Approach 2 (color banding) is easiest; does it deliver enough value to start with?
4. How do we handle blocks that have multiple semantic signals that conflict in layout treatment (e.g., a `type: risk` with `importance: low` — is it still red-banded)?
5. Should the renderer support interactive importance filtering (Approach 7) or only static renderings?
6. How does the layout strategy change for shorter documents (5-10 sections) vs. the current 20-section test case?

## References

- Semantic YAML schema spec: `ttmp/2026/05/25/XGOJA-014/.../reference/03-semantic-document-yaml-schema-spec.md`
- Test case semantic YAML: `ttmp/2026/05/25/XGOJA-014/.../design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml`
- Edward Tufte, *The Visual Display of Quantitative Information* — margin notes, sparklines, data-ink ratio
- Neville Brody, *The Graphic Language of Neville Brody* — zone-based newspaper layouts
- Dash/Zeal doc format — split-panel API documentation browsers
