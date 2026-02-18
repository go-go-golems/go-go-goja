---
Title: Cleanup Engine calllog and options API
Ticket: GC-03-CLEANUP-CALLOG
Status: active
Topics:
    - go
    - refactor
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-perf/serve_command.go
      Note: Serve command currently hangs on Ctrl-C
    - Path: engine/config.go
      Note: Legacy config shape feeding calllog behavior
    - Path: engine/runtime.go
      Note: Current engine constructor surface to be refactored
    - Path: pkg/calllog/calllog.go
      Note: Global logger implementation to be scoped per engine
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-18T10:03:33.027841216-05:00
WhatFor: ""
WhenToUse: ""
---


# Cleanup Engine calllog and options API

## Overview

This ticket cleans up engine construction API design around calllog and require
options, and removes global calllog coupling from runtime startup so call
logging can be configured per engine instance. It also includes a stability fix
for `goja-perf serve` so Ctrl-C exits cleanly without hanging.

## Key Links

- Implementation plan: `reference/01-implementation-plan.md`
- Diary: `reference/02-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- engine
- calllog
- api-design

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
