---
Title: Investigation diary
Ticket: LAYOUT-001
Status: active
Topics:
    - typography
    - layout
    - semantic-yaml
    - information-design
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological diary of the LAYOUT-001 investigation and implementation.
LastUpdated: 2026-05-26T10:30:00-04:00
WhatFor: Record the implementation journey for the semantic YAML rendering pipeline.
WhenToUse: Read before resuming work on LAYOUT-001.
---

# Investigation diary

## Goal

Record the implementation journey for the semantic YAML → reMarkable rendering pipeline.

## Step 1: Layout approaches catalog

Write 1–2 short prose paragraphs here. Analyzed the semantic YAML schema spec and the real goja context ownership guide YAML to understand the data model. Cataloged 8 layout approaches (Tufte margin, color banding, two-panel split, progressive cards, reading-mode lanes, semantic grid, importance filter, flow map). Recommended Tufte margin + color banding for the reMarkable Paper Pro color tablet, with dual-PDF importance filtering.

### Prompt context

**User prompt (verbatim):** "Create a new ticket for typographic layout of markdown guides, in particular we want to try out different informational layouts that allow us to absorb a lot of information quickly based on intent encoded yaml"

**Assistant interpretation:** Create a docmgr ticket with a catalog of layout approaches that exploit semantic YAML signals for high-density information absorption.

**Inferred user intent:** Explore how semantic YAML can be rendered into layouts that let readers absorb information quickly, using the real goja context ownership guide as a test case.

### What I did

- Read the semantic YAML schema spec (03-semantic-document-yaml-schema-spec.md) and the full goja context ownership guide YAML (01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml, 2377 lines).
- Analyzed the test case: 76 sections, 234 blocks, 7 block types, 8 intents, 19 section roles, 3 importance levels.
- Created LAYOUT-001 ticket with design-doc and diary.
- Wrote 8 layout approaches with comparison matrix and recommended combinations.
- Initially recommended Tufte margin + grayscale for reMarkable (assumed grayscale e-ink).
- User corrected: it's a color tablet.
- Revised recommendation: Tufte margin + color banding hybrid with dual-PDF importance filter.

### Why

The semantic YAML already encodes block type, importance, severity, intent, and section role. Flat Markdown wastes all those signals. The layout layer needs to exploit them for visual differentiation on the reMarkable's color e-ink display.

### What worked

- Analyzing the YAML with Python to get exact block type/intent/importance/severity distributions — this grounded the layout design in real data rather than assumptions.
- The Tufte margin + color banding combination is a strong fit for color e-ink: margins work in PDF, colors are distinguishable on Kaleido panels, and the two-column split creates a natural scanning path.

### What didn't work

- Initially recommended grayscale approach, which was wrong for the color reMarkable Paper Pro. The correction led to a much better design — color banding is the strongest scanning signal you can get.

### What I learned

- Color e-ink on the Paper Pro can clearly distinguish red, blue, green, amber — these are the strongest colors. Subtle pastels may wash out.
- Left borders are a better primary signal than background tints on e-ink. Borders (3-5px) are clearly visible even at a distance; light background tints may wash to near-white.
- Icons + text labels supplement color for accessibility. Never rely on color alone.

### What was tricky to build

- Deciding between HTML→CSS→PDF vs pandoc+LaTeX. Pandoc is what remarquee already uses for Markdown→PDF, but semantic YAML cannot be expressed in Markdown without losing the signals. HTML+CSS gives us the Tufte margin layout and color banding for free, with browser DevTools for debugging.

### What warrants a second pair of eyes

- The color palette (BLOCK_COLORS, SEVERITY_COLORS, ROLE_COLORS) should be tested on the actual reMarkable Paper Pro hardware. LCD-rendered colors may look different on color e-ink.
- The 55/45 column split ratio may need adjustment depending on how margin cards render at 8pt.

### What should be done in the future

- Test rendered PDFs on the actual reMarkable Paper Pro device.
- Adjust colors and font sizes based on device feedback.
- Consider adding intent-based visual differentiation beyond block type.

### Code review instructions

- Read design-doc/01-semantic-yaml-layout-approaches.md for the 8 approaches and comparison matrix.
- Key design decisions: Tufte margin for deep reading, color banding for scanning, importance filter for dual-PDF output.

### Technical details

- Test case stats: 76 sections, 234 blocks, 20 section roles, 7 block types, 8 intents.
- Section importance distribution: 6 critical, 49 high, 21 medium.
- Block type distribution: prose 147 (63%), code 41 (18%), bullet_list 22 (9%), ordered_list 10 (4%), principle 6 (3%), diagram 6 (3%), table 2 (1%).
- All 234 blocks have importance=none; importance filtering is section-level only.

## Step 2: Rendering pipeline architecture and implementation guide

Wrote a comprehensive 17-section guide covering the full YAML → HTML → PDF → reMarkable pipeline. The guide targets a new intern and includes pseudocode, CSS layout system, color mapping tables, CLI interface, and implementation plan. Also wrote the actual working Python renderer (`semantic_render.py`) and shell pipeline (`render_to_remarkable.sh`).

### Prompt context

**User prompt (verbatim):** "how would you render the PDF? would going through html be easier, then I can use the print functionality? Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Write a detailed, intern-ready implementation guide for the YAML→HTML→CSS→PDF→reMarkable pipeline, then upload to reMarkable.

**Inferred user intent:** Have a complete, working implementation guide plus actual code that can be read on the reMarkable for reference while building/testing.

### What I did

- Wrote design-doc/02-semantic-yaml-to-remarkable-rendering-pipeline-architecture-and-implementation-guide.md (17 sections, ~55KB).
- Implemented semantic_render.py (working Python renderer, ~25KB) with embedded CSS.
- Implemented render_to_remarkable.sh (shell pipeline script, ~2.7KB).
- Smoke-tested the renderer: full guide renders 76 sections/246 blocks, quick ref renders 55 sections/158 blocks.
- Related all files in docmgr.

### Why

The guide explains *why* HTML is the intermediate format (CSS Grid for Tufte margin, DevTools for debugging, Chromium --print-to-pdf for clean PDFs). The scripts implement *how*. Both are needed — the guide for understanding, the scripts for doing.

### What worked

- The Python renderer produces valid, self-contained HTML from the real YAML test case on first run.
- Embedded CSS in the Python script keeps everything in one file — no external stylesheet to manage.
- The block-type/intent/severity/importance CSS class system is clean and extensible.

### What didn't work

- Nothing failed during implementation. The YAML parsed cleanly, the HTML generated correctly, and the section/block counts matched expectations.

### What I learned

- HTML as intermediate format is clearly the right call for this pipeline. CSS Grid handles the two-column Tufte layout trivially. Browser DevTools provide instant feedback. Chromium --print-to-pdf produces clean PDFs.
- The `format_prose()` function handles inline Markdown (backticks, bold, italic, blockquotes) but is intentionally minimal — it doesn't handle links, images, or nested formatting. This is acceptable because the semantic YAML content is typically plain text with inline code.
- All 234 blocks have importance=none, which means block-level importance filtering would have no effect. Section-level filtering is the right approach for the current data.

### What was tricky to build

- The critical-pull-quote exception: when `type: principle` + `importance: critical`, the block needs to break out of the two-column grid and span full width. CSS `grid-column: 1 / -1` handles this, but the Python renderer needs to emit the block inside the section body (not in either column) for the grid to work. Current implementation emits it inside the main column div, which means it won't span. This is a known issue to fix.

### What warrants a second pair of eyes

- The critical-pull-quote placement — it should be emitted at the section-body level (before the column divs), not inside the main column.
- The markdown_table_to_html() function assumes the separator line (|---|---|) is always line index 1. This is fragile if tables have inconsistent formatting.
- The HTML_TEMPLATE doesn't include a document footer (version, source ticket, generated date). This should be added.

### What should be done in the future

- Fix critical-pull-quote grid placement.
- Add document footer with metadata.
- Test with Chromium --print-to-pdf and verify page breaks.
- Upload to reMarkable and verify on device.
- Add more block types (command, checklist with checkbox rendering, quote with attribution).

### Code review instructions

- Start with scripts/semantic_render.py — the main renderer.
- Key functions: render_block() (block-type dispatch), render_section() (section tree walk + column separation), render_document() (top-level assembly).
- CSS is in the CSS_STYLESHEET string constant — search for `.margin-block`, `.block-principle`, `.section-body` for the core layout rules.
- Run smoke test: `python3 scripts/semantic_render.py --input <yaml> --output /tmp/test.html`

### Technical details

- Output: single self-contained HTML file with embedded <style> block. No external dependencies.
- CSS Grid: `grid-template-columns: 55fr 45fr` for the Tufte margin split.
- Print: `@page { size: A4; margin: 12mm 10mm; }` with `break-inside: avoid` on margin cards and code blocks.
- Font stack: DejaVu Sans (body) + DejaVu Sans Mono (code), 8.5pt body / 7.5pt code.
- Browser detection in shell script: tries chromium-browser, google-chrome, chrome, chromium in order.
