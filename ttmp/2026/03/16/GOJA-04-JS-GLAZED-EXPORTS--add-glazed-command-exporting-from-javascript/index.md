---
Title: Add Glazed command exporting from JavaScript
Ticket: GOJA-04-JS-GLAZED-EXPORTS
Status: active
Topics:
    - analysis
    - architecture
    - goja
    - glazed
    - js-bindings
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/design-doc/01-js-to-glazed-command-exporting-design-and-implementation-guide.md
      Note: Primary deliverable for the ticket
    - Path: ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/design-doc/02-js-verbs-prototype-postmortem-and-code-review.md
      Note: Postmortem and code review deliverable for the prototype branch
    - Path: ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/reference/01-diary.md
      Note: Chronological investigation diary
    - Path: ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/sources/local/01-goja-js.md
      Note: Imported source note that motivated the design work
ExternalSources:
    - local:01-goja-js.md
Summary: Research ticket for designing a repo-native JS-to-Glazed command layer in go-go-goja, grounded in the current jsdoc extractor, Goja engine factory, and Glazed command system.
LastUpdated: 2026-03-16T13:56:35.761467442-04:00
WhatFor: Track the analysis and implementation planning work for exposing JavaScript-defined functions as ordinary Glazed commands in go-go-goja.
WhenToUse: Use when reviewing or implementing JS-to-Glazed command discovery, compilation, and runtime invocation in this repository.
---




# Add Glazed command exporting from JavaScript

## Overview

This ticket captures the design work for a new `pkg/jsverbs`-style subsystem in `go-go-goja`. The goal is to discover JavaScript command definitions statically, normalize them into a registry, compile them into regular Glazed commands, and invoke the matching JavaScript function at runtime through the existing Goja engine/runtime seams.

The imported source note from `sources/local/01-goja-js.md` is preserved in the ticket, but the primary deliverable here is a grounded repository-specific interpretation: which current packages provide the extraction/runtime/CLI seams already, what needs to be added, what the risks are, and how an intern should implement and validate the work.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- analysis
- architecture
- goja
- glazed
- js-bindings
- tooling

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
