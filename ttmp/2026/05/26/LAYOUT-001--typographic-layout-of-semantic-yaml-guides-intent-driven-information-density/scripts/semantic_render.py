#!/usr/bin/env python3
"""semantic_render.py — Convert semantic-document/v1 YAML to styled HTML for reMarkable Paper Pro.

Pipeline: YAML → this script → HTML+CSS → Chromium --print-to-pdf → reMarkable

Usage:
    python3 semantic_render.py \\
      --input guide.semantic.yaml \\
      --output guide-full.html

    python3 semantic_render.py \\
      --input guide.semantic.yaml \\
      --output guide-quick.html \\
      --importance-threshold high
"""

import yaml
import sys
import argparse
import re
from pathlib import Path


# ──────────────────────────────────────────────────────────────────────────────
# Color palette for reMarkable Paper Pro color e-ink
# ──────────────────────────────────────────────────────────────────────────────
# These are muted/saturated enough for color e-ink to distinguish clearly.
# Tested against the reMarkable Paper Pro's ~7-color capability.

# Colors darkened for e-ink contrast — pale tints wash out on Kaleido panels.
# Backgrounds are 2-3 steps darker than the original pastel choices.
BLOCK_COLORS = {
    "prose":       {"border": "none",      "bg": "none",     "icon": ""},
    "principle":   {"border": "#1D4ED8",   "bg": "#DBEAFE",  "icon": "☑"},   # blue (darker)
    "quote":       {"border": "#4B5563",   "bg": "#F3F4F6",  "icon": "❝"},   # gray (darker)
    "definition":  {"border": "#6D28D9",   "bg": "#EDE9FE",  "icon": "📖"},  # purple (darker)
    "risk":        {"border": "#B91C1C",   "bg": "#FEE2E2",  "icon": "⚠"},   # red (darker)
    "decision":    {"border": "#047857",   "bg": "#D1FAE5",  "icon": "✦"},   # green (darker)
    "code":        {"border": "none",      "bg": "#F1F5F9",  "icon": ""},    # LIGHT bg for e-ink readability
    "command":     {"border": "#475569",   "bg": "#F1F5F9",  "icon": "$"},   # LIGHT bg
    "diagram":     {"border": "#4B5563",   "bg": "#F1F5F9",  "icon": "📐"},  # gray (darker)
    "table":       {"border": "#4B5563",   "bg": "none",     "icon": ""},    # standard
    "bullet_list": {"border": "none",      "bg": "none",     "icon": ""},
    "ordered_list": {"border": "none",     "bg": "none",     "icon": ""},
    "checklist":   {"border": "#B45309",   "bg": "#FEF3C7",  "icon": "☐"},   # amber (darker)
}

# Severity badge colors — darkened for e-ink contrast
SEVERITY_COLORS = {
    "high":   "#B91C1C",   # dark red
    "medium": "#D97706",   # dark amber
    "low":    "#EAB308",   # dark yellow
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
    "diagram-section":      "#6B7280",  # gray
    "test-strategy":         "#7C3AED",  # purple
    "open-questions":        "#DC2626",  # red
    "conclusion":            "#059669",  # green
    "exposition":            "none",     # no stripe
}

IMPORTANCE_WEIGHTS = {"critical": 3, "high": 2, "medium": 1, "low": 0}

# Block types that go into the margin column
# NOTE: "exposition" and "prose" are NOT margin types — they go in the main column.
MARGIN_TYPES = {"principle", "quote", "definition", "risk", "decision", "checklist"}


# ──────────────────────────────────────────────────────────────────────────────
# HTML helpers
# ──────────────────────────────────────────────────────────────────────────────

def escape_html(text: str) -> str:
    return (text
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;"))


def format_prose(content: str) -> str:
    """Convert simple Markdown inline formatting to HTML."""
    content = escape_html(content)
    # backtick code spans
    content = re.sub(r'`([^`]+)`', r'<code>\1</code>', content)
    # **bold**
    content = re.sub(r'\*\*([^*]+)\*\*', r'<strong>\1</strong>', content)
    # *italic*
    content = re.sub(r'\*([^*]+)\*', r'<em>\1</em>', content)
    # > blockquotes
    content = re.sub(r'^&gt; (.+)$', r'<blockquote>\1</blockquote>', content, flags=re.MULTILINE)
    return content


def parse_list_items(content: str) -> list:
    """Parse Markdown list items into formatted HTML strings."""
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


# ──────────────────────────────────────────────────────────────────────────────
# Block rendering
# ──────────────────────────────────────────────────────────────────────────────

def render_block(block: dict) -> str:
    """Render a single block to semantic HTML."""
    btype = block.get("type", "prose")
    intent = block.get("intent", "explain")
    content = block.get("content", "")
    importance = block.get("importance", "none")
    severity = block.get("severity", None)
    language = block.get("language", None)
    diagram_type = block.get("diagram_type", None)

    goes_to_margin = btype in MARGIN_TYPES

    # CSS classes
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

    # ── Code blocks ──
    if btype == "code":
        lang = language or ""
        return (
            f'<div class="{" ".join(classes)}">'
            f'<div class="code-lang">{escape_html(lang)}</div>'
            f'<pre><code>{escape_html(content)}</code></pre>'
            f'</div>'
        )

    # ── Diagram blocks ──
    if btype == "diagram":
        dtype = diagram_type or ""
        return (
            f'<div class="{" ".join(classes)}">'
            f'<div class="diagram-type">{escape_html(dtype)}</div>'
            f'<pre class="diagram-content">{escape_html(content)}</pre>'
            f'</div>'
        )

    # ── Table blocks ──
    if btype == "table":
        return (
            f'<div class="{" ".join(classes)}">'
            f'{markdown_table_to_html(content)}'
            f'</div>'
        )

    # ── List blocks ──
    if btype in ("bullet_list", "ordered_list"):
        tag = "ul" if btype == "bullet_list" else "ol"
        items = parse_list_items(content)
        li_html = "".join(f"<li>{item}</li>" for item in items)
        return (
            f'<div class="{" ".join(classes)}">'
            f'<{tag}>{li_html}</{tag}>'
            f'</div>'
        )

    # ── Command blocks ──
    if btype == "command":
        return (
            f'<div class="{" ".join(classes)}">'
            f'<pre><code>{escape_html(content)}</code></pre>'
            f'</div>'
        )

    # ── Margin cards (principle, quote, definition, risk, decision, checklist) ──
    if goes_to_margin:
        # Critical principle → full-width pull quote
        if btype == "principle" and importance == "critical":
            return (
                f'<div class="{" ".join(classes)} critical-pull-quote">'
                f'<span class="block-icon">{icon}</span>'
                f'<span class="block-type-label">{btype}</span>'
                f'<div class="margin-content">{format_prose(content)}</div>'
                f'</div>'
            )

        severity_badge = ""
        if severity:
            sev_color = SEVERITY_COLORS.get(severity, "#999")
            severity_badge = (
                f'<span class="severity-badge" style="background:{sev_color}">'
                f'{severity}</span>'
            )
        return (
            f'<aside class="{" ".join(classes)}">'
            f'<span class="block-icon">{icon}</span>'
            f'{severity_badge}'
            f'<span class="block-type-label">{btype}</span>'
            f'<div class="margin-content">{format_prose(content)}</div>'
            f'</aside>'
        )

    # ── Default: prose in main column ──
    return (
        f'<div class="{" ".join(classes)}">'
        f'<p>{format_prose(content)}</p>'
        f'</div>'
    )


# ──────────────────────────────────────────────────────────────────────────────
# Section rendering
# ──────────────────────────────────────────────────────────────────────────────

def importance_passes(imp: str, threshold: str) -> bool:
    """Return True if a section's importance meets or exceeds the threshold."""
    return IMPORTANCE_WEIGHTS.get(imp, 0) >= IMPORTANCE_WEIGHTS.get(threshold, 0)


def merge_margin_blocks(blocks: list[str]) -> list[str]:
    """Merge sequential margin blocks of the same type into one card.

    When two or more adjacent margin blocks share the same block type
    (e.g. three principle blocks in a row), wrap their content in a single
    <aside> instead of emitting three separate cards.
    """
    if not blocks:
        return []

    # Parse each rendered block to extract (type, classes, inner_html)
    parsed = []
    for html in blocks:
        # Match <aside class="..." ...>inner</aside> or <div class="..." ...>inner</div>
        m = re.match(r'<(aside|div) class="([^"]+)"[^>]*>(.*)</\1>$', html.strip(), re.DOTALL)
        if not m:
            parsed.append((None, None, html))
            continue
        tag, classes_str, inner = m.group(1), m.group(2), m.group(3)
        # Extract block type from classes
        btype = None
        for cls in classes_str.split():
            if cls.startswith('block-') and cls != 'margin-block' and cls != 'main-block':
                btype = cls[6:]  # strip 'block-' prefix
                break
        parsed.append((btype, classes_str, inner))

    # Group sequential blocks of the same type
    merged = []
    i = 0
    while i < len(parsed):
        btype_i, classes_i, inner_i = parsed[i]
        if btype_i is None:
            # Unmergeable — keep as-is
            merged.append(blocks[i])
            i += 1
            continue

        # Collect consecutive blocks of same type
        group_inners = [inner_i]
        j = i + 1
        while j < len(parsed) and parsed[j][0] == btype_i:
            group_inners.append(parsed[j][2])
            j += 1

        if len(group_inners) == 1:
            # Single block — keep as-is
            merged.append(blocks[i])
        else:
            # Merged card: one icon/label, multiple content blocks
            color = BLOCK_COLORS.get(btype_i, BLOCK_COLORS.get("prose"))
            icon = color["icon"]
            # Build merged content: each inner block's .margin-content
            # Extract just the margin-content divs from each inner
            content_parts = []
            for inner in group_inners:
                # Pull out the margin-content div
                cm = re.search(r'<div class="margin-content">(.*?)</div>', inner, re.DOTALL)
                if cm:
                    content_parts.append(f'<div class="margin-entry">{cm.group(1)}</div>')
                else:
                    content_parts.append(f'<div class="margin-entry">{inner}</div>')

            merged_html = (
                f'<aside class="block-{btype_i} margin-block merged-card">'
                f'<span class="block-icon">{icon}</span>'
                f'<span class="block-type-label">{btype_i}</span>'
                f'<div class="margin-merged-content">{"".join(content_parts)}</div>'
                f'</aside>'
            )
            merged.append(merged_html)

        i = j

    return merged


def render_section_rows(section: dict, threshold: str | None = None) -> list[str]:
    """Render a section tree as row-aligned two-column rows.

    Important layout invariant:
    - no nested grids, so nested sections never get narrower;
    - no independent global right column, so tags/cards do not mass at top;
    - each section owns one row: left = heading/content, right = tags/cards.
    """
    importance = section.get("importance", "medium")
    if threshold and not importance_passes(importance, threshold):
        return []

    sec_id = section.get("id", "")
    title = section.get("title", "")
    level = section.get("level", 2)
    role = section.get("role", "exposition")

    role_color = ROLE_COLORS.get(role, "none")
    classes = [
        "section-row",
        f"level-{level}",
        f"section-role-{role}",
        f"importance-{importance}",
    ]
    heading_tag = f"h{min(level, 4)}"

    main_parts = []
    margin_parts = []
    for block in section.get("blocks", []):
        rendered = render_block(block)
        if block.get("type", "prose") in MARGIN_TYPES:
            margin_parts.append(rendered)
        else:
            main_parts.append(rendered)

    if role_color != "none":
        margin_parts.insert(0, (
            f'<aside class="role-tag" style="border-left-color:{role_color}">'
            f'<span class="role-label" style="color:{role_color}">{escape_html(role)}</span>'
            f'</aside>'
        ))

    role_tag = margin_parts[0] if margin_parts and 'role-tag' in margin_parts[0] else ""
    semantic_margin = margin_parts[1:] if role_tag else margin_parts
    margin_html = ""
    if role_tag:
        margin_html += role_tag
    if semantic_margin:
        margin_html += ("\n" if margin_html else "") + "\n".join(merge_margin_blocks(semantic_margin))

    title_style = f' style="color:{role_color}"' if role_color != "none" else ""
    row_html = (
        f'<section class="{" ".join(classes)}" id="{sec_id}">'
        f'<div class="section-main">'
        f'<{heading_tag} class="section-title"{title_style}>{escape_html(title)}</{heading_tag}>'
        f'{"".join(main_parts)}'
        f'</div>'
        f'<div class="section-side">{margin_html}</div>'
        f'</section>'
    )

    rows = [row_html]
    for child in section.get("sections", []):
        rows.extend(render_section_rows(child, threshold))
    return rows


# ──────────────────────────────────────────────────────────────────────────────
# Overview page
# ──────────────────────────────────────────────────────────────────────────────

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
            rows.append(
                f'<div class="overview-row" style="padding-left:{indent * 20}px">'
                f'<span class="role-dot" style="background:{role_color}"></span>'
                f'<span class="overview-title">{escape_html(title)}</span>'
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


# ──────────────────────────────────────────────────────────────────────────────
# CSS stylesheet
# ──────────────────────────────────────────────────────────────────────────────


CSS_STYLESHEET = r"""
/* ══════════════════════════════════════════════════════════════════════════════
   Semantic Document Layout — Tufte Margin + Color Banding v3
   Target: reMarkable Paper Pro (10.3" color e-ink, A4 PDF)
   
   Font size strategy: 3 sizes only
     - body: 7.5pt (everything defaults to this)
     - mono: 6.5pt (code, diagrams, inline code)
     - labels: 6pt (block-type labels, role tags, meta text)
   ══════════════════════════════════════════════════════════════════════════════ */

@page {
  size: A4;
  margin: 10mm 8mm 10mm 8mm;
}

/* ── Base ── */
*, *::before, *::after { box-sizing: border-box; }

body {
  font-family: "DejaVu Sans", "Noto Sans", sans-serif;
  font-size: 7.5pt;
  line-height: 1.4;
  color: #1E293B;
  margin: 0;
  padding: 0;
}

@media screen {
  body {
    max-width: 210mm;
    margin: 0 auto;
    padding: 10mm 8mm;
    background: #fff;
  }
}

/* ── Document header ── */
.doc-header {
  border-bottom: 1.5px solid #1E293B;
  padding-bottom: 4px;
  margin-bottom: 10px;
}

.doc-title {
  font-size: 9pt;
  font-weight: 800;
  margin: 0;
  line-height: 1.2;
}

.doc-subtitle {
  font-size: 7.5pt;
  color: #64748B;
  margin: 1px 0 2px 0;
}

.doc-meta {
  font-size: 6pt;
  color: #64748B;
  margin-top: 2px;
}

.doc-meta span { margin-right: 10px; }

/* ── Row-first two-column layout ── */
.document-body {
  display: block;
}

.section-row {
  display: grid;
  grid-template-columns: 65fr 35fr;
  gap: 16px;
  align-items: start;
  margin-bottom: 6px;
  break-inside: auto;
}

.section-main {
  grid-column: 1;
  min-width: 0;
}

.section-side {
  grid-column: 2;
  min-width: 0;
}

/* Slight hierarchy indent without changing column width */
.section-row.level-3 .section-main { padding-left: 0; }
.section-row.level-4 .section-main { padding-left: 0; }

/* Only 2 heading sizes */
h1.section-title, h2.section-title { font-size: 8pt; font-weight: 700; }
h3.section-title, h4.section-title { font-size: 7.5pt; font-weight: 700; }

.section-title {
  margin: 10px 0 2px 0;
  break-after: avoid;
  line-height: 1.25;
}

/* Role badge: hidden in heading, shown in margin */
.section-title .role-badge { display: none; }

/* Section importance — no underlines; role color carries the signal */
section.importance-critical .section-title { font-weight: 800; }
section.importance-medium .section-title { opacity: 0.92; }
section.importance-low .section-title { opacity: 0.75; }

/* ── Main column: prose ── */
.block-prose p { margin: 0 0 4px 0; }
.block-prose p:last-child { margin-bottom: 0; }

/* ── Inline code ── */
code {
  font-family: "DejaVu Sans Mono", "Noto Sans Mono", monospace;
  font-size: 6.5pt;
  background: #E2E8F0;
  color: #1E293B;
  padding: 0 2px;
  border-radius: 1px;
  font-weight: 500;
}

/* ── Code blocks ── */
.block-code {
  background: #F1F5F9;
  color: #1E293B;
  border: 1px solid #CBD5E1;
  border-radius: 2px;
  padding: 4px 6px;
  margin: 4px 0;
  overflow-x: auto;
  break-inside: avoid;
}

.block-code .code-lang {
  font-family: "DejaVu Sans", sans-serif;
  font-size: 6pt;
  text-transform: uppercase;
  color: #475569;
  margin-bottom: 2px;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.block-code pre {
  margin: 0;
  white-space: pre;
  font-family: "DejaVu Sans Mono", "Noto Sans Mono", monospace;
  font-size: 6.5pt;
  line-height: 1.3;
}

/* ── Command blocks ── */
.block-command {
  background: #F1F5F9;
  color: #1E293B;
  border-left: 2px solid #475569;
  border-radius: 2px;
  padding: 4px 6px;
  margin: 4px 0;
  break-inside: avoid;
}

.block-command pre {
  margin: 0;
  white-space: pre;
  font-family: "DejaVu Sans Mono", "Noto Sans Mono", monospace;
  font-size: 6.5pt;
}

/* ── Diagrams ── */
.block-diagram {
  background: #F1F5F9;
  border: 1.5px solid #94A3B8;
  border-radius: 2px;
  padding: 4px 6px;
  margin: 6px 0;
  break-inside: avoid;
}

.block-diagram .diagram-type {
  font-size: 6pt;
  text-transform: uppercase;
  color: #475569;
  margin-bottom: 2px;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.block-diagram pre.diagram-content {
  margin: 0;
  font-family: "DejaVu Sans Mono", "Noto Sans Mono", monospace;
  font-size: 6.5pt;
  line-height: 1.2;
  white-space: pre;
}

/* ── Tables ── */
.block-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 7pt;
  margin: 4px 0;
}

.block-table th {
  background: #E2E8F0;
  text-align: left;
  padding: 2px 4px;
  border-bottom: 1.5px solid #64748B;
  font-weight: 700;
}

.block-table td {
  padding: 1px 4px;
  border-bottom: 1px solid #94A3B8;
}

.block-table tr:nth-child(even) td { background: #F8FAFC; }
.block-table td code, .block-table th code { font-size: 6pt; }

/* ── Lists ── */
.block-bullet_list ul,
.block-ordered_list ol {
  margin: 2px 0;
  padding-left: 14px;
}

.block-bullet_list li,
.block-ordered_list li {
  margin-bottom: 1px;
  line-height: 1.35;
}

/* ── Margin column: typographic notes ── */
.margin-block {
  border-left: none;
  border-radius: 0;
  padding: 2px 0 3px 0;
  margin-bottom: 5px;
  background: transparent;
  font-size: 7pt;
  line-height: 1.3;
  break-inside: avoid;
}

.margin-block .block-icon {
  float: left;
  margin-right: 3px;
  font-size: 7.5pt;
}

.margin-block .block-type-label {
  display: block;
  font-size: 6pt;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #475569;
  margin-bottom: 1px;
  font-weight: 700;
}

.margin-block .margin-content { clear: both; }
.margin-block .margin-content p { margin: 0; }

.margin-block .margin-content blockquote {
  margin: 2px 0 0 0;
  padding-left: 6px;
  border-left: 1.5px solid #CBD5E1;
  color: #475569;
  font-style: italic;
}

/* Role tag in margin column: same typographic style as semantic labels */
.role-tag {
  border-left: none !important;
  border-radius: 0;
  padding: 0;
  /* Slightly lower than the row top so the label baseline aligns with the title baseline. */
  margin: 12px 0 5px 0;
  font-size: 6pt;
  line-height: 1.15;
  break-inside: avoid;
  background: transparent;
}

.role-tag .role-label {
  font-size: 6pt;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 700;
}

/* ── Merged margin card entries ── */
.merged-card .margin-merged-content {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.merged-card .margin-entry {
  padding-bottom: 3px;
  border-bottom: 1px dotted #CBD5E1;
}

.merged-card .margin-entry:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

/* ── Margin: per-type color via text only ── */
.block-principle.margin-block {
  border-left: none;
  padding-left: 0;
  background: transparent;
  color: #1D4ED8;
}
.block-principle .block-type-label,
.block-principle .block-icon,
.block-principle .margin-content,
.block-principle .margin-merged-content { color: #1D4ED8; }

.block-risk.margin-block { background: transparent; }
.block-risk .block-type-label,
.block-risk .block-icon { color: #B91C1C; }
.block-risk.severity-high.margin-block { font-weight: 600; }
.block-risk.severity-medium .block-type-label,
.block-risk.severity-medium .block-icon { color: #D97706; }
.block-risk.severity-low .block-type-label,
.block-risk.severity-low .block-icon { color: #A16207; }

.block-definition.margin-block { background: transparent; }
.block-definition .block-type-label,
.block-definition .block-icon { color: #6D28D9; }

.block-decision.margin-block { background: transparent; }
.block-decision .block-type-label,
.block-decision .block-icon { color: #047857; }

.block-checklist.margin-block { background: transparent; }
.block-checklist .block-type-label,
.block-checklist .block-icon { color: #B45309; }

.block-quote.margin-block {
  background: transparent;
  font-style: italic;
}
.block-quote .block-type-label,
.block-quote .block-icon { color: #4B5563; }

/* ── Critical pull quote ── */
.critical-pull-quote {
  border-left: 3px solid #1D4ED8;
  background: transparent;
  padding: 2px 0 2px 5px;
  font-size: 7.5pt;
  font-weight: 600;
  margin: 4px 0;
  border-radius: 0;
  break-inside: avoid;
}

/* ── Severity badge ── */
.severity-badge {
  display: inline-block;
  font-size: 5.5pt;
  color: white;
  padding: 0 3px;
  border-radius: 1px;
  margin-left: 2px;
  vertical-align: middle;
  font-weight: 700;
}

/* ── Overview page ── */
.overview-page { padding: 0; }

.overview-header {
  font-size: 9pt;
  font-weight: 800;
  padding-bottom: 3px;
  margin-bottom: 6px;
}

.overview-row {
  display: grid;
  grid-template-columns: 6px 1fr 2fr;
  gap: 6px;
  align-items: start;
  padding: 1px 0;
  font-size: 7pt;
  line-height: 1.3;
}

.overview-row .role-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  margin-top: 2px;
}

.overview-row .overview-title { font-weight: 700; }

.overview-row .overview-summary {
  color: #64748B;
  font-size: 6.5pt;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ── Print ── */
@media print {
  body { font-size: 7.5pt; line-height: 1.4; }
  code, pre { font-size: 6.5pt; }
  h1, h2, h3, h4 { break-after: avoid; }
  .margin-block { break-inside: avoid; }
  .block-code, .block-diagram { break-inside: avoid; }
  .document-body { break-inside: auto; }
  .screen-only { display: none; }
}

@media screen {
  section { margin-bottom: 8px; }
}
"""


# ──────────────────────────────────────────────────────────────────────────────
# HTML template
# ──────────────────────────────────────────────────────────────────────────────

HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{title}</title>
  <style>
{css}
  </style>
</head>
<body>
  <div class="doc-header">
    <h1 class="doc-title">{title}</h1>
    <div class="doc-subtitle">{subtitle}</div>
    <div class="doc-meta">
      <span>Audience: {audience}</span>
      <span>Density: {density}</span>
      <span>Modes: {reading_modes}</span>
    </div>
  </div>
  {overview}
  {body}
</body>
</html>"""


# ──────────────────────────────────────────────────────────────────────────────
# Main render function
# ──────────────────────────────────────────────────────────────────────────────

def parse_yaml(path: str) -> dict:
    """Load and validate semantic YAML."""
    with open(path) as f:
        doc = yaml.safe_load(f)
    assert doc.get("schema_version") == "semantic-document/v1", \
        f"Unsupported schema version: {doc.get('schema_version')}"
    return doc


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

    # Importance threshold label
    threshold_label = ""
    if importance_threshold:
        threshold_label = f" (importance ≥ {importance_threshold})"

    # Overview page
    overview_html = ""
    if overview_page:
        overview_html = render_overview_page(doc, importance_threshold)

    # Body — row-first layout. Each section owns one two-column row, so right
    # column tags/cards align with their section instead of piling up globally.
    section_rows = []
    for section in doc.get("sections", []):
        section_rows.extend(render_section_rows(section, importance_threshold))

    body_html = f'<div class="document-body">{"".join(section_rows)}</div>'

    # Full HTML
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
    print(f"Rendered {yaml_path} → {output_path}")

    # Stats
    total_blocks = body_html.count('class="block-')
    total_sections = body_html.count('<section ')
    print(f"  Sections: {total_sections}, Blocks: {total_blocks}")


# ──────────────────────────────────────────────────────────────────────────────
# CLI
# ──────────────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Convert semantic-document/v1 YAML to styled HTML for reMarkable Paper Pro"
    )
    parser.add_argument("--input", required=True, help="Path to .semantic.yaml file")
    parser.add_argument("--output", required=True, help="Output HTML file path")
    parser.add_argument(
        "--importance-threshold",
        choices=["critical", "high", "medium", "low"],
        default=None,
        help="Filter sections by minimum importance level (omit for full document)"
    )
    parser.add_argument(
        "--no-overview",
        action="store_true",
        help="Skip the overview/map page"
    )
    args = parser.parse_args()

    render_document(
        yaml_path=args.input,
        output_path=args.output,
        importance_threshold=args.importance_threshold,
        overview_page=not args.no_overview,
    )


if __name__ == "__main__":
    main()
