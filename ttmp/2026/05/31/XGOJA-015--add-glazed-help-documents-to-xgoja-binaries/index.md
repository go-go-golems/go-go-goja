---
Title: Add Glazed help documents to xgoja binaries
Ticket: XGOJA-015
Status: active
Topics:
    - xgoja
    - glazed
    - help-system
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket for designing provider-shipped and project-local Glazed help document bundling in generated xgoja binaries."
LastUpdated: 2026-05-31T11:36:00-04:00
WhatFor: "Track the design and future implementation of xgoja help document bundling."
WhenToUse: "Use before implementing or reviewing XGOJA-015."
---

# Add Glazed help documents to xgoja binaries

## Overview

This ticket designs how generated xgoja binaries should bundle additional Glazed help entries, including provider-owned API references and tutorials. The immediate motivating example is the Loupedeck JavaScript API reference in `loupedeck/docs/help/topics/01-loupedeck-js-api-reference.md`.

The recommended design adds provider-registered help sources plus a `help.sources` buildspec section for selecting provider-shipped or project-local help docs. Local help directories can be copied into the generated build workspace and embedded into the final binary.

## Key Links

- [Design guide](./design-doc/01-glazed-help-documents-for-xgoja-binaries-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**. The research and design package is complete; implementation remains as follow-up work.

## Topics

- xgoja
- glazed
- help-system
- documentation

## Structure

- `design-doc/` - Architecture and implementation guide.
- `reference/` - Investigation diary and reusable context.
- `playbooks/` - Command sequences and test procedures, if added later.
- `scripts/` - Temporary code and tooling, if needed later.
- `various/` - Working notes.
- `archive/` - Deprecated or reference-only artifacts.
