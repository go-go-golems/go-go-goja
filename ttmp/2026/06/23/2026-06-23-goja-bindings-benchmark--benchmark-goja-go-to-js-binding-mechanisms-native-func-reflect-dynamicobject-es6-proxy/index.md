---
Title: Benchmark goja Go-to-JS binding mechanisms (native func, reflect, DynamicObject, ES6 Proxy)
Ticket: 2026-06-23-goja-bindings-benchmark
Status: complete
Topics:
    - goja
    - js-bindings
    - testing
    - architecture
DocType: index
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-23T22:24:29.793707172-04:00
WhatFor: ""
WhenToUse: ""
---


# Benchmark goja Go-to-JS binding mechanisms (native func, reflect, DynamicObject, ES6 Proxy)

## Overview

Benchmark and document the cost of every mechanism by which the goja JavaScript runtime exposes Go objects and functions to JavaScript, with a focus on the overhead of a Go-backed ES6 Proxy with manual dispatch. Delivered a Go benchmark suite, an intern-grade design guide, a reMarkable bundle, a self-contained HTML report with an inline-SVG figure pipeline, and two Obsidian vault articles.

Headline finding: call latency is allocation-bound (~60–86 ns per allocation above the 3-alloc native minimum); a Proxy Get costs ~192 ns / 2 allocs (~1.6× a plain object); a DynamicObject matches a plain-object read (~124 ns / 1 alloc); a Proxy Apply adds no allocations over a native call.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete** (2026-06-23). All 7 tasks done; deliverables committed and the two vault articles pushed to go-go-parc origin/main.

## Topics

- goja
- js-bindings
- testing
- architecture

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
