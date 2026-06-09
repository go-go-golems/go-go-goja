---
Title: Support external Go HTTP hosts for generated xgoja runtimes
Ticket: XGOJA-EXTERNAL-HTTP-HOST
Status: active
Topics:
    - xgoja
    - goja
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/65
Summary: Ticket for the non-invasive external Go HTTP host integration plan for generated xgoja runtime packages.
LastUpdated: 2026-06-09T01:25:00-04:00
WhatFor: Track design and future implementation tasks for injecting external gojahttp.Host services into generated xgoja runtimes.
WhenToUse: Use when planning or implementing generated package service injection, HTTP provider external-host mode, route introspection, or runtime-manager validation.
---

# Support external Go HTTP hosts for generated xgoja runtimes

## Overview

This ticket tracks the non-invasive implementation plan for letting generated xgoja runtime packages run inside a Go-owned HTTP server. The immediate goal is to allow a host application to inject a `*gojahttp.Host` into the generated runtime's existing `HostServices` service bag, then have the xgoja HTTP provider use that host for `require("express")` route registration without binding its own TCP listener.

The larger provider API naming cleanup is intentionally out of scope for this ticket. It is tracked separately in GitHub issue #65.

## Key Links

- **Primary guide:** [design-doc/01-external-go-http-host-integration-implementation-guide.md](design-doc/01-external-go-http-host-integration-implementation-guide.md)
- **Diary:** [reference/01-investigation-diary.md](reference/01-investigation-diary.md)
- **Future rename issue:** https://github.com/go-go-golems/go-go-goja/issues/65
- **Related files:** See frontmatter `RelatedFiles` in the primary guide.

## Status

Current status: **active**.

Documentation work is complete. Future implementation work is listed in [tasks.md](./tasks.md).

## Topics

- xgoja
- goja
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` — Architecture and implementation guide.
- `reference/` — Investigation diary and reusable context.
- `playbooks/` — Future command sequences and test procedures.
- `scripts/` — Temporary code and tooling if implementation experiments need them.
- `sources/` — Source snapshots or generated evidence, if needed.
- `various/` — Working notes.
- `archive/` — Deprecated or reference-only artifacts.
