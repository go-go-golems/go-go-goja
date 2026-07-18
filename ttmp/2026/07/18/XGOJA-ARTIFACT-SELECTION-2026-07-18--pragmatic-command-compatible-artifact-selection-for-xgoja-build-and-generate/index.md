---
Title: Pragmatic command-compatible artifact selection for xgoja build and generate
Ticket: XGOJA-ARTIFACT-SELECTION-2026-07-18
Status: complete
Topics:
    - xgoja
    - backend
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ws://go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md
      Note: Public documentation file
    - Path: ws://go-go-goja/cmd/xgoja/v2_plan_helpers.go
      Note: Core implementation file
    - Path: ws://go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/design-doc/01-intern-guide-to-xgoja-artifact-selection.md
      Note: Primary implementation guide
ExternalSources: []
Summary: Design and implementation ticket for deterministic build/generate artifact selection using one compatible primary plus global support artifacts.
LastUpdated: 2026-07-18T17:32:01.390048914-04:00
WhatFor: Coordinate implementation, testing, review, and delivery of the xgoja multi-artifact order fix.
WhenToUse: Start here when implementing or reviewing command-compatible artifact selection.
---



# Pragmatic command-compatible artifact selection for xgoja build and generate

## Overview

This ticket fixes an order-dependent mismatch in xgoja/v2 specifications containing both build and source-generation outputs. Today, `xgoja build` and `xgoja generate` select the same first non-support artifact and then reject kinds incompatible with the invoked command. Generator rendering also derives its target independently from the original artifact list, so command output and embedded runtime metadata can disagree.

The accepted pragmatic design selects exactly one command-compatible primary, constructs a shallow scoped plan containing that primary plus global `dts` and `embedded-assets` support artifacts, and passes the scoped plan through existing generators. It deliberately defers explicit artifact flags, dependency graphs, and multi-output orchestration.

## Key documents

- [Intern guide to pragmatic xgoja artifact selection](design-doc/01-intern-guide-to-xgoja-artifact-selection.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Tasks](tasks.md)
- [Changelog](changelog.md)
- [Reproduction script](scripts/01-reproduce-artifact-order.sh)
- [Captured reproduction output](scripts/01-reproduce-artifact-order.log)

## Current status

- Failure reproduced against commit `69b69b6`.
- Architecture and generator behavior mapped.
- Pragmatic selection/scoping implementation committed in `7caaee6`.
- Command regressions and public documentation committed in `4003433`.
- Focused/full test, vet, lint, and generated-code hooks pass.
- Ready for review; release/versioning is intentionally outside this implementation ticket.

## Accepted scope

### Included

- Exactly-one command-compatible primary selection.
- Ambiguity/no-match diagnostics with artifact IDs/types.
- Shallow scoped plan preserving global support artifacts.
- Target metadata and embedded-source consistency.
- Unit, command, and embedding regressions.
- Concise v2 artifact documentation.

### Deferred

- `--artifact`.
- Artifact dependency relationships.
- Building/generating multiple compatible primaries.
- Parallel output orchestration.
- General planner/generator redesign.

## Review invariant

For one command invocation, command diagnostics, output selection, generated target metadata, artifact metadata, and embedded primary sources must all refer to the same selected primary artifact.
