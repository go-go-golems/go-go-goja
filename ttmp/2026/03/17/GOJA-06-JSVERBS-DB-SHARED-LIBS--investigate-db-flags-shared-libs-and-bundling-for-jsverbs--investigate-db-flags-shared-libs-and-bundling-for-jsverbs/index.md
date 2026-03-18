---
Title: Investigate db flags, shared libs, and bundling for jsverbs
Ticket: GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs
Status: complete
Topics:
    - go
    - glazed
    - js-bindings
    - sqlite
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/jsverbs-example/main.go
      Note: Experiment runner
    - Path: modules/database/database.go
      Note: Database module lifecycle
    - Path: pkg/jsverbs/binding.go
      Note: Section validation and binding rules
    - Path: pkg/jsverbs/runtime.go
      Note: Runtime and require-loader behavior
    - Path: pkg/jsverbs/scan.go
      Note: Core scan-time architecture
    - Path: ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/scripts
      Note: Ticket-local evidence and runnable experiments
ExternalSources: []
Summary: Ticket collecting an intern-oriented jsverbs architecture guide, a diary of the investigation, and runnable experiments for db flags, shared libraries, section reuse, and bundling behavior.
LastUpdated: 2026-03-17T14:55:57.161972342-04:00
WhatFor: Explain the current jsverbs architecture and document the present and desired behavior around db-backed verbs, shared helpers, and bundled artifacts.
WhenToUse: Use when onboarding an engineer to jsverbs or when planning runner-level APIs for shared sections or db initialization.
---



# Investigate db flags, shared libs, and bundling for jsverbs

## Overview

This ticket investigates whether `pkg/jsverbs` can support db-backed verbs, shared helper libraries, and bundle-based packaging in a way that scales beyond the example runner. The main deliverable is an intern-oriented analysis that explains the system layer by layer and grounds each claim in the current codebase and ticket-local experiments.

Current conclusion:

- `--db` flags are already possible today if the JS verb calls `require("database").configure(...)`.
- Shared runtime helpers already work today through `require()` as long as the files are in the scanned tree or bundled into the final artifact.
- Shared metadata sections do not work across files today because section lookup is file-local.
- Bundling works, but the bundled output must preserve scanner-visible command functions.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary analysis**: [design-doc/01-jsverbs-db-flags-shared-libraries-and-bundling-investigation-guide.md](./design-doc/01-jsverbs-db-flags-shared-libraries-and-bundling-investigation-guide.md)
- **Diary**: [reference/01-diary.md](./reference/01-diary.md)
- **Experiments**: `scripts/run-exp01.sh`, `scripts/run-exp02.sh`, `scripts/run-exp03.sh`

## Status

Current status: **active**

## Topics

- goja
- glazed
- javascript

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
