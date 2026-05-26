---
Title: Semantic YAML to reMarkable rendering pipeline — architecture and implementation guide
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
    - Path: go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml
      Note: Primary test case — 76 sections, 234 blocks, 20 section roles
    - Path: go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/reference/03-semantic-document-yaml-schema-spec.md
      Note: Schema spec defining the semantic YAML structure that the renderer consumes
    - Path: go-go-goja/ttmp/2026/05/26/LAYOUT-001--typographic-layout-of-semantic-yaml-guides-intent-driven-information-density/design-doc/01-semantic-yaml-layout-approaches.md
      Note: Layout approaches catalog; this guide implements the recommended Tufte+color+importance combination
    - Path: go-go-goja/ttmp/2026/05/26/LAYOUT-001--typographic-layout-of-semantic-yaml-guides-intent-driven-information-density/scripts/render_to_remarkable.sh
      Note: Shell pipeline — YAML→HTML→PDF→reMarkable upload
    - Path: go-go-goja/ttmp/2026/05/26/LAYOUT-001--typographic-layout-of-semantic-yaml-guides-intent-driven-information-density/scripts/semantic_render.py
      Note: Python renderer — YAML→HTML with embedded Tufte margin CSS and color banding
ExternalSources: []
Summary: 'Complete architecture and implementation guide for rendering semantic-document/v1 YAML into high-density, color-coded PDFs for the reMarkable Paper Pro tablet. The pipeline is: YAML → Python parser → HTML+CSS → Chromium print-to-PDF → remarquee cloud put. Covers every component, the CSS layout system, the block-type-to-visual mapping, the importance-filtered dual-PDF output, and the reMarkable upload path.'
LastUpdated: 2026-05-26T10:00:00-04:00
WhatFor: Give an intern everything they need to understand, build, and debug the full rendering pipeline from semantic YAML to a finished document on a reMarkable Paper Pro.
WhenToUse: Use when implementing the renderer, debugging layout issues, adding new block types or section roles, or modifying the visual treatment of any semantic signal.
---


# Semantic YAML to reMarkable rendering pipeline

## 1. Executive summary

This guide describes a system that takes a semantic YAML document — one that encodes what content *means* (principle, risk, definition, code, diagram) rather than just what it *looks like* — and renders it into a high-density, color-coded PDF optimized for reading on the reMarkable Paper Pro color tablet.

The pipeline has four stages:

```
semantic YAML ──▶ Python renderer ──▶ HTML + CSS ──▶ Chromium print-to-PDF ──▶ reMarkable
```

The key design decision is **HTML as the intermediate format**. CSS gives us the Tufte-style margin layout, color-coded block-type borders, section role stripes, and print page-break control — all without writing a PDF generation library. Chromium's `--print-to-pdf` flag produces the final PDF. The `remarquee cloud put` command uploads it.

We produce **two PDFs** from every input: a full guide (all content) and a quick reference (critical + high importance sections only). Both use the same CSS layout.

The visual design is a **Tufte margin with color banding**: narrative prose and code flow in the main column; principles, risks, definitions, and decisions float into the right margin as color-bordered cards. Section headers carry thin role-colored stripes. Block types are distinguished by left-border color, icon, and background tint.

---

## 2. Problem statement

A semantic YAML file for a 20-section technical guide contains 76 sections and 234 blocks. The blocks have 7 distinct types (prose, code, bullet_list, ordered_list, principle, diagram, table), 8 intents (explain, show-api-or-pseudocode, enumerate, remember, explain-flow, show-example, show-js-example, compare-or-classify), 3 importance levels across sections (critical, high, medium), and 19 section roles (exposition, implementation-phase, authoring-rule, risk-analysis, case-study, concept-foundation, proposed-architecture, etc.).

Rendering this as flat Markdown produces a document where a principle, a risk, and a transitional paragraph look identical. The reader must parse every word to find what they need. On a reMarkable tablet, where reading is the primary activity, information density and visual differentiation matter more than on a desktop browser.

The rendering system must:
- Exploit all semantic signals (block type, intent, importance, severity, section role) for visual differentiation.
- Produce output that works on color e-ink (muted but clear reds, blues, greens, ambers).
- Support small body text (8–9pt) for high information density.
- Support two reading modes: full guide and quick reference (importance-filtered).
- Be buildable and debuggable by an intern using tools already on this machine.

---

## 3. System architecture

### 3.1 Pipeline overview

```
┌──────────────────┐     ┌───────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│                  │     │                   │     │                  │     │                  │
│  semantic YAML   │────▶│  Python renderer  │────▶│   HTML + CSS     │────▶│  Chromium PDF    │
│  (.semantic.yaml)│     │  (yaml→html)      │     │  (single file)   │     │  (--print-to-pdf)│
│                  │     │                   │     │                  │     │                  │
└──────────────────┘     └───────────────────┘     └──────────────────┘     └──────────────────┘
                                                                                      │
                                                                                      ▼
                                                                              ┌──────────────────┐
                                                                              │                  │
                                                                              │  reMarkable      │
                                                                              │  (cloud put)     │
                                                                              │                  │
                                                                              └──────────────────┘
```

Each stage is a separate concern:

1. **YAML parsing**: Python reads the YAML, validates it against the schema, and walks the section tree.
2. **HTML generation**: Python emits a single self-contained HTML file with embedded CSS and inlined content.
3. **PDF rendering**: Chromium opens the HTML file headless and prints to PDF with controlled page size and margins.
4. **Upload**: `remarquee cloud put` uploads the PDF to the reMarkable cloud.

### 3.2 Why HTML as the intermediate format

Direct PDF generation (via reportlab, fpdf, or similar) requires manual layout: you must compute x/y coordinates for every block, handle line wrapping, manage column splits for the Tufte margin, and implement page-break logic. For a document with 234 blocks across 76 nested sections, that is a lot of layout code.

HTML + CSS gives us:

- **CSS columns/flexbox** for the Tufte margin split (main column + margin column).
- **CSS `@media print`** for page-break control, orphans/widows, and print-specific font sizes.
- **CSS custom properties** for the color system, which can be tuned per block type and section role without touching the Python code.
- **Browser DevTools** for debugging — open the HTML file in Chrome, inspect elements, tweak CSS in real time, then port the fix back to the Python template.
- **`--print-to-pdf`** in Chromium produces clean, searchable, text-selectable PDFs with correct font embedding.

The alternative path — **pandoc + LaTeX** — is what `remarquee upload bundle` already uses. LaTeX can produce beautiful PDFs, but custom Tufte margin layouts with per-block-type color treatment require significant LaTeX macro programming. The HTML/CSS approach is faster to implement, easier to debug, and more familiar to most interns.

### 3.3 Why not pandoc for this pipeline

The existing `remarquee upload bundle` command converts Markdown → PDF via `pandoc + xelatex`. That pipeline is good for standard Markdown documents with simple heading hierarchy.

It is not good for semantic YAML because:

- Pandoc expects Markdown input, not YAML. We would need a YAML→Markdown converter first, and then we would lose the semantic signals in the Markdown output (Markdown cannot express "this paragraph is a principle with importance=critical").
- LaTeX Tufte-margin support exists (the `tufte-latex` package), but adding per-block-type color treatment requires custom LaTeX environments and macro programming. The iteration cycle (change LaTeX → recompile → check PDF) is slower than the HTML cycle (change CSS → reload browser).
- We need two PDFs (full + importance-filtered). With the HTML approach, we pass a `--importance-filter` flag to the Python renderer. With pandoc, we would need to generate two Markdown files, two LaTeX compilations, and two sets of custom environments.

Pandoc + LaTeX remains the right choice for standard Markdown documents. This pipeline is specifically for semantic YAML → high-density color-coded layout.

---

## 4. Component design

### 4.1 Python renderer (`semantic_render.py`)

The renderer is a single Python script. It has no external dependencies beyond the standard library (`yaml`, `jinja2` if desired, or simple string building). Its job is:

1. Parse the semantic YAML.
2. Walk the section tree recursively.
3. For each section, emit the section header with role-based CSS class.
4. For each block, emit the block content wrapped in a semantic HTML element with type-based, intent-based, and importance-based CSS classes.
5. Wrap everything in an HTML template with embedded CSS.
6. Optionally filter sections by importance threshold.
7. Write the HTML file.

#### 4.1.1 Input: semantic YAML structure

The YAML has this top-level shape (from the schema spec at `03-semantic-document-yaml-schema-spec.md`):

```yaml
schema_version: semantic-document/v1

document:
  id: string
  title: string
  subtitle: string
  version: string
  source_markdown: string
  audience: [string]
  intent: string
  density: low | medium | high
  reading_modes: [string]

semantic_vocabulary:
  section_roles: [string]
  block_types: [string]
  importance: [critical, high, medium, low]

sections:
  - id: string
    title: string
    level: integer           # 1-4
    role: string             # maps to section_roles
    importance: critical | high | medium | low
    summary: string
    blocks: [Block]
    sections: [Section]      # recursive nesting
```

And blocks:

```yaml
- id: b0042
  type: prose | principle | quote | definition | risk | decision | code | command | diagram | table | bullet_list | ordered_list | checklist
  intent: orient-reader | explain | define-core-term | remember | warn | compare-or-classify | show-api-or-pseudocode | show-js-example | show-command | explain-flow | enumerate | validate | implement
  content: string
  language: string           # for code blocks
  diagram_type: string       # for diagram blocks
  caption: string
  importance: critical | high | medium | low
  severity: high | medium | low   # for risk blocks
```

#### 4.1.2 Renderer pseudocode

```python
#!/usr/bin/env python3
"""semantic_render.py — Convert semantic-document/v1 YAML to styled HTML."""

import yaml
import sys
import argparse
from pathlib import Path

# ── Color palette for reMarkable Paper Pro color e-ink ──
# These are muted/saturated enough for color e-ink to distinguish clearly.
# Tested against the reMarkable Paper Pro's 7-color capability.
BLOCK_COLORS = {
    "prose":       {"border": "none",           "bg": "none",        "icon": ""},
    "principle":   {"border": "#2563EB",        "bg": "#EFF6FF",     "icon": "☑"},  # blue
    "quote":       {"border": "#6B7280",        "bg": "#F9FAFB",     "icon": "❝"},  # gray
    "definition":  {"border": "#7C3AED",        "bg": "#F5F3FF",     "icon": "📖"}, # purple
    "risk":        {"border": "#DC2626",        "bg": "#FEF2F2",     "icon": "⚠"},  # red
    "decision":    {"border": "#059669",        "bg": "#ECFDF5",     "icon": "✦"},  # green
    "code":        {"border": "none",           "bg": "#1E293B",     "icon": ""},   # dark bg
    "command":     {"border": "#475569",        "bg": "#1E293B",     "icon": "$"},  # dark bg
    "diagram":     {"border": "#6B7280",        "bg": "#F8FAFC",     "icon": "📐"}, # gray
    "table":       {"border": "#6B7280",        "bg": "none",        "icon": ""},   # standard
    "bullet_list": {"border": "none",           "bg": "none",        "icon": ""},
    "ordered_list":{"border": "none",           "bg": "none",        "icon": ""},
    "checklist":   {"border": "#D97706",        "bg": "#FFFBEB",     "icon": "☐"},  # amber
}

SEVERITY_COLORS = {
    "high":   "#DC2626",   # red
    "medium": "#F59E0B",   # amber
    "low":    "#FCD34D",   # yellow
}

ROLE_COLORS = {
    "summary":               "#2563EB",  # blue
    "problem-statement":     "#7C3AED",  # purple
    "concept-foundation":    "#2563EB",  # blue
    "architecture-map":      "#059669",  # green
    "case-study":            "#6B7280",  # gray
    "strengths-analysis":    "#059669",  # green
    "gap-analysis":          "#D97706",  # amber
    "risk-analysis":         "#DC2626",  # red
    "proposed-architecture": "#059669",  # green
    "authoring-guidelines":  "#D97706",  # amber
    "authoring-rule":        "#D97706",  # amber
    "api-reference":         "#2563EB",  # blue
    "implementation-plan":   "#0D9488",  # teal
    "implementation-phase":  "#0D9488",  # teal
    "migration-guide":       "#0D9488",  # teal
    "diagram-section":       "#6B7280",  # gray
    "test-strategy":         "#7C3AED",  # purple
    "open-questions":        "#DC2626",  # red
    "conclusion":            "#059669",  # green
    "exposition":            "none",     # no stripe
}

IMPORTANCE_WEIGHTS = {"critical": 3, "high": 2, "medium": 1, "low": 0}


def parse_yaml(path: str) -> dict:
    """Load and validate semantic YAML."""
    with open(path) as f:
        doc = yaml.safe_load(f)
    assert doc.get("schema_version") == "semantic-document/v1", \
        f"Unsupported schema version: {doc.get('schema_version')}"
    return doc


def importance_passes(imp: str, threshold: str) -> bool:
    """Return True if a section's importance meets or exceeds the threshold."""
    return IMPORTANCE_WEIGHTS.get(imp, 0) >= IMPORTANCE_WEIGHTS.get(threshold, 0)


def render_block(block: dict) -> str:
    """Render a single block to semantic HTML."""
    btype = block.get("type", "prose")
    intent = block.get("intent", "explain")
    content = block.get("content", "")
    importance = block.get("importance", "none")
    severity = block.get("severity", None)
    language = block.get("language", None)
    diagram_type = block.get("diagram_type", None)

    # Determine if this block goes to the margin
    margin_types = {"principle", "quote", "definition", "risk", "decision", "checklist"}
    goes_to_margin = btype in margin_types

    # CSS class construction
    classes = [f"block-{btype}", f"intent-{intent}"]
    if importance in ("critical", "high", "medium", "low"):
        classes.append(f"importance-{importance}")
    if severity:
        classes.append(f"severity-{severity}")
    if goes_to_margin:
        classes.append("margin-block")
    else:
        classes.append("main-block")

    color = BLOCK_COLORS.get(btype, BLOCK_COLORS["prose"])
    icon = color["icon"]

    if btype == "code":
        lang = language or ""
        return (
            f'<div class="{" ".join(classes)}">'
            f'<div class="code-lang">{lang}</div>'
            f'<pre><code>{escape_html(content)}</code></pre>'
            f'</div>'
        )

    if btype == "diagram":
        dtype = diagram_type or ""
        return (
            f'<div class="{" ".join(classes)}">'
            f'<div class="diagram-type">{dtype}</div>'
            f'<pre class="diagram-content">{escape_html(content)}</pre>'
            f'</div>'
        )

    if btype == "table":
        return (
            f'<div class="{" ".join(classes)}">'
            f'{markdown_table_to_html(content)}'
            f'</div>'
        )

    if btype in ("bullet_list", "ordered_list"):
        tag = "ul" if btype == "bullet_list" else "ol"
        items = parse_list_items(content)
        li_html = "".join(f"<li>{item}</li>" for item in items)
        return (
            f'<div class="{" ".join(classes)}">'
            f'<{tag}>{li_html}</{tag}>'
            f'</div>'
        )

    # Default: margin card or main paragraph
    if goes_to_margin:
        severity_badge = ""
        if severity:
            sev_color = SEVERITY_COLORS.get(severity, "#999")
            severity_badge = f'<span class="severity-badge" style="background:{sev_color}">{severity}</span>'
        return (
            f'<aside class="{" ".join(classes)}">'
            f'<span class="block-icon">{icon}</span>'
            f'{severity_badge}'
            f'<span class="block-type-label">{btype}</span>'
            f'<div class="margin-content">{format_prose(content)}</div>'
            f'</aside>'
        )

    # Prose in main column
    return (
        f'<div class="{" ".join(classes)}">'
        f'<p>{format_prose(content)}</p>'
        f'</div>'
    )


def render_section(section: dict, threshold: str | None = None) -> str:
    """Render a section and its children recursively."""
    importance = section.get("importance", "medium")
    if threshold and not importance_passes(importance, threshold):
        return ""  # filtered out

    sec_id = section.get("id", "")
    title = section.get("title", "")
    level = section.get("level", 2)
    role = section.get("role", "exposition")
    summary = section.get("summary", "")

    role_color = ROLE_COLORS.get(role, "none")
    classes = [f"section-role-{role}", f"importance-{importance}"]

    heading_tag = f"h{min(level, 4)}"

    # Render blocks
    blocks_html = ""
    # Separate margin and main blocks
    margin_blocks = []
    main_blocks = []
    for block in section.get("blocks", []):
        margin_types = {"principle", "quote", "definition", "risk", "decision", "checklist"}
        if block.get("type", "prose") in margin_types:
            margin_blocks.append(render_block(block))
        else:
            main_blocks.append(render_block(block))

    # Render child sections
    child_html = ""
    for child in section.get("sections", []):
        child_html += render_section(child, threshold)

    # Role stripe (thin colored line at section header)
    stripe = ""
    if role_color != "none":
        stripe = f'<div class="role-stripe" style="background:{role_color}"></div>'

    # Section layout: main column + margin column
    margin_html = "\n".join(margin_blocks) if margin_blocks else ""

    html = (
        f'<section class="{" ".join(classes)}" id="{sec_id}">'
        f'{stripe}'
        f'<{heading_tag} class="section-title">'
        f'<span class="role-badge" style="color:{role_color}">{role}</span> '
        f'{escape_html(title)}'
        f'</{heading_tag}>'
        f'<div class="section-body">'
        f'<div class="main-column">{"".join(main_blocks)}{child_html}</div>'
        f'<div class="margin-column">{margin_html}</div>'
        f'</div>'
        f'</section>'
    )
    return html


def render_document(yaml_path: str, output_path: str,
                    importance_threshold: str | None = None,
                    overview_page: bool = True) -> None:
    """Full pipeline: YAML → HTML file."""
    doc = parse_yaml(yaml_path)
    metadata = doc.get("document", {})
    title = metadata.get("title", "Untitled")
    subtitle = metadata.get("subtitle", "")
    audience = metadata.get("audience", [])
    density = metadata.get("density", "medium")
    reading_modes = metadata.get("reading_modes", [])

    # Build document-level importance threshold label
    threshold_label = ""
    if importance_threshold:
        threshold_label = f" (importance ≥ {importance_threshold})"

    # Overview page
    overview_html = ""
    if overview_page:
        overview_html = render_overview_page(doc, importance_threshold)

    # Body
    body_html = ""
    for section in doc.get("sections", []):
        body_html += render_section(section, importance_threshold)

    # Assemble full HTML
    html = HTML_TEMPLATE.format(
        title=escape_html(title + threshold_label),
        subtitle=escape_html(subtitle),
        audience=", ".join(audience),
        density=density,
        reading_modes=", ".join(reading_modes),
        overview=overview_html,
        body=body_html,
        css=CSS_STYLESHEET,
    )

    Path(output_path).write_text(html)


def render_overview_page(doc: dict, threshold: str | None = None) -> str:
    """Render a single-page section map with role-colored dots."""
    rows = []
    def walk(sections, indent=0):
        for s in sections:
            imp = s.get("importance", "medium")
            if threshold and not importance_passes(imp, threshold):
                continue
            role = s.get("role", "exposition")
            title = s.get("title", "")
            summary = s.get("summary", "")
            role_color = ROLE_COLORS.get(role, "#999")
            imp_stars = {"critical": "★★★", "high": "★★", "medium": "★", "low": ""}
            stars = imp_stars.get(imp, "")
            rows.append(
                f'<div class="overview-row" style="padding-left:{indent * 20}px">'
                f'<span class="role-dot" style="background:{role_color}"></span>'
                f'<span class="overview-title">{escape_html(title)}</span>'
                f'<span class="overview-importance">{stars}</span>'
                f'<span class="overview-summary">{escape_html(summary)}</span>'
                f'</div>'
            )
            walk(s.get("sections", []), indent + 1)
    walk(doc.get("sections", []))
    return (
        '<div class="overview-page">'
        '<h1 class="overview-header">Document Map</h1>'
        + "\n".join(rows) +
        '</div>'
    )


# ── HTML escaping and helpers ──

def escape_html(text: str) -> str:
    return (text
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;"))

def format_prose(content: str) -> str:
    """Convert simple Markdown inline formatting to HTML."""
    # This is intentionally minimal — we trust the YAML content.
    # Handle backtick code spans
    import re
    content = escape_html(content)
    content = re.sub(r'`([^`]+)`', r'<code>\1</code>', content)
    # Handle **bold** and *italic*
    content = re.sub(r'\*\*([^*]+)\*\*', r'<strong>\1</strong>', content)
    content = re.sub(r'\*([^*]+)\*', r'<em>\1</em>', content)
    # Handle > blockquotes (principle quotes)
    content = re.sub(r'^&gt; (.+)$', r'<blockquote>\1</blockquote>', content, flags=re.MULTILINE)
    return content

def parse_list_items(content: str) -> list[str]:
    """Parse Markdown list items into plain strings."""
    lines = content.strip().split("\n")
    items = []
    for line in lines:
        line = line.strip()
        if line.startswith("- "):
            items.append(format_prose(line[2:]))
        elif re.match(r'^\d+\.\s', line):
            items.append(format_prose(re.sub(r'^\d+\.\s', '', line)))
    return items

def markdown_table_to_html(content: str) -> str:
    """Convert a Markdown table to an HTML table."""
    lines = [l.strip() for l in content.strip().split("\n") if l.strip()]
    if not lines:
        return ""
    # Parse header, separator, rows
    header_cells = [c.strip() for c in lines[0].split("|") if c.strip()]
    rows = []
    for line in lines[2:]:  # skip separator line
        cells = [c.strip() for c in line.split("|") if c.strip()]
        rows.append(cells)

    html = "<table><thead><tr>"
    for cell in header_cells:
        html += f"<th>{format_prose(cell)}</th>"
    html += "</tr></thead><tbody>"
    for row in rows:
        html += "<tr>"
        for cell in row:
            html += f"<td>{format_prose(cell)}</td>"
        html += "</tr>"
    html += "</tbody></table>"
    return html
```

#### 4.1.3 CLI interface

```
python3 semantic_render.py \
  --input guide.semantic.yaml \
  --output guide-full.html \
  [--importance-threshold high] \
  [--no-overview]
```

The `--importance-threshold` flag controls filtering. Omit it for the full guide. Pass `critical` for only critical sections, `high` for critical + high, `medium` for everything.

### 4.2 CSS layout system

The CSS is the heart of the visual design. It implements three things:
1. **Tufte margin split**: main column (55%) + margin column (45%) using CSS Grid.
2. **Color banding**: block-type-specific left borders, background tints, and icons.
3. **Print control**: page breaks, font sizes, orphans/widows.

#### 4.2.1 Page geometry

The reMarkable Paper Pro has a 10.3" display. Its native document resolution is 1404×1872 pixels. We target a PDF page size of A4 (210×297mm) with small margins, which reMarkable reflows to fit its screen.

```css
@page {
  size: A4;
  margin: 12mm 10mm 12mm 10mm;
}

@media print {
  body {
    font-size: 8.5pt;
    line-height: 1.35;
    font-family: "DejaVu Sans", "Noto Sans", sans-serif;
  }
  code, pre {
    font-family: "DejaVu Sans Mono", "Noto Sans Mono", monospace;
    font-size: 7.5pt;
  }
}
```

With 8.5pt body text on A4 with 10mm margins, we get roughly:
- Usable width: ~190mm
- Main column (55%): ~105mm — about 72 characters per line at 8.5pt
- Margin column (45%): ~85mm — enough for principle cards, risk panels, definition boxes

#### 4.2.2 Tufte margin layout

```css
.section-body {
  display: grid;
  grid-template-columns: 55fr 45fr;
  gap: 12px;
  align-items: start;
}

.main-column {
  grid-column: 1;
  /* Flow: prose, code, diagrams, tables, lists */
}

.margin-column {
  grid-column: 2;
  /* Margin cards: principles, risks, definitions, decisions, checklists */
  position: sticky;
  top: 0;
}
```

The key insight: blocks of type `prose`, `code`, `diagram`, `table`, `bullet_list`, `ordered_list` go into `main-column`. Blocks of type `principle`, `quote`, `definition`, `risk`, `decision`, `checklist` go into `margin-column`. The Python renderer separates them and emits them into the correct column div.

Within a section, margin blocks appear next to their corresponding main-column content in document order. When there are more margin blocks than main-column blocks in a section, the margin column extends vertically beyond the main column — which is fine, because CSS Grid handles uneven column heights.

#### 4.2.3 Block-type color banding

```css
/* ── Margin card base ── */

.margin-block {
  border-left: 3px solid #999;
  border-radius: 2px;
  padding: 6px 8px;
  margin-bottom: 10px;
  background: #FAFAFA;
  font-size: 8pt;
  line-height: 1.3;
  break-inside: avoid;
}

.margin-block .block-icon {
  float: left;
  margin-right: 4px;
  font-size: 10pt;
}

.margin-block .block-type-label {
  display: inline-block;
  font-size: 6.5pt;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #666;
  margin-bottom: 3px;
}

/* ── Per-type styles ── */

.block-principle {
  border-left-color: #2563EB;  /* blue */
  background: #EFF6FF;
}

.block-risk {
  border-left-color: #DC2626;  /* red */
  background: #FEF2F2;
}

.block-risk.severity-high {
  border-left-color: #DC2626;
  border-left-width: 5px;
  background: #FEE2E2;
  font-weight: 500;
}

.block-risk.severity-medium {
  border-left-color: #F59E0B;  /* amber */
  background: #FFFBEB;
}

.block-definition {
  border-left-color: #7C3AED;  /* purple */
  background: #F5F3FF;
}

.block-decision {
  border-left-color: #059669;  /* green */
  background: #ECFDF5;
  border-left-width: 4px;      /* double-rule effect */
}

.block-checklist {
  border-left-color: #D97706;  /* amber */
  background: #FFFBEB;
}

.block-quote {
  border-left-color: #6B7280;  /* gray */
  background: #F9FAFB;
  font-style: italic;
}

/* ── Severity badge ── */

.severity-badge {
  display: inline-block;
  font-size: 6pt;
  color: white;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 4px;
  vertical-align: middle;
}

/* ── Main column code blocks ── */

.block-code {
  background: #1E293B;
  color: #E2E8F0;
  border-radius: 3px;
  padding: 8px 10px;
  margin: 6px 0;
  overflow-x: auto;
  break-inside: avoid;
}

.block-code .code-lang {
  font-size: 6pt;
  text-transform: uppercase;
  color: #94A3B8;
  margin-bottom: 4px;
}

.block-code pre {
  margin: 0;
  white-space: pre;
  font-size: 7.5pt;
  line-height: 1.3;
}

/* ── Diagrams ── */

.block-diagram {
  background: #F8FAFC;
  border: 1px solid #E2E8F0;
  border-radius: 3px;
  padding: 8px 10px;
  margin: 8px 0;
  break-inside: avoid;
}

.block-diagram .diagram-type {
  font-size: 6pt;
  text-transform: uppercase;
  color: #94A3B8;
  margin-bottom: 4px;
}

.block-diagram pre {
  font-size: 7pt;
  line-height: 1.25;
}

/* ── Tables ── */

.block-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 7.5pt;
  margin: 6px 0;
}

.block-table th {
  background: #F1F5F9;
  text-align: left;
  padding: 3px 6px;
  border-bottom: 1px solid #CBD5E1;
}

.block-table td {
  padding: 2px 6px;
  border-bottom: 1px solid #E2E8F0;
}

.block-table tr:nth-child(even) td {
  background: #F8FAFC;
}
```

#### 4.2.4 Section role stripes

Each section header carries a thin colored stripe on the left, derived from the section's `role` field:

```css
.role-stripe {
  width: 4px;
  height: 100%;
  position: absolute;
  left: 0;
  top: 0;
}

section {
  position: relative;
  padding-left: 8px;
  margin-bottom: 12px;
}

.section-title {
  font-size: 12pt;
  font-weight: 700;
  margin: 8px 0 4px 0;
  break-after: avoid;
}

.section-title .role-badge {
  font-size: 6.5pt;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-right: 6px;
}

/* Section importance indicators */
section.importance-critical .section-title {
  font-size: 13pt;
  border-bottom: 2px solid #DC2626;
  padding-bottom: 3px;
}

section.importance-high .section-title {
  font-size: 12pt;
}

section.importance-medium .section-title {
  font-size: 11pt;
  color: #475569;
}

/* H3/H4 are smaller */
section section .section-title { font-size: 10pt; }
section section section .section-title { font-size: 9pt; }
```

#### 4.2.5 Overview page

```css
.overview-page {
  break-after: page;
  padding: 0;
}

.overview-header {
  font-size: 14pt;
  border-bottom: 2px solid #333;
  padding-bottom: 4px;
  margin-bottom: 8px;
}

.overview-row {
  display: grid;
  grid-template-columns: 8px 1fr 40px 2fr;
  gap: 6px;
  align-items: start;
  padding: 2px 0;
  font-size: 7.5pt;
  line-height: 1.3;
}

.overview-row .role-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-top: 3px;
}

.overview-row .overview-title {
  font-weight: 600;
}

.overview-row .overview-importance {
  color: #DC2626;
  font-size: 7pt;
  text-align: center;
}

.overview-row .overview-summary {
  color: #64748B;
  font-size: 7pt;
}
```

#### 4.2.6 Print control

```css
@media print {
  /* Avoid orphaned headings */
  h1, h2, h3, h4 {
    break-after: avoid;
  }

  /* Keep margin cards with their section */
  .margin-block {
    break-inside: avoid;
  }

  /* Keep code blocks together */
  .block-code, .block-diagram {
    break-inside: avoid;
  }

  /* Section-level page breaks for major sections */
  section.importance-critical {
    break-before: auto;  /* don't force page break, but allow one */
  }

  /* No page break inside a section body (keep main + margin together) */
  .section-body {
    break-inside: auto;
  }
}
```

### 4.3 Chromium PDF rendering

Once the HTML file is generated, we use Chromium headless to produce the PDF:

```bash
chromium-browser \
  --headless \
  --disable-gpu \
  --no-sandbox \
  --print-to-pdf="guide-full.pdf" \
  --print-to-pdf-no-header \
  --no-pdf-header-footer \
  "file:///path/to/guide-full.html"
```

Key flags:
- `--headless`: no window.
- `--print-to-pdf`: output path.
- `--no-pdf-header-footer`: suppress Chromium's default page headers/footers (page numbers, dates).
- The page size and margins come from the `@page` CSS rule in the stylesheet.

If `chromium-browser` is not available, `google-chrome` or `chrome` work identically. The important thing is that the rendering engine is Chromium (not Firefox), because `--print-to-pdf` is a Chromium-specific flag.

#### 4.3.1 Checking available browsers

```bash
# Check what's available
which chromium-browser google-chrome chrome 2>/dev/null
```

If none are available, install `chromium-browser`:

```bash
sudo apt install chromium-browser
# or on some systems:
sudo snap install chromium
```

### 4.4 reMarkable upload

After the PDF is generated, upload to the reMarkable Paper Pro:

```bash
# Dry run first
remarquee cloud put guide-full.pdf /ai/2026/05/26/LAYOUT-001 --non-interactive

# Actual upload
remarquee cloud put guide-full.pdf /ai/2026/05/26/LAYOUT-001 --non-interactive

# Quick reference version
remarquee cloud put guide-quick.pdf /ai/2026/05/26/LAYOUT-001 --non-interactive
```

The `--non-interactive` flag prevents interactive prompts. The `/ai/YYYY/MM/DD/TICKET` path convention keeps uploads organized.

To verify the upload:

```bash
remarquee cloud ls /ai/2026/05/26/LAYOUT-001 --long --non-interactive
```

---

## 5. Dual-PDF output strategy

We produce two PDFs from every semantic YAML input:

### 5.1 Full guide

```bash
python3 semantic_render.py \
  --input guide.semantic.yaml \
  --output guide-full.html

chromium-browser --headless --print-to-pdf=guide-full.pdf guide-full.html
```

All sections, all blocks, all importance levels. The Tufte margin layout applies. The overview page is included at the front.

### 5.2 Quick reference

```bash
python3 semantic_render.py \
  --input guide.semantic.yaml \
  --output guide-quick.html \
  --importance-threshold high

chromium-browser --headless --print-to-pdf=guide-quick.pdf guide-quick.html
```

Only sections with `importance: critical` or `importance: high` are included. Within those sections, all blocks are rendered (including medium/low blocks that belong to high-importance sections). The overview page shows only the included sections.

The expected size reduction for the test case:

| PDF | Sections | Expected pages |
| --- | --- | --- |
| Full guide | 76 | ~40-50 |
| Quick reference (≥high) | ~55 (49 high + 6 critical) | ~25-30 |

### 5.3 Why section-level filtering, not block-level

Block-level importance in the test case YAML is almost always unset (all 234 blocks have `importance: none`). Section-level importance is well-assigned (6 critical, 49 high, 21 medium). Filtering at the section level gives us a meaningful reduction without losing block content that belongs to important sections.

If future YAML documents have well-assigned block-level importance, we can add a `--block-importance-threshold` flag that filters individual blocks within included sections.

---

## 6. Block-type to visual mapping — complete reference

This is the definitive mapping from semantic YAML block types to visual treatments. Every renderer must implement this consistently.

### 6.1 Main-column blocks

| Block type | Column | Left border | Background | Icon | Font treatment | Special |
| --- | --- | --- | --- | --- | --- | --- |
| `prose` | main | none | white | none | 8.5pt, normal | Default paragraph |
| `code` | main | none | #1E293B (dark) | none | 7.5pt monospace, light text | Language label above |
| `command` | main | #475569 | #1E293B (dark) | $ | 7.5pt monospace, light text | Copy affordance icon |
| `diagram` | main | #E2E8F0 | #F8FAFC (light) | 📐 | 7pt monospace | diagram_type label above |
| `table` | main | none | alternating rows | none | 7.5pt | Header row tinted |
| `bullet_list` | main | none | white | • | 8.5pt | Standard list |
| `ordered_list` | main | none | white | 1. 2. 3. | 8.5pt | Numbered procedure |

### 6.2 Margin-column blocks

| Block type | Column | Left border color | Left border width | Background | Icon | Special |
| --- | --- | --- | --- | --- | --- | --- |
| `principle` | margin | #2563EB (blue) | 3px | #EFF6FF | ☑ | Critical importance gets full-width pull-quote instead |
| `risk` | margin | severity-dependent | 3-5px | severity-dependent | ⚠ | severity-high gets 5px border + darker bg |
| `risk` (high) | margin | #DC2626 (red) | 5px | #FEE2E2 | ⛔ | Bold, severity badge |
| `risk` (medium) | margin | #F59E0B (amber) | 3px | #FFFBEB | ⚠ | Normal weight |
| `definition` | margin | #7C3AED (purple) | 3px | #F5F3FF | 📖 | Term in bold |
| `decision` | margin | #059669 (green) | 4px | #ECFDF5 | ✦ | Double-rule effect from 4px width |
| `checklist` | margin | #D97706 (amber) | 3px | #FFFBEB | ☐ | Each item as checkbox row |
| `quote` | margin | #6B7280 (gray) | 3px | #F9FAFB | ❝ | Italic |

### 6.3 Critical-principle special treatment

When a block has `type: principle` AND `importance: critical`, it is promoted from the margin to a **full-width pull-quote box** spanning both columns:

```css
.block-principle.importance-critical {
  grid-column: 1 / -1;         /* span both columns */
  border-left: 5px solid #2563EB;
  background: #DBEAFE;
  padding: 10px 14px;
  font-size: 9pt;
  font-weight: 600;
  margin: 10px 0;
  border-radius: 3px;
}
```

This is the only block type that breaks the two-column layout. It is reserved for the few principles that the entire document is built around.

---

## 7. Section role to color mapping — complete reference

| Section role | Stripe color | Color name | Badge label |
| --- | --- | --- | --- |
| `summary` | #2563EB | Blue | SUMMARY |
| `problem-statement` | #7C3AED | Purple | PROBLEM |
| `concept-foundation` | #2563EB | Blue | CONCEPT |
| `architecture-map` | #059669 | Green | ARCHITECTURE |
| `case-study` | #6B7280 | Gray | CASE STUDY |
| `strengths-analysis` | #059669 | Green | STRENGTHS |
| `gap-analysis` | #D97706 | Amber | GAP |
| `risk-analysis` | #DC2626 | Red | RISK |
| `proposed-architecture` | #059669 | Green | PROPOSED |
| `authoring-guidelines` | #D97706 | Amber | AUTHORING |
| `authoring-rule` | #D97706 | Amber | RULE |
| `api-reference` | #2563EB | Blue | API |
| `implementation-plan` | #0D9488 | Teal | PLAN |
| `implementation-phase` | #0D9488 | Teal | PHASE |
| `migration-guide` | #0D9488 | Teal | MIGRATION |
| `diagram-section` | #6B7280 | Gray | DIAGRAM |
| `test-strategy` | #7C3AED | Purple | TEST |
| `open-questions` | #DC2626 | Red | OPEN |
| `conclusion` | #059669 | Green | CONCLUSION |
| `exposition` | none | — | (no stripe or badge) |

---

## 8. End-to-end workflow

### 8.1 Build script

A single shell script wraps the whole pipeline:

```bash
#!/usr/bin/env bash
# render_to_remarkable.sh — YAML → HTML → PDF → reMarkable

set -euo pipefail

INPUT="$1"
BASENAME="$(basename "$INPUT" .semantic.yaml)"
SCRIPT_DIR="$(dirname "$0")"
OUTPUT_DIR="${SCRIPT_DIR}/output"
mkdir -p "$OUTPUT_DIR"

# Detect browser
BROWSER=""
for cmd in chromium-browser google-chrome chrome; do
  if command -v "$cmd" &>/dev/null; then
    BROWSER="$cmd"
    break
  fi
done
if [ -z "$BROWSER" ]; then
  echo "ERROR: No Chromium-based browser found. Install chromium-browser."
  exit 1
fi

echo "=== Rendering full guide ==="
python3 "${SCRIPT_DIR}/semantic_render.py" \
  --input "$INPUT" \
  --output "${OUTPUT_DIR}/${BASENAME}-full.html"

"$BROWSER" --headless --disable-gpu --no-sandbox \
  --print-to-pdf="${OUTPUT_DIR}/${BASENAME}-full.pdf" \
  --no-pdf-header-footer \
  "file://${OUTPUT_DIR}/${BASENAME}-full.html"

echo "=== Rendering quick reference (importance >= high) ==="
python3 "${SCRIPT_DIR}/semantic_render.py" \
  --input "$INPUT" \
  --output "${OUTPUT_DIR}/${BASENAME}-quick.html" \
  --importance-threshold high

"$BROWSER" --headless --disable-gpu --no-sandbox \
  --print-to-pdf="${OUTPUT_DIR}/${BASENAME}-quick.pdf" \
  --no-pdf-header-footer \
  "file://${OUTPUT_DIR}/${BASENAME}-quick.html"

echo "=== PDFs generated ==="
ls -lh "${OUTPUT_DIR}/${BASENAME}"-*.pdf

echo "=== Uploading to reMarkable ==="
remarquee cloud put "${OUTPUT_DIR}/${BASENAME}-full.pdf" \
  "/ai/2026/05/26/LAYOUT-001" --non-interactive

remarquee cloud put "${OUTPUT_DIR}/${BASENAME}-quick.pdf" \
  "/ai/2026/05/26/LAYOUT-001" --non-interactive

echo "=== Verifying ==="
remarquee cloud ls /ai/2026/05/26/LAYOUT-001 --long --non-interactive

echo "=== Done ==="
```

### 8.2 Typical invocation

```bash
cd /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/26/LAYOUT-001--typographic-layout-of-semantic-yaml-guides-intent-driven-information-density/scripts

./render_to_remarkable.sh \
  /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml
```

### 8.3 Debugging workflow

During development, skip the PDF and reMarkable steps. Open the HTML directly in a browser:

```bash
python3 semantic_render.py \
  --input guide.semantic.yaml \
  --output /tmp/guide-debug.html

# Open in browser for live debugging
xdg-open /tmp/guide-debug.html
```

Use Chrome DevTools to:
- Inspect the CSS Grid layout (`display: grid` on `.section-body`).
- Check that margin blocks are in the margin column.
- Verify color values render correctly.
- Test print preview (Ctrl+P → see page breaks and font sizes).

When the layout looks right, switch to headless PDF generation.

---

## 9. Implementation plan

### Phase 1: Minimal renderer — single HTML file, no PDF

Files:
- `scripts/semantic_render.py` — the Python renderer
- `scripts/output/` — generated HTML files

Steps:
1. Implement `parse_yaml()`, `render_block()`, `render_section()`, `render_document()`.
2. Implement the CSS stylesheet as a Python string constant in `semantic_render.py`.
3. Implement the HTML template.
4. Handle all 7 block types found in the test case: `prose`, `code`, `bullet_list`, `ordered_list`, `principle`, `diagram`, `table`.
5. Implement `format_prose()` for inline Markdown (backtick code, bold, italic, blockquote).
6. Generate HTML and open in browser. Verify:
   - Main column contains prose, code, diagrams, tables, lists.
   - Margin column contains principles.
   - Section role stripes and badges appear.
   - Colors are correct.
7. Do NOT implement importance filtering or overview page yet.

### Phase 2: PDF output

Steps:
1. Add Chromium `--print-to-pdf` invocation.
2. Verify PDF page size is A4.
3. Verify font sizes in the PDF match the `@media print` rules.
4. Check page breaks — no orphaned headings, no split code blocks.
5. Tune margins if needed for reMarkable reflow.

### Phase 3: Importance filtering and dual-PDF output

Steps:
1. Implement `--importance-threshold` CLI flag.
2. Implement `importance_passes()` check in `render_section()`.
3. Generate two HTML files (full + quick reference).
4. Generate two PDFs.
5. Verify quick reference omits medium-importance sections.

### Phase 4: Overview page

Steps:
1. Implement `render_overview_page()`.
2. Add role-colored dots, importance stars, and summary text.
3. Verify it renders as the first page with a page break after it.
4. Verify filtered overview only shows included sections.

### Phase 5: reMarkable upload and validation

Steps:
1. Add `remarquee cloud put` calls to the build script.
2. Upload both PDFs.
3. Open on reMarkable Paper Pro and verify:
   - Colors are distinguishable on color e-ink.
   - Text is readable at 8.5pt.
   - Margin cards don't overflow the column.
   - Page breaks are clean.
   - Code blocks don't wrap awkwardly.
4. Adjust font sizes or colors if needed based on device testing.

### Phase 6: Polish and edge cases

Steps:
1. Handle `type: command` blocks (shell commands with copy affordance).
2. Handle `type: checklist` blocks (checkbox items).
3. Handle `type: quote` blocks.
4. Handle long code blocks that span multiple pages (use `break-inside: avoid`).
5. Handle empty sections (no blocks, only child sections).
6. Handle deeply nested sections (4 levels deep).
7. Add metadata footer to the PDF (document title, version, source ticket, generated date).
8. Test with other semantic YAML files if available.

---

## 10. File structure

```
scripts/
├── semantic_render.py          # Python renderer (YAML → HTML)
├── render_to_remarkable.sh      # Full pipeline script (YAML → HTML → PDF → upload)
└── output/                     # Generated files (gitignored)
    ├── guide-full.html
    ├── guide-full.pdf
    ├── guide-quick.html
    └── guide-quick.pdf
```

All generated files in `output/` should be added to `.gitignore`. The Python script and shell script are the only files that need version control.

---

## 11. ReMarkable Paper Pro color e-ink considerations

### 11.1 What color e-ink can do

The reMarkable Paper Pro uses a Kaleido-style color e-ink panel. It can display:

- **Clear, saturated reds, blues, and greens** — these are the strongest colors.
- **Muted ambers, purples, and teals** — visible but less vivid.
- **Grayscale** — excellent, as expected from e-ink.
- **Approximately 7 distinguishable colors** at useful contrast levels.

### 11.2 What it cannot do

- **Subtle pastels** may wash out to near-white. Our lighter background tints (`#EFF6FF`, `#F5F3FF`, `#ECFDF5`) may appear as slightly tinted off-white rather than clearly colored. This is acceptable because the *border colors* carry the primary signal, not the backgrounds.
- **Fine color gradients** render as flat bands. Don't rely on gradients.
- **Dark backgrounds on code blocks** render as dark gray, which is fine — the contrast is high.
- **Small colored text** (below ~8pt) may lose color saturation. Our colored elements are borders (3-5px wide, very visible) and badges (uppercase labels with colored text). The borders are the primary signal.

### 11.3 Design choices driven by hardware

- **Left borders, not backgrounds, are the primary signal.** A 3px colored left border is clearly visible on color e-ink even at a distance. Background tints are secondary — they help on-screen and may wash out on e-ink, but the border still identifies the type.
- **Icons supplement color.** ☑ for principle, ⚠ for risk, 📖 for definition, ✦ for decision. These work in grayscale too, making the document usable even if color is turned off.
- **No reliance on color alone.** Every color-coded element also has a text label (the `block-type-label` like "PRINCIPLE", "RISK", "DECISION") and an icon.
- **High-contrast code blocks.** Dark background with light text is the clearest way to render code on e-ink. The contrast is ~85% which is excellent for readability.

---

## 12. Intent signal usage

The `intent` field on blocks is the most underused semantic signal. In the test case, intents are distributed as:

| Intent | Count | Percentage |
| --- | --- | --- |
| `explain` | 147 | 63% |
| `show-api-or-pseudocode` | 33 | 14% |
| `enumerate` | 32 | 14% |
| `remember` | 6 | 3% |
| `explain-flow` | 6 | 3% |
| `show-example` | 6 | 3% |
| `show-js-example` | 2 | 1% |
| `compare-or-classify` | 2 | 1% |

The intent is currently **not used for visual differentiation** in the first version of the renderer. It is emitted as a CSS class (`intent-explain`, `intent-show-api-or-pseudocode`, etc.) so it can be styled later.

Potential future uses of intent:

- `remember` blocks could get a subtle "bookmark" icon or a different border style.
- `show-api-or-pseudocode` blocks could get a language-specific syntax hint badge.
- `enumerate` blocks in the margin could render as numbered callouts.
- `compare-or-classify` blocks could render as a comparison panel with two sub-panels.

For now, intent is a CSS hook waiting for design decisions. The block type is the primary visual signal.

---

## 13. Testing strategy

### 13.1 Visual regression

There is no automated visual regression testing in the first version. The workflow is:

1. Generate HTML.
2. Open in browser. Check the layout.
3. Generate PDF. Check page breaks.
4. Upload to reMarkable. Check on device.

This is acceptable because the renderer is a template — the same code produces the same output for the same input. Regressions are caught by eye during development.

### 13.2 Structural validation

Before rendering, validate the YAML:

```bash
python3 -c "
import yaml
from pathlib import Path
doc = yaml.safe_load(Path('guide.semantic.yaml').read_text())
assert doc['schema_version'] == 'semantic-document/v1'
assert doc['document']['id']
assert doc['document']['title']
# ... (see schema spec validation rules)
print('YAML valid')
"
```

### 13.3 Content completeness check

After rendering, verify that all blocks from the YAML appear in the HTML:

```python
def count_yaml_blocks(doc):
    count = 0
    def walk(sections):
        nonlocal count
        for s in sections:
            count += len(s.get('blocks', []))
            walk(s.get('sections', []))
    walk(doc['sections'])
    return count

def count_html_blocks(html):
    return html.count('class="block-')

# These should match (accounting for importance filtering)
```

### 13.4 PDF sanity checks

```bash
# Check PDF exists and has content
ls -lh guide-full.pdf guide-quick.pdf

# Check page count
python3 -c "
import subprocess
result = subprocess.run(['pdfinfo', 'guide-full.pdf'], capture_output=True, text=True)
print(result.stdout)
"
```

---

## 14. Risks and tradeoffs

### 14.1 Color e-ink color accuracy

The colors in our palette are chosen for LCD screens. Color e-ink renders them differently — typically less saturated. A red that reads as `#DC2626` on screen may appear more like `#B91C1C` on e-ink.

**Mitigation**: Test on the device. If border colors are not distinguishable, increase saturation or switch to a more limited palette (only red, blue, green, amber — the four strongest colors on Kaleido panels).

### 14.2 Small text on e-ink

8.5pt body text is small. On a high-DPI LCD this is fine. On e-ink, the effective resolution is lower (~226 DPI for the Paper Pro). Small text may appear slightly fuzzy.

**Mitigation**: The user (Manuel) has confirmed small text is acceptable. If it proves too small on device, the `@media print` font-size can be bumped to 9pt with minimal layout impact.

### 14.3 Chromium font embedding

Chromium's `--print-to-pdf` embeds fonts by default. The PDF will use whatever fonts are available on the system where Chromium runs. If DejaVu Sans is not installed, Chromium falls back to its default font, which may have different metrics.

**Mitigation**: Ensure DejaVu Sans is installed (`sudo apt install fonts-dejavu`). The PDF should be generated on the same machine where the fonts are installed.

### 14.4 Long code blocks and page breaks

A code block that spans more than one printed page will not break inside if `break-inside: avoid` is set. CSS `break-inside: avoid` is a *hint*, not a guarantee. Chromium may still break a long code block across pages.

**Mitigation**: Accept that very long code blocks may break across pages. The alternative (splitting code blocks manually in the renderer) is complex and not worth the effort for the first version.

### 14.5 Sticky margin positioning

CSS `position: sticky` for margin columns works in screen layout but is ignored in `@media print`. In the printed PDF, the margin column simply flows below the main column if it is longer.

**Mitigation**: This is actually the desired behavior for print. In a paged document, each section's margin blocks should appear alongside that section's main content. CSS Grid handles this: the margin column is within the `.section-body` grid, so it aligns per-section. Sticky positioning is only needed for on-screen scrolling, not for print.

---

## 15. Future work

1. **Intent-based styling**: Use the `intent` field to add visual differentiation beyond block type (e.g., `remember` blocks get a bookmark icon; `show-api-or-pseudocode` blocks get a language badge).
2. **Interactive HTML version**: The HTML file is already a working web page. Add a sidebar navigation (Approach 3 from the layout catalog) and interactive importance filtering (Approach 7) for browser reading.
3. **Multiple layout themes**: The CSS is fully parameterized via classes. A "dark mode" theme or a "print-optimized" theme (no colors, pure grayscale) is just a different CSS file.
4. **YAML validation schema**: Add a JSON Schema or pydantic model for `semantic-document/v1` so the renderer can validate input before processing.
5. **Performance**: For very large documents (500+ blocks), the single-file HTML may become slow in the browser. Consider splitting into per-section pages with client-side navigation.
6. **Cross-reference links**: The section `id` fields can be used to generate internal hyperlinks (`<a href="#section-id">`) within the HTML. This is not implemented in the first version.
7. **Reading-mode lanes (Approach 5)**: Generate separate HTML/PDF files for each declared reading mode, with block-level filtering based on intent relevance.

---

## 16. API reference — key functions

### `parse_yaml(path: str) -> dict`

**Input**: Path to a `.semantic.yaml` file.
**Output**: Parsed YAML as a Python dict.
**Raises**: `AssertionError` if `schema_version` is not `semantic-document/v1`.
**Side effects**: None.

### `render_block(block: dict) -> str`

**Input**: A single block dict from the YAML.
**Output**: HTML string for the block, including wrapper div/aside with semantic CSS classes.
**Key logic**:
- Determines column placement (`margin-block` vs `main-block`) based on `block.type`.
- Applies `severity-*` CSS class for risk blocks.
- Applies `importance-*` CSS class if set.
- Escapes HTML content and handles inline Markdown formatting.

### `render_section(section: dict, threshold: str | None = None) -> str`

**Input**: A section dict (recursive structure).
**Output**: HTML string for the section, its blocks, and all child sections.
**Key logic**:
- Checks importance threshold; returns `""` if the section doesn't pass.
- Separates margin blocks from main blocks.
- Emits section header with role stripe and role badge.
- Recursively renders child sections.

### `render_document(yaml_path: str, output_path: str, importance_threshold: str | None, overview_page: bool) -> None`

**Input**: YAML path, output HTML path, optional importance threshold, overview page toggle.
**Output**: Writes a complete HTML file to `output_path`.
**Side effects**: File creation.

### `render_overview_page(doc: dict, threshold: str | None) -> str`

**Input**: Full parsed YAML doc, optional importance threshold.
**Output**: HTML string for a single-page section map with role-colored dots, importance stars, and one-line summaries.

---

## 17. References

- Semantic YAML schema spec: `ttmp/2026/05/25/XGOJA-014/.../reference/03-semantic-document-yaml-schema-spec.md`
- Test case semantic YAML: `ttmp/2026/05/25/XGOJA-014/.../design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml`
- Layout approaches catalog: `ttmp/2026/05/26/LAYOUT-001/.../design-doc/01-semantic-yaml-layout-approaches.md`
- reMarkable Paper Pro display: 10.3" color e-ink, 1404×1872 pixels, Kaleido Plus panel
- Chromium `--print-to-pdf` documentation: https://chromium.googlesource.com/chromium/src/+/main/docs/headless.md
- `remarquee cloud put` command: `remarquee help put`
- Edward Tufte, *Beautiful Evidence* — margin notes, sparklines, the "cognitive style" of information display
- CSS Grid specification: https://www.w3.org/TR/css-grid-1/
- CSS Paged Media: https://www.w3.org/TR/css-page-3/
