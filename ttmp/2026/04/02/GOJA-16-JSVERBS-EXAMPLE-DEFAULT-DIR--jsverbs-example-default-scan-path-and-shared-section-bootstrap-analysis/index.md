---
Title: jsverbs-example default scan path and shared section bootstrap analysis
Ticket: GOJA-16-JSVERBS-EXAMPLE-DEFAULT-DIR
Status: active
Topics:
    - goja
    - glazed
    - documentation
    - analysis
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/jsverbs-example/main.go
      Note: Primary example-runner bootstrap file
    - Path: go-go-goja/pkg/jsverbs/binding.go
      Note: Unknown-section failure point
    - Path: go-go-goja/pkg/jsverbs/model.go
      Note: Registry and shared-section model
    - Path: go-go-goja/ttmp/2026/04/02/GOJA-16-JSVERBS-EXAMPLE-DEFAULT-DIR--jsverbs-example-default-scan-path-and-shared-section-bootstrap-analysis/design-doc/01-jsverbs-example-default-scan-path-shared-section-bootstrap-design-and-implementation-guide.md
      Note: Primary design analysis
    - Path: go-go-goja/ttmp/2026/04/02/GOJA-16-JSVERBS-EXAMPLE-DEFAULT-DIR--jsverbs-example-default-scan-path-and-shared-section-bootstrap-analysis/reference/01-investigation-diary.md
      Note: Chronological investigation record
ExternalSources: []
Summary: Investigates the jsverbs-example zero-argument failure and tracks the design, diary, and follow-up plan for fixing the example runner's default-directory behavior.
LastUpdated: 2026-04-02T08:52:36.347506866-04:00
WhatFor: Ticket workspace for analyzing the jsverbs-example default-directory failure and documenting a safe follow-up implementation plan.
WhenToUse: Use when reviewing the current diagnosis, continuing the follow-up code fix, or finding the design and diary deliverables for this issue.
---


# jsverbs-example default scan path and shared section bootstrap analysis

## Overview

This ticket captures an architecture-level investigation of why `go run ./cmd/jsverbs-example` fails from the repository root with an unknown `filters` section reference. The ticket concludes that the failure is caused by example-runner bootstrap/default-directory behavior, not by a need to load `testdata/jsverbs/basics.js` first.

The primary deliverable is a detailed intern-facing design and implementation guide explaining how `jsverbs` work end to end: scanning, section resolution, binding plans, command compilation, runtime invocation, and the exact bootstrap mismatch behind the current failure.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design Doc**: `design-doc/01-jsverbs-example-default-scan-path-shared-section-bootstrap-design-and-implementation-guide.md`
- **Diary**: `reference/01-investigation-diary.md`

## Status

Current status: **active**

## Topics

- goja
- glazed
- documentation
- analysis
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
