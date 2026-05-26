---
Title: Semantic document YAML schema spec
Ticket: XGOJA-014
Status: complete
Topics:
    - xgoja
    - providers
    - command-registration
    - goja
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.md
      Note: |-
        Source Markdown document converted into semantic YAML
        Source Markdown guide converted into semantic YAML
    - Path: ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml
      Note: |-
        Full semantic YAML rendition of the context ownership guide using this schema
        Full semantic YAML representation using this schema
ExternalSources: []
Summary: Compact schema specification for designer-facing semantic document YAML.
LastUpdated: 2026-05-26T08:55:00-04:00
WhatFor: Give a designer or layout system semantic content structure without prescribing typography or page geometry.
WhenToUse: Use when converting long-form technical Markdown into YAML that preserves informational intent for design/layout decisions.
---


# Semantic document YAML schema spec

## Goal

This document defines a small YAML schema for representing a technical guide as semantic content rather than as visual Markdown. The schema is intentionally modest. It preserves document hierarchy, block content, intent, importance, and block type so a designer can make typographic and layout decisions from meaning instead of from Markdown syntax alone.

The full YAML rendition of the context ownership guide is here:

```text
/home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-014--add-xgoja-command-providers-to-go-minitrace-loupedeck-and-css-visual-diff/design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml
```

## Context

Markdown says that some text is a paragraph, a heading, a list, or a code fence. It does not say whether that paragraph is a principle, a risk, a definition, an implementation instruction, a design decision, or a transitional explanation. Designers need those distinctions when laying out a document that should be readable as both a teaching text and an implementation reference.

This schema does not encode typography. It does not say “font size 18” or “make this red.” Instead, it says “this is a critical principle” or “this is a high-severity risk.” The designer or layout engine decides how those semantic roles should look.

## Quick reference

Top-level shape:

```yaml
schema_version: semantic-document/v1

document:
  id: string
  title: string
  subtitle: string
  version: string
  source_markdown: string
  source_ticket: string
  generated_at: string
  audience: [string]
  intent: string
  density: low | medium | high
  reading_modes: [string]
  frontmatter: object

semantic_vocabulary:
  section_roles: [string]
  block_types: [string]
  importance: [critical, high, medium, low]

sections:
  - id: string
    title: string
    level: integer
    role: string
    importance: critical | high | medium | low
    summary: string
    blocks: [Block]
    sections: [Section]
```

## Section object

A section corresponds to a heading and all content beneath it until the next heading at the same or higher level.

```yaml
id: proposed-api-direction
title: Proposed API direction
level: 2
role: proposed-architecture
importance: critical
summary: The proper change is to turn context intent into names and methods.
blocks: []
sections: []
```

Fields:

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | yes | Stable slug for cross-reference and layout anchors. |
| `title` | yes | Human-readable section title. |
| `level` | yes | Original heading level from the source document. |
| `role` | yes | Semantic role of the section. |
| `importance` | yes | Relative editorial importance. |
| `summary` | optional | Short extract or authored summary for layout previews and navigation. |
| `blocks` | yes | Ordered content blocks directly inside the section. |
| `sections` | yes | Nested child sections. |

Recommended section roles:

```yaml
section_roles:
  - summary
  - problem-statement
  - concept-foundation
  - architecture-map
  - case-study
  - strengths-analysis
  - gap-analysis
  - risk-analysis
  - proposed-architecture
  - authoring-guidelines
  - api-reference
  - implementation-plan
  - implementation-phase
  - migration-guide
  - diagram-section
  - test-strategy
  - open-questions
  - conclusion
  - exposition
```

## Block object

A block is the smallest layout unit. It preserves content and adds semantic intent.

Common fields:

```yaml
id: b0042
type: prose
intent: explain
content: >
  The runtime lifecycle context lives as long as the runtime.
```

Optional fields:

```yaml
language: go
diagram_type: ascii_sequence
caption: HTTP request entering the JS owner
importance: high
severity: high
```

Recommended block types:

| Type | Meaning | Typical layout treatment |
| --- | --- | --- |
| `prose` | Explanatory paragraph. | Normal text. |
| `principle` | Rule or invariant the reader should remember. | Pull quote, rule box, or emphasized paragraph. |
| `quote` | Quoted or emphasized text from the source. | Quote block. |
| `definition` | Term definition. | Glossary/card treatment. |
| `risk` | Problem/consequence/mitigation block. | Warning/risk panel. |
| `decision` | Design decision and rationale. | Decision record panel. |
| `code` | Source code or pseudocode. | Monospace code block. |
| `command` | Shell command. | Command block with copy affordance. |
| `diagram` | ASCII or structured diagram. | Full-width monospace diagram panel. |
| `table` | Markdown table content. | Table layout. |
| `bullet_list` | Unordered list. | List. |
| `ordered_list` | Ordered list. | Numbered procedure or sequence. |
| `checklist` | Checkbox list. | Checklist card. |

Recommended intents:

```yaml
intents:
  - orient-reader
  - explain
  - define-core-term
  - remember
  - warn
  - compare-or-classify
  - show-api-or-pseudocode
  - show-js-example
  - show-command
  - explain-flow
  - enumerate
  - validate
  - implement
```

## Importance and severity

`importance` is editorial priority. It helps determine visual prominence.

```yaml
importance: critical | high | medium | low
```

`severity` is only for risks, warnings, and failure modes.

```yaml
severity: high | medium | low
```

The two should not be conflated. A low-severity detail can still be important for navigation. A high-severity risk can be visually strong even if it appears in a lower-level section.

## Designer-facing layout rules

A designer can map semantics to layout without changing the content file.

Example rules:

```yaml
layout_rules:
  - match:
      type: principle
      importance: critical
    treatment: pull_quote_box

  - match:
      type: definition
    treatment: glossary_card

  - match:
      type: risk
      severity: high
    treatment: warning_panel

  - match:
      type: diagram
    treatment: full_width_monospaced_panel

  - match:
      role: implementation-phase
    treatment: numbered_process_section
```

These rules are intentionally separate from the semantic content. The content file says what the material is; the design system says how it should be presented.

## Usage example

A small semantic section:

```yaml
sections:
  - id: runner-definition
    title: What a Runner is
    level: 3
    role: api-reference
    importance: high
    summary: A Runner is the runtime-owned serialized execution API for the VM.
    blocks:
      - id: b0001
        type: prose
        intent: define-core-term
        content: >
          A Runner is the runtime-owned serialized execution API for the VM.
          It is not a general goroutine runner.

      - id: b0002
        type: code
        intent: show-api-or-pseudocode
        language: go
        content: |
          type Runner interface {
              Call(ctx context.Context, op string, fn CallFunc) (any, error)
              Post(ctx context.Context, op string, fn PostFunc) error
              Shutdown(context.Context) error
              IsClosed() bool
          }
    sections: []
```

A renderer can treat the first block as a definition paragraph and the second block as an API reference, even though both would be “just text” in Markdown.

## Generation notes

The current semantic YAML file was generated from the Markdown guide using a lightweight Markdown block parser and semantic classifier. The parser preserves the full source content in order. It assigns section roles and block types by simple heading/content heuristics, then emits valid YAML.

The generated YAML should be treated as a designer-facing content interchange file, not as the canonical source of truth. The Markdown guide remains the authored source. The YAML can be regenerated and then hand-enriched if the design process needs more precise semantics.

## Validation

The generated YAML should parse with a standard YAML parser:

```bash
python3 - <<'PY'
import yaml
from pathlib import Path
p = Path('design-doc/01-goja-context-ownership-and-runtime-package-composition-guide.semantic.yaml')
yaml.safe_load(p.read_text())
print('ok')
PY
```

Minimum validation rules:

- `schema_version` must equal `semantic-document/v1`.
- `document.id` and `document.title` must be present.
- Every section must have `id`, `title`, `level`, `role`, `importance`, `blocks`, and `sections`.
- Every block must have `id`, `type`, `intent`, and `content`.
- `code` blocks should include `language` when known.
- `diagram` blocks should include `diagram_type` when known.

## Scope boundaries

This schema deliberately does not model:

- page size;
- font families;
- colors;
- margins;
- exact line breaks outside code/diagram blocks;
- output-specific assets;
- cross-platform typography constraints.

Those belong in a layout or theme layer. The semantic YAML should remain portable and content-focused.
