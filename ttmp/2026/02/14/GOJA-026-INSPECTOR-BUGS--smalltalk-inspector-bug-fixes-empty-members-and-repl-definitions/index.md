---
Title: 'Smalltalk Inspector Bug Fixes: Empty Members and REPL Definitions'
Ticket: GOJA-026-INSPECTOR-BUGS
Status: active
Topics:
    - smalltalk-inspector
    - bugs
    - debugging
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/smalltalk-inspector/app/model.go
      Note: buildMembers() and buildGlobals() - root cause of Bug 1 and 2
    - Path: cmd/smalltalk-inspector/app/update.go
      Note: handleGlobalsKey() Enter handler and MsgEvalResult handler - root cause of Bug 2 and 3
    - Path: pkg/inspector/runtime/introspect.go
      Note: InspectObject() - existing API needed for fixes
    - Path: pkg/inspector/runtime/session.go
      Note: GlobalValue() - existing API needed for fixes
    - Path: testdata/inspector-test.js
      Note: Test fixture for reproduction
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-14T16:07:14.390891718-05:00
WhatFor: ""
WhenToUse: ""
---


# Smalltalk Inspector Bug Fixes: Empty Members and REPL Definitions

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- smalltalk-inspector
- bugs
- debugging

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
