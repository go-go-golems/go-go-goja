---
Title: 'PR #28 Review: REPL Service Architecture Bug Report and Regression Analysis'
Ticket: GOJA-044-PR28-REPL-SERVICE-REVIEW
Status: active
Topics:
    - goja
    - repl
    - code-review
    - architecture
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: CQ-8 10 commands in one file
    - Path: pkg/jsparse/resolve.go
      Note: BUG-1 crash when Program.Body is empty
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: CQ-7 legacy evaluator with duplicated logic
    - Path: pkg/replapi/config.go
      Note: CQ-3 dual SessionOptions naming
    - Path: pkg/repldb/types.go
      Note: BUG-3 missing json struct tags causes PascalCase
    - Path: pkg/replhttp/handler.go
      Note: |-
        BUG-2 no panic recovery
        CQ-9 manual string-split routing
    - Path: pkg/replsession/evaluate.go
      Note: BUG-1 empty source crash
    - Path: pkg/replsession/observe.go
      Note: BUG-5 sessionBound timing
    - Path: pkg/replsession/policy.go
      Note: CQ-6 dead PersistPolicy fields
    - Path: pkg/replsession/rewrite.go
      Note: |-
        IIFE wrapper is the root cause for BUG-6
        CQ-2 static analysis functions mixed with rewrite logic
    - Path: pkg/replsession/types.go
      Note: CQ-10 32 DTO structs need grouping comments
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-20T09:29:58.182295998-04:00
WhatFor: ""
WhenToUse: ""
---



# PR #28 Review: REPL Service Architecture Bug Report and Regression Analysis

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- repl
- code-review
- architecture
- testing

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
