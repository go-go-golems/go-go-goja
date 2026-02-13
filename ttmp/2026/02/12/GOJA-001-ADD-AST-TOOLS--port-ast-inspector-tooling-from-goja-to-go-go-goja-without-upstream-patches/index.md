---
Title: Port AST inspector tooling from goja to go-go-goja without upstream patches
Ticket: GOJA-001-ADD-AST-TOOLS
Status: active
Topics:
    - goja
    - analysis
    - tooling
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Analysis ticket for extracting a reusable JS parsing/completion framework plus inspector-specific tooling in go-go-goja."
LastUpdated: 2026-02-12T16:13:54.828262977-05:00
WhatFor: "Track analysis, decisions, and execution plan for inspector migration."
WhenToUse: "Read this first to understand ticket status and jump to detailed docs."
---

# Port AST inspector tooling from goja to go-go-goja without upstream patches

## Overview

This ticket captures a full technical analysis for moving all inspector functionality out of the local `goja` fork and into `go-go-goja`, with a split architecture between reusable JS analysis APIs and inspector-specific UI tooling.

Primary objective:
- keep upstream `github.com/dop251/goja` unmodified for this feature set
- preserve inspector behavior and test coverage in `go-go-goja`
- produce a general purpose reusable JS parsing/completion framework for dev tools, diagnostics, and better errors

Current status:
- analysis complete
- implementation pending

## Key Links

- Migration analysis: `reference/02-porting-analysis.md`
- Working diary: `reference/01-diary.md`
- Task list: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- goja
- analysis
- tooling
- migration

## Tasks

See `tasks.md` for concrete implementation checklist and completed analysis work.

## Changelog

See `changelog.md` for dated updates.

## Structure

- `design/` - Architecture and design documents
- `reference/` - Analysis, diary, and reusable technical notes
- `playbooks/` - Command sequences and validation procedures
- `scripts/` - Temporary tooling/scripts for ticket work
- `various/` - Scratch notes and intermediate artifacts
- `archive/` - Deprecated or historical artifacts
