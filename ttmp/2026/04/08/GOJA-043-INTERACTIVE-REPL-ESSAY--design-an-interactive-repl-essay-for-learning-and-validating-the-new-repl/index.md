---
Title: Design an interactive REPL essay for learning and validating the new REPL
Ticket: GOJA-043-INTERACTIVE-REPL-ESSAY
Status: active
Topics:
    - repl
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:repl-essay(1).jsx
Summary: Design-first ticket for a Bret Victor-style interactive article that teaches the new REPL by exercising the real session and HTTP APIs.
LastUpdated: 2026-04-14T20:19:36.91046559-04:00
WhatFor: Plan a high-signal interactive teaching artifact that doubles as a behavioral validation surface for the new REPL.
WhenToUse: Use when designing or implementing an interactive article, dynamic essay, or tutorial UI for the REPL/session stack.
---


# Design an interactive REPL essay for learning and validating the new REPL

## Overview

This ticket captures the design for an interactive article or dynamic essay that teaches the new REPL by directly exercising the real system. The goal is not to ship the article in this ticket. The goal is to define the strongest possible teaching and validation artifact first, including which parts must be live, which widgets matter, which APIs are already available, and which missing surfaces should be added before implementation.

The guiding idea is simple: the article should not merely describe the REPL. It should let the reader create sessions, run cells, inspect rewrites, compare profiles, visualize runtime diffs, replay history, and observe timeout recovery against the real backend behavior. That makes the article educational and operational at the same time.

## Key Links

- Main design doc: [design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md](./design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md)
- UX handoff: [design-doc/02-ux-handoff-for-the-interactive-repl-essay.md](./design-doc/02-ux-handoff-for-the-interactive-repl-essay.md)
- Frontend implementation guide: [design-doc/03-frontend-implementation-guide-for-modular-storybook-repl-essay-ui.md](./design-doc/03-frontend-implementation-guide-for-modular-storybook-repl-essay-ui.md)
- Diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Existing REPL usage doc: [/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/doc/04-repl-usage.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/doc/04-repl-usage.md)
- REPL hardening report: [/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/08/repl-hardening-project-report.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/08/repl-hardening-project-report.md)
- GOJA-040 persistence guide: [/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-040-PERSISTENCE-CORRECTNESS--fix-repl-persistence-correctness-and-sqlite-integrity/design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-040-PERSISTENCE-CORRECTNESS--fix-repl-persistence-correctness-and-sqlite-integrity/design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md)
- GOJA-041 evaluation guide: [/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/design-doc/01-evaluation-control-analysis-design-and-implementation-guide.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/design-doc/01-evaluation-control-analysis-design-and-implementation-guide.md)
- GOJA-042 cleanup guide: [/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-042-REPL-CLEANUP--refactor-session-kernel-and-api-shape-cleanup/design-doc/01-repl-cleanup-analysis-design-and-implementation-guide.md](/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-042-REPL-CLEANUP--refactor-session-kernel-and-api-shape-cleanup/design-doc/01-repl-cleanup-analysis-design-and-implementation-guide.md)

## Status

Current status: **active**

This ticket is active as a design and planning deliverable. The UX handoff is now included, and implementation remains future work.

## Topics

- repl
- documentation

## Tasks

See [tasks.md](./tasks.md) for the completed deliverables and deferred follow-up implementation work.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
