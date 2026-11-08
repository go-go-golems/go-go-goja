---
Title: 'Host runtime options for embedders: bridge owner, optional globals, plain-VM jsverbs invoke, pure-Go jsverbs scanner'
Ticket: GOJA-069
Status: active
Topics:
    - engine
    - jsverbs
    - parser
    - upstream
    - goja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Engine host options, plain-VM jsverbs invoke, pure-Go jsverbs scanner with tree-sitter opt-in"
LastUpdated: 2026-11-07T14:12:59.484029229-05:00
WhatFor: ""
WhenToUse: ""
---

# Host runtime options for embedders: bridge owner, optional globals, plain-VM jsverbs invoke, pure-Go jsverbs scanner

## Overview

Changes go-go-goja needs so that slop (SLOP3-016) can host actors on an engine runtime and run app CLI verbs without cgo grammars:

- `engine.WithRuntimeBridgeOwner` (RuntimeOption): replace the owner modules find through runtimebridge, before modules register.
- `engine.WithoutDefaultGlobals` (builder Option): skip console, console timers, performance, Buffer, URL.
- `jsverbs.Registry.InvokeInVM`: run a verb on a caller-owned plain `*goja.Runtime`; a still-pending promise is returned to the host.
- jsverbs scans with goja's own parser by default (pure Go); tree-sitter moves to `pkg/jsverbs/treesitter` (opt-in via `ScanOptions.Parser`); parity tests over all 22 verb trees in the repository.
- Explicit `__verb__` metadata was already required (833edd5); stale docs fixed.

Diary: `reference/01-diary.md`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- engine
- jsverbs
- parser
- upstream
- goja

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
