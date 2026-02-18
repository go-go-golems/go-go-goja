---
Title: Display Flamegraphs In goja-perf Web UI
Ticket: GJ-03-FLAMEGRAPH-UI
Status: active
Topics:
    - ui
    - tooling
    - goja
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-perf/serve_command.go
      Note: Main server and dashboard templates to extend with Profiles view
    - Path: cmd/goja-perf/serve_streaming.go
      Note: Run status lifecycle that should coexist with profile view state
    - Path: cmd/goja-perf/phase1_types.go
      Note: Run report schema extension point for profile metadata
    - Path: ttmp/2026/02/18/GJ-03-FLAMEGRAPH-UI--display-flamegraphs-in-goja-perf-web-ui/reference/01-flamegraph-web-ui-plan.md
      Note: Handoff-ready implementation plan for frontend developer
ExternalSources: []
Summary: Ticket for planning flamegraph/profile artifact visualization in the upgraded goja-perf web dashboard.
LastUpdated: 2026-02-18T15:22:30-05:00
WhatFor: Coordinate backend and frontend work to expose flamegraph artifacts safely and clearly.
WhenToUse: Use when implementing or reviewing profile/flamegraph support in goja-perf serve mode.
---

# Display Flamegraphs In goja-perf Web UI

## Overview

This ticket captures the implementation plan to display flamegraph/profile
artifacts in the upgraded `goja-perf` web UI. It defines the data contract,
backend endpoint changes, UI integration strategy, and acceptance criteria for
handoff to frontend implementation.

## Key Links

- Plan doc: `reference/01-flamegraph-web-ui-plan.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- ui
- tooling
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
