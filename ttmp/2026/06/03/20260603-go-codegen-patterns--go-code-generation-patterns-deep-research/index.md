---
Title: 'Go Code Generation Patterns: Deep Research'
Ticket: 20260603-go-codegen-patterns
Status: active
Topics:
    - go
    - code-generation
    - patterns
    - metaprogramming
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-03T17:13:54.026085551-04:00
WhatFor: ""
WhenToUse: ""
---

# Go Code Generation Patterns: Deep Research

## Overview

This ticket contains a deep research survey of Go code generation patterns. The primary deliverables are:

1. **Comprehensive Research Report** (`design-doc/`) — a 750-line intern-friendly document covering six pattern families: `go:generate` tools, `text/template` generation, `go/ast` rewriting, schema-first codegen, compile-time binary builders (xcaddy-style), and compile-time DI (Wire).
2. **Research Logbook** (`reference/`) — evaluations of 18 external sources with usefulness, staleness, and action ratings.
3. **Investigation Diary** (`reference/`) — chronological record of search, defuddle, source fetching, writing, and upload steps.
4. **External Sources** (`sources/articles/`) — 17 defuddled/curl'd articles and source files for offline reference.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- go
- code-generation
- patterns
- metaprogramming

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
