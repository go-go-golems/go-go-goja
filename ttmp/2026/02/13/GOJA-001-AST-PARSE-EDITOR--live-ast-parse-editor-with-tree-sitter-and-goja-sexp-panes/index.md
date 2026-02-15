---
Title: Live AST parse editor with tree-sitter and goja SEXP panes
Ticket: GOJA-001-AST-PARSE-EDITOR
Status: complete
Topics:
    - goja
    - analysis
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/02/13/GOJA-001-AST-PARSE-EDITOR--live-ast-parse-editor-with-tree-sitter-and-goja-sexp-panes/analysis/01-tree-sitter-ast-live-sexp-editor-analysis.md
      Note: Primary technical analysis deliverable
    - Path: go-go-goja/ttmp/2026/02/13/GOJA-001-AST-PARSE-EDITOR--live-ast-parse-editor-with-tree-sitter-and-goja-sexp-panes/reference/01-diary.md
      Note: Detailed diary of commands
ExternalSources: []
Summary: Ticket index for analysis and delivery artifacts for a live tree-sitter + goja AST SEXP editor design.
LastUpdated: 2026-02-14T20:00:09.291765527-05:00
WhatFor: Central navigation for GOJA-001-AST-PARSE-EDITOR docs and outcomes.
WhenToUse: Use to quickly locate the analysis, diary, tasks, changelog, and upload destination.
---



# Live AST parse editor with tree-sitter and goja SEXP panes

## Overview

This ticket captures a detailed implementation analysis for a new live 3-pane editor in `go-go-goja`:

- left: editable JavaScript source
- middle: tree-sitter CST rendered as LISP SEXP
- right: goja AST rendered as LISP SEXP when parse-valid

The analysis references current `pkg/jsparse` and `cmd/inspector` code paths and provides a file-by-file, phased implementation plan.

## Key Links

- Analysis: `analysis/01-tree-sitter-ast-live-sexp-editor-analysis.md`
- Diary: `reference/01-diary.md`
- Changelog: `changelog.md`
- reMarkable upload target: `/ai/2026/02/13/GOJA-001-AST-PARSE-EDITOR/GOJA-001-AST-PARSE-EDITOR Analysis`

## Status

Current status: **active**

## Topics

- goja
- analysis
- tooling

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
