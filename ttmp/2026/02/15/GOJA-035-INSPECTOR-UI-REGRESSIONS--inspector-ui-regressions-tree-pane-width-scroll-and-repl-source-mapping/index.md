---
Title: 'Inspector UI regressions: tree pane width/scroll and REPL source mapping'
Ticket: GOJA-035-INSPECTOR-UI-REGRESSIONS
Status: complete
Topics:
    - go
    - goja
    - inspector
    - bugfix
    - ui
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Tree pane split and render behavior.
    - Path: go-go-goja/cmd/inspector/app/tree_list.go
      Note: AST row display shaping.
    - Path: go-go-goja/cmd/smalltalk-inspector/app/model.go
      Note: REPL source fallback.
    - Path: go-go-goja/cmd/smalltalk-inspector/app/update.go
      Note: REPL source append pipeline.
ExternalSources: []
Summary: Ticket workspace for fixing inspector tree pane UX issues and REPL source-window regression.
LastUpdated: 2026-02-15T15:20:04.956662591-05:00
WhatFor: Track analysis, implementation, and validation of GOJA-035 bugfix work.
WhenToUse: Use when reviewing or extending these UI fixes.
---


# Inspector UI regressions: tree pane width/scroll and REPL source mapping

## Overview

This ticket fixes two regressions reported after inspector refactors:

1. `cmd/inspector` tree pane ergonomics were poor in tmux/narrow widths.
2. `cmd/smalltalk-inspector` no longer showed REPL source when selecting REPL-defined globals.

## Key Links

- Analysis: `design/01-tmux-analysis-tree-pane-width-scroll-and-repl-symbol-source-regression.md`
- Diary: `reference/01-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Repro script + captures: `scripts/`

## Status

Current status: **active** (implementation complete, pending commit/push in this working session).
