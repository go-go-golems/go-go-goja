---
Title: Embed files into generated xgoja binaries
Ticket: XGOJA-016
Status: active
Topics:
    - architecture
    - fs
    - goja
    - goja-nodejs
    - modules
    - providers
    - runtime
    - templates
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket workspace for designing arbitrary embedded asset support in generated xgoja binaries, exposed through the Goja fs module by runtime configuration."
LastUpdated: 2026-06-01T08:09:12.063977354-04:00
WhatFor: "Track the research, design, implementation plan, and delivery artifacts for XGOJA-016."
WhenToUse: "Use this index to find the implementation guide, diary, scripts, tasks, and changelog for embedded xgoja assets."
---

# Embed files into generated xgoja binaries

## Overview

This ticket designs support for embedding arbitrary local files into generated xgoja binaries and making those files readable from JavaScript through `require("fs")` / `require("node:fs")` when a runtime profile explicitly configures embedded asset mounts.

The recommended design generalizes xgoja's existing embedded jsverbs/help-doc pipeline into a top-level `assets:` buildspec section, passes a generated `embed.FS` through the xgoja app host, and refactors `modules/fs` behind backends so embedded mounts can be read-only without granting host filesystem access.

## Key links

- [Design / implementation guide](./design-doc/01-embedding-files-into-xgoja-binaries.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)
- [Experiment script](./scripts/01-inspect-current-embedded-sources.sh)
- [Experiment output](./scripts/01-inspect-current-embedded-sources.out)

## Status

Current status: **active**. The design package is ready for implementation review; no product code has been changed in this ticket.

## Topics

- architecture
- fs
- goja
- goja-nodejs
- modules
- providers
- runtime
- templates
- xgoja

## Structure

- `design-doc/` — intern-oriented architecture, design, pseudocode, API references, file references, and phased implementation plan.
- `reference/` — chronological investigation diary.
- `scripts/` — temporary investigation script and captured output.
- `playbooks/`, `sources/`, `various/`, `archive/` — available for follow-up implementation artifacts.
