---
Title: Self-contained bun demo and bundling playbook
Ticket: BUN-004
Status: complete
Topics:
    - bun
    - bundling
    - docs
    - typescript
    - goja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:46:17.649400311-05:00
WhatFor: Track the relocation of the bun demo assets and the creation of a comprehensive bundling playbook.
WhenToUse: Use when working on bun-based bundling demos or the documentation playbook.
---


# Self-contained bun demo and bundling playbook

## Overview

This ticket relocates the bun demo JS workspace under `cmd/bun-demo` and captures the full bundling workflow in a detailed playbook. The end state is a self-contained demo directory and a standalone doc that walks through installing dependencies, bundling with TypeScript + assets, embedding the output, and running it in Goja.

## Key Links

- [Bun demo relocation and playbook plan](./analysis/01-bun-demo-relocation-and-playbook-plan.md)
- [Diary](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- bun
- bundling
- docs
- typescript
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
